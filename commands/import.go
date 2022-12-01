package commands

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/mylxsw/heimdall/reader"
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
		InputFiles:  array.Filter(c.StringSlice("file"), func(f string, i int) bool { return f != "" }),
		Table:       c.String("table"),
		FieldsMap:   fieldsMap,
		Includes:    includes,
		Excludes:    ternary.If(len(includes) > 0, []string{}, excludes),
		CSVSepertor: rune(c.String("csv-sepertor")[0]),
		UsingTx:     c.Bool("tx"),
		DryRun:      c.Bool("dry-run"),
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

	fieldsMap, err := resolveFieldsMapFromTable(db, opt.Table)
	if err != nil {
		return err
	}

	for k, v := range fieldsMap {
		if _, ok := opt.FieldsMap[k]; !ok {
			opt.FieldsMap[k] = v
		}
	}

	allowFields := make(map[string]string, 0)
	for k, v := range opt.FieldsMap {
		if len(opt.Includes) > 0 {
			if array.In(v, opt.Includes) {
				allowFields[k] = v
			}
		} else if len(opt.Excludes) > 0 {
			if !array.In(v, opt.Excludes) {
				allowFields[k] = v
			}
		} else {
			allowFields[k] = v
		}
	}

	walker := reader.MergeWalkers(array.Map(
		opt.InputFiles,
		func(f string, _ int) reader.FileWalker {
			return reader.CreateFileWalker(f, opt.CSVSepertor, false, false)
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

		if err := importData(tx, opt.Table, allowFields, walker); err != nil {
			defer log.Errorf("import data failed, all changes have been rolled back: %v", err)
			return tx.Rollback()
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

	return importData(db, opt.Table, allowFields, walker)
}

// resolveFieldsMapFromTable resolve fields map from table
func resolveFieldsMapFromTable(db *sql.DB, targetTable string) (map[string]string, error) {
	fieldsMap := make(map[string]string)
	rows, err := db.Query("SHOW COLUMNS FROM " + targetTable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var field, _type, _null, _key, _default, _extra *string
		if err := rows.Scan(&field, &_type, &_null, &_key, &_default, &_extra); err != nil {
			return nil, err
		}

		fieldsMap[*field] = *field
	}

	return fieldsMap, nil
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
func importData(tx Tx, table string, fieldMap map[string]string, fileWalker reader.FileWalker) (err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("panic: %v", err1)
		}
	}()

	log.With(fieldMap).Debug("fields map")

	var sqlTemplate string
	var fields []string
	var fieldIndexs map[string]int

	var successCount, failedCount int
	if err := fileWalker(
		func(filepath string, headers []string) error {
			fieldIndexs = resolveFieldIndexs(headers, fieldMap)
			if len(fieldIndexs) == 0 {
				return fmt.Errorf("no field matched for %s, headers: %v, fields map: %v", filepath, headers, fieldMap)
			}

			sqlTemplate, fields = buildSQLTemplate(table, fieldIndexs)
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
