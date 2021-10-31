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

// +build go1.10

package apmelasticsearch

import (
	"net/http"
	"strings"
)

func requestName(req *http.Request) string {
	const prefix = "Elasticsearch:"
	path := strings.TrimLeft(req.URL.Path, "/")

	var b strings.Builder
	b.Grow(len(prefix) + 1 + len(req.Method) + 1 + len(path))
	b.WriteString(prefix)
	b.WriteRune(' ')
	b.WriteString(req.Method)
	b.WriteRune(' ')
	b.WriteString(path)
	return b.String()
}
