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

// Package perfmon implements a Metricbeat metricset for reading Windows
// performance counters.
package perfmon

//go:generate go run mkpdh_defs.go
//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_pdh_windows.go" -goarch amd64 -output defs_pdh_windows_amd64.go
//go:generate go run ../run.go -cmd "go tool cgo -godefs defs_pdh_windows.go" -goarch 386 -output defs_pdh_windows_386.go
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zpdh_windows.go pdh_windows.go
//go:generate gofmt -w defs_pdh_windows_amd64.go defs_pdh_windows_386.go zpdh_windows.go
