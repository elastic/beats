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

//go:build windows
// +build windows

package npipe

import (
	"context"
	"net"
	"os/user"
	"strings"

	winio "github.com/Microsoft/go-winio"
	"github.com/pkg/errors"
)

// NewListener creates a new Listener receiving events over a named pipe.
func NewListener(name, sd string) (net.Listener, error) {
	c := &winio.PipeConfig{
		SecurityDescriptor: sd,
	}

	l, err := winio.ListenPipe(name, c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to listen on the named pipe %s", name)
	}

	return l, nil
}

// TransformString takes an input type name defined as a URI like `npipe:///hello` and transform it into
// `\\.\pipe\hello`
func TransformString(name string) string {
	if strings.HasPrefix(name, "npipe:///") {
		path := strings.TrimPrefix(name, "npipe:///")
		return `\\.\pipe\` + path
	}

	if strings.HasPrefix(name, `\\.\pipe\`) {
		return name
	}

	return name
}

// DialContext create a Dial to be use with an http.Client to connect to a pipe.
func DialContext(npipe string) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		return winio.DialPipeContext(ctx, npipe)
	}
}

// Dial create a Dial to be use with an http.Client to connect to a pipe.
func Dial(npipe string) func(string, string) (net.Conn, error) {
	return func(_, _ string) (net.Conn, error) {
		return winio.DialPipe(npipe, nil)
	}
}

// DefaultSD returns a default SecurityDescriptor which is the minimal required permissions to be
// able to write to the named pipe. The security descriptor is returned in SDDL format.
//
// Docs: https://docs.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-string-format
func DefaultSD(forUser string) (string, error) {
	var u *user.User
	var err error
	// No user configured we fallback to the current running user.
	if len(forUser) == 0 {
		u, err = user.Current()
		if err != nil {
			return "", errors.Wrap(err, "failed to retrieve the current user")
		}
	} else {
		u, err = user.Lookup(forUser)
		if err != nil {
			return "", errors.Wrapf(err, "failed to retrieve the user %s", forUser)
		}
	}

	// Named pipe security and access rights.
	// We create the pipe and the specific users should only be able to write to it.
	// See docs: https://docs.microsoft.com/en-us/windows/win32/ipc/named-pipe-security-and-access-rights
	// String definition: https://docs.microsoft.com/en-us/windows/win32/secauthz/ace-strings
	// Give generic read/write access to the specified user.
	descriptor := "D:P(A;;GA;;;" + u.Uid + ")"
	if u.Username == "NT AUTHORITY\\SYSTEM" {
		// running as SYSTEM, include Administrators group so Administrators can talk over
		// the named pipe to the running Elastic Agent system process
		// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
		descriptor += "(A;;GA;;;S-1-5-32-544)" // Administrators group
	}
	return descriptor, nil
}
