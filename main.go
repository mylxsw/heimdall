package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/str"
	"github.com/xuri/excelize/v2"

	"github.com/facebook/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mylxsw/mysql-querier/extracter"
	"gopkg.in/yaml.v3"
)

var (
	// GitCommit Git 版本
	GitCommit string
	// Version 应用版本
	Version string
)
var outputVersion bool

var mysqlHost, mysqlUser, mysqlPassword, mysqlDB string
var mysqlPort int
var sqlStr string
var format, output string
var queryTimeout time.Duration
var fields string

func main() {

	flag.StringVar(&mysqlHost, "host", "127.0.0.1", "MySQL 主机地址")
	flag.StringVar(&mysqlDB, "db", "", "MySQL 数据库名")
	flag.StringVar(&mysqlPassword, "password", "", "MySQL 密码")
	flag.StringVar(&mysqlUser, "user", "root", "MySQL 用户")
	flag.IntVar(&mysqlPort, "port", 3306, "MySQL 端口")
	flag.StringVar(&sqlStr, "sql", "", "要执行的 SQL 查询语句，如果不指定则从标准输入读取")
	flag.StringVar(&format, "format", "table", "输出格式： json/yaml/plain/table/csv/html/markdown/xlsx/xml")
	flag.StringVar(&output, "output", "", "将输出写入到文件，默认直接输出到标准输出")
	flag.BoolVar(&outputVersion, "version", false, "输出版本信息")
	flag.DurationVar(&queryTimeout, "timeout", 10*time.Second, "查询超时时间")
	flag.StringVar(&fields, "fields", "", "查询字段列表，默认为全部字段，字段之间使用英文逗号分隔")

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

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
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

	if len(fields) > 0 {
		fieldNames := array.Filter(strings.Split(fields, ","), func(item string) bool { return strings.TrimSpace(item) != "" })
		colNames = array.Filter(fieldNames, func(item string) bool {
			return str.InIgnoreCase(item, colNames)
		})

		kvs = array.Map(kvs, func(item map[string]interface{}) map[string]interface{} {
			res := make(map[string]interface{})
			for k, v := range item {
				if str.InIgnoreCase(k, fieldNames) {
					res[k] = v
				}
			}

			return res
		})
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
	case "xml":
		if err := printXML(writer, kvs, sqlStr); err != nil {
			panic(err)
		}
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
		if len(kvs) > 10 {
			row := table.Row{}
			if len(colNames) > 1 {
				row = append(row, "Total")
				for i := 0; i < len(colNames)-1; i++ {
					row = append(row, len(kvs))
				}
			} else {
				row = append(row, fmt.Sprintf("Total %d", len(kvs)))
			}

			t.AppendFooter(row, table.RowConfig{AutoMerge: true})
		}
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

type XMLField struct {
	XMLName xml.Name    `xml:"field"`
	Name    string      `xml:"name,attr"`
	Value   interface{} `xml:",chardata"`
}

type XMLRow struct {
	XMLName xml.Name `xml:"row"`
	Value   []XMLField
}

type XMLResultSet struct {
	XMLName   xml.Name `xml:"resultset"`
	Statement string   `xml:"statement,attr"`
	XMLNS     string   `xml:"xmlns:xsi,attr"`
	Value     []XMLRow
}

func printXML(w io.Writer, data []map[string]interface{}, sqlStr string) error {
	result := XMLResultSet{
		Statement: sqlStr,
		XMLNS:     "http://www.w3.org/2001/XMLSchema-instance",
		Value: array.Map(data, func(item map[string]interface{}) XMLRow {
			row := XMLRow{Value: make([]XMLField, 0)}
			for k, v := range item {
				row.Value = append(row.Value, XMLField{
					Name:  k,
					Value: v,
				})
			}

			return row
		}),
	}

	marshalData, err := xml.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, xml.Header+string(marshalData))
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
