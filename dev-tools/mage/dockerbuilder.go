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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
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
		return nil, err
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
		return errors.Wrapf(err, "failed to clean existing build directory %s", b.buildDir)
	}

	if err := b.copyFiles(); err != nil {
		return err
	}

	if err := b.prepareBuild(); err != nil {
		return errors.Wrap(err, "failed to prepare build")
	}

	tag, err := b.dockerBuild()
	tries := 3
	for err != nil && tries != 0 {
		fmt.Println(">> Building docker images again (after 10 s)")
		// This sleep is to avoid hitting the docker build issues when resources are not available.
		time.Sleep(time.Second * 10)
		tag, err = b.dockerBuild()
		tries -= 1
	}
	if err != nil {
		return errors.Wrap(err, "failed to build docker")
	}

	if err := b.dockerSave(tag); err != nil {
		return errors.Wrap(err, "failed to save docker as artifact")
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
			return errors.Wrapf(err, "failed to copy from %s to %s", f.Source, target)
		}
	}
	return nil
}

func (b *dockerBuilder) prepareBuild() error {
	elasticBeatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
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

			err = b.ExpandFile(path, target, data)
			if err != nil {
				return errors.Wrapf(err, "expanding template '%s' to '%s'", path, target)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return b.expandDockerfile(templatesDir, data)
}

func isDockerFile(path string) bool {
	path = filepath.Base(path)
	return strings.HasPrefix(path, "Dockerfile") || strings.HasPrefix(path, "docker-entrypoint")
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

	type fileExpansion struct {
		source string
		target string
	}
	for _, file := range []fileExpansion{{dockerfile, "Dockerfile.tmpl"}, {entrypoint, "docker-entrypoint.tmpl"}} {
		target := strings.TrimSuffix(
			filepath.Join(b.buildDir, file.target),
			".tmpl",
		)
		path := filepath.Join(templatesDir, file.source)
		err := b.ExpandFile(path, target, data)
		if err != nil {
			return errors.Wrapf(err, "expanding template '%s' to '%s'", path, target)
		}
	}

	return nil
}

func (b *dockerBuilder) dockerBuild() (string, error) {
	tag := fmt.Sprintf("%s:%s", b.imageName, b.Version)
	if b.Snapshot {
		tag = tag + "-SNAPSHOT"
	}
	if repository, _ := b.ExtraVars["repository"]; repository != "" {
		tag = fmt.Sprintf("%s/%s", repository, tag)
	}
	return tag, sh.Run("docker", "build", "-t", tag, b.buildDir)
}

func (b *dockerBuilder) dockerSave(tag string) error {
	if _, err := os.Stat(distributionsDir); os.IsNotExist(err) {
		err := os.MkdirAll(distributionsDir, 0750)
		if err != nil {
			return fmt.Errorf("cannot create folder for docker artifacts: %+v", err)
		}
	}
	// Save the container as artifact
	outputFile := b.OutputFile
	if outputFile == "" {
		outputTar, err := b.Expand(defaultBinaryName+".docker.tar.gz", map[string]interface{}{
			"Name": b.imageName,
		})
		if err != nil {
			return err
		}
		outputFile = filepath.Join(distributionsDir, outputTar)
	}
	var stderr bytes.Buffer
	cmd := exec.Command("docker", "save", tag)
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}

	err = func() error {
		f, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer f.Close()

		w := gzip.NewWriter(f)
		defer w.Close()

		_, err = io.Copy(w, stdout)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	if err = cmd.Wait(); err != nil {
		if errmsg := strings.TrimSpace(stderr.String()); errmsg != "" {
			err = errors.Wrap(errors.New(errmsg), err.Error())
		}
		return err
	}
	return errors.Wrap(CreateSHA512File(outputFile), "failed to create .sha512 file")
}
