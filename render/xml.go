package render

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/mylxsw/go-utils/array"
)

type XMLField struct {
	XMLName xml.Name    `xml:"field"`
	Name    string      `xml:"name,attr"`
	Value   interface{} `xml:",chardata"`
}

type XMLRow struct {
	XMLName xml.Name `xml:"row"`
	Value   []XMLField
}

type XMLResultSet struct {
	XMLName   xml.Name `xml:"resultset"`
	Statement string   `xml:"statement,attr"`
	XMLNS     string   `xml:"xmlns:xsi,attr"`
	Value     []XMLRow
}

func XML(w io.Writer, colNames []string, data []map[string]interface{}, sqlStr string) error {
	result := XMLResultSet{
		Statement: sqlStr,
		XMLNS:     "http://www.w3.org/2001/XMLSchema-instance",
		Value: array.Map(data, func(item map[string]interface{}) XMLRow {
			row := XMLRow{Value: make([]XMLField, 0)}
			for k, v := range item {
				row.Value = append(row.Value, XMLField{
					Name:  k,
					Value: v,
				})
			}

			return row
		}),
	}

	marshalData, err := xml.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, xml.Header+string(marshalData))
	if err != nil {
		return err
	}

	return nil
}
