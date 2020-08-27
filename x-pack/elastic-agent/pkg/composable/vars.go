package composable

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/elastic/go-ucfg"
)

var varsRegex = regexp.MustCompile(`{{([\p{L}\d\s\-|.'"]*)}}`)

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
			invalidErr := fmt.Errorf("invalid variable: %s", value[r[i]:r[i+1]])
			values, ok := splitPipes([]rune(value[r[i+2]:r[i+3]]))
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

// splitPipes split value at |, ignoring | in quotes.
func splitPipes(s []rune) ([]string, bool) {
	next := 0
	in := false
	result := []string{}
	for i := 0; i < len(s); i++ {
		if s[i] == '|' && !in {
			// split on |, because not in a string
			result = append(result, string(s[next:i]))
			next = i + 1
		} else if unicode.IsSpace(s[i]) && !in {
			// space only allowed when in string
			return result, false
		} else if s[i] == '"' {
			if !in {
				// start of a double quoted string
				in = true
			} else if i > 0 && s[i-1] != '\\' {
				// end of double quoted string; \" would not exit out of string
				in = false
			}
		} else if s[i] == '\'' {
			if !in {
				// start of a single quoted string
				in = true
			} else if i > 0 && s[i-1] != '\\' {
				// end of single quoted string; \" would not exit out of string
				in = false
			}
		}
	}
	if in {
		return result, false
	}
	return append(result, string(s[next:])), true
}

func varPrefixMatched(val string, key string) bool {
	s := strings.SplitN(val, ".", 2)
	return s[0] == key
}
