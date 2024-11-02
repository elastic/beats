// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Validator is a CRC validation function type. It applies the provided validator
// to the challenge sent by an API and returns an HTTP status code and response
// body to be sent to the challenging API server.
type Validator func(*crcValidator, mapstr.M) (status int, resp string, _ error)

func (v *crcValidator) validate(m mapstr.M) (status int, resp string, _ error) {
	return v.validator(v, m)
}

type crcValidator struct {
	provider  string    // Name of the webhook provider
	key       string    // Key to identify CRC requests (optional)
	value     string    // Value of the field to identify CRC requests (optional)
	challenge string    // Key of the challenge token
	secret    string    // Webhook's secret token
	validator Validator // Function to process the CRC request
	output    mapstr.M  // Output JSON template
}

// Create new CRC handler based in the webhook provider
func newCRC(name, secret string) *crcValidator {
	fn, ok := crcProviders[strings.ToLower(name)]
	if !ok {
		return nil
	}
	return fn(secret)
}

// newZoomCRC returns a CRC handler for the Zoom API
func newZoomCRC(secretValue string) *crcValidator {
	return &crcValidator{
		provider:  "zoom",
		key:       "event",
		value:     "endpoint.url_validation",
		challenge: "payload.plainToken",
		secret:    secretValue,
		validator: validateZoomCRC,
		output: mapstr.M{
			"plainToken":     "",
			"encryptedToken": "",
		},
	}
}

func validateZoomCRC(crc *crcValidator, obj mapstr.M) (status int, resp string, _ error) {
	/* Verify it is a CRC request. It must contain the following data:
	{
	  "payload": {
	    "plainToken": ""
	  },
	  "event": "endpoint.url_validation"
	}
	*/
	event, ok := obj["event"].(string)
	if !ok || event != "endpoint.url_validation" {
		return 0, "", errNotCRC
	}

	payload, ok := obj["payload"].(map[string]interface{})
	if !ok {
		return 0, "", errNotCRC
	}

	challengeValue, ok := payload["plainToken"].(string)
	if !ok {
		return 0, "", errNotCRC
	} else if challengeValue == "" {
		err := fmt.Errorf("failed decoding %q from CRC request", crc.challenge)
		return http.StatusBadRequest, "", err
	}

	// Generate hash based on the plainToken
	hash := hmac.New(sha256.New, []byte(crc.secret))
	var err error
	_, err = hash.Write([]byte(challengeValue))
	if err != nil {
		return http.StatusInternalServerError, "", err
	}
	encryptedToken := hex.EncodeToString(hash.Sum(nil))

	// Generate response
	crc.output["plainToken"] = challengeValue
	crc.output["encryptedToken"] = encryptedToken

	response, err := json.Marshal(crc.output)
	if err != nil {
		return http.StatusInternalServerError, "", err
	}

	return http.StatusOK, string(response), nil
}
