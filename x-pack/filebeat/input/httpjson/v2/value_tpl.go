// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
)

type valueTpl struct {
	*template.Template
}

func (t *valueTpl) Unpack(in string) error {
	tpl, err := template.New("").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"now":            now,
			"formatDate":     formatDate,
			"parseDate":      parseDate,
			"getRFC5988Link": getRFC5988Link,
		}).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

func (t *valueTpl) Execute(trCtx transformContext, tr *transformable, defaultVal string) (val string) {
	defer func() {
		if r := recover(); r != nil {
			err, _ := r.(error)
			fmt.Println(err)
			_ = err
			// TODO: find alternative to this ugliness
			val = defaultVal
		}
	}()

	buf := new(bytes.Buffer)
	data := common.MapStr{}

	_, _ = data.Put("header", tr.header.Clone())
	_, _ = data.Put("body", tr.body.Clone())
	_, _ = data.Put("url.value", tr.url.String())
	_, _ = data.Put("url.params", tr.url.Query())
	_, _ = data.Put("cursor", trCtx.cursor.Clone())
	_, _ = data.Put("last_event", trCtx.lastEvent.Clone())
	_, _ = data.Put("last_response.body", trCtx.lastResponse.body.Clone())
	_, _ = data.Put("last_response.header", trCtx.lastResponse.header.Clone())
	_, _ = data.Put("last_response.url.value", trCtx.lastResponse.url.String())
	_, _ = data.Put("last_response.url.params", trCtx.lastResponse.url.Query())

	if err := t.Template.Execute(buf, data); err != nil {
		return defaultVal
	}

	return buf.String()
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

func formatDate(date time.Time, layout string, tz ...string) string {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	} else {
		layout = time.RFC3339
	}

	if len(tz) > 0 {
		if loc, err := time.LoadLocation(tz[0]); err == nil {
			date = date.In(loc)
		} else {
			date = date.UTC()
		}
	} else {
		date = date.UTC()
	}

	return date.Format(layout)
}

func parseDate(date, layout string) time.Time {
	if found := predefinedLayouts[layout]; found != "" {
		layout = found
	} else {
		layout = time.RFC3339
	}

	t, err := time.Parse(layout, date)
	if err != nil {
		return time.Time{}
	}

	return t
}

func now(add ...time.Duration) time.Time {
	now := time.Now()
	if len(add) == 0 {
		return now
	}
	return now.Add(add[0])
}

func getRFC5988Link(links, rel string) string {
	return ""
}
