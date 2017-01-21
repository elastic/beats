package harvester

import "regexp"

// MatchAnyRegexps checks if the text matches any of the regular expressions
func MatchAnyRegexps(regexps []*regexp.Regexp, text string) bool {

	for _, rexp := range regexps {
		if rexp.MatchString(text) {
			return true
		}
	}

	return false
}
