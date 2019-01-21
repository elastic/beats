// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkg

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	moduleName    = "system"
	metricsetName = "package"
	namespace     = "system.audit.package"

	redhat = "redhat"
	debian = "debian"
	darwin = "darwin"

	bucketName              = "package.v1"
	bucketKeyPackages       = "packages"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

type eventAction uint8

const (
	eventActionExistingPackage eventAction = iota
	eventActionPackageInstalled
	eventActionPackageRemoved
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingPackage:
		return "existing_package"
	case eventActionPackageInstalled:
		return "package_installed"
	case eventActionPackageRemoved:
		return "package_removed"
	default:
		return ""
	}
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// MetricSet collects data about the system's packages.
type MetricSet struct {
	mb.BaseMetricSet
	config    config
	log       *logp.Logger
	cache     *cache.Cache
	bucket    datastore.Bucket
	lastState time.Time
	osFamily  string
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

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		config:        config,
		log:           logp.NewLogger(metricsetName),
		cache:         cache.New(),
		bucket:        bucket,
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

	// Load from disk: Time when state was last sent
	err = bucket.Load(bucketKeyStateTimestamp, func(blob []byte) error {
		if len(blob) > 0 {
			return ms.lastState.UnmarshalBinary(blob)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !ms.lastState.IsZero() {
		ms.log.Debugf("Last state was sent at %v. Next state update by %v.", ms.lastState, ms.lastState.Add(ms.config.effectiveStatePeriod()))
	} else {
		ms.log.Debug("No state timestamp found")
	}

	// Load from disk: Packages
	packages, err := ms.restorePackagesFromDisk()
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore packages from disk")
	}
	ms.log.Debugf("Restored %d packages from disk", len(packages))

	ms.cache.DiffAndUpdateCache(convertToCacheable(packages))

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate || ms.cache.IsEmpty() {
		ms.log.Debugf("State update needed (needsStateUpdate=%v, cache.IsEmpty()=%v)", needsStateUpdate, ms.cache.IsEmpty())
		err := ms.reportState(report)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
		}
		ms.log.Debugf("Next state update by %v", ms.lastState.Add(ms.config.effectiveStatePeriod()))
	}

	err := ms.reportChanges(report)
	if err != nil {
		ms.log.Error(err)
		report.Error(err)
	}
}

// reportState reports all installed packages on the system.
func (ms *MetricSet) reportState(report mb.ReporterV2) error {
	ms.lastState = time.Now()

	packages, err := getPackages(ms.osFamily)
	if err != nil {
		return errors.Wrap(err, "failed to get packages")
	}
	ms.log.Debugf("Found %v packages", len(packages))

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}
	for _, pkg := range packages {
		event := packageEvent(pkg, eventTypeState, eventActionExistingPackage)
		event.RootFields.Put("event.id", stateID.String())
		report.Event(event)
	}

	// This will initialize the cache with the current packages
	ms.cache.DiffAndUpdateCache(convertToCacheable(packages))

	// Save time so we know when to send the state again (config.StatePeriod)
	timeBytes, err := ms.lastState.MarshalBinary()
	if err != nil {
		return err
	}
	err = ms.bucket.Store(bucketKeyStateTimestamp, timeBytes)
	if err != nil {
		return errors.Wrap(err, "error writing state timestamp to disk")
	}

	return ms.savePackagesToDisk(packages)
}

// reportChanges detects and reports any changes to installed packages on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	packages, err := getPackages(ms.osFamily)
	if err != nil {
		return errors.Wrap(err, "failed to get packages")
	}
	ms.log.Debugf("Found %v packages", len(packages))

	installed, removed := ms.cache.DiffAndUpdateCache(convertToCacheable(packages))

	for _, cacheValue := range installed {
		report.Event(packageEvent(cacheValue.(*Package), eventTypeEvent, eventActionPackageInstalled))
	}

	for _, cacheValue := range removed {
		report.Event(packageEvent(cacheValue.(*Package), eventTypeEvent, eventActionPackageRemoved))
	}

	if len(installed) > 0 || len(removed) > 0 {
		return ms.savePackagesToDisk(packages)
	}

	return nil
}

func packageEvent(pkg *Package, eventType string, action eventAction) mb.Event {
	return mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"kind":   eventType,
				"action": action.String(),
			},
			"message": packageMessage(pkg, action),
		},
		MetricSetFields: pkg.toMapStr(),
	}
}

func packageMessage(pkg *Package, action eventAction) string {
	var actionString string
	switch action {
	case eventActionExistingPackage:
		actionString = "is already installed"
	case eventActionPackageInstalled:
		actionString = "installed"
	case eventActionPackageRemoved:
		actionString = "removed"
	}

	return fmt.Sprintf("Package %v (%v) %v",
		pkg.Name, pkg.Version, actionString)
}

func convertToCacheable(packages []*Package) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(packages))

	for _, p := range packages {
		c = append(c, p)
	}

	return c
}

// restorePackagesFromDisk loads the packages from disk.
func (ms *MetricSet) restorePackagesFromDisk() (packages []*Package, err error) {
	var decoder *gob.Decoder
	err = ms.bucket.Load(bucketKeyPackages, func(blob []byte) error {
		if len(blob) > 0 {
			buf := bytes.NewBuffer(blob)
			decoder = gob.NewDecoder(buf)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if decoder != nil {
		for {
			pkg := new(Package)
			err = decoder.Decode(pkg)
			if err == nil {
				packages = append(packages, pkg)
			} else if err == io.EOF {
				// Read all packages
				break
			} else {
				return nil, errors.Wrap(err, "error decoding packages")
			}
		}
	}

	return packages, nil
}

// Save packages to disk.
func (ms *MetricSet) savePackagesToDisk(packages []*Package) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, pkg := range packages {
		err := encoder.Encode(*pkg)
		if err != nil {
			return errors.Wrap(err, "error encoding packages")
		}
	}

	err := ms.bucket.Store(bucketKeyPackages, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing packages to disk")
	}
	return nil
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
		err = errors.Errorf("unknown OS %v - this should not have happened", osFamily)
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
