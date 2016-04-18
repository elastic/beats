// +build !integration

package status

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostParse(t *testing.T) {
	var tests = []struct {
		host string
		url  string
		err  string
	}{
		{"", "", "error parsing apache host: empty host"},
		{":80", "", "error parsing apache host: parse :80: missing protocol scheme"},
		{"localhost", "http://localhost/server-status?auto=", ""},
		{"localhost/ServerStatus", "http://localhost/ServerStatus?auto=", ""},
		{"127.0.0.1", "http://127.0.0.1/server-status?auto=", ""},
		{"https://127.0.0.1", "https://127.0.0.1/server-status?auto=", ""},
		{"[2001:db8:0:1]:80", "http://[2001:db8:0:1]:80/server-status?auto=", ""},
		{"https://admin:secret@127.0.0.1", "https://admin:secret@127.0.0.1/server-status?auto=", ""},
	}

	for _, test := range tests {
		u, err := getURL("", "", defaultPath, test.host)
		if err != nil && test.err != "" {
			assert.Equal(t, test.err, err.Error())
		} else if assert.NoError(t, err, "unexpected error") {
			assert.Equal(t, test.url, u.String())
		}
	}
}

func TestRedactPassword(t *testing.T) {
	rawURL := "https://admin:secret@127.0.0.1"
	u, err := url.Parse(rawURL)
	if assert.NoError(t, err) {
		assert.Equal(t, "https://admin@127.0.0.1", redactPassword(*u))
		// redactPassword shall not modify the URL.
		assert.Equal(t, rawURL, u.String())
	}
}
