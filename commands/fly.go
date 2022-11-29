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
	}
}

func resolveFlyOption(c *cli.Context) FlyOption {
	sqlStr := c.String("sql")
	if sqlStr == "" {
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
	}
}

func FlyCommand(c *cli.Context) error {
	if !c.Bool("debug") {
		log.All().LogLevel(level.Info)
	}

	opt := resolveFlyOption(c)

	if opt.SQL == "" {
		return fmt.Errorf("--sql or -s is required")
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("create sqlite3 database failed: %w", err)
	}
	defer db.Close()

	if err := createMemoryDatabaseForFly(opt, db); err != nil {
		return err
	}

	w := ternary.IfElseLazy(opt.Output != "", func() io.WriteCloser {
		return must.Must(os.Create(opt.Output))
	}, func() io.WriteCloser {
		return os.Stdout
	})
	defer w.Close()

	handler := query.NewStandardQueryWriterWithDB(db, opt.TargetTableForSQLFormat, opt.QueryTimeout)
	_, err = handler(opt.SQL, nil, opt.Format, w, opt.NoHeader)

	return err
}

const (
	memoryTableIDField = "____rowid"
)

func createMemoryDatabaseForFly(opt FlyOption, db *sql.DB) error {
	walker := reader.MergeWalkers(array.Map(
		opt.InputFiles,
		func(f string, _ int) reader.FileWalker { return reader.CreateFileWalker(f, opt.CSVSepertor) })...,
	)
	if walker == nil {
		return fmt.Errorf("no file avaiable: only support csv or xlsx files")
	}

	var tableName string
	var tableFields []string
	var index = 1
	fileIndexs := array.BuildMap(opt.InputFiles, func(val string, i int) (string, int) { return val, i })

	return walker(
		func(filepath string, headers []string) error {
			tableName = fmt.Sprintf("table_%d", fileIndexs[filepath])
			tableFields = append([]string{memoryTableIDField}, array.Map(headers, func(h string, _ int) string { return h })...)

			createSQL := fmt.Sprintf(
				"CREATE TABLE %s (%s int PRIMARY KEY NOT NULL, %s);",
				tableName,
				memoryTableIDField,
				strings.Join(array.Map(headers, func(h string, _ int) string { return fmt.Sprintf("%s TEXT", h) }), ","),
			)

			log.With(log.Fields{
				"sql": createSQL,
			}).Debugf("create table")

			if _, err := db.Exec(createSQL); err != nil {
				return fmt.Errorf("create table %s failed: %w", tableName, err)
			}

			return nil
		},
		func(filepath string, id string, data []string) error {
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
}
