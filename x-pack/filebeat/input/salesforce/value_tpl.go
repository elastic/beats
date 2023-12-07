package salesforce

import (
	"errors"
	"fmt"
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
			"sprintf":                             fmt.Sprintf,
			"formatCurrentTimeWithDurationOffset": formatCurrentTimeWithDurationOffset,
			"formatUnixTimeAsRFC3339":             formatUnixTimeAsRFC3339,
			"formatUnixTimeMilliAsRFC3339":        formatUnixTimeMilliAsRFC3339,
			"formatUnixTimeNanoAsRFC3339":         formatUnixTimeNanoAsRFC3339,
			"parseRFC3339Timestamp":               parseRFC3339Timestamp,
		}).
		Delims(leftDelim, rightDelim).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

func parseRFC3339Timestamp(s string) string {
	now := time.Now().UTC()
	_, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return now.Format(time.RFC3339)
	}
	return s
}

func formatUnixTimeAsRFC3339(sec int64) string {
	return time.Unix(sec, 0).Format(time.RFC3339)
}

func formatUnixTimeMilliAsRFC3339(ms int64) string {
	return time.Unix(0, ms*1e6).Format(time.RFC3339)
}

func formatUnixTimeNanoAsRFC3339(ns int64) string {
	return time.Unix(0, ns).Format(time.RFC3339)
}

func formatCurrentTimeWithDurationOffset(duration string) string {
	now := time.Now().UTC()

	if duration == "" {
		return now.Format(time.RFC3339)
	}

	d, err := time.ParseDuration(duration)
	if err != nil {
		return now.Format(time.RFC3339)
	}

	// Consume [-+]?
	var neg bool

	c := duration[0]
	if c == '-' || c == '+' {
		neg = c == '-'
	}

	// Example:
	// * If duration d is -1h, then now()-1h
	// * If duration d is  1h, then now()+1h
	if neg {
		return now.Add(-d).Format(time.RFC3339)
	}
	return now.Add(d).Format(time.RFC3339)
}
