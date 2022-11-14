# DB-Tools

DB-Tools is a database tools specially designed for MySQL. Using it, you can directly import xlsx or csv file to database or export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text, etc.

## Command Line Options

**db-tools** support below commands

- **import** data from xlsx or csv file to database table
- **export** SQL query results to various file formats

### import

Using **import** command, you can import data from xlsx or csv file to a table.

The following command line options are supported：

- **--host value**, **-H value** MySQL host (default: "127.0.0.1")
- **--port value**, **-P value** MySQL port (default: 3306)
- **--user value**, **-u value** MySQL user (default: "root")
- **--password value**, **-p value** MySQL password
- **--database value**, **-d value** MySQL database
- **--debug**, **-D** Debug mode (default: false)
- **--input value**, **-i value** input excel or csv file path
- **--table value**, **-t value** target table name
- **--field value**, **-f value** *[ --field value, -f value ]* field map, eg: excel_field:db_field, this flag can be specified multiple times
- **--include value**, **-I value** *[ --include value, -I value ]* include fields, if set, only these fields will be imported, this flag can be specified multiple times
- **--exclude value**, **-E value** *[ --exclude value, -E value ]* exclude fields, if set, these fields will be ignored, this flag can be specified multiple times
- **--csv-sepertor value** csv file sepertor, default is ',' (default: ",")
- **--help**, **-h** show help (default: false)

### export

Using **export** command, you can export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text, etc. 

The following command line options are supported：

- **--host value**, **-H value** MySQL host (default: "127.0.0.1")
- **--port value**, **-P value** MySQL port (default: 3306)
- **--user value**, **-u value** MySQL user (default: "root")
- **--password value**, **-p value** MySQL password
- **--database value**, **-d value** MySQL database
- **--debug**, **-D** Debug mode (default: false)
- **--sql value**, **-s value** SQL statement
- **--format value**, **-f value** output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain (default: "csv")
- **--output value**, **-o value** write output to a file, default output directly to STDOUT
- **--streaming**, **-S** whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/xlsx/json/plain (default: false)
- **--no-header**, **-n** do not write table header (default: false)
- **--timeout value**, **-t value** query timeout, when the stream option is specified, this option is invalid (default: 0s)
- **--xlsx-max-row value** the maximum number of rows per sheet in an Excel file, including the row where the header is located (default: 1048576)
- **--help**, **-h** show help (default: false)

## Examples

Import a xlsx file to database table 

```bash
db-tools import --host 127.0.0.1 --port 3306 --database example --user root --password root \
    --table people \
    -field 区域:area \ 
    -field 姓名:name \
    -field 身份证号码:idcard \
    --input ~/Downloads/data.xlsx
```

Export new businesses in the last 30 days to an Excel file

```bash
db-tools export --database example --host 127.0.0.1 --user root --password root \
      --sql "SELECT id, name AS '企业名称', address AS '企业地址', city_name AS '城市', district_name AS '区县', DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS '创建时间' FROM enterprise WHERE created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) ORDER BY id DESC" \
      --streaming \
      --format xlsx \
      --output 最近30天新增企业列表.xlsx
```