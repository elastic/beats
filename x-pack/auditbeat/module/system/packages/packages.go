// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package packages

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/go-sysinfo/types"

	"github.com/OneOfOne/xxhash"

	"github.com/OneOfOne/xxhash"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-sysinfo"
)

const (
	moduleName    = "system"
	metricsetName = "packages"

	redhat = "redhat"
	debian = "debian"
	darwin = "darwin"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about the host.
type MetricSet struct {
	mb.BaseMetricSet
	config   Config
	osFamily string
	cache    *cache.Cache
	log      *logp.Logger
}

// Package represents information for a package.
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

// Hash creates a hash for Package.
func (pkg Package) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(pkg.Name)
	h.WriteString(pkg.InstallTime.String())
	return h.Sum64()
}

func (pkg Package) toMapStr() common.MapStr {
	return common.MapStr{
		"name":        pkg.Name,
		"version":     pkg.Version,
		"release":     pkg.Release,
		"arch":        pkg.Arch,
		"license":     pkg.License,
		"installtime": pkg.InstallTime,
		"size":        pkg.Size,
		"summary":     pkg.Summary,
		"url":         pkg.URL,
	}
}

func getOS() (*types.OSInfo, error) {
	host, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "error getting the OS")
	}

	hostInfo := host.Info()
	if hostInfo.OS == nil {
		return nil, errors.New("no host info")
	}

	return hostInfo.OS, nil
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(moduleName),
	}

	if os, err := getOS(); err == nil {
		switch os.Family {
		case redhat, debian, darwin:
			ms.osFamily = os.Family
		default:
			return nil, fmt.Errorf("this metricset does not support OS family %v", os.Family)
		}
	} else if err != nil {
		return nil, err
	}

	if config.ReportChanges {
		ms.cache = cache.New()
	}

	return ms, nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	packages, err := getPackages(ms.osFamily)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
		return
	}

	if ms.cache != nil && !ms.cache.IsEmpty() {
		installed, removed := ms.cache.DiffAndUpdateCache(convertToCacheable(packages))

		for _, pkgInfo := range installed {
			pkgInfoMapStr := pkgInfo.(*Package).toMapStr()
			pkgInfoMapStr.Put("status", "new")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"package": pkgInfoMapStr,
				},
			})
		}

		for _, pkgInfo := range removed {
			pkgInfoMapStr := pkgInfo.(*Package).toMapStr()
			pkgInfoMapStr.Put("status", "removed")

			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"package": pkgInfoMapStr,
				},
			})
		}
	} else {
		// Report all installed packages
		var pkgInfos []common.MapStr

		for _, pkgInfo := range packages {
			pkgInfoMapStr := pkgInfo.toMapStr()
			pkgInfoMapStr.Put("status", "installed")

			pkgInfos = append(pkgInfos, pkgInfoMapStr)
		}

		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"package": pkgInfos,
			},
		})

		if ms.cache != nil {
			// This will initialize the cache with the current packages
			ms.cache.DiffAndUpdateCache(convertToCacheable(packages))
		}
	}
}

func convertToCacheable(packages []*Package) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(packages))

	for _, p := range packages {
		c = append(c, p)
	}

	return c
}

func getPackages(osFamily string) (packages []*Package, err error) {
	switch osFamily {
	case redhat:
		// TODO: Implement RPM
		err = errors.New("RPM not yet supported")
	case debian:
		packages, err = listDebPackages()
		if err != nil {
			err = errors.Wrap(err, "error getting DEB packages")
		}
	case darwin:
		packages, err = listBrewPackages()
		if err != nil {
			err = errors.Wrap(err, "error getting Homebrew packages")
		}
	default:
		panic("unknown OS - this should not have happened")
	}

	return
}

func listDebPackages() ([]*Package, error) {
	const statusFile = "/var/lib/dpkg/status"
	file, err := os.Open(statusFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening '%s'", statusFile)
	}
	defer file.Close()

	var packages []*Package
	pkg := &Package{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			// empty line signals new package
			packages = append(packages, pkg)
			pkg = &Package{}
			continue
		}
		if strings.HasPrefix(line, " ") {
			// not interested in multi-lines for now
			continue
		}
		words := strings.SplitN(line, ":", 2)
		if len(words) != 2 {
			return nil, fmt.Errorf("the following line was unexpected (no ':' found): '%s'", line)
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
				return nil, errors.Wrapf(err, "error converting %s to int", value)
			}
		default:
			continue
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error scanning file")
	}
	return packages, nil
}

func listBrewPackages() ([]*Package, error) {
	const cellarPath = "/usr/local/Cellar"

	packageDirs, err := ioutil.ReadDir(cellarPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "%s does not exist - is Homebrew installed?", cellarPath)
	} else if err != nil {
		return nil, errors.Wrapf(err, "error reading directory %s", cellarPath)
	}

	var packages []*Package
	for _, packageDir := range packageDirs {
		if !packageDir.IsDir() {
			continue
		}
		pkgPath := path.Join(cellarPath, packageDir.Name())
		versions, err := ioutil.ReadDir(pkgPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading directory: %s", pkgPath)
		}

		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			pkg := &Package{
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
				count--
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
