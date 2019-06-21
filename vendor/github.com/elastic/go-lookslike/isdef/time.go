package isdef

import (
	"time"

	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// IsEqualToTime ensures that the actual value is the given time, regardless of zone.
func IsEqualToTime(to time.Time) IsDef {
	return Is("equal to time", func(path llpath.Path, v interface{}) *llresult.Results {
		actualTime, ok := v.(time.Time)
		if !ok {
			return llresult.SimpleResult(path, false, "Value %t was not a time.Time", v)
		}

		if actualTime.Equal(to) {
			return llresult.ValidResult(path)
		}

		return llresult.SimpleResult(path, false, "actual(%v) != expected(%v)", actualTime, to)
	})
}
