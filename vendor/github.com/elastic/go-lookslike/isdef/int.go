package isdef

import (
	"fmt"

	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

func intGtChecker(than int) ValueValidator {
	return func(path llpath.Path, v interface{}) *llresult.Results {
		n, ok := v.(int)
		if !ok {
			msg := fmt.Sprintf("%v is a %T, but was expecting an int!", v, v)
			return llresult.SimpleResult(path, false, msg)
		}

		if n > than {
			return llresult.ValidResult(path)
		}

		return llresult.SimpleResult(
			path,
			false,
			fmt.Sprintf("%v is not greater than %v", n, than),
		)
	}
}

// IsIntGt tests that a value is an int greater than.
func IsIntGt(than int) IsDef {
	return Is("greater than", intGtChecker(than))
}
