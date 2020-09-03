// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/elastic/go-ucfg"
)

var varsRegex = regexp.MustCompile(`\${([\p{L}\d\s\\\-_|.'"]*)}`)

// ErrNoMatch is return when the replace didn't fail, just that no vars match to perform the replace.
var ErrNoMatch = fmt.Errorf("no matching vars")

// Vars is a context of variables that also contain a list of processors that go with the mapping.
type Vars struct {
	Mapping map[string]interface{}

	ProcessorsKey string
	Processors    Processors
}

// Replace returns a new value based on variable replacement.
func (v *Vars) Replace(value string) (Node, error) {
	var processors []map[string]interface{}
	c, err := ucfg.NewFrom(v.Mapping, ucfg.PathSep("."))
	if err != nil {
		return nil, err
	}
	matchIdxs := varsRegex.FindAllSubmatchIndex([]byte(value), -1)
	if !validBrackets(value, matchIdxs) {
		return nil, fmt.Errorf("starting ${ is missing ending }")
	}

	result := ""
	lastIndex := 0
	for _, r := range matchIdxs {
		for i := 0; i < len(r); i += 4 {
			vars, err := extractVars(value[r[i+2]:r[i+3]])
			if err != nil {
				return nil, fmt.Errorf(`error parsing variable "%s": %s`, value[r[i]:r[i+1]], err)
			}
			set := false
			for _, val := range vars {
				switch val.(type) {
				case *constString:
					result += value[lastIndex:r[0]] + val.Value()
					set = true
				case *varString:
					if r[i] == 0 && r[i+1] == len(value) {
						// possible for complete replacement of object, because the variable
						// is not inside of a string
						child, err := c.Child(val.Value(), -1, ucfg.PathSep("."))
						if err == nil {
							ast, err := NewASTFromConfig(child)
							if err == nil {
								if v.ProcessorsKey != "" && varPrefixMatched(val.Value(), v.ProcessorsKey) {
									processors = v.Processors
								}
								return attachProcessors(ast.root, processors), nil
							}
						}
					}
					replace, err := c.String(val.Value(), -1, ucfg.PathSep("."))
					if err == nil {
						result += value[lastIndex:r[0]] + replace
						set = true
						if v.ProcessorsKey != "" && varPrefixMatched(val.Value(), v.ProcessorsKey) {
							processors = v.Processors
						}
					}
				}
				if set {
					break
				}
			}
			if !set {
				return NewStrVal(""), ErrNoMatch
			}
			lastIndex = r[1]
		}
	}
	return NewStrValWithProcessors(result+value[lastIndex:], processors), nil
}

// validBrackets returns true when all starting {$ have a matching ending }.
func validBrackets(s string, matchIdxs [][]int) bool {
	result := ""
	lastIndex := 0
	match := false
	for _, r := range matchIdxs {
		match = true
		for i := 0; i < len(r); i += 4 {
			result += s[lastIndex:r[0]]
			lastIndex = r[1]
		}
	}
	if !match {
		return !strings.Contains(s, "${")
	}
	return !strings.Contains(result, "${")
}

type varI interface {
	Value() string
}

type varString struct {
	value string
}

func (v *varString) Value() string {
	return v.value
}

type constString struct {
	value string
}

func (v *constString) Value() string {
	return v.value
}

func extractVars(i string) ([]varI, error) {
	const out = rune(0)

	quote := out
	constant := false
	escape := false
	is := make([]rune, 0, len(i))
	res := make([]varI, 0)
	for _, r := range i {
		if r == '|' {
			if escape {
				return nil, fmt.Errorf(`variable pipe cannot be escaped; remove \ before |`)
			}
			if quote == out {
				if constant {
					res = append(res, &constString{string(is)})
				} else if len(is) > 0 {
					if is[len(is)-1] == '.' {
						return nil, fmt.Errorf("variable cannot end with '.'")
					}
					res = append(res, &varString{string(is)})
				}
				is = is[:0] // slice to zero length; to keep allocated memory
				constant = false
			} else {
				is = append(is, r)
			}
			continue
		}
		if !escape && (r == '"' || r == '\'') {
			if quote == out {
				// start of unescaped quote
				quote = r
				constant = true
			} else if quote == r {
				// end of unescaped quote
				quote = out
			} else {
				is = append(is, r)
			}
			continue
		}
		// escape because of backslash (\); except when it is the second backslash of a pair
		escape = !escape && r == '\\'
		if r == '\\' {
			if !escape {
				is = append(is, r)
			}
		} else if quote != out || !unicode.IsSpace(r) {
			is = append(is, r)
		}
	}
	if quote != out {
		return nil, fmt.Errorf(`starting %s is missing ending %s`, string(quote), string(quote))
	}
	if constant {
		res = append(res, &constString{string(is)})
	} else if len(is) > 0 {
		if is[len(is)-1] == '.' {
			return nil, fmt.Errorf("variable cannot end with '.'")
		}
		res = append(res, &varString{string(is)})
	}
	return res, nil
}

func varPrefixMatched(val string, key string) bool {
	s := strings.SplitN(val, ".", 2)
	return s[0] == key
}
