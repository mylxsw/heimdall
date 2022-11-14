package main

import (
	"fmt"
	"os"

	"github.com/mylxsw/db-exporter/commands"
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
	app.Version = Version
	app.Flags = commands.BuildGlobalFlags()
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
				fmt.Printf("Version=%s, GitCommit=%s\n", Version, GitCommit)
				return nil
			},
		},
	}

	must.NoError(app.Run(os.Args))
}
