package commands

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/query"
	"github.com/mylxsw/heimdall/reader"
	"github.com/mylxsw/heimdall/render"
	"github.com/urfave/cli/v2"
)

// ImportOption import option
type ImportOption struct {
	InputFiles  []string
	Table       string
	FieldsMap   map[string]string
	Includes    []string
	Excludes    []string
	CSVSepertor rune
	UsingTx     bool
	DryRun      bool

	CreateTable          bool
	UseColumnNumAsName   bool
	Beta                 bool
	WithCreateTime       bool
	TableStructureFormat string
}

// resolveImportOption resolve import option
func resolveImportOption(c *cli.Context) ImportOption {
	fieldsMap := make(map[string]string)
	for _, field := range c.StringSlice("field") {
		parts := strings.SplitN(field, ":", 2)
		if len(parts) == 2 {
			fieldsMap[parts[0]] = parts[1]
		} else {
			fieldsMap[parts[0]] = parts[0]
		}
	}

	includes := c.StringSlice("include")
	excludes := c.StringSlice("exclude")
	if len(includes) > 0 && len(excludes) > 0 {
		log.WithFields(log.Fields{
			"includes": includes,
			"excludes": excludes,
		}).Warning("includes and excludes are both set, excludes will be ignored")
	}

	return ImportOption{
		InputFiles:           array.Filter(c.StringSlice("file"), func(f string, i int) bool { return f != "" }),
		Table:                c.String("table"),
		FieldsMap:            fieldsMap,
		Includes:             includes,
		Excludes:             ternary.If(len(includes) > 0, []string{}, excludes),
		CSVSepertor:          rune(c.String("csv-sepertor")[0]),
		UsingTx:              c.Bool("tx"),
		DryRun:               c.Bool("dry-run"),
		CreateTable:          c.Bool("create-table"),
		UseColumnNumAsName:   c.Bool("use-column-num"),
		Beta:                 c.Bool("beta"),
		WithCreateTime:       c.Bool("with-ts"),
		TableStructureFormat: c.String("table-structure-format"),
	}
}

// BuildImportFlags build import flags
func BuildImportFlags() []cli.Flag {
	return append(BuildGlobalFlags(), []cli.Flag{
		&cli.StringSliceFlag{Name: "file", Aliases: []string{"i", "input"}, Usage: "input excel or csv file path, this flag can be specified multiple times for importing multiple files at the same time", Required: true},
		&cli.StringFlag{Name: "table", Aliases: []string{"t"}, Usage: "target table name", Required: true},
		&cli.StringSliceFlag{Name: "field", Aliases: []string{"f"}, Usage: "field map, eg: excel_field:db_field, this flag can be specified multiple times"},
		&cli.StringSliceFlag{Name: "include", Aliases: []string{"I"}, Usage: "include fields, if set, only these fields will be imported, this flag can be specified multiple times"},
		&cli.StringSliceFlag{Name: "exclude", Aliases: []string{"E"}, Usage: "exclude fields, if set, these fields will be ignored, this flag can be specified multiple times"},
		&cli.StringFlag{Name: "csv-sepertor", Value: ",", Usage: "csv file sepertor, default is ','"},
		&cli.BoolFlag{Name: "tx", Aliases: []string{"T"}, Usage: "import data using transaction, all success or all failure, only work with InnoDB or other engines that support transaction"},
		&cli.BoolFlag{Name: "dry-run", Usage: "perform import tests to verify correctness of imported files, but do not commit transactions, only work with InnoDB or other engines that support transaction"},
		&cli.BoolFlag{Name: "create-table", Usage: "create table automatically if not exists"},
		&cli.BoolFlag{Name: "use-column-num", Value: false, Usage: "use column number as column name, start from 1, for example: col_1, col_2..."},
		&cli.BoolFlag{Name: "with-ts", Usage: "add created_at column to table"},
		&cli.StringFlag{Name: "table-structure-format", Usage: "if set, the table structure will be output to the stdout with the specified format, support: json, yaml, table"},
	}...)
}

// ImportCommand import command
func ImportCommand(c *cli.Context) error {
	if !c.Bool("debug") {
		log.All().LogLevel(level.Info)
	}

	opt := resolveImportOption(c)
	globalOpt := resolveGlobalOption(c)

	db, err := sql.Open("mysql", globalOpt.DSN())
	if err != nil {
		return err
	}
	defer db.Close()

	if globalOpt.ConnectTimeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), globalOpt.ConnectTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("database is unreached: %w", err)
		}
	}

	walker := reader.MergeWalkers(array.Map(
		opt.InputFiles,
		func(f string, _ int) reader.FileWalker {
			return reader.CreateFileWalker(f, opt.CSVSepertor, false, opt.Beta)
		})...,
	)
	if walker == nil {
		return fmt.Errorf("no file avaiable: only support csv or xlsx files")
	}

	if opt.UsingTx || opt.DryRun {
		log.Debugf("import data using transaction")

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if err := importData(opt, tx, walker); err != nil {
			defer log.Errorf("import data failed, all changes have been rolled back: %v", err)
			return tx.Rollback()
		}

		if opt.TableStructureFormat != "" {
			printTableStructure(db, globalOpt.Database, opt.Table, opt.TableStructureFormat)
		}

		if opt.DryRun {
			log.Debugf("import test finished, all changes have been rolled back")
			return tx.Rollback()
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("commit transaction failed, all changes have been rolled back: %v", err)
			return err
		}

		return nil
	}

	if err := importData(opt, db, walker); err != nil {
		return err
	}

	if opt.TableStructureFormat != "" {
		printTableStructure(db, globalOpt.Database, opt.Table, opt.TableStructureFormat)
	}

	return nil
}

