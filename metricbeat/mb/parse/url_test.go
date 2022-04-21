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

package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/helper/dialer"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

const (
	rawURL_unix  = "http+unix:///var/lib/docker.sock"
	rawURL_npipe = "http+npipe://./pipe/custom"
)

type expected struct {
	URI          string
	SanitizedURI string
	Host         string
	User         string
	Password     string
}

func assertTest(t *testing.T, hostData mb.HostData, err error, exp *expected) {
	assert.NoError(t, err)
	assert.Equal(t, exp.URI, hostData.URI)
	assert.Equal(t, exp.SanitizedURI, hostData.SanitizedURI)
	assert.Equal(t, exp.Host, hostData.Host)
	assert.Equal(t, exp.User, hostData.User)
	assert.Equal(t, exp.Password, hostData.Password)
}

func TestParseURL(t *testing.T) {
	t.Run("http", func(t *testing.T) {
		rawURL := "https://admin:secret@127.0.0.1:8080?hello=world"
		hostData, err := ParseURL(rawURL, "http", "root", "passwd", "/test", "auto")
		if assert.NoError(t, err) {
			assert.Equal(t, "https://admin:secret@127.0.0.1:8080/test?auto=&hello=world", hostData.URI)
			assert.Equal(t, "https://127.0.0.1:8080/test?auto=&hello=world", hostData.SanitizedURI)
			assert.Equal(t, "127.0.0.1:8080", hostData.Host)
			assert.Equal(t, "admin", hostData.User)
			assert.Equal(t, "secret", hostData.Password)
		}
	})

	t.Run("http+ipv6", func(t *testing.T) {
		rawURL := "[2001:db8:85a3:0:0:8a2e:370:7334]:8080"
		hostData, err := ParseURL(rawURL, "https", "", "", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "https://[2001:db8:85a3:0:0:8a2e:370:7334]:8080", hostData.URI)
		}
	})

	t.Run("unix", func(t *testing.T) {
		rawURL := "unix:///var/lib/docker.sock"
		hostData, err := ParseURL(rawURL, "tcp", "", "", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "unix:///var/lib/docker.sock", hostData.URI)
			assert.Equal(t, "unix:///var/lib/docker.sock", hostData.SanitizedURI)
			assert.Equal(t, "/var/lib/docker.sock", hostData.Host)
			assert.Equal(t, "", hostData.User)
			assert.Equal(t, "", hostData.Password)
		}
	})

	t.Run("http+unix at root", func(t *testing.T) {
		hostData, err := ParseURL(rawURL_unix, "http", "", "", "", "")
		if assert.NoError(t, err) {
			transport, ok := hostData.Transport.(*dialer.UnixDialerBuilder)
			assert.True(t, ok)
			assert.Equal(t, "/var/lib/docker.sock", transport.Path)
		}
		exp := &expected{
			URI:          "http://unix",
			SanitizedURI: "http://unix",
			Host:         "unix",
			User:         "",
			Password:     "",
		}
		assertTest(t, hostData, err, exp)
	})

	t.Run("http+unix with path", func(t *testing.T) {
		hostData, err := ParseURL(rawURL_unix, "http", "", "", "apath", "")
		if assert.NoError(t, err) {
			transport, ok := hostData.Transport.(*dialer.UnixDialerBuilder)
			assert.True(t, ok)
			assert.Equal(t, "/var/lib/docker.sock", transport.Path)
		}
		exp := &expected{
			URI:          "http://unix/apath",
			SanitizedURI: "http://unix/apath",
			Host:         "unix",
			User:         "",
			Password:     "",
		}
		assertTest(t, hostData, err, exp)
	})

	t.Run("http+npipe at root", func(t *testing.T) {
		hostData, err := ParseURL(rawURL_npipe, "http", "", "", "", "")
		if assert.NoError(t, err) {
			transport, ok := hostData.Transport.(*dialer.NpipeDialerBuilder)
			assert.True(t, ok)
			assert.Equal(t, `\\.\pipe\custom`, transport.Path)
		}
		exp := &expected{
			URI:          "http://npipe",
			SanitizedURI: "http://npipe",
			Host:         "npipe",
			User:         "",
			Password:     "",
		}
		assertTest(t, hostData, err, exp)
	})

	t.Run("http+npipe with path", func(t *testing.T) {
		hostData, err := ParseURL(rawURL_npipe, "http", "", "", "apath", "")
		if assert.NoError(t, err) {
			transport, ok := hostData.Transport.(*dialer.NpipeDialerBuilder)
			assert.True(t, ok)
			assert.Equal(t, `\\.\pipe\custom`, transport.Path)
		}
		exp := &expected{
			URI:          "http://npipe/apath",
			SanitizedURI: "http://npipe/apath",
			Host:         "npipe",
			User:         "",
			Password:     "",
		}
		assertTest(t, hostData, err, exp)
	})

	t.Run("http+npipe short with", func(t *testing.T) {
		rawURL := "http+npipe:///custom"
		hostData, err := ParseURL(rawURL, "http", "", "", "apath", "")
		if assert.NoError(t, err) {
			transport, ok := hostData.Transport.(*dialer.NpipeDialerBuilder)
			assert.True(t, ok)
			assert.Equal(t, `\\.\pipe\custom`, transport.Path)
			assert.Equal(t, "http://npipe/apath", hostData.URI)
			assert.Equal(t, "http://npipe/apath", hostData.SanitizedURI)
			assert.Equal(t, "npipe", hostData.Host)
			assert.Equal(t, "", hostData.User)
			assert.Equal(t, "", hostData.Password)
		}
	})

	t.Run("npipe", func(t *testing.T) {
		rawURL := "npipe://./pipe/docker_engine"
		hostData, err := ParseURL(rawURL, "tcp", "", "", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "npipe://./pipe/docker_engine", hostData.URI)
			assert.Equal(t, "npipe://./pipe/docker_engine", hostData.SanitizedURI)
			assert.Equal(t, "/pipe/docker_engine", hostData.Host)
			assert.Equal(t, "", hostData.User)
			assert.Equal(t, "", hostData.Password)
		}
	})

	t.Run("set default user", func(t *testing.T) {
		rawURL := "http://:secret@localhost"
		h, err := ParseURL(rawURL, "https", "root", "passwd", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "http://root:secret@localhost", h.URI)
			assert.Equal(t, "root", h.User)
			assert.Equal(t, "secret", h.Password)
		}
	})

	t.Run("set default password", func(t *testing.T) {
		rawURL := "http://admin@localhost"
		h, err := ParseURL(rawURL, "https", "root", "passwd", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "http://admin:passwd@localhost", h.URI)
			assert.Equal(t, "admin", h.User)
			assert.Equal(t, "passwd", h.Password)
		}
	})

	t.Run("don't overwrite empty password", func(t *testing.T) {
		rawURL := "http://admin:@localhost"
		h, err := ParseURL(rawURL, "https", "root", "passwd", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "http://admin:@localhost", h.URI)
			assert.Equal(t, "admin", h.User)
			assert.Equal(t, "", h.Password)
		}
	})

	t.Run("oracle", func(t *testing.T) {
		rawURL := "oracle://admin:secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/@127.0.0.1:8080/ORCLCDB"
		hostData, err := ParseURL(rawURL, "oracle", "", "", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "oracle://admin:secret%25~%60%21%40%23$%25%5E&%2A%28%29_+=-%7B%5B%7D%5D%7C%27%3A;%3E.%3C,%3F%2F@127.0.0.1:8080/ORCLCDB", hostData.URI)
			assert.Equal(t, "oracle://127.0.0.1:8080/ORCLCDB", hostData.SanitizedURI)
			assert.Equal(t, "127.0.0.1:8080", hostData.Host)
			assert.Equal(t, "admin", hostData.User)
			assert.Equal(t, "secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/", hostData.Password)
		}
		rawURL = "oracle://admin/secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/@127.0.0.1:8080/ORCLCDB"
		hostData, err = ParseURL(rawURL, "oracle", "", "", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "oracle://admin:secret%25~%60%21%40%23$%25%5E&%2A%28%29_+=-%7B%5B%7D%5D%7C%27%3A;%3E.%3C,%3F%2F@127.0.0.1:8080/ORCLCDB", hostData.URI)
			assert.Equal(t, "oracle://127.0.0.1:8080/ORCLCDB", hostData.SanitizedURI)
			assert.Equal(t, "127.0.0.1:8080", hostData.Host)
			assert.Equal(t, "admin", hostData.User)
			assert.Equal(t, "secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/", hostData.Password)
		}
		rawURL = "oracle://127.0.0.1:8080/ORCLCDB"
		hostData, err = ParseURL(rawURL, "oracle", "admin", "password", "", "")
		if assert.NoError(t, err) {
			assert.Equal(t, "oracle://127.0.0.1:8080/ORCLCDB", hostData.URI)
			assert.Equal(t, "oracle://127.0.0.1:8080/ORCLCDB", hostData.SanitizedURI)
			assert.Equal(t, "127.0.0.1:8080", hostData.Host)
			assert.Equal(t, "", hostData.User)
			assert.Equal(t, "", hostData.Password)
		}
	})
}

