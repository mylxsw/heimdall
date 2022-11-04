package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
)

type Writer interface {
	Write(data []string) error
	Close() error
}

func StreamRender(output string, format string, noHeader bool, colNames []string, stream <-chan map[string]interface{}) error {
	if format == "xlsx" {
		return streamRenderXlsx(output, noHeader, colNames, stream)
	}

	return streamRender(output, format, stream, noHeader, colNames)
}

func streamRender(output string, format string, stream <-chan map[string]interface{}, noHeader bool, colNames []string) error {
	startTime := time.Now()
	w := ternary.IfElseLazy(output != "", func() io.WriteCloser {
		f := must.Must(os.Create(output))
		// 写入 BOM 表头
		if format == "csv" {
			must.Must(f.WriteString("\xEF\xBB\xBF"))
		}

		return f
	}, func() io.WriteCloser {
		return os.Stdout
	})
	defer w.Close()

	var total int
	switch format {
	case "json":
		for item := range stream {
			total++
			must.Must(w.Write(must.Must(json.Marshal(item))))
			must.Must(w.Write([]byte("\n")))
		}
	case "csv":
		csvWriter := csv.NewWriter(w)
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

			must.Must(w.Write([]byte(fmt.Sprintln(strings.Join(lines, ", ")))))
		}
	}

	if output != "" {
		log.Debugf("write to %s, total %d records, %s elapsed", output, total, time.Since(startTime))
	}

	return nil
}

func streamRenderXlsx(output string, noHeader bool, colNames []string, stream <-chan map[string]interface{}) error {
	startTime := time.Now()
	if output == "" {
		panic("xlsx format must specify output file")
	}

	w := must.Must(NewExcelWriter(output, ternary.If(noHeader, []string{}, colNames)))
	defer w.Close()

	var total int
	for item := range stream {
		total++
		line := make([]string, 0)
		for _, colName := range colNames {
			line = append(line, resolveValue(item[colName]))
		}

		must.NoError(w.Write(line))
	}

	log.Debugf("write to %s, total %d records, %s elapsed", output, total, time.Since(startTime))

	return nil
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
