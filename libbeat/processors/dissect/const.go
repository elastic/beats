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

package dissect

import (
	"errors"
	"regexp"
)

var (
	// delimiterRE tokenizes the following string into walkable with extracted delimiter + key.
	// string:
	// ` %{key}, %{key/2}`
	// into:
	// [["", "key" ], [", ", "key/2"]]
	delimiterRE = regexp.MustCompile("(?s)(.*?)%\\{([^}]*?)}")
	suffixRE    = regexp.MustCompile("(.+?)(/(\\d{1,2}))?(->)?$")

	skipFieldPrefix      = "?"
	appendFieldPrefix    = "+"
	indirectFieldPrefix  = "&"
	appendIndirectPrefix = "+&"
	indirectAppendPrefix = "&+"
	greedySuffix         = "->"
	pointerFieldPrefix   = "*"

	defaultJoinString = " "

	errParsingFailure            = errors.New("parsing failure")
	errInvalidTokenizer          = errors.New("invalid dissect tokenizer")
	errEmpty                     = errors.New("empty string provided")
	errMixedPrefixIndirectAppend = errors.New("mixed prefix `&+`")
	errMixedPrefixAppendIndirect = errors.New("mixed prefix `&+`")
	errEmptyKey                  = errors.New("empty key")
)
