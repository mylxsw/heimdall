package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/heimdall/commands"
	"github.com/mylxsw/heimdall/query"
	"github.com/urfave/cli/v2"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

var (
	GitCommit   string
	Version     string
	CompileTime string
	Debug       string
)

func main() {
	if Debug != "true" {
		defer func() {
			if err := recover(); err != nil {
				fmt.Fprintf(os.Stderr, "😨 %v\n", err)
			}
		}()
	}

	app := cli.NewApp()
	app.Name = "heimdall"
	app.Usage = "tools for database import and export(query)"
	app.Copyright = "© 2022 mylxsw"
	app.Compiled, _ = time.Parse(time.RFC3339, CompileTime)
	app.Description = "Heimdall is a database tools specially designed for MySQL. Using it, you can directly import xlsx or csv file to database or export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text"
	app.EnableBashCompletion = true
	app.Suggest = true
	app.UseShortOptionHandling = true
	app.Version = fmt.Sprintf("%s %s", Version, GitCommit)
	app.Authors = []*cli.Author{
		{
			Name:  "mylxsw",
			Email: "mylxsw@aicode.cc",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:      "export",
			Aliases:   []string{"query"},
			Usage:     "export or query data from database",
			UsageText: `heimdall export --database example --format json --sql 'select * from users'`,
			Action:    commands.ExportCommand,
			Flags:     commands.BuildExportFlags(),
		},
		{
			Name:      "import",
			Aliases:   []string{"load"},
			Usage:     "import or load data to database",
			UsageText: "heimdall import --tx --database example --table users --file users.csv --file users.xlsx --field 姓名:name --field 年龄:age",
			Action:    commands.ImportCommand,
			Flags:     commands.BuildImportFlags(),
		},
		{
			Name:      "fly",
			Aliases:   []string{"query-file"},
			Usage:     "query data from input file using sql directly",
			UsageText: `heimdall fly --file data.csv --file data2.csv --sql "SELECT table_0.id 'ID', table_0.name '名称', table_0.created_at '创建时间', count(*) as '字段数量' FROM table_0 LEFT JOIN table_1 ON table_0.id = table_1.ref_id WHERE table_1.deleted_at = '' GROUP BY table_0.id ORDER BY count(*) DESC LIMIT 10" -f table`,
			Action:    commands.FlyCommand,
			Flags:     commands.BuildFlyFlags(),
		},
		{
			Name:      "convert",
			Usage:     "convert data from xlsx/csv to other formats: " + strings.Join(query.SupportedStandardFormats, ", "),
			UsageText: `heimdall convert --file data.csv --format json --include id --include name --include updated_at`,
			Action:    commands.ConvertCommand,
			Flags:     commands.BuildConvertFlags(),
		},
		{
			Name:  "version",
			Usage: "show version",
			Action: func(c *cli.Context) error {
				fmt.Println(string(must.Must(json.Marshal(log.Fields{"version": Version, "commit": GitCommit}))))
				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		if Debug == "true" {
			panic(err)
		}

		fmt.Fprintf(os.Stderr, "😨 %s\n", err)
	}

}
