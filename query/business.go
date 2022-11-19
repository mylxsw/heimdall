package query

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/heimdall/render"
)

// QueryWriteHandler is a function definition for query write handler
type QueryWriteHandler func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error)

// NewStreamingQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// The SQL query and the writing of the results are all streamed to reduce memory usage
func NewStreamingQueryWriter(dbConnStr string, targetTableForSQLFormat string, connectTimeout time.Duration) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error) {
		if !array.In(format, []string{"csv", "json", "plain", "xlsx", "sql"}) {
			return 0, fmt.Errorf("streaming only supports csv/json/plain/xlsx/sql format, the current format is %s", format)
		}

		db, err := sql.Open("mysql", dbConnStr)
		if err != nil {
			return 0, err
		}
		defer db.Close()

		if connectTimeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
			defer cancel()

			if err := db.PingContext(ctx); err != nil {
				return 0, fmt.Errorf("database is unreached: %w", err)
			}
		}

		colNames, stream, err := StreamQueryDB(db, sqlStr, args)
		if err != nil {
			return 0, err
		}

		return render.StreamingRender(output, format, noHeader, colNames, stream, targetTableForSQLFormat)
	}
}

// NewStandardQueryWriter create a function that executes SQL in the database
// and writes the returned results to a file in the specified format.
// Querying and writing are done at one time, and all intermediate process data will be loaded into memory
func NewStandardQueryWriter(dbConnStr string, targetTableForSQLFormat string, connectTimeout time.Duration, queryTimeout time.Duration) QueryWriteHandler {
	return func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool) (int, error) {
		rs, err := Query(dbConnStr, sqlStr, args, connectTimeout, queryTimeout)
		if err != nil {
			return 0, err
		}

		colNames, kvs := rs.SplitColumnAndKVs()

		writer, err := render.Render(format, noHeader, colNames, kvs, sqlStr, targetTableForSQLFormat)
		if err != nil {
			return 0, err
		}

		if _, err := writer.WriteTo(output); err != nil {
			return 0, err
		}

		return len(kvs), nil
	}
}
