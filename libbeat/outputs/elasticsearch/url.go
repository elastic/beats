// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
