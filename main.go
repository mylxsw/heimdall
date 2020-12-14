package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/facebook/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
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
var format string

func main() {

	flag.StringVar(&mysqlHost, "host", "127.0.0.1", "MySQL Host")
	flag.StringVar(&mysqlDB, "db", "", "MySQL Database")
	flag.StringVar(&mysqlPassword, "password", "", "MySQL Password")
	flag.StringVar(&mysqlUser, "user", "root", "MySQL User")
	flag.IntVar(&mysqlPort, "port", 3306, "MySQL Port")
	flag.StringVar(&sqlStr, "sql", "", "The SQL to be executed, if not specified, read from the standard input pipe")
	flag.StringVar(&format, "format", "json", "Output format: json/yaml/plain")
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

	switch format {
	case "json":
		if err := printJSON(os.Stdout, kvs); err != nil {
			panic(err)
		}
	case "yaml":
		if err := printYAML(os.Stdout, kvs); err != nil {
			panic(err)
		}
	case "plain":
		for _, kv := range kvs {
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(lines, strings.ReplaceAll(fmt.Sprintf("%s=%v", colName, kv[colName]), "\n", "\\n"))
			}

			fmt.Println(strings.Join(lines, ", "))
		}
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
