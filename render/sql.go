package render

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/heimdall/extracter"
)

func buildSQLInsertStr(targetTableForSQLFormat string, cols []extracter.Column, item map[string]interface{}) string {
	colValues := make([]string, 0)
	for _, col := range cols {
		colValues = append(colValues, buildSQLValue(col, item[col.Name]))
	}

	colNames := array.Map(cols, func(col extracter.Column) string { return col.Name })
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", targetTableForSQLFormat, strings.Join(colNames, ","), strings.Join(colValues, ","))
}

func buildSQLValue(col extracter.Column, value interface{}) string {
	if value == nil {
		return "NULL"
	}

	if v1, ok := value.(int64); ok {
		return fmt.Sprintf("%d", v1)
	}

	if v1, ok := value.(float64); ok {
		return strconv.FormatFloat(v1, 'f', -1, 64)
	}

	switch col.Type {
	case extracter.ColumnTypeDate:
		return fmt.Sprintf("'%s'", value.(time.Time).Format("2006-01-02"))
	case extracter.ColumnTypeDatetime, extracter.ColumnTypeTimestamp:
		return fmt.Sprintf("'%s'", value.(time.Time).Format("2006-01-02 15:04:05"))
	case extracter.ColumnTypeBinary, extracter.ColumnTypeVarbinary:
		return resolveValue(col, value)
	case extracter.ColumnTypeTinyblob, extracter.ColumnTypeMediumblob,
		extracter.ColumnTypeBlob, extracter.ColumnTypeLongblob:
		return resolveValue(col, value)
	case extracter.ColumnTypeDecimal:
		return resolveValue(col, value)
	}

	return fmt.Sprintf("'%s'", extracter.Sanitize(resolveValue(col, value)))
}

func SQL(w io.Writer, targetTable string, cols []extracter.Column, data []map[string]interface{}, sqlStr string) error {
	for _, kv := range data {
		_, err := w.Write([]byte(buildSQLInsertStr(targetTable, cols, kv)))
		if err != nil {
			return err
		}
	}

	return nil
}
