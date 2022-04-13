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

package gotool

import "github.com/magefile/mage/sh"

type goNoticeGenerator func(opts ...ArgOpt) error

// NoticeGenerator runs `go-license-detector` and provides optionals for adding command line arguments.
var NoticeGenerator goNoticeGenerator = runGoNoticeGenerator

func runGoNoticeGenerator(opts ...ArgOpt) error {
	args := buildArgs(opts).build()
	return sh.RunV("go-licence-detector", args...)
}

func (goNoticeGenerator) Dependencies(path string) ArgOpt   { return flagArg("-in", path) }
func (goNoticeGenerator) IncludeIndirect() ArgOpt           { return flagBoolIf("-includeIndirect", true) }
func (goNoticeGenerator) Rules(path string) ArgOpt          { return flagArg("-rules", path) }
func (goNoticeGenerator) Overrides(path string) ArgOpt      { return flagArg("-overrides", path) }
func (goNoticeGenerator) NoticeTemplate(path string) ArgOpt { return flagArg("-noticeTemplate", path) }
func (goNoticeGenerator) NoticeOutput(path string) ArgOpt   { return flagArg("-noticeOut", path) }
func (goNoticeGenerator) DepsOutput(path string) ArgOpt     { return flagArg("-depsOut", path) }