func TestURLHostParserBuilder(t *testing.T) {
	const rawURL = "http://example.com"

	var cases = []struct {
		config  map[string]interface{}
		builder URLHostParserBuilder
		url     string
	}{
		{map[string]interface{}{"path": "/path"}, URLHostParserBuilder{PathConfigKey: "path", DefaultPath: "/default"}, "http://example.com/path"},
		{map[string]interface{}{}, URLHostParserBuilder{PathConfigKey: "path", DefaultPath: "/default"}, "http://example.com/default"},
		{map[string]interface{}{}, URLHostParserBuilder{DefaultPath: "/default"}, "http://example.com/default"},
		{map[string]interface{}{"username": "guest"}, URLHostParserBuilder{}, "http://guest@example.com"},
		{map[string]interface{}{"username": "guest", "password": "secret"}, URLHostParserBuilder{}, "http://guest:secret@example.com"},
		{map[string]interface{}{"password": "secret"}, URLHostParserBuilder{}, "http://:secret@example.com"},
		{map[string]interface{}{"basepath": "/foo"}, URLHostParserBuilder{DefaultPath: "/default"}, "http://example.com/foo/default"},
		{map[string]interface{}{"basepath": "foo/"}, URLHostParserBuilder{DefaultPath: "/default"}, "http://example.com/foo/default"},
		{map[string]interface{}{"basepath": "/foo/"}, URLHostParserBuilder{DefaultPath: "/default"}, "http://example.com/foo/default"},
		{map[string]interface{}{"basepath": "foo"}, URLHostParserBuilder{DefaultPath: "/default"}, "http://example.com/foo/default"},
		{map[string]interface{}{"basepath": "foo"}, URLHostParserBuilder{DefaultPath: "/queryParams", QueryParams: mb.QueryParams{"key": "value"}.String()}, "http://example.com/foo/queryParams?key=value"},
	}

	for _, test := range cases {
		m := mbtest.NewTestModule(t, test.config)
		hostParser := test.builder.Build()

		hp, err := hostParser(m, rawURL)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, test.url, hp.URI)
	}
}

