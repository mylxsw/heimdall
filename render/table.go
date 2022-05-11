package render

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mylxsw/go-utils/array"
)

func Table(writer io.Writer, colNames []string, kvs []map[string]interface{}) {
	render(writer, colNames, kvs, "table")
}

func Markdown(writer io.Writer, colNames []string, kvs []map[string]interface{}) {
	render(writer, colNames, kvs, "markdown")
}

func CSV(writer io.Writer, colNames []string, kvs []map[string]interface{}) {
	render(writer, colNames, kvs, "csv")
}

func HTML(writer io.Writer, colNames []string, kvs []map[string]interface{}) {
	render(writer, colNames, kvs, "html")
}

func render(writer io.Writer, colNames []string, kvs []map[string]interface{}, typ string) {
	t := table.NewWriter()
	t.SetOutputMirror(writer)
	t.AppendHeader(array.Map(colNames, func(name string) interface{} { return name }))
	t.AppendRows(array.Map(kvs, func(kv map[string]interface{}) table.Row {
		row := table.Row{}
		for _, colName := range colNames {
			row = append(row, kv[colName])
		}

		return row
	}))

	switch typ {
	case "markdown":
		t.RenderMarkdown()
	case "html":
		t.RenderHTML()
	case "csv":
		t.RenderCSV()
	default:
		if len(kvs) > 10 {
			row := table.Row{}
			if len(colNames) > 1 {
				row = append(row, "Total")
				for i := 0; i < len(colNames)-1; i++ {
					row = append(row, len(kvs))
				}
			} else {
				row = append(row, fmt.Sprintf("Total %d", len(kvs)))
			}

			t.AppendFooter(row, table.RowConfig{AutoMerge: true})
		}
		t.Render()
	}
}
