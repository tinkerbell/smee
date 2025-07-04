package script

import (
	"bytes"
	"text/template"
)

func GenerateTemplate(d any, script string) (string, error) {
	t := template.New("auto.ipxe")
	t, err := t.Parse(script)
	if err != nil {
		return "", err
	}
	buffer := new(bytes.Buffer)
	if err := t.Execute(buffer, d); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
