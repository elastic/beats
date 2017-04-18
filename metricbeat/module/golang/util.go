package golang

import (
	"bytes"
	"github.com/elastic/beats/libbeat/logp"
	"strings"
)

/**
Convert cmd array to cmd line
*/
func GetCmdStr(v interface{}) interface{} {
	switch t := v.(type) {
	case []interface{}:
		var buffer bytes.Buffer
		strs := v.([]interface{})
		for _, v := range strs {
			buffer.WriteString(v.(string))
			buffer.WriteString(" ")
		}
		return strings.TrimRight(buffer.String(), " ")
	default:
		logp.Debug("golang", "unexpected cmdline, %v, %v", t, v)
		return v
	}

}
