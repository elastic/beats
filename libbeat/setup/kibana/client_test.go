package kibana

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorJson(t *testing.T) {
	// also common 200: {"objects":[{"id":"apm-*","type":"index-pattern","error":{"message":"[doc][index-pattern:test-*]: version conflict, document already exists (current version [1])"}}]}
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"objects":[{"id":"test-*","type":"index-pattern","error":{"message":"action [indices:data/write/bulk[s]] is unauthorized for user [test]"}}]}`))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		http: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Error(t, err)
}

func TestErrorBadJson(t *testing.T) {
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{`))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		http: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.Error(t, err)
}

func TestSuccess(t *testing.T) {
	kibanaTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"objects":[{"id":"test-*","type":"index-pattern","updated_at":"2018-01-24T19:04:13.371Z","version":1}]}`))
	}))
	defer kibanaTs.Close()

	conn := Connection{
		URL:  kibanaTs.URL,
		http: http.DefaultClient,
	}
	code, _, err := conn.Request(http.MethodPost, "", url.Values{}, nil)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
}
