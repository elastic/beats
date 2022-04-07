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

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/idxmgmt"
	"github.com/elastic/beats/v8/libbeat/version"
)

type stdoutClient struct {
	ver common.Version
	f   *os.File
}

type fileClient struct {
	ver common.Version
	dir string
}

func newIdxmgmtClient(dir string, ver string) idxmgmt.FileClient {
	if dir == "" {
		c, err := newStdoutClient(ver)
		if err != nil {
			fatalf("Error creating stdout writer: %+v.", err)
		}
		return c
	}
	c, err := newFileClient(dir, ver)
	if err != nil {
		fatalf("Error creating directory: %+v.", err)
	}
	return c
}

func newStdoutClient(ver string) (*stdoutClient, error) {
	if ver == "" {
		ver = version.GetDefaultVersion()
	}
	v, err := common.NewVersion(ver)
	if err != nil {
		return nil, err
	}
	return &stdoutClient{ver: *v, f: os.Stdout}, nil
}

func newFileClient(dir string, ver string) (*fileClient, error) {
	if ver == "" {
		ver = version.GetDefaultVersion()
	}
	path, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("Writing to directory %s", path))
	return &fileClient{ver: *common.MustNewVersion(ver), dir: path}, nil
}

func (c *stdoutClient) GetVersion() common.Version {
	return c.ver
}

func (c *stdoutClient) Write(_ string, _ string, body string) error {
	_, err := c.f.WriteString(body)
	return err
}

func (c *fileClient) GetVersion() common.Version {
	return c.ver
}

func (c *fileClient) Write(component string, name string, body string) error {
	path := filepath.Join(c.dir, component)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(path, fmt.Sprintf("%s.json", name)))
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = f.WriteString(body)
	return err
}

func fatalf(msg string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, vs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func fatalfInitCmd(err error) {
	fatalf("Failed to initialize 'export' command: %+v.", err)
}
