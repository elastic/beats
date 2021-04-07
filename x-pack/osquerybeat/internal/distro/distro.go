// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package distro

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	ErrUnsupportedOS = errors.New("unsupported OS")
)

var (
	DataDir        = filepath.Clean("build/data")
	DataInstallDir = filepath.Join(DataDir, "install")
	DataCacheDir   = filepath.Join(DataDir, "cache")
)

const (
	osqueryDownloadBaseURL = "https://pkg.osquery.io"
	osqueryName            = "osquery"
	osqueryDName           = "osqueryd"
	osqueryPath            = "usr/local/bin"
	osqueryVersion         = "4.7.0"
	osqueryMSIExt          = ".msi"
	osqueryPkgExt          = ".pkg"

	osqueryDistroDarwinSHA256  = "31244705a497f7b33eaee6b4995cea9a4b55a3b9b0f20ea4bab400ff8798cbb4"
	osqueryDistroLinuxSHA256   = "2086b1e2bf47b25a5eb64e35d516f222b2bd1c50610a71916ebb29af9d0ec210"
	osqueryDistroWindowsSHA256 = "54a98345e7f5ad6819f5516e7f340795cf42b83f4fda221c4a10bfd83f803758"
)

func OsquerydVersion() string {
	return osqueryVersion
}

func OsquerydFilename() string {
	if runtime.GOOS == "windows" {
		return osqueryDName + ".exe"
	}
	return osqueryDName
}

func OsquerydPath(dir string) string {
	return filepath.Join(dir, OsquerydFilename())
}

func OsquerydDistroPath() string {
	return OsquerydPath(osqueryPath)
}

func OsquerydDistroFilename() string {
	return OsquerydDistroPlatformFilename(runtime.GOOS)
}

func OsquerydDistroPlatformFilename(platform string) string {
	switch platform {
	case "windows":
		return osqueryName + "-" + osqueryVersion + osqueryMSIExt
	case "darwin":
		return osqueryName + "-" + osqueryVersion + osqueryPkgExt
	}
	return OsquerydFilename()
}

type Spec struct {
	PackSuffix string
	SHA256Hash string
	Extract    bool
}

func (s Spec) DistroFilename() string {
	return osqueryName + "-" + osqueryVersion + s.PackSuffix
}

func (s Spec) DistroFilepath(dir string) string {
	return filepath.Join(dir, s.DistroFilename())
}

func (s Spec) InstalledFilename() string {
	if s.Extract {
		return osqueryDName
	}
	return s.DistroFilename()
}

func (s Spec) InstalledMode() os.FileMode {
	if s.Extract {
		return 0755
	}
	return 0644
}

func (s Spec) URL(osname string) string {
	return osqueryDownloadBaseURL + "/" + osname + "/" + s.DistroFilename()
}

var specs = map[string]Spec{
	"linux":   {"_1.linux_x86_64.tar.gz", osqueryDistroLinuxSHA256, true},
	"darwin":  {osqueryPkgExt, osqueryDistroDarwinSHA256, false},
	"windows": {osqueryMSIExt, osqueryDistroWindowsSHA256, false},
}

func GetSpec(osname string) (spec Spec, err error) {
	if spec, ok := specs[osname]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%s: %w", osname, ErrUnsupportedOS)
}
