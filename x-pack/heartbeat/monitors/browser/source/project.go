// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package source

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var projectLogger = logp.NewLogger("browser.project")

type ProjectSource struct {
	Content         string `config:"content" json:"content"`
	TargetDirectory string
	fetched         bool
	mtx             sync.Mutex
}

var ErrNoContent = fmt.Errorf("no 'content' value specified for project monitor source")

func (p *ProjectSource) Validate() error {
	if !regexp.MustCompile(`\S`).MatchString(p.Content) {
		return ErrNoContent
	}

	return nil
}

func (p *ProjectSource) Fetch() error {
	// We only need to unzip the source exactly once
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.fetched {
		projectLogger.Debugf("browser project: re-use already unpacked source: %s", p.Workdir())
		return nil
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return err
	}

	tf, err := os.CreateTemp(os.TempDir(), "elastic-synthetics-zip-")
	if err != nil {
		return fmt.Errorf("could not create tmpfile for project monitor source: %w", err)
	}
	defer os.Remove(tf.Name())

	// copy the encoded contents in to a temp file for unzipping later
	_, err = io.Copy(tf, bytes.NewReader(decodedBytes))
	if err != nil {
		return err
	}

	p.TargetDirectory, err = os.MkdirTemp(os.TempDir(), "elastic-synthetics-unzip-")
	if err != nil {
		return fmt.Errorf("could not make temp dir for unzipping project source: %w", err)
	}

	projectLogger.Debugf("browser project: unpack source: %s", p.Workdir())

	err = os.Chmod(p.TargetDirectory, defaultMod)
	if err != nil {
		return fmt.Errorf("failed assigning default mode %s to temp dir: %w", defaultMod, err)
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
		err = setupProjectDir(context.Background(), p.Workdir())
		if err != nil {
			return fmt.Errorf("setting up project dir failed: %w", err)
		}
	}

	// We've succeeded, mark the fetch as a success
	p.fetched = true
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
func setupProjectDir(ctx context.Context, workdir string) error {
	out, err := exec.CommandContext(ctx, "npm", "root", "-g").CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot resolve global npm root: %w: %s", err, strings.TrimSpace(string(out)))
	}

	globalPath := filepath.Join(strings.TrimSpace(string(out)), "@elastic", "synthetics")
	if _, err := os.Stat(globalPath); err != nil {
		return fmt.Errorf("global synthetics package not found at %s: %w", globalPath, err)
	}

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
	pkgFile := filepath.Join(workdir, "package.json")
	err = os.WriteFile(pkgFile, pkgJsonContent, defaultMod)
	if err != nil {
		return err
	}
	err = os.Chmod(pkgFile, defaultMod) // Double tap because of umask
	if err != nil {
		return fmt.Errorf("failed assigning default mode %s to package.json: %w", defaultMod, err)
	}

	// setup the project linking to the global synthetics library
	return runSimpleCommand(
		exec.CommandContext(
			ctx,
			"npm", "install",
			"--no-audit",           // Prevent audit checks that require internet
			"--no-update-notifier", // Prevent update checks that require internet
			"--no-fund",            // No need for package funding messages here
			"--package-lock=false", // no need to write package lock here
			"--progress=false",     // no need to display progress
		), workdir)
}

func (p *ProjectSource) Workdir() string {
	return p.TargetDirectory
}

func (p *ProjectSource) Close() error {
	projectLogger.Debugf("browser project: close project source: %s", p.Workdir())

	if p.TargetDirectory != "" {
		return os.RemoveAll(p.TargetDirectory)
	}
	return nil
}

func runSimpleCommand(cmd *exec.Cmd, dir string) error {
	cmd.Dir = dir
	projectLogger.Infof("Running %s in %s", cmd, dir)
	output, err := cmd.CombinedOutput()
	projectLogger.Infof("Ran %s (%d) got '%s': (%s) as (%d/%d)", cmd, cmd.ProcessState.ExitCode(), string(output), err, syscall.Getuid(), syscall.Geteuid())
	return err
}

func (p *ProjectSource) Decode() error {
	return nil
}
