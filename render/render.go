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
	if format == "csv" {
		// Write BOM header for UTF-8
		must.Must(output.Write([]byte("\xEF\xBB\xBF")))
	}

	var total int
	switch format {
	case "json":
		for item := range stream {
			total++
			must.Must(output.Write(must.Must(json.Marshal(item))))
			must.Must(output.Write([]byte("\n")))
		}
	case "csv":
		csvWriter := csv.NewWriter(output)
		defer csvWriter.Flush()

		if !noHeader {
			must.NoError(csvWriter.Write(colNames))
		}

		for item := range stream {
			total++
			line := make([]string, 0)
			for _, colName := range colNames {
				line = append(line, resolveValue(item[colName]))
			}

			must.NoError(csvWriter.Write(line))
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

			must.Must(output.Write([]byte(fmt.Sprintln(strings.Join(lines, ", ")))))
		}
	}

	return total, nil
}

func streamRenderXlsx(output io.Writer, noHeader bool, colNames []string, stream <-chan map[string]interface{}) (int, error) {
	tmpFilename := createTempFilename() + ".xlsx"
	w := must.Must(NewExcelWriter(tmpFilename, ternary.If(noHeader, []string{}, colNames)))
	defer func() {
		must.NoError(w.Close())
		must.Must(io.Copy(output, must.Must(os.Open(tmpFilename))))
		_ = os.Remove(tmpFilename)
	}()

	var total int
	for item := range stream {
		total++
		line := make([]string, 0)
		for _, colName := range colNames {
			line = append(line, resolveValue(item[colName]))
		}

		must.NoError(w.Write(line))
	}

	return total, nil
}

func resolveValue(value interface{}) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%v", value)
}

func Render(format string, noHeader bool, colNames []string, kvs []map[string]interface{}, sqlStr string) *bytes.Buffer {
	writer := bytes.NewBuffer(nil)

	switch format {
	case "json":
		JSON(writer, kvs)
	case "yaml":
		YAML(writer, kvs)
	case "table":
		Table(writer, noHeader, colNames, kvs)
	case "markdown":
		Markdown(writer, noHeader, colNames, kvs)
	case "csv":
		CSV(writer, noHeader, colNames, kvs)
	case "html":
		HTML(writer, noHeader, colNames, kvs)
	case "xlsx":
		XLSX(writer, noHeader, colNames, kvs)
	case "xml":
		XML(writer, colNames, kvs, sqlStr)
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

			writer.Write([]byte(fmt.Sprintln(strings.Join(lines, ", "))))
		}
	}

	return writer
}

func createTempFilename() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("mysql-querier-%d-%d.tmp", time.Now().UnixNano(), rand.Intn(1000000000)))
}
