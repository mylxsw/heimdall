package commands

import (
	"io"
	"os"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/db-exporter/query"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/urfave/cli/v2"
)

type ExportOption struct {
	SQL        string
	Format     string
	Output     string
	Streaming  bool
	NoHeader   bool
	Timeout    time.Duration
	XLSXMaxRow int
}

func BuildExportFlags() []cli.Flag {
	return append(BuildGlobalFlags(), []cli.Flag{
		&cli.StringFlag{Name: "sql", Aliases: []string{"s"}, Value: "", Usage: "SQL statement"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "csv", Usage: "output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "streaming", Aliases: []string{"S"}, Value: false, Usage: "whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/json/plain"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.DurationFlag{Name: "timeout", Aliases: []string{"t"}, Value: 0, Usage: "query timeout, when the stream option is specified, this option is invalid"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
	}...)
}

func resolveExportOption(c *cli.Context) ExportOption {
	sqlStr := c.String("sql")
	if sqlStr == "" {
		sqlStr = readStdin()
	}

	return ExportOption{
		SQL:        sqlStr,
		Format:     c.String("format"),
		Output:     c.String("output"),
		Streaming:  c.Bool("streaming"),
		NoHeader:   c.Bool("no-header"),
		Timeout:    c.Duration("timeout"),
		XLSXMaxRow: c.Int("xlsx-max-row"),
	}
}

func ExportCommand(c *cli.Context) error {
	if !c.Bool("debug") {
		log.All().LogLevel(level.Info)
	}

	gOpt := resolveGlobalOption(c)
	expOpt := resolveExportOption(c)

	handler := ternary.IfLazy(
		expOpt.Streaming,
		func() query.QueryWriteHandler { return query.NewStreamingQueryWriter(gOpt.DSN()) },
		func() query.QueryWriteHandler { return query.NewStandardQueryWriter(gOpt.DSN(), expOpt.Timeout) },
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
