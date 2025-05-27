// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//This file was contributed to by generative AI

//go:build integration

package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func timeBetweenLogEntries(t *testing.T, l1, l2 string) time.Duration {
	type entry struct {
		TS string `json:"@timestamp"`
	}

	e1 := entry{}
	if err := json.Unmarshal([]byte(l1), &e1); err != nil {
		t.Fatalf("cannot parse log entry. Err: %s. Entry: %s", err, l1)
	}

	e2 := entry{}
	if err := json.Unmarshal([]byte(l2), &e2); err != nil {
		t.Fatalf("cannot parse log entry. Err: %s. Entry: %s", err, l1)
	}

	t1, err := time.Parse("2006-01-02T15:04:05Z0700", e1.TS)
	if err != nil {
		t.Fatalf("cannot parse time from first log entry: %s", err)
	}

	t2, err := time.Parse("2006-01-02T15:04:05Z0700", e2.TS)
	if err != nil {
		t.Fatalf("cannot parse time from second log entry: %s", err)
	}

	return t2.Sub(t1)
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		t.Fatalf("cannot stat file: %s", err)
	}

	return true
}

func waitForEOF(t *testing.T, filebeat *integration.BeatProc, files []string) {
	for _, path := range files {
		if runtime.GOOS == "windows" {
			path = strings.ReplaceAll(path, `\`, `\\`)
		}
		eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", path)

		require.Eventuallyf(
			t,
			func() bool {
				return filebeat.GetLogLine(eofMsg) != ""
			},
			5*time.Second,
			100*time.Millisecond,
			"EOF log not found for %q", path,
		)
	}
}

func waitForDidnotChange(t *testing.T, filebeat *integration.BeatProc, files []string) {
	for _, path := range files {
		eofMsg := fmt.Sprintf("File didn't change: %s", path)

		require.Eventuallyf(
			t,
			func() bool {
				return filebeat.GetLogLine(eofMsg) != ""
			},
			5*time.Second,
			100*time.Millisecond,
			"'File didn't change' log not found for %q", path,
		)
	}
}

// getEventsMsgFromES gets the 'message' field from all documents
// in `index`. If Elasticsearch returns an status code other than 200
// nil is returned. `size` sets the number of documents returned
func getEventsMsgFromES(t *testing.T, index string, size int) []string {
	t.Helper()
	// Step 1: Get the Elasticsearch URL
	esURL := integration.GetESURL(t, "http")

	// Step 2: Format the search URL for the `foo` datastream
	searchURL, err := integration.FormatDataStreamSearchURL(t, esURL, index)
	require.NoError(t, err, "Failed to format datastream search URL")

	// Step 3: Add query parameters
	queryParams := searchURL.Query()

	// Add the `size` (the number of documents returned) parameter
	queryParams.Set("size", strconv.Itoa(size))
	// Order the events in ascending order
	queryParams.Set("sort", "@timestamp:asc")
	// Only request the field we need
	queryParams.Set("_source", "message")
	searchURL.RawQuery = queryParams.Encode()

	// Step 4: Perform the HTTP GET request using integration.HttpDo
	statusCode, body, err := integration.HttpDo(t, "GET", searchURL)
	require.NoError(t, err, "Failed to perform HTTP request")
	if statusCode != 200 {
		return nil
	}

	// Step 5: Parse the response body to extract events
	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Message string `json:"message"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	err = json.Unmarshal(body, &searchResult)
	require.NoError(t, err, "Failed to parse response body")

	// Step 6: Extract the `message` field from each event and return the messages
	messages := []string{}
	for _, hit := range searchResult.Hits.Hits {
		messages = append(messages, hit.Source.Message)
	}

	return messages
}

// DisablingProxy is a HTTP proxy that can be disabled/enabled at runtime
type DisablingProxy struct {
	mu      sync.RWMutex
	enabled bool
	target  *url.URL
}

// ServeHTTP handles incoming requests and forwards them to the target if
// the proxy is enabled.
func (d *DisablingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if !d.enabled {
		http.Error(w, "Proxy is disabled", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(d.target)
	proxy.ServeHTTP(w, r)
}

// Enable enables the proxy.
func (d *DisablingProxy) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

// Disable disables the proxy.
func (d *DisablingProxy) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
}

type FileWatcher struct {
	watcher       *fsnotify.Watcher
	targetFile    string
	mu            sync.Mutex
	stopChan      chan struct{}
	eventCallback func(event fsnotify.Event)
	t             testing.TB
}

// NewFileWatcher creates a new FileWatcher instance.
func NewFileWatcher(t testing.TB, targetFile string) *FileWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %s", err)
	}

	return &FileWatcher{
		watcher:    watcher,
		targetFile: targetFile,
		stopChan:   make(chan struct{}),
		t:          t,
	}
}

// Start begins watching the file's directory for changes.
func (f *FileWatcher) Start() {
	dir := filepath.Dir(f.targetFile)

	err := f.watcher.Add(dir)
	if err != nil {
		f.t.Fatalf("failed to add directory to watcher: %s", err)
	}

	go f.watch()
}

// Stop stops the file-watching process.
func (f *FileWatcher) Stop() {
	close(f.stopChan)
	f.watcher.Close()
}

// SetEventCallback sets a callback function for file events.
// To check the event that happened, use event.Has.
func (f *FileWatcher) SetEventCallback(callback func(event fsnotify.Event)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eventCallback = callback
}

// watch processes file events and errors.
func (f *FileWatcher) watch() {
	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}

			if event.Name == f.targetFile {
				f.mu.Lock()
				if f.eventCallback != nil {
					f.eventCallback(event)
				}
				f.mu.Unlock()
			}

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}

			f.t.Errorf("FileWatcher failed: %s", err)
			return

		case <-f.stopChan:
			return
		}
	}
}

var logFileLines = []string{
	"You can't connect the panel without connecting the wireless AGP panel!",
	"We need to back up the haptic FTP hard drive!",
	"Indexing the array won't do anything, we need to parse the neural SMTP system!",
	"I'Ll generate the haptic TCP pixel, that should transmitter the JBOD application!",
	"I'Ll quantify the wireless XSS driver, that should port the HTTP driver!",
	"If we connect the program, we can get to the ADP alarm through the back-end EXE pixel!",
	"I'Ll generate the primary SSL port, that should firewall the IB firewall!",
	"I'Ll program the digital RSS bus, that should sensor the JSON system!",
	"Hacking the feed won't do anything, we need to input the optical PNG microchip!",
	"We need to synthesize the solid state GB port!",
}
