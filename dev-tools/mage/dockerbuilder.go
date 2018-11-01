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

type dockerTemplateData struct {
	BeatName          string
	Version           string
	License           string
	Env               map[string]string
	LinuxCapabilities string
	User              string
}

type dockerBuilder struct {
	PackageSpec
}

func newDockerBuilder(spec PackageSpec) (*dockerBuilder, error) {
	return &dockerBuilder{
		PackageSpec: spec,
	}, nil
}

func (b *dockerBuilder) Build() error {
	buildDir := filepath.Join(b.packageDir, "docker-build")
	beatDir := filepath.Join(buildDir, "beat")
	// TODO: defer removal of buildDir

	elasticBeatsDir, err := ElasticBeatsDir()
	if err != nil {
		return err
	}
	templatesDir := filepath.Join(elasticBeatsDir, "dev-tools/packaging/templates/docker")

	for _, f := range b.Files {
		target := filepath.Join(beatDir, f.Target)
		if err := Copy(f.Source, target); err != nil {
			return errors.Wrapf(err, "failed to copy from %s to %s", f.Source, target)
		}
	}

	/* TODO:
	data := dockerTemplateData{
		BeatName: b.Name,
		Version:  b.Version,
		License:  b.License,
	}
	*/
	data := map[string]interface{}{
		"From":              "centos:7", // TODO: Parametrize this
		"BeatName":          b.Name,
		"Version":           b.Version,
		"License":           b.License,
		"Env":               map[string]string{},
		"LinuxCapabilities": "",
		"User":              b.Name,
	}

	// TODO: Expand templates from packages.yml?
	err = filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		/* TODO: Why there is an error here?
		if err != nil {
			return err
		}
		*/
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
	if err != nil {
		return err
	}

	// TODO: Tag container on build
	repository := "docker.elastic.co/beats" // TODO: Parametrize this
	tag := fmt.Sprintf("%s/%s:%s", repository, b.Name, b.Version)
	return sh.RunCmd("docker", "build", "-t", tag, buildDir)()
}
