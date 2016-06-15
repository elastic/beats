package transport

import (
	"expvar"
	"net"
)

type IOStats struct {
	Read, Write, ReadErrors, WriteErrors *expvar.Int
}

type statsConn struct {
	net.Conn
	stats *IOStats
}

func StatsDialer(d Dialer, s *IOStats) Dialer {
	return ConnWrapper(d, func(c net.Conn) net.Conn {
		return &statsConn{c, s}
	})
}

func (s *statsConn) Read(b []byte) (int, error) {
	n, err := s.Conn.Read(b)
	if err != nil {
		s.stats.ReadErrors.Add(1)
	}
	s.stats.Read.Add(int64(n))
	return n, err
}

func (s *statsConn) Write(b []byte) (int, error) {
	n, err := s.Conn.Write(b)
	if err != nil {
		s.stats.WriteErrors.Add(1)
	}
	s.stats.Write.Add(int64(n))
	return n, err
}
