//go:build integration

package integration

import (
	"encoding/json"
	"errors"
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

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
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
	logData := strings.Join(logFileLines, "\n")
	logData += "\n" // Filebeat needs the '\n' to read the last line
	if err := os.WriteFile(logFile, []byte(logData), 0o644); err != nil {
		t.Fatalf("cannot write log file '%s': %s", logFile, err)
	}

	fileWatcher := NewFileWatcher(t, logFile, true)
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			t.Errorf("File %s should not have been removed, removal happened at %s",
				time.Now().Format(time.RFC3339Nano), event.Name)
		}
	})
	fileWatcher.Start()
	defer fileWatcher.Stop()

	esURL := integration.GetESURL(t, "http")

	// Create and start the proxy server
	proxy := &ProxyController{target: &esURL, enabled: true}
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
		return len(msgs) == len(logFileLines)
	}, time.Second*10, time.Millisecond*100, "not all log messages have been found on ES")

	dataShippedTs := time.Now()
	fileRemovedChan := make(chan time.Time)
	// All events have been found, allow file to be removed
	fileWatcher.SetEventCallback(func(event fsnotify.Event) {
		if event.Has(fsnotify.Remove) {
			fileRemovedChan <- time.Now()
		}
	})

	// Wait for the file to be removed
	require.Eventually(t, func() bool {
		_, err := os.Stat(logFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false
			}
			t.Fatalf("cannot stat file: %s", err)
		}

		return true
	}, 10*time.Second, time.Second, "file has not been removed")

	timeout := time.NewTimer(time.Second * 15)
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
		t.Fatalf("file was not removed within 15s")
	}

	for i, msg := range msgs {
		if msg != logFileLines[i] {
			t.Errorf("want: %q, have: %q", logFileLines[i], msg)
		}
	}
}

func getEventsMsgFromES(t *testing.T, index string, size int) []string {
	t.Helper()
	// Step 1: Get the Elasticsearch URL
	esURL := integration.GetESURL(t, "http")

	// Step 2: Format the search URL for the `foo` datastream
	searchURL, err := integration.FormatDataStreamSearchURL(t, esURL, index)
	require.NoError(t, err, "Failed to format datastream search URL")

	// Step 3: Add the `size` parameter to fetch up to 200 messages
	queryParams := searchURL.Query()
	queryParams.Set("size", strconv.Itoa(size))
	searchURL.RawQuery = queryParams.Encode()

	// Step 4: Perform the HTTP GET request using integration.HttpDo
	_, body, err := integration.HttpDo(t, "GET", searchURL)
	require.NoError(t, err, "Failed to perform HTTP request")

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

// ProxyController controls the state of the proxy (enabled/disabled).
type ProxyController struct {
	mu      sync.RWMutex
	enabled bool
	target  *url.URL
}

// ServeHTTP handles incoming requests and forwards them to the target if enabled.
func (p *ProxyController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		http.Error(w, "Proxy is disabled", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(p.target)
	proxy.ServeHTTP(w, r)
}

// Enable enables the proxy.
func (p *ProxyController) Enable() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = true
}

// Disable disables the proxy.
func (p *ProxyController) Disable() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = false
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
func NewFileWatcher(t testing.TB, targetFile string, failOnDelete bool) *FileWatcher {
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

// Start begins watching the file and its directory for changes.
func (fw *FileWatcher) Start() {
	dir := filepath.Dir(fw.targetFile)

	err := fw.watcher.Add(dir)
	if err != nil {
		fw.t.Fatalf("failed to add directory to watcher: %s", err)
	}

	go fw.watch()
}

// Stop stops the file-watching process.
func (fw *FileWatcher) Stop() {
	close(fw.stopChan)
	fw.watcher.Close()
}

// SetEventCallback sets a callback function for file events.
// To check the event that happened, use event.Has.
func (fw *FileWatcher) SetEventCallback(callback func(event fsnotify.Event)) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.eventCallback = callback
}

// watch processes file events and errors.
func (fw *FileWatcher) watch() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			if event.Name == fw.targetFile {
				fw.t.Logf("[%s] Event %s", time.Now().Format(time.RFC3339Nano), event.Op.String())
				fw.mu.Lock()
				if fw.eventCallback != nil {
					fw.eventCallback(event)
				}
				fw.mu.Unlock()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}

			fw.t.Errorf("FileWatcher failed: %s", err)
		case <-fw.stopChan:
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
