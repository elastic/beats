package harvester

import (
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Decoder wraps a reader for decoding input to utf-8 on read.
type Decoder func(io.Reader) io.Reader

var encodings = map[string]encoding.Encoding{
	// default
	"nop":   Plain,
	"plain": Plain,

	// utf8 (validate input) - shadow htmlindex utf8 codecs not validating input
	"unicode-1-1-utf-8": utf8Encoding{},
	"utf-8":             utf8Encoding{},
	"utf8":              utf8Encoding{},

	// utf16
	// "utf-16be-bom": unicode.UTF16(unicode.BigEndian, unicode.UseBOM),

	// simplified chinese
	"gbk": simplifiedchinese.GBK, // shadow htmlindex using 'GB10830' for GBK

	// 8bit charmap encodings
	"iso8859-6e": charmap.ISO8859_6E,
	"iso8859-6i": charmap.ISO8859_6I,
	"iso8859-8e": charmap.ISO8859_8E,
	"iso8859-8i": charmap.ISO8859_8I,
}

// Plain file encoding not transforming any read bytes.
var Plain = encoding.Nop

// UTF-8 encoding copying input to output sequence replacing invalid UTF-8
// converted to '\uFFFD'.
//
// See: http://encoding.spec.whatwg.org/#replacement
type utf8Encoding struct{}

func (utf8Encoding) NewDecoder() transform.Transformer {
	return encoding.Replacement.NewEncoder()
}

func (utf8Encoding) NewEncoder() transform.Transformer {
	return encoding.Replacement.NewEncoder()
}

// Find returns
func findEncoding(name string) (encoding.Encoding, bool) {
	if name == "" {
		return Plain, true
	}
	d, ok := encodings[strings.ToLower(name)]
	if ok {
		return d, ok
	}

	codec, err := htmlindex.Get(name)
	return codec, err == nil
}
