package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mylxsw/go-utils/must"
)

func StreamRender(w io.Writer, format string, colNames []string, stream <-chan map[string]interface{}) error {
	switch format {
	case "json":
		for item := range stream {
			must.Must(w.Write(must.Must(json.Marshal(item))))
			must.Must(w.Write([]byte("\n")))
		}
	case "csv":
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		must.NoError(csvWriter.Write(colNames))
		for item := range stream {
			line := make([]string, 0)
			for _, colName := range colNames {
				line = append(line, fmt.Sprintf("%v", item[colName]))
			}

			must.NoError(csvWriter.Write(line))
		}
	default:
		for item := range stream {
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(lines, strings.ReplaceAll(fmt.Sprintf("%s=%v", colName, item[colName]), "\n", "\\n"))
			}

			must.Must(w.Write([]byte(fmt.Sprintln(strings.Join(lines, ", ")))))
		}
	}

	return nil
}

func Render(format string, colNames []string, kvs []map[string]interface{}, sqlStr string) *bytes.Buffer {
	writer := bytes.NewBuffer(nil)

	switch format {
	case "json":
		JSON(writer, kvs)
	case "yaml":
		YAML(writer, kvs)
	case "table":
		Table(writer, colNames, kvs)
	case "markdown":
		Markdown(writer, colNames, kvs)
	case "csv":
		CSV(writer, colNames, kvs)
	case "html":
		HTML(writer, colNames, kvs)
	case "xlsx":
		XLSX(writer, colNames, kvs)
	case "xml":
		XML(writer, colNames, kvs, sqlStr)
	default:
		for _, kv := range kvs {
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(lines, strings.ReplaceAll(fmt.Sprintf("%s=%v", colName, kv[colName]), "\n", "\\n"))
			}

			writer.Write([]byte(fmt.Sprintln(strings.Join(lines, ", "))))
		}
	}

	return writer
}
