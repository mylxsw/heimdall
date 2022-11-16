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
)

var (
	GitCommit   string
	Version     string
	CompileTime string
)

func main() {
	app := cli.NewApp()
	app.Name = "heimdall"
	app.Usage = "tools for database import and export"
	app.Copyright = "© 2022 mylxsw"
	app.Compiled, _ = time.Parse(time.RFC3339, CompileTime)
	app.Description = "Heimdall is a database tools specially designed for MySQL. Using it, you can directly import xlsx or csv file to database or export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text"
	app.EnableBashCompletion = true
	app.Suggest = true
	app.Version = fmt.Sprintf("%s %s", Version, GitCommit)
	app.Authors = []*cli.Author{
		{
			Name:  "mylxsw",
			Email: "mylxsw@aicode.cc",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:   "export",
			Usage:  "export data from database",
			Action: commands.ExportCommand,
			Flags:  commands.BuildExportFlags(),
		},
		{
			Name:   "import",
			Usage:  "import data to database",
			Action: commands.ImportCommand,
			Flags:  commands.BuildImportFlags(),
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

	must.NoError(app.Run(os.Args))
}
