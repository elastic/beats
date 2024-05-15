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

	osqueryCertsPEM         = "certs.pem"
	osqueryCertsPath        = "certs/" + osqueryCertsPEM
	osqueryLinuxPath        = "opt/osquery/bin"
	osqueryCertsLinuxPath   = "opt/osquery/share/osquery/certs/" + osqueryCertsPEM
	osqueryCertsDarwinPath  = "private/var/osquery/certs/" + osqueryCertsPEM
	osqueryCertsWindowsPath = "osquery/certs/" + osqueryCertsPEM

	osqueryLensesLinuxDir  = "opt/osquery/share/osquery/lenses"
	osqueryLensesDarwinDir = "private/var/osquery/lenses"

	osqueryLensesDir = "lenses"

	osqueryVersion = "5.10.2"
	osqueryMSIExt  = ".msi"
	osqueryPkgExt  = ".pkg"

	osqueryDistroDarwinSHA256   = "a01d1f7da016f1e6bed54955e97982d491b7e55311433ff0fc985269160633af"
	osqueryDistroLinuxSHA256    = "61ef2351a07dbc36ae9ebff605e8a7ecc4e09a07ac11f540d2aed78c143addbe"
	osqueryDistroLinuxARMSHA256 = "106ea8a90dff0ccff852f44137848fe47ab9e8cfd27e5cd3a5ef963024b0564b"
	osqueryDistroWindowsSHA256  = "f5a6955db724559638e43aef181e26eadfe4bfb827907ffd134d9abb0512cc58"
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

func OsquerydCertsPath(dir string) string {
	return filepath.Join(dir, osqueryCertsPath)
}

func OsquerydLensesDir(dir string) string {
	return filepath.Join(dir, osqueryLensesDir)
}

func OsquerydDarwinDistroPath() string {
	return osqueryDarwinPath
}

func OsquerydLinuxDistroPath() string {
	return OsquerydPath(osqueryLinuxPath)
}

func OsquerydCertsLinuxDistroPath() string {
	return osqueryCertsLinuxPath
}

func OsquerydCertsDarwinDistroPath() string {
	return osqueryCertsDarwinPath
}

func OsquerydCertsWindowsDistroPath() string {
	return osqueryCertsWindowsPath
}

func OsquerydLensesLinuxDistroDir() string {
	return osqueryLensesLinuxDir
}

func OsquerydLensesDarwinDistroDir() string {
	return osqueryLensesDarwinDir
}

func OsquerydDistroFilename() string {
	return OsquerydDistroPlatformFilename(runtime.GOOS)
}

func OsquerydDistroPlatformFilename(platform string) string {
	switch platform {
	case "windows":
		return OsquerydFilenameForOS(platform)
	case "darwin":
		return OsquerydDarwinApp()
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
	{"darwin", "amd64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"darwin", "arm64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"windows", "amd64"}: {osqueryMSIExt, osqueryDistroWindowsSHA256, true},
}

func GetSpec(osarch OSArch) (spec Spec, err error) {
	if spec, ok := specs[osarch]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%v: %w", osarch, ErrUnsupportedOS)
}
