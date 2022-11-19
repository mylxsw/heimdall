package render

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/extracter"
	"github.com/xuri/excelize/v2"
)

func XLSX(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}) error {
	exf := excelize.NewFile()
	exfCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	lineNo := 0

	if !noHeader {
		lineNo++
		for i, col := range cols {
			if err := exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[i], lineNo), col.Name); err != nil {
				return err
			}
		}
	}

	for _, kv := range kvs {
		lineNo++
		for j, col := range cols {
			if err := exf.SetCellValue("Sheet1", fmt.Sprintf("%s%d", exfCols[j], lineNo), kv[col.Name]); err != nil {
				return err
			}
		}
	}

	return exf.Write(writer)
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

func (w *ExcelWriter) Write(data []string) (err error) {
	defer func() {
		if w.rowNum >= MaxRowNumInSheet {
			w.rowNum = 0
			if err1 := w.stream.Flush(); err != nil {
				err = err1
				return
			}

			w.sheetNum++
			w.excel.NewSheet("Sheet" + strconv.Itoa(w.sheetNum))
			stream, err1 := w.excel.NewStreamWriter("Sheet" + strconv.Itoa(w.sheetNum))
			if err != nil {
				err = err1
				return
			}

			w.stream = stream
		}
	}()

	if len(w.headers) > 0 && w.rowNum == 0 {
		w.rowNum++
		if err := w.stream.SetRow(
			"A"+strconv.Itoa(w.rowNum),
			array.Map(w.headers, func(item string) interface{} { return item }),
		); err != nil {
			return err
		}
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

func streamRenderXlsx(output io.Writer, noHeader bool, cols []extracter.Column, stream <-chan map[string]interface{}) (total int, err error) {
	tmpFilename := createTempFilename() + ".xlsx"

	colNames := array.Map(cols, func(col extracter.Column) string { return col.Name })
	w, err := NewExcelWriter(tmpFilename, ternary.If(noHeader, []string{}, colNames))
	if err != nil {
		return 0, err
	}

	defer func() {
		if err1 := w.Close(); err1 != nil {
			err = err1
			return
		}

		f, err1 := os.Open(tmpFilename)
		if err1 != nil {
			err = err1
			return
		}
		defer func() {
			_ = f.Close()
			_ = os.Remove(tmpFilename)
		}()

		if _, err1 = io.Copy(output, f); err1 != nil {
			err = err1
			return
		}
	}()

	for item := range stream {
		total++
		line := make([]string, 0)
		for _, col := range cols {
			line = append(line, resolveValue(col, item[col.Name]))
		}

		if err := w.Write(line); err != nil {
			return 0, err
		}
	}

	return total, nil
}
