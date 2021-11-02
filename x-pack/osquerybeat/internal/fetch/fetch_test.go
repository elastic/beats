// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
)

func TestDownload(t *testing.T) {
	ctx := context.Background()

	localFilePathUUID := func() string {
		return uuid.Must(uuid.NewV4()).String()
	}
	tests := []struct {
		Name          string
		Path          string
		LocalFilePath string
		Status        int
		Payload       string
		Hash          string
		ErrStr        string
	}{
		{
			Name:          "Http OK",
			Path:          "/ok",
			LocalFilePath: localFilePathUUID(),
			Status:        http.StatusOK,
			Payload:       "serenity now",
			Hash:          "d1071dfdfd6a5bdf08d9b110f664731cf327cc3d341038f0739699690b599281",
		},
		{
			Name:          "Http OK, empty local file path",
			Path:          "/ok2",
			LocalFilePath: "",
			Status:        http.StatusOK,
			Payload:       "serenity now",
			Hash:          "d1071dfdfd6a5bdf08d9b110f664731cf327cc3d341038f0739699690b599281",
			ErrStr:        "no such file or directory",
		},
		{
			Name:          "Http not found",
			Path:          "/notfound",
			LocalFilePath: localFilePathUUID(),
			Payload:       "file not found",
			Status:        http.StatusNotFound,
			ErrStr:        "file not found",
		},
	}

	mux := http.NewServeMux()
	for _, tc := range tests {
		mux.HandleFunc(tc.Path, func(payload string, status int) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, payload, status)
			}
		}(tc.Payload, tc.Status))
	}

	svr := httptest.NewServer(mux)
	defer svr.Close()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			hash, err := Download(ctx, svr.URL+tc.Path, tc.LocalFilePath)
			defer os.Remove(tc.LocalFilePath)

			if err != nil {
				if tc.ErrStr == "" {
					t.Fatal("unexpected download error:", err)
				}
				return
			}

			diff := cmp.Diff(tc.Hash, hash)
			if diff != "" {
				t.Fatal(diff)
			}

		})
	}

}
