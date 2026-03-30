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

// Windows ARM URL: https://github.com/osquery/osquery/releases/download/5.19.0/osquery-5.19.0.windows_arm64.zip
const (
	osqueryDownloadBaseURL       = "https://pkg.osquery.io"
	osqueryDownloadGithubBaseURL = "https://github.com/osquery/osquery/releases/download"
	osqueryName                  = "osquery"
	osqueryDName                 = "osqueryd"
	osqueryDarwinApp             = "osquery.app"
	osqueryDarwinPath            = "opt/osquery/lib/" + osqueryDarwinApp

	osqueryCertsPEM            = "certs.pem"
	osqueryCertsPath           = "certs/" + osqueryCertsPEM
	osqueryLinuxPath           = "opt/osquery/bin"
	osqueryCertsLinuxPath      = "opt/osquery/share/osquery/certs/" + osqueryCertsPEM
	osqueryCertsDarwinPath     = "private/var/osquery/certs/" + osqueryCertsPEM
	osqueryCertsWindowsPath    = "osquery/certs/" + osqueryCertsPEM
	osqueryCertsWindowsZipPath = "Program Files/" + osqueryCertsWindowsPath

	osqueryLensesLinuxDir  = "opt/osquery/share/osquery/lenses"
	osqueryLensesDarwinDir = "private/var/osquery/lenses"

	osqueryLensesDir = "lenses"

	osqueryVersion = "5.19.0"
	osqueryMSIExt  = ".msi"
	osqueryPkgExt  = ".pkg"
	osqueryZipExt  = ".zip"

	osqueryDistroDarwinSHA256     = "f24b62075c22e3edcbb635bc0b118a221f69584b5d70c7caa78475bb4fd3ade4"
	osqueryDistroLinuxSHA256      = "3b9583cc037a2c5a7405d567083398103256a42e0fcfd026a11cc7aef410c772"
	osqueryDistroLinuxARMSHA256   = "fa649f57bfb1a6cb836a9af6e280b79382ff4a4e885c5998359c12e5caa051d3"
	osqueryDistroWindowsSHA256    = "6fe06cab43a31c596e4001616eee66fb32556bf5c228c4a4ba6daf2897edc1a3"
	osqueryDistroWindowsZipSHA256 = "298f4dfb2dad2bdf666b337d9d258c4bcc5254ad546c4710b24fcc864096c587"
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

func OsquerydWindowsZipPath() string {
	return filepath.Join(osqueryName+"-"+osqueryVersion+".windows_arm64", "Program Files", "osquery", "osqueryd", "osqueryd.exe")
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

func OsquerydCertsWindowsZipDistroPath() string {
	return osqueryName + "-" + osqueryVersion + ".windows_arm64" + "/" + osqueryCertsWindowsZipPath
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
	if s.PackSuffix == osqueryZipExt {
		// Currently the only file whose source is a zip is the Windows ARM64 one
		return osqueryName + "-" + osqueryVersion + ".windows_arm64" + s.PackSuffix
	}
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
	if s.PackSuffix == osqueryZipExt {
		return osqueryDownloadGithubBaseURL + "/" + osqueryVersion + "/" + s.DistroFilename()
	}
	return osqueryDownloadBaseURL + "/" + osname + "/" + s.DistroFilename()
}

var specs = map[OSArch]Spec{
	{"linux", "amd64"}:   {"_1.linux_x86_64.tar.gz", osqueryDistroLinuxSHA256, true},
	{"linux", "arm64"}:   {"_1.linux_aarch64.tar.gz", osqueryDistroLinuxARMSHA256, true},
	{"darwin", "amd64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"darwin", "arm64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"windows", "amd64"}: {osqueryMSIExt, osqueryDistroWindowsSHA256, true},
	{"windows", "arm64"}: {osqueryZipExt, osqueryDistroWindowsZipSHA256, true},
}

func GetSpec(osarch OSArch) (spec Spec, err error) {
	if spec, ok := specs[osarch]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%v: %w", osarch, ErrUnsupportedOS)
}
