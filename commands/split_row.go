package commands

import (
	"fmt"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/xuri/excelize/v2"
)

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
func SplitExcelToParts(slient bool, src string, headerRowEndNum, perFileRecordCount int) error {
	prg := NewProgressbar(!slient, "opening file ...")
	defer prg.Close()

	srcFile, err := excelize.OpenFile(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	logger := NewLogger()
	defer logger.Flush()

	prg.Reset(0, "loading ...")

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

		prg.Reset(len(rows[headerRowEndNum:]), "processing ...")

		for j, rang := range SplitNumToRange(len(rows)-headerRowEndNum, perFileRecordCount) {
			if err := func(j int, rang NumRange) error {
				destFile := excelize.NewFile()
				defer destFile.Close()

				// 重命名默认的 Sheet 名称为 src 文件中的名称
				destFile.SetSheetName(destFile.GetSheetName(0), sheet)
				if err := copyExcelHeaders(srcFile, destFile, sheet, &rows, headerRowEndNum); err != nil {
					return err
				}

				for i, row := range rows[rang.Start+headerRowEndNum : rang.Start+headerRowEndNum+rang.Count] {
					prg.Add(1)
					if err := destFile.SetSheetRow(sheet, fmt.Sprintf("A%d", headerRowEndNum+i+1), &row); err != nil {
						return err
					}
				}

				destFilename := fmt.Sprintf("%s.part%d.xlsx", strings.TrimSuffix(src, ".xlsx"), j+1)
				if err := destFile.SaveAs(destFilename); err != nil {
					return err
				}

				logger.Add(fmt.Sprintf("save file %s", destFilename))

				return nil
			}(j, rang); err != nil {
				return err
			}
		}
	}

	return nil
}
