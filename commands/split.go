package commands

import (
	"fmt"

	"github.com/mylxsw/asteria/event"
	"github.com/mylxsw/asteria/filter"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/must"
	"github.com/urfave/cli/v2"
	"github.com/xuri/excelize/v2"
)

type SplitOption struct {
	InputFile       string
	Slient          bool
	Debug           bool
	Mode            string
	PerfileLimit    int
	HeaderRowEndNum int
	ColumnIndex     string
}

func BuildSplitFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel file path, currently only support xlsx format", Required: true},
		&cli.BoolFlag{Name: "slient", Value: false, Usage: "do not print warning log"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Value: false, Usage: "debug mode"},
		&cli.StringFlag{Name: "mode", Aliases: []string{"m"}, Usage: "split method: row, column, sheet", Value: "row"},
		&cli.IntFlag{Name: "perfile-limit", Aliases: []string{"p"}, Value: 1000, Usage: "the maximum number of records per file, only valid when mode=row"},
		&cli.IntFlag{Name: "header-row-num", Aliases: []string{"r"}, Value: 1, Usage: "table header row maximum row number, only valid when mode=row or mode=column"},
		&cli.StringFlag{Name: "column-index", Aliases: []string{"c"}, Usage: "specifies the index of the column to split, such as 'A', 'AA', only valid when mode=column"},
	}
}

func resolveSplitOption(c *cli.Context) SplitOption {
	return SplitOption{
		InputFile:       c.String("input"),
		Slient:          c.Bool("slient"),
		Debug:           c.Bool("debug"),
		Mode:            c.String("mode"),
		PerfileLimit:    c.Int("perfile-limit"),
		HeaderRowEndNum: c.Int("header-row-num"),
		ColumnIndex:     c.String("column-index"),
	}
}

func SplitCommand(c *cli.Context) error {
	opt := resolveSplitOption(c)
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

	switch opt.Mode {
	case "column":
		if opt.ColumnIndex == "" {
			return fmt.Errorf("column-index is required when mode=column")
		}

		return SplitExcelByColumn(opt.Slient, opt.InputFile, opt.HeaderRowEndNum, opt.ColumnIndex)
	case "sheet":
		return SplitExcelBySheets(opt.Slient, opt.InputFile)
	default:
		return SplitExcelToParts(opt.Slient, opt.InputFile, opt.HeaderRowEndNum, opt.PerfileLimit)
	}
}

func copyExcelHeaders(srcFile *excelize.File, destFile *excelize.File, sheet string, rows *[][]string, headerRowEndNum int) error {
	// 复制表头字段导目标文件
	for i, row := range (*rows)[:headerRowEndNum] {
		for j, cell := range row {
			cellName, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				return err
			}
			destFile.SetCellValue(sheet, cellName, cell)
		}
	}

	// 合并表头字段单元格
	func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("something went wrong while merging cells: %v", err)
			}
		}()

		for _, cell := range must.Must(srcFile.GetMergeCells(sheet)) {
			_, startRowNum, _ := excelize.SplitCellName(cell.GetStartAxis())
			_, endRowNum, _ := excelize.SplitCellName(cell.GetEndAxis())

			if startRowNum > headerRowEndNum || endRowNum > headerRowEndNum {
				continue
			}

			if err := destFile.MergeCell(sheet, cell.GetStartAxis(), cell.GetEndAxis()); err != nil {
				panic(fmt.Errorf("merge cell for %s failed: %v", sheet, err))
			}
		}
	}()

	return nil
}
