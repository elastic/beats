package udp

import (
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const maxMessageSize = 20
const timeout = time.Second * 15

type info struct {
	message []byte
	addr    net.Addr
}

func TestReceiveEventFromUDP(t *testing.T) {
	if runtime.GOOS == "window" {
		t.Skipf("Skipped on windows")
	}

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
	host := "0.0.0.0:10000"
	config := &Config{Host: host, MaxMessageSize: maxMessageSize, Timeout: timeout}
	fn := func(message []byte, addr net.Addr) {
		ch <- info{message: message, addr: addr}
	}
	s := New(config, fn)
	go func() {
		s.Start()
	}()
	defer s.Stop()

	// Give it some time to accept UDP connection.
	for !s.IsRunning() {
		time.Sleep(time.Second * 2)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, err := net.Dial("udp", host)
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
			assert.NotNil(t, info.addr)
		})
	}
}
