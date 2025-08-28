// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"os"
	"path/filepath"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
)

const defaultArch = "amd64"

func CustomizePackaging() {
	for _, args := range devtools.Packages {
		distFile := distro.OsquerydDistroPlatformFilename(args.OS)

		// The minimal change to fix the issue for 7.13
		// https://github.com/elastic/beats/issues/25762
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

		// If macOS bundle osquery.app, preserve the directories and files permissions
		if distFile == distro.OsquerydDarwinApp() {
			packFile.PreserveMode = true
		}

		args.Spec.Files[distFile] = packFile

		// Certs
		certsFile := devtools.PackageFile{
			Mode:   0640,
			Source: filepath.Join(distro.GetDataInstallDir(distro.OSArch{OS: args.OS, Arch: arch}), "certs", "certs.pem"),
		}

		args.Spec.Files[filepath.Join("certs", "certs.pem")] = certsFile

		// Augeas lenses are not available for Windows
		if args.OS != "windows" {
			args.Spec.Files["lenses"] = devtools.PackageFile{Source: filepath.Join(distro.GetDataInstallDir(distro.OSArch{OS: args.OS, Arch: arch}), "lenses")}
		}
	}
}
