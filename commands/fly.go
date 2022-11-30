package commands

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
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
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "csv", Usage: "output format, support " + strings.Join(query.SupportedStandardFormats, ", ")},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.DurationFlag{Name: "query-timeout", Aliases: []string{"t"}, Value: 120 * time.Second, Usage: "query timeout, when the stream option is specified, this option is invalid"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
		&cli.StringFlag{Name: "table", Value: "", Usage: "when the format is sql, specify the table name"},
		&cli.BoolFlag{Name: "use-column-num", Value: false, Usage: "use column number as column name, start from 0, for example: col_0, col_1, col_2..."},
		&cli.BoolFlag{Name: "show-tables", Value: false, Usage: "show all tables in the database"},
		&cli.StringFlag{Name: "temp-ds", Value: ":memory:", Usage: "the temporary database uri"},
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

	tableNames, err := createMemoryDatabaseForFly(opt, db)
	if err != nil {
		return err
	}

	handler := query.NewStandardQueryWriterWithDB(db, opt.TargetTableForSQLFormat, opt.QueryTimeout)

	if opt.ShowTables {
		for i, tableName := range tableNames {
			fmt.Printf("Table: %s ⇢ %s\n", tableName, opt.InputFiles[i])

			if _, err := handler(fmt.Sprintf("PRAGMA table_info(%s)", tableName), nil, "table", os.Stdout, false); err != nil {
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

	_, err = handler(opt.SQL, nil, opt.Format, w, opt.NoHeader)

	return err
}

const (
	memoryTableIDField = "__rowid"
)

func createMemoryDatabaseForFly(opt FlyOption, db *sql.DB) ([]string, error) {
	walker := reader.MergeWalkers(array.Map(
		opt.InputFiles,
		func(f string, _ int) reader.FileWalker { return reader.CreateFileWalker(f, opt.CSVSepertor) })...,
	)
	if walker == nil {
		return nil, fmt.Errorf("no file avaiable: only support csv or xlsx files")
	}

	var tableName string
	var tableFields, tableNames []string
	var index = 1
	fileIndexs := array.BuildMap(opt.InputFiles, func(val string, i int) (string, int) { return val, i })

	err := walker(
		func(filepath string, headers []string) error {
			tableName = fmt.Sprintf("table_%d", fileIndexs[filepath])
			tableNames = append(tableNames, tableName)
			tableFields = append([]string{memoryTableIDField}, array.Map(headers, func(h string, i int) string {
				if opt.UseColumnNumAsName {
					return fmt.Sprintf("col_%d", i)
				}

				return h
			})...)

			createSQL := fmt.Sprintf(
				"CREATE TABLE %s (%s int PRIMARY KEY NOT NULL, %s);",
				tableName,
				memoryTableIDField,
				strings.Join(array.Map(headers, func(h string, i int) string {
					if opt.UseColumnNumAsName {
						return fmt.Sprintf("col_%d TEXT", i)
					}
					return fmt.Sprintf("%s TEXT", h)
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

	return array.Distinct(tableNames), err
}
