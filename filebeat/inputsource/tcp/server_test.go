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

package tcp

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var defaultConfig = Config{
	Timeout:        time.Minute * 5,
	MaxMessageSize: 20 * humanize.MiByte,
}

type info struct {
	message string
	mt      inputsource.NetworkMetadata
}

func TestErrorOnEmptyLineDelimiter(t *testing.T) {
	c := conf.NewConfig()
	config := defaultConfig
	err := c.Unpack(&config)
	assert.Error(t, err)
}

func TestReceiveEventsAndMetadata(t *testing.T) {
	expectedMessages := generateMessages(5, 100)
	largeMessages := generateMessages(10, 4096)
	extraLargeMessages := generateMessages(2, 65*1024)
	randomGeneratedText := randomString(900000)

	tests := []struct {
		name             string
		cfg              map[string]interface{}
		framing          streaming.FramingType
		delimiter        []byte
		splitFunc        bufio.SplitFunc
		expectedMessages []string
		messageSent      string
	}{
		{
			name:             "NewLine",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("\n"),
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\n"),
		},
		{
			name:             "NewLineWithCR",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("\r\n"),
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\r\n"),
		},
		{
			name:             "CustomDelimiter",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte(";"),
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, ";"),
		},
		{
			name:             "MultipleCharsCustomDelimiter",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("<END>"),
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "<END>"),
		},
		{
			name:             "SingleCharCustomDelimiterMessageWithoutBoundaries",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte(";"),
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "MultipleCharCustomDelimiterMessageWithoutBoundaries",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("<END>"),
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "NewLineMessageWithoutBoundaries",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("\n"),
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "NewLineLargeMessagePayload",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("\n"),
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, "\n"),
		},
		{
			name:             "CustomLargeMessagePayload",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte(";"),
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, ";"),
		},
		{
			name:             "ReadRandomLargePayload",
			cfg:              map[string]interface{}{},
			framing:          streaming.FramingDelimiter,
			delimiter:        []byte("\n"),
			expectedMessages: []string{randomGeneratedText},
			messageSent:      randomGeneratedText,
		},
		{
			name:      "MaxReadBufferReachedUserConfigured",
			framing:   streaming.FramingDelimiter,
			delimiter: []byte("\n"),
			cfg: map[string]interface{}{
				"max_message_size": 50000,
			},
			expectedMessages: []string{},
			messageSent:      randomGeneratedText,
		},
		{
			name:      "MaxBufferSizeSet",
			framing:   streaming.FramingDelimiter,
			delimiter: []byte("\n"),
			cfg: map[string]interface{}{
				"max_message_size": 66 * 1024,
			},
			expectedMessages: extraLargeMessages,
			messageSent:      strings.Join(extraLargeMessages, "\n"),
		},
		{
			name:      "rfc6587 framing non-transparent",
			framing:   streaming.FramingRFC6587,
			delimiter: []byte("\n"),
			cfg:       map[string]interface{}{},
			expectedMessages: []string{
				"<9> message 0",
				"<6> msg 1",
				"<3> message 2",
			},
			messageSent: "<9> message 0\n<6> msg 1\n<3> message 2",
		},
		{
			name:      "rfc6587 framing octet",
			cfg:       map[string]interface{}{},
			framing:   streaming.FramingRFC6587,
			delimiter: []byte("\n"),
			expectedMessages: []string{
				"<9> message 0",
				"<6> msg 1",
				"<3> message 2",
			},
			messageSent: "13 <9> message 09 <6> msg 113 <3> message 2",
		},
		{
			name:      "rfc6587 framing octet embedded newline",
			cfg:       map[string]interface{}{},
			framing:   streaming.FramingRFC6587,
			delimiter: []byte("\n"),
			expectedMessages: []string{
				"<9> message \n0",
				"<6> msg \n1",
				"<3> message \n2",
			},
			messageSent: "14 <9> message \n010 <6> msg \n114 <3> message \n2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan *info, len(test.expectedMessages))
			defer close(ch)
			to := func(message []byte, mt inputsource.NetworkMetadata) {
				ch <- &info{message: string(message), mt: mt}
			}
			test.cfg["host"] = "localhost:0"
			cfg, _ := conf.NewConfigFrom(test.cfg)
			config := defaultConfig
			err := cfg.Unpack(&config)
			if !assert.NoError(t, err) {
				return
			}

			splitFunc, err := streaming.SplitFunc(test.framing, test.delimiter)
			if !assert.NoError(t, err) {
				return
			}

			factory := streaming.SplitHandlerFactory(inputsource.FamilyTCP, logp.NewLogger("test"), MetadataCallback, to, splitFunc)
			server, err := New(&config, factory)
			if !assert.NoError(t, err) {
				return
			}
			err = server.Start()
			if !assert.NoError(t, err) {
				return
			}
			defer server.Stop()

			conn, err := net.Dial("tcp", server.Listener.Listener.Addr().String())
			require.NoError(t, err)
			fmt.Fprint(conn, test.messageSent)
			conn.Close()

			var events []*info

			for len(events) < len(test.expectedMessages) {
				select {
				case event := <-ch:
					events = append(events, event)
				default:
				}
			}

			for idx, e := range events {
				assert.Equal(t, test.expectedMessages[idx], e.message)
				assert.NotNil(t, e.mt.RemoteAddr)
			}
		})
	}
}

func TestReceiveNewEventsConcurrently(t *testing.T) {
	workers := 4
	eventsCount := 100
	ch := make(chan *info, eventsCount*workers)
	defer close(ch)
	to := func(message []byte, mt inputsource.NetworkMetadata) {
		ch <- &info{message: string(message), mt: mt}
	}
	cfg, err := conf.NewConfigFrom(map[string]interface{}{"host": "127.0.0.1:0"})
	if !assert.NoError(t, err) {
		return
	}
	config := defaultConfig
	err = cfg.Unpack(&config)
	if !assert.NoError(t, err) {
		return
	}

	factory := streaming.SplitHandlerFactory(inputsource.FamilyTCP, logp.NewLogger("test"), MetadataCallback, to, bufio.ScanLines)

	server, err := New(&config, factory)
	if !assert.NoError(t, err) {
		return
	}
	err = server.Start()
	if !assert.NoError(t, err) {
		return
	}
	defer server.Stop()

	samples := generateMessages(eventsCount, 1024)
	for w := 0; w < workers; w++ {
		go func() {
			conn, err := net.Dial("tcp", server.Listener.Listener.Addr().String())
			defer conn.Close()
			assert.NoError(t, err)
			for _, sample := range samples {
				fmt.Fprintln(conn, sample)
			}
		}()
	}

	var events []*info
	for len(events) < eventsCount*workers {
		select {
		case event := <-ch:
			events = append(events, event)
		default:
		}
	}
}

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
