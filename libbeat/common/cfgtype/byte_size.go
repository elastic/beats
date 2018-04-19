package cfgtype

import (
	"unicode"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
)

// ByteSize defines a new configuration option that will parse `go-humanize` compatible values into a
// int64 when the suffix is valid or will fallback to bytes.
type ByteSize int64

// Unpack converts a size defined from a human readable format into bytes.
func (s *ByteSize) Unpack(v string) error {
	sz, err := humanize.ParseBytes(v)
	if isRawBytes(v) {
		cfgwarn.Deprecate("7.0", "size now requires a unit (KiB, MiB, etc...), current value: %s.", v)
	}
	if err != nil {
		return err
	}

	*s = ByteSize(sz)
	return nil
}

func isRawBytes(v string) bool {
	for _, c := range v {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
