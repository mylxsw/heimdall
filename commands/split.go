package commands

import (
	"fmt"
	"strings"

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
	PerfileLimit    int
	HeaderRowEndNum int
}

func BuildSplitFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel file path, currently only support xlsx format", Required: true},
		&cli.BoolFlag{Name: "slient", Value: false, Usage: "do not print warning log"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Value: false, Usage: "debug mode"},
		&cli.IntFlag{Name: "perfile-limit", Aliases: []string{"p"}, Value: 1000, Usage: "the maximum number of records per file"},
		&cli.IntFlag{Name: "header-row-num", Aliases: []string{"r"}, Value: 1, Usage: "table header row maximum row number"},
	}
}

func resolveSplitOption(c *cli.Context) SplitOption {
	return SplitOption{
		InputFile:       c.String("input"),
		Slient:          c.Bool("slient"),
		Debug:           c.Bool("debug"),
		PerfileLimit:    c.Int("perfile-limit"),
		HeaderRowEndNum: c.Int("header-row-num"),
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

	return SplitExcelToParts(opt.InputFile, opt.HeaderRowEndNum, opt.PerfileLimit)
}

// NumRange 数字范围
type NumRange struct {
	// Start 起始位置
	Start int
	// Count 最大允许数量
	Count int
}

// SplitNumToRange 将 total 拆分为多个 NumRange，每个 NumRange 的 Count 不超过 per
func SplitNumToRange(total int, per int) []NumRange {
	var ranges []NumRange

	for i := 0; i < total; i += per {
		if i+per > total {
			ranges = append(ranges, NumRange{Start: i, Count: total - i})
		} else {
			ranges = append(ranges, NumRange{Start: i, Count: per})
		}
	}

	return ranges
}

// SplitExcelToParts 将 Excel 文件拆分为多个文件，每个文件的记录数不超过 perFileRecordCount
func SplitExcelToParts(src string, headerRowEndNum, perFileRecordCount int) error {
	srcFile, err := excelize.OpenFile(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	for i, sheet := range srcFile.GetSheetList() {
		rows, err := srcFile.GetRows(sheet)
		if err != nil {
			return err
		}

		if i > 0 && len(rows) > 0 {
			log.Warningf("file has more than one sheet, only the first sheet will be processed")
			break
		}

		if len(rows) < headerRowEndNum+1 {
			continue
		}

		for j, rang := range SplitNumToRange(len(rows)-headerRowEndNum, perFileRecordCount) {
			if err := func(j int, rang NumRange) error {
				destFile := excelize.NewFile()
				defer destFile.Close()

				// 重命名默认的 Sheet 名称为 src 文件中的名称
				destFile.SetSheetName(destFile.GetSheetName(0), sheet)

				// 复制表头字段导目标文件
				for i, row := range rows[:headerRowEndNum] {
					for j, cell := range row {
						cellName, err := excelize.CoordinatesToCellName(j+1, i+1)
						if err != nil {
							return err
						}
						destFile.SetCellValue(sheet, cellName, cell)
					}
				}

				// 合并表头字段单元格
				for _, cell := range must.Must(srcFile.GetMergeCells(sheet)) {
					_, startRowNum, _ := excelize.SplitCellName(cell.GetStartAxis())
					_, endRowNum, _ := excelize.SplitCellName(cell.GetEndAxis())

					if startRowNum > headerRowEndNum || endRowNum > headerRowEndNum {
						continue
					}

					if err := destFile.MergeCell(sheet, cell.GetStartAxis(), cell.GetEndAxis()); err != nil {
						return err
					}
				}

				for i, row := range rows[rang.Start+headerRowEndNum : rang.Start+headerRowEndNum+rang.Count] {
					if err := destFile.SetSheetRow(sheet, fmt.Sprintf("A%d", headerRowEndNum+i+1), &row); err != nil {
						return err
					}
				}

				destFilename := fmt.Sprintf("%s.part%d.xlsx", strings.TrimSuffix(src, ".xlsx"), j+1)
				if err := destFile.SaveAs(destFilename); err != nil {
					return err
				}

				log.Infof("save file %s", destFilename)

				return nil
			}(j, rang); err != nil {
				return err
			}
		}
	}

	return nil
}
