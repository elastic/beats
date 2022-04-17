// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"os"
	"path/filepath"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/distro"
)

const defaultArch = "amd64"

func CustomizePackaging() {
	for _, args := range devtools.Packages {
		distFile := distro.OsquerydDistroPlatformFilename(args.OS)

		// The minimal change to fix the issue for 7.13
		// https://github.com/menderesk/beats/issues/25762
		// TODO: this could be moved to dev-tools/packaging/packages.yml for the next release
		var mode os.FileMode = 0644
		// If distFile is osqueryd binary then it should be executable
		if distFile == distro.OsquerydFilenameForOS(args.OS) {
			mode = 0750
		}
		arch := defaultArch
		if args.Arch != "" {
			arch = args.Arch
		}
		packFile := devtools.PackageFile{
			Mode:   mode,
			Source: filepath.Join(distro.GetDataInstallDir(distro.OSArch{OS: args.OS, Arch: arch}), distFile),
		}
		args.Spec.Files[distFile] = packFile
	}
}
