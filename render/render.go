package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
)

type Writer interface {
	Write(data []string) error
	Close() error
}

func StreamingRender(output io.Writer, format string, noHeader bool, colNames []string, stream <-chan map[string]interface{}) (int, error) {
	if format == "xlsx" {
		return streamRenderXlsx(output, noHeader, colNames, stream)
	}

	return streamingRender(output, format, stream, noHeader, colNames)
}

func streamingRender(output io.Writer, format string, stream <-chan map[string]interface{}, noHeader bool, colNames []string) (int, error) {
	var total int
	switch format {
	case "json":
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
		// Write BOM header for UTF-8
		if _, err := output.Write([]byte("\xEF\xBB\xBF")); err != nil {
			return 0, err
		}

		csvWriter := csv.NewWriter(output)
		defer csvWriter.Flush()

		if !noHeader {
			if err := csvWriter.Write(colNames); err != nil {
				return 0, err
			}
		}

		for item := range stream {
			total++
			line := make([]string, 0)
			for _, colName := range colNames {
				line = append(line, resolveValue(item[colName]))
			}

			if err := csvWriter.Write(line); err != nil {
				return 0, err
			}
		}
	default:
		for item := range stream {
			total++
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(
					lines,
					strings.ReplaceAll(ternary.IfLazy(
						noHeader,
						func() string { return resolveValue(item[colName]) },
						func() string { return fmt.Sprintf("%s=%v", colName, resolveValue(item[colName])) },
					), "\n", "\\n"),
				)
			}

			if _, err := output.Write([]byte(fmt.Sprintln(strings.Join(lines, ", ")))); err != nil {
				return 0, err
			}
		}
	}

	return total, nil
}

func streamRenderXlsx(output io.Writer, noHeader bool, colNames []string, stream <-chan map[string]interface{}) (total int, err error) {
	tmpFilename := createTempFilename() + ".xlsx"
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
		for _, colName := range colNames {
			line = append(line, resolveValue(item[colName]))
		}

		if err := w.Write(line); err != nil {
			return 0, err
		}
	}

	return total, nil
}

func resolveValue(value interface{}) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%v", value)
}

func Render(format string, noHeader bool, colNames []string, kvs []map[string]interface{}, sqlStr string) (*bytes.Buffer, error) {
	writer := bytes.NewBuffer(nil)

	switch format {
	case "json":
		return writer, JSON(writer, kvs)
	case "yaml":
		return writer, YAML(writer, kvs)
	case "table":
		return writer, Table(writer, noHeader, colNames, kvs)
	case "markdown":
		return writer, Markdown(writer, noHeader, colNames, kvs)
	case "csv":
		return writer, CSV(writer, noHeader, colNames, kvs)
	case "html":
		return writer, HTML(writer, noHeader, colNames, kvs)
	case "xlsx":
		return writer, XLSX(writer, noHeader, colNames, kvs)
	case "xml":
		return writer, XML(writer, colNames, kvs, sqlStr)
	default:
		for _, kv := range kvs {
			lines := make([]string, 0)
			for _, colName := range colNames {
				lines = append(lines, strings.ReplaceAll(ternary.IfLazy(
					noHeader,
					func() string { return resolveValue(kv[colName]) },
					func() string { return fmt.Sprintf("%s=%v", colName, resolveValue(kv[colName])) },
				), "\n", "\\n"))
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
	return filepath.Join(os.TempDir(), fmt.Sprintf("mysql-querier-%d-%d.tmp", time.Now().UnixNano(), rand.Intn(1000000000)))
}
