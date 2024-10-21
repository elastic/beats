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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type logger interface {
	// Logf logs the message for this runner.
	Logf(format string, args ...any)
}

type sshClient struct {
	ip       string
	username string
	auth     ssh.AuthMethod
	logger   logger
	c        *ssh.Client
}

// NewClient creates a new SSH client connection to the host.
func NewClient(ip string, username string, sshAuth ssh.AuthMethod, logger logger) SSHClient {
	return &sshClient{
		ip:       ip,
		username: username,
		auth:     sshAuth,
		logger:   logger,
	}
}

// Connect connects to the host.
func (s *sshClient) Connect(ctx context.Context) error {
	var lastErr error
	config := &ssh.ClientConfig{
		User:            s.username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // it's the tests framework test
		Auth:            []ssh.AuthMethod{s.auth},
		Timeout:         30 * time.Second,
	}
	addr := net.JoinHostPort(s.ip, "22")

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return fmt.Errorf("unable to resolve ssh address %q :%w", addr, err)
	}
	delay := 1 * time.Second
	for {
		if ctx.Err() != nil {
			if lastErr == nil {
				return ctx.Err()
			}
			return lastErr
		}
		if lastErr != nil {
			s.logger.Logf("ssh connect error: %q, will try again in %s", lastErr, delay)
			time.Sleep(delay)
			delay = 2 * delay

		}
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			lastErr = fmt.Errorf("error dialing tcp address %q :%w", addr, err)
			continue
		}
		err = conn.SetKeepAlive(true)
		if err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("error setting TCP keepalive for ssh to %q :%w", addr, err)
			continue
		}
		err = conn.SetKeepAlivePeriod(config.Timeout)
		if err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("error setting TCP keepalive period for ssh to %q :%w", addr, err)
			continue
		}
		sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			_ = conn.Close()
			lastErr = fmt.Errorf("error NewClientConn for ssh to %q :%w", addr, err)
			continue
		}
		s.c = ssh.NewClient(sshConn, chans, reqs)
		return nil
	}
}

// ConnectWithTimeout connects to the host with a timeout.
func (s *sshClient) ConnectWithTimeout(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Connect(ctx)
}

// Close closes the client.
func (s *sshClient) Close() error {
	if s.c != nil {
		err := s.c.Close()
		s.c = nil
		return err
	}
	return nil
}

// Reconnect disconnects and reconnected to the host.
func (s *sshClient) Reconnect(ctx context.Context) error {
	_ = s.Close()
	return s.Connect(ctx)
}

// ReconnectWithTimeout disconnects and reconnected to the host with a timeout.
func (s *sshClient) ReconnectWithTimeout(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Reconnect(ctx)
}

// NewSession opens a new Session for this host.
func (s *sshClient) NewSession() (*ssh.Session, error) {
	return s.c.NewSession()
}

// Exec runs a command on the host.
func (s *sshClient) Exec(ctx context.Context, cmd string, args []string, stdin io.Reader) ([]byte, []byte, error) {
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	var session *ssh.Session
	cmdArgs := []string{cmd}
	cmdArgs = append(cmdArgs, args...)
	cmdStr := strings.Join(cmdArgs, " ")
	session, err := s.NewSession()
	if err != nil {
		s.logger.Logf("new session failed: %q, trying reconnect", err)
		lErr := s.Reconnect(ctx)
		if lErr != nil {
			return nil, nil, fmt.Errorf("ssh reconnect failed: %w, after new session failed: %w", lErr, err)
		}
		session, lErr = s.NewSession()
		if lErr != nil {
			return nil, nil, fmt.Errorf("new session failed after reconnect: %w, original new session failure was: %w", lErr, err)
		}
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if stdin != nil {
		session.Stdin = stdin
	}
	err = session.Run(cmdStr)
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("could not run %q though SSH: %w",
			cmdStr, err)
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

// ExecWithRetry runs the command on loop waiting the interval between calls
func (s *sshClient) ExecWithRetry(ctx context.Context, cmd string, args []string, interval time.Duration) ([]byte, []byte, error) {
	var lastErr error
	var lastStdout []byte
	var lastStderr []byte
	for {
		// the length of time for running the command is not blocked on the interval
		// don't create a new context with the interval as its timeout
		stdout, stderr, err := s.Exec(ctx, cmd, args, nil)
		if err == nil {
			return stdout, stderr, nil
		}
		s.logger.Logf("ssh exec error: %q, will try again in %s", err, interval)
		lastErr = err
		lastStdout = stdout
		lastStderr = stderr

		// wait for the interval or ctx to be cancelled
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return lastStdout, lastStderr, lastErr
			}
			return nil, nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// Copy copies the filePath to the host at dest.
func (s *sshClient) Copy(filePath string, dest string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	fs, err := f.Stat()
	if err != nil {
		return err
	}

	session, err := s.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	w, err := session.StdinPipe()
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("scp -t %s", dest)
	if err := session.Start(cmd); err != nil {
		_ = w.Close()
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- session.Wait()
	}()

	_, err = fmt.Fprintf(w, "C%#o %d %s\n", fs.Mode().Perm(), fs.Size(), dest)
	if err != nil {
		_ = w.Close()
		<-errCh
		return err
	}
	_, err = io.Copy(w, f)
	if err != nil {
		_ = w.Close()
		<-errCh
		return err
	}
	_, _ = fmt.Fprint(w, "\x00")
	_ = w.Close()
	return <-errCh
}

// GetFileContents returns the file content.
func (s *sshClient) GetFileContents(ctx context.Context, filename string, opts ...FileContentsOpt) ([]byte, error) {
	var stdout bytes.Buffer
	err := s.GetFileContentsOutput(ctx, filename, &stdout, opts...)
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

// GetFileContentsOutput returns the file content writing into output.
func (s *sshClient) GetFileContentsOutput(ctx context.Context, filename string, output io.Writer, opts ...FileContentsOpt) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	var fco fileContentsOpts
	fco.command = "cat"
	for _, opt := range opts {
		opt(&fco)
	}

	session, err := s.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = output
	err = session.Run(fmt.Sprintf("%s %s", fco.command, filename))
	if err != nil {
		return err
	}
	return nil
}
