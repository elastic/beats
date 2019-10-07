// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

	"iso8859-1":  enc(charmap.ISO8859_1),  // latin-1
	"iso8859-2":  enc(charmap.ISO8859_2),  // latin-2
	"iso8859-3":  enc(charmap.ISO8859_3),  // latin-3
	"iso8859-4":  enc(charmap.ISO8859_4),  // latin-4
	"iso8859-5":  enc(charmap.ISO8859_5),  // latin/cyrillic
	"iso8859-6":  enc(charmap.ISO8859_6),  // latin/arabic
	"iso8859-7":  enc(charmap.ISO8859_7),  // latin/greek
	"iso8859-8":  enc(charmap.ISO8859_8),  // latin/hebrew
	"iso8859-9":  enc(charmap.ISO8859_9),  // latin-5
	"iso8859-10": enc(charmap.ISO8859_10), // latin-6
	"iso8859-13": enc(charmap.ISO8859_13), // latin-7
	"iso8859-14": enc(charmap.ISO8859_14), // latin-8
	"iso8859-15": enc(charmap.ISO8859_15), // latin-9
	"iso8859-16": enc(charmap.ISO8859_16), // latin-10

	// ibm codepages
	"cp437":       enc(charmap.CodePage437),
	"cp850":       enc(charmap.CodePage850),
	"cp852":       enc(charmap.CodePage852),
	"cp855":       enc(charmap.CodePage855),
	"cp858":       enc(charmap.CodePage858),
	"cp860":       enc(charmap.CodePage860),
	"cp862":       enc(charmap.CodePage862),
	"cp863":       enc(charmap.CodePage863),
	"cp865":       enc(charmap.CodePage865),
	"cp866":       enc(charmap.CodePage866),
	"ebcdic-037":  enc(charmap.CodePage037),
	"ebcdic-1040": enc(charmap.CodePage1140),
	"ebcdic-1047": enc(charmap.CodePage1047),

	// cyrillic
	"koi8r": enc(charmap.KOI8R),
	"koi8u": enc(charmap.KOI8U),

	// macintosh
	"macintosh":          enc(charmap.Macintosh),
	"macintosh-cyrillic": enc(charmap.MacintoshCyrillic),

	// windows
	"windows1250": enc(charmap.Windows1250), // central and eastern european
	"windows1251": enc(charmap.Windows1251), // russian, serbian cyrillic
	"windows1252": enc(charmap.Windows1252), // legacy
	"windows1253": enc(charmap.Windows1253), // modern greek
	"windows1254": enc(charmap.Windows1254), // turkish
	"windows1255": enc(charmap.Windows1255), // hebrew
	"windows1256": enc(charmap.Windows1256), // arabic
	"windows1257": enc(charmap.Windows1257), // estonian, latvian, lithuanian
	"windows1258": enc(charmap.Windows1258), // vietnamese
	"windows874":  enc(charmap.Windows874),

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
