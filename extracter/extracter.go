package extracter

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"

	"github.com/mylxsw/go-utils/array"

	"github.com/mylxsw/coll"
)

// Column is a sql column info
type Column struct {
	Name     string       `json:"name"`
	Type     string       `json:"type"`
	ScanType reflect.Type `json:"scan_type"`
}

// Rows sql rows object
type Rows struct {
	Columns  []Column                 `json:"columns"`
	DataSets []map[string]interface{} `json:"data_sets"`
}

func (rows *Rows) SplitColumnAndKVs() (columnNames []string, kvs []map[string]interface{}) {
	return array.Map(rows.Columns, func(col Column) string {
		return col.Name
	}), rows.DataSets
}

// Extract export sql rows to Rows object one by one
func ExtractStream(rows *sql.Rows) ([]Column, <-chan map[string]interface{}, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}

	columnNames := array.Map(types, func(t *sql.ColumnType) Column {
		return Column{
			Name:     t.Name(),
			Type:     t.DatabaseTypeName(),
			ScanType: t.ScanType(),
		}
	})

	dataSetChan := make(chan map[string]interface{})
	go func() {
		defer close(dataSetChan)

		for rows.Next() {
			var data = array.Map(types, func(item *sql.ColumnType) interface{} {
				var tt interface{}
				return &tt
			})

			if err := rows.Scan(data...); err != nil {
				panic(err)
			}

			rowRaw := coll.MustNew(data).Map(func(k *interface{}, index int) interface{} {
				if k == nil || *k == nil {
					return nil
				}

				res := fmt.Sprintf("%s", *k)
				// TODO The conversion of all SQL types is mainly implemented here,
				/// and only a few commonly used ones are currently implemented
				switch types[index].DatabaseTypeName() {
				case "INT", "TINYINT", "BIGINT", "MEDIUMINT", "SMALLINT":
					intRes, _ := strconv.Atoi(res)
					return intRes
				case "DECIMAL":
					floatRes, _ := strconv.ParseFloat(res, 64)
					return floatRes
				case "BIT":
					return (*k).([]uint8)[0]
				default:
				}

				return res
			}).Items().([]interface{})

			rowData := make(map[string]interface{}, 0)
			for i, col := range columnNames {
				rowData[col.Name] = rowRaw[i]
			}

			dataSetChan <- rowData
		}
	}()

	return columnNames, dataSetChan, nil
}

// Extract export sql rows to Rows object
func Extract(rows *sql.Rows) (*Rows, error) {
	columnNames, dataSetChan, err := ExtractStream(rows)
	if err != nil {
		return nil, err
	}

	dataSets := make([]map[string]interface{}, 0)
	for row := range dataSetChan {
		dataSets = append(dataSets, row)
	}

	res := Rows{Columns: columnNames, DataSets: dataSets}
	return &res, nil
}
