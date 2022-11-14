package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/db-tools/commands"
	"github.com/mylxsw/go-utils/must"
	"github.com/urfave/cli/v2"

	_ "github.com/go-sql-driver/mysql"
)

var (
	GitCommit string
	Version   string
)

func main() {
	app := cli.NewApp()
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
