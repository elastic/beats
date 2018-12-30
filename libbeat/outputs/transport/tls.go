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

package transport

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/testing"
)

// TLSConfig is the interface used to configure a tcp client or server from a `Config`
type TLSConfig = tlscommon.TLSConfig

// TLSVersion type for TLS version.
type TLSVersion = tlscommon.TLSVersion

// Define all the possible TLS version.
const (
	TLSVersionSSL30 = tlscommon.TLSVersionSSL30
	TLSVersion10    = tlscommon.TLSVersion10
	TLSVersion11    = tlscommon.TLSVersion11
	TLSVersion12    = tlscommon.TLSVersion12
)

// Constants of the supported verification mode.
const (
	VerifyFull = tlscommon.VerifyFull
	VerifyNone = tlscommon.VerifyNone
)

func TLSDialer(forward Dialer, config *TLSConfig, timeout time.Duration) (Dialer, error) {
	return TestTLSDialer(testing.NullDriver, forward, config, timeout)
}

func TestTLSDialer(
	d testing.Driver,
	forward Dialer,
	config *TLSConfig,
	timeout time.Duration,
) (Dialer, error) {
	var lastTLSConfig *tls.Config
	var lastNetwork string
	var lastAddress string
	var m sync.Mutex

	return DialerFunc(func(network, address string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6":
		default:
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		var tlsConfig *tls.Config
		m.Lock()
		if network == lastNetwork && address == lastAddress {
			tlsConfig = lastTLSConfig
		}
		if tlsConfig == nil {
			tlsConfig = config.BuildModuleConfig(host)
			lastNetwork = network
			lastAddress = address
			lastTLSConfig = tlsConfig
		}
		m.Unlock()

		return tlsDialWith(d, forward, network, address, timeout, tlsConfig, config)
	}), nil
}

func tlsDialWith(
	d testing.Driver,
	dialer Dialer,
	network, address string,
	timeout time.Duration,
	tlsConfig *tls.Config,
	config *TLSConfig,
) (net.Conn, error) {
	socket, err := dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}

	conn := tls.Client(socket, tlsConfig)

	withTimeout := timeout > 0
	if withTimeout {
		if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
			d.Fatal("timeout", err)
			_ = conn.Close()
			return nil, err
		}
	}

	if tlsConfig.InsecureSkipVerify {
		d.Warn("security", "server's certificate chain verification is disabled")
	} else {
		d.Info("security", "server's certificate chain verification is enabled")
	}

	err = conn.Handshake()
	d.Fatal("handshake", err)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// remove timeout if handshake was subject to timeout:
	if withTimeout {
		conn.SetDeadline(time.Time{})
	}

	if err := postVerifyTLSConnection(d, conn, config); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func postVerifyTLSConnection(d testing.Driver, conn *tls.Conn, config *TLSConfig) error {
	st := conn.ConnectionState()

	if !st.HandshakeComplete {
		err := errors.New("incomplete handshake")
		d.Fatal("incomplete handshake", err)
		return err
	}

	d.Info("TLS version", fmt.Sprintf("%v", TLSVersion(st.Version)))

	// no more checks if no extra configs available
	if config == nil {
		return nil
	}

	versions := config.Versions
	if versions == nil {
		versions = tlscommon.TLSDefaultVersions
	}
	versionOK := false
	for _, version := range versions {
		versionOK = versionOK || st.Version == uint16(version)
	}
	if !versionOK {
		err := fmt.Errorf("tls version %v not configured", TLSVersion(st.Version))
		d.Fatal("TLS version", err)
		return err
	}

	return nil
}
