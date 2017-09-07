// +build !integration

package common

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUrl(t *testing.T) {
	// List of inputs / outputs that must match after fetching url
	// Setting a path without a scheme is not allowed. Example: 192.168.1.1:9200/hello
	inputOutput := map[string]string{

		"":                     "http://localhost:9200",
		"http://localhost":     "http://localhost:9200",
		"http://localhost:80":  "http://localhost:80",
		"http://localhost:80/": "http://localhost:80/",
		"http://localhost/":    "http://localhost:9200/",

		// no schema + hostname
		"33f3600fd5c1bb599af557c36a4efb08.host":       "http://33f3600fd5c1bb599af557c36a4efb08.host:9200",
		"33f3600fd5c1bb599af557c36a4efb08.host:12345": "http://33f3600fd5c1bb599af557c36a4efb08.host:12345",
		"localhost":        "http://localhost:9200",
		"localhost:80":     "http://localhost:80",
		"localhost:80/":    "http://localhost:80/",
		"localhost/":       "http://localhost:9200/",
		"localhost/mypath": "http://localhost:9200/mypath",

		// shema + ipv4
		"http://192.168.1.1:80":        "http://192.168.1.1:80",
		"https://192.168.1.1:80/hello": "https://192.168.1.1:80/hello",
		"http://192.168.1.1":           "http://192.168.1.1:9200",
		"http://192.168.1.1/hello":     "http://192.168.1.1:9200/hello",

		// no schema + ipv4
		"192.168.1.1":          "http://192.168.1.1:9200",
		"192.168.1.1:80":       "http://192.168.1.1:80",
		"192.168.1.1/hello":    "http://192.168.1.1:9200/hello",
		"192.168.1.1:80/hello": "http://192.168.1.1:80/hello",

		// schema + ipv6
		"http://[2001:db8::1]:80":                              "http://[2001:db8::1]:80",
		"http://[2001:db8::1]":                                 "http://[2001:db8::1]:9200",
		"https://[2001:db8::1]:9200":                           "https://[2001:db8::1]:9200",
		"http://FE80:0000:0000:0000:0202:B3FF:FE1E:8329":       "http://[FE80:0000:0000:0000:0202:B3FF:FE1E:8329]:9200",
		"http://[2001:db8::1]:80/hello":                        "http://[2001:db8::1]:80/hello",
		"http://[2001:db8::1]/hello":                           "http://[2001:db8::1]:9200/hello",
		"https://[2001:db8::1]:9200/hello":                     "https://[2001:db8::1]:9200/hello",
		"http://FE80:0000:0000:0000:0202:B3FF:FE1E:8329/hello": "http://[FE80:0000:0000:0000:0202:B3FF:FE1E:8329]:9200/hello",

		// no schema + ipv6
		"2001:db8::1":            "http://[2001:db8::1]:9200",
		"[2001:db8::1]:80":       "http://[2001:db8::1]:80",
		"[2001:db8::1]":          "http://[2001:db8::1]:9200",
		"2001:db8::1/hello":      "http://[2001:db8::1]:9200/hello",
		"[2001:db8::1]:80/hello": "http://[2001:db8::1]:80/hello",
		"[2001:db8::1]/hello":    "http://[2001:db8::1]:9200/hello",
	}

	for input, output := range inputOutput {
		urlNew, err := MakeURL("", "", input, 9200)
		assert.Nil(t, err)
		assert.Equal(t, output, urlNew, fmt.Sprintf("input: %v", input))
	}

	inputOutputWithDefaults := map[string]string{
		"http://localhost":                          "http://localhost:9200/hello",
		"http://localhost/test":                     "http://localhost:9200/test",
		"192.156.4.5":                               "https://192.156.4.5:9200/hello",
		"http://username:password@es.found.io:9324": "http://username:password@es.found.io:9324/hello",
	}

	for input, output := range inputOutputWithDefaults {
		urlNew, err := MakeURL("https", "/hello", input, 9200)
		assert.Nil(t, err)
		assert.Equal(t, output, urlNew)
	}
}

func TestURLParamsEncode(t *testing.T) {
	inputOutputWithParams := map[string]string{
		"http://localhost": "http://localhost:5601?dashboard=first&dashboard=second",
	}

	params := url.Values{}
	params.Add("dashboard", "first")
	params.Add("dashboard", "second")

	for input, output := range inputOutputWithParams {
		urlNew, err := MakeURL("", "", input, 5601)
		urlWithParams := EncodeURLParams(urlNew, params)
		assert.Nil(t, err)
		assert.Equal(t, output, urlWithParams)
	}
}
