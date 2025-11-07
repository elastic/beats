// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"strings"
	"text/template"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type valueTpl struct {
	*template.Template
}

var (
	errEmptyTemplateResult = errors.New("template result is empty")
	errExecuteTemplate     = errors.New("template execution failed")
)

// Execute executes the template with the given data. If the template execution
// fails, then the defaultVal is used if it is not nil. Execute will return
// variable substituted query with nil error.
func (t *valueTpl) Execute(data any, defaultVal *valueTpl, log *logp.Logger) (val string, err error) {
	fallback := func(err error) (string, error) {
		if defaultVal != nil {
			log.Debugf("template execution error: %s", err)
			log.Info("fallback to default template")
			return defaultVal.Execute(mapstr.M{}, nil, log)
		}
		return "", err
	}

	defer func() {
		if r := recover(); r != nil {
			val, err = fallback(errExecuteTemplate)
		}
		if err != nil {
			log.Debugf("template execution failed %s", err)
		}
	}()

	buf := new(strings.Builder)

	err = t.Template.Execute(buf, data)
	if err != nil {
		return fallback(err)
	}

	val = buf.String()
	if val == "" {
		return fallback(errEmptyTemplateResult)
	}

	return val, nil
}

// Unpack parses the given string as a template.
func (t *valueTpl) Unpack(in string) error {
	// Custom delimiters to prevent issues when using template values as part of
	// other Go templates.
	const (
		leftDelim  = "[["
		rightDelim = "]]"
	)

	tpl, err := template.New("").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"now":           timeNow,
			"parseDuration": parseDuration,
			"parseTime":     parseTime,
			"formatTime":    formatTime,
		}).
		Delims(leftDelim, rightDelim).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

// parseDuration parses a duration string and returns the time.Duration value.
func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

// predefinedLayouts contains some predefined layouts that are commonly used.
var predefinedLayouts = map[string]string{
	"ANSIC":             time.ANSIC,
	"UnixDate":          time.UnixDate,
	"RubyDate":          time.RubyDate,
	"RFC822":            time.RFC822,
	"RFC822Z":           time.RFC822Z,
	"RFC850":            time.RFC850,
	"RFC1123":           time.RFC1123,
	"RFC1123Z":          time.RFC1123Z,
	"RFC3339":           time.RFC3339,      // 2006-01-02T15:04:05Z07:00
	"CustomRFC3339Like": formatRFC3339Like, // 2006-01-02T15:04:05.999Z
	"RFC3339Nano":       time.RFC3339Nano,
	"Kitchen":           time.Kitchen,
}

// parseTime parses a time string using the given layout. There are also some
// predefined layouts that can be used; see predefinedLayouts for more.
func parseTime(ts, layout string) time.Time {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	}

	t, _ := time.Parse(layout, ts)
	return t
}

// formatTime formats a time using the given layout. There are also some
// predefined layouts that can be used; see predefinedLayouts for more.
func formatTime(t time.Time, layout string) string {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	}

	return t.Format(layout)
}
