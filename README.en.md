# Heimdall [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![MariaDB](https://img.shields.io/badge/MariaDB-003545?style=for-the-badge&logo=mariadb&logoColor=white) ![MySQL](https://img.shields.io/badge/mysql-%2300f.svg?style=for-the-badge&logo=mysql&logoColor=white) ![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white) 

[中文文档](./README.md)

Heimdall is a tool to perform import, export, direct SQL query and format conversion on Excel files. Currently supports directly writing SQL to query and export Excel files, export data in MySQL database to files in various formats such as json, yaml, markdown, csv, xlsx, html, sql, etc., import xlsx or csv files to MySQL database and Xlsx, csv file format conversion, file splitting and other functions.

## Command Line Options

**heimdall** support below commands

- **fly** (aka **query-file**) query data from input file using sql directly
- **import** (aka **load**) data from xlsx or csv file to database table
- **export** (aka **query**) SQL query results to various file formats
- **convert** convert data from xlsx/csv to other formats: csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql
- **split** split a large Excel file into multiple small files, each containing a specified number of rows at most 


### fly/query-file

Using **fly/query-file** command, you can query data from input file using sql directly.

Each input file is used as a table, and the naming format is **table_[serial number]**, starting from the first file, table_0, table_1, table_2 and so on.

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
- **--beta** enable beta feature, when this flag is set, the loading performance for large excel file will be improved, may be unstable, use at your own risk

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
- **--create-table** automatically create table structure
- **--use-column-num** Use column numbers as column names, starting from 1, such as col_1, col_2...
- **--with-ts** When creating the table structure, automatically add the created_at field to identify the time of import
- **--table-structure-format value** When this option is specified, the table structure information will be output after the import is complete, supporting `table`, `json`, `yaml`, `markdown`, `html`, `csv`, `xml` 

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

### split

Using **split** command, you can split a large Excel file into multiple small files, each containing a specified number of rows at most.

```bash
heimdall split --file data.xlsx --perfile-limit 1000 --header-row-num 2
```

The following command line options are supported：

- **--file value**, **-i value**, **--input value** input excel file path, currently only support xlsx format
- **--slient** do not print warning log (default: false)
- **--debug**, **-D** debug mode (default: false)
- **--perfile-limit value**, **-p value** the maximum number of records per file, only valid when mode=row (default: 1000)
- **--header-row-num value**, **-r value** table header row maximum row number, only valid when mode=row or mode=column (default: 1)
- **--mode value**, **-m value** split method: row, column, sheet (default: "row")
- **--column-index value**, **-c value** specifies the index of the column to split, such as 'A', 'AA', only valid when mode=column

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

Compare two Excel files containing population data, traverse the person information in file-1.csv in file-2.xlsx according to the ID number, find the existing person, and output the information of these person

```bash
heimdall fly --beta \
    -i ./file-1.csv \
    -i ./file-2.xlsx  \
    --sql "select xingming as '姓名', shenfenzhenghaoma as '身份证号码', shengri as '生日', xingbie as '性别', dizhi as '地址', lianxidianhua as '联系电话' from table_0 where table_0.shenfenzhenghaoma NOT IN (select IDCARDNO from table_1)" \
    --format xlsx \
    --output diff.xlsx
```

The exported data is deduplicated according to the ID number, and only one piece of data with the same ID number is kept

```bash
heimdall fly --beta \
    -i ./diff.xlsx \
    --sql "select xingming as '姓名', shenfenzhenghaoma as '身份证号码', shengri as '生日', xingbie as '性别', dizhi as '地址', lianxidianhua as '联系电话' from table_0 where __rowid IN (SELECT max(__rowid) FROM table_0 GROUP BY shenfenzhenghaoma)" \
    --format xlsx \
    --output diff-no-repeat.xlsx
```

Split the Excel file into multiple Excel files according to the dimension of the first column A

```bash
heimdall split -i ./data.xlsx -m column -c A
```

Split an Excel file into multiple files with up to 1000 rows of data each

```bash
heimdall split -m row --file data.xlsx --perfile-limit 1000 --header-row-num 2
```

