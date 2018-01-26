package tcp

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
)

type testingOutlet struct {
	ch chan *util.Data
}

func (o *testingOutlet) OnEvent(data *util.Data) bool {
	o.ch <- data
	return true
}

func newTestingOutlet(ch chan *util.Data) *testingOutlet {
	return &testingOutlet{ch: ch}
}

// This could be extracted into testing utils and we could add some unicode chars to the charsets.
func randomString(l int) string {
	charsets := []byte("abcdefghijklmnopqrstuvwzyzABCDEFGHIJKLMNOPQRSTUVWZYZ0123456789")
	message := make([]byte, l)
	for i := range message {
		message[i] = charsets[rand.Intn(len(charsets))]
	}
	return string(message)
}

func generateMessages(c int, l int) []string {
	messages := make([]string, c)
	for i := range messages {
		messages[i] = randomString(l)
	}
	return messages
}

func TestErrorOnEmptyLineDelimiter(t *testing.T) {
	cfg := map[string]interface{}{
		"line_delimiter": "",
	}

	c, _ := common.NewConfigFrom(cfg)
	forwarder := harvester.NewForwarder(nil)
	_, err := NewHarvester(forwarder, c)
	assert.Error(t, err)
}

func TestOverrideHostAndPort(t *testing.T) {
	host := "127.0.0.1:10000"
	cfg := map[string]interface{}{
		"host": host,
	}
	c, _ := common.NewConfigFrom(cfg)
	forwarder := harvester.NewForwarder(nil)
	harvester, err := NewHarvester(forwarder, c)
	defer harvester.Stop()
	go func() {
		err := harvester.Run()
		assert.NoError(t, err)
	}()
	conn, err := net.Dial("tcp", host)
	defer conn.Close()
	assert.NoError(t, err)
}

func TestReceiveNewEventsConcurrently(t *testing.T) {
	workers := 4
	eventsCount := 100
	ch := make(chan *util.Data, eventsCount*workers)
	defer close(ch)
	to := newTestingOutlet(ch)
	forwarder := harvester.NewForwarder(to)
	cfg := common.NewConfig()
	harvester, err := NewHarvester(forwarder, cfg)
	defer harvester.Stop()
	if !assert.NoError(t, err) {
		return
	}
	go func() {
		err := harvester.Run()
		assert.NoError(t, err)
	}()

	samples := generateMessages(eventsCount, 1024)
	for w := 0; w < workers; w++ {
		go func() {
			conn, err := net.Dial("tcp", "localhost:9000")
			defer conn.Close()
			assert.NoError(t, err)
			for _, sample := range samples {
				fmt.Fprintln(conn, sample)
			}
		}()
	}

	var events []*util.Data
	for len(events) < eventsCount*workers {
		select {
		case event := <-ch:
			events = append(events, event)
		case <-time.After(time.Second * 10):
			t.Fatal("timeout when waiting on channel")
			return
		}
	}
}

func TestReceiveEventsAndMetadata(t *testing.T) {
	expectedMessages := generateMessages(5, 100)
	largeMessages := generateMessages(10, 4096)

	tests := []struct {
		name             string
		cfg              map[string]interface{}
		expectedMessages []string
		messageSent      string
	}{
		{
			name:             "NewLine",
			cfg:              map[string]interface{}{},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\n"),
		},
		{
			name:             "NewLineWithCR",
			cfg:              map[string]interface{}{},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\r\n"),
		},
		{
			name: "CustomDelimiter",
			cfg: map[string]interface{}{
				"line_delimiter": ";",
			},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, ";"),
		},
		{
			name: "MultipleCharsCustomDelimiter",
			cfg: map[string]interface{}{
				"line_delimiter": "<END>",
			},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "<END>"),
		},
		{
			name: "SingleCharCustomDelimiterMessageWithoutBoudaries",
			cfg: map[string]interface{}{
				"line_delimiter": ";",
			},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name: "MultipleCharCustomDelimiterMessageWithoutBoundaries",
			cfg: map[string]interface{}{
				"line_delimiter": "<END>",
			},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name: "NewLineMessageWithoutBoundaries",
			cfg: map[string]interface{}{
				"line_delimiter": "\n",
			},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name: "NewLineLargeMessagePayload",
			cfg: map[string]interface{}{
				"line_delimiter": "\n",
			},
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, "\n"),
		},
		{
			name: "CustomLargeMessagePayload",
			cfg: map[string]interface{}{
				"line_delimiter": ";",
			},
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, ";"),
		},
		{
			name:             "MaxReadBufferReached",
			cfg:              map[string]interface{}{},
			expectedMessages: []string{},
			messageSent:      randomString(900000),
		},
		{
			name: "MaxReadBufferReachedUserConfigured",
			cfg: map[string]interface{}{
				"max_read_message": 50000,
			},
			expectedMessages: []string{},
			messageSent:      randomString(600000),
		},
	}

	port := 9000
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan *util.Data, len(test.expectedMessages))
			defer close(ch)
			to := newTestingOutlet(ch)
			forwarder := harvester.NewForwarder(to)
			test.cfg["host"] = fmt.Sprintf("localhost:%d", port)
			cfg, _ := common.NewConfigFrom(test.cfg)
			harvester, err := NewHarvester(forwarder, cfg)
			if !assert.NoError(t, err) {
				return
			}
			defer func() {
				harvester.Stop()
			}()
			go func() {
				err := harvester.Run()
				assert.NoError(t, err)
			}()

			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			assert.NoError(t, err)
			fmt.Fprint(conn, test.messageSent)
			conn.Close()

			var events []*util.Data

			for len(events) < len(test.expectedMessages) {
				select {
				case event := <-ch:
					events = append(events, event)
				case <-time.After(time.Second * 10):
					t.Fatal("could not drain all the elements")
					return
				}
			}

			for idx, e := range events {
				event := e.GetEvent()
				message, err := event.GetValue("message")
				assert.NoError(t, err)
				assert.Equal(t, test.expectedMessages[idx], message)
				meta := e.GetMetadata()
				_, ok := meta["hostnames"]
				assert.True(t, ok)
				_, ok = meta["ip_address"]
				assert.True(t, ok)
			}
		})
		port++
	}
}
