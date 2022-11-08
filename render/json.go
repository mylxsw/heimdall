package render

import (
	"encoding/json"
	"fmt"
	"io"
)

func JSON(w io.Writer, data interface{}) error {
	marshalData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprint(w, string(marshalData))
	return err
}
