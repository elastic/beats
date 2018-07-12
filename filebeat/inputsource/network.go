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
