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

type ProjectSource struct {
	Content         string `config:"content" json:"content"`
	TargetDirectory string
}

var ErrNoContent = fmt.Errorf("no 'content' value specified for project monitor source")

func (p *ProjectSource) Validate() error {
	if !regexp.MustCompile(`\S`).MatchString(p.Content) {
		return ErrNoContent
	}

	return nil
}

func (p *ProjectSource) Fetch() error {
	decodedBytes, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return err
	}

	tf, err := ioutil.TempFile(os.TempDir(), "elastic-synthetics-zip-")
	if err != nil {
		return fmt.Errorf("could not create tmpfile for project monitor source: %w", err)
	}
	defer os.Remove(tf.Name())

	// copy the encoded contents in to a temp file for unzipping later
	_, err = io.Copy(tf, bytes.NewReader(decodedBytes))
	if err != nil {
		return err
	}

	p.TargetDirectory, err = ioutil.TempDir(os.TempDir(), "elastic-synthetics-unzip-")
	if err != nil {
		return fmt.Errorf("could not make temp dir for unzipping project source: %w", err)
	}

	err = unzip(tf, p.Workdir(), "")
	if err != nil {
		p.Close()
		return err
	}

	// Offline is not required for project resources as we are only linking
	// to the globally installed agent, but useful for testing purposes
	if !Offline() {
		// set up npm project and ensure synthetics is installed
		err = setupProjectDir(p.Workdir())
		if err != nil {
			return fmt.Errorf("setting up project dir failed: %w", err)
		}
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
// baked in to the Heartbeat image to maintain compatibility and
// allows us to control the synthetics agent version
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
		Name:    "project-journey",
		Private: true,
		Dependencies: mapstr.M{
			"@elastic/synthetics": symlinkPath,
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

	// setup the project linking to the global synthetics library
	return runSimpleCommand(exec.Command("npm", "install"), workdir)
}

func (p *ProjectSource) Workdir() string {
	return p.TargetDirectory
}

func (p *ProjectSource) Close() error {
	if p.TargetDirectory != "" {
		return os.RemoveAll(p.TargetDirectory)
	}
	return nil
}
