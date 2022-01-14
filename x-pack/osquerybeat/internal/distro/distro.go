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
	osqueryDarwinApp       = "osquery.app"
	osqueryDarwinPath      = "opt/osquery/lib/" + osqueryDarwinApp

	osqueryLinuxPath = "opt/osquery/bin"
	osqueryVersion   = "5.0.1"
	osqueryMSIExt    = ".msi"
	osqueryPkgExt    = ".pkg"

	osqueryDistroDarwinSHA256   = "ec58996e64637d861ccead8dc6bc8865662728f6e5bc2694a3c92f0f4a371095"
	osqueryDistroLinuxSHA256    = "acac95714d388f02d5f417b0aaf86de7dbb8f6b3788340a6f8517ee2cd314235"
	osqueryDistroLinuxARMSHA256 = "712b704036929df14cbe5d3e41bd4e0ae325e698296691763af46dc0d6e77394"
	osqueryDistroWindowsSHA256  = "e0d01f56e0739a0ce2b3beb03ecea277ed146754884e225cba45083043442acc"
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

func OsquerydFilenameForOS(os string) string {
	if os == "windows" {
		return osqueryDName + ".exe"
	}
	return osqueryDName
}

func OsquerydFilename() string {
	return OsquerydFilenameForOS(runtime.GOOS)
}

func OsquerydDarwinApp() string {
	return osqueryDarwinApp
}

func OsquerydPathForOS(os, dir string) string {
	return filepath.Join(dir, OsquerydFilenameForOS(os))
}

func OsquerydPath(dir string) string {
	return OsquerydPathForOS(runtime.GOOS, dir)
}

func OsquerydDarwinDistroPath() string {
	return osqueryDarwinPath
}

func OsquerydLinuxDistroPath() string {
	return OsquerydPath(osqueryLinuxPath)
}

func OsquerydDistroFilename() string {
	return OsquerydDistroPlatformFilename(runtime.GOOS)
}

func OsquerydDistroPlatformFilename(platform string) string {
	switch platform {
	case "windows":
		return OsquerydFilenameForOS(platform)
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
