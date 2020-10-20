package transforms

import (
	"bytes"
	"html/template"
)

type Template struct {
	*template.Template
}

func (t *Template) Unpack(in string) error {
	tpl, err := template.New("").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"now":            now,
			"formatDate":     formatDate,
			"parseDate":      parseDate,
			"getRFC5988Link": getRFC5988Link,
		}).
		Parse(in)
	if err != nil {
		return err
	}

	*t = Template{Template: tpl}

	return nil
}

func (t *Template) Execute(tr *Transformable, defaultVal string) (val string) {
	defer func() {
		if r := recover(); r != nil {
			// really ugly
			val = defaultVal
		}
	}()

	buf := new(bytes.Buffer)
	data := map[string]interface{}{
		"header":     tr.Headers.Clone(),
		"body":       tr.Body.Clone(),
		"url.value":  tr.URL.String(),
		"url.params": tr.URL.Query(),
		// "cursor":        tr.Cursor.Clone(),
		// "last_event":    tr.LastEvent,
		// "last_response": tr.LastResponse.Clone(),
	}
	if err := t.Template.Execute(buf, data); err != nil {
		return defaultVal
	}
	return buf.String()
}
