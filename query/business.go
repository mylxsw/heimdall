package query

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/mylxsw/db-exporter/render"
	"github.com/mylxsw/go-utils/array"
)

// QueryWriteHandler is a function definition for query write handler
type QueryWriteHandler func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error)

// NewStreamingQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// The SQL query and the writing of the results are all streamed to reduce memory usage
func NewStreamingQueryWriter(dbConnStr string) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error) {
		if !array.In(format, []string{"csv", "json", "plain", "xlsx"}) {
			return 0, fmt.Errorf("streaming only supports csv/json/plain/xlsx format, the current format is %s", format)
		}

		db, err := sql.Open("mysql", dbConnStr)
		if err != nil {
			return 0, err
		}
		defer db.Close()

		colNames, stream, err := StreamQueryDB(db, sqlStr, args)
		if err != nil {
			return 0, err
		}

		return render.StreamingRender(output, format, noHeader, colNames, stream)
	}
}

// NewStandardQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// Querying and writing are done at one time, and all intermediate process data will be loaded into memory
func NewStandardQueryWriter(dbConnStr string, queryTimeout time.Duration) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error) {
		rs, err := Query(dbConnStr, sqlStr, args, queryTimeout)
		if err != nil {
			return 0, err
		}

		colNames, kvs := rs.SplitColumnAndKVs()

		writer, err := render.Render(format, noHeader, colNames, kvs, sqlStr)
		if err != nil {
			return 0, err
		}

		if _, err := writer.WriteTo(output); err != nil {
			return 0, err
		}

		return len(kvs), nil
	}
}
