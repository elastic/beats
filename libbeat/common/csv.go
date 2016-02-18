package common

import (
	"bytes"
	"encoding/csv"
	"strings"
)

// DumpInCSVFormat takes a set of fields and rows and returns a string
// representing the CSV representation for the fields and rows.
func DumpInCSVFormat(fields []string, rows [][]string) string {

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	for i, field := range fields {
		fields[i] = strings.Replace(field, "\n", "\\n", -1)
	}
	if len(fields) > 0 {
		writer.Write(fields)
	}

	for _, row := range rows {
		for i, field := range row {
			field = strings.Replace(field, "\n", "\\n", -1)
			field = strings.Replace(field, "\r", "\\r", -1)
			row[i] = field
		}
		writer.Write(row)
	}
	writer.Flush()

	csv := buf.String()
	return csv
}
