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

package gotype

import "strings"

type tagOptions struct {
	squash    bool
	omitEmpty bool
}

var defaultTagOptions = tagOptions{
	squash:    false,
	omitEmpty: false,
}

func parseTags(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	if len(s) == 0 {
		return "", defaultTagOptions
	}
	opts := defaultTagOptions
	for _, opt := range s[1:] {
		switch strings.TrimSpace(opt) {
		case "squash", "inline":
			opts.squash = true
		case "omitempty":
			opts.omitEmpty = true
		}
	}
	return strings.TrimSpace(s[0]), opts
}
