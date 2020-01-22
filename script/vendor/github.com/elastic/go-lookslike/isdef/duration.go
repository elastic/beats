package isdef

import (
	"fmt"
	"time"

	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// IsDuration tests that the given value is a duration.
var IsDuration = Is("is a duration", func(path llpath.Path, v interface{}) *llresult.Results {
	if _, ok := v.(time.Duration); ok {
		return llresult.ValidResult(path)
	}
	return llresult.SimpleResult(
		path,
		false,
		fmt.Sprintf("Expected a time.duration, got '%v' which is a %T", v, v),
	)
})
