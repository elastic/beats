// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Bad linter!
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"

	"github.com/menderesk/beats/v7/libbeat/common/useragent"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/version"
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
			"hmac":                hmacStringHex,
			"base64Encode":        base64Encode,
			"base64EncodeNoPad":   base64EncodeNoPad,
			"base64Decode":        base64Decode,
			"base64DecodeNoPad":   base64DecodeNoPad,
			"join":                join,
			"sprintf":             fmt.Sprintf,
			"hmacBase64":          hmacStringBase64,
			"uuid":                uuidString,
			"userAgent":           userAgentString,
			"beatInfo":            beatInfo,
			"urlEncode":           urlEncode,
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

const defaultTimeLayout = "RFC3339"

var predefinedLayouts = map[string]string{
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
		ly = defaultTimeLayout
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
		layout = defaultTimeLayout
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
		return vv.Int()
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

func base64Encode(values ...string) string {
	data := strings.Join(values, "")
	if data == "" {
		return ""
	}

	return base64.StdEncoding.EncodeToString([]byte(data))
}

func base64EncodeNoPad(values ...string) string {
	data := strings.Join(values, "")
	if data == "" {
		return ""
	}

	return base64.RawStdEncoding.EncodeToString([]byte(data))
}

func base64Decode(enc string) string {
	dec, _ := base64.StdEncoding.DecodeString(enc)
	return string(dec)
}

func base64DecodeNoPad(enc string) string {
	dec, _ := base64.RawStdEncoding.DecodeString(enc)
	return string(dec)
}

func hmacString(hmacType string, hmacKey []byte, data string) []byte {
	if data == "" {
		return nil
	}
	// Create a new HMAC by defining the hash type and the key (as byte array)
	var mac hash.Hash
	switch hmacType {
	case "sha256":
		mac = hmac.New(sha256.New, hmacKey)
	case "sha1":
		mac = hmac.New(sha1.New, hmacKey)
	default:
		// Upstream config validation prevents this from happening.
		return nil
	}
	// Write Data to it
	mac.Write([]byte(data))

	// Get result and encode as bytes
	return mac.Sum(nil)
}

func hmacStringHex(hmacType string, hmacKey string, values ...string) string {
	data := strings.Join(values[:], "")
	if data == "" {
		return ""
	}
	bytes := hmacString(hmacType, []byte(hmacKey), data)
	// Get result and encode as hexadecimal string
	return hex.EncodeToString(bytes)
}

func hmacStringBase64(hmacType string, hmacKey string, values ...string) string {
	data := strings.Join(values[:], "")
	if data == "" {
		return ""
	}
	bytes := hmacString(hmacType, []byte(hmacKey), data)

	// Get result and encode as hexadecimal string
	return base64.StdEncoding.EncodeToString(bytes)
}

func uuidString() string {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return ""
	}
	return uuid.String()
}

// join concatenates the elements of its first argument to create a single string. The separator
// string sep is placed between elements in the resulting string. If the first argument is not of
// type string or []string, its elements will be stringified.
func join(v interface{}, sep string) string {
	// check for []string or string to avoid using reflect
	switch t := v.(type) {
	case []string:
		return strings.Join(t, sep)
	case string:
		return t
	}

	// if we have a slice of a different type, convert it to []string
	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice, reflect.Array:
		s := reflect.ValueOf(v)
		vs := make([]string, s.Len())
		for i := 0; i < s.Len(); i++ {
			vs[i] = fmt.Sprint(s.Index(i))
		}
		return strings.Join(vs, sep)
	}

	// return the stringified single value
	return fmt.Sprint(v)
}

func userAgentString(values ...string) string {
	return useragent.UserAgent("Filebeat", values...)
}

func beatInfo() map[string]string {
	return map[string]string{
		"goos":      runtime.GOOS,
		"goarch":    runtime.GOARCH,
		"commit":    version.Commit(),
		"buildtime": version.BuildTime().String(),
		"version":   version.GetDefaultVersion(),
	}
}

func urlEncode(value string) string {
	if value == "" {
		return ""
	}
	return url.QueryEscape(value)
}
