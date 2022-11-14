package reader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/xuri/excelize/v2"
)

type FileWalker func(headerCB func(headers []string) error, dataCB func(id string, data []string) error) error

func CreateFileWalker(filePath string, csvSepertor rune) FileWalker {
	if strings.HasSuffix(filePath, ".xlsx") {
		return createExcelFileWalker(filePath)
	}

	if strings.HasSuffix(filePath, ".csv") {
		return createCSVFileWalker(filePath, csvSepertor)
	}

	return nil
}

func createCSVFileWalker(filePath string, csvSepertor rune) FileWalker {
	return func(headerCB func(headers []string) error, dataCB func(id string, data []string) error) error {
		f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
		if err != nil {
			return err
		}

		csvReader := csv.NewReader(f)
		csvReader.Comma = csvSepertor
		index := 0
		for {
			index++
			record, err := csvReader.Read()
			if err != nil {
				if err == io.EOF {
					break
				}

				return err
			}

			if index == 1 {
				if err := headerCB(record); err != nil {
					return err
				}

				continue
			}

			if err := dataCB(fmt.Sprintf("%d", index), record); err != nil {
				log.WithFields(log.Fields{"index": index}).Error(err)
			}
		}

		return nil
	}
}

func createExcelFileWalker(filePath string) FileWalker {
	return func(headerCB func(headers []string) error, dataCB func(id string, data []string) error) error {
		f, err := excelize.OpenFile(filePath)
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

			if err := headerCB(rows[0]); err != nil {
				return err
			}

			for rowNum, row := range rows[1:] {
				if err := dataCB(fmt.Sprintf("%s#%d", sheet, rowNum), row); err != nil {
					log.WithFields(log.Fields{"sheet": sheet, "row": rowNum}).Error(err)
				}
			}
		}

		return nil
	}
}
