package query

import (
	"context"
	"database/sql"
	"time"

	"github.com/mylxsw/mysql-querier/extracter"
)

func Query(mysqlConnStr string, sqlStr string, queryTimeout time.Duration) (*extracter.Rows, error) {
	db, err := sql.Open("mysql", mysqlConnStr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	return extracter.Extract(rows)
}
