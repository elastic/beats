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
// +build windows

package wineventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestPublisherMetadataStore(t *testing.T) {
	logp.TestingSetup()

	s, err := NewPublisherMetadataStore(
		NilHandle,
		"Microsoft-Windows-Security-Auditing",
		logp.NewLogger("metadata"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	assert.NotEmpty(t, s.Events)
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
}
