package commands

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SplitExcelBySheets 按照 Sheets 拆分 Excel 为多个文件
func SplitExcelBySheets(slient bool, src string) error {
	prg := NewProgressbar(!slient, "opening file ...")
	defer prg.Close()

	srcFile, err := excelize.OpenFile(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	logger := NewLogger()
	defer logger.Flush()

	for _, sheet := range srcFile.GetSheetList() {
		prg.Reset(0, fmt.Sprintf("loading sheet %s ...", sheet))

		rows, err := srcFile.GetRows(sheet)
		if err != nil {
			return err
		}

		if len(rows) == 0 {
			continue
		}

		prg.Reset(len(rows), fmt.Sprintf("processing sheet %s ...", sheet))

		if err := func(sheet string, rows [][]string) error {

			destFile := excelize.NewFile()
			defer destFile.Close()

			// 重命名默认的 Sheet 名称为 src 文件中的名称
			destFile.SetSheetName(destFile.GetSheetName(0), sheet)

			for i, row := range rows {
				prg.Add(1)
				if err := destFile.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &row); err != nil {
					return err
				}
			}

			destFilename := fmt.Sprintf("%s.%s.xlsx", strings.TrimSuffix(src, ".xlsx"), sheet)
			if err := destFile.SaveAs(destFilename); err != nil {
				return err
			}

			logger.Add(fmt.Sprintf("save file %s", destFilename))

			return nil
		}(sheet, rows); err != nil {
			return err
		}
	}

	return nil
}
