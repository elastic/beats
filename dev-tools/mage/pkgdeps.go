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

	"github.com/magefile/mage/sh"
)

type PackageInstaller struct {
	table map[PlatformDescription][]PackageDependency
}

type PlatformDescription struct {
	Name       string
	Arch       string
	DefaultTag string
}

type PackageDependency struct {
	archTag      string
	dependencies []string
}

var (
	LinuxAMD64    = PlatformDescription{Name: "linux/amd64", Arch: "", DefaultTag: ""} // builders run on amd64 platform
	LinuxARM64    = PlatformDescription{Name: "linux/arm64", Arch: "arm64", DefaultTag: "arm64"}
	LinuxARM5     = PlatformDescription{Name: "linux/arm5", Arch: "armel", DefaultTag: "armel"}
	LinuxARM6     = PlatformDescription{Name: "linux/arm6", Arch: "armel", DefaultTag: "armel"}
	LinuxARM7     = PlatformDescription{Name: "linux/arm7", Arch: "armhf", DefaultTag: "armhf"}
	LinuxMIPS     = PlatformDescription{Name: "linux/mips", Arch: "mips", DefaultTag: "mips"}
	LinuxMIPSLE   = PlatformDescription{Name: "linux/mipsle", Arch: "mipsel", DefaultTag: "mipsel"}
	LinuxMIPS64LE = PlatformDescription{Name: "linux/mips64le", Arch: "mips64el", DefaultTag: "mips64el"}
	LinuxPPC64LE  = PlatformDescription{Name: "linux/ppc64le", Arch: "ppc64el", DefaultTag: "ppc64el"}
	LinuxS390x    = PlatformDescription{Name: "linux/s390x", Arch: "s390x", DefaultTag: "s390x"}
)

func NewPackageInstaller() *PackageInstaller {
	return &PackageInstaller{}
}

func (i *PackageInstaller) AddEach(ps []PlatformDescription, names ...string) *PackageInstaller {
	for _, p := range ps {
		i.Add(p, names...)
	}
	return i
}

func (i *PackageInstaller) Add(p PlatformDescription, names ...string) *PackageInstaller {
	i.AddPackages(p, p.Packages(names...))
	return i
}

func (i *PackageInstaller) AddPackages(p PlatformDescription, details ...PackageDependency) *PackageInstaller {
	if i.table == nil {
		i.table = map[PlatformDescription][]PackageDependency{}
	}
	i.table[p] = append(i.table[p], details...)
	return i
}

func (i *PackageInstaller) Installer(name string) func() error {
	var platform PlatformDescription
	for p := range i.table {
		if p.Name == name {
			platform = p
		}
	}

	if platform.Name == "" {
		return func() error { return nil }
	}

	return func() error {
		return i.Install(platform)
	}
}

func (i *PackageInstaller) Install(p PlatformDescription) error {
	packages := map[string]struct{}{}
	for _, details := range i.table[p] {
		for _, name := range details.List() {
			packages[name] = struct{}{}
		}
	}

	j, lst := 0, make([]string, len(packages))
	for name := range packages {
		lst[j], j = name, j+1
	}

	return installDependencies(p.Arch, lst...)
}

func installDependencies(arch string, pkgs ...string) error {
	if arch != "" {
		err := sh.Run("dpkg", "--add-architecture", arch)
		if err != nil {
			return fmt.Errorf("error while adding architecture: %w", err)
		}
	}

	if err := sh.Run("apt-get", "update"); err != nil {
		return err
	}

	params := append([]string{"install", "-y",
		"--no-install-recommends",

		// Journalbeat is built with old versions of Debian that don't update
		// their repositories, so they have expired keys.
		// Allow unauthenticated packages.
		// This was not enough: "-o", "Acquire::Check-Valid-Until=false",
		"--allow-unauthenticated",
	}, pkgs...)
	return sh.Run("apt-get", params...)
}

func (p PlatformDescription) Packages(names ...string) PackageDependency {
	return PackageDependency{}.WithTag(p.DefaultTag).Add(names...)
}

func (p PackageDependency) Add(deps ...string) PackageDependency {
	if len(deps) == 0 {
		return p
	}

	// always copy to ensure that we never share or overwrite slices due to capacity being too large
	p.dependencies = append(make([]string, 0, len(p.dependencies)+len(deps)), p.dependencies...)
	p.dependencies = append(p.dependencies, deps...)
	return p
}

func (p PackageDependency) WithTag(tag string) PackageDependency {
	p.archTag = tag
	return p
}

func (p PackageDependency) List() []string {
	if p.archTag == "" {
		return p.dependencies
	}

	names := make([]string, len(p.dependencies))
	for i, name := range p.dependencies {
		names[i] = fmt.Sprintf("%v:%v", name, p.archTag)
	}
	return names
}
