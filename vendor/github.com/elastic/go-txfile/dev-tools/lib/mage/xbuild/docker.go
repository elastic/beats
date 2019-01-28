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

package xbuild

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

// DockerImage provides based on downloadable docker images.
type DockerImage struct {
	Image   string
	Workdir string
	Volumes map[string]string
	Env     map[string]string
}

// Build pulls the required image.
func (p DockerImage) Build() error {
	return sh.Run("docker", "pull", p.Image)
}

// Run executes the command in a temporary docker container. The container is
// deleted after its execution.
func (p DockerImage) Run(env map[string]string, cmdAndArgs ...string) error {
	spec := []string{"run", "--rm", "-i", "-t"}
	for k, v := range mergeEnv(p.Env, env) {
		spec = append(spec, "-e", fmt.Sprintf("%v=%v", k, v))
	}
	for k, v := range p.Volumes {
		spec = append(spec, "-v", fmt.Sprintf("%v:%v", k, v))
	}
	if w := p.Workdir; w != "" {
		spec = append(spec, "-w", w)
	}

	spec = append(spec, p.Image)
	for _, v := range cmdAndArgs {
		if v != "" {
			spec = append(spec, v)
		}
	}

	return sh.RunV("docker", spec...)
}

func mergeEnv(a, b map[string]string) map[string]string {
	merged := make(map[string]string, len(a)+len(b))
	copyEnv(merged, a)
	copyEnv(merged, b)
	return merged
}

func copyEnv(to, from map[string]string) {
	for k, v := range from {
		to[k] = v
	}
}
