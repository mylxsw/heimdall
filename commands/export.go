package commands

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/query"
	"github.com/urfave/cli/v2"
)

type ExportOption struct {
	SQL                     string
	Format                  string
	Output                  string
	Streaming               bool
	NoHeader                bool
	QueryTimeout            time.Duration
	XLSXMaxRow              int
	TargetTableForSQLFormat string
}

func BuildExportFlags() []cli.Flag {
	return append(BuildGlobalFlags(), []cli.Flag{
		&cli.StringFlag{Name: "sql", Aliases: []string{"s", "query"}, Value: "", Usage: "SQL statement(if not set, read from STDIN, end with ';')"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "csv", Usage: "output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "streaming", Aliases: []string{"S"}, Value: false, Usage: "whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/xlsx/json/plain/sql"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.DurationFlag{Name: "query-timeout", Aliases: []string{"t"}, Value: 120 * time.Second, Usage: "query timeout, when the stream option is specified, this option is invalid"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
		&cli.StringFlag{Name: "table", Value: "", Usage: "when the format is sql, specify the table name"},
	}...)
}

func resolveExportOption(c *cli.Context) ExportOption {
	sqlStr := c.String("sql")
	if sqlStr == "" {
		sqlStr = readAll(os.Stdin, ';')
	}

	return ExportOption{
		SQL:                     sqlStr,
		Format:                  c.String("format"),
		Output:                  c.String("output"),
		Streaming:               c.Bool("streaming"),
		NoHeader:                c.Bool("no-header"),
		QueryTimeout:            c.Duration("query-timeout"),
		XLSXMaxRow:              c.Int("xlsx-max-row"),
		TargetTableForSQLFormat: c.String("table"),
	}
}

func ExportCommand(c *cli.Context) error {
	if !c.Bool("debug") {
		log.All().LogLevel(level.Info)
	}

	gOpt := resolveGlobalOption(c)
	expOpt := resolveExportOption(c)

	if expOpt.Format == "sql" && expOpt.TargetTableForSQLFormat == "" {
		return fmt.Errorf("when the format is sql, the table name is required")
	}

	handler := ternary.IfLazy(
		expOpt.Streaming,
		func() query.QueryWriteHandler {
			return query.NewStreamingQueryWriter(gOpt.DSN(), expOpt.TargetTableForSQLFormat, gOpt.ConnectTimeout)
		},
		func() query.QueryWriteHandler {
			return query.NewStandardQueryWriter(gOpt.DSN(), expOpt.TargetTableForSQLFormat, gOpt.ConnectTimeout, expOpt.QueryTimeout)
		},
	)

	w := ternary.IfElseLazy(expOpt.Output != "", func() io.WriteCloser {
		return must.Must(os.Create(expOpt.Output))
	}, func() io.WriteCloser {
		return os.Stdout
	})
	defer w.Close()

	startTime := time.Now()
	total := must.Must(handler(expOpt.SQL, nil, expOpt.Format, w, expOpt.NoHeader))

	log.Debugf("write to %s, total %d records, %s elapsed", ternary.If(expOpt.Output == "", "STDOUT", expOpt.Output), total, time.Since(startTime))

	return nil
}
