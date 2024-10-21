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

package ssh

import (
	"context"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient is a *ssh.Client that provides a nice interface to work with.
type SSHClient interface {
	// Connect connects to the host.
	Connect(ctx context.Context) error

	// ConnectWithTimeout connects to the host with a timeout.
	ConnectWithTimeout(ctx context.Context, timeout time.Duration) error

	// Close closes the client.
	Close() error

	// Reconnect disconnects and reconnected to the host.
	Reconnect(ctx context.Context) error

	// ReconnectWithTimeout disconnects and reconnected to the host with a timeout.
	ReconnectWithTimeout(ctx context.Context, timeout time.Duration) error

	// NewSession opens a new Session for this host.
	NewSession() (*ssh.Session, error)

	// Exec runs a command on the host.
	Exec(ctx context.Context, cmd string, args []string, stdin io.Reader) ([]byte, []byte, error)

	// ExecWithRetry runs the command on loop waiting the interval between calls
	ExecWithRetry(ctx context.Context, cmd string, args []string, interval time.Duration) ([]byte, []byte, error)

	// Copy copies the filePath to the host at dest.
	Copy(filePath string, dest string) error

	// GetFileContents returns the file content.
	GetFileContents(ctx context.Context, filename string, opts ...FileContentsOpt) ([]byte, error)

	// GetFileContentsOutput returns the file content writing to output.
	GetFileContentsOutput(ctx context.Context, filename string, output io.Writer, opts ...FileContentsOpt) error
}
