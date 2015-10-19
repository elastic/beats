package harvester

import (
	"io"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Decoder wraps a reader for decoding input to utf-8 on read.
type Decoder func(io.Reader) io.Reader

var encodings = map[string]Decoder{
	// default
	"nop":   Plain,
	"plain": Plain,

	// utf8 (validate input)
	"utf-8": trans(encoding.UTF8Validator),

	// utf16
	"utf-16be-bom": enc(unicode.UTF16(unicode.BigEndian, unicode.UseBOM)),
	"utf-16be":     enc(unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)),
	"utf-16le":     enc(unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)),

	// traditional chinese
	"big5": enc(traditionalchinese.Big5),

	// simplified chinese
	"gb18030":  enc(simplifiedchinese.GB18030),
	"gbk":      enc(simplifiedchinese.GBK),
	"hzgb2312": enc(simplifiedchinese.HZGB2312),

	// korean
	"euckr": enc(korean.EUCKR),

	// japanese
	"eucjp":     enc(japanese.EUCJP),
	"iso2022jp": enc(japanese.ISO2022JP),
	"shiftjis":  enc(japanese.ShiftJIS),

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
	return d, ok
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
