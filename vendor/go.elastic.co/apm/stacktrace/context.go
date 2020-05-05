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
	"bufio"
	"net/http"
	"os"

	"go.elastic.co/apm/model"
)

// SetContext sets the source context for the given stack frames,
// with the specified number of pre- and post- lines.
func SetContext(setter ContextSetter, frames []model.StacktraceFrame, pre, post int) error {
	for i := 0; i < len(frames); i++ {
		if err := setter.SetContext(&frames[i], pre, post); err != nil {
			return err
		}
	}
	return nil
}

// ContextSetter is an interface that can be used for setting the source
// context for a stack frame.
type ContextSetter interface {
	// SetContext sets the source context for the given stack frame,
	// with the specified number of pre- and post- lines.
	SetContext(frame *model.StacktraceFrame, pre, post int) error
}

// FileSystemContextSetter returns a ContextSetter that sets context
// by reading file contents from the provided http.FileSystem.
func FileSystemContextSetter(fs http.FileSystem) ContextSetter {
	if fs == nil {
		panic("fs is nil")
	}
	return &fileSystemContextSetter{fs}
}

type fileSystemContextSetter struct {
	http.FileSystem
}

func (s *fileSystemContextSetter) SetContext(frame *model.StacktraceFrame, pre, post int) error {
	if frame.Line <= 0 {
		return nil
	}
	f, err := s.Open(frame.AbsolutePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var lineno int
	var line string
	preLines := make([]string, 0, pre)
	postLines := make([]string, 0, post)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineno++
		if lineno > frame.Line+post {
			break
		}
		switch {
		case lineno == frame.Line:
			line = scanner.Text()
		case lineno < frame.Line && lineno >= frame.Line-pre:
			preLines = append(preLines, scanner.Text())
		case lineno > frame.Line && lineno <= frame.Line+post:
			postLines = append(postLines, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	frame.ContextLine = line
	frame.PreContext = preLines
	frame.PostContext = postLines
	return nil
}
