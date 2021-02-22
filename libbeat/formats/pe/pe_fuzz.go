// +build fuzz

package pe

import (
	"bytes"
)

func Fuzz(data []byte) int {
	Parse(bytes.NewReader(data))
	return 0
}
