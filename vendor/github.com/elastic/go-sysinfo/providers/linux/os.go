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

package linux

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

const (
	osRelease      = "/etc/os-release"
	lsbRelease     = "/etc/lsb-release"
	distribRelease = "/etc/*-release"
	versionGrok    = `(?P<version>(?P<major>[0-9]+)\.?(?P<minor>[0-9]+)?\.?(?P<patch>\w+)?)(?: \((?P<codename>\w+)\))?`
)

var (
	// distribReleaseRegexp parses the /etc/<distrib>-release file. See man lsb-release.
	distribReleaseRegexp = regexp.MustCompile(`(?P<name>[\w]+).* ` + versionGrok)

	// versionRegexp parses version numbers (e.g. 6 or 6.1 or 6.1.0 or 6.1.0_20150102).
	versionRegexp = regexp.MustCompile(versionGrok)
)

// familyMap contains a mapping of family -> []platforms.
var familyMap = map[string][]string{
	"redhat": {"redhat", "fedora", "centos", "scientific", "oraclelinux", "amzn", "rhel"},
	"debian": {"debian", "ubuntu", "raspbian"},
	"suse":   {"suse", "sles", "opensuse"},
}

var platformToFamilyMap map[string]string

func init() {
	platformToFamilyMap = map[string]string{}
	for family, platformList := range familyMap {
		for _, platform := range platformList {
			platformToFamilyMap[platform] = family
		}
	}
}

func OperatingSystem() (*types.OSInfo, error) {
	return getOSInfo("")
}

func getOSInfo(baseDir string) (*types.OSInfo, error) {
	osInfo, err := getOSRelease(baseDir)
	if err != nil {
		// Fallback
		return findDistribRelease(baseDir)
	}

	// For the redhat family, enrich version info with data from
	// /etc/[distrib]-release because the minor and patch info isn't always
	// present in os-release.
	if osInfo.Family != "redhat" {
		return osInfo, nil
	}

	distInfo, err := findDistribRelease(baseDir)
	if err != nil {
		return osInfo, err
	}
	osInfo.Major = distInfo.Major
	osInfo.Minor = distInfo.Minor
	osInfo.Patch = distInfo.Patch
	osInfo.Codename = distInfo.Codename
	return osInfo, nil
}

func getOSRelease(baseDir string) (*types.OSInfo, error) {
	lsbRel, _ := ioutil.ReadFile(filepath.Join(baseDir, lsbRelease))

	osRel, err := ioutil.ReadFile(filepath.Join(baseDir, osRelease))
	if err != nil {
		return nil, err
	}
	if len(osRel) == 0 {
		return nil, errors.Errorf("%v is empty", osRelease)
	}

	return parseOSRelease(append(lsbRel, osRel...))
}

func parseOSRelease(content []byte) (*types.OSInfo, error) {
	fields := map[string]string{}

	s := bufio.NewScanner(bytes.NewReader(content))
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())

		// Skip blank lines and comments.
		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}

		parts := bytes.SplitN(s.Bytes(), []byte("="), 2)
		if len(parts) != 2 {
			continue
		}

		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		fields[key] = val

		// Trim quotes.
		val, err := strconv.Unquote(val)
		if err == nil {
			fields[key] = strings.TrimSpace(val)
		}
	}

	if s.Err() != nil {
		return nil, s.Err()
	}

	return makeOSInfo(fields)
}

func makeOSInfo(osRelease map[string]string) (*types.OSInfo, error) {
	os := &types.OSInfo{
		Platform: osRelease["ID"],
		Name:     osRelease["NAME"],
		Version:  osRelease["VERSION"],
		Build:    osRelease["BUILD_ID"],
		Codename: osRelease["VERSION_CODENAME"],
	}

	if os.Codename == "" {
		// Some OSes uses their own CODENAME keys (e.g UBUNTU_CODENAME) or we
		// can get the DISTRIB_CODENAME value from the lsb-release data.
		for k, v := range osRelease {
			if strings.Contains(k, "CODENAME") {
				os.Codename = v
				break
			}
		}
	}

	if os.Platform == "" {
		// Fallback to the first word of the NAME field.
		parts := strings.SplitN(os.Name, " ", 2)
		if len(parts) > 0 {
			os.Platform = strings.ToLower(parts[0])
		}
	}

	if os.Version != "" {
		// Try parsing info from the version.
		keys := versionRegexp.SubexpNames()
		for i, m := range versionRegexp.FindStringSubmatch(os.Version) {
			switch keys[i] {
			case "major":
				os.Major, _ = strconv.Atoi(m)
			case "minor":
				os.Minor, _ = strconv.Atoi(m)
			case "patch":
				os.Patch, _ = strconv.Atoi(m)
			case "codename":
				if os.Codename == "" {
					os.Codename = m
				}
			}
		}
	}

	os.Family = platformToFamilyMap[strings.ToLower(os.Platform)]
	return os, nil
}

func findDistribRelease(baseDir string) (*types.OSInfo, error) {
	matches, err := filepath.Glob(filepath.Join(baseDir, distribRelease))
	if err != nil {
		return nil, err
	}
	for _, path := range matches {
		if strings.HasSuffix(path, osRelease) || strings.HasSuffix(path, lsbRelease) {
			continue
		}

		info, err := os.Lstat(path)
		if err != nil || info.Size() == 0 {
			continue
		}

		return getDistribRelease(path)
	}

	return nil, errors.New("no /etc/<distrib>-release file found")
}

func getDistribRelease(file string) (*types.OSInfo, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	parts := bytes.SplitN(data, []byte("\n"), 2)
	if len(parts) != 2 {
		return nil, errors.Errorf("failed to parse %v", file)
	}

	// Use distrib as platform name.
	var platform string
	if parts := strings.SplitN(filepath.Base(file), "-", 2); len(parts) > 0 {
		platform = strings.ToLower(parts[0])
	}

	return parseDistribRelease(platform, parts[0])
}

func parseDistribRelease(platform string, content []byte) (*types.OSInfo, error) {
	var (
		line = string(bytes.TrimSpace(content))
		keys = distribReleaseRegexp.SubexpNames()
		os   = &types.OSInfo{Platform: platform}
	)

	for i, m := range distribReleaseRegexp.FindStringSubmatch(line) {
		switch keys[i] {
		case "name":
			os.Name = m
		case "version":
			os.Version = m
		case "major":
			os.Major, _ = strconv.Atoi(m)
		case "minor":
			os.Minor, _ = strconv.Atoi(m)
		case "patch":
			os.Patch, _ = strconv.Atoi(m)
		case "codename":
			os.Version += " (" + m + ")"
			os.Codename = m
		}
	}

	os.Family = platformToFamilyMap[strings.ToLower(os.Platform)]
	return os, nil
}
