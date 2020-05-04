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

package stacktrace

import (
	"runtime"
	"strings"
)

//go:generate /bin/bash generate_library.bash std ..

// AppendStacktrace appends at most n entries to frames,
// skipping skip frames starting with AppendStacktrace,
// and returns the extended slice. If n is negative, then
// all stack frames will be appended.
//
// See RuntimeFrame for information on what details are included.
func AppendStacktrace(frames []Frame, skip, n int) []Frame {
	if n == 0 {
		return frames
	}
	var pc []uintptr
	if n > 0 && n <= 10 {
		pc = make([]uintptr, n)
		pc = pc[:runtime.Callers(skip+1, pc)]
	} else {
		// n is negative or > 10, allocate space for 10
		// and make repeated calls to runtime.Callers
		// until we've got all the frames or reached n.
		pc = make([]uintptr, 10)
		m := 0
		for {
			m += runtime.Callers(skip+m+1, pc[m:])
			if m < len(pc) || m == n {
				pc = pc[:m]
				break
			}
			// Extend pc's length, ensuring its length
			// extends to its new capacity to minimise
			// the number of calls to runtime.Callers.
			pc = append(pc, 0)
			for len(pc) < cap(pc) {
				pc = append(pc, 0)
			}
		}
	}
	return AppendCallerFrames(frames, pc, n)
}

// AppendCallerFrames appends to n frames for the PCs in callers,
// and returns the extended slice. If n is negative, all available
// frames will be added. Multiple frames may exist for the same
// caller/PC in the case of function call inlining.
//
// See RuntimeFrame for information on what details are included.
func AppendCallerFrames(frames []Frame, callers []uintptr, n int) []Frame {
	if len(callers) == 0 {
		return frames
	}
	runtimeFrames := runtime.CallersFrames(callers)
	for i := 0; n < 0 || i < n; i++ {
		runtimeFrame, more := runtimeFrames.Next()
		frames = append(frames, RuntimeFrame(runtimeFrame))
		if !more {
			break
		}
	}
	return frames
}

// RuntimeFrame returns a Frame based on the given runtime.Frame.
//
// The resulting Frame will have the file path, package-qualified
// function name, and line number set. The function name can be
// split using SplitFunctionName, and the absolute path of the
// file and its base name can be determined using standard filepath
// functions.
func RuntimeFrame(in runtime.Frame) Frame {
	return Frame{
		File:     in.File,
		Function: in.Function,
		Line:     in.Line,
	}
}

// SplitFunctionName splits the function name as formatted in
// runtime.Frame.Function, and returns the package path and
// function name components.
func SplitFunctionName(in string) (packagePath, function string) {
	function = in
	if function == "" {
		return "", ""
	}
	// The last part of a package path will always have "."
	// encoded as "%2e", so we can pick off the package path
	// by finding the last part of the package path, and then
	// the proceeding ".".
	//
	// Unexported method names may contain the package path.
	// In these cases, the method receiver will be enclosed
	// in parentheses, so we can treat that as the start of
	// the function name.
	sep := strings.Index(function, ".(")
	if sep >= 0 {
		packagePath = unescape(function[:sep])
		function = function[sep+1:]
	} else {
		offset := 0
		if sep := strings.LastIndex(function, "/"); sep >= 0 {
			offset = sep
		}
		if sep := strings.IndexRune(function[offset+1:], '.'); sep >= 0 {
			packagePath = unescape(function[:offset+1+sep])
			function = function[offset+1+sep+1:]
		}
	}
	return packagePath, function
}

func unescape(s string) string {
	var n int
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			n++
		}
	}
	if n == 0 {
		return s
	}
	bytes := make([]byte, 0, len(s)-2*n)
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '%' && i+2 < len(s) {
			b = fromhex(s[i+1])<<4 | fromhex(s[i+2])
			i += 2
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}

func fromhex(b byte) byte {
	if b >= 'a' {
		return 10 + b - 'a'
	}
	return b - '0'
}
