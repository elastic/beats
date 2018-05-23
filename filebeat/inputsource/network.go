package inputsource

import (
	"net"
)

// Network interface implemented by TCP and UDP input source.
type Network interface {
	Start() error
	Stop()
}

// NetworkMetadata defines common information that we can retrieve from a remote connection.
type NetworkMetadata struct {
	RemoteAddr net.Addr
	Truncated  bool
	TLS        *TLSMetadata
}

// TLSMetadata defines information about the current SSL connection.
type TLSMetadata struct {
	TLSVersion       string
	CipherSuite      string
	ServerName       string
	PeerCertificates []string
}

// NetworkFunc defines callback executed when a new event is received from a network source.
type NetworkFunc = func(data []byte, metadata NetworkMetadata)
