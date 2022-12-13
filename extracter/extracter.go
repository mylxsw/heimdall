package extracter

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/mylxsw/go-utils/array"
)

// Column is a sql column info
type Column struct {
	Name     string       `json:"name"`
	Type     ColumnType   `json:"type"`
	ScanType reflect.Type `json:"scan_type"`
}

// Rows sql rows object
type Rows struct {
	Columns  []Column                 `json:"columns"`
	DataSets []map[string]interface{} `json:"data_sets"`
}

// Extract export sql rows to Rows object one by one
//
//	CREATE TABLE demo
//	(
//		id                  INT AUTO_INCREMENT,
//		col_bit             BIT                              NULL,
//		co_blob             BLOB                             NULL,
//		col_char            CHAR                             NULL,
//		col_varchar         VARCHAR(255)                     NULL,
//		col_binary          BINARY(1000)                     NULL,
//		col_varbinary       VARBINARY(1000)                  NULL,
//		col_date            DATE                             NULL,
//		col_datetime        DATETIME                         NULL,
//		col_timestamp       TIMESTAMP                        NULL,
//		col_time            TIME                             NULL,
//		col_year            YEAR                             NULL,
//		col_decimal         DECIMAL                          NULL,
//		col_double          DOUBLE                           NULL,
//		col_float           FLOAT                            NULL,
//		col_enum            ENUM ('S', 'M', 'L')             NULL,
//		col_geometry        GEOMETRY                         NULL,
//		col_tinyint         TINYINT                          NULL,
//		col_bigint          BIGINT                           NULL,
//		col_smallint_unsign SMALLINT UNSIGNED                NULL,
//		col_json            JSON                             NULL,
//		col_set             SET ('S', 'M', 'L', 'XL', 'XXL') NULL,
//		CONSTRAINT demo_pk
//			PRIMARY KEY (id)
//	);
func ExtractStream(rows *sql.Rows) ([]Column, <-chan map[string]interface{}, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}

	columnNames := array.Map(types, func(t *sql.ColumnType, _ int) Column {
		return Column{
			Name:     t.Name(),
			Type:     ColumnType(t.DatabaseTypeName()),
			ScanType: t.ScanType(),
		}
	})

	dataSetChan := make(chan map[string]interface{})
	go func() {
		defer close(dataSetChan)

		for rows.Next() {
			var data = array.Map(types, func(item *sql.ColumnType, _ int) interface{} {
				return reflect.New(reflectColumnType(item)).Interface()
			})

			if err := rows.Scan(data...); err != nil {
				panic(err)
			}

			rowRaw := array.Map(data, func(k interface{}, index int) interface{} {
				return parseValue(k, types[index])
			})

			rowData := make(map[string]interface{}, 0)
			for i, col := range columnNames {
				rowData[col.Name] = rowRaw[i]
			}

			dataSetChan <- rowData
		}
	}()

	return columnNames, dataSetChan, nil
}

type ColumnType string

