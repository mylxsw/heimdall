package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mylxsw/go-utils/array"
	"github.com/xuri/excelize/v2"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/facebook/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mylxsw/mysql-querier/extracter"
	"gopkg.in/yaml.v3"
)

var (
	// Git 版本
	GitCommit string
	// 应用版本
	Version string
)
var outputVersion bool

var mysqlHost, mysqlUser, mysqlPassword, mysqlDB string
var mysqlPort int
var sqlStr string
var format, output string

func main() {

	flag.StringVar(&mysqlHost, "host", "127.0.0.1", "MySQL Host")
	flag.StringVar(&mysqlDB, "db", "", "MySQL Database")
	flag.StringVar(&mysqlPassword, "password", "", "MySQL Password")
	flag.StringVar(&mysqlUser, "user", "root", "MySQL User")
	flag.IntVar(&mysqlPort, "port", 3306, "MySQL Port")
	flag.StringVar(&sqlStr, "sql", "", "The SQL to be executed, if not specified, read from the standard input pipe")
	flag.StringVar(&format, "format", "json", "Output format: json/yaml/plain/table/csv/html/markdown/xlsx")
	flag.StringVar(&output, "output", "", "Write output to a file, default write to stdout")
	flag.BoolVar(&outputVersion, "version", false, "Output version information")

	flag.Parse()

	if outputVersion {
		fmt.Printf("Version=%s, GitCommit=%s\n", Version, GitCommit)
		return
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?loc=Local&parseTime=true", mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDB))
	if err != nil {
		panic(err)
	}

	if sqlStr == "" {
		sqlStr = readStdin()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	results, err := extracter.Extract(rows)
	if err != nil {
		panic(err)
	}

	kvs := make([]map[string]interface{}, 0)
	for _, row := range results.DataSets {
		rowData := make(map[string]interface{})
		for i, col := range row {
			rowData[results.Columns[i].Name] = col
		}

		kvs = append(kvs, rowData)
	}

	colNames := make([]string, 0)
	for _, col := range results.Columns {
		colNames = append(colNames, col.Name)
	}

	writer := bytes.NewBuffer(nil)

	switch format {
	case "json":
		if err := printJSON(writer, kvs); err != nil {
			panic(err)
		}
	case "yaml":
		if err := printYAML(writer, kvs); err != nil {
			panic(err)
		}
	case "table":
		renderTable(writer, colNames, kvs, "table")
	case "markdown":
		renderTable(writer, colNames, kvs, "markdown")
	case "csv":
		renderTable(writer, colNames, kvs, "csv")
	case "html":
		renderTable(writer, colNames, kvs, "html")
	case "xlsx":
		exf := excelize.NewFile()
		exfCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
		for i, colName := range colNames {
			_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[i], 1), colName)
		}

		for i, kv := range kvs {
			for j, colName := range colNames {
				_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[j], i+2), kv[colName])
			}
		}

		_ = exf.Write(writer)
	default:
		for _, kv := range kvs {
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(lines, strings.ReplaceAll(fmt.Sprintf("%s=%v", colName, kv[colName]), "\n", "\\n"))
			}

			writer.WriteString(fmt.Sprintln(strings.Join(lines, ", ")))
		}
	}

	if output != "" {
		if err := ioutil.WriteFile(output, writer.Bytes(), os.ModePerm); err != nil {
			panic(err)
		}
	} else {
		_, _ = writer.WriteTo(os.Stdout)
	}
}

func renderTable(writer io.Writer, colNames []string, kvs []map[string]interface{}, typ string) {
	t := table.NewWriter()
	t.SetOutputMirror(writer)
	t.AppendHeader(array.Map(colNames, func(name string) interface{} { return name }))
	t.AppendRows(array.Map(kvs, func(kv map[string]interface{}) table.Row {
		row := table.Row{}
		for _, colName := range colNames {
			row = append(row, kv[colName])
		}

		return row
	}))

	switch typ {
	case "markdown":
		t.RenderMarkdown()
	case "html":
		t.RenderHTML()
	case "csv":
		t.RenderCSV()
	default:
		t.Render()
	}
}

func printYAML(w io.Writer, data interface{}) error {
	marshalData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, string(marshalData))
	return err
}

func printJSON(w io.Writer, data interface{}) error {
	marshalData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, string(marshalData))
	return err
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
