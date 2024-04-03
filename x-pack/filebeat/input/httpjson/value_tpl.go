// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/useragent"
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
			"add":                 add,
			"base64Decode":        base64Decode,
			"base64DecodeNoPad":   base64DecodeNoPad,
			"base64Encode":        base64Encode,
			"base64EncodeNoPad":   base64EncodeNoPad,
			"beatInfo":            beatInfo,
			"div":                 div,
			"formatDate":          formatDate,
			"getRFC5988Link":      getRFC5988Link,
			"hash":                hashStringHex,
			"hashBase64":          hashStringBase64,
			"hexDecode":           hexDecode,
			"hmac":                hmacStringHex,
			"hmacBase64":          hmacStringBase64,
			"join":                join,
			"toJSON":              toJSON,
			"max":                 max,
			"min":                 min,
			"mul":                 mul,
			"now":                 now,
			"parseDate":           parseDate,
			"parseDateInTZ":       parseDateInTZ,
			"parseDuration":       parseDuration,
			"parseTimestamp":      parseTimestamp,
			"parseTimestampMilli": parseTimestampMilli,
			"parseTimestampNano":  parseTimestampNano,
			"replaceAll":          replaceAll,
			"sprintf":             fmt.Sprintf,
			"toInt":               toInt,
			"urlEncode":           urlEncode,
			"userAgent":           userAgentString,
			"uuid":                uuidString,
		}).
		Delims(leftDelim, rightDelim).
		Parse(in)
	if err != nil {
		return err
	}

	*t = valueTpl{Template: tpl}

	return nil
}

func (t *valueTpl) Execute(trCtx *transformContext, tr transformable, targetName string, defaultVal *valueTpl, log *logp.Logger) (val string, err error) {
	fallback := func(err error) (string, error) {
		if defaultVal != nil {
			log.Debugw("template execution: falling back to default value", "target", targetName)
			return defaultVal.Execute(emptyTransformContext(), transformable{}, targetName, nil, log)
		}
		return "", err
	}

	defer func() {
		if r := recover(); r != nil {
			val, err = fallback(errExecutingTemplate)
		}
		if err != nil {
			log.Debugw("template execution failed", "target", targetName, "error", err)
		}
		tryDebugTemplateValue(targetName, val, log)
	}()

	buf := new(bytes.Buffer)
	data := tr.Clone()
	data.Put("cursor", trCtx.cursorMap())
	data.Put("first_event", trCtx.firstEventClone())
	data.Put("last_event", trCtx.lastEventClone())
	data.Put("last_response", trCtx.lastResponseClone().templateValues())
	if trCtx.firstResponse != nil {
		data.Put("first_response", trCtx.firstResponseClone().templateValues())
	}
	// This is only set when chaining is used
	if trCtx.parentTrCtx != nil {
		data.Put("parent_last_response", trCtx.parentTrCtx.lastResponseClone().templateValues())
	}

	if err := t.Template.Execute(buf, data); err != nil {
		return fallback(err)
	}

	val = buf.String()
	if val == "" || strings.Contains(val, "<no value>") {
		return fallback(errEmptyTemplateResult)
	}
	return val, nil
}

