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

//go:build linux

package parser

// The auditd parser only functions on Linux (it is a stub elsewhere), so its
// decode-buffer reuse torture case and coverage requirement are registered here.
func init() {
	reuseTortureCases = append(reuseTortureCases, reuseCaseSpec{
		name:    "auditd",
		parsers: []map[string]interface{}{{"auditd": map[string]interface{}{}}},
		input: "type=SYSCALL msg=audit(1364481363.243:24287): arch=c000003e syscall=2 success=yes exit=0\n" +
			"type=CWD msg=audit(1364481363.243:24287): cwd=\"/home/user\"\n" +
			"type=PATH msg=audit(1364481363.243:24287): item=0 name=\"/etc/passwd\"\n",
		expected: []string{
			`type=SYSCALL msg=audit(1364481363.243:24287): arch=c000003e syscall=2 success=yes exit=0`,
			`type=CWD msg=audit(1364481363.243:24287): cwd="/home/user"`,
			`type=PATH msg=audit(1364481363.243:24287): item=0 name="/etc/passwd"`,
		},
	})
	requiredReuseParsers = append(requiredReuseParsers, "auditd")
}
