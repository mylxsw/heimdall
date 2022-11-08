package query

import (
	"context"
	"database/sql"
	"time"

	"github.com/mylxsw/db-exporter/extracter"
	"github.com/mylxsw/go-utils/array"
)

// Query query data from MySQL database, and return the result as a map, return all data at once
func Query(mysqlConnStr string, sqlStr string, args []interface{}, queryTimeout time.Duration) (*extracter.Rows, error) {
	db, err := sql.Open("mysql", mysqlConnStr)
	if err != nil {
		return nil, err
	}

	defer db.Close()

	return QueryDB(db, sqlStr, args, queryTimeout)
}

// QueryDB query data from MySQL database, and return the result as a map, return all data at once
func QueryDB(db *sql.DB, sqlStr string, args []interface{}, queryTimeout time.Duration) (*extracter.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return extracter.Extract(rows)
}

// StreamQuery query data from MySQL database, and return the result one by one using channel
func StreamQueryDB(db *sql.DB, sqlStr string, args []interface{}) ([]string, <-chan map[string]interface{}, error) {
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, nil, err
	}

	cols, stream, err := extracter.ExtractStream(rows)

	colNames := array.Map(cols, func(col extracter.Column) string { return col.Name })
	return colNames, stream, err
}
