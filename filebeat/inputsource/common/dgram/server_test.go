package dgram

import (
	"errors"
	"net"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestListenerRunReturnsErrorWhenConnectionFails(t *testing.T) {
	l := NewListener(
		"test family",
		"not used",
		nil,
		func() (net.PacketConn, error) {
			return nil, errors.New("some error")
		},
		&ListenerConfig{},
		logp.NewNopLogger(),
	)

	if err := l.Run(t.Context()); err == nil {
		t.Fatal("expecting an error from Listener.Run")
	}
}
