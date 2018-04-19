package udp

import (
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/inputsource"
)

const maxMessageSize = 20
const timeout = time.Second * 15

type info struct {
	message []byte
	mt      inputsource.NetworkMetadata
}

func TestReceiveEventFromUDP(t *testing.T) {
	tests := []struct {
		name     string
		message  []byte
		expected []byte
	}{
		{
			name:     "Sending a message under the MaxMessageSize limit",
			message:  []byte("Hello world"),
			expected: []byte("Hello world"),
		},
		{
			name:     "Sending a message over the MaxMessageSize limit will truncate the message",
			message:  []byte("Hello world not so nice"),
			expected: []byte("Hello world not so n"),
		},
	}

	ch := make(chan info)
	host := "localhost:0"
	config := &Config{Host: host, MaxMessageSize: maxMessageSize, Timeout: timeout}
	fn := func(message []byte, metadata inputsource.NetworkMetadata) {
		ch <- info{message: message, mt: metadata}
	}
	s := New(config, fn)
	err := s.Start()
	if !assert.NoError(t, err) {
		return
	}
	defer s.Stop()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, err := net.Dial("udp", s.Listener.LocalAddr().String())
			if !assert.NoError(t, err) {
				return
			}
			defer conn.Close()

			_, err = conn.Write(test.message)
			if !assert.NoError(t, err) {
				return
			}
			info := <-ch
			assert.Equal(t, test.expected, info.message)
			if runtime.GOOS == "windows" {
				if len(test.expected) < len(test.message) {
					assert.Nil(t, info.mt.RemoteAddr)
					assert.True(t, info.mt.Truncated)
				} else {
					assert.NotNil(t, info.mt.RemoteAddr)
					assert.False(t, info.mt.Truncated)
				}
			} else {
				assert.NotNil(t, info.mt.RemoteAddr)
				assert.False(t, info.mt.Truncated)
			}
		})
	}
}
