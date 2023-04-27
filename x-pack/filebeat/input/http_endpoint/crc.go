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

	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Validator func(crcValidator, mapstr.M) (int, string, error)

type crcValidator struct {
	provider       string    // Name of the webhook provider
	key            string    // Key to identify CRC requests (optional)
	value          string    // Value of the field to identify CRC requests (optional)
	challenge      string    // key of the challenge token
	challengeValue string    // Value of the challenge token
	secret         string    // Webhook's secret token
	validator      Validator // Function to calculate the challenge
	output         mapstr.M  // Output JSON template
}

// Create new CRC handler based in the webhook provider
func newCRC(crcProvider string, secret string) crcValidator {
	var newCRC crcValidator
	switch strings.ToLower(crcProvider) {
	case "zoom":
		newCRC = newZoomCRC(secret)
	default:
		// Do nothing
	}
	return newCRC
}

// Initialize CRC struct for Zoom provider
func newZoomCRC(secretValue string) crcValidator {
	return crcValidator{
		provider:       "zoom",
		key:            "event",
		value:          "endpoint.url_validation",
		challenge:      "payload.plainToken",
		challengeValue: "",
		secret:         secretValue,
		validator:      validateZoomCRC,
		output: mapstr.M{
			"plainToken":     "",
			"encryptedToken": "",
		},
	}
}

// Validate a CRC request for Zoom
func validateZoomCRC(crc crcValidator, obj mapstr.M) (int, string, error) {
	// Verify it is a CRC request
	if crc.key != "" && crc.value != "" {
		crcValue, found := jsontransform.SearchJSONKeys(obj, crc.key)
		if !found {
			return 0, "", errNotCRC
		}
		crcValue, ok := crcValue.(string)
		if !ok {
			err := fmt.Errorf("failed decoding '%s' from CRC request", crc.key)
			return 0, "", err
		} else if crcValue != crc.value {
			return 0, "", errNotCRC
		}
	}

	challengeValue, found := jsontransform.SearchJSONKeys(obj, crc.challenge)
	if !found {
		return 0, "", errNotCRC
	}

	var ok bool
	crc.challengeValue, ok = challengeValue.(string)
	if !ok {
		err := fmt.Errorf("failed decoding '%s' from CRC request", crc.challenge)
		return 0, "", err
	}

	// Generate hash based on the plainToken
	hash := hmac.New(sha256.New, []byte(crc.secret))
	var err error
	_, err = hash.Write([]byte(crc.challengeValue))
	if err != nil {
		return 0, "", err
	}
	encryptedToken := hex.EncodeToString(hash.Sum(nil))

	// Generate response
	crc.output["plainToken"] = crc.challengeValue
	crc.output["encryptedToken"] = encryptedToken

	response, err := json.Marshal(crc.output)
	if err != nil {
		return 0, "", err
	}

	return http.StatusOK, string(response), nil
}
