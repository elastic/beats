package salesforce

import (
	"fmt"
	"text/template"
)

// we define custom delimiters to prevent issues when using template values as part of other Go templates.
const (
	leftDelim  = "[["
	rightDelim = "]]"
)

type valueTpl struct {
	*template.Template
}

func (t *valueTpl) Unpack(in string) error {
	tpl, err := template.New("").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"sprintf": fmt.Sprintf,
		}).
		Delims(leftDelim, rightDelim).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}
