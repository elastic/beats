// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/hex"
	"encoding/json"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"net/http"
)

func validateCRC(h *httpHandler, plainToken string) (string, int, error) {
	var response string
    var status int
    var err error

	switch strings.ToLower(h.CRCProvider){
	case "zoom":
		response, status, err = generateZoomCRC(h.secretValue, plainToken)
	default:
		h.log.Debugw("Unable to validate CRC request. Unrecognized provider.")
		err = fmt.Errorf("unrecognized CRC provider")
	}

	return response, status, err
}

// Generate CRC response to validate a CRC request
func generateZoomCRC(secretValue string, plainToken string) (string, int, error) {
	hash := hmac.New(sha256.New, []byte(secretValue))
	var err error
	_, err = hash.Write([]byte(plainToken))
	if err != nil {
		return "", 0, err
	}
	encryptedToken := hex.EncodeToString(hash.Sum(nil))

	jsonMap := make(map[string]string)
	jsonMap["plainToken"] = plainToken
	jsonMap["encryptedToken"] = encryptedToken

	response, err := json.Marshal(jsonMap)
	if err != nil {
		return "", 0, err
	}

	return string(response), http.StatusOK, nil
}
