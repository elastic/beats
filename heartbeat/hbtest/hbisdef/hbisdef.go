package hbisdef

import (
	"time"

	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

var IsTime = isdef.Is("a timestamp", func(path llpath.Path, v interface{}) *llresult.Results {
	if _, ok := v.(time.Time); !ok {
		return llresult.SimpleResult(path, false, "'%v' is not a time.Time, it is a '%T'", v, v)
	}
	return llresult.ValidResult(path)
})
