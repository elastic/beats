package json

import "github.com/elastic/go-structform/internal/unsafe"

var (
	nullSymbol  = []byte("null")
	trueSymbol  = []byte("true")
	falseSymbol = []byte("false")
	commaSymbol = []byte(",")

	invalidCharSym = []byte(`\ufffd`)
)

func str2Bytes(s string) []byte {
	return unsafe.Str2Bytes(s)
}

func bytes2Str(b []byte) string {
	return unsafe.Bytes2Str(b)
}
