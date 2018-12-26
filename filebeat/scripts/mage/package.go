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
	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

const (
	dirModuleGenerated   = "build/package/module"
	dirModulesDGenerated = "build/package/modules.d"
)

// CustomizePackaging modifies the package specs to add the modules and
// modules.d directory. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.
func CustomizePackaging() {
	var (
		moduleTarget = "module"
		module       = mage.PackageFile{
			Mode:   0644,
			Source: dirModuleGenerated,
		}

		modulesDTarget = "modules.d"
		modulesD       = mage.PackageFile{
			Mode:    0644,
			Source:  dirModulesDGenerated,
			Config:  true,
			Modules: true,
		}
	)

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.TarGz, mage.Zip, mage.Docker:
				args.Spec.Files[moduleTarget] = module
				args.Spec.Files[modulesDTarget] = modulesD
			case mage.Deb, mage.RPM:
				args.Spec.Files["/usr/share/{{.BeatName}}/"+moduleTarget] = module
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			case mage.DMG:
				args.Spec.Files["/Library/Application Support/{{.BeatVendor}}/{{.BeatName}}/"+moduleTarget] = module
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
			break
		}
	}
}

// PrepareModulePackagingOSS generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func PrepareModulePackagingOSS() error {
	return prepareModulePackaging([]struct{ Src, Dst string }{
		{mage.OSSBeatDir("module"), dirModuleGenerated},
		{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
	}...)
}

// PrepareModulePackagingXPack generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func PrepareModulePackagingXPack() error {
	return prepareModulePackaging([]struct{ Src, Dst string }{
		{mage.OSSBeatDir("module"), dirModuleGenerated},
		{"module", dirModuleGenerated},
		{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
		{"modules.d", dirModulesDGenerated},
	}...)
}

// prepareModulePackaging generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func prepareModulePackaging(files ...struct{ Src, Dst string }) error {
	// This depends on the modules.d directory being up-to-date.
	mg.Deps(mage.GenerateDirModulesD)

	// Clean any existing generated directories.
	if err := mage.Clean([]string{dirModuleGenerated, dirModulesDGenerated}); err != nil {
		return err
	}

	for _, copyAction := range files {
		err := (&mage.CopyTask{
			Source:  copyAction.Src,
			Dest:    copyAction.Dst,
			Mode:    0644,
			DirMode: 0755,
			Exclude: []string{
				"/_meta",
				"/test",
				"fields.go",
			},
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}
