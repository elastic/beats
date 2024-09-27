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

package mage

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

type dockerBuilder struct {
	PackageSpec

	imageName string
	buildDir  string
	beatDir   string
}

func newDockerBuilder(spec PackageSpec) (*dockerBuilder, error) {
	imageName, err := spec.ImageName()
	if err != nil {
		return nil, fmt.Errorf("failed to get image name: %w", err)
	}

	buildDir := filepath.Join(spec.packageDir, "docker-build")
	beatDir := filepath.Join(buildDir, "beat")

	return &dockerBuilder{
		PackageSpec: spec,
		imageName:   imageName,
		buildDir:    buildDir,
		beatDir:     beatDir,
	}, nil
}

func (b *dockerBuilder) Build() error {
	if err := os.RemoveAll(b.buildDir); err != nil {
		return fmt.Errorf("failed to clean existing build directory %s: %w", b.buildDir, err)
	}

	if err := b.copyFiles(); err != nil {
		return fmt.Errorf("failed to copy files: %w", err)
	}

	if err := b.prepareBuild(); err != nil {
		return fmt.Errorf("failed to prepare build: %w", err)
	}

	tag, err := b.dockerBuild()

	const maxRetries = 3
	const retryInterval = 10 * time.Second

	for retries := 0; err != nil && retries < maxRetries; retries++ {
		fmt.Println(">> Building docker images again (after 10 s)")
		time.Sleep(retryInterval)
		tag, err = b.dockerBuild()
	}
	if err != nil {
		return fmt.Errorf("failed to build docker: %w", err)
	}

	if err := b.dockerSave(tag); err != nil {
		return fmt.Errorf("failed to save docker as artifact: %w", err)
	}

	return nil
}

func (b *dockerBuilder) modulesDirs() []string {
	var modulesd []string
	for _, f := range b.Files {
		if f.Modules {
			modulesd = append(modulesd, f.Target)
		}
	}
	return modulesd
}

func (b *dockerBuilder) exposePorts() []string {
	if ports, _ := b.ExtraVars["expose_ports"]; ports != "" {
		return strings.Split(ports, ",")
	}
	return nil
}

func (b *dockerBuilder) copyFiles() error {
	for _, f := range b.Files {
		target := filepath.Join(b.beatDir, f.Target)
		if err := Copy(f.Source, target); err != nil {
			if f.SkipOnMissing && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to copy from %s to %s: %w", f.Source, target, err)
		}
	}
	return nil
}

func (b *dockerBuilder) prepareBuild() error {
	elasticBeatsDir, err := ElasticBeatsDir()
	if err != nil {
		return fmt.Errorf("failed to get ElasticBeatsDir: %w", err)
	}
	templatesDir := filepath.Join(elasticBeatsDir, "dev-tools/packaging/templates/docker")

	data := map[string]interface{}{
		"ExposePorts": b.exposePorts(),
		"ModulesDirs": b.modulesDirs(),
	}

	err = filepath.Walk(templatesDir, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() && !isDockerFile(path) {
			target := strings.TrimSuffix(
				filepath.Join(b.buildDir, filepath.Base(path)),
				".tmpl",
			)

			if err := b.ExpandFile(path, target, data); err != nil {
				return fmt.Errorf("expanding template '%s' to '%s': %w", path, target, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk templates directory: %w", err)
	}

	return b.expandDockerfile(templatesDir, data)
}

func isDockerFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "Dockerfile") || strings.HasPrefix(base, "docker-entrypoint")
}

func (b *dockerBuilder) expandDockerfile(templatesDir string, data map[string]interface{}) error {
	dockerfile := "Dockerfile.tmpl"
	if f, found := b.ExtraVars["dockerfile"]; found {
		dockerfile = f
	}

	entrypoint := "docker-entrypoint.tmpl"
	if e, found := b.ExtraVars["docker_entrypoint"]; found {
		entrypoint = e
	}

	files := []struct {
		source string
		target string
	}{
		{dockerfile, "Dockerfile.tmpl"},
		{entrypoint, "docker-entrypoint.tmpl"},
	}

	for _, file := range files {
		target := strings.TrimSuffix(
			filepath.Join(b.buildDir, file.target),
			".tmpl",
		)
		path := filepath.Join(templatesDir, file.source)
		if err := b.ExpandFile(path, target, data); err != nil {
			return fmt.Errorf("expanding template '%s' to '%s': %w", path, target, err)
		}
	}

	return nil
}

func (b *dockerBuilder) dockerBuild() (string, error) {
	tag := fmt.Sprintf("%s:%s", b.imageName, b.Version)
	if b.Snapshot {
		tag += "-SNAPSHOT"
	}
	if repository, _ := b.ExtraVars["repository"]; repository != "" {
		tag = fmt.Sprintf("%s/%s", repository, tag)
	}
	return tag, sh.Run("docker", "build", "-t", tag, b.buildDir)
}

func (b *dockerBuilder) dockerSave(tag string) error {
	if err := os.MkdirAll(distributionsDir, 0750); err != nil {
		return fmt.Errorf("cannot create folder for docker artifacts: %w", err)
	}

	// Save the container as artifact
	outputFile := b.OutputFile
	if outputFile == "" {
		outputTar, err := b.Expand(defaultBinaryName+".docker.tar.gz", map[string]interface{}{
			"Name": b.imageName,
		})
		if err != nil {
			return fmt.Errorf("failed to expand output file name: %w", err)
		}
		outputFile = filepath.Join(distributionsDir, outputTar)
	}

	var stderr bytes.Buffer
	cmd := exec.Command("docker", "save", tag)
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker save command: %w", err)
	}

	if err := func() error {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()

		w := gzip.NewWriter(f)
		defer w.Close()

		if _, err = io.Copy(w, stdout); err != nil {
			return fmt.Errorf("failed to copy docker save output: %w", err)
		}
		return nil
	}(); err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		if errmsg := strings.TrimSpace(stderr.String()); errmsg != "" {
			err = fmt.Errorf("%w: %s", err, errmsg)
		}
		return fmt.Errorf("docker save command failed: %w", err)
	}

	if err = CreateSHA512File(outputFile); err != nil {
		return fmt.Errorf("failed to create .sha512 file: %w", err)
	}
	return nil
}
