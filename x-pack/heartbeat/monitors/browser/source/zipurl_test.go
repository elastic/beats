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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source/fixtures"
)

func TestSimpleCases(t *testing.T) {
	address, teardown := setupTests()
	defer teardown()

	type testCase struct {
		name         string
		cfg          common.MapStr
		wantFetchErr bool
	}
	testCases := []testCase{
		{
			"basics",
			common.MapStr{
				"url":     fmt.Sprintf("http://%s/fixtures/todos.zip", address),
				"folder":  "/",
				"retries": 3,
			},
			false,
		},
		{
			"targetdir",
			common.MapStr{
				"url":              fmt.Sprintf("http://%s/fixtures/todos.zip", address),
				"folder":           "/",
				"retries":          3,
				"target_directory": "/tmp/synthetics/blah",
			},
			false,
		},
		{
			"auth success",
			common.MapStr{
				"url":      fmt.Sprintf("http://%s/fixtures/todos.zip", address),
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "testpass",
			},
			false,
		},
		{
			"auth failure",
			common.MapStr{
				"url":      fmt.Sprintf("http://%s/fixtures/todos.zip", address),
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "badpass",
			},
			true,
		},
		{
			"bad proxy",
			common.MapStr{
				"url":     fmt.Sprintf("http://%s/fixtures/todos.zip", address),
				"folder":  "/",
				"retries": 3,
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			zus, err := dummyZus(tc.cfg)
			require.NoError(t, err)

			require.NotNil(t, zus.httpClient)

			if tc.wantFetchErr == true {
				err := zus.Fetch()
				require.Error(t, err)
				return
			}

			fetchAndCheckDir(t, zus)
		})
	}
}

func TestZipUrlWithSameEtag(t *testing.T) {
	address, teardown := setupTests()
	defer teardown()

	zus, err := dummyZus(common.MapStr{
		"url":     fmt.Sprintf("http://%s/fixtures/todos.zip", address),
		"folder":  "/",
		"retries": 3,
	})
	require.NoError(t, err)
	err = zus.Fetch()
	defer zus.Close()
	require.NoError(t, err)

	etag := zus.etag
	target := zus.TargetDirectory
	err = zus.Fetch()
	require.NoError(t, err)
	require.Equalf(t, zus.etag, etag, "etag should be same")
	require.Equal(t, zus.TargetDirectory, target, "Target directory should be same")
}

func TestZipUrlWithBadUrl(t *testing.T) {
	_, teardown := setupTests()
	defer teardown()

	zus, err := dummyZus(common.MapStr{
		"url":     "http://notahost.notadomaintoehutoeuhn",
		"folder":  "/",
		"retries": 2,
	})
	require.NoError(t, err)
	err = zus.Fetch()
	defer zus.Close()
	require.Error(t, err)
}

func setupTests() (addr string, teardown func()) {
	// go offline, so we dont invoke npm install for unit tests
	GoOffline()

	srv := createServer()
	address := srv.Addr
	return address, func() {
		GoOnline()
		srv.Shutdown(context.Background())
	}
}

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
	err := zip.Fetch()
	defer zip.Close()
	require.NoError(t, err)

	fixtures.TestTodosFiles(t, zip.Workdir())
	// check if the working directory is deleted
	require.NoError(t, zip.Close())
	_, err = os.Stat(zip.TargetDirectory)
	require.True(t, os.IsNotExist(err), "TargetDirectory %s should have been deleted", zip.TargetDirectory)
}

func dummyZus(conf map[string]interface{}) (*ZipURLSource, error) {
	zusw := &ZipURLSourceWrapper{}
	err := zusw.Unpack(conf)
	return zusw.zus, err
}
