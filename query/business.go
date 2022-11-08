package query

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/mylxsw/db-exporter/render"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/must"
)

// QueryWriteHandler is a function definition for query write handler
type QueryWriteHandler func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) int

// NewStreamingQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// The SQL query and the writing of the results are all streamed to reduce memory usage
func NewStreamingQueryWriter(dbConnStr string) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) int {
		if !array.In(format, []string{"csv", "json", "plain", "xlsx"}) {
			panic(fmt.Sprintf("streaming only supports csv/json/plain/xlsx format, the current format is %s", format))
		}

		db := must.Must(sql.Open("mysql", dbConnStr))
		defer db.Close()

		colNames, stream := must.Must2(StreamQueryDB(db, sqlStr, args))
		return must.Must(render.StreamingRender(output, format, noHeader, colNames, stream))
	}
}

// NewStandardQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// Querying and writing are done at one time, and all intermediate process data will be loaded into memory
func NewStandardQueryWriter(dbConnStr string, queryTimeout time.Duration) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) int {
		colNames, kvs := must.Must(Query(dbConnStr, sqlStr, args, queryTimeout)).SplitColumnAndKVs()

		writer := render.Render(format, noHeader, colNames, kvs, sqlStr)
		must.Must(writer.WriteTo(output))

		return len(kvs)
	}
}
