# mysql-querier

MySQL-Querier is a small tool used to execute SQL query statements in MySQL and output the returned results in JSON/YAML/text lines

```bash
-db string
    MySQL Database
-format string
    Output format: json/yaml/plain (default "json")
-host string
    MySQL Host (default "127.0.0.1")
-password string
    MySQL Password
-port int
    MySQL Port (default 3306)
-sql string
    The SQL to be executed, if not specified, read from the standard input pipe
-user string
    MySQL User (default "root")
-version
    Output version information
```