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
	defer fileWatcher.Stop()
	fileWatcher.Start()

	esUrl := integration.GetESURL(t, "http")
	user := esUrl.User.Username()
	pass, _ := esUrl.User.Password()
	vars := map[string]any{
		"homePath": workDir,
		"logfile":  logFile,
		"testdata": testDataPath,
		"esHost":   (&esUrl).String(),
		"user":     user,
		"pass":     pass,
		"index":    index,
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

	fileWatcher.SetFailOnDelete(false)
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

	for i, msg := range msgs {
		if msg != logFileLines[i] {
			t.Errorf("Log entry %d is different than expected", i)
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
	failOnDelete  bool
	mu            sync.Mutex
	stopChan      chan struct{}
	eventCallback func(event fsnotify.Event)
	errorCallback func(err error)
	t             testing.TB
}

// NewFileWatcher creates a new FileWatcher instance.
func NewFileWatcher(t testing.TB, targetFile string, failOnDelete bool) *FileWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %s", err)
	}

	return &FileWatcher{
		watcher:      watcher,
		targetFile:   targetFile,
		failOnDelete: failOnDelete,
		stopChan:     make(chan struct{}),
		t:            t,
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
func (fw *FileWatcher) SetEventCallback(callback func(event fsnotify.Event)) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.eventCallback = callback
}

// SetErrorCallback sets a callback function for errors.
func (fw *FileWatcher) SetErrorCallback(callback func(err error)) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.errorCallback = callback
}

// SetFailOnDelete dynamically updates the behavior of the FileWatcher
// to determine whether the test should fail if the watched file is deleted.
//
// This method is thread-safe and can be called at any time during the test.
func (fw *FileWatcher) SetFailOnDelete(fail bool) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.failOnDelete = fail
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
				// TODO: Update to use the callback
				if event.Has(fsnotify.Remove) && fw.failOnDelete {
					fw.t.Errorf("[%s] File %s could not have been removed", time.Now().Format(time.RFC3339Nano), event.Name)
				}

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

			fw.mu.Lock()
			if fw.errorCallback != nil {
				fw.errorCallback(err)
			} else {
				fw.t.Errorf("Watcher error: %s", err)
			}
			fw.mu.Unlock()

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
