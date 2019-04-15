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

package export

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/idxmgmt"

	"github.com/elastic/beats/libbeat/version"

	"github.com/elastic/beats/libbeat/common"
)

func fatalf(msg string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, vs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func fatalfInitCmd(err error) {
	fatalf("Failed to initialize 'export' command: %+v.", err)
}

func newIdxmgmtClient(dir string, version string) idxmgmt.FileClient {
	if dir == "" {
		return newStdoutClient(version)
	}
	c, err := newFileClient(dir, version)
	if err != nil {
		fatalf("Error creating directory: %+v.", err)
	}
	return c
}

type stdoutClient struct {
	v common.Version
	f *os.File
}

func newStdoutClient(v string) *stdoutClient {
	if v == "" {
		v = version.GetDefaultVersion()
	}
	return &stdoutClient{v: *common.MustNewVersion(v), f: os.Stdout}
}

func (c *stdoutClient) GetVersion() common.Version {
	return c.v
}

func (c *stdoutClient) Write(_ string, body string) error {
	c.f.WriteString(body)
	return nil
}

type fileClient struct {
	v common.Version
	d string
}

func newFileClient(dir string, v string) (*fileClient, error) {
	if v == "" {
		v = version.GetDefaultVersion()
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(d, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return &fileClient{v: *common.MustNewVersion(v), d: d}, nil
}

func (c *fileClient) GetVersion() common.Version {
	return c.v
}

func (c *fileClient) Write(name string, body string) error {
	f, err := os.Create(filepath.Join(c.d, fmt.Sprintf("%s.json", name)))
	defer f.Close()
	if err != nil {
		return err
	}
	f.WriteString(body)
	return nil
}
