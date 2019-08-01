// +build gofuzz

package rpm

import "bytes"

// Fuzz tests the parsing and error handling of random byte arrays using
// https://github.com/dvyukov/go-fuzz.
func Fuzz(data []byte) int {
	if p, err := ReadPackageFile(bytes.NewReader(data)); err != nil {
		// handled errors are not very interesting
		return 0
	} else {
		// call some tag handlers
		_ = p.String()
		_ = p.Requires()
		_ = p.Conflicts()
		_ = p.Obsoletes()
		_ = p.Provides()

		// read all index values
		for _, h := range p.Headers {
			for _, x := range h.Indexes {
				switch x.Type {
				case IndexDataTypeBinary:
					_ = h.Indexes.BytesByTag(x.Tag)

				case IndexDataTypeChar, IndexDataTypeInt8, IndexDataTypeInt16, IndexDataTypeInt32, IndexDataTypeInt64:
					_ = h.Indexes.IntsByTag(x.Tag)
					_ = h.Indexes.IntByTag(x.Tag)

				case IndexDataTypeString, IndexDataTypeI8NString, IndexDataTypeStringArray:
					_ = h.Indexes.StringsByTag(x.Tag)
					_ = h.Indexes.StringByTag(x.Tag)
				}
			}
		}

		// everything worked with random input... interesting :|
		return 1
	}
}
