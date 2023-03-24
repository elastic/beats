package tracer

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

type Tracer interface {
	Write(string) error
	Close() error
}

type SockTracer struct {
	path string
	sock net.Conn
}

func NewSockTracer(path string, wait time.Duration) (st SockTracer, err error) {
	st.path = path

	started := time.Now()
	for {
		if time.Now().Sub(started) > wait {
			return st, fmt.Errorf("wait time for sock trace exceeded: %s", wait)
		}
		if _, err := os.Stat(st.path); err == nil {
			logp.L().Info("socktracer found file for unix socket: %s, will attemp to connect")
			fmt.Printf("WHUT %s\n", st.path)
			break
		} else {
			delay := time.Millisecond * 250
			logp.L().Info("socktracer could not find file for unix socket at: %s, will retry in %s", delay)
			fmt.Printf("HUH\n")
			time.Sleep(delay)
		}
	}

	st.sock, err = net.Dial("unix", path)
	if err != nil {
		return SockTracer{}, fmt.Errorf("could not create sock tracer at %s: %w", path, err)
	}

	return st, nil
}

func (st SockTracer) Write(message string) error {
	// Note, we don't need to worry about partial writes here: https://pkg.go.dev/io?utm_source=godoc#Writer
	// an error will be returned here, which shouldn't really happen with unix sockets only
	_, err := st.sock.Write([]byte(message))
	return err
}

func (st SockTracer) Close() error {
	return st.sock.Close()
}

type NoopTracer struct{}

func NewNoopTracer() NoopTracer {
	return NoopTracer{}
}

func (nt NoopTracer) Write(message string) error {
	return nil
}

func (nt NoopTracer) Close() error {
	return nil
}
