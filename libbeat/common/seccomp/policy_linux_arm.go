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

package seccomp

import "github.com/elastic/go-seccomp-bpf"

func init() {
<<<<<<< HEAD:libbeat/common/seccomp/policy_linux_arm.go
	defaultPolicy = &seccomp.Policy{
		DefaultAction: seccomp.ActionAllow,
		Syscalls: []seccomp.SyscallGroup{
			{
				Action: seccomp.ActionErrno,
				Names: []string{
					"execve",
					"execveat",
					"fork",
					"vfork",
				},
			},
		},
	}
}
=======
	if err := asset.SetFields("packetbeat", "Applayer", asset.ModuleFieldsPri, AssetApplayer); err != nil {
		panic(err)
	}
}

// AssetApplayer returns asset data.
// This is the base64 encoded gzipped contents of protos/applayer.
func AssetApplayer() string {
	return "eJwBAAD//wAAAAE="
}
>>>>>>> Introduce local fields generation:packetbeat/protos/applayer/fields.go
