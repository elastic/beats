package outputs

import (
	"bytes"
	"github.com/elastic/beats/libbeat/common"
	"regexp"
	"text/template"
)

// Format a given event based on a format string
// Sample formatter looks like: "%{@timestamp} %{message}"
func FormatEvent(event common.MapStr, format string) ([]byte, error) {
	var err error
	var buffer bytes.Buffer

	r := regexp.MustCompile("\\%{(.*?)\\}")
	tl := r.ReplaceAllString(format, "{{or (index . \"$1\") \"\"}}")

	t, err := template.New("template").Parse(tl)
	if err != nil {
		return nil, err
	}

	err = t.Execute(&buffer, event)
	if err != nil {
		return nil, err
	}

	return []byte(buffer.Bytes()), nil
}
