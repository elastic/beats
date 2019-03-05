// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build windows

package windows

const (
	// This process access rights are missing from Go's syscall package as of 1.10.3

	// PROCESS_VM_READ right allows to read memory from the target process.
	PROCESS_VM_READ = 0x10

	// PROCESS_QUERY_LIMITED_INFORMATION right allows to access a subset of the
	// information granted by PROCESS_QUERY_INFORMATION. Not available in XP
	// and Server 2003.
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
)
