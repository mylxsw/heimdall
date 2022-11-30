package commands

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/extracter"
	"github.com/mylxsw/heimdall/query"
	"github.com/mylxsw/heimdall/reader"
	"github.com/urfave/cli/v2"
)

type FlyOption struct {
	SQL         string
	InputFiles  []string
	CSVSepertor rune

	Format                  string
	Output                  string
	NoHeader                bool
	QueryTimeout            time.Duration
	XLSXMaxRow              int
	TargetTableForSQLFormat string

	UseColumnNumAsName bool
	ShowTables         bool
	TempDS             string
}

func BuildFlyFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "sql", Aliases: []string{"s", "query"}, Value: "", Usage: "SQL statement(if not set, read from STDIN, end with ';')"},
		&cli.StringSliceFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel or csv file path, this flag can be specified multiple times for importing multiple files at the same time", Required: true},
		&cli.StringFlag{Name: "csv-sepertor", Value: ",", Usage: "csv file sepertor, default is ','"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "table", Usage: "output format, support " + strings.Join(query.SupportedStandardFormats, ", ")},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.DurationFlag{Name: "query-timeout", Aliases: []string{"t"}, Value: 120 * time.Second, Usage: "query timeout, when the stream option is specified, this option is invalid"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
		&cli.StringFlag{Name: "table", Value: "", Usage: "when the format is sql, specify the table name"},
		&cli.BoolFlag{Name: "use-column-num", Value: false, Usage: "use column number as column name, start from 0, for example: col_0, col_1, col_2..."},
		&cli.BoolFlag{Name: "show-tables", Value: false, Usage: "show all tables in the database"},
		&cli.StringFlag{Name: "temp-ds", Value: ":memory:", Usage: "the temporary database uri, such as file:data.db?cache=shared, more options: https://www.sqlite.org/c3ref/open.html"},
	}
}

func resolveFlyOption(c *cli.Context) FlyOption {
	showTables := c.Bool("show-tables")
	sqlStr := c.String("sql")
	if sqlStr == "" && !showTables {
		sqlStr = readAll(os.Stdin, ';')
	}

	return FlyOption{
		SQL:         strings.Trim(strings.TrimSpace(sqlStr), ";"),
		InputFiles:  array.Filter(c.StringSlice("file"), func(f string, _ int) bool { return f != "" }),
		CSVSepertor: rune(c.String("csv-sepertor")[0]),

		Format:                  c.String("format"),
		Output:                  c.String("output"),
		NoHeader:                c.Bool("no-header"),
		QueryTimeout:            c.Duration("query-timeout"),
		XLSXMaxRow:              c.Int("xlsx-max-row"),
		TargetTableForSQLFormat: c.String("table"),
		UseColumnNumAsName:      c.Bool("use-column-num"),
		ShowTables:              showTables,
		TempDS:                  c.String("temp-ds"),
	}
}

func FlyCommand(c *cli.Context) error {
	if !c.Bool("debug") {
		log.All().LogLevel(level.Info)
	}

	opt := resolveFlyOption(c)

	if opt.SQL == "" && !opt.ShowTables {
		return fmt.Errorf("--sql or -s is required")
	}

	db, err := sql.Open("sqlite3", opt.TempDS)
	if err != nil {
		return fmt.Errorf("create sqlite3 database failed: %w", err)
	}
	defer db.Close()

	tables, err := createMemoryDatabaseForFly(opt, db)
	if err != nil {
		return err
	}

	handler := query.NewStandardQueryWriterWithDB(db, opt.TargetTableForSQLFormat, opt.QueryTimeout)

	if opt.ShowTables {
		for _, table := range tables {
			fmt.Printf("◇ Table: %s ⇢ %s\n", table.Name, table.Filename)

			dataProcesser := func(r *extracter.Rows) {
				r.Columns = append(append(append([]extracter.Column{}, r.Columns[0:1]...), extracter.Column{Name: "original"}), r.Columns[1:]...)

				for i, row := range r.DataSets {
					row["pk"] = ternary.If(row["pk"].(int64) > 0, "Y", "N")
					row["notnull"] = ternary.If(row["notnull"].(int64) > 0, "Y", "N")

					if i > 0 {
						// row["name"] == memoryTableIDField
						row["original"] = table.OriginalColumns[i-1]
					}
				}
			}

			if _, err := handler(fmt.Sprintf("PRAGMA table_info(%s)", table.Name), nil, "table", os.Stdout, false, dataProcesser); err != nil {
				return err
			}

			fmt.Println()
		}
		return nil
	}

	w := ternary.IfElseLazy(opt.Output != "", func() io.WriteCloser {
		return must.Must(os.Create(opt.Output))
	}, func() io.WriteCloser {
		return os.Stdout
	})
	defer w.Close()

	_, err = handler(opt.SQL, nil, opt.Format, w, opt.NoHeader, nil)

	return err
}

