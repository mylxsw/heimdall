package render

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

func YAML(w io.Writer, data interface{}) {
	marshalData, err := yaml.Marshal(data)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprint(w, string(marshalData))
	if err != nil {
		panic(err)
	}
}
