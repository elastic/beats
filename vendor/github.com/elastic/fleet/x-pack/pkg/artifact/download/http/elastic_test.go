// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
)

const (
	beatName      = "filebeat"
	version       = "7.1.1"
	sourcePattern = "/downloads/beats/filebeat/"
	source        = "http://artifacts.elastic.co/downloads/beats/"
)

type testCase struct {
	system string
	arch   string
}

func TestDownload(t *testing.T) {
	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	timeout := 30 * time.Second
	testCases := getTestCases()
	elasticClient := getElasticCoClient()

	config := &artifact.Config{
		BeatsSourceURI:  source,
		TargetDirectory: targetDir,
		Timeout:         timeout,
	}

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s-binary-%s", testCase.system, testCase.arch)
		t.Run(testName, func(t *testing.T) {
			config.OperatingSystem = testCase.system
			config.Architecture = testCase.arch

			testClient := NewDownloaderWithClient(config, elasticClient)
			artifactPath, err := testClient.Download(beatName, version)
			if err != nil {
				t.Fatal(err)
			}

			_, err = os.Stat(artifactPath)
			if err != nil {
				t.Fatal(err)
			}

			os.Remove(artifactPath)
		})
	}
}

func TestVerify(t *testing.T) {
	// skip so beats are not fetched from upstream, test only locally when change is made
	t.Skip()

	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	timeout := 30 * time.Second
	testCases := getRandomTestCases()

	config := &artifact.Config{
		BeatsSourceURI:  source,
		TargetDirectory: targetDir,
		Timeout:         timeout,
	}

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s-binary-%s", testCase.system, testCase.arch)
		t.Run(testName, func(t *testing.T) {
			config.OperatingSystem = testCase.system
			config.Architecture = testCase.arch

			testClient := NewDownloader(config)

			artifact, err := testClient.Download(beatName, version)
			if err != nil {
				t.Fatal(err)
			}

			_, err = os.Stat(artifact)
			if err != nil {
				t.Fatal(err)
			}

			testVerifier, err := NewVerifier(config)
			if err != nil {
				t.Fatal(err)
			}

			isOk, err := testVerifier.Verify(beatName, version)
			if err != nil {
				t.Fatal(err)
			}

			if !isOk {
				t.Fatal("verify failed")
			}

			os.Remove(artifact)
		})
	}
}

func getTestCases() []testCase {
	// always test random package to save time
	return []testCase{
		{"linux", "32"},
		{"linux", "64"},
		{"darwin", "32"},
		{"darwin", "64"},
		{"windows", "32"},
		{"windows", "64"},
	}
}

func getRandomTestCases() []testCase {
	tt := getTestCases()

	rand.Seed(time.Now().UnixNano())
	first := rand.Intn(len(tt))
	second := rand.Intn(len(tt))

	return []testCase{
		tt[first],
		tt[second],
	}
}

func getElasticCoClient() http.Client {
	correctValues := map[string]struct{}{
		fmt.Sprintf("%s-%s-%s", beatName, version, "i386.deb"):            struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "amd64.deb"):           struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "i686.rpm"):            struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "x86_64.rpm"):          struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "linux-x86.tar.gz"):    struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "linux-x86_64.tar.gz"): struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "windows-x86.zip"):     struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "windows-x86_64.zip"):  struct{}{},
		fmt.Sprintf("%s-%s-%s", beatName, version, "darwin-x86.tar.gz"):   struct{}{},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		packageName := r.URL.Path[len(sourcePattern):]
		if _, ok := correctValues[packageName]; !ok {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.Write([]byte(packageName))
	})
	server := httptest.NewServer(handler)

	return http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, server.Listener.Addr().String())
			},
		},
	}
}
