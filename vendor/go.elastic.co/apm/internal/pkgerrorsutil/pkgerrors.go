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

package pkgerrorsutil

import (
	"reflect"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"

	"go.elastic.co/apm/stacktrace"
)

var (
	uintptrType             = reflect.TypeOf(uintptr(0))
	runtimeFrameType        = reflect.TypeOf(runtime.Frame{})
	errorsStackTraceUintptr = uintptrType.ConvertibleTo(reflect.TypeOf(*new(errors.Frame)))
	errorsStackTraceFrame   = reflect.TypeOf(*new(errors.Frame)).ConvertibleTo(runtimeFrameType)
)

// AppendStacktrace appends stack frames to out, based on stackTrace.
func AppendStacktrace(stackTrace errors.StackTrace, out *[]stacktrace.Frame, limit int) {
	// github.com/pkg/errors 0.8.x and earlier represent
	// stack frames as uintptr; 0.9.0 and later represent
	// them as runtime.Frames.
	//
	// TODO(axw) drop support for older github.com/pkg/errors
	// versions when we release go.elastic.co/apm v2.0.0.
	if errorsStackTraceUintptr {
		pc := make([]uintptr, len(stackTrace))
		for i, frame := range stackTrace {
			pc[i] = *(*uintptr)(unsafe.Pointer(&frame))
		}
		*out = stacktrace.AppendCallerFrames(*out, pc, limit)
	} else if errorsStackTraceFrame {
		if limit >= 0 && len(stackTrace) > limit {
			stackTrace = stackTrace[:limit]
		}
		for _, frame := range stackTrace {
			rf := (*runtime.Frame)(unsafe.Pointer(&frame))
			*out = append(*out, stacktrace.RuntimeFrame(*rf))
		}
	}
}
