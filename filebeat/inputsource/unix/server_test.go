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

package unix

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func defaultConfig() Config {
	return Config{
		Timeout:        time.Minute * 5,
		MaxMessageSize: 20 * humanize.MiByte,
		SocketType:     StreamSocket,
	}
}

type info struct {
	message string
	mt      inputsource.NetworkMetadata
}

func TestErrorOnInvalidSocketType(t *testing.T) {
	config := &Config{
		SocketType: SocketType(7),
	}
	_, err := New(logp.L(), config, nil)
	assert.Error(t, err)
}

func TestErrorOnEmptyLineDelimiter(t *testing.T) {
	config := &Config{
		SocketType:    StreamSocket,
		LineDelimiter: "",
	}
	_, err := New(logp.L(), config, nil)
	assert.Error(t, err)
}

func TestReceiveEventsAndMetadata(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test is only supported on non-windows. See https://github.com/elastic/beats/issues/19641")
		return
	}
	expectedMessages := generateMessages(5, 100)
	largeMessages := generateMessages(10, 4096)
	extraLargeMessages := generateMessages(2, 65*1024)
	randomGeneratedText := randomString(900000)

	tests := []struct {
		name             string
		cfg              map[string]interface{}
		expectedMessages []string
		messageSent      string
	}{
		{
			name:             "NewLine",
			cfg:              map[string]interface{}{"line_delimiter": "\n"},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\n"),
		},
		{
			name:             "NewLineWithCR",
			cfg:              map[string]interface{}{"line_delimiter": "\r\n"},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "\r\n"),
		},
		{
			name:             "CustomDelimiter",
			cfg:              map[string]interface{}{"line_delimiter": ";"},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, ";"),
		},
		{
			name:             "MultipleCharsCustomDelimiter",
			cfg:              map[string]interface{}{"line_delimiter": "<END>"},
			expectedMessages: expectedMessages,
			messageSent:      strings.Join(expectedMessages, "<END>"),
		},
		{
			name:             "SingleCharCustomDelimiterMessageWithoutBoundaries",
			cfg:              map[string]interface{}{"line_delimiter": ";"},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "MultipleCharCustomDelimiterMessageWithoutBoundaries",
			cfg:              map[string]interface{}{"line_delimiter": "<END>"},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "NewLineMessageWithoutBoundaries",
			cfg:              map[string]interface{}{"line_delimiter": "\n"},
			expectedMessages: []string{"hello"},
			messageSent:      "hello",
		},
		{
			name:             "NewLineLargeMessagePayload",
			cfg:              map[string]interface{}{"line_delimiter": "\n"},
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, "\n"),
		},
		{
			name:             "CustomLargeMessagePayload",
			cfg:              map[string]interface{}{"line_delimiter": ";"},
			expectedMessages: largeMessages,
			messageSent:      strings.Join(largeMessages, ";"),
		},
		{
			name:             "ReadRandomLargePayload",
			cfg:              map[string]interface{}{"line_delimiter": "\n"},
			expectedMessages: []string{randomGeneratedText},
			messageSent:      randomGeneratedText,
		},
		{
			name: "MaxReadBufferReachedUserConfigured",
			cfg: map[string]interface{}{
				"line_delimiter":   "\n",
				"max_message_size": 50000,
			},
			expectedMessages: []string{},
			messageSent:      randomGeneratedText,
		},
		{
			name: "MaxBufferSizeSet",
			cfg: map[string]interface{}{
				"line_delimiter":   "\n",
				"max_message_size": 66 * 1024,
			},
			expectedMessages: extraLargeMessages,
			messageSent:      strings.Join(extraLargeMessages, "\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan *info, len(test.expectedMessages))
			defer close(ch)
			to := func(message []byte, mt inputsource.NetworkMetadata) {
				ch <- &info{message: string(message), mt: mt}
			}
			path := filepath.Join(os.TempDir(), "test.sock")
			test.cfg["path"] = path
			cfg, _ := conf.NewConfigFrom(test.cfg)
			config := defaultConfig()
			err := cfg.Unpack(&config)
			if !assert.NoError(t, err) {
				return
			}

			server, err := New(logp.L(), &config, to)
			if !assert.NoError(t, err) {
				return
			}
			err = server.Start()
			if !assert.NoError(t, err) {
				return
			}
			defer server.Stop()

			conn, err := net.Dial("unix", path)
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
			}
		})
	}
}

func TestSocketOwnershipAndMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("changing socket ownership is only supported on non-windows")
		return
	}

	groups, err := os.Getgroups()
	require.NoError(t, err)

	if len(groups) <= 1 {
		t.Skip("no group that we can change to")
		return
	}

	group, err := user.LookupGroupId(strconv.Itoa(groups[1]))
	require.NoError(t, err)

	path := filepath.Join(os.TempDir(), "test.sock")
	cfg, _ := conf.NewConfigFrom(map[string]interface{}{
		"path":           path,
		"group":          group.Name,
		"mode":           "0740",
		"line_delimiter": "\n",
	})
	config := defaultConfig()
	err = cfg.Unpack(&config)
	require.NoError(t, err)

	server, err := New(logp.L(), &config, nil)
	require.NoError(t, err)
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	info, err := file.Lstat(path)
	require.NoError(t, err)
	require.NotEqual(t, 0, info.Mode()&os.ModeSocket)
	require.Equal(t, os.FileMode(0o740), info.Mode().Perm())
	gid, err := info.GID()
	require.NoError(t, err)
	require.Equal(t, group.Gid, strconv.Itoa(gid))
}

func TestSocketCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test is only supported on non-windows. See https://github.com/elastic/beats/issues/21757")
		return
	}
	path := filepath.Join(os.TempDir(), "test.sock")
	mockStaleSocket, err := net.Listen("unix", path)
	require.NoError(t, err)
	defer mockStaleSocket.Close()

	cfg, _ := conf.NewConfigFrom(map[string]interface{}{
		"path":           path,
		"line_delimiter": "\n",
	})
	config := defaultConfig()
	require.NoError(t, cfg.Unpack(&config))
	server, err := New(logp.L(), &config, nil)
	require.NoError(t, err)
	err = server.Start()
	require.NoError(t, err)
	server.Stop()
}

func TestSocketCleanupRefusal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping due to windows FileAttributes bug https://github.com/golang/go/issues/33357")
		return
	}
	path := filepath.Join(os.TempDir(), "test.sock")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	defer os.Remove(path)

	cfg, _ := conf.NewConfigFrom(map[string]interface{}{
		"path":           path,
		"line_delimiter": "\n",
	})
	config := defaultConfig()
	require.NoError(t, cfg.Unpack(&config))
	server, err := New(logp.L(), &config, nil)
	require.NoError(t, err)
	err = server.Start()
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to remove file at location")
}

func TestReceiveNewEventsConcurrently(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test is only supported on non-windows. See https://github.com/elastic/beats/issues/21757")
		return
	}

	for socketType := range socketTypes {
		if runtime.GOOS == "darwin" && socketType == "datagram" {
			t.Skip("test is only supported on linux. See https://github.com/elastic/beats/issues/22775")
			return
		}

		t.Run("socket_type "+socketType, func(t *testing.T) {
			workers := 1
			eventsCount := 100
			path := filepath.Join(os.TempDir(), "test.sock")
			ch := make(chan *info, eventsCount*workers)
			defer close(ch)
			to := func(message []byte, mt inputsource.NetworkMetadata) {
				ch <- &info{message: string(message), mt: mt}
			}
			cfg, err := conf.NewConfigFrom(map[string]interface{}{
				"path":           path,
				"line_delimiter": "\n",
				"socket_type":    socketType,
			})
			if !assert.NoError(t, err) {
				return
			}
			config := defaultConfig()
			err = cfg.Unpack(&config)
			if !assert.NoError(t, err) {
				return
			}

			server, err := New(logp.L(), &config, to)
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
				if socketType == "stream" {
					go sendOverUnixStream(t, path, samples)
				} else if socketType == "datagram" {
					go sendOverUnixDatagram(t, path, samples)
				}
			}

			var events []*info
			for len(events) < eventsCount*workers {
				select {
				case event := <-ch:
					events = append(events, event)
				default:
				}
			}
		})
	}
}

func sendOverUnixStream(t *testing.T, path string, samples []string) {
	conn, err := net.Dial("unix", path)
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()

	for _, sample := range samples {
		fmt.Fprintln(conn, sample)
	}
}

func sendOverUnixDatagram(t *testing.T, path string, samples []string) {
	conn, err := net.Dial("unixgram", path)
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	for _, sample := range samples {
		fmt.Fprintln(conn, sample)
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
		messages[i] = randomString(l) + "-" + strconv.Itoa(i)
	}
	return messages
}
