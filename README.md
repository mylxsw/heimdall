# Heimdall [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


![MariaDB](https://img.shields.io/badge/MariaDB-003545?style=for-the-badge&logo=mariadb&logoColor=white) ![MySQL](https://img.shields.io/badge/mysql-%2300f.svg?style=for-the-badge&logo=mysql&logoColor=white) ![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white) 


Heimdall is a database tools specially designed for MySQL. Using it, you can directly import xlsx or csv file to database or export SQL query results to various file formats, convert xlsx/csv to other formats, query xlsx/csv file using sql. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/SQL/text, etc.

## Command Line Options

**heimdall** support below commands

- **import** (aka **load**) data from xlsx or csv file to database table
- **export** (aka **query**) SQL query results to various file formats
- **fly** (aka **query-file**) query data from input file using sql directly
- **convert** convert data from xlsx/csv to other formats: csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql

### import/load

Using **import/load** command, you can import data from xlsx or csv file to a table.

```bash
heimdall import --tx --database example --table users \
    --file users.csv --file users.xlsx \
    --field 姓名:name \
    --field 年龄:age
```

The following command line options are supported：

- **--host value**, **-H value** MySQL host (default: "127.0.0.1")
- **--port value**, **-P value** MySQL port (default: 3306)
- **--user value**, **-u value** MySQL user (default: "root")
- **--password value**, **-p value** MySQL password
- **--database value**, **-d value** MySQL database
- **--connect-timeout value** database connect timeout (default: 3s)
- **--debug**, **-D** Debug mode (default: false)
- **--file value**, **-i value**, **--input value** *[ --file value, -i value, --input value ]* input excel or csv file path, this flag can be specified multiple times for importing multiple files at the same time
- **--table value**, **-t value** target table name
- **--field value**, **-f value** *[ --field value, -f value ]* field map, eg: excel_field:db_field, this flag can be specified multiple times
- **--include value**, **-I value** *[ --include value, -I value ]* include fields, if set, only these fields will be imported, this flag can be specified multiple times
- **--exclude value**, **-E value** *[ --exclude value, -E value ]* exclude fields, if set, these fields will be ignored, this flag can be specified multiple times
- **--csv-sepertor value** csv file sepertor, default is ',' (default: ",")
- **--tx**, **-T** import data using transaction, all success or all failure, only work with InnoDB or other engines that support transaction (default: false)
- **--dry-run** perform import tests to verify correctness of imported files, but do not commit transactions, only work with InnoDB or other engines that support transaction (default: false)
- **--help**, **-h** show help (default: false)

### export/query

Using **export/query** command, you can export SQL query results to various file formats. Currently, it supports JSON/YAML/Markdown/CSV/XLSX/HTML/text, etc. 

```bash
heimdall export --database example --format json --sql 'select * from users'
```

The following command line options are supported：

- **--host value**, **-H value** MySQL host (default: "127.0.0.1")
- **--port value**, **-P value** MySQL port (default: 3306)
- **--user value**, **-u value** MySQL user (default: "root")
- **--password value**, **-p value** MySQL password
- **--database value**, **-d value** MySQL database
- **--connect-timeout value** database connect timeout (default: 3s)
- **--debug**, **-D** Debug mode (default: false)
- **--sql value**, **-s value** SQL statement
- **--format value**, **-f value** output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (default: "csv")
- **--output value**, **-o value** write output to a file, default output directly to STDOUT
- **--streaming**, **-S** whether to use streaming output, if using streaming output, it will not wait for the query to complete, but output line by line during the query process. The output format only supports csv/xlsx/json/plain/sql (default: false)
- **--no-header**, **-n** do not write table header (default: false)
- **--query-timeout value**, **-t value** query timeout, when the stream option is specified, this option is invalid (default: 2m0s)
- **--xlsx-max-row value** the maximum number of rows per sheet in an Excel file, including the row where the header is located (default: 1048576)
- **--table value** when the format is sql, specify the table name
- **--help**, **-h** show help (default: false)

### fly/query-file

Using **fly/query-file** command, you can query data from input file using sql directly.

```bash
heimdall fly --file data.csv --file data2.csv \
    --sql "SELECT table_0.id 'ID', table_0.name '名称', table_0.created_at '创建时间', count(*) as '字段数量' FROM table_0 LEFT JOIN table_1 ON table_0.id = table_1.ref_id WHERE table_1.deleted_at = '' GROUP BY table_0.id ORDER BY count(*) DESC LIMIT 10" \
    -f table
```

The following command line options are supported：

- **--sql value**, **-s value**, **--query value** SQL statement(if not set, read from STDIN, end with ';')
- **--file value**, **-i value**, **--input value** *[ --file value, -i value, --input value ]* input excel or csv file path, you can use the form TABLE:FILE to specify the table name corresponding to the file, this flag can be specified multiple times for importing multiple files at the same time
- **--csv-sepertor value** csv file sepertor, default is ',' (default: ",")
- **--format value**, **-f value** output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (default: "table")
- **--output value**, **-o value** write output to a file, default output directly to STDOUT
- **--no-header**, **-n** do not write table header (default: false)
- **--query-timeout value**, **-t value** query timeout, when the stream option is specified, this option is invalid (default: 2m0s)
- **--xlsx-max-row value** the maximum number of rows per sheet in an Excel file, including the row where the header is located (default: 1048576)
- **--table value** when the format is sql, specify the table name
- **--use-column-num** use column number as column name, start from 1, for example: col_1, col_2... (default: false)
- **--show-tables** show all tables in the database (default: false)
- **--temp-ds value** the temporary database uri, such as file:data.db?cache=shared, more options: https://www.sqlite.org/c3ref/open.html (default: ":memory:")
- **--slient** do not print warning log (default: false)
- **--debug**, **-D** Debug mode (default: false)

### convert

Using **convert** command, you can convert data from xlsx/csv to other formats: csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql.

```bash
heimdall convert --file data.csv --format json --include id --include name --include updated_at
```

The following command line options are supported：

- **--file value**, **-i value**, **--input value** input excel or csv file path
- **--csv-sepertor value** csv file sepertor, default is ',' (default: ",")
- **--format value**, **-f value** output format, support csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (default: "table")
- **--output value**, **-o value** write output to a file, default output directly to STDOUT
- **--no-header, -n** do not write table header (default: false)
- **--xlsx-max-row value** the maximum number of rows per sheet in an Excel file, including the row where the header is located (default: 1048576)
- **--table value** when the format is sql, specify the table name
- **--slient** do not print warning log (default: false)
- **--debug, -D** Debug mode (default: false)
- **--include value**, **-I value** *[ --include value, -I value ]* include fields, if set, only these fields will be output, this flag can be specified multiple times
- **--exclude value**, **-E value** *[ --exclude value, -E value ]* exclude fields, if set, these fields will be ignored, this flag can be specified multiple times

## Examples

Import a xlsx file to database table 

```bash
heimdall import --host 127.0.0.1 --port 3306 --database example --user root --password root \
    --table people \
    --field 区域:area \ 
    --field 姓名:name \
    --field 身份证号码:idcard \
    --file ~/Downloads/data.xlsx
```

Export new businesses in the last 30 days to an Excel file

```bash
heimdall export --database example --host 127.0.0.1 --user root --password root \
      --sql "SELECT id, name AS '企业名称', address AS '企业地址', city_name AS '城市', district_name AS '区县', DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS '创建时间' FROM enterprise WHERE created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) ORDER BY id DESC" \
      --streaming \
      --format xlsx \
      --output 最近30天新增企业列表.xlsx
```