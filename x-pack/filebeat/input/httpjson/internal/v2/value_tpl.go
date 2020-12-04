// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

type valueTpl struct {
	*template.Template
}

func (t *valueTpl) Unpack(in string) error {
	tpl, err := template.New("").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"now":                 now,
			"parseDate":           parseDate,
			"formatDate":          formatDate,
			"parseDuration":       parseDuration,
			"parseTimestamp":      parseTimestamp,
			"parseTimestampMilli": parseTimestampMilli,
			"parseTimestampNano":  parseTimestampNano,
			"getRFC5988Link":      getRFC5988Link,
			"toInt":               toInt,
			"add":                 add,
		}).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

func (t *valueTpl) Execute(trCtx *transformContext, tr transformable, defaultVal *valueTpl, log *logp.Logger) (val string) {
	fallback := func(err error) string {
		if err != nil {
			log.Debugf("template execution failed: %v", err)
		}
		if defaultVal != nil {
			log.Debugf("template execution: falling back to default value")
			return defaultVal.Execute(emptyTransformContext(), transformable{}, nil, log)
		}
		return ""
	}

	defer func() {
		if r := recover(); r != nil {
			val = fallback(r.(error))
		}
		log.Debugf("template execution: evaluated template %q", val)
	}()

	buf := new(bytes.Buffer)
	data := tr.Clone()
	data.Put("cursor", trCtx.cursorMap())
	data.Put("last_event", trCtx.lastEventClone())
	data.Put("last_response", trCtx.lastResponseClone().templateValues())

	if err := t.Template.Execute(buf, data); err != nil {
		return fallback(err)
	}

	val = buf.String()
	if val == "" || strings.Contains(val, "<no value>") {
		return fallback(nil)
	}
	return val
}

var (
	predefinedLayouts = map[string]string{
		"ANSIC":       time.ANSIC,
		"UnixDate":    time.UnixDate,
		"RubyDate":    time.RubyDate,
		"RFC822":      time.RFC822,
		"RFC822Z":     time.RFC822Z,
		"RFC850":      time.RFC850,
		"RFC1123":     time.RFC1123,
		"RFC1123Z":    time.RFC1123Z,
		"RFC3339":     time.RFC3339,
		"RFC3339Nano": time.RFC3339Nano,
		"Kitchen":     time.Kitchen,
	}
)

func now(add ...time.Duration) time.Time {
	now := timeNow().UTC()
	if len(add) == 0 {
		return now
	}
	return now.Add(add[0])
}

func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

func parseDate(date string, layout ...string) time.Time {
	var ly string
	if len(layout) == 0 {
		ly = "RFC3339"
	} else {
		ly = layout[0]
	}
	if found := predefinedLayouts[ly]; found != "" {
		ly = found
	}

	t, err := time.Parse(ly, date)
	if err != nil {
		return time.Time{}
	}

	return t.UTC()
}

func formatDate(date time.Time, layouttz ...string) string {
	var layout, tz string
	switch {
	case len(layouttz) == 0:
		layout = "RFC3339"
	case len(layouttz) == 1:
		layout = layouttz[0]
	case len(layouttz) > 1:
		layout, tz = layouttz[0], layouttz[1]
	}

	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	}

	if loc, err := time.LoadLocation(tz); err == nil {
		date = date.In(loc)
	} else {
		date = date.UTC()
	}

	return date.Format(layout)
}

func parseTimestamp(s int64) time.Time {
	return time.Unix(s, 0).UTC()
}

func parseTimestampMilli(ms int64) time.Time {
	return time.Unix(0, ms*1e6).UTC()
}

func parseTimestampNano(ns int64) time.Time {
	return time.Unix(0, ns).UTC()
}

var regexpLinkRel = regexp.MustCompile(`<(.*)>;.*\srel\="?([^;"]*)`)

func getRFC5988Link(rel string, links []string) string {
	for _, link := range links {
		if !regexpLinkRel.MatchString(link) {
			continue
		}

		matches := regexpLinkRel.FindStringSubmatch(link)
		if len(matches) != 3 {
			continue
		}

		if matches[2] != rel {
			continue
		}

		return matches[1]
	}

	return ""
}

func toInt(s string) int {
	i, _ := strconv.ParseInt(s, 10, 64)
	return int(i)
}

func add(vs ...int) int {
	var sum int
	for _, v := range vs {
		sum += v
	}
	return sum
}
