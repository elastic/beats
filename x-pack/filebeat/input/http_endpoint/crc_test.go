// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func Test_validateZoomCRC(t *testing.T) {
	testCases := []struct {
		name         string        // Sub-test name.
		crc          *crcValidator // Load CRC parameters.
		inputJSON    mapstr.M      // Input JSON event.
		wantStatus   int           // Expected response code.
		wantResponse string        // Expected response message.
		wantError    error         // Expected error
	}{
		{
			name: "valid request",
			crc:  newCRC("Zoom", "secretValueTest"),
			inputJSON: mapstr.M{
				"payload": map[string]interface{}{
					"plainToken": "qgg8vlvZRS6UYooatFL8Aw",
				},
				"event_ts": int64(1654503849680),
				"event":    "endpoint.url_validation",
			},
			wantStatus:   http.StatusOK,
			wantResponse: `{"encryptedToken":"70c1f2e2e6ca2d39297490d1f9142c7d701415ea8e6151f6562a08fa657a40ff","plainToken":"qgg8vlvZRS6UYooatFL8Aw"}`,
			wantError:    nil,
		},
		{
			name: "not CRC request",
			crc:  newCRC("Zoom", "secretValueTest"),
			inputJSON: mapstr.M{
				"key": "sample_event",
			},
			wantStatus:   0,
			wantResponse: "",
			wantError:    errNotCRC,
		},
		{
			name: "empty challenge value",
			crc:  newCRC("Zoom", "secretValueTest"),
			inputJSON: mapstr.M{
				"payload": map[string]interface{}{
					"plainToken": "",
				},
				"event_ts": int64(1654503849680),
				"event":    "endpoint.url_validation",
			},
			wantStatus:   http.StatusBadRequest,
			wantResponse: "",
			wantError:    fmt.Errorf("failed decoding \"payload.plainToken\" from CRC request"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute CRC validation
			responseCode, responseBody, err := tc.crc.validator(tc.crc, tc.inputJSON)

			// Validate responses
			assert.Equal(t, tc.wantStatus, responseCode)
			assert.Equal(t, tc.wantResponse, responseBody)
			assert.Equal(t, tc.wantError, err)
		})
	}
}
