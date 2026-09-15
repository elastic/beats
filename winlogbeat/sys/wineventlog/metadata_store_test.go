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

//go:build windows

package wineventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestPublisherMetadataStore(t *testing.T) {
	logp.TestingSetup()

	s, err := NewPublisherMetadataStore(
		NilHandle,
		"Microsoft-Windows-Security-Auditing",
		0,
		logp.NewLogger("metadata"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	assert.NotEmpty(t, s.EventsByVersion)
	assert.NotEmpty(t, s.EventsNewest)
	assert.Empty(t, s.EventFingerprints)

	t.Run("event_metadata_from_handle", func(t *testing.T) {
		log := openLog(t, security4752File)
		defer log.Close()

		h := mustNextHandle(t, log)
		defer h.Close()

		em, err := newEventMetadataFromEventHandle(s.Metadata, h)
		if err != nil {
			t.Fatal(err)
		}

		assert.EqualValues(t, 4752, em.EventID)
		assert.EqualValues(t, 0, em.Version)
		assert.Empty(t, em.MsgStatic)
		assert.NotNil(t, em.MsgTemplate)
		assert.NotEmpty(t, em.EventData)
	})

	t.Run("publisher_message_is_preferred_on_event_data_mismatch", func(t *testing.T) {
		log := openLog(t, security4752File)
		defer log.Close()

		h := mustNextHandle(t, log)
		defer h.Close()

		defaultEM := s.EventsNewest[4752]
		require.NotNil(t, defaultEM, "expected event 4752 in the publisher metadata")
		require.NotNil(t, defaultEM.MsgTemplate, "expected a message template for event 4752")

		// Simulate an event whose data does not match the parameters declared
		// in the publisher metadata (e.g. Schannel events whose XML
		// representation does not include the __binLength/binaryData template
		// parameters) by adding an extra parameter to the metadata.
		originalParams := defaultEM.EventData.Params
		defaultEM.EventData.Params = append(originalParams[:len(originalParams):len(originalParams)], EventData{Name: "__binLength"})
		t.Cleanup(func() { defaultEM.EventData.Params = originalParams })

		em := s.getEventMetadata(4752, 0, 12345, h)
		require.NotNil(t, em, "expected event metadata for the mismatched event")
		require.NotSame(t, defaultEM, em, "expected unique event metadata built from the event handle")

		// The message template must come from the publisher metadata, not
		// from formatting the event handle with the template inserts. The
		// latter (EvtFormatMessageEvent) only substitutes the inserts for
		// string parameters and bakes zero values for all other types into
		// the cached template (e.g. "(PID: 0)"). Both representations must
		// be replaced together so a stale event-handle MsgStatic cannot win
		// over the publisher MsgTemplate in formatMessage.
		assert.Same(t, defaultEM.MsgTemplate, em.MsgTemplate, "expected the message template from the publisher metadata to be used")
		assert.Equal(t, defaultEM.MsgStatic, em.MsgStatic, "expected the static message from the publisher metadata to be used")
		assert.Empty(t, em.MsgStatic, "publisher MsgTemplate must clear any event-handle MsgStatic")
	})
}
