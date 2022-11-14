package commands

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mylxsw/asteria/level"
	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/mylxsw/go-utils/ternary"
	"github.com/urfave/cli/v2"
	"github.com/xuri/excelize/v2"
)

// ImportOption import option
type ImportOption struct {
	InputFile string
	Table     string
	FieldsMap map[string]string
	Includes  []string
	Excludes  []string
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
		InputFile: c.String("input"),
		Table:     c.String("table"),
		FieldsMap: fieldsMap,
		Includes:  includes,
		Excludes:  ternary.If(len(includes) > 0, []string{}, excludes),
	}
}

// BuildImportFlags build import flags
func BuildImportFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "input", Aliases: []string{"i"}, Usage: "input excel file path", Required: true},
		&cli.StringFlag{Name: "table", Aliases: []string{"t"}, Usage: "target table name", Required: true},
		&cli.StringSliceFlag{Name: "field", Aliases: []string{"f"}, Usage: "field map, eg: excel_field:db_field"},
		&cli.StringSliceFlag{Name: "include", Aliases: []string{"I"}, Usage: "include fields, if set, only these fields will be imported"},
		&cli.StringSliceFlag{Name: "exclude", Aliases: []string{"E"}, Usage: "exclude fields, if set, these fields will be ignored"},
	}
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

	reader := ExcelFileReader{
		filePath:  opt.InputFile,
		fieldsMap: allowFields,
		table:     opt.Table,
	}

	return importData(db, reader)
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
		var field, _type, _null, _key, _default, _extra string
		if err := rows.Scan(&field, &_type, &_null, &_key, &_default, &_extra); err != nil {
			return nil, err
		}

		fieldsMap[field] = field
	}

	return fieldsMap, nil
}

type FileReader interface {
	Walk(cb func(id string, row []string, sqlTemplate string, fieldIndexs map[string]int) error) error
}

type ExcelFileReader struct {
	filePath  string
	fieldsMap map[string]string
	table     string
}

func (r ExcelFileReader) Walk(cb func(id string, row []string, sqlTemplate string, fieldIndexs map[string]int) error) error {
	f, err := excelize.OpenFile(r.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return err
		}

		if len(rows) < 2 {
			continue
		}

		sqlTemplate, fields := buildSQLAndFields(r.table, resolveFieldIndexs(rows[0], r.fieldsMap))
		for rowNum, row := range rows[1:] {
			if err := cb(fmt.Sprintf("%s#%d", sheet, rowNum), row, sqlTemplate, fields); err != nil {
				log.WithFields(log.Fields{"sheet": sheet, "row": rowNum}).Error(err)
			}
		}
	}

	return nil
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

// buildSQLAndFields build sql and fields
func buildSQLAndFields(table string, fieldIndexs map[string]int) (string, map[string]int) {
	fields := array.FromMapKeys(fieldIndexs)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ", "), strings.Join(array.Repeat("?", len(fields)), ",")), fieldIndexs
}

// importData import excel file
func importData(db *sql.DB, fileReader FileReader) error {
	var successCount, failedCount int
	fileReader.Walk(func(id string, row []string, sqlTemplate string, fieldIndexs map[string]int) error {
		var args []interface{}
		for fieldName := range fieldIndexs {
			if fieldIndexs[fieldName] < len(row) {
				args = append(args, row[fieldIndexs[fieldName]])
			}
		}

		if _, err := db.Exec(sqlTemplate, args...); err != nil {
			failedCount++
			log.With(log.Fields{
				"sql":  sqlTemplate,
				"args": args,
				"line": id,
			}).Errorf("exec sql failed: %v", err)
			return err
		}

		successCount++
		log.WithFields(log.Fields{
			"args": args,
			"line": id,
		}).Debugf("insert success %s", id)
		return nil
	})

	log.WithFields(log.Fields{
		"success": successCount,
		"fail":    failedCount,
	}).Debugf("import success")

	return nil
}
