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

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamDeleteRealESFSNotify(t *testing.T) {
	gracePeriod, err := time.ParseDuration("5s")
	if err != nil {
		t.Fatalf("cannot parse grace period duration: %s", err)
	}
	delta := time.Second

	index := "test-delete" + uuid.Must(uuid.NewV4()).String()
	testDataPath, err := filepath.Abs("./testdata")
	if err != nil {
		t.Fatalf("cannot get absolute path for 'testdata': %s", err)
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	logFile := filepath.Join(workDir, "log.log")
	logData := strings.Join(logFileLines[:5], "\n")
	logData += "\n" // Filebeat needs the '\n' to read the last line
	if err := os.WriteFile(logFile, []byte(logData), 0o644); err != nil {
		t.Fatalf("cannot write log file '%s': %s", logFile, err)
	}

	fileWatcher := NewFileWatcher(t, logFile)
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			t.Errorf("File %s should not have been removed, removal happened at %s",
				event.Name,
				time.Now().Format(time.RFC3339Nano))
		}
	})
	fileWatcher.Start()
	defer fileWatcher.Stop()

	esURL := integration.GetESURL(t, "http")

	// Create and start the proxy server
	proxy := &DisablingProxy{target: &esURL, enabled: true}
	server := &http.Server{
		Addr:    "localhost:9201",
		Handler: proxy,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Proxy server failed: %s", err)
		}
	}()
	defer server.Close()

	proxyURL, err := url.Parse(server.Addr)
	if err != nil {
		t.Fatalf("cannot parse proxy URL: %s", err)
	}

	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	vars := map[string]any{
		"homePath":    workDir,
		"logfile":     logFile,
		"testdata":    testDataPath,
		"esHost":      proxyURL.String(),
		"user":        user,
		"pass":        pass,
		"index":       index,
		"gracePeriod": gracePeriod.String(),
	}

	cfgYAML := getConfig(t, vars, "delete", "real-es.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// Wait for data in ES
	msgs := []string{}
	require.Eventually(t, func() bool {
		msgs = getEventsMsgFromES(t, index, 200)
		return len(msgs) == len(logFileLines)/2
	}, time.Second*10, time.Millisecond*100, "not all log messages have been found on ES")

	// Wait for 1/2 of the grace period and add more data
	time.Sleep(gracePeriod / 2)

	// Add more data to the file
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("cannot open logfile to append data: %s", err)
	}
	logData2 := strings.Join(logFileLines[5:], "\n")
	logData2 += "\n"
	if _, err := f.WriteString(logData2); err != nil {
		t.Fatalf("could not append data to log file: %s", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("cannot flush log file: %s", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("cannot close log file: %s", err)
	}

	// Disable (aka block) the output
	proxy.Disable()

	// Wait twice the grace period before unblocking the output
	blockedTimer := time.NewTimer(gracePeriod * 2)
	<-blockedTimer.C

	// Ensure log file still exists
	if !fileExists(t, logFile) {
		t.Fatal("file was removed while output was blocked")
	}

	// Unblock the output
	proxy.Enable()

	// Wait for the remaining data to be ingested
	msgs = []string{}
	require.Eventually(t, func() bool {
		msgs = getEventsMsgFromES(t, index, 200)
		return len(msgs) == len(logFileLines)
	}, time.Second*10, time.Millisecond*100, "not all log messages have been found on ES")

	dataShippedTs := time.Now()
	fileRemovedChan := make(chan time.Time)
	// All events have been found, allow file to be removed
	// and get the removal timestamp
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			fileRemovedChan <- time.Now()
		}
	})

	deleteTimeout := gracePeriod * 3
	timeout := time.NewTimer(deleteTimeout)
	select {
	case fileRemovedTs := <-fileRemovedChan:
		timeElapsed := fileRemovedTs.Sub(dataShippedTs)
		if timeElapsed < gracePeriod-delta {
			t.Fatalf("file was removed %s after data ingested (%s acceptable delta), but grace period was set to %s",
				timeElapsed,
				delta,
				gracePeriod)
		}
	case <-timeout.C:
		t.Fatalf("file was not removed within %d", deleteTimeout)
	}

	// Ensure the messages were ingested in the correct order
	for i, msg := range msgs {
		if msg != logFileLines[i] {
			t.Errorf("Line %d: want: %q, have: %q", i, logFileLines[i], msg)
		}
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
