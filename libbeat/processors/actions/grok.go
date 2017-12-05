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
			allowedFields("field", "patterns", "additional_patern_definitions", "when")))
}

func newGrok(c common.Config) (processors.Processor, error) {
	type config struct {
		Field                        string   `config:"field"`
		Patterns                     []string `config:"patterns"`
		AdditionalPatternDefinitions []string `config:"additional_pattern_definitions"`
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
		var patternStart string
		if pattern[0] == '^' {
			patternStart = pattern
		} else {
			patternStart = "^" + pattern
		}
		regexps[i], err = regexp.Compile(patternStart)
		if err != nil {
			logp.Warn("Error compiling regular expression: `%s'", pattern)
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
				matches := regexp.FindStringSubmatch(field)
				if matches != nil {
					subexps := regexp.SubexpNames()
					for i, subexp := range subexps {
						if i > 0 {
							event[subexp] = matches[i]
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
