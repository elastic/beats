package encoding

import (
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type EncodingFactory func(io.Reader) (Encoding, error)

type Encoding encoding.Encoding

var encodings = map[string]EncodingFactory{
	// default
	"nop":   Plain,
	"plain": Plain,

	// utf8 (validate input) - shadow htmlindex utf8 codecs not validating input
	"unicode-1-1-utf-8": utf8Encoding,
	"utf-8":             utf8Encoding,
	"utf8":              utf8Encoding,

	// simplified chinese
	"gbk": enc(simplifiedchinese.GBK), // shadow htmlindex using 'GB10830' for GBK

	// 8bit charmap encodings
	"iso8859-6e": enc(charmap.ISO8859_6E),
	"iso8859-6i": enc(charmap.ISO8859_6I),
	"iso8859-8e": enc(charmap.ISO8859_8E),
	"iso8859-8i": enc(charmap.ISO8859_8I),

	// utf16 bom codecs (seekable data source required)
	"utf-16-bom":   utf16BOMRequired,
	"utf-16be-bom": utf16BOMBigEndian,
	"utf-16le-bom": utf16BOMLittleEndian,
}

// Plain file encoding not transforming any read bytes.
var Plain = enc(encoding.Nop)

// UTF-8 encoding copying input to output sequence replacing invalid UTF-8
// converted to '\uFFFD'.
//
// See: http://encoding.spec.whatwg.org/#replacement
var utf8Encoding = enc(mixed{})

// FindEncoding searches for an EncodingFactoryby name.
func FindEncoding(name string) (EncodingFactory, bool) {
	if name == "" {
		return Plain, true
	}
	d, ok := encodings[strings.ToLower(name)]
	if ok {
		return d, ok
	}

	codec, err := htmlindex.Get(name)
	if err != nil {
		return nil, false
	}
	return enc(codec), true
}

func enc(e Encoding) EncodingFactory {
	return func(io.Reader) (Encoding, error) {
		return e, nil
	}
}
