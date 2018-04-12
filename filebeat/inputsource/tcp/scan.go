package tcp

import (
	"bufio"
	"bytes"
)

// factoryDelimiter return a function to split line using a custom delimiter supporting multibytes
// delimiter, the delimiter is stripped from the returned value.
func factoryDelimiter(delimiter []byte) bufio.SplitFunc {
	return func(data []byte, eof bool) (int, []byte, error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, delimiter); i >= 0 {
			return i + len(delimiter), dropDelimiter(data[0:i], delimiter), nil
		}
		if eof {
			return len(data), dropDelimiter(data, delimiter), nil
		}
		return 0, nil, nil
	}
}

func dropDelimiter(data []byte, delimiter []byte) []byte {
	if len(data) > len(delimiter) &&
		bytes.Equal(data[len(data)-len(delimiter):len(data)], delimiter) {
		return data[0 : len(data)-len(delimiter)]
	}
	return data
}
