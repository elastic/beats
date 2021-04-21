// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"path/filepath"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
)

func CustomizePackaging() {
	for _, args := range devtools.Packages {
		distFile := distro.OsquerydDistroPlatformFilename(args.OS)

		packFile := devtools.PackageFile{
			Mode:   0644,
			Source: filepath.Join(distro.DataInstallDir, distFile),
		}
		args.Spec.Files[distFile] = packFile
	}
}
