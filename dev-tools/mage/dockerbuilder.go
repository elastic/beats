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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"

	"github.com/pkg/errors"
)

type dockerBuilder struct {
	PackageSpec
}

func newDockerBuilder(spec PackageSpec) (*dockerBuilder, error) {
	return &dockerBuilder{
		PackageSpec: spec,
	}, nil
}

func (b *dockerBuilder) Build() error {
	buildDir := b.buildDir()
	if err := os.RemoveAll(buildDir); err != nil {
		return errors.Wrapf(err, "failed to clean existing build directory %s", buildDir)
	}

	if err := b.copyFiles(); err != nil {
		return err
	}

	if err := b.prepareBuild(); err != nil {
		return errors.Wrap(err, "failed to prepare build")
	}

	tag, err := b.dockerBuild()
	if err != nil {
		return errors.Wrap(err, "failed to build docker")
	}

	if err := b.dockerSave(tag); err != nil {
		return errors.Wrap(err, "failed to save docker as artifact")
	}

	return nil
}

func (b *dockerBuilder) buildDir() string {
	return filepath.Join(b.packageDir, "docker-build")
}

func (b *dockerBuilder) beatDir() string {
	return filepath.Join(b.buildDir(), "beat")
}

func (b *dockerBuilder) copyFiles() error {
	beatDir := b.beatDir()
	for _, f := range b.Files {
		target := filepath.Join(beatDir, f.Target)
		if err := Copy(f.Source, target); err != nil {
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
		"From":              "centos:7", // TODO: Parametrize this
		"BeatName":          b.Name,
		"Version":           b.Version,
		"Vendor":            b.Vendor,
		"License":           b.License,
		"Env":               map[string]string{},
		"LinuxCapabilities": "",
		"User":              b.Name,
	}

	buildDir := b.buildDir()
	return filepath.Walk(templatesDir, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			target := strings.TrimSuffix(
				filepath.Join(buildDir, filepath.Base(path)),
				".tmpl",
			)

			err = expandFile(path, target, data)
			if err != nil {
				return errors.Wrapf(err, "expanding template '%s' to '%s'", path, target)
			}
		}
		return nil
	})
}

func (b *dockerBuilder) dockerBuild() (string, error) {
	repository := "docker.elastic.co/beats"                                   // TODO: Parametrize this
	tag := fmt.Sprintf("%s:%s", filepath.Join(repository, b.Name), b.Version) // TODO: What about OSS?
	return tag, sh.Run("docker", "build", "-t", tag, b.buildDir())
}

func (b *dockerBuilder) dockerSave(tag string) error {
	// Save the container as artifact
	outputFile := b.OutputFile
	if outputFile == "" {
		outputTar, err := b.Expand(defaultBinaryName + ".docker.tar")
		if err != nil {
			return err
		}
		outputFile = filepath.Join(distributionsDir, outputTar)
	}
	if err := sh.Run("docker", "save", "-o", outputFile, tag); err != nil {
		return err
	}
	return errors.Wrap(CreateSHA512File(outputFile), "failed to create .sha512 file")
}
