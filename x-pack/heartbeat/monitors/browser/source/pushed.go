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
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type PushedSource struct {
	Content         string `config:"content" json:"content"`
	TargetDirectory string
}

var ErrNoContent = fmt.Errorf("no 'content' value specified for pushed monitor source")

func (p *PushedSource) Validate() error {
	if !regexp.MustCompile(`\S`).MatchString(p.Content) {
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
	_, err = io.Copy(tf, bytes.NewReader(decodedBytes))
	if err != nil {
		return err
	}

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
	Name         string   `json:"name"`
	Private      bool     `json:"private"`
	Dependencies mapstr.M `json:"dependencies"`
}

// setupProjectDir sets ups the required package.json file and
// links the synthetics dependency to the globally installed one that is
// baked in to the HB to maintain compatibility and allows us to control the
// agent version
func setupProjectDir(workdir string) error {
	fname, err := exec.LookPath("elastic-synthetics")
	if err == nil {
		fname, err = filepath.Abs(fname)
	}
	if err != nil {
		return fmt.Errorf("cannot resolve global synthetics library: %w", err)
	}

	globalPath := strings.Replace(fname, "bin/elastic-synthetics", "lib/node_modules/@elastic/synthetics", 1)
	symlinkPath := fmt.Sprintf("file:%s", globalPath)
	pkgJson := PackageJSON{
		Name:    "pushed-journey",
		Private: true,
		Dependencies: mapstr.M{
			"@elastic/synthetics": symlinkPath,
		},
	}
	pkgJsonContent, err := json.MarshalIndent(pkgJson, "", "  ")
	if err != nil {
		return err
	}
	//nolint:gosec //for permission
	err = ioutil.WriteFile(filepath.Join(workdir, "package.json"), pkgJsonContent, 0755)
	if err != nil {
		return err
	}

	// setup the project linking to the global synthetics library
	return runSimpleCommand(exec.Command("npm", "link"), workdir)
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
