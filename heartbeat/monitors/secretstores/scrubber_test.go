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
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// scrubberFor returns the Scrubber of the given secret values.
func scrubberFor(values ...string) *Scrubber {
	t := newTracker()
	ctx := withTracker(context.Background(), t)
	for _, v := range values {
		Track(ctx, v)
	}
	return t.scrubber()
}

func TestScrubberString(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		in     string
		want   string
	}{
		{name: "value", values: []string{"s3cr3t-token"}, in: "Bearer s3cr3t-token", want: "Bearer <hidden>"},
		{name: "every occurrence", values: []string{"s3cr3t-token"}, in: "s3cr3t-token s3cr3t-token", want: "<hidden> <hidden>"},
		{name: "several values", values: []string{"first-secret", "second-secret"}, in: "second-secret first-secret", want: "<hidden> <hidden>"},
		{name: "longer value first", values: []string{"s3cr3t", "s3cr3t-token"}, in: "s3cr3t-token", want: "<hidden>"},
		{name: "query encoded", values: []string{"a b&c=d-secret"}, in: "https://host/?t=a+b%26c%3Dd-secret", want: "https://host/?t=<hidden>"},
		{name: "path encoded", values: []string{"a b/c-secret"}, in: "https://host/a%20b%2Fc-secret", want: "https://host/<hidden>"},
		{name: "no match", values: []string{"s3cr3t-token"}, in: "nothing to hide", want: "nothing to hide"},
		{name: "json escaped", values: []string{"line1\n\"quoted\"-secret"}, in: `{"key":"line1\n\"quoted\"-secret"}`, want: `{"key":"<hidden>"}`},
		{name: "json html escaped", values: []string{"a<b>&c-secret"}, in: `{"key":"a\u003cb\u003e\u0026c-secret"}`, want: `{"key":"<hidden>"}`},
		{name: "url path", values: []string{"a b,c/d-secret"}, in: (&url.URL{Scheme: "https", Host: "host", Path: "/a b,c/d-secret"}).String(), want: "https://host/<hidden>"},
		{name: "uri component", values: []string{"a b&c-secret"}, in: "https://host/?t=a%20b%26c-secret", want: "https://host/?t=<hidden>"},
		{name: "multi-line value", values: []string{"-----BEGIN KEY-----\r\nMIIEvQIBADANBg\r\n-----END KEY-----"}, in: "stdout: MIIEvQIBADANBg", want: "stdout: <hidden>"},
		{name: "short line of multi-line value", values: []string{"first-line\nabc"}, in: "abc", want: "abc"},
		{name: "short value", values: []string{"abcde"}, in: "abcde", want: "abcde"},
		{name: "empty value", values: []string{""}, in: "text", want: "text"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The scrubber is nil when no value is long enough to scrub.
			assert.Equal(t, tc.want, scrubberFor(tc.values...).String(tc.in), "unexpected scrubbed text")
		})
	}
}

func TestScrubberEvent(t *testing.T) {
	const secret = "s3cr3t-token"
	s := scrubberFor(secret)

	event := &beat.Event{Fields: mapstr.M{
		"url":     mapstr.M{"full": "https://host/?t=" + secret, "port": 443},
		"nested":  map[string]any{"list": []any{"x", secret}},
		"tags":    []string{"a", secret},
		"objects": []mapstr.M{{"key": secret}},
		"plain":   "unchanged",
		"count":   3,
		"synthetics": mapstr.M{
			"payload": mapstr.M{"message": "failed with " + secret},
			"blob":    "base64-" + secret,
		},
	}}
	s.Event(event)

	assert.Equal(t, mapstr.M{
		"url":     mapstr.M{"full": "https://host/?t=<hidden>", "port": 443},
		"nested":  map[string]any{"list": []any{"x", "<hidden>"}},
		"tags":    []string{"a", "<hidden>"},
		"objects": []mapstr.M{{"key": "<hidden>"}},
		"plain":   "unchanged",
		"count":   3,
		"synthetics": mapstr.M{
			"payload": mapstr.M{"message": "failed with <hidden>"},
			"blob":    "base64-" + secret,
		},
	}, event.Fields, "the event must not have the secret, except in the screenshot blob")
}

