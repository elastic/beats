// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

func TestDownloadBodyError(t *testing.T) {
	// This tests the scenario where the download encounters a network error
	// part way through the download, while copying the response body.

	type connKey struct{}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		conn := r.Context().Value(connKey{}).(net.Conn)
		conn.Close()
	}))
	defer srv.Close()
	client := srv.Client()
	srv.Config.ConnContext = func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, connKey{}, c)
	}

	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	config := &artifact.Config{
		SourceURI:       srv.URL,
		TargetDirectory: targetDir,
		OperatingSystem: "linux",
		Architecture:    "64",
	}

	testClient := NewDownloaderWithClient(config, *client)
	artifactPath, err := testClient.Download(context.Background(), beatSpec, version)
	if err == nil {
		os.Remove(artifactPath)
		t.Fatal("expected Download to return an error")
	}
}
