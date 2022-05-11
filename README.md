# mysql-querier

MySQL-Querier is a small tool used to execute SQL query statements in MySQL and output the returned results in JSON/YAML/Markdown/CSV/XLSX/HTML/text lines

```bash
-db string
      MySQL 数据库名
-fields string
      查询字段列表，默认为全部字段，字段之间使用英文逗号分隔
-format string
      输出格式： json/yaml/plain/table/csv/html/markdown/xlsx/xml (default "table")
-host string
      MySQL 主机地址 (default "127.0.0.1")
-output string
      将输出写入到文件，默认直接输出到标准输出
-password string
      MySQL 密码
-port int
      MySQL 端口 (default 3306)
-sql string
      要执行的 SQL 查询语句，如果不指定则从标准输入读取
-timeout duration
      查询超时时间 (default 10s)
-user string
      MySQL 用户 (default "root")
-version
      输出版本信息
```
