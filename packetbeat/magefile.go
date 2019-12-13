// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build mage

package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/dev-tools/mage"
	packetbeat "github.com/elastic/beats/packetbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/common"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Packetbeat analyzes network traffic and sends the data to Elasticsearch."
}

// Aliases provides compatibility with CI while we transition all Beats
// to having common testing targets.
var Aliases = map[string]interface{}{
	"goTestUnit": GoUnitTest, // dev-tools/jenkins_ci.ps1 uses this.
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	if dep, found := crossBuildDeps[devtools.Platform.Name]; found {
		mg.Deps(dep)
	}

	params := devtools.DefaultGolangCrossBuildArgs()
	if flags, found := libpcapLDFLAGS[devtools.Platform.Name]; found {
		params.Env = map[string]string{
			"CGO_LDFLAGS": flags,
		}
	}
	if flags, found := libpcapCFLAGS[devtools.Platform.Name]; found {
		params.Env["CGO_CFLAGS"] = flags
	}

	return devtools.GolangCrossBuild(params)
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	mg.Deps(patchCGODirectives)
	defer undoPatchCGODirectives()

	// These Windows builds write temporary .s and .o files into the packetbeat
	// dir so they cannot be run in parallel. Changing to a different CWD does
	// not change where the temp files get written so that cannot be used as a
	// fix.
	if err := devtools.CrossBuild(devtools.ForPlatforms("windows"), devtools.Serially()); err != nil {
		return err
	}

	return devtools.CrossBuild(devtools.ForPlatforms("!windows"))
}

// CrossBuildXPack cross-builds the beat with XPack for all target platforms.
func CrossBuildXPack() error {
	mg.Deps(patchCGODirectives)
	defer undoPatchCGODirectives()

	// These Windows builds write temporary .s and .o files into the packetbeat
	// dir so they cannot be run in parallel. Changing to a different CWD does
	// not change where the temp files get written so that cannot be used as a
	// fix.
	if err := devtools.CrossBuildXPack(devtools.ForPlatforms("windows"), devtools.Serially()); err != nil {
		return err
	}

	return devtools.CrossBuildXPack(devtools.ForPlatforms("!windows"))
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatPackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()
	customizePackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildXPack, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update updates the generated files.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, includeList, fieldDocs)
}

// Config generates the config files.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, packetbeat.ConfigFileParams(), ".")
}

func includeList() error {
	options := devtools.DefaultIncludeListOptions()
	options.ImportDirs = []string{"protos/*"}
	options.ModuleDirs = nil
	return devtools.GenerateIncludeListGo(options)
}

// Fields generates fields.yml and fields.go files for the Beat.
func Fields() {
	mg.Deps(libbeatAndPacketbeatCommonFieldsGo, protosFieldsGo)
	mg.Deps(fieldsYML)
}

// libbeatAndPacketbeatCommonFieldsGo generates a fields.go containing both
// libbeat and packetbeat's common fields.
func libbeatAndPacketbeatCommonFieldsGo() error {
	if err := devtools.GenerateFieldsYAML(); err != nil {
		return err
	}
	return devtools.GenerateAllInOneFieldsGo()
}

// protosFieldsGo generates a fields.go for each protocol.
func protosFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("protos")
}

// fieldsYML generates the fields.yml file containing all fields.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML("protos")
}

func fieldDocs() error {
	return devtools.Docs.FieldDocs("fields.yml")
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards("protos")
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	fmt.Println(">> integTest: Complete (no tests require the integ test environment)")
}

