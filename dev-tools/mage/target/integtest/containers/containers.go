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

package containers

import (
	"fmt"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/magefile/mage/mg"
)

type IntegTest mg.Namespace

// BuildDockerComposeImages builds the integration test containers.
func (IntegTest) BuildContainers() error {
	return devtools.BuildIntegTestContainers()
}

// StartIntegTestContainers starts the integration test containers, waits until they are healthy, and leaves them in the background.
func (IntegTest) StartContainers() error {
	return devtools.StartIntegTestContainers()
}

// StopIntegTestContainers stops the containers started by StartIntegTestContainers.
func (IntegTest) StopContainers() error {
	return devtools.StopIntegTestContainers()
}

// PrintIntegTestComposeProject prints the compose project name used by the integ test docker-compose project.
// Pass this to docker-compose with the -p option to interact with running containers.
func (IntegTest) PrintComposeProject() {
	fmt.Println(devtools.DockerComposeProjectName())
}
