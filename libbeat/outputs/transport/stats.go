package transport

import (
	"net"
)

type IOStatser interface {
	WriteError()
	WriteBytes(int)

	ReadError()
	ReadBytes(int)
}

type statsConn struct {
	net.Conn
	stats IOStatser
}

func StatsDialer(d Dialer, s IOStatser) Dialer {
	return ConnWrapper(d, func(c net.Conn) net.Conn {
		return &statsConn{c, s}
	})
}

func (s *statsConn) Read(b []byte) (int, error) {
	n, err := s.Conn.Read(b)
	if err != nil {
		s.stats.ReadError()
	}
	s.stats.ReadBytes(n)
	return n, err
}

func (s *statsConn) Write(b []byte) (int, error) {
	n, err := s.Conn.Write(b)
	if err != nil {
		s.stats.WriteError()
	}
	s.stats.WriteBytes(n)
	return n, err
}
