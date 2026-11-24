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

// Windows ARM URL: https://github.com/osquery/osquery/releases/download/{{osqueryVersion}}/osquery-{{osqueryVersion}}.windows_arm64.zip
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

	osqueryVersion = "5.24.1"
	osqueryMSIExt  = ".msi"
	osqueryPkgExt  = ".pkg"
	osqueryZipExt  = ".zip"

	osqueryDistroDarwinSHA256     = "950cc119f68c4d292f3e9407171205633e7174e13fc14ab593f48c75d907bd4f"
	osqueryDistroLinuxSHA256      = "31435ee69c43449eba612d0f3df3f2bccfe59805ab9a14fadc550bdc34814f2e"
	osqueryDistroLinuxARMSHA256   = "121246c1c0f798dde5218d308f56e112d44f2b81fb28d215ebc5ed932a8c9bed"
	osqueryDistroWindowsSHA256    = "36cc35dea559270c48040d894adde22785150bd2fcfe8f2489b86ead4688b004"
	osqueryDistroWindowsZipSHA256 = "a14a5d605298eb1eee90c9df59da341114d8b1b2014b3cfe557fa66940a83c6f"
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
