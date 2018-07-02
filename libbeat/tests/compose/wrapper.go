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

package compose

import (
	"context"
	"os/exec"
)

type wrapperDriver struct {
	Name  string
	Files []string
}

func (d *wrapperDriver) LockFile() string {
	return d.Files[0] + ".lock"
}

func (d *wrapperDriver) cmd(ctx context.Context, arg ...string) *exec.Cmd {
	var files []string
	for _, f := range d.Files {
		files = append(files, "-f", f)
	}
	args := append(files, arg...)
	return exec.CommandContext(ctx, "docker-compose", args...)
}

func (d *wrapperDriver) Up(ctx context.Context, opts UpOptions, service string) error {
	return nil
}

func (d *wrapperDriver) Kill(ctx context.Context, signal string, service string) error {
	return nil
}

func (d *wrapperDriver) Ps(ctx context.Context, filter ...string) ([]map[string]string, error) {
	return nil, nil
}

func (d *wrapperDriver) Containers(ctx context.Context, projectFilter Filter, filter ...string) ([]string, error) {
	return nil, nil
}
