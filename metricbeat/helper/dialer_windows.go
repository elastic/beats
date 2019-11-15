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

//+build !windows

package helper

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

func makeDialer(t time.Duration, uri string) (transport.Dialer, string, error) {
	if strings.Contains(uri, "unix://") {
		return nil, fmt.Errorf(
			"cannot use %s as the URI, unix sockets are not supported on Windows, use npipe instead",
			uri,
		)
	}

	if strings.HasPrefix(uri, "http+npipe://") || strings.HasPrefix(uri, "npipe://") {
		u, err := url.Parse(uri)
		if err != nil {
			return nil, "", errors.Wrap(err, "fail to parse URI")
		}

		sockFile := u.Path

		q := u.Query()
		path := q.Get("__path")
		if path != "" {
			path, err = url.PathUnescape(path)
			if err != nil {
				return nil, "", fmt.Errorf("could not unescape resource path %s", path)
			}
		}
		q.Del("__path")

		var qStr string
		if encoded := q.Encode(); encoded != "" {
			qStr = "?" + encoded
		}

		return npipe.DialContext(npipe.TransformString(p)), "http://npipe/" + path + qStr, nil
	}

	return transport.NetDialer(t), uri, nil
}
