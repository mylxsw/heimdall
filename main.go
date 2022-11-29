package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/heimdall/commands"
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
			Usage:     "query data from input using sql",
			UsageText: "heimdall fly --sql 'select * from table' --input users.csv",
			Action:    commands.FlyCommand,
			Flags:     commands.BuildFlyFlags(),
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
