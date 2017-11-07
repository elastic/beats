package parse

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

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
