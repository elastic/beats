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

package hbtestllext

import (
	"time"

	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// IsTime checks that the value is a time.Time instance.
var IsTime = isdef.Is("time", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(time.Time)
	if !ok {
		return llresult.SimpleResult(path, false, "expected a time.Time")
	}
	return llresult.ValidResult(path)
})

var IsInt64 = isdef.Is("positiveInt64", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(int64)
	if !ok {
		return llresult.SimpleResult(path, false, "expected an int64")
	}
	return llresult.ValidResult(path)
})

var IsDuration = isdef.Is("duration", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(time.Duration)
	if !ok {
		return llresult.SimpleResult(path, false, "expected a duration")
	}
	return llresult.ValidResult(path)
})