// TestOracleUrlParser function tests OracleUrlParser function with different urls
func TestOracleUrlParser(t *testing.T) {
	tests := []struct {
		arg          string
		wantHost     string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{"oracle://admin:secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/@127.0.0.1:8080/ORCLCDB", "oracle://127.0.0.1:8080/ORCLCDB", "admin", "secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/", false},
		{"oracle://admin/secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/@127.0.0.1:8080/ORCLCDB", "oracle://127.0.0.1:8080/ORCLCDB", "admin", "secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/", false},
		{"admin:secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/@127.0.0.1:8080/ORCLCDB", "oracle://127.0.0.1:8080/ORCLCDB", "admin", "secret%~`!@#$%^&*()_+=-{[}]|':;>.<,?/", false},
		{"admin@127.0.0.1:8080/ORCLCDB", "", "", "", true},
		{"127.0.0.1:8080/ORCLCDB", "oracle://127.0.0.1:8080/ORCLCDB", "", "", false},
	}
	for _, tt := range tests {
		t.Run("oracle", func(t *testing.T) {
			gotHost, gotUsername, gotPassword, err := OracleUrlParser(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("OracleUrlParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotHost != tt.wantHost {
				t.Errorf("OracleUrlParser() gotHost = %v, want %v", gotHost, tt.wantHost)
			}
			if gotUsername != tt.wantUsername {
				t.Errorf("OracleUrlParser() gotUsername = %v, want %v", gotUsername, tt.wantUsername)
			}
			if gotPassword != tt.wantPassword {
				t.Errorf("OracleUrlParser() gotPassword = %v, want %v", gotPassword, tt.wantPassword)
			}
		})
	}
}
