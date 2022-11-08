package render

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

func YAML(w io.Writer, data interface{}) error {
	marshalData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, string(marshalData))
	return err
}