const (
	memoryTableIDField = "__rowid"
)

type Table struct {
	Name            string
	Filename        string
	Columns         []string
	OriginalColumns []string
}

func createMemoryDatabaseForFly(opt FlyOption, db *sql.DB) ([]Table, error) {
	walker := reader.MergeWalkers(array.Map(
		opt.InputFiles,
		func(f string, _ int) reader.FileWalker {
			return reader.CreateFileWalker(f, opt.CSVSepertor, opt.ShowTables)
		})...,
	)
	if walker == nil {
		return nil, fmt.Errorf("no file avaiable: only support csv or xlsx files")
	}

	var tables = make([]Table, 0)

	var tableName string
	var tableFields []string
	var index = 1
	fileIndexs := array.BuildMap(opt.InputFiles, func(val string, i int) (string, int) { return val, i })

	err := walker(
		func(filepath string, headers []string) error {
			tableName = fmt.Sprintf("table_%d", fileIndexs[filepath])
			tableFields = append([]string{memoryTableIDField}, array.Map(headers, func(h string, i int) string {
				if opt.UseColumnNumAsName {
					return fmt.Sprintf("col_%d", i)
				}

				return slugifyColumnName(h)
			})...)

			tables = append(tables, Table{
				Name:            tableName,
				Columns:         tableFields,
				OriginalColumns: headers,
				Filename:        filepath,
			})

			createSQL := fmt.Sprintf(
				"CREATE TABLE %s (%s int PRIMARY KEY NOT NULL, %s);",
				tableName,
				memoryTableIDField,
				strings.Join(array.Map(headers, func(h string, i int) string {
					if opt.UseColumnNumAsName {
						return fmt.Sprintf("col_%d TEXT", i)
					}
					return fmt.Sprintf("%s TEXT", slugifyColumnName(h))
				}), ","),
			)

			if _, err := db.Exec(createSQL); err != nil {
				return fmt.Errorf("create table %s failed: %w", tableName, err)
			}

			return nil
		},
		func(filepath string, id string, data []string) error {
			if opt.ShowTables {
				return nil
			}

			defer func() { index++ }()

			maxSize := ternary.If(len(data) > (len(tableFields)-1), len(tableFields)-1, len(data))

			placeholders := strings.Join(array.Repeat("?", maxSize), ",")
			fields := strings.Join(tableFields[0:maxSize+1], ",")

			insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%d, %s);", tableName, fields, index, placeholders)

			if _, err := db.Exec(insertSQL, array.Map(data, func(d string, i int) any { return d })...); err != nil {
				return fmt.Errorf("insert data failed: %w", err)
			}

			return nil
		},
	)

	return array.DistinctBy(tables, func(item Table) string { return item.Name }), err
}

func slugifyColumnName(name string) string {
	arg := pinyin.NewArgs()
	arg.Separator = ""
	arg.Fallback = func(r rune, a pinyin.Args) []string {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return []string{string(r)}
		}

		return []string{}
	}

	p := strings.Join(pinyin.LazyPinyin(name, arg), "")
	return strings.ReplaceAll(p, " ", "_")
}