func tryDebugTemplateValue(target, val string, log *logp.Logger) {
	switch target {
	case "Authorization", "Proxy-Authorization":
		// ignore filtered headers
	default:
		log.Debugw("evaluated template", "target", target, "value", val)
	}
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

// parseDateInTZ parses a date string within a specified timezone, returning a time.Time
// 'tz' is the timezone (offset or IANA name) for parsing
func parseDateInTZ(date string, tz string, layout ...string) time.Time {
	var ly string
	if len(layout) == 0 {
		ly = defaultTimeLayout
	} else {
		ly = layout[0]
	}
	if found := predefinedLayouts[ly]; found != "" {
		ly = found
	}

	var loc *time.Location
	// Attempt to parse timezone as offset in various formats
	for _, format := range []string{"-07", "-0700", "-07:00"} {
		t, err := time.Parse(format, tz)
		if err != nil {
			continue
		}
		name, offset := t.Zone()
		loc = time.FixedZone(name, offset)
		break
	}

	// If parsing tz as offset fails, try loading location by name
	if loc == nil {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC // Default to UTC on error
		}
	}

	// Using Parse allows us not to worry about the timezone
	// as the predefined timezone is applied afterwards
	t, err := time.Parse(ly, date)
	if err != nil {
		return time.Time{}
	}

	// Manually create a new time object with the parsed date components and the desired location
	// It allows interpreting the parsed time in the specified timezone
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	nanosec := t.Nanosecond()
	localTime := time.Date(year, month, day, hour, min, sec, nanosec, loc)

	// Convert the time to UTC to standardize the output
	return localTime.UTC()
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

var regexpLinkRel = regexp.MustCompile(`<(.*)>.*;\s*rel\=("[^"]*"|[^"][^;]*[^"])`)

func getMatchLink(rel string, linksSplit []string) string {
	for _, link := range linksSplit {
		if !regexpLinkRel.MatchString(link) {
			continue
		}

		matches := regexpLinkRel.FindStringSubmatch(link)
		if len(matches) != 3 {
			continue
		}

		linkRel := matches[2]
		if len(linkRel) > 1 && linkRel[0] == '"' { // We can only have a leading quote if we also have a separate trailing quote.
			linkRel = linkRel[1 : len(linkRel)-1]
		}
		if linkRel != rel {
			continue
		}

		return matches[1]
	}
	return ""
}

func getRFC5988Link(rel string, links []string) string {
	if len(links) == 1 && strings.Count(links[0], "rel=") > 1 {
		linksSplit := strings.Split(links[0], ",")
		return getMatchLink(rel, linksSplit)
	}
	return getMatchLink(rel, links)
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

func min(arg1, arg2 reflect.Value) (interface{}, error) {
	lessThan, err := lt(arg1, arg2)
	if err != nil {
		return nil, err
	}

	// arg1 is < arg2.
	if lessThan {
		return arg1.Interface(), nil
	}
	return arg2.Interface(), nil
}

func max(arg1, arg2 reflect.Value) (interface{}, error) {
	lessThan, err := lt(arg1, arg2)
	if err != nil {
		return nil, err
	}

	// arg1 is < arg2.
	if lessThan {
		return arg2.Interface(), nil
	}
	return arg1.Interface(), nil
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

	// Get result and encode as base64 string
	return base64.StdEncoding.EncodeToString(bytes)
}

func hashStringHex(typ string, values ...string) string {
	// Get result and encode as hexadecimal string
	return hex.EncodeToString(hashStrings(typ, values))
}

func hashStringBase64(typ string, values ...string) string {
	// Get result and encode as base64 string
	return base64.StdEncoding.EncodeToString(hashStrings(typ, values))
}

func hashStrings(typ string, data []string) []byte {
	var h hash.Hash
	switch typ {
	case "sha256":
		h = sha256.New()
	case "sha1":
		h = sha1.New()
	default:
		// Upstream config validation prevents this from happening.
		return nil
	}
	for _, d := range data {
		h.Write([]byte(d))
	}
	return h.Sum(nil)
}

func hexDecode(enc string) string {
	decodedString, err := hex.DecodeString(enc)
	if err != nil {
		return ""
	}
	return string(decodedString)
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
	return useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String(), values...)
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

// replaceAll returns a copy of the string s with all non-overlapping instances
// of old replaced by new.
//
// Note that the order of the arguments differs from Go's [strings.ReplaceAll] to
// make pipelining more ergonomic. This allows s to be piped in because it is
// the final argument. For example,
//
//	[[ "some value" | replaceAll "some" "my" ]]  // == "my value"
func replaceAll(old, new, s string) string {
	return strings.ReplaceAll(s, old, new)
}

// toJSON converts the given structure into a JSON string.
func toJSON(i interface{}) (string, error) {
	result, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("toJSON failed: %w", err)
	}
	return string(bytes.TrimSpace(result)), nil
}
