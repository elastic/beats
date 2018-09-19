// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packages

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-sysinfo"
)

const (
	moduleName    = "system"
	metricsetName = "packages"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(moduleName),
	}, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	host, err := sysinfo.Host()
	if err != nil {
		ms.log.Errorw("Error getting the OS", "error", err)
	}

	hostInfo := host.Info()
	if hostInfo.OS == nil {
		ms.log.Errorw("No OS info from sysinfo.Host", "error", err)
	}

	var packages []Package
	switch hostInfo.OS.Family {
	case "redhat":
		packages, err = listRPMPackages()
		if err != nil {
			ms.log.Errorw("Error getting RPM packages", "error", err)
		}
	case "debian":
		packages, err = listDebPackages()
		if err != nil {
			ms.log.Errorw("Error getting DEB packages", "error", err)
		}
	case "darwin":
		packages, err = listBrewPackages()
		if err != nil {
			ms.log.Errorw("Error getting Homebrew packages", "error", err)
		}
	default:
		ms.log.Errorw("No logic for getting packages for OS family", "os", hostInfo.OS.Family)
	}

	var pkgInfos []common.MapStr

	for _, pkg := range packages {
		pkgInfos = append(pkgInfos, common.MapStr{
			"package.name":        pkg.Name,
			"package.version":     pkg.Version,
			"package.release":     pkg.Release,
			"package.arch":        pkg.Arch,
			"package.license":     pkg.License,
			"package.installtime": pkg.InstallTime,
			"package.size":        pkg.Size,
			"package.summary":     pkg.Summary,
			"package.url":         pkg.URL,
		})
	}

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"packages": pkgInfos,
		},
	})
}

type Package struct {
	Name        string
	Version     string
	Release     string
	Arch        string
	License     string
	InstallTime time.Time
	Size        uint64
	Summary     string
	URL         string
}

/*
The following functions copied from https://github.com/tsg/listpackages/blob/master/main.go
*/
func listRPMPackages() ([]Package, error) {
	format := "%{NAME}|%{VERSION}|%{RELEASE}|%{ARCH}|%{LICENSE}|%{INSTALLTIME}|%{SIZE}|%{URL}|%{SUMMARY}\\n"
	out, err := exec.Command("/usr/bin/rpm", "--qf", format, "-qa").Output()
	if err != nil {
		return nil, fmt.Errorf("Error running rpm -qa command: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	packages := []Package{}
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		words := strings.SplitN(line, "|", 9)
		if len(words) < 9 {
			return nil, fmt.Errorf("Line '%s' doesn't have enough elements", line)
		}
		pkg := Package{
			Name:    words[0],
			Version: words[1],
			Release: words[2],
			Arch:    words[3],
			License: words[4],
			// install time - 5
			// size - 6
			URL:     words[7],
			Summary: words[8],
		}
		ts, err := strconv.ParseInt(words[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error converting %s to string: %v", words[5], err)
		}
		pkg.InstallTime = time.Unix(ts, 0)

		pkg.Size, err = strconv.ParseUint(words[6], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error converting %s to string: %v", words[6], err)
		}

		packages = append(packages, pkg)

	}

	return packages, nil
}

func listDebPackages() ([]Package, error) {
	statusFile := "/var/lib/dpkg/status"
	file, err := os.Open(statusFile)
	if err != nil {
		return nil, fmt.Errorf("Error opening '%s': %v", statusFile, err)
	}
	defer file.Close()

	packages := []Package{}
	pkg := &Package{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			// empty line signals new package
			packages = append(packages, *pkg)
			pkg = &Package{}
			continue
		}
		if strings.HasPrefix(line, " ") {
			// not interested in multi-lines for now
			continue
		}
		words := strings.SplitN(line, ":", 2)
		if len(words) != 2 {
			return nil, fmt.Errorf("The following line was unexpected (no ':' found): '%s'", line)
		}
		value := strings.TrimSpace(words[1])
		switch strings.ToLower(words[0]) {
		case "package":
			pkg.Name = value
		case "architecture":
			pkg.Arch = value
		case "version":
			pkg.Version = value
		case "description":
			pkg.Summary = value
		case "installed-size":
			pkg.Size, err = strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Error converting %s to int: %v", value, err)
			}
		default:
			continue
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error scanning file: %v", err)
	}
	return packages, nil
}

func listBrewPackages() ([]Package, error) {
	cellarPath := "/usr/local/Cellar"

	cellarInfo, err := os.Stat(cellarPath)
	if err != nil {
		return nil, fmt.Errorf("Homebrew cellar not found in %s: %v", cellarPath, err)
	}
	if !cellarInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", cellarPath)
	}

	packageDirs, err := ioutil.ReadDir(cellarPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading directory %s: %v", cellarPath, err)
	}

	packages := []Package{}
	for _, packageDir := range packageDirs {
		if !packageDir.IsDir() {
			continue
		}
		pkgPath := path.Join(cellarPath, packageDir.Name())
		versions, err := ioutil.ReadDir(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("Error reading directory: %s: %v", pkgPath, err)
		}
		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			pkg := Package{
				Name:        packageDir.Name(),
				Version:     version.Name(),
				InstallTime: version.ModTime(),
			}

			// read formula
			formulaPath := path.Join(cellarPath, pkg.Name, pkg.Version, ".brew", pkg.Name+".rb")
			file, err := os.Open(formulaPath)
			if err != nil {
				//fmt.Printf("WARNING: Can't get formula for package %s-%s\n", pkg.Name, pkg.Version)
				// TODO: follow the path from INSTALL_RECEIPT.json to find the formula
				continue
			}
			scanner := bufio.NewScanner(file)
			count := 15 // only look into the first few lines of the formula
			for scanner.Scan() {
				count -= 1
				if count == 0 {
					break
				}
				line := scanner.Text()
				if strings.HasPrefix(line, "  desc ") {
					pkg.Summary = strings.Trim(line[7:], " \"")
				} else if strings.HasPrefix(line, "  homepage ") {
					pkg.URL = strings.Trim(line[11:], " \"")
				}
			}

			packages = append(packages, pkg)
		}
	}
	return packages, nil
}
