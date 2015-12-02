package elasticsearch

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Creates the url based on the url configuration.
// Adds missing parts with defaults (scheme, host, port)
func getURL(defaultScheme string, defaultPath string, rawURL string) (string, error) {

	if defaultScheme == "" {
		defaultScheme = "http"
	}

	addr, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	scheme := addr.Scheme
	host := addr.Host
	port := "9200"

	// sanitize parse errors if url does not contain scheme
	// if parse url looks funny, prepend schema and try again:
	if addr.Scheme == "" || (addr.Host == "" && addr.Path == "" && addr.Opaque != "") {
		rawURL = fmt.Sprintf("%v://%v", defaultScheme, rawURL)
		if tmpAddr, err := url.Parse(rawURL); err == nil {
			addr = tmpAddr
			scheme = addr.Scheme
			host = addr.Host
		} else {
			// If url doesn't have a scheme, host is written into path. For example: 192.168.3.7
			scheme = defaultScheme
			host = addr.Path
			addr.Path = ""
		}
	}

	if host == "" {
		host = "localhost"
	} else {
		// split host and optional port
		if splitHost, splitPort, err := net.SplitHostPort(host); err == nil {
			host = splitHost
			port = splitPort
		}

		// Check if ipv6
		if strings.Count(host, ":") > 1 && strings.Count(host, "]") == 0 {
			host = "[" + host + "]"
		}
	}

	// Assign default path if not set
	if addr.Path == "" {
		addr.Path = defaultPath
	}

	// reconstruct url
	addr.Scheme = scheme
	addr.Host = host + ":" + port
	return addr.String(), nil
}

func makeURL(url, path string, params map[string]string) string {
	u := url + path
	if len(params) > 0 {
		u = u + "?" + urlEncode(params)
	}
	return u
}

// Encode parameters in url
func urlEncode(params map[string]string) string {
	values := url.Values{}

	for key, val := range params {
		values.Add(key, string(val))
	}
	return values.Encode()
}

// Create path out of index, docType and id that is used for querying Elasticsearch
func makePath(index string, docType string, id string) (string, error) {

	var path string
	if len(docType) > 0 {
		if len(id) > 0 {
			path = fmt.Sprintf("/%s/%s/%s", index, docType, id)
		} else {
			path = fmt.Sprintf("/%s/%s", index, docType)
		}
	} else {
		if len(id) > 0 {
			if len(index) > 0 {
				path = fmt.Sprintf("/%s/%s", index, id)
			} else {
				path = fmt.Sprintf("/%s", id)
			}
		} else {
			path = fmt.Sprintf("/%s", index)
		}
	}
	return path, nil
}
