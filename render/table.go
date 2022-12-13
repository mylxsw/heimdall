package render

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/heimdall/extracter"
)

func Table(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}) error {
	return render(writer, noHeader, cols, kvs, "table")
}

func Markdown(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}) error {
	return render(writer, noHeader, cols, kvs, "markdown")
}

func CSV(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}) error {
	// Write BOM header for UTF-8
	if _, err := writer.Write([]byte("\xEF\xBB\xBF")); err != nil {
		return err
	}

	return render(writer, noHeader, cols, kvs, "csv")
}

func HTML(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}) error {
	return render(writer, noHeader, cols, kvs, "html")
}

func render(writer io.Writer, noHeader bool, cols []extracter.Column, kvs []map[string]interface{}, typ string) error {
	t := table.NewWriter()
	t.SetOutputMirror(writer)
	if !noHeader {
		t.AppendHeader(array.Map(cols, func(col extracter.Column, _ int) interface{} { return col.Name }))
	}
	t.AppendRows(array.Map(kvs, func(kv map[string]interface{}, _ int) table.Row {
		row := table.Row{}
		for _, col := range cols {
			if v, ok := kv[col.Name]; ok && v != nil {
				row = append(row, v)
			} else {
				row = append(row, "")
			}
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
			if len(cols) > 1 {
				row = append(row, "Total")
				for i := 0; i < len(cols)-1; i++ {
					row = append(row, len(kvs))
				}
			} else {
				row = append(row, fmt.Sprintf("Total %d", len(kvs)))
			}

			t.AppendFooter(row, table.RowConfig{AutoMerge: true})
		}
		t.Render()
	}

	return nil
}
