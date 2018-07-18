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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func testPackageSpec() PackageSpec {
	return PackageSpec{
		Name:     "brewbeat",
		Version:  "7.0.0",
		Snapshot: true,
		OS:       "windows",
		Arch:     "x86_64",
		Files: map[string]PackageFile{
			"brewbeat.yml": PackageFile{
				Source: "./testdata/config.yml",
				Mode:   0644,
			},
			"README.txt": PackageFile{
				Content: "Hello! {{.Version}}\n",
				Mode:    0644,
			},
		},
	}
}

func TestPackageZip(t *testing.T) {
	testPackage(t, PackageZip)
}

func TestPackageTarGz(t *testing.T) {
	testPackage(t, PackageTarGz)
}

func TestPackageRPM(t *testing.T) {
	if err := HaveDocker(); err != nil {
		t.Skip("docker is required")
	}

	testPackage(t, PackageRPM)
}

func TestPackageDeb(t *testing.T) {
	if err := HaveDocker(); err != nil {
		t.Skip("docker is required")
	}

	testPackage(t, PackageDeb)
}

func testPackage(t testing.TB, pack func(PackageSpec) error) {
	spec := testPackageSpec().Evaluate()

	readme := spec.Files["README.txt"]
	readmePath := filepath.ToSlash(filepath.Clean(readme.Source))
	assert.True(t, strings.HasPrefix(readmePath, packageStagingDir))

	if err := pack(spec); err != nil {
		t.Fatal(err)
	}
}

func TestRepoRoot(t *testing.T) {
	repo, err := GetProjectRepoInfo()
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "github.com/elastic/beats", repo.RootImportPath)
	assert.True(t, filepath.IsAbs(repo.RootDir))
	cwd := filepath.Join(repo.RootDir, repo.SubDir)
	assert.Equal(t, CWD(), cwd)
}

func TestDumpVariables(t *testing.T) {
	out, err := dumpVariables()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestLoadSpecs(t *testing.T) {
	pkgs, err := LoadSpecs("files/packages.yml")
	if err != nil {
		t.Fatal(err)
	}

	for flavor, s := range pkgs {
		out, err := yaml.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if testing.Verbose() {
			t.Log("Packaging flavor:", flavor, "\n", string(out))
		}
	}
}
