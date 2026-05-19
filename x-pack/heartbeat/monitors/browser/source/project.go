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

type ProjectSource struct {
	Content         string `config:"content" json:"content"`
	TargetDirectory string
	fetched         bool
	mtx             sync.Mutex
	log             *logp.Logger
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
		if p.log != nil {
			p.log.Debugf("browser project: re-use already unpacked source: %s", p.Workdir())
		}
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

	if p.log != nil {
		p.log.Debugf("browser project: unpack source: %s", p.Workdir())
	}

	err = os.Chmod(p.TargetDirectory, defaultMod)
	if err != nil {
		return fmt.Errorf("failed assigning default mode %s to temp dir: %w", defaultMod, err)
	}

	err = unzip(tf, p.Workdir(), "")
	if err != nil {
		p.Close()
		return err
	}

	// set up npm project and ensure synthetics is installed;
	// npm install is skipped in offline mode (testing) but package.json is always created
	err = setupProjectDir(context.Background(), p.log, p.Workdir())
	if err != nil {
		return fmt.Errorf("setting up project dir failed: %w", err)
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

// findNPMPath locates npm, checking common macOS paths because launchd services omit /usr/local/bin and /opt/homebrew/bin from PATH.
func findNPMPath() (string, error) {
	if path, err := exec.LookPath("npm"); err == nil {
		return path, nil
	}
	for _, candidate := range []string{"/usr/local/bin/npm", "/opt/homebrew/bin/npm"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("npm not found in PATH or common locations (/usr/local/bin, /opt/homebrew/bin)")
}

// setupProjectDir sets ups the required package.json file and
// links the synthetics dependency to the globally installed one that is
// baked in to the Heartbeat image to maintain compatibility and
// allows us to control the synthetics agent version
func setupProjectDir(ctx context.Context, log *logp.Logger, workdir string) error {
	if Offline() {
		// In offline mode (testing) write a minimal package.json and skip all npm operations.
		return writePackageJSON(workdir, "file:offline")
	}

	npmPath, err := findNPMPath()
	if err != nil {
		return err
	}

	out, err := exec.CommandContext(ctx, npmPath, "root", "-g").CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot resolve global npm root: %w: %s", err, strings.TrimSpace(string(out)))
	}

	globalPath := filepath.Join(strings.TrimSpace(string(out)), "@elastic", "synthetics")
	if _, err := os.Stat(globalPath); err != nil {
		return fmt.Errorf("global synthetics package not found at %s: %w", globalPath, err)
	}

	if err := writePackageJSON(workdir, fmt.Sprintf("file:%s", globalPath)); err != nil {
		return err
	}

	// link the project to the globally installed synthetics library
	return runSimpleCommand(log,
		exec.CommandContext(
			ctx,
			npmPath, "install",
			"--no-audit",           // Prevent audit checks that require internet
			"--no-update-notifier", // Prevent update checks that require internet
			"--no-fund",            // No need for package funding messages here
			"--package-lock=false", // no need to write package lock here
			"--progress=false",     // no need to display progress
		), workdir)
}

func writePackageJSON(workdir, symlinkPath string) error {
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
	if err := os.WriteFile(pkgFile, pkgJsonContent, defaultMod); err != nil {
		return err
	}
	if err := os.Chmod(pkgFile, defaultMod); err != nil { // Double tap because of umask
		return fmt.Errorf("failed assigning default mode %s to package.json: %w", defaultMod, err)
	}
	return nil
}

func (p *ProjectSource) Workdir() string {
	return p.TargetDirectory
}

func (p *ProjectSource) Close() error {
	if p.log != nil {
		p.log.Debugf("browser project: close project source: %s", p.Workdir())
	}

	if p.TargetDirectory != "" {
		return os.RemoveAll(p.TargetDirectory)
	}
	return nil
}

func runSimpleCommand(log *logp.Logger, cmd *exec.Cmd, dir string) error {
	cmd.Dir = dir
	if log != nil {
		log.Infof("Running %s in %s", cmd, dir)
	}
	output, err := cmd.CombinedOutput()
	if log != nil {
		if cmd.ProcessState != nil {
			log.Infof("Ran %s (%d) got '%s': (%v) as (%d/%d)", cmd, cmd.ProcessState.ExitCode(), string(output), err, syscall.Getuid(), syscall.Geteuid())
		} else {
			log.Infof("Command %s did not start: (%v) as (%d/%d)", cmd, err, syscall.Getuid(), syscall.Geteuid())
		}
	}
	return err
}

func (p *ProjectSource) Decode() error {
	return nil
}
