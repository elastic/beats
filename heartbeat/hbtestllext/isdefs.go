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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
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

var IsUint16 = isdef.Is("positiveUInt16", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(uint16)
	if !ok {
		return llresult.SimpleResult(path, false, "expected a uint16")
	}
	return llresult.ValidResult(path)
})

var IsMonitorState = isdef.Is("isState", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(monitorstate.State)
	if !ok {
		return llresult.SimpleResult(path, false, "expected a monitorstate.State")
	}
	return llresult.ValidResult(path)
})

var IsMonitorStateInLocation = func(locName string) isdef.IsDef {
	locPattern := fmt.Sprintf("^%s-[a-z0-9]+-0$", locName)
	stateIdMatch := regexp.MustCompile(locPattern)
	return isdef.Is("isState", func(path llpath.Path, v interface{}) *llresult.Results {
		s, ok := v.(monitorstate.State)
		if !ok {
			return llresult.SimpleResult(path, false, "expected a monitorstate.State")
		}

		if !stateIdMatch.MatchString(s.ID) {
			return llresult.SimpleResult(path, false, fmt.Sprintf("ID %s does not match regexp pattern /%s/", s.ID, locPattern))
		}
		return llresult.ValidResult(path)
	})
}

var IsECSErr = func(expectedErr *ecserr.ECSErr) isdef.IsDef {
	return isdef.Is("matches ECS ERR", func(path llpath.Path, v interface{}) *llresult.Results {
		// This conditional is a bit awkward, apparently there's a bug in lookslike where a pointer
		// value is de-referenced, so a given *ecserr.ECSErr turns into an ecserr.ECSErr
		var givenErr *ecserr.ECSErr
		givenErrNoPtr, ok := v.(ecserr.ECSErr)
		if !ok {
			return llresult.SimpleResult(path, false, "ecserr.ECSErr expected, got %v", v)
		}
		givenErr = &givenErrNoPtr

		if expectedErr.Code != givenErr.Code {
			return llresult.SimpleResult(path, false, "ECS error code does not match, expected %s, got %s", expectedErr.Code, givenErr.Code)
		}

		if expectedErr.Type != givenErr.Type {
			return llresult.SimpleResult(path, false, "ECS error type does not match, expected %s, got %s", expectedErr.Type, givenErr.Type)
		}

		if expectedErr.Message != givenErr.Message {
			return llresult.SimpleResult(path, false, "ECS error message does not match, expected %s, got %s", expectedErr.Message, givenErr.Message)
		}

		return llresult.ValidResult(path)
	})
}

var IsECSErrMatchingCode = func(ecode ecserr.ECode, messageContains string) isdef.IsDef {
	return isdef.Is("matches ECS ERR", func(path llpath.Path, v interface{}) *llresult.Results {
		// This conditional is a bit awkward, apparently there's a bug in lookslike where a pointer
		// value is de-referenced, so a given *ecserr.ECSErr turns into an ecserr.ECSErr
		var givenErr *ecserr.ECSErr
		givenErrNoPtr, ok := v.(ecserr.ECSErr)
		if !ok {
			return llresult.SimpleResult(path, false, "ecserr.ECSErr expected, got %v", v)
		}
		givenErr = &givenErrNoPtr

		if ecode != givenErr.Code {
			return llresult.SimpleResult(path, false, "ECS error code does not match, expected %s, got %s", ecode, givenErr.Code)
		}

		if !strings.Contains(givenErr.Message, messageContains) {
			return llresult.SimpleResult(path, false, "ECS error type does not match, expected '%s' to contain '%s'", givenErr.Message, messageContains)
		}

		return llresult.ValidResult(path)
	})
}
