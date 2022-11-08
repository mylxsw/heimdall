package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/db-exporter/query"
	"github.com/mylxsw/db-exporter/render"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"

	_ "github.com/go-sql-driver/mysql"
)

var (
	GitCommit string
	Version   string
)
var outputVersion bool

var mysqlHost, mysqlUser, mysqlPassword, mysqlDB string
var mysqlPort int
var sqlStr string
var format, output string
var queryTimeout time.Duration
var streamingOutput, noHeader bool
var debug bool

func main() {

	flag.StringVar(&mysqlHost, "host", "127.0.0.1", "MySQL host")
	flag.StringVar(&mysqlDB, "db", "", "MySQL database name")
	flag.StringVar(&mysqlPassword, "password", "", "MySQL password")
	flag.StringVar(&mysqlUser, "user", "root", "MySQL username")
	flag.IntVar(&mysqlPort, "port", 3306, "MySQL port")
	flag.StringVar(&sqlStr, "sql", "", "SQL query to execute, read from STDIN if not specified")
	flag.StringVar(&format, "format", "csv", "Output format: json/yaml/plain/table/csv/html/markdown/xlsx/xml")
	flag.StringVar(&output, "output", "", "Write output to a file, default output directly to STDOUT")
	flag.BoolVar(&outputVersion, "version", false, "Output version info")
	flag.DurationVar(&queryTimeout, "timeout", 10*time.Second, "Query timeout, when the stream option is specified, this option is invalid")
	flag.BoolVar(&streamingOutput, "streaming", false, "Whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/json/plain")
	flag.BoolVar(&noHeader, "no-header", false, "Do not write table header")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.IntVar(&render.MaxRowNumInSheet, "xlsx-max-row", 1048576, "The maximum number of rows per sheet in an Excel file, including the row where the header is located")

	flag.Parse()

	if !debug {
		log.All().LogLevel(level.Info)
	}

	if outputVersion {
		fmt.Printf("Version=%s, GitCommit=%s\n", Version, GitCommit)
		return
	}

	if sqlStr == "" {
		sqlStr = readStdin()
	}

	dbConnStr := query.BuildConnStr(mysqlDB, mysqlUser, mysqlPassword, mysqlHost, mysqlPort)
	handler := ternary.IfLazy(
		streamingOutput,
		func() query.QueryWriteHandler { return query.NewStreamingQueryWriter(dbConnStr) },
		func() query.QueryWriteHandler { return query.NewStandardQueryWriter(dbConnStr, queryTimeout) },
	)

	w := ternary.IfElseLazy(output != "", func() io.WriteCloser {
		return must.Must(os.Create(output))
	}, func() io.WriteCloser {
		return os.Stdout
	})
	defer w.Close()

	startTime := time.Now()
	total := must.Must(handler(sqlStr, nil, format, w, noHeader))

	log.Debugf("write to %s, total %d records, %s elapsed", output, total, time.Since(startTime))
}

func readStdin() string {
	reader := bufio.NewReader(os.Stdin)
	var result []rune
	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}

		result = append(result, input)
	}

	return string(result)
}
