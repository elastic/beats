package http

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"net/url"
	"strings"
)

func addToURL(urlStr string, params map[string]string) string {
	if strings.HasSuffix(urlStr, "/") {
		urlStr = strings.TrimSuffix(urlStr, "/")
	}
	if len(params) == 0 {
		return urlStr
	}
	values := url.Values{}
	for key, val := range params {
		values.Add(key, val)
	}
	return common.EncodeURLParams(urlStr, values)
}

func parseProxyURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	parsedUrl, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(parsedUrl.Scheme, "http") {
		return parsedUrl, err
	}
	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}
