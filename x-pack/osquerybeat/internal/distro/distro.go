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

	osqueryVersion = "5.7.0"
	osqueryMSIExt  = ".msi"
	osqueryPkgExt  = ".pkg"

	osqueryDistroDarwinSHA256   = "6147f5632d7ae0bfb5c485f3b2211057179ceed18904b375007f23043408261f"
	osqueryDistroLinuxSHA256    = "0e34979a795be205b2a1bdf0b85464a34951fe94b462bd7c23bd894e24baceef"
	osqueryDistroLinuxARMSHA256 = "86826541444fe192b3a5f54d0e7d9f83375fc242c44e741fcd0766c8f933cac0"
	osqueryDistroWindowsSHA256  = "0e54fec4b8b8a800e3bdd178a2473db9de25e53b8ea800775b7f9cc8c847be1c"
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
	{"windows", "amd64"}: {osqueryMSIExt, osqueryDistroWindowsSHA256, false},
}

func GetSpec(osarch OSArch) (spec Spec, err error) {
	if spec, ok := specs[osarch]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%v: %w", osarch, ErrUnsupportedOS)
}
