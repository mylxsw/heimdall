package query

import "fmt"

// BuildConnStr build a MySQL database connection string
func BuildConnStr(mysqlDB, mysqlUser, mysqlPassword, mysqlHost string, mysqlPort int) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?loc=Local&parseTime=true", mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDB)
}
