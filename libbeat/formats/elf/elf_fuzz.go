// +build fuzz

package elf

import (
	"bytes"
)

func Fuzz(data []byte) int {
	Parse(bytes.NewReader(data))
	return 0
}
