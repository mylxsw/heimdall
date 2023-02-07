# Heimdall [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


![MariaDB](https://img.shields.io/badge/MariaDB-003545?style=for-the-badge&logo=mariadb&logoColor=white) ![MySQL](https://img.shields.io/badge/mysql-%2300f.svg?style=for-the-badge&logo=mysql&logoColor=white) ![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white) 

[English Version](./README.en.md)

Heimdall 是一款对 Excel 文件执行导入、导出、直接 SQL 查询和格式转换的工具。当前支持直接写 SQL 对 Excel 文件进行查询和导出，将 MySQL 数据库中的数据导出为 json、yaml、markdown、csv、xlsx、html、sql 等多种格式的文件，导入 xlsx 或者 csv 文件到 MySQL 数据库以及 xlsx、csv 文件的格式转换，文件拆分等功能。

## 命令行选项

**heimdall** 支持下列命令

- **fly** (或者 **query-file**) 使用 SQL 直接对 xlsx、csv 文件进行查询并导出为 json、yaml、markdown、csv、xlsx、html、sql 等多种格式的文件
- **import** (或者 **load**) 将 xlsx、csv 文件中的数据导入到 MySQL 数据库
- **export** (或者 **query**) 将 MySQL 中的数据，按照 SQL 的查询结果导出 json、yaml、markdown、csv、xlsx、html、sql 等多种格式的文件
- **convert** 将 xlsx、csv 文件转换为其它格式如 json、yaml、markdown、csv、xlsx、html、sql 等
- **split** 将一个比较大的 xlsx 文件拆分为多个文件，当前支持按照行数、按照某一列的值、按照 Sheet 进行拆分

### fly/query-file

使用 **fly/query-file** 命令，可以直接对输入的 xlsx、csv 文件写 SQL 进行查询和过滤处理，并且导出为 json、yaml、markdown、csv、xlsx、html、sql 等多种格式的文件。

每一个输入（`--file`）的文件都会作为一个数据库表，表名命名格式为 **table_序号**，序号从第一个文件开始，按照 `table_0`，`table_1`，`table_2` 以此类推。

```bash
heimdall fly --file data.csv --file data2.csv \
    --sql "SELECT table_0.id 'ID', table_0.name '名称', table_0.created_at '创建时间', count(*) as '字段数量' FROM table_0 LEFT JOIN table_1 ON table_0.id = table_1.ref_id WHERE table_1.deleted_at = '' GROUP BY table_0.id ORDER BY count(*) DESC LIMIT 10" \
    -f table
```

支持下面这些命令行选项：

- **--sql value**, **-s value**, **--query value** SQL 语句 (如果没有指定，则会从标准输入 STDIN 中读取，直到遇到';'结束)
- **--file value**, **-i value**, **--input value** *[ --file value, -i value, --input value ]* 要查询的 xlsx 或者 csv 文件路径，可以使用 `TABLE:FILE` 的形式来为文件指定表名，该选项可以指定多次，用于一次对多个文件进行连表查询
- **--csv-sepertor value** csv 文件分隔符 (默认值: ",")
- **--format value**, **-f value** 输出格式，支持 csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (默认值: "table")
- **--output value**, **-o value** 输出路径，默认直接输出到标准输出 STDOUT
- **--no-header**, **-n** 不要输出表头
- **--query-timeout value**, **-t value** 查询超时时间，当指定 `stream` 选项时，该选项无效 (默认值: 2m0s)
- **--xlsx-max-row value** 输出格式为 xlsx 时，指定每个 Sheet 中最大的行数（包含表头），超过该值时会自动拆分到多个 Sheet (默认值: 1048576)
- **--table value** 输出格式为 sql 时，指定 sql 语句中的表名
- **--use-column-num** 使用列编号作为列名，从 1 开始，如 col_1, col_2...
- **--show-tables** 查看当前文件对应的所有表和字段
- **--temp-ds value** 临时数据库的 URI，默认使用内存数据库，可以指定文件来为多次查询加速，例如 file:data.db?cache=shared, 更多选项查看: https://www.sqlite.org/c3ref/open.html (默认值: ":memory:")
- **--slient** 不要输出警告日志
- **--debug**, **-D** 启用调试模式
- **--beta** 允许 beta 特性，当指定该选项时，大型 xlsx 文件的加载速度会有大幅度提升，目前该功能可能会存在不稳定的因素，请谨慎使用

### import/load

使用 **import/load** 命令，可以将 xlsx、csv 文件导入到 MySQL 表中。

```bash
heimdall import --tx --database example --table users \
    --file users.csv --file users.xlsx \
    --field 姓名:name \
    --field 年龄:age
```

支持下面这些命令行选项：

- **--host value**, **-H value** MySQL 主机地址 (default: "127.0.0.1")
- **--port value**, **-P value** MySQL 端口 (default: 3306)
- **--user value**, **-u value** MySQL 用户名 (default: "root")
- **--password value**, **-p value** MySQL 密码
- **--database value**, **-d value** MySQL 数据库
- **--connect-timeout value** 数据库连接超时时间 (default: 3s)
- **--debug**, **-D** 启用调试模式 (default: false)
- **--file value**, **-i value**, **--input value** *[ --file value, -i value, --input value ]* 输入文件路径，支持 xlsx、csv，该选项可以指定多次，用于同时导入多个文件
- **--table value**, **-t value** 要导入的表名称
- **--field value**, **-f value** *[ --field value, -f value ]* 字段关系，如: excel_field:db_field, 该选项可以指定多次
- **--include value**, **-I value** *[ --include value, -I value ]* 包含字段白名单，如果指定，则只有白名单中的字段将会被导入，该选项可以指定多次
- **--exclude value**, **-E value** *[ --exclude value, -E value ]* 排除字段，如果指定，这里的字段将会被忽略，该选项可以指定多次
- **--csv-sepertor value** csv 文件分隔符 (默认值: ",")
- **--tx**, **-T** 启用事务支持，所有文件的导入全部成功或者全部失败，只有支持事务的数据存储引擎支持，如 InnoDB 等
- **--dry-run** 执行导入测试以验证，只有支持事务的存储引擎支持

### export/query

使用 **export/query** 命令，可以写一个 SQL 来查询 MySQL 数据库中的数据，将查询结果导出到 json、yaml、markdown、csv、xlsx、html、sql 等多种格式的文件。

```bash
heimdall export --database example --format json --sql 'select * from users'
```

支持下面这些命令行选项：

- **--host value**, **-H value** MySQL 主机地址 (默认值: "127.0.0.1")
- **--port value**, **-P value** MySQL 端口 (默认值: 3306)
- **--user value**, **-u value** MySQL 用户 (默认值: "root")
- **--password value**, **-p value** MySQL 密码
- **--database value**, **-d value** MySQL 数据库
- **--connect-timeout value** 数据库连接超时时间 (默认值: 3s)
- **--debug**, **-D** 启用调试模式
- **--sql value**, **-s value** SQL 查询语句
- **--format value**, **-f value** 输出格式，支持 csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (默认值: "csv")
- **--output value**, **-o value** 输出路径，默认直接输出到标准输出 STDOUT
- **--streaming**, **-S** 是否使用流式输出，如果使用该选项，数据将会在查询过程中一行一行的写入到输出文件，使用该选项可以显著降低内存占用和数据库的查询负担。使用该选项时，输出格式只支持 csv、xlsx、json、plain、sql
- **--no-header**, **-n** 不要输出表头 
- **--query-timeout value**, **-t value** 查询超时时间，当指定 stream 选项时，该选项无效 (默认值: 2m0s)
- **--xlsx-max-row value**  输出格式为 xlsx 时，指定每个 Sheet 中最大的行数（包含表头），超过该值时会自动拆分到多个 Sheet (默认值: 1048576)
- **--table value** 输出格式为 sql 时，指定 sql 语句中的表名

### convert

使用 **convert** 命令，可以将 xlsx、csv 文件转换为 json、yaml、markdown、csv、xlsx、html、sql 等格式。

```bash
heimdall convert --file data.csv --format json --include id --include name --include updated_at
```

支持下面这些命令行选项：

- **--file value**, **-i value**, **--input value** 要转换格式的 xlsx 或者 csv 文件路径
- **--csv-sepertor value** csv 文件分隔符 (默认值: ",")
- **--format value**, **-f value** 输出格式，支持 csv, json, yaml, xml, table, html, markdown, xlsx, plain, sql (默认值: "table")
- **--output value**, **-o value** 输出路径，默认直接输出到标准输出 STDOUT
- **--no-header, -n** 不要输出表头
- **--xlsx-max-row value** 输出格式为 xlsx 时，指定每个 Sheet 中最大的行数（包含表头），超过该值时会自动拆分到多个 Sheet (默认值: 1048576)
- **--table value** 输出格式为 sql 时，指定 sql 语句中的表名
- **--slient** 不要输出警告日志
- **--debug, -D** 启用调试模式
- **--include value**, **-I value** *[ --include value, -I value ]* 包含字段白名单，如果指定，则只有白名单中的字段将会输出，该选项可以指定多次
- **--exclude value**, **-E value** *[ --exclude value, -E value ]* 排除字段，如果指定，这里的字段将会被忽略，该选项可以指定多次

### split

使用 **split** 命令，可以将一个比较大的 xlsx 文件拆分为多个小文件，支持按照行数、指定的列值以及 Sheet 进行拆分。

```bash
heimdall split --file data.xlsx --perfile-limit 1000 --header-row-num 2
```

支持下面这些命令行选项：

- **--file value**, **-i value**, **--input value** 要拆分的 xlsx 文件路径，只支持 xlsx 文件
- **--slient** 不要输出警告信息
- **--debug**, **-D** 启用调试模式
- **--perfile-limit value**, **-p value** 每个文件中包含的最大行数，当 mode 为 row 时有效 (默认值: 1000)
- **--header-row-num value**, **-r value** 表格中表头行数，只有 mode 为 row 和 column 时有效 (默认值: 1)
- **--mode value**, **-m value** 文件拆分方式: row, column, sheet (默认值: "row")
- **--column-index value**, **-c value** 指定要按照哪一列的值进行拆分，如 'A', 'AA', 只在 mode 为 column 时有效

## 示例

将一个 xlsx 文件导入到数据库 `example` 的 `people` 表中。

```bash
heimdall import --host 127.0.0.1 --port 3306 --database example --user root --password root \
    --table people \
    --field 区域:area \ 
    --field 姓名:name \
    --field 身份证号码:idcard \
    --file ~/Downloads/data.xlsx
```

对数据库 `example` 执行 SQL，查询出创建时间为 30 天内的企业客户列表，按照 id 倒序排列，导出为 xlsx 文件。

```bash
heimdall export --database example --host 127.0.0.1 --user root --password root \
      --sql "SELECT id, name AS '企业名称', address AS '企业地址', city_name AS '城市', district_name AS '区县', DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS '创建时间' FROM enterprise WHERE created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) ORDER BY id DESC" \
      --streaming \
      --format xlsx \
      --output 最近30天新增企业列表.xlsx
```

比较两个包含人员信息的文件，遍历文件 `file-0.csv`，对于其中的每一行数据，取其中的 `身份证号码` 字段，在 `file-1.xlsx` 文件中查找是否存在，如果存在则将该条数据输出到一个新的 xlsx 文件。

```bash
heimdall fly --beta \
    -i ./file-0.csv \
    -i ./file-1.xlsx  \
    --sql "select xingming as '姓名', shenfenzhenghaoma as '身份证号码', shengri as '生日', xingbie as '性别', dizhi as '地址', lianxidianhua as '联系电话' from table_0 where table_0.shenfenzhenghaoma NOT IN (select IDCARDNO from table_1)" \
    --format xlsx \
    --output result.xlsx
```

将 xlsx 文件中的数据，按照`身份证号码`这一列进行去重，只保留其中一条数据，导出为新的 `xlsx` 文件。

```bash
heimdall fly --beta \
    -i ./data.xlsx \
    --sql "select xingming as '姓名', shenfenzhenghaoma as '身份证号码', shengri as '生日', xingbie as '性别', dizhi as '地址', lianxidianhua as '联系电话' from table_0 where __rowid IN (SELECT max(__rowid) FROM table_0 GROUP BY shenfenzhenghaoma)" \
    --format xlsx \
    --output result.xlsx
```

将一个 xlsx 文件以列 A 的值进行拆分

```bash
heimdall split -i ./data.xlsx -m column -c A
```

将一个 xlsx 文件拆分为每个文件 1000 条数据的多个文件

```bash
heimdall split -m row --file data.xlsx --perfile-limit 1000 --header-row-num 2
```