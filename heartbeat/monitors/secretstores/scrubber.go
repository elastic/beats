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
	"cmp"
	"context"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	// scrubbed replaces secret values, as it does for URL passwords.
	scrubbed = "<hidden>"

	// minScrubLen is the length under which values are not scrubbed. They
	// are easy to guess, and would garble unrelated text.
	minScrubLen = 4
)

type trackerKey struct{}

// tracker collects the secret values stores put in one monitor config.
type tracker struct {
	mu     sync.Mutex
	values map[string]struct{}
}

func withTracker(ctx context.Context, t *tracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, t)
}

// Track records value, a secret that a Store put in the config returned by
// Replace, so that Heartbeat removes it from the events and errors of the
// monitor. Stores must call it for every value they substitute, cached or
// not. It does nothing when ctx isn't the context of a Replace call.
func Track(ctx context.Context, value string) {
	t, ok := ctx.Value(trackerKey{}).(*tracker)
	if !ok || value == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.values == nil {
		t.values = map[string]struct{}{}
	}
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
		// Settings such as URLs get percent-encoded when published.
		forms = append(forms, v, url.QueryEscape(v), url.PathEscape(v))
	}
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
	return &Scrubber{key: strings.Join(forms, "\x00"), replacer: strings.NewReplacer(oldnew...)}
}

// Scrubber removes the secret values a Resolver substituted from the events
// and errors of a monitor. A nil Scrubber does nothing.
type Scrubber struct {
	key      string
	replacer *strings.Replacer
}

// sameAs reports whether s and o scrub the same values, so one can be used
// instead of the other.
func (s *Scrubber) sameAs(o *Scrubber) bool {
	if s == nil || o == nil {
		return s == o
	}
	return s.key == o.key
}

// String returns v without the secret values.
func (s *Scrubber) String(v string) string {
	if s == nil {
		return v
	}
	return s.replacer.Replace(v)
}

// Error returns err with a message without the secret values. The result
// wraps err and keeps its reason type, so the monitor reports it as before.
func (s *Scrubber) Error(err error) error {
	if s == nil || err == nil {
		return err
	}
	msg := err.Error()
	clean := s.replacer.Replace(msg)
	if clean == msg {
		return err
	}
	typ := "io"
	//nolint:errorlint // look.Reason classifies the outermost error only.
	if r, ok := err.(reason.Reason); ok {
		typ = r.Type()
	}
	return &scrubbedError{msg: clean, err: err, typ: typ}
}

type scrubbedError struct {
	msg string
	err error
	typ string
}

func (e *scrubbedError) Error() string { return e.msg }
func (e *scrubbedError) Unwrap() error { return e.err }
func (e *scrubbedError) Type() string  { return e.typ }

// Event removes the secret values from the fields of event. Maps and slices
// that hold one are copied, never modified: jobs can share them between
// events.
func (s *Scrubber) Event(event *beat.Event) {
	if s == nil {
		return
	}
	if fields, changed := scrubMap(s, event.Fields, ""); changed {
		event.Fields = fields
	}
}

// Job returns job, and the continuation jobs it returns, scrubbing the
// events and errors they produce.
func (s *Scrubber) Job(job jobs.Job) jobs.Job {
	if s == nil {
		return job
	}
	return jobs.Wrap(job, func(j jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := j(event)
			s.Event(event)
			return cont, s.Error(err)
		}
	})
}

// value returns v without the secret values, and whether it changed. key is
// the name of the field holding v.
func (s *Scrubber) value(v any, key string) (any, bool) {
	switch v := v.(type) {
	case string:
		out := s.replacer.Replace(v)
		return out, out != v
	case mapstr.M:
		return scrubMap(s, v, key)
	case map[string]any:
		return scrubMap(s, v, key)
	case []any:
		return scrubSlice(s, v, key)
	case []string:
		return scrubSlice(s, v, key)
	case []mapstr.M:
		return scrubSlice(s, v, key)
	case []map[string]any:
		return scrubSlice(s, v, key)
	}
	return v, false
}

func scrubMap[M ~map[string]any](s *Scrubber, m M, key string) (M, bool) {
	var out M
	for k, v := range m {
		// Screenshots are large, and can't hold a secret.
		if key == "synthetics" && k == "blob" {
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
		out[i] = nv.(E)
	}
	if out == nil {
		return in, false
	}
	return out, true
}
