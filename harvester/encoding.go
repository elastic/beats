package harvester

import (
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Decoder wraps a reader for decoding input to utf-8 on read.
type Decoder func(io.Reader) io.Reader

var encodings = map[string]Decoder{
	// default
	"nop":   Plain,
	"plain": Plain,

	// utf8 (validate input) - shadow htmlindex utf8 codecs not validating input
	"unicode-1-1-utf-8": trans(encoding.UTF8Validator),
	"utf-8":             trans(encoding.UTF8Validator),
	"utf8":              trans(encoding.UTF8Validator),

	// utf16
	"utf-16be-bom": enc(unicode.UTF16(unicode.BigEndian, unicode.UseBOM)),

	// simplified chinese
	"gbk": enc(simplifiedchinese.GBK), // shadow htmlindex using 'GB10830' for GBK

	// 8bit charmap encodings
	"iso8859-6e": enc(charmap.ISO8859_6E),
	"iso8859-6i": enc(charmap.ISO8859_6I),
	"iso8859-8e": enc(charmap.ISO8859_8E),
	"iso8859-8i": enc(charmap.ISO8859_8I),
}

// Plain file encoding not transforming any read bytes.
var Plain = nopEnc

// Find returns
func findEncoding(name string) (Decoder, bool) {
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

func nopEnc(r io.Reader) io.Reader { return r }

func enc(encoding encoding.Encoding) Decoder {
	return trans(encoding.NewDecoder())
}

func trans(t transform.Transformer) Decoder {
	return func(r io.Reader) io.Reader {
		return transform.NewReader(r, t)
	}
}
