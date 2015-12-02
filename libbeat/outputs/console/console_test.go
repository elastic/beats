package console

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/stretchr/testify/assert"
)

// capture stdout and return captured string
func withStdout(fn func()) (string, error) {
	stdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()

	outC := make(chan string)
	go func() {
		// capture all output
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		r.Close()
		outC <- buf.String()
	}()

	fn()
	w.Close()
	result := <-outC
	return result, err
}

func event(k, v string) common.MapStr {
	return common.MapStr{k: v}
}

func run(c *console, events ...common.MapStr) (string, error) {
	return withStdout(func() {
		for _, event := range events {
			c.PublishEvent(nil, time.Now(), event)
		}
	})
}

func TestConsoleOneEvent(t *testing.T) {
	lines, err := run(newConsole(false), event("event", "myevent"))
	assert.Nil(t, err)
	expected := "{\"event\":\"myevent\"}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleOneEventIndented(t *testing.T) {
	lines, err := run(newConsole(true), event("event", "myevent"))
	assert.Nil(t, err)
	expected := "{\n  \"event\": \"myevent\"\n}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleMultipleEvents(t *testing.T) {
	lines, err := run(newConsole(false),
		event("event", "event1"),
		event("event", "event2"),
		event("event", "event3"),
	)

	assert.Nil(t, err)
	expected := "{\"event\":\"event1\"}\n{\"event\":\"event2\"}\n{\"event\":\"event3\"}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleMultipleEventsIndented(t *testing.T) {
	lines, err := run(newConsole(true),
		event("event", "event1"),
		event("event", "event2"),
		event("event", "event3"),
	)

	assert.Nil(t, err)
	expected := "{\n  \"event\": \"event1\"\n}\n" +
		"{\n  \"event\": \"event2\"\n}\n" +
		"{\n  \"event\": \"event3\"\n}\n"
	assert.Equal(t, expected, lines)
}
