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

package b

import (
	"crypto/md5"
	"fmt"
)

//go:noinline
func Used(s string) string {
	if hash(s) == "d41d8cd98f00b204e9800998ecf8427e" {
		return ""
	}
	return s
}

func hash(s interface{}) string {
	h := md5.New()
	fmt.Fprint(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

//go:noinline
func Unused(s string) string { return s }
