// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSendMetadata(t *testing.T) {
	beatUUID, err := uuid.NewV4()
	accessToken := "dummy_access_token"
	metadata := common.MapStr{
		"a": "b",
		"c": 4,
		"d": []interface{}{1, "2", 3},
	}
	if err != nil {
		t.Fatalf("error while generating Beat ID: %v", err)
	}

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		// Check correct path is used
		assert.Equal(t, fmt.Sprintf("/api/beats/agent/%s", beatUUID.String()), r.URL.Path)

		request := struct {
			Metadata common.MapStr `json:"metadata"`
		}{}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatal(err)
		}

		expectedMeta, err := json.Marshal(metadata)
		assert.Nil(t, err)

		actualMetadata, err := json.Marshal(request.Metadata)
		assert.Nil(t, err)

		assert.Equal(t, expectedMeta, actualMetadata)

		fmt.Fprintf(w, `{"success": true}`)
	}))
	defer server.Close()

	authClient := AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}
	err = authClient.UpdateMetadata(metadata)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBadMetadataUpdateRequest(t *testing.T) {
	metadata := common.MapStr{"a": "b"}
	useCases := []struct {
		statusCode    int
		success       bool
		message       string
		expectedError error
		metadata      common.MapStr
	}{
		{401, false, "access-token is not a valid auth type to change beat status", fmt.Errorf("access-token is not a valid auth type to change beat status"), metadata},
		{401, false, "Invalid access token", fmt.Errorf("Invalid access token"), metadata},
		{404, false, "Beat not found", fmt.Errorf("no configuration found, you need to enroll your Beat"), metadata},
		{200, true, "", nil, metadata},
		{200, true, "", nil, nil},
	}

	beatUUID, err := uuid.NewV4()
	accessToken := "dummy_access_token"
	if err != nil {
		t.Fatalf("error while generating Beat ID: %v", err)
	}

	for _, useCase := range useCases {
		t.Run(useCase.message, func(t *testing.T) {
			server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()

				w.WriteHeader(useCase.statusCode)
				response := struct {
					Success bool   `json:"success"`
					Message string `json:"message,omitempty"`
				}{useCase.success, useCase.message}

				responseString, err := json.Marshal(response)
				if err != nil {
					t.Fatal(err)
				}
				rs := string(responseString)
				fmt.Println(rs)
				fmt.Fprintf(w, rs)
			}))
			defer server.Close()

			authClient := AuthClient{Client: client, AccessToken: accessToken, BeatUUID: beatUUID}
			err = authClient.UpdateMetadata(metadata)

			if useCase.expectedError == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, useCase.expectedError.Error(), err.Error())
			}
		})
	}
}
