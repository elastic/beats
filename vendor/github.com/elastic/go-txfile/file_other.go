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

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package txfile

// computePlatformMmapSize computes the maximum amount of bytes to be mmaped,
// depending on the actual file size and the configured maximum file size.
func computePlatformMmapSize(fileSize, maxSize, pageSize uint) (uint, reason) {
	return computeMmapSize(fileSize, maxSize, pageSize)
}
