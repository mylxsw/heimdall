# mysql-querier

MySQL-Querier 是一款专为 MySQL 设计的数据库查询导出工具，使用它可以将 SQL 的查询结果导出为多种文件格式，目前支持 JSON/YAML/Markdown/CSV/XLSX/HTML/text 等。

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
-stream
    	是否使用流式输出，如果使用流式输出，则不会等待查询完成，而是在查询过程中逐行输出，输出格式 format 只支持 csv/json/plain
-timeout duration
      查询超时时间 (default 10s)
-user string
      MySQL 用户 (default "root")
-version
      输出版本信息
```

使用示例

```bash
# 导出近 30 天新增的企业到 Excel 文件
mysql-querier -db example -host 127.0.0.1 -user root -password root \
      -sql "SELECT id, name AS '企业名称', address AS '企业地址', city_name AS '城市', district_name AS '区县', DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS '创建时间' FROM enterprise WHERE created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) ORDER BY id DESC" \
      -format csv \
      -output 最近30天新增企业列表.csv
```