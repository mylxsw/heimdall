package query

import (
	"context"
	"database/sql"
	"time"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/mysql-querier/extracter"
)

func Query(mysqlConnStr string, sqlStr string, queryTimeout time.Duration) (*extracter.Rows, error) {
	db, err := sql.Open("mysql", mysqlConnStr)
	if err != nil {
		return nil, err
	}

	defer db.Close()

	return QueryDB(db, sqlStr, queryTimeout)
}

func QueryDB(db *sql.DB, sqlStr string, queryTimeout time.Duration) (*extracter.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return extracter.Extract(rows)
}

func StreamQueryDB(db *sql.DB, sqlStr string) ([]string, <-chan map[string]interface{}, error) {
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, nil, err
	}

	cols, stream, err := extracter.ExtractStream(rows)

	colNames := array.Map(cols, func(col extracter.Column) string { return col.Name })
	return colNames, stream, err
}
