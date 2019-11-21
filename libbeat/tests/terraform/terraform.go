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

package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/stretchr/testify/suite"
)

type TerraformSuite struct {
	suite.Suite

	Dir  string
	Vars Vars
}

func (s *TerraformSuite) Output(name string) (string, error) {
	var stdout bytes.Buffer
	cmd := s.terraformCmd("output", name)
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (s *TerraformSuite) SetupSuite() {
	err := s.terraformCmd("init").Run()
	s.Require().NoError(err, "terraform init")

	vars := s.Vars.args()
	args := []string{"apply", "-auto-approve"}
	args = append(args, vars...)
	err = s.terraformCmd(args...).Run()
	s.Require().NoError(err, "terraform apply")
}

func (s *TerraformSuite) TearDownSuite() {
	// TODO: Allow to skip destroy
	vars := s.Vars.args()
	args := []string{"destroy", "-auto-approve"}
	args = append(args, vars...)
	err := s.terraformCmd(args...).Run()
	s.Require().NoError(err, "terraform destroy")
}

func (s *TerraformSuite) terraformCmd(args ...string) *exec.Cmd {
	terraform := exec.Command("terraform", args...)
	terraform.Dir = s.Dir
	// TODO: Log only if debug is enabled
	terraform.Stderr = os.Stderr
	return terraform
}

type Vars map[string]string

func (v Vars) args() (args []string) {
	for name, value := range v {
		args = append(args, fmt.Sprintf("-var=%s=%s", name, value))
	}
	return
}
