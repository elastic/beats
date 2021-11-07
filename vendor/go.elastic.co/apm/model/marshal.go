// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package model // import "go.elastic.co/apm/model"

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/internal/apmstrings"
	"go.elastic.co/fastjson"
)

//go:generate sh generate.sh

// MarshalFastJSON writes the JSON representation of t to w.
func (t Time) MarshalFastJSON(w *fastjson.Writer) error {
	w.Int64(time.Time(t).UnixNano() / int64(time.Microsecond))
	return nil
}

// UnmarshalJSON unmarshals the JSON data into t.
func (t *Time) UnmarshalJSON(data []byte) error {
	var usec int64
	if err := json.Unmarshal(data, &usec); err != nil {
		return err
	}
	*t = Time(time.Unix(usec/1000000, (usec%1000000)*1000).UTC())
	return nil
}

// UnmarshalJSON unmarshals the JSON data into v.
func (v *HTTPSpanContext) UnmarshalJSON(data []byte) error {
	var httpSpanContext struct {
		URL        string
		StatusCode int `json:"status_code"`
	}
	if err := json.Unmarshal(data, &httpSpanContext); err != nil {
		return err
	}
	u, err := url.Parse(httpSpanContext.URL)
	if err != nil {
		return err
	}
	v.URL = u
	v.StatusCode = httpSpanContext.StatusCode
	return nil
}

// MarshalFastJSON writes the JSON representation of v to w.
func (v *HTTPSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.URL != nil {
		beforeURL := w.Size()
		w.RawString(`"url":"`)
		if v.marshalURL(w) {
			w.RawByte('"')
			first = false
		} else {
			w.Rewind(beforeURL)
		}
	}
	if v.StatusCode > 0 {
		if !first {
			w.RawByte(',')
		}
		w.RawString(`"status_code":`)
		w.Int64(int64(v.StatusCode))
	}
	w.RawByte('}')
	return nil
}

func (v *HTTPSpanContext) marshalURL(w *fastjson.Writer) bool {
	if v.URL.Scheme != "" {
		if !marshalScheme(w, v.URL.Scheme) {
			return false
		}
		w.RawString("://")
	} else {
		w.RawString("http://")
	}
	w.StringContents(v.URL.Host)
	if v.URL.Path == "" {
		w.RawByte('/')
	} else {
		if v.URL.Path[0] != '/' {
			w.RawByte('/')
		}
		w.StringContents(v.URL.Path)
	}
	if v.URL.RawQuery != "" {
		w.RawByte('?')
		w.StringContents(v.URL.RawQuery)
	}
	if v.URL.Fragment != "" {
		w.RawByte('#')
		w.StringContents(v.URL.Fragment)
	}
	return true
}

// MarshalFastJSON writes the JSON representation of v to w.
func (v *URL) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Hash != "" {
		const prefix = ",\"hash\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Hash)
	}
	if v.Hostname != "" {
		const prefix = ",\"hostname\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Hostname)
	}
	if v.Path != "" {
		const prefix = `,"pathname":"`
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if v.Path[0] != '/' {
			w.RawByte('/')
		}
		w.StringContents(v.Path)
		w.RawByte('"')
	}
	if v.Port != "" {
		const prefix = ",\"port\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Port)
	}
	schemeBegin := -1
	schemeEnd := -1
	if v.Protocol != "" {
		before := w.Size()
		const prefix = ",\"protocol\":\""
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		schemeBegin = w.Size()
		if marshalScheme(w, v.Protocol) {
			schemeEnd = w.Size()
			w.RawByte('"')
		} else {
			w.Rewind(before)
		}
	}
	if v.Search != "" {
		const prefix = ",\"search\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Search)
	}
	if schemeEnd != -1 && v.Hostname != "" {
		before := w.Size()
		w.RawString(",\"full\":")
		if !v.marshalFullURL(w, w.Bytes()[schemeBegin:schemeEnd]) {
			w.Rewind(before)
		}
	}
	w.RawByte('}')
	return nil
}

func marshalScheme(w *fastjson.Writer, scheme string) bool {
	// Canonicalize the scheme to lowercase. Don't use
	// strings.ToLower, as it's too general and requires
	// additional memory allocations.
	//
	// The scheme should start with a letter, and may
	// then be followed by letters, digits, '+', '-',
	// and '.'. We don't validate the scheme here, we
	// just use those restrictions as a basis for
	// optimization; anything not in that set will
	// mean the full URL is omitted.
	for i := 0; i < len(scheme); i++ {
		c := scheme[i]
		switch {
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.':
			w.RawByte(c)
		case c >= 'A' && c <= 'Z':
			w.RawByte(c + 'a' - 'A')
		default:
			return false
		}
	}
	return true
}

