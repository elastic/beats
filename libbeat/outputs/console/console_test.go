// +build !integration

package console

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codecs/format"
	"github.com/elastic/beats/libbeat/outputs/codecs/json"
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

func run(codec outputs.Codec, events ...common.MapStr) (string, error) {
	return withStdout(func() {
		c, _ := newConsole(codec)
		for _, event := range events {
			c.PublishEvent(nil, outputs.Options{}, outputs.Data{Event: event})
		}
	})
}

func TestConsoleOneEvent(t *testing.T) {
	lines, err := run(json.New(false), event("event", "myevent"))
	assert.Nil(t, err)
	expected := "{\"event\":\"myevent\"}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleOneEventIndented(t *testing.T) {
	lines, err := run(json.New(true), event("event", "myevent"))
	assert.Nil(t, err)
	expected := "{\n  \"event\": \"myevent\"\n}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleOneEventFormatted(t *testing.T) {
	lines, err := run(
		format.New(fmtstr.MustCompileEvent("%{[event]}")),
		event("event", "myevent"),
	)
	assert.Nil(t, err)
	expected := "myevent\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleMultipleEvents(t *testing.T) {
	lines, err := run(json.New(false),
		event("event", "event1"),
		event("event", "event2"),
		event("event", "event3"),
	)

	assert.Nil(t, err)
	expected := "{\"event\":\"event1\"}\n{\"event\":\"event2\"}\n{\"event\":\"event3\"}\n"
	assert.Equal(t, expected, lines)
}

func TestConsoleMultipleEventsIndented(t *testing.T) {
	lines, err := run(json.New(true),
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
