package query

import "fmt"

func BuildConnStr(mysqlDB, mysqlUser, mysqlPassword, mysqlHost string, mysqlPort int) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?loc=Local&parseTime=true", mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDB)
}