func (v *URL) marshalFullURL(w *fastjson.Writer, scheme []byte) bool {
	w.RawByte('"')
	before := w.Size()
	w.RawBytes(scheme)
	w.RawString("://")

	const maxRunes = 1024
	runes := w.Size() - before // scheme is known to be all single-byte runes
	if runes >= maxRunes {
		// Pathological case, scheme >= 1024 runes.
		w.Rewind(before + maxRunes)
		w.RawByte('"')
		return true
	}

	// Track how many runes we encode, and stop once we've hit the limit.
	rawByte := func(v byte) {
		if runes == maxRunes {
			return
		}
		w.RawByte(v)
		runes++
	}
	stringContents := func(v string) {
		remaining := maxRunes - runes
		truncated, n := apmstrings.Truncate(v, remaining)
		if n > 0 {
			w.StringContents(truncated)
			runes += n
		}
	}

	if strings.IndexByte(v.Hostname, ':') == -1 {
		stringContents(v.Hostname)
	} else {
		rawByte('[')
		stringContents(v.Hostname)
		rawByte(']')
	}
	if v.Port != "" {
		rawByte(':')
		stringContents(v.Port)
	}
	if v.Path != "" {
		if !strings.HasPrefix(v.Path, "/") {
			rawByte('/')
		}
		stringContents(v.Path)
	}
	if v.Search != "" {
		rawByte('?')
		stringContents(v.Search)
	}
	if v.Hash != "" {
		rawByte('#')
		stringContents(v.Hash)
	}
	w.RawByte('"')
	return true
}

func (l *Log) isZero() bool {
	return l.Message == ""
}

func (e *Exception) isZero() bool {
	return e.Message == ""
}

func (c Cookies) isZero() bool {
	return len(c) == 0
}

// MarshalFastJSON writes the JSON representation of c to w.
func (c Cookies) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
outer:
	for i := len(c) - 1; i >= 0; i-- {
		for j := i + 1; j < len(c); j++ {
			if c[i].Name == c[j].Name {
				continue outer
			}
		}
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		w.String(c[i].Name)
		w.RawByte(':')
		w.String(c[i].Value)
	}
	w.RawByte('}')
	return nil
}

// UnmarshalJSON unmarshals the JSON data into c.
func (c *Cookies) UnmarshalJSON(data []byte) error {
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*c = make([]*http.Cookie, 0, len(m))
	for k, v := range m {
		*c = append(*c, &http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	sort.Slice(*c, func(i, j int) bool {
		return (*c)[i].Name < (*c)[j].Name
	})
	return nil
}

func (hs Headers) isZero() bool {
	return len(hs) == 0
}

// MarshalFastJSON writes the JSON representation of h to w.
func (hs Headers) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	for i, h := range hs {
		if i != 0 {
			w.RawByte(',')
		}
		w.String(h.Key)
		w.RawByte(':')
		if len(h.Values) == 1 {
			// Just one item, add the item directly.
			w.String(h.Values[0])
		} else {
			// Zero or multiple items, include them all.
			w.RawByte('[')
			for i, v := range h.Values {
				if i != 0 {
					w.RawByte(',')
				}
				w.String(v)
			}
			w.RawByte(']')
		}
	}
	w.RawByte('}')
	return nil
}

// MarshalFastJSON writes the JSON representation of h to w.
func (*Header) MarshalFastJSON(w *fastjson.Writer) error {
	panic("unreachable")
}

// UnmarshalJSON unmarshals the JSON data into c.
func (hs *Headers) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		switch v := v.(type) {
		case string:
			*hs = append(*hs, Header{Key: k, Values: []string{v}})
		case []interface{}:
			var values []string
			for _, v := range v {
				switch v := v.(type) {
				case string:
					values = append(values, v)
				default:
					return errors.Errorf("expected string, got %T", v)
				}
			}
			*hs = append(*hs, Header{Key: k, Values: values})
		default:
			return errors.Errorf("expected string or []string, got %T", v)
		}
	}
	sort.Slice(*hs, func(i, j int) bool {
		return (*hs)[i].Key < (*hs)[j].Key
	})
	return nil
}

// MarshalFastJSON writes the JSON representation of c to w.
func (c *ExceptionCode) MarshalFastJSON(w *fastjson.Writer) error {
	if c.String != "" {
		w.String(c.String)
	} else {
		w.Float64(c.Number)
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON data into c.
func (c *ExceptionCode) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		c.String = v
	case float64:
		c.Number = v
	default:
		return errors.Errorf("expected string or number, got %T", v)
	}
	return nil
}

// isZero is used by fastjson to implement omitempty.
func (c *ExceptionCode) isZero() bool {
	return c.String == "" && c.Number == 0
}

