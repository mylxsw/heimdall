package commands

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"
	"github.com/mylxsw/asteria/event"
	"github.com/mylxsw/asteria/filter"
	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/maps"
	"github.com/mylxsw/go-utils/must"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/extracter"
	"github.com/mylxsw/heimdall/query"
	"github.com/mylxsw/heimdall/reader"
	"github.com/urfave/cli/v2"
)

type FlyOption struct {
	SQL         string
	InputFiles  []string
	CSVSepertor rune
	Slient      bool
	Debug       bool

	Format                  string
	Output                  string
	NoHeader                bool
	QueryTimeout            time.Duration
	XLSXMaxRow              int
	TargetTableForSQLFormat string

	UseColumnNumAsName bool
	ShowTables         bool
	TempDS             string
	Beta               bool
}

func BuildFlyFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "sql", Aliases: []string{"s", "query"}, Value: "", Usage: "SQL statement(if not set, read from STDIN, end with ';')"},
		&cli.StringSliceFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel or csv file path, you can use the form TABLE:FILE to specify the table name corresponding to the file, this flag can be specified multiple times for importing multiple files at the same time", Required: true},
		&cli.StringFlag{Name: "csv-sepertor", Value: ",", Usage: "csv file sepertor, default is ','"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "table", Usage: "output format, support " + strings.Join(query.SupportedStandardFormats, ", ")},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "", Usage: "write output to a file, default output directly to STDOUT"},
		&cli.BoolFlag{Name: "no-header", Aliases: []string{"n"}, Value: false, Usage: "do not write table header"},
		&cli.DurationFlag{Name: "query-timeout", Aliases: []string{"t"}, Value: 120 * time.Second, Usage: "query timeout, when the stream option is specified, this option is invalid"},
		&cli.IntFlag{Name: "xlsx-max-row", Value: 1048576, Usage: "the maximum number of rows per sheet in an Excel file, including the row where the header is located"},
		&cli.StringFlag{Name: "table", Value: "", Usage: "when the format is sql, specify the table name"},
		&cli.BoolFlag{Name: "use-column-num", Value: false, Usage: "use column number as column name, start from 1, for example: col_1, col_2..."},
		&cli.BoolFlag{Name: "show-tables", Value: false, Usage: "show all tables in the database"},
		&cli.StringFlag{Name: "temp-ds", Value: ":memory:", Usage: "the temporary database uri, such as file:data.db?cache=shared, more options: https://www.sqlite.org/c3ref/open.html"},
		&cli.BoolFlag{Name: "slient", Value: false, Usage: "do not print warning log"},
		&cli.BoolFlag{Name: "debug", Aliases: []string{"D"}, Value: false, Usage: "debug mode"},
		&cli.BoolFlag{Name: "beta", Usage: "enable beta feature, when this flag is set, the loading performance for large excel file will be improved, may be unstable, use at your own risk"},
	}
}

func resolveFlyOption(c *cli.Context) FlyOption {
	showTables := c.Bool("show-tables")
	sqlStr := c.String("sql")
	if sqlStr == "" && !showTables {
		sqlStr = readAll(os.Stdin, ';')
	}

	return FlyOption{
		SQL:         strings.Trim(strings.TrimSpace(sqlStr), ";"),
		InputFiles:  array.Filter(c.StringSlice("file"), func(f string, _ int) bool { return f != "" }),
		CSVSepertor: rune(c.String("csv-sepertor")[0]),

		Format:                  c.String("format"),
		Output:                  c.String("output"),
		NoHeader:                c.Bool("no-header"),
		QueryTimeout:            c.Duration("query-timeout"),
		XLSXMaxRow:              c.Int("xlsx-max-row"),
		TargetTableForSQLFormat: c.String("table"),
		UseColumnNumAsName:      c.Bool("use-column-num"),
		ShowTables:              showTables,
		TempDS:                  c.String("temp-ds"),
		Slient:                  c.Bool("slient"),
		Debug:                   c.Bool("debug"),
		Beta:                    c.Bool("beta"),
	}
}

func FlyCommand(c *cli.Context) error {
	opt := resolveFlyOption(c)

	if !opt.Debug {
		log.All().LogLevel(level.Info)
	}

	if opt.Slient {
		log.AddGlobalFilter(func(filter filter.Filter) filter.Filter {
			return func(evt event.Event) {
				if evt.Level == level.Warning {
					return
				}

				filter(evt)
			}
		})
	}

	if opt.SQL == "" && !opt.ShowTables {
		return fmt.Errorf("--sql or -s is required")
	}

	db, err := sql.Open("sqlite3", opt.TempDS)
	if err != nil {
		return fmt.Errorf("create sqlite3 database failed: %w", err)
	}
	defer db.Close()

	tables, err := createMemoryDatabaseForFly(opt, db)
	if err != nil {
		return err
	}

	handler := query.NewStandardQueryWriterWithDB(db, opt.TargetTableForSQLFormat, opt.QueryTimeout)
	if opt.ShowTables {
		return showTables(tables, handler)
	}

	w := ternary.IfElseLazy(
		opt.Output != "",
		func() io.WriteCloser { return must.Must(os.Create(opt.Output)) },
		func() io.WriteCloser { return os.Stdout },
	)
	defer w.Close()

	bar := NewProgressbar(!opt.Slient, "processing, be patient ...")
	defer bar.Clear()

	_, err = handler(opt.SQL, nil, opt.Format, w, opt.NoHeader, nil)

	return err
}

