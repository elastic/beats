package elasticsearch

import (
	"fmt"
	"net/url"
	"strings"
)

func addToURL(url, path, pipeline string, params map[string]string) string {
	if strings.HasSuffix(url, "/") && strings.HasPrefix(path, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	if len(params) == 0 && pipeline == "" {
		return url + path
	}

	return strings.Join([]string{
		url, path, "?", urlEncode(pipeline, params),
	}, "")
}

// Encode parameters in url
func urlEncode(pipeline string, params map[string]string) string {
	values := url.Values{}

	for key, val := range params {
		values.Add(key, string(val))
	}

	if pipeline != "" {
		values.Add("pipeline", pipeline)
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

// TODO: make this reusable. Same definition in elasticsearch monitoring module
func parseProxyURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}

	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}
