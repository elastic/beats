// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/authority"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

type mockStore struct {
	Err     error
	Called  bool
	Content []byte
}

func (m *mockStore) Save(in io.Reader) error {
	m.Called = true
	if m.Err != nil {
		return m.Err
	}

	buf := new(bytes.Buffer)
	io.Copy(buf, in)
	m.Content = buf.Bytes()
	return nil
}

func TestEnroll(t *testing.T) {
	log, _ := logger.New("tst", false)

	t.Run("fail to save is propagated", withTLSServer(
		func(t *testing.T) *http.ServeMux {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/fleet/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`
{
    "action": "created",
    "item": {
       "id": "a9328860-ec54-11e9-93c4-d72ab8a69391",
        "active": true,
        "policy_id": "69f3f5a0-ec52-11e9-93c4-d72ab8a69391",
        "type": "PERMANENT",
        "enrolled_at": "2019-10-11T18:26:37.158Z",
        "user_provided_metadata": {
						"custom": "customize"
				},
        "local_metadata": {
            "platform": "linux",
            "version": "8.0.0"
        },
        "actions": [],
        "access_api_key": "my-access-token"
    }
}`))
			})
			return mux
		}, func(t *testing.T, caBytes []byte, host string) {
			caFile, err := bytesToTMPFile(caBytes)
			require.NoError(t, err)
			defer os.Remove(caFile)

			url := "https://" + host
			store := &mockStore{Err: errors.New("fail to save")}
			cmd, err := newEnrollCmdWithStore(
				log,
				&enrollCmdOption{
					URL:                  url,
					CAs:                  []string{caFile},
					EnrollAPIKey:         "my-enrollment-token",
					UserProvidedMetadata: map[string]interface{}{"custom": "customize"},
				},
				"",
				store,
			)
			require.NoError(t, err)

			streams, _, _, _ := cli.NewTestingIOStreams()
			err = cmd.Execute(context.Background(), streams)
			require.Error(t, err)
		},
	))

	t.Run("successfully enroll with TLS and save access api key in the store", withTLSServer(
		func(t *testing.T) *http.ServeMux {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/fleet/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`
{
    "action": "created",
    "item": {
       "id": "a9328860-ec54-11e9-93c4-d72ab8a69391",
        "active": true,
        "policy_id": "69f3f5a0-ec52-11e9-93c4-d72ab8a69391",
        "type": "PERMANENT",
        "enrolled_at": "2019-10-11T18:26:37.158Z",
        "user_provided_metadata": {
						"custom": "customize"
				},
        "local_metadata": {
            "platform": "linux",
            "version": "8.0.0"
        },
        "actions": [],
        "access_api_key": "my-access-api-key"
    }
}`))
			})
			return mux
		}, func(t *testing.T, caBytes []byte, host string) {
			caFile, err := bytesToTMPFile(caBytes)
			require.NoError(t, err)
			defer os.Remove(caFile)

			url := "https://" + host
			store := &mockStore{}
			cmd, err := newEnrollCmdWithStore(
				log,
				&enrollCmdOption{
					URL:                  url,
					CAs:                  []string{caFile},
					EnrollAPIKey:         "my-enrollment-api-key",
					UserProvidedMetadata: map[string]interface{}{"custom": "customize"},
				},
				"",
				store,
			)
			require.NoError(t, err)

			streams, _, _, _ := cli.NewTestingIOStreams()
			err = cmd.Execute(context.Background(), streams)
			require.NoError(t, err)

			config, err := readConfig(store.Content)

			require.NoError(t, err)
			require.Equal(t, "my-access-api-key", config.AccessAPIKey)
			require.Equal(t, host, config.Client.Host)
		},
	))

	t.Run("successfully enroll when a slash is defined at the end of host", withServer(
		func(t *testing.T) *http.ServeMux {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/fleet/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`
{
    "action": "created",
    "item": {
        "id": "a9328860-ec54-11e9-93c4-d72ab8a69391",
        "active": true,
        "policy_id": "69f3f5a0-ec52-11e9-93c4-d72ab8a69391",
        "type": "PERMANENT",
        "enrolled_at": "2019-10-11T18:26:37.158Z",
        "user_provided_metadata": {
						"custom": "customize"
				},
        "local_metadata": {
            "platform": "linux",
            "version": "8.0.0"
        },
        "actions": [],
        "access_api_key": "my-access-api-key"
    }
}`))
			})
			return mux
		}, func(t *testing.T, host string) {
			url := "http://" + host + "/"
			store := &mockStore{}
			cmd, err := newEnrollCmdWithStore(
				log,
				&enrollCmdOption{
					URL:                  url,
					CAs:                  []string{},
					EnrollAPIKey:         "my-enrollment-api-key",
					Insecure:             true,
					UserProvidedMetadata: map[string]interface{}{"custom": "customize"},
				},
				"",
				store,
			)
			require.NoError(t, err)

			streams, _, _, _ := cli.NewTestingIOStreams()
			err = cmd.Execute(context.Background(), streams)
			require.NoError(t, err)

			require.True(t, store.Called)

			config, err := readConfig(store.Content)

			require.NoError(t, err)
			require.Equal(t, "my-access-api-key", config.AccessAPIKey)
			require.Equal(t, host, config.Client.Host)
		},
	))

	t.Run("successfully enroll without TLS and save access api key in the store", withServer(
		func(t *testing.T) *http.ServeMux {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/fleet/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`
{
    "action": "created",
    "item": {
        "id": "a9328860-ec54-11e9-93c4-d72ab8a69391",
        "active": true,
        "policy_id": "69f3f5a0-ec52-11e9-93c4-d72ab8a69391",
        "type": "PERMANENT",
        "enrolled_at": "2019-10-11T18:26:37.158Z",
        "user_provided_metadata": {
						"custom": "customize"
				},
        "local_metadata": {
            "platform": "linux",
            "version": "8.0.0"
        },
        "actions": [],
        "access_api_key": "my-access-api-key"
    }
}`))
			})
			return mux
		}, func(t *testing.T, host string) {
			url := "http://" + host
			store := &mockStore{}
			cmd, err := newEnrollCmdWithStore(
				log,
				&enrollCmdOption{
					URL:                  url,
					CAs:                  []string{},
					EnrollAPIKey:         "my-enrollment-api-key",
					Insecure:             true,
					UserProvidedMetadata: map[string]interface{}{"custom": "customize"},
				},
				"",
				store,
			)
			require.NoError(t, err)

			streams, _, _, _ := cli.NewTestingIOStreams()
			err = cmd.Execute(context.Background(), streams)
			require.NoError(t, err)

			require.True(t, store.Called)

			config, err := readConfig(store.Content)

			require.NoError(t, err)
			require.Equal(t, "my-access-api-key", config.AccessAPIKey)
			require.Equal(t, host, config.Client.Host)
		},
	))

	t.Run("fail to enroll without TLS", withServer(
		func(t *testing.T) *http.ServeMux {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/fleet/agents/enroll", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`
{
		"statusCode": 500,
		"error": "Internal Server Error"
}`))
			})
			return mux
		}, func(t *testing.T, host string) {
			url := "http://" + host
			store := &mockStore{}
			cmd, err := newEnrollCmdWithStore(
				log,
				&enrollCmdOption{
					URL:                  url,
					CAs:                  []string{},
					EnrollAPIKey:         "my-enrollment-token",
					Insecure:             true,
					UserProvidedMetadata: map[string]interface{}{"custom": "customize"},
				},
				"",
				store,
			)
			require.NoError(t, err)

			streams, _, _, _ := cli.NewTestingIOStreams()
			err = cmd.Execute(context.Background(), streams)
			require.Error(t, err)
			require.False(t, store.Called)
		},
	))
}

func withServer(
	m func(t *testing.T) *http.ServeMux,
	test func(t *testing.T, host string),
) func(t *testing.T) {
	return func(t *testing.T) {
		s := httptest.NewServer(m(t))
		defer s.Close()
		test(t, s.Listener.Addr().String())
	}
}

func withTLSServer(
	m func(t *testing.T) *http.ServeMux,
	test func(t *testing.T, caBytes []byte, host string),
) func(t *testing.T) {
	return func(t *testing.T) {
		ca, err := authority.NewCA()
		require.NoError(t, err)
		pair, err := ca.GeneratePair()
		require.NoError(t, err)

		serverCert, err := tls.X509KeyPair(pair.Crt, pair.Key)
		require.NoError(t, err)

		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		s := http.Server{
			Handler: m(t),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{serverCert},
			},
		}

		// Uses the X509KeyPair pair defined in the TLSConfig struct instead of file on disk.
		go s.ServeTLS(listener, "", "")

		test(t, ca.Crt(), "localhost:"+strconv.Itoa(port))
	}
}

func bytesToTMPFile(b []byte) (string, error) {
	f, err := ioutil.TempFile("", "prefix")
	if err != nil {
		return "", err
	}
	f.Write(b)
	if err := f.Close(); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func readConfig(raw []byte) (*configuration.FleetAgentConfig, error) {
	r := bytes.NewReader(raw)
	config, err := config.NewConfigFrom(r)
	if err != nil {
		return nil, err
	}

	cfg := configuration.DefaultConfiguration()
	if err := config.Unpack(cfg); err != nil {
		return nil, err
	}
	return cfg.Fleet, nil
}
