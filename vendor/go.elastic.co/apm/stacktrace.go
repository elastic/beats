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

package apm

import (
	"path/filepath"

	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
)

func appendModelStacktraceFrames(out []model.StacktraceFrame, in []stacktrace.Frame) []model.StacktraceFrame {
	for _, f := range in {
		out = append(out, modelStacktraceFrame(f))
	}
	return out
}

func modelStacktraceFrame(in stacktrace.Frame) model.StacktraceFrame {
	var abspath string
	file := in.File
	if file != "" {
		if filepath.IsAbs(file) {
			abspath = file
		}
		file = filepath.Base(file)
	}
	packagePath, function := stacktrace.SplitFunctionName(in.Function)
	return model.StacktraceFrame{
		AbsolutePath: abspath,
		File:         file,
		Line:         in.Line,
		Function:     function,
		Module:       packagePath,
		LibraryFrame: stacktrace.IsLibraryPackage(packagePath),
	}
}
