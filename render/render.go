package render

import (
	"bytes"
	"fmt"
	"strings"
)

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
