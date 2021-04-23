// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source/fixtures"
)

func createServer() (addr *http.Server) {
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

func fetchAndCheckDir(t *testing.T, zip *ZipURLSource) {
	// go offline, so we dont invoke npm install for unit tests
	GoOffline()
	defer GoOnline()

	err := zip.Fetch()
	defer zip.Close()
	require.NoError(t, err)

	fixtures.TestTodosFiles(t, zip.Workdir())
	// check if the working directory is deleted
	require.NoError(t, zip.Close())
	_, err = os.Stat(zip.TargetDirectory)
	require.True(t, os.IsNotExist(err), "TargetDirectory %s should have been deleted", zip.TargetDirectory)
}

func zipUrlFetchNoAuth(t *testing.T, address string) {
	zus := &ZipURLSource{
		URL:     fmt.Sprintf("http://%s/fixtures/todos.zip", address),
		Folder:  "/",
		Retries: 3,
	}
	fetchAndCheckDir(t, zus)
}

func zipUrlFetchWithAuth(t *testing.T, address string) {
	zus := &ZipURLSource{
		URL:      fmt.Sprintf("http://%s/fixtures/todos.zip", address),
		Folder:   "/",
		Retries:  3,
		Username: "testuser",
		Password: "testpass",
	}
	fetchAndCheckDir(t, zus)
}

func zipUrlTargetDirectory(t *testing.T, address string) {
	zus := &ZipURLSource{
		URL:             fmt.Sprintf("http://%s/fixtures/todos.zip", address),
		Folder:          "/",
		TargetDirectory: "/tmp/synthetics/blah",
	}
	fetchAndCheckDir(t, zus)
}

func zipUrlWithSameEtag(t *testing.T, address string) {
	zus := ZipURLSource{
		URL:     fmt.Sprintf("http://%s/fixtures/todos.zip", address),
		Folder:  "/",
		Retries: 3,
	}
	err := zus.Fetch()
	defer zus.Close()
	require.NoError(t, err)

	etag := zus.etag
	target := zus.TargetDirectory
	err = zus.Fetch()
	require.NoError(t, err)
	require.Equalf(t, zus.etag, etag, "etag should be same")
	require.Equal(t, zus.TargetDirectory, target, "Target directory should be same")
}

func TestZipUrlLogic(t *testing.T) {
	GoOffline()
	defer GoOnline()

	srv := createServer()
	address := srv.Addr
	defer srv.Shutdown(context.Background())

	zipUrlFetchNoAuth(t, address)
	zipUrlFetchWithAuth(t, address)
	zipUrlTargetDirectory(t, address)
	zipUrlWithSameEtag(t, address)
}
