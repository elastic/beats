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

package parse

import (
	"fmt"
	"net"
	"net/url"
	p "path"
	"strings"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

// URLHostParserBuilder builds a tailored HostParser for used with host strings
// that are URLs.
type URLHostParserBuilder struct {
	PathConfigKey   string
	DefaultPath     string
	DefaultUsername string
	DefaultPassword string
	DefaultScheme   string
	QueryParams     string
}

// Build returns a new HostParser function whose behavior is influenced by the
// options set in URLHostParserBuilder.
func (b URLHostParserBuilder) Build() mb.HostParser {
	return func(module mb.Module, host string) (mb.HostData, error) {
		conf := map[string]interface{}{}
		err := module.UnpackConfig(conf)
		if err != nil {
			return mb.HostData{}, err
		}

		query, ok := conf["query"]
		if ok {
			queryMap, ok := query.(map[string]interface{})
			if !ok {
				return mb.HostData{}, errors.Errorf("'query' config for module %v is not a map", module.Name())
			}

			b.QueryParams = mb.QueryParams(queryMap).String()
		}

		var user, pass, path, basePath string
		t, ok := conf["username"]
		if ok {
			user, ok = t.(string)
			if !ok {
				return mb.HostData{}, errors.Errorf("'username' config for module %v is not a string", module.Name())
			}
		} else {
			user = b.DefaultUsername
		}
		t, ok = conf["password"]
		if ok {
			pass, ok = t.(string)
			if !ok {
				return mb.HostData{}, errors.Errorf("'password' config for module %v is not a string", module.Name())
			}
		} else {
			pass = b.DefaultPassword
		}
		t, ok = conf[b.PathConfigKey]
		if ok {
			path, ok = t.(string)
			if !ok {
				return mb.HostData{}, errors.Errorf("'%v' config for module %v is not a string", b.PathConfigKey, module.Name())
			}
		} else {
			path = b.DefaultPath
		}
		// Normalize path
		path = strings.Trim(path, "/")

		t, ok = conf["basepath"]
		if ok {
			basePath, ok = t.(string)
			if !ok {
				return mb.HostData{}, errors.Errorf("'basepath' config for module %v is not a string", module.Name())
			}
		}
		// Normalize basepath
		basePath = strings.Trim(basePath, "/")

		// Combine paths and normalize
		fullPath := strings.Trim(p.Join(basePath, path), "/")

		return ParseURL(host, b.DefaultScheme, user, pass, fullPath, b.QueryParams)
	}
}

// NewHostDataFromURL returns a new HostData based on the contents of the URL.
// If the URLs scheme is "unix" or end is "unix" (e.g. "http+unix://") then
// the HostData.Host field is set to the URLs path instead of the URLs host,
// the same happens for "npipe".
func NewHostDataFromURL(u *url.URL) mb.HostData {
	var user, pass string
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}

	host := u.Host
	if strings.HasSuffix(u.Scheme, "unix") || strings.HasSuffix(u.Scheme, "npipe") {
		host = u.Path
	}

	return mb.HostData{
		URI:          u.String(),
		SanitizedURI: redactURLCredentials(u).String(),
		Host:         host,
		User:         user,
		Password:     pass,
	}
}

// ParseURL returns HostData object from a raw 'host' value and a series of
// defaults that are added to the URL if not present in the rawHost value.
// Values from the rawHost take precedence over the defaults.
func ParseURL(rawHost, scheme, user, pass, path, query string) (mb.HostData, error) {
	u, err := getURL(rawHost, scheme, user, pass, path, query)
	if err != nil {
		return mb.HostData{}, err
	}

	return NewHostDataFromURL(u), nil
}

// SetURLUser set the user credentials in the given URL. If the username or
// password is not set in the URL then the default is used (if provided).
func SetURLUser(u *url.URL, defaultUser, defaultPass string) {
	var user, pass string
	var userIsSet, passIsSet bool
	if u.User != nil {
		user = u.User.Username()
		if user != "" {
			userIsSet = true
		}
		pass, passIsSet = u.User.Password()
	}

	if !userIsSet && defaultUser != "" {
		userIsSet = true
		user = defaultUser
	}

	if !passIsSet && defaultPass != "" {
		passIsSet = true
		pass = defaultPass
	}

	if userIsSet && passIsSet {
		u.User = url.UserPassword(user, pass)
	} else if userIsSet {
		u.User = url.User(user)
	}
}

// getURL constructs a URL from the rawHost value and adds the provided user,
// password, path, and query params if one was not set in the rawURL value.
func getURL(rawURL, scheme, username, password, path, query string) (*url.URL, error) {
	if parts := strings.SplitN(rawURL, "://", 2); len(parts) != 2 {
		// Add scheme.
		rawURL = fmt.Sprintf("%s://%s", scheme, rawURL)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %v", err)
	}

	SetURLUser(u, username, password)

	if !strings.HasSuffix(u.Scheme, "unix") && !strings.HasSuffix(u.Scheme, "npipe") {
		if u.Host == "" {
			return nil, fmt.Errorf("error parsing URL: empty host")
		}

		// Validate the host. The port is optional.
		host, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			if strings.Contains(err.Error(), "missing port") {
				host = u.Host
			} else {
				return nil, fmt.Errorf("error parsing URL: %v", err)
			}
		}
		if host == "" {
			return nil, fmt.Errorf("error parsing URL: empty host")
		}
	}

	if u.Path == "" && path != "" {
		// The path given in the host config takes precedence over the
		// default path.
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		u.Path = path
	}

	//Adds the query params in the url
	u, err = SetQueryParams(u, query)
	return u, err
}

// SetQueryParams adds the query params to existing query parameters overwriting any
// keys that already exist.
func SetQueryParams(u *url.URL, query string) (*url.URL, error) {
	q := u.Query()
	params, err := url.ParseQuery(query)
	if err != nil {
		return u, err
	}
	for key, values := range params {
		for _, v := range values {
			q.Set(key, v)
		}
	}
	u.RawQuery = q.Encode()
	return u, nil

}

// redactURLCredentials returns the URL as a string with the username and
// password redacted.
func redactURLCredentials(u *url.URL) *url.URL {
	redacted := *u
	redacted.User = nil
	return &redacted
}