const (
	ColumnTypeBit               ColumnType = "BIT"
	ColumnTypeTinyblob          ColumnType = "TINYBLOB"
	ColumnTypeMediumblob        ColumnType = "MEDIUMBLOB"
	ColumnTypeBlob              ColumnType = "BLOB"
	ColumnTypeLongblob          ColumnType = "LONGBLOB"
	ColumnTypeBinary            ColumnType = "BINARY"
	ColumnTypeVarbinary         ColumnType = "VARBINARY"
	ColumnTypeDate              ColumnType = "DATE"
	ColumnTypeDatetime          ColumnType = "DATETIME"
	ColumnTypeTime              ColumnType = "TIME"
	ColumnTypeTimestamp         ColumnType = "TIMESTAMP"
	ColumnTypeDecimal           ColumnType = "DECIMAL"
	ColumnTypeDouble            ColumnType = "DOUBLE"
	ColumnTypeFloat             ColumnType = "FLOAT"
	ColumnTypeTinyint           ColumnType = "TINYINT"
	ColumnTypeSmallint          ColumnType = "SMALLINT"
	ColumnTypeMediumint         ColumnType = "MEDIUMINT"
	ColumnTypeInt               ColumnType = "INT"
	ColumnTypeBigint            ColumnType = "BIGINT"
	ColumnTypeYear              ColumnType = "YEAR"
	ColumnTypeUnsignedTinyint   ColumnType = "UNSIGNED TINYINT"
	ColumnTypeUnsignedSmallint  ColumnType = "UNSIGNED SMALLINT"
	ColumnTypeUnsignedMediumint ColumnType = "UNSIGNED MEDIUMINT"
	ColumnTypeUnsignedInt       ColumnType = "UNSIGNED INT"
	ColumnTypeUnsignedBigint    ColumnType = "UNSIGNED BIGINT"
	ColumnTypeGeometry          ColumnType = "GEOMETRY"
	ColumnTypeJson              ColumnType = "JSON"
	ColumnTypeTinytext          ColumnType = "TINYTEXT"
	ColumnTypeText              ColumnType = "TEXT"
	ColumnTypeMediumtext        ColumnType = "MEDIUMTEXT"
	ColumnTypeLongtext          ColumnType = "LONGTEXT"
	ColumnTypeChar              ColumnType = "CHAR"
	ColumnTypeVarchar           ColumnType = "VARCHAR"
	ColumnTypeSet               ColumnType = "SET"
	ColumnTypeEnum              ColumnType = "ENUM"
)

func parseValue(k interface{}, typ *sql.ColumnType) interface{} {
	// BIT,
	// TINYBLOB, MEDIUMBLOB, BLOB, LONGBLOB,
	// BINARY, VARBINARY,
	// DATE, DATETIME, TIME, TIMESTAMP,
	// DECIMAL, DOUBLE, FLOAT,
	// TINYINT, SMALLINT, MEDIUMINT, INT, BIGINT, YEAR
	// UNSIGNED TINYINT, UNSIGNED SMALLINT, UNSIGNED MEDIUMINT, UNSIGNED INT, UNSIGNED BIGINT,
	// GEOMETRY,
	// JSON,
	// TINYTEXT, TEXT, MEDIUMTEXT, LONGTEXT,
	// CHAR, VARCHAR,
	// SET
	// ENUM,
	switch s := k.(type) {
	case nil:
		return nil
	case *sql.NullInt64:
		if s.Valid {
			return s.Int64
		}

		return nil
	case *sql.NullFloat64:
		if s.Valid {
			return s.Float64
		}
		return nil
	case *sql.RawBytes:
		if len(*s) > 0 {
			switch ColumnType(typ.DatabaseTypeName()) {
			case ColumnTypeBinary, ColumnTypeVarbinary,
				ColumnTypeTinyblob, ColumnTypeMediumblob, ColumnTypeBlob, ColumnTypeLongblob:
				return fmt.Sprintf("_binary '%s'", Sanitize(string(*s)))
			}

			return string(*s)
		}

		return nil
	case *sql.NullString:
		if s.Valid {
			return s.String
		}

		return nil
	case *sql.NullTime:
		if s.Valid {
			return s.Time
		}

		return nil
	case **interface{}:
		if s == nil || *s == nil {
			return nil
		}

		return **s
	}

	return fmt.Sprintf("%v", k)
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

func reflectColumnType(tp *sql.ColumnType) reflect.Type {
	switch tp.ScanType().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.TypeOf(sql.NullInt64{})
	case reflect.Float32, reflect.Float64:
		return reflect.TypeOf(sql.NullFloat64{})
	case reflect.String:
		return reflect.TypeOf(sql.NullString{})
	}

	switch tp.DatabaseTypeName() {
	case "BLOB", "BINARY":
		return reflect.TypeOf(sql.RawBytes{})
	case "VARCHAR", "TEXT", "DECIMAL", "JSON":
		return reflect.TypeOf(sql.NullString{})
	case "BIGINT", "TINYINT", "INT":
		return reflect.TypeOf(sql.NullInt64{})
	case "DOUBLE":
		return reflect.TypeOf(sql.NullFloat64{})
	case "DATETIME", "TIMESTAMP", "DATE":
		return reflect.TypeOf(sql.NullTime{})
	}

	return tp.ScanType()
}
