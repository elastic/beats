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

package sysinit

import (
	"os"
	"path/filepath"

	"github.com/elastic/gosigar"
)

func InitModule(config string) {
	configureHostFS(config)
}

func configureHostFS(config string) {
	dir := config
	// Set environment variables for gopsutil.
	os.Setenv("HOST_PROC", filepath.Join(dir, "/proc"))
	os.Setenv("HOST_SYS", filepath.Join(dir, "/sys"))
	os.Setenv("HOST_ETC", filepath.Join(dir, "/etc"))

	// Set proc location for gosigar.
	gosigar.Procd = filepath.Join(dir, "/proc")
}