// TestScrubberEventDoesNotModifySharedValues checks jobs can keep sharing
// the maps they put in events, as the pipeline can still read them.
func TestScrubberEventDoesNotModifySharedValues(t *testing.T) {
	const secret = "s3cr3t-token"
	shared := mapstr.M{"full": "https://host/?t=" + secret}
	tags := []string{secret}
	event := &beat.Event{Fields: mapstr.M{"url": shared, "tags": tags}}

	scrubberFor(secret).Event(event)

	assert.Equal(t, "https://host/?t="+secret, shared["full"], "the shared map must not be modified")
	assert.Equal(t, []string{secret}, tags, "the shared slice must not be modified")
	assert.Equal(t, mapstr.M{"full": "https://host/?t=<hidden>"}, event.Fields["url"], "the event must have the scrubbed copy")
}

func TestScrubberEventWithoutSecrets(t *testing.T) {
	fields := mapstr.M{"url": mapstr.M{"full": "https://host/"}}
	event := &beat.Event{Fields: fields}

	scrubberFor("s3cr3t-token").Event(event)
	(*Scrubber)(nil).Event(event)

	assert.Equal(t, mapstr.M{"url": mapstr.M{"full": "https://host/"}}, event.Fields, "the event must be left as is")
	assert.Equal(t, fields, event.Fields, "the fields must not be copied without a secret")
}

func TestScrubberError(t *testing.T) {
	const secret = "s3cr3t-token"
	s := scrubberFor(secret)

	t.Run("nil", func(t *testing.T) {
		assert.NoError(t, s.Error(nil), "no error must stay no error")
	})

	t.Run("without secret", func(t *testing.T) {
		err := errors.New("failed")
		assert.Same(t, err, s.Error(err), "an error without the secret must be returned as is")
	})

	t.Run("nil scrubber", func(t *testing.T) {
		err := errors.New("failed with " + secret)
		assert.Same(t, err, (*Scrubber)(nil).Error(err), "a nil scrubber must return the error as is")
	})

	t.Run("with secret", func(t *testing.T) {
		orig := errors.New("failed with " + secret)
		err := s.Error(orig)
		assert.EqualError(t, err, "failed with <hidden>", "the message must not have the secret")
		assert.ErrorIs(t, err, orig, "the original error must be wrapped")

		r, ok := err.(reason.Reason) //nolint:errorlint // look.Reason does the same.
		require.True(t, ok, "the error must be a reason")
		assert.Equal(t, "io", r.Type(), "errors default to the io type")
	})

	t.Run("ecs stack trace", func(t *testing.T) {
		stack := "at login (token=" + secret + ")"
		orig := ecserr.NewECSErrWithStack("journey", "FAILED", "step failed", &stack)
		err := s.Error(fmt.Errorf("wrapped: %w", orig))

		var ecs *ecserr.ECSErr
		require.ErrorAs(t, err, &ecs, "the error must have an ECS error")
		require.NotNil(t, ecs.StackTrace, "the stack trace must be kept")
		assert.Equal(t, "at login (token=<hidden>)", *ecs.StackTrace, "the stack trace must not have the secret")
		assert.Equal(t, "step failed", ecs.Message, "the message must be kept")
		assert.Equal(t, "at login (token="+secret+")", *orig.StackTrace, "the original error must not be modified")
	})

	t.Run("keeps the reason type", func(t *testing.T) {
		err := s.Error(reason.ValidateFailed(errors.New("invalid " + secret)))
		r, ok := err.(reason.Reason) //nolint:errorlint // look.Reason does the same.
		require.True(t, ok, "the error must be a reason")
		assert.Equal(t, "validate", r.Type(), "the reason type must be kept")
		assert.Equal(t, "invalid <hidden>", r.Error(), "the message must not have the secret")
	})
}

// TestScrubberErrorInBrowserEvent checks the error a browser monitor
// publishes, which the summarizer takes from the ECS error in the chain.
func TestScrubberErrorInBrowserEvent(t *testing.T) {
	const secret = "s3cr3t-token"
	stack := "at login (token=" + secret + ")"
	err := scrubberFor(secret).Error(ecserr.NewECSErrWithStack("journey", "FAILED", "failed with "+secret, &stack))

	event := &beat.Event{Fields: mapstr.M{}}
	summarizer.NewBrowserErrPlugin().EachEvent(event, err)

	got, getErr := event.GetValue("error")
	require.NoError(t, getErr, "the event must have an error")
	ecs, ok := got.(*ecserr.ECSErr)
	require.True(t, ok, "the error must be an ECS error, got %T", got)
	assert.Equal(t, "failed with <hidden>", ecs.Message, "error.message must not have the secret")
	require.NotNil(t, ecs.StackTrace, "error.stack_trace must be kept")
	assert.Equal(t, "at login (token=<hidden>)", *ecs.StackTrace, "error.stack_trace must not have the secret")
}
