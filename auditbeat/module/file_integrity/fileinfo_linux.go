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

package file_integrity

import (
<<<<<<< HEAD:auditbeat/module/file_integrity/fileinfo_linux.go
	"syscall"
	"time"
)

func fileTimes(stat *syscall.Stat_t) (atime, mtime, ctime time.Time) {
	return time.Unix(0, stat.Atim.Nano()).UTC(),
		time.Unix(0, stat.Mtim.Nano()).UTC(),
		time.Unix(0, stat.Ctim.Nano()).UTC()
=======
	"context"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
)

type Logouter interface {
	Logout(ctx context.Context) error
}

// Logout performs log out on vSphere API client with backoff retry.
func Logout(ctx context.Context, client Logouter) error {
	r := backoff.NewRetryer(3, 500*time.Millisecond, 1*time.Minute)
	return r.Retry(ctx, func() error {
		return client.Logout(ctx)
	})
>>>>>>> 6b6941eed ([gcp] Add metadata cache (#44432)):metricbeat/module/vsphere/client/logout.go
}
