package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/extracter"
)

type Writer interface {
	Write(data []string) error
	Close() error
}

func StreamingRender(output io.Writer, format string, noHeader bool, cols []extracter.Column, stream <-chan map[string]interface{}, targetTableForSQLFormat string) (int, error) {
	switch format {
	case "xlsx":
		return streamRenderXlsx(output, noHeader, cols, stream)
	case "json":
		var total int
		for item := range stream {
			total++
			if _, err := output.Write(must.Must(json.Marshal(item))); err != nil {
				return 0, err
			}
			if _, err := output.Write([]byte("\n")); err != nil {
				return 0, err
			}
		}
	case "csv":
		return streamRenderCSV(output, stream, noHeader, cols)
	case "sql":
		var total int
		for item := range stream {
			total++
			if _, err := output.Write([]byte(buildSQLInsertStr(targetTableForSQLFormat, cols, item))); err != nil {
				return 0, err
			}
		}
		return total, nil

	}

	var total int
	for item := range stream {
		total++
		lines := make([]string, 0)
		for _, col := range cols {
			lines = append(
				lines,
				ternary.IfLazy(
					noHeader,
					func() string { return extracter.Sanitize(resolveValue(col, item[col.Name])) },
					func() string {
						return fmt.Sprintf("%s=%v", col.Name, extracter.Sanitize(resolveValue(col, item[col.Name])))
					},
				),
			)
		}

		if _, err := output.Write([]byte(fmt.Sprintln(strings.Join(lines, ", ")))); err != nil {
			return 0, err
		}
	}

	return total, nil
}

func resolveValue(col extracter.Column, value interface{}) string {
	if value == nil {
		return ""
	}

	if v1, ok := value.(int64); ok {
		return fmt.Sprintf("%d", v1)
	}

	if v1, ok := value.(float64); ok {
		return strconv.FormatFloat(v1, 'f', -1, 64)
	}

	switch col.Type {
	case extracter.ColumnTypeDate:
		return value.(time.Time).Format("2006-01-02")
	case extracter.ColumnTypeDatetime, extracter.ColumnTypeTimestamp:
		if v1, ok := value.(time.Time); ok {
			return v1.Format("2006-01-02 15:04:05")
		}

		return ""
	}

	return fmt.Sprintf("%v", value)
}

func Render(format string, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}, sqlStr string, targetTableForSQLFormat string) (*bytes.Buffer, error) {
	writer := bytes.NewBuffer(nil)

	switch format {
	case "json":
		for _, item := range kvs {
			if err := JSON(writer, item); err != nil {
				return nil, err
			}
			fmt.Fprintf(writer, "\n")
		}

		return writer, nil
	case "yaml":
		return writer, YAML(writer, kvs)
	case "table":
		return writer, Table(writer, noHeader, cols, kvs)
	case "markdown":
		return writer, Markdown(writer, noHeader, cols, kvs)
	case "csv":
		_, err := renderCSVAll(writer, kvs, noHeader, cols)
		return writer, err
	case "html":
		return writer, HTML(writer, noHeader, cols, kvs)
	case "xlsx":
		return writer, XLSX(writer, noHeader, cols, kvs)
	case "xml":
		return writer, XML(writer, cols, kvs, sqlStr)
	case "sql":
		return writer, SQL(writer, targetTableForSQLFormat, cols, kvs, sqlStr)
	default:
		for _, kv := range kvs {
			lines := make([]string, 0)
			for _, col := range cols {
				lines = append(lines, ternary.IfLazy(
					noHeader,
					func() string { return extracter.Sanitize(resolveValue(col, kv[col.Name])) },
					func() string {
						return fmt.Sprintf("%s=%v", col.Name, extracter.Sanitize(resolveValue(col, kv[col.Name])))
					},
				))
			}

			_, err := writer.Write([]byte(fmt.Sprintln(strings.Join(lines, ", "))))
			if err != nil {
				return nil, err
			}
		}
	}

	return writer, nil
}

func createTempFilename() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("heimdall-export-%d-%d.tmp", time.Now().UnixNano(), rand.Intn(1000000000)))
}