// UnitTest executes the unit tests.
func UnitTest() {
	mg.SerialDeps(GoUnitTest, PythonUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.SerialDeps(Fields, devtools.BuildSystemTestBinary)
	return devtools.PythonNoseTest(devtools.DefaultPythonTestUnitArgs())
}

// -----------------------------------------------------------------------------
// Customizations specific to Packetbeat.
// - Config file contains an OS specific device name (affects darwin, windows).
// - Must compile libpcap or winpcap during cross-compilation.
// - On Linux libpcap is statically linked. Darwin and Windows are dynamic.

const (
	libpcapURL    = "https://s3.amazonaws.com/beats-files/deps/libpcap-1.8.1.tar.gz"
	libpcapSHA256 = "673dbc69fdc3f5a86fb5759ab19899039a8e5e6c631749e48dcd9c6f0c83541e"
)

const (
	linuxPcapLDFLAGS = "-L/libpcap/libpcap-1.8.1 -lpcap"
	linuxPcapCFLAGS  = "-I /libpcap/libpcap-1.8.1"
)

var libpcapLDFLAGS = map[string]string{
	"linux/386":      linuxPcapLDFLAGS,
	"linux/amd64":    linuxPcapLDFLAGS,
	"linux/arm64":    linuxPcapLDFLAGS,
	"linux/armv5":    linuxPcapLDFLAGS,
	"linux/armv6":    linuxPcapLDFLAGS,
	"linux/armv7":    linuxPcapLDFLAGS,
	"linux/mips":     linuxPcapLDFLAGS,
	"linux/mipsle":   linuxPcapLDFLAGS,
	"linux/mips64":   linuxPcapLDFLAGS,
	"linux/mips64le": linuxPcapLDFLAGS,
	"linux/ppc64le":  linuxPcapLDFLAGS,
	"linux/s390x":    linuxPcapLDFLAGS,
	"darwin/amd64":   "-lpcap",
	"windows/amd64":  "-L /libpcap/win/WpdPack/Lib/x64 -lwpcap",
	"windows/386":    "-L /libpcap/win/WpdPack/Lib -lwpcap",
}

var libpcapCFLAGS = map[string]string{
	"linux/386":      linuxPcapCFLAGS,
	"linux/amd64":    linuxPcapCFLAGS,
	"linux/arm64":    linuxPcapCFLAGS,
	"linux/armv5":    linuxPcapCFLAGS,
	"linux/armv6":    linuxPcapCFLAGS,
	"linux/armv7":    linuxPcapCFLAGS,
	"linux/mips":     linuxPcapCFLAGS,
	"linux/mipsle":   linuxPcapCFLAGS,
	"linux/mips64":   linuxPcapCFLAGS,
	"linux/mips64le": linuxPcapCFLAGS,
	"linux/ppc64le":  linuxPcapCFLAGS,
	"linux/s390x":    linuxPcapCFLAGS,
	"windows/amd64":  "-I /libpcap/win/WpdPack/Include",
	"windows/386":    "-I /libpcap/win/WpdPack/Include",
}

var crossBuildDeps = map[string]func() error{
	"linux/386":      buildLibpcapLinux386,
	"linux/amd64":    buildLibpcapLinuxAMD64,
	"linux/arm64":    buildLibpcapLinuxARM64,
	"linux/armv5":    buildLibpcapLinuxARMv5,
	"linux/armv6":    buildLibpcapLinuxARMv6,
	"linux/armv7":    buildLibpcapLinuxARMv7,
	"linux/mips":     buildLibpcapLinuxMIPS,
	"linux/mipsle":   buildLibpcapLinuxMIPSLE,
	"linux/mips64":   buildLibpcapLinuxMIPS64,
	"linux/mips64le": buildLibpcapLinuxMIPS64LE,
	"linux/ppc64le":  buildLibpcapLinuxPPC64LE,
	"linux/s390x":    buildLibpcapLinuxS390x,
	"windows/amd64":  installLibpcapWindowsAMD64,
	"windows/386":    installLibpcapWindows386,
}

// buildLibpcapFromSource builds libpcap from source because the library needs
// to be compiled with -fPIC.
// See https://github.com/elastic/beats/pull/4217.
func buildLibpcapFromSource(params map[string]string) error {
	tarFile, err := devtools.DownloadFile(libpcapURL, "/libpcap")
	if err != nil {
		return errors.Wrap(err, "failed to download libpcap source")
	}

	if err = devtools.VerifySHA256(tarFile, libpcapSHA256); err != nil {
		return err
	}

	if err = devtools.Extract(tarFile, "/libpcap"); err != nil {
		return errors.Wrap(err, "failed to extract libpcap")
	}

	var configureArgs []string
	for k, v := range params {
		if strings.HasPrefix(k, "-") {
			delete(params, k)
			configureArgs = append(configureArgs, k+"="+v)
		}
	}

	// Use sh -c here because sh.Run does not expose a way to change the CWD.
	// This command only runs in Linux so this is fine.
	return sh.RunWith(params, "sh", "-c",
		"cd /libpcap/libpcap-1.8.1 && "+
			"./configure --enable-usb=no --enable-bluetooth=no --enable-dbus=no "+strings.Join(configureArgs, " ")+"&& "+
			"make")
}

func buildLibpcapLinux386() error {
	return buildLibpcapFromSource(map[string]string{
		"CFLAGS":  "-m32",
		"LDFLAGS": "-m32",
	})
}

func buildLibpcapLinuxAMD64() error {
	return buildLibpcapFromSource(map[string]string{})
}

func buildLibpcapLinuxARM64() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "aarch64-unknown-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxARMv5() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "arm-linux-gnueabi",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxARMv6() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "arm-linux-gnueabi",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxARMv7() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "arm-linux-gnueabihf",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxMIPS() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "mips-unknown-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxMIPSLE() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "mipsle-unknown-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxMIPS64() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "mips64-unknown-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxMIPS64LE() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "mips64le-unknown-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxPPC64LE() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "powerpc64le-linux-gnu",
		"--with-pcap": "linux",
	})
}

