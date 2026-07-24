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

package systemtests

import (
	"os"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// DockerTestResolver is a resolver meant for use with the containerized system tests.
// The logic here is extremely simple: if USE_HOSTFS is set, return that for the resolver
func DockerTestResolver(logger *logp.Logger) resolve.Resolver {
	if path, set := os.LookupEnv("HOSTFS"); set {
		logger.Infof("Using %s for container tests", path)
		return resolve.NewTestResolver(path)
	}
	return resolve.NewTestResolver("/")
}
