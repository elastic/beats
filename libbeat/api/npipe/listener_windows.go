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

package npipe

import (
	"context"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/Microsoft/go-winio"
)

// ntAuthoritySystemSID is a well-known SID used by the NT AUTHORITY\SYSTEM account.
const ntAuthoritySystemSID = "S-1-5-18"

// NewListener creates a new Listener receiving events over a named pipe.
func NewListener(name, sd string) (net.Listener, error) {
	c := &winio.PipeConfig{
		SecurityDescriptor: sd,
	}

	l, err := winio.ListenPipe(name, c)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on the named pipe %s: %w", name, err)
	}

	return l, nil
}

// TransformString takes an input type name defined as a URI like
// `npipe:///hello` and transforms it into // `\\.\pipe\hello`
func TransformString(name string) string {
	if strings.HasPrefix(name, "npipe:///") {
		path := strings.TrimPrefix(name, "npipe:///")
		return `\\.\pipe\` + path
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

// DefaultSD returns a default SecurityDescriptor that specifies the minimal required permissions to be
// able to write to the named pipe. The security descriptor is returned in SDDL format.
//
// Docs: https://docs.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-string-format
func DefaultSD(forUser string) (string, error) {
	sid, err := lookupSID(forUser)
	if err != nil {
		return "", err
	}

	// Named pipe security and access rights.
	// We create the pipe and the specific users should only be able to write to it.
	// See docs: https://docs.microsoft.com/en-us/windows/win32/ipc/named-pipe-security-and-access-rights
	// String definition: https://docs.microsoft.com/en-us/windows/win32/secauthz/ace-strings
	// Give generic read/write access to the specified user.
	descriptor := "D:P(A;;GA;;;" + sid + ")"
	if sid == ntAuthoritySystemSID {
		// running as SYSTEM, include Administrators group so Administrators can talk over
		// the named pipe to the running Elastic Agent system process
		// https://support.microsoft.com/en-us/help/243330/well-known-security-identifiers-in-windows-operating-systems
		descriptor += "(A;;GA;;;S-1-5-32-544)" // Administrators group
	}
	return descriptor, nil
}

// lookupSID returns the SID of the specified username. If username is empty the
// SID of the current user is returned.
func lookupSID(username string) (string, error) {
	if username == "" {
		sid, err := currentUserSID()
		if err != nil {
			return "", fmt.Errorf("failed to lookup the SID of current user: %w", err)
		}
		return sid, nil
	}

	sid, _, _, err := syscall.LookupSID("", username)
	if err != nil {
		return "", fmt.Errorf("failed to lookup the SID for user %q: %w", username, err)
	}
	sidString, err := sid.String()
	if err != nil {
		return "", fmt.Errorf("failed to convert the SID for user %q to string: %w", username, err)
	}
	return sidString, nil
}

// currentUserSID returns the SID of the user running the current process.
func currentUserSID() (string, error) {
	t, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer t.Close()

	u, err := t.GetTokenUser()
	if err != nil {
		return "", err
	}

	sid, err := u.User.Sid.String()
	if err != nil {
		return "", err
	}

	return sid, nil
}
