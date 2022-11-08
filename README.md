# DB-Exporter

DB-Exporter is a database query export tool specially designed for MySQL. Using it, you can directly export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text, etc.

## Command Line Options

The following command line options are supported：

- `-db` string
    MySQL database name
- `-debug`
    Enable debug mode
- `-format` string
    Output format: json/yaml/plain/table/csv/html/markdown/xlsx/xml (default "csv")
- `-host` string
    MySQL host (default "127.0.0.1")
- `-no-header`
    Do not write table header
- `-output` string
    Write output to a file, default output directly to STDOUT
- `-password` string
    MySQL password
- `-port` int
    MySQL port (default 3306)
- `-sql` string
    SQL query to execute, read from STDIN if not specified
- `-streaming`
    Whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/json/plain
- `-timeout` duration
    Query timeout, when the stream option is specified, this option is invalid (default 10s)
- `-user` string
    MySQL username (default "root")
- `-version`
    Output version info
- `-xlsx-max-row` int
    The maximum number of rows per sheet in an Excel file, including the row where the header is located (default 1048576)

## Examples

Export new businesses in the last 30 days to an Excel file

```bash
db-exporter -db example -host 127.0.0.1 -user root -password root \
      -sql "SELECT id, name AS '企业名称', address AS '企业地址', city_name AS '城市', district_name AS '区县', DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS '创建时间' FROM enterprise WHERE created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) ORDER BY id DESC" \
      -format csv \
      -output 最近30天新增企业列表.csv
```