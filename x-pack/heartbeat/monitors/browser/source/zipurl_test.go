// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/source/fixtures"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestSimpleCases(t *testing.T) {
	type testCase struct {
		name         string
		cfg          mapstr.M
		tlsServer    bool
		wantFetchErr bool
	}
	testCases := []testCase{
		{
			"basics",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
			},
			false,
			false,
		},
		{
			"targetdir",
			mapstr.M{
				"folder":           "/",
				"retries":          3,
				"target_directory": "/tmp/synthetics/blah",
			},
			false,
			false,
		},
		{
			"auth success",
			mapstr.M{
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "testpass",
			},
			false,
			false,
		},
		{
			"auth failure",
			mapstr.M{
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "badpass",
			},
			false,
			true,
		},
		{
			"ssl ignore cert errors",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
				"ssl": mapstr.M{
					"enabled":           "true",
					"verification_mode": "none",
				},
			},
			true,
			false,
		},
		{
			"bad ssl",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
				"ssl": mapstr.M{
					"enabled":                 "true",
					"certificate_authorities": []string{},
				},
			},
			true,
			true,
		},
	}

	for _, tc := range testCases {
		url, teardown := setupTests(tc.tlsServer)
		defer teardown()
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg["url"] = fmt.Sprintf("%s/fixtures/todos.zip", url)
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
	address, teardown := setupTests(false)
	defer teardown()

	zus, err := dummyZus(mapstr.M{
		"url":     fmt.Sprintf("%s/fixtures/todos.zip", address),
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
	_, teardown := setupTests(false)
	defer teardown()

	zus, err := dummyZus(mapstr.M{
		"url":     "http://notahost.notadomaintoehutoeuhn",
		"folder":  "/",
		"retries": 2,
	})
	require.NoError(t, err)
	err = zus.Fetch()
	defer zus.Close()
	require.Error(t, err)
}

func setupTests(tls bool) (addr string, teardown func()) {
	// go offline, so we dont invoke npm install for unit tests
	GoOffline()

	srv := createServer(tls)
	address := srv.URL
	return address, func() {
		GoOnline()
		srv.Close()
	}
}

func createServer(tls bool) (addr *httptest.Server) {
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

	var srv *httptest.Server
	if tls {
		srv = httptest.NewTLSServer(mux)
	} else {
		srv = httptest.NewServer(mux)
	}

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
	zus := &ZipURLSource{}
	y, _ := yaml.Marshal(conf)
	c, err := common.NewConfigWithYAML(y, string(y))
	if err != nil {
		return nil, err
	}
	err = c.Unpack(zus)
	return zus, err
}
