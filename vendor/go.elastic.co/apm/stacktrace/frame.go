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

// Frame describes a stack frame.
type Frame struct {
	// File is the filename of the location of the stack frame.
	// This may be either the absolute or base name of the file.
	File string

	// Line is the 1-based line number of the location of the
	// stack frame, or zero if unknown.
	Line int

	// Function is the name of the function name for this stack
	// frame. This should be package-qualified, and may be split
	// using stacktrace.SplitFunctionName.
	Function string
}
