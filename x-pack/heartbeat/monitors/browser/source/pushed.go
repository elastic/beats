// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/elastic/beats/v7/libbeat/common"
)

type PushedSource struct {
	Content         string `config:"content" json:"content"`
	TargetDirectory string
}

var ErrNoContent = fmt.Errorf("no 'content' value specified for pushed monitor source")

func (p *PushedSource) Validate() error {
	if !regexp.MustCompile("\\S").MatchString(p.Content) {
		return ErrNoContent
	}

	return nil
}

func (p *PushedSource) Fetch() error {
	decodedBytes, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return err
	}

	tf, err := ioutil.TempFile("/tmp", "elastic-synthetics-zip-")
	if err != nil {
		return fmt.Errorf("could not create tmpfile for pushed monitor source: %w", err)
	}
	defer os.Remove(tf.Name())

	// copy the encoded contents in to a temp file for unzipping later
	io.Copy(tf, bytes.NewReader(decodedBytes))

	p.TargetDirectory, err = ioutil.TempDir("/tmp", "elastic-synthetics-unzip-")
	if err != nil {
		return fmt.Errorf("could not make temp dir for unzipping pushed source: %w", err)
	}

	err = unzip(tf, p.Workdir(), "")
	if err != nil {
		p.Close()
		return err
	}

	// set up npm project and ensure synthetics is installed
	err = setupProjectDir(p.Workdir())
	if err != nil {
		return fmt.Errorf("setting up project dir failed: %w", err)
	}

	return nil
}

type PackageJSON struct {
	Name         string        `json:"name"`
	Private      bool          `json:"private"`
	Dependencies common.MapStr `json:"dependencies"`
}

func setupProjectDir(workdir string) error {
	// TODO: Link to the globally installed synthetics version
	fname, err := exec.LookPath("elastic-synthetics")
	if err == nil {
		fname, err = filepath.Abs(fname)
	}

	pkgJson := PackageJSON{
		Name:    "pushed-journey",
		Private: true,
		Dependencies: common.MapStr{
			"@elastic/synthetics": "",
		},
	}
	pkgJsonContent, err := json.MarshalIndent(pkgJson, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(workdir, "package.json"), pkgJsonContent, 0755)
	if err != nil {
		return err
	}
	err = runSimpleCommand(exec.Command("npm", "install"), workdir)
	return err
}

func (p *PushedSource) Workdir() string {
	return p.TargetDirectory
}

func (p *PushedSource) Close() error {
	if p.TargetDirectory != "" {
		return os.RemoveAll(p.TargetDirectory)
	}
	return nil
}
