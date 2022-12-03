package commands

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/mylxsw/asteria/event"
	"github.com/mylxsw/asteria/filter"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/maps"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/extracter"
	"github.com/mylxsw/heimdall/query"
	"github.com/mylxsw/heimdall/reader"
	"github.com/mylxsw/heimdall/render"
	"github.com/urfave/cli/v2"
)

type ConvertOption struct {
	InputFile   string
	CSVSepertor rune
	Slient      bool
	Debug       bool

	Format                  string
	Output                  string
	NoHeader                bool
	XLSXMaxRow              int
	TargetTableForSQLFormat string

	Includes []string
	Excludes []string
}

func BuildConvertFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel or csv file path", Required: true},
		&cli.StringFlag{Name: "csv-sepertor", Value: ",", Usage: "csv file sepertor, default is ','"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "table", Usage: "output format, support " + strings.Join(query.SupportedStandardFormats, ", ")},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
		&cli.StringFlag{Name: "table", Value: "", Usage: "when the format is sql, specify the table name"},
		&cli.BoolFlag{Name: "slient", Value: false, Usage: "do not print warning log"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Value: false, Usage: "Debug mode"},
		&cli.StringSliceFlag{Name: "include", Aliases: []string{"I"}, Usage: "include fields, if set, only these fields will be output, this flag can be specified multiple times"},
		&cli.StringSliceFlag{Name: "exclude", Aliases: []string{"E"}, Usage: "exclude fields, if set, these fields will be ignored, this flag can be specified multiple times"},
	}
}

func resolveConvertOption(c *cli.Context) ConvertOption {
	includes := c.StringSlice("include")
	excludes := c.StringSlice("exclude")
	if len(includes) > 0 && len(excludes) > 0 {
		log.WithFields(log.Fields{
			"includes": includes,
			"excludes": excludes,
		}).Warning("includes and excludes are both set, excludes will be ignored")
	}

	return ConvertOption{
		InputFile:               c.String("input"),
		CSVSepertor:             rune(c.String("csv-sepertor")[0]),
		Format:                  c.String("format"),
		Output:                  c.String("output"),
		NoHeader:                c.Bool("no-header"),
		XLSXMaxRow:              c.Int("xlsx-max-row"),
		TargetTableForSQLFormat: c.String("table"),
		Slient:                  c.Bool("slient"),
		Debug:                   c.Bool("debug"),

		Includes: includes,
		Excludes: ternary.If(len(includes) > 0, []string{}, excludes),
	}
}

func ConvertCommand(c *cli.Context) error {
	opt := resolveConvertOption(c)
	if !opt.Debug {
		log.All().LogLevel(level.Info)
	}

	if opt.Slient {
		log.AddGlobalFilter(func(filter filter.Filter) filter.Filter {
			return func(evt event.Event) {
				if evt.Level == level.Warning {
					return
				}

				filter(evt)
			}
		})
	}

	if opt.InputFile == "" {
		return fmt.Errorf("input file (--file) is required")
	}

	if opt.Format == "sql" && opt.TargetTableForSQLFormat == "" {
		return fmt.Errorf("when the format is sql, the table name (--table) is required")
	}

	walker := reader.CreateFileWalker(opt.InputFile, opt.CSVSepertor, false, false)
	if walker == nil {
		return fmt.Errorf("no file avaiable: only support csv or xlsx files")
	}

	cols := make([]extracter.Column, 0)
	kvs := make([]map[string]interface{}, 0)
	if err := walker(
		func(filepath string, headers []string) error {
			cols = array.Map(headers, func(header string, _ int) extracter.Column {
				if len(opt.Includes) > 0 {
					if !str.InIgnoreCase(header, opt.Includes) {
						return extracter.Column{}
					}
				} else if len(opt.Excludes) > 0 {
					if str.InIgnoreCase(header, opt.Excludes) {
						return extracter.Column{}
					}
				}

				return extracter.Column{Name: header, Type: extracter.ColumnTypeVarchar, ScanType: reflect.TypeOf("")}
			})
			return nil
		},
		func(filepath string, id string, data []string) error {
			kvs = append(kvs, maps.Filter(array.BuildMap(data, func(item string, i int) (string, interface{}) {
				if i > len(cols)-1 || cols[i].Name == "" {
					return "", nil
				}

				return cols[i].Name, item
			}), func(_ interface{}, k string) bool { return k != "" }))
			return nil
		},
	); err != nil {
		return err
	}

	cols = array.Filter(cols, func(col extracter.Column, _ int) bool { return col.Name != "" })
	res, err := render.Render(opt.Format, false, cols, kvs, "", opt.TargetTableForSQLFormat)
	if err != nil {
		return err
	}

	w := ternary.IfElseLazy(
		opt.Output != "",
		func() io.WriteCloser { return must.Must(os.Create(opt.Output)) },
		func() io.WriteCloser { return os.Stdout },
	)
	defer w.Close()

	_, err = w.Write(res.Bytes())
	return err
}
