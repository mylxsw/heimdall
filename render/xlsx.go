package render

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

func XLSX(writer io.Writer, colNames []string, kvs []map[string]interface{}) {
	exf := excelize.NewFile()
	exfCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	for i, colName := range colNames {
		_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[i], 1), colName)
	}

	for i, kv := range kvs {
		for j, colName := range colNames {
			_ = exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[j], i+2), kv[colName])
		}
	}

	_ = exf.Write(writer)
}
