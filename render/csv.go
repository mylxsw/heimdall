package render

import (
	"encoding/csv"
	"io"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/heimdall/extracter"
)

func streamRenderCSV(output io.Writer, stream <-chan map[string]interface{}, noHeader bool, cols []extracter.Column) (int, error) {
	return renderCSV(output, noHeader, cols, func(cb func(item map[string]interface{}) error) error {
		for item := range stream {
			if err := cb(item); err != nil {
				return err
			}
		}
		return nil
	})
}

func renderCSVAll(output io.Writer, kvs []map[string]interface{}, noHeader bool, cols []extracter.Column) (int, error) {
	return renderCSV(output, noHeader, cols, func(cb func(item map[string]interface{}) error) error {
		for _, item := range kvs {
			if err := cb(item); err != nil {
				return err
			}
		}

		return nil
	})
}

func renderCSV(output io.Writer, noHeader bool, cols []extracter.Column, cb func(cb func(item map[string]interface{}) error) error) (int, error) {
	var total int
	// Write BOM header for UTF-8
	if _, err := output.Write([]byte("\xEF\xBB\xBF")); err != nil {
		return 0, err
	}

	csvWriter := csv.NewWriter(output)
	defer csvWriter.Flush()

	if !noHeader {
		colNames := array.Map(cols, func(col extracter.Column, _ int) string { return col.Name })
		if err := csvWriter.Write(colNames); err != nil {
			return 0, err
		}
	}

	if err := cb(func(item map[string]interface{}) error {
		total++
		line := make([]string, 0)
		for _, col := range cols {
			line = append(line, resolveValue(col, item[col.Name]))
		}

		if err := csvWriter.Write(line); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return total, nil
}
