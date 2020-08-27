package composable

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/elastic/go-ucfg"
)

var varsRegex = regexp.MustCompile(`{{([\p{L}\d\s\\\-|.'"]*)}}`)

// NoMatchErr is return when the replace didn't fail, just that no vars match to perform the replace.
var NoMatchErr = fmt.Errorf("no matching vars")

// Vars is a context of variables that also contain a list of processors that go with the mapping.
type Vars struct {
	Mapping map[string]interface{}

	ProcessorsKey string
	Processors    []map[string]interface{}
}

// VarsCallback is callback called when the current vars state changes.
type VarsCallback func([]Vars)

// Replace returns a new value based on variable replacement.
func (v *Vars) Replace(value string) (string, []map[string]interface{}, error) {
	var processors []map[string]interface{}
	c, err := ucfg.NewFrom(v.Mapping, ucfg.PathSep("."))
	if err != nil {
		return "", nil, err
	}

	result := ""
	lastIndex := 0
	for _, r := range varsRegex.FindAllSubmatchIndex([]byte(value), -1) {
		for i := 0; i < len(r); i += 4 {
			varContent := value[r[i+2]:r[i+3]]
			invalidErr := fmt.Errorf("invalid variable: %s", value[r[i]:r[i+1]])
			values, ok := splitPipes(varContent)
			if !ok {
				return "", nil, invalidErr
			}
			set := false
			for _, val := range values {
				if val == "" {
					continue
				}
				if val[0] == '"' || val[0] == '\'' {
					result += value[lastIndex:r[0]] + val[1:len(val)-1]
					set = true
					break
				}
				if val[len(val)-1] == '.' {
					return "", nil, invalidErr
				}
				replace, err := c.String(val, -1, ucfg.PathSep("."))
				if err == nil {
					result += value[lastIndex:r[0]] + replace
					set = true
					if v.ProcessorsKey != "" && varPrefixMatched(val, v.ProcessorsKey) {
						processors = v.Processors
					}
					break
				}
			}
			if !set {
				return "", nil, NoMatchErr
			}
			lastIndex = r[1]
		}
	}
	return result + value[lastIndex:], processors, nil
}

// stripSpace removes all the spaces unless they are quoted
func stripSpace(s string) ([]rune, bool) {
	const out = rune(0)

	quote := out
	escape := false
	rs := make([]rune, 0, len(s))
	for _, r := range s {
		if !escape {
			if r == '"' || r == '\'' {
				if quote == out {
					// start of unescaped quote
					quote = r
				} else if quote == r {
					// end of unescaped quote
					quote = out
				}
			}
		}
		// escape because of backslash (\); except when it is the second backslash of a pair
		escape = !escape && r == '\\'
		if quote != out || !unicode.IsSpace(r) {
			rs = append(rs, r)
		}
	}
	if quote != out {
		return []rune(""), false
	}
	return rs, true
}

// stripEscapes strips escapes from quoted strings.
func stripEscapes(s string) string {
	if s == "" {
		return s
	}
	if s[0] == '\'' {
		return strings.Replace(s, `\'`, `'`, -1)
	}
	if s[0] == '"' {
		return strings.Replace(s, `\"`, `"`, -1)
	}
	return s
}

// splitPipes split value at |, ignoring | in quotes.
func splitPipes(input string) ([]string, bool) {
	const out = rune(0)
	s, ok := stripSpace(input)
	if !ok {
		return nil, false
	}

	next := 0
	result := []string{}
	quote := out
	escape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '|' && quote == out {
			// split on |, because not in a string
			result = append(result, stripEscapes(string(s[next:i])))
			next = i + 1
		} else if !escape && (s[i] == '"' || s[i] == '\'') {
			if quote == out {
				// start of unescaped quote
				quote = s[i]
			} else if quote == s[i] {
				// end of unescaped quote
				quote = out
			}
		}
		// escape because of backslash (\); except when it is the second backslash of a pair
		escape = !escape && s[i] == '\\'
	}
	if quote != out {
		return nil, false
	}
	return append(result, stripEscapes(string(s[next:]))), true
}

func varPrefixMatched(val string, key string) bool {
	s := strings.SplitN(val, ".", 2)
	return s[0] == key
}