func showTables(tables []Table, handler func(sqlStr string, args []interface{}, format string, output io.Writer, noHeader bool, dataProcesser func(*extracter.Rows)) (int, error)) error {
	if _, err := handler("SELECT filename file, name 'table', created_at FROM meta", nil, "table", os.Stdout, false, nil); err != nil {
		return err
	}

	for _, table := range tables {
		fmt.Printf("\n◇ Table: %s ⇢ %s\n", table.Name, table.Filename)

		dataProcesser := func(r *extracter.Rows) {
			r.Columns = array.Filter(
				append(append(append([]extracter.Column{}, r.Columns[0:1]...), extracter.Column{Name: "original"}), r.Columns[1:]...),
				func(col extracter.Column, _ int) bool {
					return !strings.EqualFold(col.Name, "dflt_value")
				},
			)

			for i, row := range r.DataSets {
				row["pk"] = ternary.If(row["pk"].(int64) > 0, "Y", "N")
				row["notnull"] = ternary.If(row["notnull"].(int64) > 0, "Y", "N")

				if i > 0 {
					row["original"] = table.OriginalColumns[i-1]
				}
			}
		}

		if _, err := handler(fmt.Sprintf("PRAGMA table_info(%s)", table.Name), nil, "table", os.Stdout, false, dataProcesser); err != nil {
			return err
		}

		fmt.Println()
	}

	return nil
}

const (
	memoryTableIDField  = "__rowid"
	maxColumnNameLength = 30
)

type Table struct {
	Name            string
	Filename        string
	Hash            string
	Columns         []string
	OriginalColumns []string
}

// queryMaxMetaID query max meta id from database
func queryMaxMetaID(db *sql.DB) (int, error) {
	row := db.QueryRow("SELECT MAX(id) FROM meta")
	var maxID sql.NullInt64
	if err := row.Scan(&maxID); err != nil {
		return 0, fmt.Errorf("query max meta id failed: %w", err)
	}

	return int(maxID.Int64), nil
}

// queryMeta query meta from database
func queryMeta(db *sql.DB, filename string) (*Table, error) {
	row := db.QueryRow("SELECT filename, hash, name, columns, original_columns FROM meta WHERE filename = ?", filename)
	var meta Table
	var originalColumns, columns string
	if err := row.Scan(&meta.Filename, &meta.Hash, &meta.Name, &columns, &originalColumns); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("query meta failed: %w", err)
	}

	meta.Columns = strings.Split(columns, ",")
	meta.OriginalColumns = strings.Split(originalColumns, ",")

	return &meta, nil
}

// queryMetas query all metas from database
func queryMetas(db *sql.DB) ([]Table, error) {
	rows, err := db.Query("SELECT filename, hash, name, columns, original_columns FROM meta")
	if err != nil {
		return nil, fmt.Errorf("query metas failed: %w", err)
	}
	defer rows.Close()

	var metas []Table
	for rows.Next() {
		var meta Table
		var originalColumns, columns string
		if err := rows.Scan(&meta.Filename, &meta.Hash, &meta.Name, &columns, &originalColumns); err != nil {
			return nil, fmt.Errorf("scan meta failed: %w", err)
		}

		meta.Columns = strings.Split(columns, ",")
		meta.OriginalColumns = strings.Split(originalColumns, ",")

		metas = append(metas, meta)
	}

	return metas, nil
}

// initMemoryDatabaseMeta init meta table
func initMemoryDatabaseMeta(opt FlyOption, db *sql.DB) error {
	createSQL := `CREATE TABLE IF NOT EXISTS meta (id int PRIMARY KEY NOT NULL, filename, hash, name, columns, original_columns, created_at);`
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("create meta table failed: %w", err)
	}

	return nil
}

