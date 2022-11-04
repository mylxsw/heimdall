package render

import (
	"fmt"
	"io"
	"strconv"

	"github.com/mylxsw/go-utils/array"
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

type ExcelWriter struct {
	filename string
	excel    *excelize.File
	stream   *excelize.StreamWriter
	rowNum   int
	sheetNum int
	headers  []string
}

// Excel 单个 Sheet 最多支持 1048576 行
// https://support.microsoft.com/en-us/office/excel-specifications-and-limits-1672b34d-7043-467e-8e27-269d656771c3
var MaxRowNumInSheet = 1048576

func NewExcelWriter(filename string, headers []string) (Writer, error) {
	f := excelize.NewFile()
	streamWriter, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return nil, err
	}

	return &ExcelWriter{
		excel:    f,
		stream:   streamWriter,
		filename: filename,
		sheetNum: 1,
		headers:  headers,
	}, nil
}

func (w *ExcelWriter) Write(data []string) error {
	defer func() {
		if w.rowNum >= MaxRowNumInSheet {
			w.rowNum = 0
			w.stream.Flush()
			w.sheetNum++
			w.excel.NewSheet("Sheet" + strconv.Itoa(w.sheetNum))
			stream, err := w.excel.NewStreamWriter("Sheet" + strconv.Itoa(w.sheetNum))
			if err != nil {
				panic(err)
			}

			w.stream = stream
		}
	}()

	if len(w.headers) > 0 && w.rowNum == 0 {
		w.rowNum++
		w.stream.SetRow(
			"A"+strconv.Itoa(w.rowNum),
			array.Map(w.headers, func(item string) interface{} { return item }),
		)
	}

	w.rowNum++
	return w.stream.SetRow(
		"A"+strconv.Itoa(w.rowNum),
		array.Map(data, func(item string) interface{} { return item }),
	)
}

func (w *ExcelWriter) Close() error {
	if err := w.stream.Flush(); err != nil {
		return err
	}
	return w.excel.SaveAs(w.filename)
}
