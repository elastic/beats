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

package tracelog

import (
	"fmt"
	"os"
	"strings"
)

type Logger interface {
	Println(...interface{})
	Printf(string, ...interface{})
}

type stderrLogger struct{}

type nilLogger struct{}

func Get(selector string) Logger {
	if isEnabled(selector) {
		return (*stderrLogger)(nil)
	}
	return (*nilLogger)(nil)
}

func isEnabled(selector string) bool {
	v := os.Getenv("TRACE_SELECTOR")
	if v == "" {
		return true
	}

	selectors := strings.Split(v, ",")
	for _, sel := range selectors {
		if selector == strings.TrimSpace(sel) {
			return true
		}
	}
	return false
}

func (*nilLogger) Println(...interface{})        {}
func (*nilLogger) Printf(string, ...interface{}) {}

func (*stderrLogger) Println(vs ...interface{})          { fmt.Fprintln(os.Stderr, vs...) }
func (*stderrLogger) Printf(s string, vs ...interface{}) { fmt.Fprintf(os.Stderr, s, vs...) }
