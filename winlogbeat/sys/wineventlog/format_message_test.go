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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMessage(t *testing.T) {
	log := openLog(t, security4752File)
	defer log.Close()

	evtHandle := mustNextHandle(t, log)
	defer evtHandle.Close()

	publisherMetadata, err := NewPublisherMetadata(NilHandle, "Microsoft-Windows-Security-Auditing")
	if err != nil {
		t.Fatal(err)
	}
	defer publisherMetadata.Close()

	t.Run("getMessageStringFromHandle", func(t *testing.T) {
		t.Run("no_metadata", func(t *testing.T) {
			t.Skip("This currently fails under Win10. The message strings are returned even though no metadata is passed.")
			// Metadata is required unless the events were forwarded with "RenderedText".
			_, err := getMessageStringFromHandle(nil, evtHandle, nil)
			assert.Error(t, err)
		})

		t.Run("with_metadata", func(t *testing.T) {
			// When no values are passed in then event data from the event is
			// substituted into the message.
			msg, err := getMessageStringFromHandle(publisherMetadata, evtHandle, nil)
			if err != nil {
				t.Fatal(err)
			}
			assert.Contains(t, msg, "CN=Administrator,CN=Users,DC=TEST,DC=SAAS")
		})

		t.Run("custom_values", func(t *testing.T) {
			// Substitute custom values into the message.
			msg, err := getMessageStringFromHandle(publisherMetadata, evtHandle, templateInserts.Slice())
			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t, msg, `{{eventParam $ 2}}`)

			// NOTE: In this test case I noticed the messages contains
			//   "Logon ID:               0x0"
			// but it should contain
			//   "Logon ID:               {{eventParam $ 9}}"
			//
			// This may mean that certain windows.GUID values cannot be
			// substituted with string values. So we shouldn't rely on this
			// method to create text/templates. Instead we can use the
			// getMessageStringFromMessageID (see test below) that works as
			// expected.
			//
			// Note: This is not the case under 32-bit Windows 7.
			//       Disabling the assertion for now.
			//assert.NotContains(t, msg, `{{eventParam $ 9}}`)
		})
	})

	t.Run("getMessageStringFromMessageID", func(t *testing.T) {
		// Get the message ID for event 4752.
		itr, err := publisherMetadata.EventMetadataIterator()
		if err != nil {
			t.Fatal(err)
		}
		defer itr.Close()

		var messageID uint32
		for itr.Next() {
			id, err := itr.EventID()
			if err != nil {
				t.Fatal(err)
			}
			if id == 4752 {
				messageID, err = itr.MessageID()
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		if messageID == 0 {
			t.Fatal("message ID for event 4752 not found")
		}

		t.Run("no_metadata", func(t *testing.T) {
			// Metadata is required to find the message file.
			_, err := getMessageStringFromMessageID(nil, messageID, nil)
			assert.Error(t, err)
		})

		t.Run("with_metadata", func(t *testing.T) {
			// When no values are passed in then the raw message is returned
			// with place-holders like %1 and %2.
			msg, err := getMessageStringFromMessageID(publisherMetadata, messageID, nil)
			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t, msg, "%9")
		})

		t.Run("custom_values", func(t *testing.T) {
			msg, err := getMessageStringFromMessageID(publisherMetadata, messageID, templateInserts.Slice())
			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t, msg, `{{eventParam $ 2}}`)
			assert.Contains(t, msg, `{{eventParam $ 9}}`)
		})
	})

	t.Run("getEventXML", func(t *testing.T) {
		t.Run("no_metadata", func(t *testing.T) {
			t.Skip("This currently fails under Win10. The event XML is returned even though no metadata is passed.")
			// It needs the metadata handle to add the message to the XML.
			_, err := getEventXML(nil, evtHandle)
			assert.Error(t, err)
		})

		t.Run("with_metadata", func(t *testing.T) {
			xml, err := getEventXML(publisherMetadata, evtHandle)
			if err != nil {
				t.Fatal(err)
			}

			assert.True(t, strings.HasPrefix(xml, "<Event"))
			assert.True(t, strings.HasSuffix(xml, "</Event>"))
		})
	})
}