func buildLibpcapLinuxS390x() error {
	return buildLibpcapFromSource(map[string]string{
		"--host":      "s390x-ibm-linux-gnu",
		"--with-pcap": "linux",
	})
}

func installLibpcapWindowsAMD64() error {
	mg.SerialDeps(installWinpcap, generateWin64StaticWinpcap)
	return nil
}

func installLibpcapWindows386() error {
	return installWinpcap()
}

func installWinpcap() error {
	log.Println("Install Winpcap")
	const wpdpackURL = "https://www.winpcap.org/install/bin/WpdPack_4_1_2.zip"

	winpcapZip, err := devtools.DownloadFile(wpdpackURL, "/")
	if err != nil {
		return err
	}

	if err = devtools.Extract(winpcapZip, "/libpcap/win"); err != nil {
		return err
	}

	return nil
}

func generateWin64StaticWinpcap() error {
	log.Println(">> Generating 64-bit winpcap static lib")

	// Notes: We are using absolute path to make sure the files
	// are available for x-pack build.
	// Ref: https://github.com/elastic/beats/issues/1259
	defer devtools.DockerChown(devtools.MustExpand("{{elastic_beats_dir}}/{{.BeatName}}/lib"))
	return devtools.RunCmds(
		// Requires mingw-w64-tools.
		[]string{"gendef", devtools.MustExpand("{{elastic_beats_dir}}/{{.BeatName}}/lib/windows-64/wpcap.dll")},
		[]string{"mv", "wpcap.def", devtools.MustExpand("{{ elastic_beats_dir}}/{{.BeatName}}/lib/windows-64/wpcap.def")},
		[]string{"x86_64-w64-mingw32-dlltool", "--as-flags=--64",
			"-m", "i386:x86-64", "-k",
			"--output-lib", "/libpcap/win/WpdPack/Lib/x64/libwpcap.a",
			"--input-def", devtools.MustExpand("{{elastic_beats_dir}}/{{.BeatName}}/lib/windows-64/wpcap.def")},
	)
}

const pcapGoFile = "{{ elastic_beats_dir }}/vendor/github.com/tsg/gopacket/pcap/pcap.go"

var cgoDirectiveRegex = regexp.MustCompile(`(?m)#cgo .*(?:LDFLAGS|CFLAGS).*$`)

func patchCGODirectives() error {
	// cgo directives do not support GOARM tags so we will clear the tags
	// and set them via CGO_LDFLAGS and CGO_CFLAGS.
	// Ref: https://github.com/golang/go/issues/7211
	f := devtools.MustExpand(pcapGoFile)
	log.Println("Patching", f, cgoDirectiveRegex.String())
	return devtools.FindReplace(f, cgoDirectiveRegex, "")
}

func undoPatchCGODirectives() error {
	return sh.Run("git", "checkout", devtools.MustExpand(pcapGoFile))
}

// customizePackaging modifies the device in the configuration files based on
// the target OS.
func customizePackaging() {
	var (
		configYml = devtools.PackageFile{
			Mode:   0600,
			Source: "{{.PackageDir}}/{{.BeatName}}.yml",
			Config: true,
			Dep: func(spec devtools.PackageSpec) error {
				c := packetbeat.ConfigFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return devtools.Config(devtools.ShortConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
		referenceConfigYml = devtools.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/{{.BeatName}}.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				c := packetbeat.ConfigFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return devtools.Config(devtools.ReferenceConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
	)

	for _, args := range devtools.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case devtools.TarGz, devtools.Zip:
				args.Spec.ReplaceFile("{{.BeatName}}.yml", configYml)
				args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfigYml)
			case devtools.Deb, devtools.RPM, devtools.DMG:
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", configYml)
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfigYml)
			case devtools.Docker:
				args.Spec.ExtraVar("linux_capabilities", "cap_net_raw,cap_net_admin=eip")
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}

			// Match the first package type then continue.
			break
		}
	}
}
