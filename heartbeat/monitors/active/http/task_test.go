package http

import (
	"net"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestSplitHostnamePort(t *testing.T) {
	var urlTests = []struct {
		scheme        string
		host          string
		expectedHost  string
		expectedPort  uint16
		expectedError error
	}{
		{
			"http",
			"foo",
			"foo",
			80,
			nil,
		},
		{
			"http",
			"www.foo.com",
			"www.foo.com",
			80,
			nil,
		},
		{
			"http",
			"www.foo.com:8080",
			"www.foo.com",
			8080,
			nil,
		},
		{
			"https",
			"foo",
			"foo",
			443,
			nil,
		},
		{
			"http",
			"foo:81",
			"foo",
			81,
			nil,
		},
		{
			"https",
			"foo:444",
			"foo",
			444,
			nil,
		},
		{
			"httpz",
			"foo",
			"foo",
			81,
			&net.AddrError{},
		},
	}
	for _, test := range urlTests {
		url := &url.URL{
			Scheme: test.scheme,
			Host:   test.host,
		}
		request := &http.Request{
			URL: url,
		}
		host, port, err := splitHostnamePort(request)
		if err != nil {
			if test.expectedError == nil {
				t.Error(err)
			} else if reflect.TypeOf(err) != reflect.TypeOf(test.expectedError) {
				t.Errorf("Expected %T but got %T", err, test.expectedError)
			}
			continue
		}
		if host != test.expectedHost {
			t.Errorf("Unexpected host for %#v: expected %q, got %q", request, test.expectedHost, host)
		}
		if port != test.expectedPort {
			t.Errorf("Unexpected port for %#v: expected %q, got %q", request, test.expectedPort, port)
		}
	}
}
