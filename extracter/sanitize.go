package extracter

import "strings"

// https://github.com/jamf/go-mysqldump/blob/master/sanitize.go
var lazyMySQLReplacer *strings.Replacer

// Sanitize MySQL based on
// https://dev.mysql.com/doc/refman/8.0/en/string-literals.html table 9.1
// needs to be placed in either a single or a double quoted string
func Sanitize(input string) string {
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
