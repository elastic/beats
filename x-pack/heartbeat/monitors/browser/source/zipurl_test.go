// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source/fixtures"
)

func createServer(t *testing.T) (addr *http.Server) {
	_, filename, _, _ := runtime.Caller(0)
	fixturesPath := path.Join(filepath.Dir(filename), "fixtures")
	fileServer := http.FileServer(http.Dir(fixturesPath))

	mux := http.NewServeMux()
	mux.HandleFunc("/fixtures/", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Add("Etag", "123456")
		user, pass, hasAuth := req.BasicAuth()
		if hasAuth && (user != "testuser" || pass != "testpass") {
			resp.WriteHeader(403)
			resp.Write([]byte("forbidden"))
		}
		http.StripPrefix("/fixtures", fileServer).ServeHTTP(resp, req)
	})

	srv := &http.Server{Addr: "localhost:1234", Handler: mux}
	go func() {
		srv.ListenAndServe()
	}()

	return srv
}

func TestZipUrlFetchNoAuth(t *testing.T) {
	srv := createServer(t)
	defer srv.Shutdown(context.Background())
	zus := ZipURLSource{
		URL:     fmt.Sprintf("http://%s/fixtures/todos.zip", srv.Addr),
		Folder:  "/",
		Retries: 3,
	}
	err := zus.Fetch()
	defer zus.Close()
	require.NoError(t, err)
	fixtures.TestTodosFiles(t, zus.Workdir())
}

func TestZipUrlFetchWithAuth(t *testing.T) {
	srv := createServer(t)
	defer srv.Shutdown(context.Background())
	zus := ZipURLSource{
		URL:      fmt.Sprintf("http://%s/fixtures/todos.zip", srv.Addr),
		Folder:   "/",
		Retries:  3,
		Username: "testuser",
		Password: "testpass",
	}
	err := zus.Fetch()
	defer zus.Close()
	require.NoError(t, err)
	fixtures.TestTodosFiles(t, zus.Workdir())
}