// createMemoryDatabaseForFly create memory database
func createMemoryDatabaseForFly(opt FlyOption, db *sql.DB) ([]Table, error) {
	if err := initMemoryDatabaseMeta(opt, db); err != nil {
		return nil, err
	}

	tableMetas := array.BuildMap(opt.InputFiles, func(val string, i int) (string, Table) {
		var tableName, filename string
		segs := strings.SplitN(val, ":", 2)
		if len(segs) == 2 {
			tableName = segs[0]
			filename = segs[1]
		} else {
			tableName = fmt.Sprintf("table_%d", must.Must(queryMaxMetaID(db))+i)
			filename = val
		}

		return filename, Table{
			Name:     tableName,
			Filename: filename,
			Hash:     must.Must(fileHash(filename)),
		}
	})

	// 过滤需要更新的文件，如果文件 hash 和数据库中原有的一致，则不需要更新
	inputFiles := array.FromMap(maps.Filter(tableMetas, func(t Table, _ string) bool {
		meta := must.Must(queryMeta(db, t.Filename))
		if meta == nil {
			return true
		}

		if meta.Hash != t.Hash {
			return true
		}

		return false
	}))

	if len(inputFiles) > 0 {
		walker := reader.MergeWalkers(array.Map(
			inputFiles,
			func(t Table, _ int) reader.FileWalker {
				return reader.CreateFileWalker(t.Filename, opt.CSVSepertor, opt.ShowTables && opt.TempDS == ":memory:", opt.Beta || opt.ShowTables)
			})...,
		)
		if walker == nil {
			return nil, fmt.Errorf("no file avaiable: only support csv or xlsx files")
		}

		var currentTableName string
		var currentTableFields []string
		var recordIndex = 1

		bar := NewProgressbar(!opt.Slient, "Initializing ...")
		defer bar.Clear()

		if err := walker(
			func(filepath string, headers []string) error {
				bar.Describe("Loading ...")
				bar.Add(1)

				currentTableName = tableMetas[filepath].Name
				fields := array.Map(createDBFieldsFromHeaders(headers, opt.UseColumnNumAsName), func(h DatabaseField, i int) string {
					return h.Field
				})

				currentTableFields = append([]string{memoryTableIDField}, fields...)
				createSQL := fmt.Sprintf(
					"CREATE TABLE %s (%s int PRIMARY KEY NOT NULL, %s);",
					currentTableName,
					memoryTableIDField,
					strings.Join(fields, ","),
				)

				if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", currentTableName)); err != nil {
					return fmt.Errorf("drop table %s failed: %w", currentTableName, err)
				}

				if _, err := db.Exec(createSQL); err != nil {
					return fmt.Errorf("create table %s failed: %w", currentTableName, err)
				}

				return addTableMeta(db, filepath, currentTableName, tableMetas[filepath].Hash, currentTableFields, headers)
			},
			func(filepath string, id string, data []string) error {
				if opt.ShowTables && opt.TempDS == ":memory:" {
					return nil
				}

				defer func() {
					recordIndex++
					bar.Add(1)
				}()

				maxSize := ternary.If(len(data) > (len(currentTableFields)-1), len(currentTableFields)-1, len(data))

				placeholders := strings.Join(array.Repeat("?", maxSize), ",")
				fields := strings.Join(currentTableFields[0:maxSize+1], ",")

				insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%d, %s);", currentTableName, fields, recordIndex, placeholders)

				if _, err := db.Exec(insertSQL, array.Map(data, func(d string, i int) any { return d })...); err != nil {
					return fmt.Errorf("insert data failed: %w", err)
				}

				return nil
			},
		); err != nil {
			return nil, err
		}
	}

	tables, err := queryMetas(db)
	if err != nil {
		return nil, err
	}

	return array.DistinctBy(tables, func(item Table) string { return item.Name }), nil
}

// addTableMeta add table meta to database
func addTableMeta(db *sql.DB, filepath string, tableName string, hash string, currentTableFields []string, headers []string) error {
	if _, err := db.Exec("DELETE FROM meta WHERE filename = ?", filepath); err != nil {
		return fmt.Errorf("delete meta for %s failed: %w", tableName, err)
	}

	if _, err := db.Exec(
		"INSERT INTO meta (id, filename, hash, name, columns, original_columns, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		must.Must(queryMaxMetaID(db))+1,
		filepath,
		hash,
		tableName,
		strings.Join(currentTableFields, ","),
		strings.Join(headers, ","),
		time.Now().Format("2006-01-02 15:04:05"),
	); err != nil {
		return fmt.Errorf("insert meta for %s failed: %w", tableName, err)
	}

	return nil
}

func slugifyColumnName(name string) string {
	arg := pinyin.NewArgs()
	arg.Separator = ""
	arg.Fallback = func(r rune, a pinyin.Args) []string {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return []string{string(r)}
		}

		return []string{}
	}

	p := strings.Join(pinyin.LazyPinyin(name, arg), "")
	return strings.ReplaceAll(p, " ", "_")
}

func createDBFieldsFromHeaders(headers []string, useColumnNumAsName bool) []DatabaseField {
	return array.Map(headers, func(h string, i int) DatabaseField {
		if useColumnNumAsName {
			return DatabaseField{
				Field: fmt.Sprintf("col_%d", i+1),
				Name:  h,
				Index: i,
			}
		}
		name := slugifyColumnName(h)
		if name == "" || len(name) > maxColumnNameLength {
			log.Warningf("column name [%s] is invalid (empty or too long), use col_%d instead", extracter.Sanitize(h), i+1)
			return DatabaseField{
				Field: fmt.Sprintf("col_%d", i+1),
				Name:  h,
				Index: i,
			}
		}

		if !unicode.IsLetter(rune(name[0])) {
			log.Warningf("column name [%s] is invalid, use col_%d instead", extracter.Sanitize(h), i+1)
			name = fmt.Sprintf("col_%d", i+1)
		}

		return DatabaseField{
			Field: name,
			Name:  h,
			Index: i,
		}
	})
}

type DatabaseField struct {
	Index int
	Field string
	Name  string
}
