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
	osqueryVersion         = "4.9.0"
	osqueryMSIExt          = ".msi"
	osqueryPkgExt          = ".pkg"

	osqueryDistroDarwinSHA256   = "3f9ab772596f4da69687a2d7db9a382535b5eabf2346abd452b24666b8f25102"
	osqueryDistroLinuxSHA256    = "4187f5b76e21ce96f765e509f96cb04708c43596b7609eba0e5a10ab5fdf58c5"
	osqueryDistroLinuxARMSHA256 = "f3f5f3d6d81d727aad52a50b9f252aea3d7200add643eca06e9f462e66daeb18"
	osqueryDistroWindowsSHA256  = "ae5e8b5948f3e2783aadc66e8b9d5d417b8606b39abf79f06af466c3455ce249"
)

type OSArch struct {
	OS   string
	Arch string
}

func (o OSArch) String() string {
	return o.OS + ":" + o.Arch
}

func OsquerydVersion() string {
	return osqueryVersion
}

func GetDataInstallDir(osarch OSArch) string {
	return filepath.Join(DataInstallDir, osarch.OS, osarch.Arch)
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

var specs = map[OSArch]Spec{
	{"linux", "amd64"}:   {"_1.linux_x86_64.tar.gz", osqueryDistroLinuxSHA256, true},
	{"linux", "arm64"}:   {"_1.linux_aarch64.tar.gz", osqueryDistroLinuxARMSHA256, true},
	{"darwin", "amd64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, false},
	{"windows", "amd64"}: {osqueryMSIExt, osqueryDistroWindowsSHA256, false},
}

func GetSpec(osarch OSArch) (spec Spec, err error) {
	if spec, ok := specs[osarch]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%v: %w", osarch, ErrUnsupportedOS)
}
