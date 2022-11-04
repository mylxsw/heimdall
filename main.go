package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/mysql-querier/query"
	"github.com/mylxsw/mysql-querier/render"

	_ "github.com/go-sql-driver/mysql"
)

var (
	// GitCommit Git 版本
	GitCommit string
	// Version 应用版本
	Version string
)
var outputVersion bool

var mysqlHost, mysqlUser, mysqlPassword, mysqlDB string
var mysqlPort int
var sqlStr string
var format, output string
var queryTimeout time.Duration
var fields string
var streamOutput, noHeader bool
var debug bool

func main() {

	flag.StringVar(&mysqlHost, "host", "127.0.0.1", "MySQL 主机地址")
	flag.StringVar(&mysqlDB, "db", "", "MySQL 数据库名")
	flag.StringVar(&mysqlPassword, "password", "", "MySQL 密码")
	flag.StringVar(&mysqlUser, "user", "root", "MySQL 用户")
	flag.IntVar(&mysqlPort, "port", 3306, "MySQL 端口")
	flag.StringVar(&sqlStr, "sql", "", "要执行的 SQL 查询语句，如果不指定则从标准输入读取")
	flag.StringVar(&format, "format", "csv", "输出格式： json/yaml/plain/table/csv/html/markdown/xlsx/xml")
	flag.StringVar(&output, "output", "", "将输出写入到文件，默认直接输出到标准输出")
	flag.BoolVar(&outputVersion, "version", false, "输出版本信息")
	flag.DurationVar(&queryTimeout, "timeout", 10*time.Second, "查询超时时间，当指定 stream 选项时，该选项无效")
	flag.StringVar(&fields, "fields", "", "查询字段列表，默认为全部字段，字段之间使用英文逗号分隔")
	flag.BoolVar(&streamOutput, "stream", false, "是否使用流式输出，如果使用流式输出，则不会等待查询完成，而是在查询过程中逐行输出，输出格式 format 只支持 csv/json/plain")
	flag.BoolVar(&noHeader, "no-header", false, "不输出表头")
	flag.BoolVar(&debug, "debug", false, "是否开启调试模式")
	flag.IntVar(&render.MaxRowNumInSheet, "xlsx-max-row", 1048576, "Excel 文件每个 Sheet 最大的行数，包含表头")

	flag.Parse()

	if !debug {
		log.All().LogLevel(level.Info)
	}

	if outputVersion {
		fmt.Printf("Version=%s, GitCommit=%s\n", Version, GitCommit)
		return
	}

	if sqlStr == "" {
		sqlStr = readStdin()
	}

	allowFields := array.Filter(strings.Split(fields, ","), func(item string) bool { return strings.TrimSpace(item) != "" })

	dbConnStr := query.BuildConnStr(mysqlDB, mysqlUser, mysqlPassword, mysqlHost, mysqlPort)
	handler := ternary.IfLazy(
		streamOutput,
		func() query.QueryWriteHandler { return query.NewStreamQueryWriter(dbConnStr) },
		func() query.QueryWriteHandler { return query.NewStandardQueryWriter(dbConnStr, queryTimeout) },
	)

	handler(sqlStr, allowFields, format, output, noHeader)
}

func readStdin() string {
	reader := bufio.NewReader(os.Stdin)
	var result []rune
	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}

		result = append(result, input)
	}

	return string(result)
}
