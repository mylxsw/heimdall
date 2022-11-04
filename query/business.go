package query

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/str"
	"github.com/mylxsw/mysql-querier/render"
)

type QueryWriteHandler func(sqlStr string, fields []string, format, output string, noHeader bool)

func NewStreamQueryWriter(dbConnStr string) QueryWriteHandler {
	return func(sqlStr string, fields []string, format, output string, noHeader bool) {
		if !array.In(format, []string{"csv", "json", "plain", "xlsx"}) {
			panic(fmt.Sprintf("stream output only support csv/json/plain/xlsx format, current format is %s", format))
		}

		db := must.Must(sql.Open("mysql", dbConnStr))
		defer db.Close()

		colNames, stream := must.Must2(StreamQueryDB(db, sqlStr))

		if len(fields) > 0 {
			colNames = array.Filter(fields, func(item string) bool {
				return str.InIgnoreCase(item, colNames)
			})
		}

		must.NoError(render.StreamRender(output, format, noHeader, colNames, stream))
	}
}

func NewStandardQueryWriter(dbConnStr string, queryTimeout time.Duration) QueryWriteHandler {
	return func(sqlStr string, fields []string, format, output string, noHeader bool) {
		startTime := time.Now()

		colNames, kvs := must.Must(Query(dbConnStr, sqlStr, queryTimeout)).SplitColumnAndKVs()
		if len(fields) > 0 {
			colNames = array.Filter(fields, func(item string) bool {
				return str.InIgnoreCase(item, colNames)
			})

			kvs = array.Map(kvs, func(item map[string]interface{}) map[string]interface{} {
				res := make(map[string]interface{})
				for k, v := range item {
					if str.InIgnoreCase(k, fields) {
						res[k] = v
					}
				}

				return res
			})
		}

		writer := render.Render(format, noHeader, colNames, kvs, sqlStr)
		if output != "" {
			if err := os.WriteFile(output, writer.Bytes(), os.ModePerm); err != nil {
				panic(err)
			}

			log.Debugf("write to %s, total %d records, %s elapsed", output, len(kvs), time.Since(startTime))
		} else {
			_, _ = writer.WriteTo(os.Stdout)
		}
	}
}
