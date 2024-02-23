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

package fileout

import (
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/fmtstr"

	"github.com/elastic/beats/v7/libbeat/beat"
)

var isWindowsPath = os.PathSeparator == '\\'

// PathFormatString is a wrapper around EventFormatString for the
// handling paths with a format expression that has access to the timestamp format.
// It has special handling for paths, specifically for windows path separator
// which would be interpreted as an escape character. This formatter double escapes
// the path separator so it is properly interpreted by the fmtstr processor
type PathFormatString struct {
	efs *fmtstr.EventFormatString
}

// Run executes the format string returning a new expanded string or an error
// if execution or event field expansion fails.
func (fs *PathFormatString) Run(timestamp time.Time) (string, error) {
	placeholderEvent := &beat.Event{
		Timestamp: timestamp,
	}
	return fs.efs.Run(placeholderEvent)
}

// Unpack tries to initialize the PathFormatString from provided value
// (which must be a string). Unpack method satisfies go-ucfg.Unpacker interface
// required by config.C, in order to use PathFormatString with
// `common.(*Config).Unpack()`.
func (fs *PathFormatString) Unpack(v interface{}) error {
	path, ok := v.(string)
	if !ok {
		return nil
	}

	if isWindowsPath {
		path = strings.ReplaceAll(path, "\\", "\\\\")
	}

	fs.efs = &fmtstr.EventFormatString{}
	return fs.efs.Unpack(path)
}
