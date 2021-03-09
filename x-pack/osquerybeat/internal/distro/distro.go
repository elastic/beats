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
	osqueryVersion         = "4.6.0"
	osqueryMSIExt          = ".msi"
	osqueryPkgExt          = ".pkg"

	osqueryDistroDarwinSHA256  = "c037742f9f7e416955c0a38cf450e00989fad07f34aef60aba8d3a923502177c"
	osqueryDistroLinuxSHA256   = "f74319fd264e16217f676c44b9780d967d46a90289b7bc75c440ba1c62a558ee"
	osqueryDistroWindowsSHA256 = "18845659c46e7cde4e569b0b158d5158ef31eca1535a9d0174d825ae6e1c731f"
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
