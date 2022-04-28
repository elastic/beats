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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

func TestDownloadBodyError(t *testing.T) {
	// This tests the scenario where the download encounters a network error
	// part way through the download, while copying the response body.

	type connKey struct{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		conn, ok := r.Context().Value(connKey{}).(net.Conn)
		if ok {
			conn.Close()
		}
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

	log := newRecordLogger()
	testClient := NewDownloaderWithClient(log, config, *client)
	artifactPath, err := testClient.Download(context.Background(), beatSpec, version)
	os.Remove(artifactPath)
	if err == nil {
		t.Fatal("expected Download to return an error")
	}

	require.Len(t, log.info, 1, "download error not logged at info level")
	assert.Equal(t, log.info[0].record, "download from %s failed at %s @ %sps: %s")
	require.Len(t, log.warn, 1, "download error not logged at warn level")
	assert.Equal(t, log.warn[0].record, "download from %s failed at %s @ %sps: %s")
}

func TestDownloadLogProgressWithLength(t *testing.T) {
	fileSize := 100 * units.MiB
	chunks := 100
	chunk := make([]byte, fileSize/chunks)
	delayBetweenChunks := 10 * time.Millisecond
	totalTime := time.Duration(chunks) * (delayBetweenChunks + 1*time.Millisecond)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(fileSize))
		w.WriteHeader(http.StatusOK)
		for i := 0; i < chunks; i++ {
			_, err := w.Write(chunk)
			if err != nil {
				panic(err)
			}
			w.(http.Flusher).Flush()
			<-time.After(delayBetweenChunks)
		}
	}))
	defer srv.Close()
	client := srv.Client()

	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	config := &artifact.Config{
		SourceURI:       srv.URL,
		TargetDirectory: targetDir,
		OperatingSystem: "linux",
		Architecture:    "64",
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: totalTime,
		},
	}

	log := newRecordLogger()
	testClient := NewDownloaderWithClient(log, config, *client)
	artifactPath, err := testClient.Download(context.Background(), beatSpec, version)
	os.Remove(artifactPath)
	require.NoError(t, err, "Download should not have errored")

	// 2 files are downloaded so 4 log messages are expected in the info level and only the complete is over the warn
	// window as 2 log messages for warn.
	require.Len(t, log.info, 4)
	assert.Equal(t, log.info[0].record, "download progress from %s is %s/%s (%.2f%% complete) @ %sps")
	assert.Equal(t, log.info[1].record, "download from %s completed in %s @ %sps")
	assert.Equal(t, log.info[2].record, "download progress from %s is %s/%s (%.2f%% complete) @ %sps")
	assert.Equal(t, log.info[3].record, "download from %s completed in %s @ %sps")
	require.Len(t, log.warn, 2)
	assert.Equal(t, log.warn[0].record, "download from %s completed in %s @ %sps")
	assert.Equal(t, log.warn[1].record, "download from %s completed in %s @ %sps")
}

func TestDownloadLogProgressWithoutLength(t *testing.T) {
	fileSize := 100 * units.MiB
	chunks := 100
	chunk := make([]byte, fileSize/chunks)
	delayBetweenChunks := 10 * time.Millisecond
	totalTime := time.Duration(chunks) * (delayBetweenChunks + 1*time.Millisecond)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		for i := 0; i < chunks; i++ {
			_, err := w.Write(chunk)
			if err != nil {
				panic(err)
			}
			w.(http.Flusher).Flush()
			<-time.After(delayBetweenChunks)
		}
	}))
	defer srv.Close()
	client := srv.Client()

	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	config := &artifact.Config{
		SourceURI:       srv.URL,
		TargetDirectory: targetDir,
		OperatingSystem: "linux",
		Architecture:    "64",
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: totalTime,
		},
	}

	log := newRecordLogger()
	testClient := NewDownloaderWithClient(log, config, *client)
	artifactPath, err := testClient.Download(context.Background(), beatSpec, version)
	os.Remove(artifactPath)
	require.NoError(t, err, "Download should not have errored")

	// 2 files are downloaded so 4 log messages are expected in the info level and only the complete is over the warn
	// window as 2 log messages for warn.
	require.Len(t, log.info, 4)
	assert.Equal(t, log.info[0].record, "download progress from %s has fetched %s @ %sps")
	assert.Equal(t, log.info[1].record, "download from %s completed in %s @ %sps")
	assert.Equal(t, log.info[2].record, "download progress from %s has fetched %s @ %sps")
	assert.Equal(t, log.info[3].record, "download from %s completed in %s @ %sps")
	require.Len(t, log.warn, 2)
	assert.Equal(t, log.warn[0].record, "download from %s completed in %s @ %sps")
	assert.Equal(t, log.warn[1].record, "download from %s completed in %s @ %sps")
}

type logMessage struct {
	record string
	args   []interface{}
}

type recordLogger struct {
	lock sync.RWMutex
	info []logMessage
	warn []logMessage
}

func newRecordLogger() *recordLogger {
	return &recordLogger{
		info: make([]logMessage, 0, 10),
		warn: make([]logMessage, 0, 10),
	}
}

func (f *recordLogger) Infof(record string, args ...interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.info = append(f.info, logMessage{record, args})
}

func (f *recordLogger) Warnf(record string, args ...interface{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.warn = append(f.warn, logMessage{record, args})
}