// printTableStructure print table structure
func printTableStructure(db *sql.DB, targetDB string, targetTable string, format string) {
	rows, err := query.QueryDB(
		db,
		"SELECT COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT FROM INFORMATION_SCHEMA.COLUMNS WHERE table_name = ? AND table_schema = ?",
		[]interface{}{targetTable, targetDB},
		30*time.Second,
	)
	if err != nil {
		log.Errorf("query table structure failed: %v", err)
		return
	}

	buf, err := render.Render(format, false, rows.Columns, rows.DataSets, "", "")
	if err != nil {
		log.Errorf("render table structure failed: %v", err)
		return
	}

	fmt.Println(buf.String())

	fields := array.Map(rows.DataSets, func(row map[string]interface{}, _ int) string {
		if row["COLUMN_COMMENT"] != "" {
			return fmt.Sprintf("%s AS %s", row["COLUMN_NAME"], strconv.Quote(row["COLUMN_COMMENT"].(string)))
		}

		return row["COLUMN_NAME"].(string)
	})

	log.Infof("QUERY SQL: SELECT %s FROM %s\n", strings.Join(fields, ", "), targetTable)
}

func resolveFieldIndexs(header []string, fieldsMap map[string]string) map[string]int {
	fieldIndexs := make(map[string]int)
	for i, field := range header {
		if field == "" {
			continue
		}

		if fieldName, ok := fieldsMap[field]; ok {
			fieldIndexs[fieldName] = i
		}
	}

	return fieldIndexs
}

// buildSQLTemplate build sql template
func buildSQLTemplate(table string, fieldIndexs map[string]int) (string, []string) {
	fields := array.FromMapKeys(fieldIndexs)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ", "), strings.Join(array.Repeat("?", len(fields)), ",")), fields
}

type Tx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// importData import excel file
func importData(opt ImportOption, tx Tx, fileWalker reader.FileWalker) (err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("panic: %v", err1)
		}
	}()

	log.With(opt.FieldsMap).Debug("fields map")

	var sqlTemplate string
	var fields []string
	var fieldIndexs map[string]int

	var successCount, failedCount int
	if err := fileWalker(
		func(filepath string, headers []string) error {
			dbFields := resolveAllowFields(
				array.Map(createDBFieldsFromHeaders(headers, opt.UseColumnNumAsName), func(field DatabaseField, _ int) DatabaseField {
					mapV, ok := opt.FieldsMap[field.Name]
					if ok {
						field.Field = mapV
					}

					return field
				}),
				opt.Includes,
				opt.Excludes,
			)

			if opt.CreateTable {
				fieldLines := array.Map(dbFields, func(f DatabaseField, _ int) string {
					return fmt.Sprintf("%s TEXT NULL DEFAULT NULL COMMENT %s", f.Field, strconv.Quote(f.Name))
				})

				if opt.WithCreateTime {
					fieldLines = append(fieldLines, "created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP")
				}

				createSQL := fmt.Sprintf(
					"CREATE TABLE %s (id int PRIMARY KEY AUTO_INCREMENT, %s);",
					opt.Table,
					strings.Join(fieldLines, ","),
				)

				if _, err := tx.Exec(createSQL); err != nil {
					return fmt.Errorf("create table %s failed: %w", opt.Table, err)
				}
			}

			fieldsMap := array.BuildMap(dbFields, func(f DatabaseField, _ int) (string, string) {
				return f.Name, f.Field
			})

			fieldIndexs = resolveFieldIndexs(headers, fieldsMap)
			if len(fieldIndexs) == 0 {
				return fmt.Errorf("no field matched for %s, headers: %v, fields map: %v", filepath, headers, fieldsMap)
			}

			sqlTemplate, fields = buildSQLTemplate(opt.Table, fieldIndexs)
			return nil
		},
		func(filepath string, id string, row []string) error {
			var args []interface{}
			for _, fieldName := range fields {
				if fieldIndexs[fieldName] < len(row) {
					arg := strings.TrimSpace(row[fieldIndexs[fieldName]])
					if arg != "" {
						args = append(args, arg)
					} else {
						args = append(args, nil)
					}
				} else {
					args = append(args, nil)
				}
			}

			if len(array.Filter(args, func(arg interface{}, _ int) bool { return arg != nil })) == 0 {
				log.WithFields(log.Fields{"file": filepath}).Warningf("skip empty row: %s", id)
				return nil
			}

			if _, err := tx.Exec(sqlTemplate, args...); err != nil {
				failedCount++
				log.WithFields(log.Fields{
					"sql":  sqlTemplate,
					"args": args,
					"line": id,
					"file": filepath,
				}).Errorf("exec sql failed: %v", err)
				return err
			}

			successCount++
			log.WithFields(log.Fields{
				"args": args,
				"line": id,
				"file": filepath,
			}).Infof("insert success %s", id)
			return nil
		},
	); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"success": successCount,
		"fail":    failedCount,
	}).Infof("import success")

	return nil
}

func resolveAllowFields(fields []DatabaseField, includes []string, excludes []string) []DatabaseField {
	allowFields := make([]DatabaseField, 0)
	for _, f := range fields {
		if len(includes) > 0 {
			if array.In(f.Field, includes) {
				allowFields = append(allowFields, f)
			}
		} else if len(excludes) > 0 {
			if !array.In(f.Field, excludes) {
				allowFields = append(allowFields, f)
			}
		} else {
			allowFields = append(allowFields, f)
		}
	}

	return allowFields
}
