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

package secretstores

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	// scrubbed replaces secret values, as it does for URL passwords.
	scrubbed = "<hidden>"

	// minScrubLen is the length under which values are not scrubbed. They
	// are easy to guess, and would garble unrelated text.
	minScrubLen = 6
)

type trackerKey struct{}

// tracker collects the secret values stores put in one monitor config.
type tracker struct {
	mu     sync.Mutex
	values map[string]struct{}
}

func newTracker() *tracker {
	return &tracker{values: map[string]struct{}{}}
}

func withTracker(ctx context.Context, t *tracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, t)
}

// Track records value, a secret that a Store put in the config returned by
// Replace, so that Heartbeat removes it from the events and log lines of the
// monitor. Stores must call it for every value they substitute, cached or
// not. It does nothing when ctx isn't the context of a Replace call.
func Track(ctx context.Context, value string) {
	t, ok := ctx.Value(trackerKey{}).(*tracker)
	if !ok || value == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.values[value] = struct{}{}
}

// scrubber returns a Scrubber for the tracked values, or nil without any.
func (t *tracker) scrubber() *Scrubber {
	t.mu.Lock()
	defer t.mu.Unlock()

	var forms []string
	for v := range t.values {
		if len(v) < minScrubLen {
			continue
		}
		// Settings such as URLs get percent-encoded when published, and
		// values with quotes or newlines get escaped inside JSON.
		query := url.QueryEscape(v)
		forms = append(forms, v, query, strings.ReplaceAll(query, "+", "%20"),
			url.PathEscape(v), (&url.URL{Path: v}).EscapedPath())
		forms = append(forms, jsonEscaped(v)...)
		// Output read line by line, such as the browser runner's, splits
		// multi-line values such as PEM keys.
		if strings.Contains(v, "\n") {
			for line := range strings.SplitSeq(v, "\n") {
				if line = strings.TrimSuffix(line, "\r"); len(line) >= minScrubLen {
					forms = append(forms, line)
				}
			}
		}
	}
	return newScrubber(forms)
}

// mergeScrubbers returns a Scrubber of the values of all of ss, or nil
// without any.
func mergeScrubbers(ss []*Scrubber) *Scrubber {
	var forms []string
	for _, s := range ss {
		if s != nil {
			forms = append(forms, s.forms...)
		}
	}
	return newScrubber(forms)
}

// newScrubber returns a Scrubber replacing forms, which it sorts, or nil
// without any.
func newScrubber(forms []string) *Scrubber {
	if len(forms) == 0 {
		return nil
	}
	// A Replacer prefers the first of the matches at a position: put longer
	// values first, so a secret that contains another is replaced whole.
	slices.SortFunc(forms, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(b), len(a)), cmp.Compare(a, b))
	})
	forms = slices.Compact(forms)

	oldnew := make([]string, 0, 2*len(forms))
	for _, f := range forms {
		oldnew = append(oldnew, f, scrubbed)
	}
	return &Scrubber{forms: forms, replacer: strings.NewReplacer(oldnew...)}
}

// jsonEscaped returns v as it appears inside a JSON string, with and without
// the HTML escaping of Go's encoder.
func jsonEscaped(v string) []string {
	forms := make([]string, 0, 2)
	for _, escapeHTML := range []bool{true, false} {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(escapeHTML)
		// Encoding a string can't fail.
		_ = enc.Encode(v)
		// Drop the quotes and the newline Encode adds.
		encoded := strings.TrimSuffix(b.String(), "\n")
		forms = append(forms, encoded[1:len(encoded)-1])
	}
	return forms
}

// Scrubber removes the secret values a Resolver substituted from the events
// and log lines of a monitor. A nil Scrubber does nothing.
type Scrubber struct {
	forms    []string
	replacer *strings.Replacer
}

// replace returns v without the secret values. The Replacer copies every
// string it's given, so it only runs on the few that hold a value.
func (s *Scrubber) replace(v string) string {
	for _, f := range s.forms {
		if strings.Contains(v, f) {
			return s.replacer.Replace(v)
		}
	}
	return v
}

// String returns v without the secret values.
func (s *Scrubber) String(v string) string {
	if s == nil {
		return v
	}
	return s.replace(v)
}

// equal reports whether s and o scrub the same values.
func (s *Scrubber) equal(o *Scrubber) bool {
	if s == nil || o == nil {
		return s == o
	}
	return slices.Equal(s.forms, o.forms)
}

// Event removes the secret values from the fields of event, except the kept
// ones. Maps and slices that hold one are copied, never modified: jobs can
// share them between events. Other values, such as structs, are left as is:
// the pipeline converts them to maps before the Redactor runs.
func (s *Scrubber) Event(event *beat.Event) {
	if s == nil || event == nil {
		return
	}
	if fields, changed := scrubMap(s, event.Fields, ""); changed {
		event.Fields = fields
	}
}

// value returns v without the secret values, and whether it changed. key is
// the name of the field holding v.
func (s *Scrubber) value(v any, key string) (any, bool) {
	switch tv := v.(type) {
	case string:
		out := s.replace(tv)
		if out == tv {
			// Returning v and not tv avoids boxing the string again.
			return v, false
		}
		return out, true
	case mapstr.M:
		return scrubMap(s, tv, key)
	case map[string]any:
		return scrubMap(s, tv, key)
	case []any:
		return scrubSlice(s, tv, key)
	case []string:
		return scrubSlice(s, tv, key)
	case []mapstr.M:
		return scrubSlice(s, tv, key)
	case []map[string]any:
		return scrubSlice(s, tv, key)
	}
	return v, false
}

func scrubMap[M ~map[string]any](s *Scrubber, m M, key string) (M, bool) {
	var out M
	for k, v := range m {
		if kept(key, k) {
			continue
		}
		nv, changed := s.value(v, k)
		if !changed {
			continue
		}
		if out == nil {
			out = maps.Clone(m)
		}
		out[k] = nv
	}
	if out == nil {
		return m, false
	}
	return out, true
}

// kept reports whether the field k of the field parent is never scrubbed.
// Heartbeat sets these from its own fixed values or from the monitor
// settings, and consumers find, classify and alert on events with them: a
// secret equal to one of their values must not change them. Screenshots are
// large, and can't hold a secret.
func kept(parent, k string) bool {
	switch parent {
	case "":
		return k == "summary" || k == "state"
	case "monitor":
		switch k {
		case "id", "name", "type", "status", "check_group", "origin", "project":
			return true
		}
	case "event":
		return k == "type" || k == "dataset"
	case "synthetics":
		return k == "type" || k == "blob"
	}
	return false
}

func scrubSlice[S ~[]E, E any](s *Scrubber, in S, key string) (S, bool) {
	var out S
	for i, item := range in {
		nv, changed := s.value(item, key)
		if !changed {
			continue
		}
		if out == nil {
			out = slices.Clone(in)
		}
		typed, ok := nv.(E)
		if !ok {
			// Only reachable if value changed the type of an element.
			continue
		}
		out[i] = typed
	}
	if out == nil {
		return in, false
	}
	return out, true
}
