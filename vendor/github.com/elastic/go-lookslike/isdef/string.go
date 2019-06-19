package isdef

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// isStrCheck is a helper for IsDefs that must assert that the value is a string first.
func isStrCheck(path llpath.Path, v interface{}) (str string, errorResults *llresult.Results) {
	strV, ok := v.(string)

	if !ok {
		return "", llresult.SimpleResult(
			path,
			false,
			fmt.Sprintf("Unable to convert '%v' to string", v),
		)
	}

	return strV, nil
}

// IsString checks that the given value is a string.
var IsString = Is("is a string", func(path llpath.Path, v interface{}) *llresult.Results {
	_, errorResults := isStrCheck(path, v)
	if errorResults != nil {
		return errorResults
	}

	return llresult.ValidResult(path)
})

// IsNonEmptyString checks that the given value is a string and has a length > 1.
var IsNonEmptyString = Is("is a non-empty string", func(path llpath.Path, v interface{}) *llresult.Results {
	strV, errorResults := isStrCheck(path, v)
	if errorResults != nil {
		return errorResults
	}

	if len(strV) == 0 {
		return llresult.SimpleResult(path, false, "String '%s' should not be empty", strV)
	}

	return llresult.ValidResult(path)
})

// IsStringMatching checks whether a value matches the given regexp.
func IsStringMatching(regexp *regexp.Regexp) IsDef {
	return Is("is string matching regexp", func(path llpath.Path, v interface{}) *llresult.Results {
		strV, errorResults := isStrCheck(path, v)
		if errorResults != nil {
			return errorResults
		}

		if !regexp.MatchString(strV) {
			return llresult.SimpleResult(
				path,
				false,
				fmt.Sprintf("String '%s' did not match regexp %s", strV, regexp.String()),
			)
		}

		return llresult.ValidResult(path)
	})
}

// IsStringContaining validates that the the actual value contains the specified substring.
func IsStringContaining(needle string) IsDef {
	return Is("is string containing", func(path llpath.Path, v interface{}) *llresult.Results {
		strV, errorResults := isStrCheck(path, v)
		if errorResults != nil {
			return errorResults
		}

		if !strings.Contains(strV, needle) {
			return llresult.SimpleResult(
				path,
				false,
				fmt.Sprintf("String '%s' did not contain substring '%s'", strV, needle),
			)
		}

		return llresult.ValidResult(path)
	})
}
