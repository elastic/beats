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

package mage

import (
	devtools "github.com/elastic/beats/v8/dev-tools/mage"
)

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
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

// -----------------------------------------------------------------------------
// Customizations specific to Packetbeat.
// - Config file contains an OS specific device name (affects darwin, windows).
// - libpcap or winpcap is compiled on the cross-compile docker image.
// - On Linux libpcap is statically linked. Darwin and Windows are dynamic.

const (
	linuxPcapLDFLAGS = "-L/libpcap/libpcap-1.8.1 -lpcap"
	linuxPcapCFLAGS  = "-I /libpcap/libpcap-1.8.1"
)

var libpcapLDFLAGS = map[string]string{
	"linux/386":      "-L/libpcap/libpcap-1.8.1-i386 -lpcap",
	"linux/amd64":    "-L/libpcap/libpcap-1.8.1-amd64 -lpcap",
	"linux/arm64":    linuxPcapLDFLAGS,
	"linux/armv5":    linuxPcapLDFLAGS,
	"linux/armv6":    linuxPcapLDFLAGS,
	"linux/armv7":    linuxPcapLDFLAGS,
	"linux/mips":     "-L/libpcap/libpcap-1.8.1-mips -lpcap",
	"linux/mipsle":   "-L/libpcap/libpcap-1.8.1-mipsel -lpcap",
	"linux/mips64":   "-L/libpcap/libpcap-1.8.1-mips64 -lpcap",
	"linux/mips64le": "-L/libpcap/libpcap-1.8.1-mips64el -lpcap",
	"linux/ppc64le":  "-L/libpcap/libpcap-1.8.1-ppc64el -lpcap",
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
