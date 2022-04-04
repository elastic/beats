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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestPublisherMetadata(t *testing.T) {
	// Modern Application
	testPublisherMetadata(t, "Microsoft-Windows-PowerShell")
	// Modern Application that uses UserData in XML
	testPublisherMetadata(t, "Microsoft-Windows-Eventlog")
	// Classic with messages (no event-data XML templates).
	testPublisherMetadata(t, "Microsoft-Windows-Security-SPP")
	// Classic without message metadata (no event-data XML templates).
	testPublisherMetadata(t, "Windows Error Reporting")
}

func testPublisherMetadata(t *testing.T, provider string) {
	t.Run(provider, func(t *testing.T) {
		md, err := NewPublisherMetadata(NilHandle, provider)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		defer md.Close()

		t.Run("publisher_guid", func(t *testing.T) {
			v, err := md.PublisherGUID()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("PublisherGUID: %v", v)
		})

		t.Run("resource_file_path", func(t *testing.T) {
			v, err := md.ResourceFilePath()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("ResourceFilePath: %v", v)
		})

		t.Run("parameter_file_path", func(t *testing.T) {
			v, err := md.ParameterFilePath()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("ParameterFilePath: %v", v)
		})

		t.Run("message_file_path", func(t *testing.T) {
			v, err := md.MessageFilePath()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("MessageFilePath: %v", v)
		})

		t.Run("help_link", func(t *testing.T) {
			v, err := md.HelpLink()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("HelpLink: %v", v)
		})

		t.Run("publisher_message_id", func(t *testing.T) {
			v, err := md.PublisherMessageID()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("PublisherMessageID: %v", v)
		})

		t.Run("publisher_message", func(t *testing.T) {
			v, err := md.PublisherMessage()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			t.Logf("PublisherMessage: %v", v)
		})

		t.Run("keywords", func(t *testing.T) {
			values, err := md.Keywords()
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if testing.Verbose() {
				for _, value := range values {
					t.Logf("%+v", value)
				}
			}
		})

		t.Run("opcodes", func(t *testing.T) {
			values, err := md.Opcodes()
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if testing.Verbose() {
				for _, value := range values {
					t.Logf("%+v", value)
				}
			}
		})

		t.Run("levels", func(t *testing.T) {
			values, err := md.Levels()
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if testing.Verbose() {
				for _, value := range values {
					t.Logf("%+v", value)
				}
			}
		})

		t.Run("tasks", func(t *testing.T) {
			values, err := md.Tasks()
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if testing.Verbose() {
				for _, value := range values {
					t.Logf("%+v", value)
				}
			}
		})

		t.Run("channels", func(t *testing.T) {
			values, err := md.Channels()
			if err != nil {
				t.Fatalf("%+v", err)
			}

			if testing.Verbose() {
				for _, value := range values {
					t.Logf("%+v", value)
				}
			}
		})

		t.Run("event_metadata", func(t *testing.T) {
			itr, err := md.EventMetadataIterator()
			if err != nil {
				t.Fatalf("%+v", err)
			}
			defer itr.Close()

			for itr.Next() {
				eventID, err := itr.EventID()
				assert.NoError(t, err)
				t.Logf("eventID=%v (id=%v, qualifier=%v)", eventID,
					0xFFFF&eventID,           // Lower 16 bits are the event ID.
					(0xFFFF0000&eventID)>>16) // Upper 16 bits are the qualifier.

				version, err := itr.Version()
				assert.NoError(t, err)
				t.Logf("version=%v", version)

				channel, err := itr.Channel()
				assert.NoError(t, err)
				t.Logf("channel=%v", channel)

				level, err := itr.Level()
				assert.NoError(t, err)
				t.Logf("level=%v", level)

				opcode, err := itr.Opcode()
				assert.NoError(t, err)
				t.Logf("opcode=%v", opcode)

				task, err := itr.Task()
				assert.NoError(t, err)
				t.Logf("task=%v", task)

				keyword, err := itr.Keyword()
				assert.NoError(t, err)
				t.Logf("keyword=%v", keyword)

				messageID, err := itr.MessageID()
				assert.NoError(t, err)
				t.Logf("messageID=%v", messageID)

				template, err := itr.Template()
				assert.NoError(t, err)
				t.Logf("template=%v", template)

				message, err := itr.Message()
				assert.NoError(t, err)
				t.Logf("message=%v", message)
			}
			if err = itr.Err(); err != nil {
				t.Fatalf("%+v", err)
			}
		})
	})
}

func TestNewPublisherMetadataUnknown(t *testing.T) {
	_, err := NewPublisherMetadata(NilHandle, "Fake-Publisher")
	assert.Equal(t, windows.ERROR_FILE_NOT_FOUND, errors.Cause(err))
}
