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
	"strings"
)

var ErrUnsupportedOS = errors.New("unsupported OS")

var (
	DataDir        = filepath.Clean("build/data")
	DataInstallDir = filepath.Join(DataDir, "install")
	DataCacheDir   = filepath.Join(DataDir, "cache")
)

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

	osqueryVersion = "5.23.1"
	osqueryPkgExt  = ".pkg"
	osqueryZipExt  = ".zip"

	osqueryDistroDarwinSHA256        = "9f40cea0358759ab2ee871c577055657e3cc2c7cbe5c1247f764245941178aa6"
	osqueryDistroLinuxSHA256         = "0f37a478a1dbda24b67c81551e32d734b392c5a2f5deb156bf1c41ca204cfa67"
	osqueryDistroLinuxARMSHA256      = "9ae763820166f75f19970b5147b1930a308865a923ab127f4b8bbaea7b69962a"
	osqueryDistroWindowsARMZipSHA256 = "0913d05cc3fc92dd9253c945caacde10a776408f267cc1cc853a05de24dba900"
	osqueryDistroWindowsX86ZipSHA256 = "7bd411050ef6b5aae1b23956aec0dc5ce6e800c5656f0cd463ac70a6e1bdf30b"
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
	return OsquerydWindowsZipPlatformPath("windows_arm64")
}

func OsquerydWindowsZipPlatformPath(platform string) string {
	return filepath.Join(osqueryName+"-"+osqueryVersion+"."+platform, "Program Files", "osquery", "osqueryd", "osqueryd.exe")
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
	return OsquerydCertsWindowsZipPlatformDistroPath("windows_arm64")
}

func OsquerydCertsWindowsZipPlatformDistroPath(platform string) string {
	return osqueryName + "-" + osqueryVersion + "." + platform + "/" + osqueryCertsWindowsZipPath
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
		// The short suffix is used by the Windows ARM64 archive.
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
		return 0o755
	}
	return 0o644
}

func (s Spec) URL(osname string) string {
	if strings.HasSuffix(s.PackSuffix, osqueryZipExt) {
		return osqueryDownloadGithubBaseURL + "/" + osqueryVersion + "/" + s.DistroFilename()
	}
	return osqueryDownloadBaseURL + "/" + osname + "/" + s.DistroFilename()
}

var specs = map[OSArch]Spec{
	{"linux", "amd64"}:   {"_1.linux_x86_64.tar.gz", osqueryDistroLinuxSHA256, true},
	{"linux", "arm64"}:   {"_1.linux_aarch64.tar.gz", osqueryDistroLinuxARMSHA256, true},
	{"darwin", "amd64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"darwin", "arm64"}:  {osqueryPkgExt, osqueryDistroDarwinSHA256, true},
	{"windows", "amd64"}: {".windows_x86_64.zip", osqueryDistroWindowsX86ZipSHA256, true},
	{"windows", "arm64"}: {osqueryZipExt, osqueryDistroWindowsARMZipSHA256, true},
}

func GetSpec(osarch OSArch) (spec Spec, err error) {
	if spec, ok := specs[osarch]; ok {
		return spec, nil
	}
	return spec, fmt.Errorf("%v: %w", osarch, ErrUnsupportedOS)
}
