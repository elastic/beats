package actions

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

type grok struct {
	Field    string
	Patterns []*regexp.Regexp
}

func init() {
	processors.RegisterPlugin("grok",
		configChecked(newGrok,
			requireFields("field", "patterns"),
			allowedFields("field", "patterns", "additional_pattern_definitions", "when")))
}

func newGrok(c common.Config) (processors.Processor, error) {
	type config struct {
		Field                        string            `config:"field"`
		Patterns                     []string          `config:"patterns"`
		AdditionalPatternDefinitions map[string]string `config:"additional_pattern_definitions"`
	}

	var myconfig config
	err := c.Unpack(&myconfig)
	if err != nil {
		logp.Warn("Error unpacking config for grok")
		return nil, fmt.Errorf("fail to unpack the grok configuration: %s", err)
	}

	regexps := make([]*regexp.Regexp, len(myconfig.Patterns))
	errInRegexps := false

	for i, pattern := range myconfig.Patterns {
		expandedPattern, err := grokExpandPattern(pattern, []string{}, myconfig.AdditionalPatternDefinitions)
		if err != nil {
			logp.Warn("Error compiling regular expression: `%s', %s", pattern, err)
			errInRegexps = true
		}
		var patternStart string
		if expandedPattern[0] == '^' {
			patternStart = expandedPattern
		} else {
			patternStart = "^" + expandedPattern
		}
		regexps[i], err = regexp.Compile(patternStart)
		if err != nil {
			logp.Warn("Error compiling regular expression: `%s', %s", pattern, err)
			logp.Warn("Pattern exanded: %s", expandedPattern)
			errInRegexps = true
		}
	}
	if errInRegexps {
		return nil, fmt.Errorf("Error compiling regexps")
	}
	return grok{Field: myconfig.Field, Patterns: regexps}, nil
}

func (g grok) Run(event common.MapStr) (common.MapStr, error) {

	fieldi, err := event.GetValue(g.Field)
	if err == nil {
		field, ok := fieldi.(string)
		if ok {
			for _, regexp := range g.Patterns {
				matches := regexp.FindStringSubmatchIndex(field)
				if matches != nil {
					subexps := regexp.SubexpNames()
					for i, subexp := range subexps {
						if len(subexp) > 0 {
							if matches[2*i] >= 0 {
								event[subexp] = field[matches[2*i]:matches[2*i+1]]
							}
						}
					}
					break
				}
			}
		}
	}

	return event, nil
}

func (g grok) String() string {
	var name = "grok={field:" + g.Field + ", patterns = [ "
	for i, regexp := range g.Patterns {
		if i > 0 {
			name = name + ", " + regexp.String()
		} else {
			name = name + regexp.String()
		}
	}
	name = name + "]}"
	return name
}

var grokRegexp = regexp.MustCompile(`%\{(\w+)(?::(\w+))?\}`)

func grokExpandPattern(pattern string, knownGrokNames []string, customPatterns map[string]string) (string, error) {
	matches := grokRegexp.FindAllStringSubmatchIndex(pattern, -1)
	var result []byte
	if matches == nil {
		return pattern, nil
	}
	i := 0
	var errList []error
	for _, match := range matches {
		patternName := pattern[match[2]:match[3]]
		patternExpand, err := grokSearchPattern(patternName, knownGrokNames, customPatterns)
		if err != nil {
			errList = append(errList, err)
			continue
		}
		if len(errList) == 0 {
			if len(match) >= 6 && match[4] >= 0 && match[5] >= 0 {
				substName := pattern[match[4]:match[5]]
				patternExpand = namedMatch(patternExpand, substName)
			} else {
				patternExpand = unNamedMatch(patternExpand)
			}
			if match[0] >= i+1 {
				result = append(result, pattern[i:match[0]]...)
			}
			result = append(result, patternExpand...)
		}
		i = match[1]
	}
	if len(errList) != 0 {
		return "", fmt.Errorf("Error parsing grok pattern: %v", errList)
	}
	if i < len(pattern) {
		result = append(result, pattern[i:]...)
	}
	return string(result), nil
}

func grokSearchPattern(patternName string, knownGrokNames []string, customPatterns map[string]string) (string, error) {
	recursion := false
	for _, usedName := range knownGrokNames {
		if usedName == patternName {
			recursion = true
		}
	}
	if recursion {
		return "", fmt.Errorf("detected recursion in grok name '%s'", patternName)
	}

	patterns := getGrokBuiltinPattern()
	regexpVal, ok := customPatterns[patternName]
	if !ok {
		regexpVal, ok = patterns[patternName]
	}
	if !ok {
		return "", fmt.Errorf("unknown grok name '%s'", patternName)
	}
	knownGrokNames2 := append(knownGrokNames, patternName)
	return grokExpandPattern(regexpVal, knownGrokNames2, customPatterns)
}

func namedMatch(pattern string, name string) string {
	return "(?P<" + name + ">" + pattern + ")"
}

func unNamedMatch(pattern string) string {
	return "(?:" + pattern + ")"
}
