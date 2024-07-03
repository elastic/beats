package journalctl

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"testing"
)

//go:embed testdata/coredump.json
var coredumpJSON []byte

// TestEventWithNonStringData ensures the Reader can read data that is not a
// string. There is at least one real example of that: coredumps.
// This test uses a real example captured from journalctl -o json.
//
// If needed more test cases can be added in the future
func TestEventWithNonStringData(t *testing.T) {
	stdout := io.NopCloser(&bytes.Buffer{})
	stderr := io.NopCloser(&bytes.Buffer{})
	r := Reader{
		dataChan: make(chan []byte),
		errChan:  make(chan error),
		stdout:   stdout,
		stderr:   stderr,
	}

	go func() {
		r.dataChan <- []byte(coredumpJSON)
	}()

	_, err := r.Next(context.Background())
	if err != nil {
		t.Fatalf("did not expect an error: %s", err)
	}
}
