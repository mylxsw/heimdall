package commands_test

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/array"
	"github.com/xuri/excelize/v2"
)

func TestSplit(t *testing.T) {
	headerLine := 2
	headers, records, err := splitExcel("/Users/mylxsw/Downloads/2.xlsx", headerLine)
	if err != nil {
		t.Error(err)
		return
	}

	for _, header := range headers {
		log.Infof("header: %v", header)
	}

	for _, record := range records {
		log.Infof("record: %v", record)
	}
}

func splitExcel(filename string, headerLine int) (headers [][]string, records [][]string, err error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	for i, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil, nil, err
		}

		if i > 0 && len(rows) > 0 {
			log.Warningf("file has more than one sheet, only the first sheet will be processed")
			break
		}

		if len(rows) < headerLine+1 {
			continue
		}

		headers = append(headers, rows[:headerLine]...)
		records = append(records, rows[headerLine:]...)
	}

	return headers, records, nil
}

func splitCSV(filename string, headerLine int, csvSepertor rune) (headers [][]string, records [][]string, err error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, err
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

			return nil, nil, err
		}

		if index == 1 {
			if err := headerCB(filePath, array.Map(record, func(item string, _ int) string { return clean(item) })); err != nil {
				log.WithFields(log.Fields{"file": filePath}).Errorf("handle header failed: %s", err)
				return err
			}

			continue
		}

		if err := dataCB(filePath, fmt.Sprintf("%d", index), record); err != nil {
			log.WithFields(log.Fields{"index": index, "file": filePath}).Errorf("handle data failed: %s", err)
		}
	}

	return headers, records, nil
}
