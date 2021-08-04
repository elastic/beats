// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// we define custom delimiters to prevent issues when using template values as part of other Go templates.
const (
	leftDelim  = "[["
	rightDelim = "]]"
)

var (
	errEmptyTemplateResult = errors.New("the template result is empty")
	errExecutingTemplate   = errors.New("the template execution failed")
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
			"mul":                 mul,
			"div":                 div,
			"hmac":                hmacString,
		}).
		Delims(leftDelim, rightDelim).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

func (t *valueTpl) Execute(trCtx *transformContext, tr transformable, defaultVal *valueTpl, log *logp.Logger) (val string, err error) {
	fallback := func(err error) (string, error) {
		if defaultVal != nil {
			log.Debugf("template execution: falling back to default value")
			return defaultVal.Execute(emptyTransformContext(), transformable{}, nil, log)
		}
		return "", err
	}

	defer func() {
		if r := recover(); r != nil {
			val, err = fallback(errExecutingTemplate)
		}
		if err != nil {
			log.Debugf("template execution failed: %v", err)
		}
		log.Debugf("template execution: evaluated template %q", val)
	}()

	buf := new(bytes.Buffer)
	data := tr.Clone()
	data.Put("cursor", trCtx.cursorMap())
	data.Put("first_event", trCtx.firstEventClone())
	data.Put("last_event", trCtx.lastEventClone())
	data.Put("last_response", trCtx.lastResponseClone().templateValues())

	if err := t.Template.Execute(buf, data); err != nil {
		return fallback(err)
	}

	val = buf.String()
	if val == "" || strings.Contains(val, "<no value>") {
		return fallback(errEmptyTemplateResult)
	}
	return val, nil
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

func toInt(v interface{}) int64 {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int64(vv.Int())
	case reflect.Float32, reflect.Float64:
		return int64(vv.Float())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(vv.Uint())
	case reflect.String:
		f, _ := strconv.ParseFloat(vv.String(), 64)
		return int64(f)
	default:
		return 0
	}
}

func add(vs ...int64) int64 {
	var sum int64
	for _, v := range vs {
		sum += v
	}
	return sum
}

func mul(a, b int64) int64 {
	return a * b
}

func div(a, b int64) int64 {
	return a / b
}

func hmacString(hmacType string, hmacKey string, values ...string) string {
	data := strings.Join(values[:], "")
	if data == "" {
		return ""
	}
	// Create a new HMAC by defining the hash type and the key (as byte array)
	var mac hash.Hash
	switch hmacType {
	case "sha256":
		mac = hmac.New(sha256.New, []byte(hmacKey))
	case "sha1":
		mac = hmac.New(sha1.New, []byte(hmacKey))
	default:
		// Upstream config validation prevents this from happening.
		return ""
	}
	// Write Data to it
	mac.Write([]byte(data))

	// Get result and encode as hexadecimal string
	return hex.EncodeToString(mac.Sum(nil))
}
