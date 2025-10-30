//go:build integration

package filestream

import (
	"os"
	"path/filepath"
	"testing"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// TestFileWatcherClosedHarvesterNotification tests the case:
// - watch runs and sees a new file, it sends a create event
// - data is added to the file
// - watch runs and sends a write event
// - notification with a smaller size is received
// - scan runs again, uses the size from the notification and sends a write event
// - closedHarvesters is empty at the end of this scan
func TestFileWatcherClosedHarvesterNotification(t *testing.T) {
	dir := integration.CreateTempDir(
		t,
		filepath.Join("..", "..", "build", "integration-tests"),
	)

	// Create a 3000 bytes file
	logFilePath := filepath.Join(dir, "log.log")
	integration.WriteLogFile(t, logFilePath, 50, false)

	cfg := defaultFileWatcherConfig()
	fw, err := newFileWatcher(
		logptest.NewFileLogger(t, filepath.Join("..", "..", "build", "integration-tests")).Logger,
		[]string{filepath.Join(dir, "*.log")},
		cfg,
		false,
		false,
		mustFingerprintIdentifier(),
		mustSourceIdentifier("foo-id"),
	)
	if err != nil {
		t.Fatalf("cannot create file watcher: %s", err)
	}
	// Use a buffered channel so we don't block when writing on the channel
	fw.events = make(chan loginp.FSEvent, 1)

	// Scan the file system once
	fw.watch(t.Context())
	evt := <-fw.events
	requireOperation(t, evt, loginp.OpCreate)

	// Write to the file again, so we get a write operation
	integration.WriteLogFile(t, logFilePath, 10, true)
	fw.watch(t.Context())
	evt = <-fw.events
	requireOperation(t, evt, loginp.OpWrite)

	// Check the filewatcher state
	// Use the path from the event to be consistent with the
	// fileWatcher implementation
	stateSize := fw.prev[evt.NewPath].Size()
	// 50 bytes per line, 60 lines = 3000 bytes
	if stateSize != 3000 {
		t.Fatalf(
			"fileWatcher internal state is different from file size, expecting %d got %d",
			3000,
			stateSize)
	}

	// Notify the harvester has closed with a smaller size
	fw.processNotification(loginp.HarvesterStatus{
		ID:   evt.SrcID,
		Size: 2500, // anything smaller than the real size
	})

	// Ensure closedHarvester is populated
	if _, ok := fw.closedHarvesters[evt.SrcID]; !ok {
		t.Fatal("closed harvester notification did not populate 'closedHarvesters'")
	}
	fw.watch(t.Context())
	evt = <-fw.events
	// Because of the notification sent with a smaller size than the actual file
	// we should get a write operation
	requireOperation(t, evt, loginp.OpWrite)

	// And closedHarvesters must be empty
	l := len(fw.closedHarvesters)
	if l != 0 {
		t.Fatalf("expecting 'closedHarvesters' to be empty, got %d items", l)
	}
}

// TestFileWatcherClosedHarvesterNotificationFullyIngested tests the default
// case of a harvester closing after fully ingesting the file. It also ensure
// entries in closedHarvesters are correctly removed.
func TestFileWatcherClosedHarvesterNotificationFullyIngested(t *testing.T) {
	dir := integration.CreateTempDir(
		t,
		filepath.Join("..", "..", "build", "integration-tests"),
	)

	// Create a 3000 bytes file
	logFilePath := filepath.Join(dir, "log.log")
	integration.WriteLogFile(t, logFilePath, 50, false)

	cfg := defaultFileWatcherConfig()

	fw, err := newFileWatcher(
		logptest.NewFileLogger(t, filepath.Join("..", "..", "build", "integration-tests")).Logger,
		[]string{filepath.Join(dir, "*.log")},
		cfg,
		false,
		false,
		mustFingerprintIdentifier(),
		mustSourceIdentifier("foo-id"),
	)
	if err != nil {
		t.Fatalf("cannot create file watcher: %s", err)
	}
	// Use a buffered channel so we don't block when writing on the channel
	fw.events = make(chan loginp.FSEvent, 1)

	// Scan the file system once
	fw.watch(t.Context())
	evt := <-fw.events
	requireOperation(t, evt, loginp.OpCreate)

	// Notify the harvester has closed, file fully ingested
	fw.processNotification(loginp.HarvesterStatus{
		ID:   evt.SrcID,
		Size: 3000,
	})

	// Ensure closedHarvester is populated
	if _, ok := fw.closedHarvesters[evt.SrcID]; !ok {
		t.Fatal("closed harvester notification did not populate 'closedHarvesters'")
	}
	fw.watch(t.Context())

	// The fileWatcher state has not changed, no events should be generated
	eventsWritten := len(fw.events)
	if eventsWritten != 0 {
		t.Fatalf("expecting 0 events generated, got %d", eventsWritten)
	}

	// closedHarvesters must be empty
	l := len(fw.closedHarvesters)
	if l != 0 {
		t.Fatalf("expecting 'closedHarvesters' to be empty, got %d items", l)
	}
}

func TestFileWatcherClosedHarvesterNotificationOfRemovedFile(t *testing.T) {
	dir := integration.CreateTempDir(
		t,
		filepath.Join("..", "..", "build", "integration-tests"),
	)

	// Create a 3000 bytes file
	logFilePath := filepath.Join(dir, "log.log")
	integration.WriteLogFile(t, logFilePath, 50, false)

	cfg := defaultFileWatcherConfig()

	fw, err := newFileWatcher(
		logptest.NewFileLogger(t, filepath.Join("..", "..", "build", "integration-tests")).Logger,
		[]string{filepath.Join(dir, "*.log")},
		cfg,
		false,
		false,
		mustFingerprintIdentifier(),
		mustSourceIdentifier("foo-id"),
	)
	if err != nil {
		t.Fatalf("cannot create file watcher: %s", err)
	}
	// Use a buffered channel so we don't block when writing on the channel
	fw.events = make(chan loginp.FSEvent, 1)

	// Scan the file system once
	fw.watch(t.Context())
	evt := <-fw.events
	requireOperation(t, evt, loginp.OpCreate)

	// Notify the harvester has closed, file fully ingested
	fw.processNotification(loginp.HarvesterStatus{
		ID:   evt.SrcID,
		Size: 3000,
	})

	// Ensure closedHarvester is populated
	if _, ok := fw.closedHarvesters[evt.SrcID]; !ok {
		t.Fatal("closed harvester notification did not populate 'closedHarvesters'")
	}

	// Remove the file
	if err := os.Remove(logFilePath); err != nil {
		t.Fatalf("cannot remove log file: %s", err)
	}

	// A delete event must be generated
	fw.watch(t.Context())
	evt = <-fw.events
	requireOperation(t, evt, loginp.OpDelete)

	// closedHarvesters must be empty
	l := len(fw.closedHarvesters)
	if l != 0 {
		t.Fatalf("expecting 'closedHarvesters' to be empty, got %d items", l)
	}
}

func TestFileWatcherClosedHarvesterNotificationRenamedFile(t *testing.T) {
	dir := integration.CreateTempDir(
		t,
		filepath.Join("..", "..", "build", "integration-tests"),
	)

	// Create a 3000 bytes file
	logFilePath := filepath.Join(dir, "log.log")
	integration.WriteLogFile(t, logFilePath, 50, false)

	cfg := defaultFileWatcherConfig()

	fw, err := newFileWatcher(
		logptest.NewFileLogger(t, filepath.Join("..", "..", "build", "integration-tests")).Logger,
		[]string{filepath.Join(dir, "*.log")},
		cfg,
		false,
		false,
		mustFingerprintIdentifier(),
		mustSourceIdentifier("foo-id"),
	)
	if err != nil {
		t.Fatalf("cannot create file watcher: %s", err)
	}
	// Use a buffered channel so we don't block when writing on the channel
	fw.events = make(chan loginp.FSEvent, 1)

	// Scan the file system once
	fw.watch(t.Context())
	evt := <-fw.events
	requireOperation(t, evt, loginp.OpCreate)

	// Notify the harvester has closed, file fully ingested
	fw.processNotification(loginp.HarvesterStatus{
		ID:   evt.SrcID,
		Size: 3000,
	})

	// Ensure closedHarvester is populated
	if _, ok := fw.closedHarvesters[evt.SrcID]; !ok {
		t.Fatal("closed harvester notification did not populate 'closedHarvesters'")
	}

	// Remove the file
	newPath := filepath.Join(dir, "log1.log")
	if err := os.Rename(logFilePath, newPath); err != nil {
		t.Fatalf("cannot rename log file: %s", err)
	}

	// A rename event must be generated
	fw.watch(t.Context())
	evt = <-fw.events
	requireOperation(t, evt, loginp.OpRename)

	// closedHarvesters still hold the file's entry
	if _, ok := fw.closedHarvesters[evt.SrcID]; !ok {
		t.Fatal("closedHarvesters must still contain the entry for a renamed file/")
	}
}

func requireOperation(t *testing.T, evt loginp.FSEvent, op loginp.Operation) {
	t.Helper()
	if evt.Op != op {
		t.Fatalf("expecting operation %q, got: %q", op.String(), evt.Op.String())
	}
}