// MarshalFastJSON writes the JSON representation of b to w.
func (b *RequestBody) MarshalFastJSON(w *fastjson.Writer) error {
	if b.Form != nil {
		w.RawByte('{')
		first := true
		for k, v := range b.Form {
			if first {
				first = false
			} else {
				w.RawByte(',')
			}
			w.String(k)
			w.RawByte(':')
			if len(v) == 1 {
				// Just one item, add the item directly.
				w.String(v[0])
			} else {
				// Zero or multiple items, include them all.
				w.RawByte('[')
				first := true
				for _, v := range v {
					if first {
						first = false
					} else {
						w.RawByte(',')
					}
					w.String(v)
				}
				w.RawByte(']')
			}
		}
		w.RawByte('}')
	} else {
		w.String(b.Raw)
	}
	return nil
}

// UnmarshalJSON unmarshals the JSON data into b.
func (b *RequestBody) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		b.Raw = v
		return nil
	case map[string]interface{}:
		form := make(url.Values, len(v))
		for k, v := range v {
			switch v := v.(type) {
			case string:
				form.Set(k, v)
			case []interface{}:
				for _, v := range v {
					switch v := v.(type) {
					case string:
						form.Add(k, v)
					default:
						return errors.Errorf("expected string, got %T", v)
					}
				}
			default:
				return errors.Errorf("expected string or []string, got %T", v)
			}
		}
		b.Form = form
	default:
		return errors.Errorf("expected string or map, got %T", v)
	}
	return nil
}

func (m StringMap) isZero() bool {
	return len(m) == 0
}

// MarshalFastJSON writes the JSON representation of m to w.
func (m StringMap) MarshalFastJSON(w *fastjson.Writer) (firstErr error) {
	w.RawByte('{')
	first := true
	for _, item := range m {
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		w.String(item.Key)
		w.RawByte(':')
		if err := fastjson.Marshal(w, item.Value); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return nil
}

// UnmarshalJSON unmarshals the JSON data into m.
func (m *StringMap) UnmarshalJSON(data []byte) error {
	var mm map[string]string
	if err := json.Unmarshal(data, &mm); err != nil {
		return err
	}
	*m = make(StringMap, 0, len(mm))
	for k, v := range mm {
		*m = append(*m, StringMapItem{Key: k, Value: v})
	}
	sort.Slice(*m, func(i, j int) bool {
		return (*m)[i].Key < (*m)[j].Key
	})
	return nil
}

// MarshalFastJSON exists to prevent code generation for StringMapItem.
func (*StringMapItem) MarshalFastJSON(*fastjson.Writer) error {
	panic("unreachable")
}

func (m IfaceMap) isZero() bool {
	return len(m) == 0
}

// MarshalFastJSON writes the JSON representation of m to w.
func (m IfaceMap) MarshalFastJSON(w *fastjson.Writer) (firstErr error) {
	w.RawByte('{')
	first := true
	for _, item := range m {
		if first {
			first = false
		} else {
			w.RawByte(',')
		}
		w.String(item.Key)
		w.RawByte(':')
		if err := fastjson.Marshal(w, item.Value); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return nil
}

// UnmarshalJSON unmarshals the JSON data into m.
func (m *IfaceMap) UnmarshalJSON(data []byte) error {
	var mm map[string]interface{}
	if err := json.Unmarshal(data, &mm); err != nil {
		return err
	}
	*m = make(IfaceMap, 0, len(mm))
	for k, v := range mm {
		*m = append(*m, IfaceMapItem{Key: k, Value: v})
	}
	sort.Slice(*m, func(i, j int) bool {
		return (*m)[i].Key < (*m)[j].Key
	})
	return nil
}

// MarshalFastJSON exists to prevent code generation for IfaceMapItem.
func (*IfaceMapItem) MarshalFastJSON(*fastjson.Writer) error {
	panic("unreachable")
}

func (id *TraceID) isZero() bool {
	return *id == TraceID{}
}

// MarshalFastJSON writes the JSON representation of id to w.
func (id *TraceID) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('"')
	writeHex(w, id[:])
	w.RawByte('"')
	return nil
}

// UnmarshalJSON unmarshals the JSON data into id.
func (id *TraceID) UnmarshalJSON(data []byte) error {
	_, err := hex.Decode(id[:], data[1:len(data)-1])
	return err
}

func (id *SpanID) isZero() bool {
	return *id == SpanID{}
}

// UnmarshalJSON unmarshals the JSON data into id.
func (id *SpanID) UnmarshalJSON(data []byte) error {
	_, err := hex.Decode(id[:], data[1:len(data)-1])
	return err
}

// MarshalFastJSON writes the JSON representation of id to w.
func (id *SpanID) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('"')
	writeHex(w, id[:])
	w.RawByte('"')
	return nil
}

func (t *ErrorTransaction) isZero() bool {
	return *t == ErrorTransaction{}
}

func (t *MetricsTransaction) isZero() bool {
	return *t == MetricsTransaction{}
}

func (s *MetricsSpan) isZero() bool {
	return *s == MetricsSpan{}
}

func writeHex(w *fastjson.Writer, v []byte) {
	const hextable = "0123456789abcdef"
	for _, v := range v {
		w.RawByte(hextable[v>>4])
		w.RawByte(hextable[v&0x0f])
	}
}
