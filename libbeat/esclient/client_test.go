package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		version     *common.Version
		assertionFn assert.ValueAssertionFunc
	}{
		"v6": {
			common.MustNewVersion("6.8.4"),
			func(t assert.TestingT, c interface{}, rest ...interface{}) bool {
				return assert.Nil(t, rest[0]) &&
					assert.IsType(t, &Client{}, c) &&
					assert.NotNil(t, c.(*Client).e6) &&
					assert.Nil(t, c.(*Client).e7) &&
					assert.Nil(t, c.(*Client).e8)
			},
		},
		"v7": {
			common.MustNewVersion("7.6.1"),
			func(t assert.TestingT, c interface{}, rest ...interface{}) bool {
				return assert.Nil(t, rest[0]) &&
					assert.IsType(t, &Client{}, c) &&
					assert.NotNil(t, c.(*Client).e7) &&
					assert.Nil(t, c.(*Client).e6) &&
					assert.Nil(t, c.(*Client).e8)
			},
		},
		"v8": {
			common.MustNewVersion("8.0.0"),
			func(t assert.TestingT, c interface{}, rest ...interface{}) bool {
				return assert.Nil(t, rest[0]) &&
					assert.IsType(t, &Client{}, c) &&
					assert.NotNil(t, c.(*Client).e8) &&
					assert.Nil(t, c.(*Client).e6) &&
					assert.Nil(t, c.(*Client).e7)
			},
		},
		"not_supported": {
			common.MustNewVersion("5.5.0"),
			func(t assert.TestingT, c interface{}, rest ...interface{}) bool {
				return assert.Nil(t, c) &&
					assert.IsType(t, assert.AnError, rest[0]) &&
					assert.Error(t, rest[0].(error)) &&
					assert.Equal(t, ErrUnsupportedVersion, rest[0].(error))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				io.WriteString(rw, `{"version": {"number": "`+test.version.String()+`"}}`)
			}))
			defer server.Close()

			c, err := New(WithAddresses(server.URL))

			if !test.assertionFn(t, c, err) {
				t.FailNow()
			}
		})
	}
}
