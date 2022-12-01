package reader

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/thedatashed/xlsxreader"
	"github.com/xuri/excelize/v2"
)

type FileWalker func(headerCB func(filepath string, headers []string) error, dataCB func(filepath string, id string, data []string) error) error

func MergeWalkers(walkers ...FileWalker) FileWalker {
	walkers = array.Filter(walkers, func(walker FileWalker, i int) bool { return walker != nil })
	if len(walkers) == 0 {
		return nil
	}

	return func(headerCB func(filepath string, headers []string) error, dataCB func(filepath string, id string, data []string) error) error {
		for _, walker := range walkers {
			if err := walker(headerCB, dataCB); err != nil {
				return err
			}
		}

		return nil
	}
}

func CreateFileWalker(filePath string, csvSepertor rune, onlyHeader bool, beta bool) FileWalker {
	if strings.HasSuffix(filePath, ".xlsx") {
		if beta {
			return createExcelFileStreamWalker(filePath, onlyHeader)
		}

		return createExcelFileWalker(filePath, onlyHeader)
	}

	if strings.HasSuffix(filePath, ".csv") {
		return createCSVFileWalker(filePath, csvSepertor, onlyHeader)
	}

	return nil
}

func createCSVFileWalker(filePath string, csvSepertor rune, onlyHeader bool) FileWalker {
	return func(headerCB func(filepath string, headers []string) error, dataCB func(filepath string, id string, data []string) error) error {
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

			if onlyHeader && index > 1 {
				break
			}

			if index == 1 {
				if err := headerCB(filePath, record); err != nil {
					log.WithFields(log.Fields{"file": filePath}).Errorf("handle header failed: %s", err)
					return err
				}

				continue
			}

			if err := dataCB(filePath, fmt.Sprintf("%d", index), record); err != nil {
				log.WithFields(log.Fields{"index": index, "file": filePath}).Errorf("handle data failed: %s", err)
			}
		}

		return nil
	}
}

func createExcelFileWalker(filePath string, onlyHeader bool) FileWalker {
	return func(headerCB func(filepath string, headers []string) error, dataCB func(filepath string, id string, data []string) error) error {
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		for i, sheet := range f.GetSheetList() {
			rows, err := f.GetRows(sheet)
			if err != nil {
				return err
			}

			if i > 0 && len(rows) > 0 {
				log.Warningf("file %s has more than one sheet, only the first sheet will be processed", filePath)
				break
			}

			if len(rows) < 2 {
				continue
			}

			if err := headerCB(filePath, rows[0]); err != nil {
				log.WithFields(log.Fields{"file": filePath}).Errorf("handle header failed: %s", err)
				return err
			}

			if !onlyHeader {
				for rowNum, row := range rows[1:] {
					if err := dataCB(filePath, fmt.Sprintf("%s#%d", sheet, rowNum), row); err != nil {
						log.WithFields(log.Fields{"sheet": sheet, "row": rowNum, "file": filePath}).Errorf("handle data failed: %s", err)
					}
				}
			}
		}

		return nil
	}
}

func createExcelFileStreamWalker(filePath string, onlyHeader bool) FileWalker {
	return func(headerCB func(filepath string, headers []string) error, dataCB func(filepath string, id string, data []string) error) error {
		xl, err := xlsxreader.OpenFile(filePath)
		if err != nil {
			return err
		}
		defer xl.Close()

		for i, sheet := range xl.Sheets {
			var columnCount, index int
			for row := range xl.ReadRows(sheet) {
				index++
				if i > 0 {
					log.Warningf("file %s has more than one sheet, only the first sheet will be processed", filePath)
					return nil
				}

				if onlyHeader && index > 1 {
					break
				}

				if index == 1 {
					values := array.Map(row.Cells, func(cell xlsxreader.Cell, _ int) string { return cell.Value })
					columnCount = len(values)

					if err := headerCB(filePath, values); err != nil {
						log.WithFields(log.Fields{"file": filePath}).Errorf("handle header failed: %s", err)
						return err
					}

					continue
				}

				values := make([]string, columnCount)
				for i, cell := range row.Cells {
					if cell.ColumnIndex() == -1 {
						values[i] = cell.Value
					} else {
						if cell.ColumnIndex() >= columnCount {
							continue
						}
						values[cell.ColumnIndex()] = cell.Value
					}
				}

				if err := dataCB(filePath, fmt.Sprintf("%s#%d", sheet, row.Index), values); err != nil {
					log.WithFields(log.Fields{"index": row.Index, "file": filePath}).Errorf("handle data failed: %s", err)
				}
			}
		}

		return nil
	}
}
