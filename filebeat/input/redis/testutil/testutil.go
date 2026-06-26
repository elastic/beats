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

package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	rd "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	// SlowlogCommand is the Redis command used to generate slowlog events in tests.
	SlowlogCommand = "EVAL"

	defaultHost    = "localhost"
	defaultPort    = "6379"
	defaultTLSPort = "6380"
)

// HostPort returns the plain Redis host:port used by integration tests.
func HostPort() string {
	return fmt.Sprintf("%s:%s",
		getOrDefault(os.Getenv("REDIS_HOST"), defaultHost),
		getOrDefault(os.Getenv("REDIS_PORT"), defaultPort),
	)
}

// TLSHostPort returns the TLS Redis host:port used by authentication tests.
func TLSHostPort() string {
	return fmt.Sprintf("%s:%s",
		getOrDefault(os.Getenv("REDIS_HOST"), defaultHost),
		getOrDefault(os.Getenv("REDIS_TLS_PORT"), defaultTLSPort),
	)
}

// CreateClient creates a plain Redis connection pool for integration tests.
func CreateClient(t *testing.T) *rd.Pool {
	t.Helper()
	return newPool(t, HostPort(), "", false)
}

// CreateTLSClient creates a TLS-enabled Redis connection pool for authentication tests.
func CreateTLSClient(t *testing.T, password string) *rd.Pool {
	t.Helper()
	return newPool(t, TLSHostPort(), password, true)
}

// ConfigureSlowlog configures Redis to log all commands to the slowlog.
func ConfigureSlowlog(t *testing.T, pool *rd.Pool) {
	t.Helper()

	conn := pool.Get()
	defer func() {
		require.NoError(t, conn.Close())
	}()

	_, err := conn.Do("CONFIG", "SET", "slowlog-log-slower-than", 0)
	require.NoError(t, err, "failed to configure redis slowlog threshold")
}

// EmitInputData periodically runs a slow Redis script to generate slowlog entries.
func EmitInputData(t *testing.T, ctx context.Context, pool *rd.Pool) {
	t.Helper()

	script := "local i = 0 for j=1,500000 do i = i + j end return i"

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		conn := pool.Get()
		defer func() {
			err := conn.Close()
			require.NoError(t, err)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := conn.Do("EVAL", script, 0)
				require.NoError(t, err)
			}
		}
	}()
}

func newPool(t *testing.T, hostPort, password string, useTLS bool) *rd.Pool {
	t.Helper()

	idleTimeout := 60 * time.Second

	var dialOptions []rd.DialOption
	if password != "" {
		dialOptions = append(dialOptions, rd.DialPassword(password))
	}
	dialOptions = append(dialOptions,
		rd.DialConnectTimeout(idleTimeout),
		rd.DialReadTimeout(idleTimeout),
		rd.DialWriteTimeout(idleTimeout),
	)

	if useTLS {
		enabled := true
		certs := absCertPaths(t)

		tlsConfig, err := tlscommon.LoadTLSConfig(&tlscommon.Config{
			Enabled: &enabled,
			CAs:     []string{certs.CA},
			Certificate: tlscommon.CertificateConfig{
				Certificate: certs.Cert,
				Key:         certs.Key,
			},
		}, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		dialOptions = append(dialOptions,
			rd.DialUseTLS(true),
			rd.DialTLSConfig(tlsConfig.ToConfig()),
		)
	}

	return &rd.Pool{
		MaxActive:   10,
		MaxIdle:     10,
		Wait:        true,
		IdleTimeout: idleTimeout,
		Dial: func() (rd.Conn, error) {
			return rd.Dial("tcp", hostPort, dialOptions...)
		},
		TestOnBorrow: func(c rd.Conn, borrowedAt time.Time) error {
			if time.Since(borrowedAt) < idleTimeout {
				return nil
			}

			_, err := c.Do("PING")
			return err
		},
	}
}

type certPaths struct {
	CA   string
	Cert string
	Key  string
}

func absCertPaths(t *testing.T) certPaths {
	t.Helper()

	dir := certDir()
	paths := certPaths{
		CA:   filepath.Join(dir, "root-ca.pem"),
		Cert: filepath.Join(dir, "server-cert.pem"),
		Key:  filepath.Join(dir, "server-key.pem"),
	}

	var err error
	paths.CA, err = filepath.Abs(paths.CA)
	require.NoError(t, err)
	paths.Cert, err = filepath.Abs(paths.Cert)
	require.NoError(t, err)
	paths.Key, err = filepath.Abs(paths.Key)
	require.NoError(t, err)

	return paths
}

func certDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to determine redis cert directory")
	}
	return filepath.Join(filepath.Dir(file), "..", "_meta", "certs")
}

func getOrDefault(s, defaultString string) string {
	if s == "" {
		return defaultString
	}
	return s
}
