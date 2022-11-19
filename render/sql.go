package render

import (
	"fmt"
	"io"
	"strings"
)

func buildSQLInsertStr(targetTableForSQLFormat string, colNames []string, item map[string]interface{}) string {
	colValues := make([]string, 0)
	for _, colName := range colNames {
		colValues = append(colValues, fmt.Sprintf("'%s'", sanitize(resolveValue(item[colName]))))
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n", targetTableForSQLFormat, strings.Join(colNames, ","), strings.Join(colValues, ","))
}

func SQL(w io.Writer, targetTable string, colNames []string, data []map[string]interface{}, sqlStr string) error {
	for _, kv := range data {
		_, err := w.Write([]byte(buildSQLInsertStr(targetTable, colNames, kv)))
		if err != nil {
			return err
		}
	}

	return nil
}

// https://github.com/jamf/go-mysqldump/blob/master/sanitize.go
var lazyMySQLReplacer *strings.Replacer

// sanitize MySQL based on
// https://dev.mysql.com/doc/refman/8.0/en/string-literals.html table 9.1
// needs to be placed in either a single or a double quoted string
func sanitize(input string) string {
	if lazyMySQLReplacer == nil {
		lazyMySQLReplacer = strings.NewReplacer(
			"\x00", "\\0",
			"'", "\\'",
			"\"", "\\\"",
			"\b", "\\b",
			"\n", "\\n",
			"\r", "\\r",
			// "\t", "\\t", Tab literals are acceptable in reads
			"\x1A", "\\Z", // ASCII 26 == x1A
			"\\", "\\\\",
			// "%", "\\%",
			// "_", "\\_",
		)
	}
	return lazyMySQLReplacer.Replace(input)
}
