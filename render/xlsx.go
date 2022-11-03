package render

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

func XLSX(writer io.Writer, noHeader bool, colNames []string, kvs []map[string]interface{}) {
	exf := excelize.NewFile()
	exfCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	lineNo := 0

	if !noHeader {
		lineNo++
		for i, colName := range colNames {
			_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[i], lineNo), colName)
		}
	}

	for _, kv := range kvs {
		lineNo++
		for j, colName := range colNames {
			_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[j], lineNo), kv[colName])
		}
	}

	_ = exf.Write(writer)
}
