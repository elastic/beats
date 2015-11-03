package elasticsearch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlEncode(t *testing.T) {

	params := map[string]string{
		"q": "agent:appserver1",
	}
	url := urlEncode(params)

	if url != "q=agent%3Aappserver1" {
		t.Errorf("Fail to encode params: %s", url)
	}

	params = map[string]string{
		"wife":    "sarah",
		"husband": "joe",
	}

	url = urlEncode(params)

	if url != "husband=joe&wife=sarah" {
		t.Errorf("Fail to encode params: %s", url)
	}
}

func TestMakePath(t *testing.T) {
	path, err := makePath("twitter", "tweet", "1")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/tweet/1" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = makePath("twitter", "", "_refresh")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter/_refresh" {
		t.Errorf("Wrong path created: %s", path)
	}

	path, err = makePath("", "", "_bulk")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/_bulk" {
		t.Errorf("Wrong path created: %s", path)
	}
	path, err = makePath("twitter", "", "")
	if err != nil {
		t.Errorf("Fail to create path: %s", err)
	}
	if path != "/twitter" {
		t.Errorf("Wrong path created: %s", path)
	}

}

func TestGetUrl(t *testing.T) {

	// List of inputs / outputs that must match after fetching url
	// Setting a path without a scheme is not allowed. Example: 192.168.1.1:9200/hello
	inputOutput := map[string]string{
		// shema + hostname
		"":                     "http://localhost:9200",
		"http://localhost":     "http://localhost:9200",
		"http://localhost:80":  "http://localhost:80",
		"http://localhost:80/": "http://localhost:80/",
		"http://localhost/":    "http://localhost:9200/",

		// no schema + hostname
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
		urlNew, err := getURL("", "", input)
		assert.Nil(t, err)
		assert.Equal(t, output, urlNew, fmt.Sprintf("input: %v", input))
	}

	inputOutputWithDefaults := map[string]string{
		"http://localhost":                          "http://localhost:9200/hello",
		"192.156.4.5":                               "https://192.156.4.5:9200/hello",
		"http://username:password@es.found.io:9324": "http://username:password@es.found.io:9324/hello",
	}
	for input, output := range inputOutputWithDefaults {
		urlNew, err := getURL("https", "/hello", input)
		assert.Nil(t, err)
		assert.Equal(t, output, urlNew)
	}
}
