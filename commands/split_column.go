package commands

import (
	"fmt"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/xuri/excelize/v2"
)

// SplitExcelByColumn 按照指定的列来拆分 Excel 文件
func SplitExcelByColumn(slient bool, src string, headerRowEndNum int, columnIndex string) error {
	logger := NewLogger()
	defer logger.Flush()

	column, err := excelize.ColumnNameToNumber(columnIndex)
	if err != nil {
		return fmt.Errorf("invalid column-index: %w", err)
	}

	prg := NewProgressbar(!slient, "opening src file ...")
	defer prg.Close()

	srcFile, err := excelize.OpenFile(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	prg.Reset(-1, "loading ...")

	for i, sheet := range srcFile.GetSheetList() {
		rows, err := srcFile.GetRows(sheet)
		if err != nil {
			return err
		}

		if i > 0 && len(rows) > 0 {
			log.Warningf("file has more than one sheet, only the first sheet will be processed")
			break
		}

		prg.Reset(len(rows[headerRowEndNum:]), "processing ...")

		destFileMap := make(map[string]*ExcelFileWithIndex, 0)
		for _, row := range rows[headerRowEndNum:] {
			prg.Add(1)
			colVal := ternary.IfElseLazy(len(row) < column, func() string { return "" }, func() string { return row[column-1] })
			d, ok := destFileMap[colVal]
			if !ok {
				destFile := excelize.NewFile()
				// 重命名默认的 Sheet 名称为 src 文件中的名称
				destFile.SetSheetName(destFile.GetSheetName(0), sheet)
				if err := copyExcelHeaders(srcFile, destFile, sheet, &rows, headerRowEndNum); err != nil {
					return err
				}

				d = &ExcelFileWithIndex{File: destFile, Index: headerRowEndNum}
				destFileMap[colVal] = d
			}

			d.Index++
			if err := d.File.SetSheetRow(sheet, fmt.Sprintf("A%d", d.Index), &row); err != nil {
				return err
			}
		}

		prg.Reset(len(destFileMap), "writing files ...")

		for colName, d := range destFileMap {
			prg.Add(1)

			destFilename := fmt.Sprintf("%s.%s.xlsx", strings.TrimSuffix(src, ".xlsx"), colName)
			if err := d.File.SaveAs(destFilename); err != nil {
				return err
			}

			d.File.Close()

			logger.Add(fmt.Sprintf("save file %s", destFilename))
		}
	}

	return nil
}

type ExcelFileWithIndex struct {
	File  *excelize.File
	Index int
}
