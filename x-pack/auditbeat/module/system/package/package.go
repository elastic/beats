// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package pkg

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gofrs/uuid"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/cache"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
)

const (
	moduleName    = "system"
	metricsetName = "package"
	namespace     = "system.audit.package"

	bucketName              = "package.v1"
	bucketKeyPackages       = "packages"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

var (
	rpmPath            = "/var/lib/rpm"
	dpkgPath           = "/var/lib/dpkg"
	homebrewCellarPath = "/usr/local/Cellar"
)

type eventAction uint8

const (
	eventActionExistingPackage eventAction = iota
	eventActionPackageInstalled
	eventActionPackageRemoved
	eventActionPackageUpdated
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingPackage:
		return "existing_package"
	case eventActionPackageInstalled:
		return "package_installed"
	case eventActionPackageRemoved:
		return "package_removed"
	case eventActionPackageUpdated:
		return "package_updated"
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
	system.SystemMetricSet
	config    config
	log       *logp.Logger
	cache     *cache.Cache
	bucket    datastore.Bucket
	lastState time.Time

	suppressNoPackageWarnings bool
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
	Error       error
}

// Hash creates a hash for Package.
func (pkg Package) Hash() uint64 {
	h := xxhash.New()
	h.WriteString(pkg.Name)
	h.WriteString(pkg.Version)
	h.WriteString(pkg.Release)
	binary.Write(h, binary.LittleEndian, pkg.Size)
	return h.Sum64()
}

func (pkg Package) toMapStr() common.MapStr {
	mapstr := common.MapStr{
		"name":    pkg.Name,
		"version": pkg.Version,
	}

	if pkg.Release != "" {
		mapstr.Put("release", pkg.Release)
	}

	if pkg.Arch != "" {
		mapstr.Put("arch", pkg.Arch)
	}

	if pkg.License != "" {
		mapstr.Put("license", pkg.License)
	}

	if !pkg.InstallTime.IsZero() {
		mapstr.Put("installtime", pkg.InstallTime)
	}

	if pkg.Size != 0 {
		mapstr.Put("size", pkg.Size)
	}

	if pkg.Summary != "" {
		mapstr.Put("summary", pkg.Summary)
	}

	if pkg.URL != "" {
		mapstr.Put("url", pkg.URL)
	}

	return mapstr
}

// entityID creates an ID that uniquely identifies this package across machines.
func (pkg Package) entityID(hostID string) string {
	h := system.NewEntityHash()
	h.Write([]byte(hostID))
	h.Write([]byte(pkg.Name))
	h.Write([]byte(pkg.Version))
	return h.Sum()
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", moduleName, metricsetName)

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrapf(err, "failed to unpack the %v/%v config", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		SystemMetricSet: system.NewSystemMetricSet(base),
		config:          config,
		log:             logp.NewLogger(metricsetName),
		cache:           cache.New(),
		bucket:          bucket,
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
	var errs multierror.Errors

	errs = append(errs, closeDataset())

	if ms.bucket != nil {
		errs = append(errs, ms.bucket.Close())
	}

	return errs.Err()
}

// Fetch collects data about the host. It is invoked periodically.
func (ms *MetricSet) Fetch(report mb.ReporterV2) {
	needsStateUpdate := time.Since(ms.lastState) > ms.config.effectiveStatePeriod()
	if needsStateUpdate {
		ms.log.Debug("Sending state")
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

	packages, err := ms.getPackages()
	if err != nil {
		return errors.Wrap(err, "failed to get packages")
	}

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}
	for _, pkg := range packages {
		event := ms.packageEvent(pkg, eventTypeState, eventActionExistingPackage)
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
	packages, err := ms.getPackages()
	if err != nil {
		return errors.Wrap(err, "failed to get packages")
	}

	newInCache, missingFromCache := ms.cache.DiffAndUpdateCache(convertToCacheable(packages))
	newPackages := convertToPackage(newInCache)
	missingPackages := convertToPackage(missingFromCache)

	// Package names of updated packages
	updated := make(map[string]struct{})

	for _, missingPkg := range missingPackages {
		found := false

		// Using an inner loop is less efficient than using a map, but in this case
		// we do not expect a lot of installed or removed packages all at once.
		for _, newPkg := range newPackages {
			if missingPkg.Name == newPkg.Name {
				found = true
				updated[newPkg.Name] = struct{}{}
				report.Event(ms.packageEvent(newPkg, eventTypeEvent, eventActionPackageUpdated))
				break
			}
		}

		if !found {
			report.Event(ms.packageEvent(missingPkg, eventTypeEvent, eventActionPackageRemoved))
		}
	}

	for _, newPkg := range newPackages {
		if _, contains := updated[newPkg.Name]; !contains {
			report.Event(ms.packageEvent(newPkg, eventTypeEvent, eventActionPackageInstalled))
		}
	}

	if len(newPackages) > 0 || len(missingPackages) > 0 {
		return ms.savePackagesToDisk(packages)
	}

	return nil
}

func convertToPackage(cacheValues []interface{}) []*Package {
	packages := make([]*Package, 0, len(cacheValues))

	for _, c := range cacheValues {
		packages = append(packages, c.(*Package))
	}

	return packages
}

func (ms *MetricSet) packageEvent(pkg *Package, eventType string, action eventAction) mb.Event {
	event := mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"kind":   eventType,
				"action": action.String(),
			},
			"message": packageMessage(pkg, action),
		},
		MetricSetFields: pkg.toMapStr(),
	}

	if ms.HostID() != "" {
		event.MetricSetFields.Put("entity_id", pkg.entityID(ms.HostID()))
	}

	if pkg.Error != nil {
		event.RootFields.Put("error.message", pkg.Error.Error())
	}

	return event
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
	case eventActionPackageUpdated:
		actionString = "updated"
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

func (ms *MetricSet) getPackages() (packages []*Package, err error) {
	var foundPackageManager bool

	_, err = os.Stat(rpmPath)
	if err == nil {
		foundPackageManager = true

		rpmPackages, err := listRPMPackages()
		if err != nil {
			return nil, errors.Wrap(err, "error getting RPM packages")
		}
		ms.log.Debugf("RPM packages: %v", len(rpmPackages))

		packages = append(packages, rpmPackages...)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "error opening %v", rpmPath)
	}

	_, err = os.Stat(dpkgPath)
	if err == nil {
		foundPackageManager = true

		dpkgPackages, err := listDebPackages()
		if err != nil {
			return nil, errors.Wrap(err, "error getting DEB packages")
		}
		ms.log.Debugf("DEB packages: %v", len(dpkgPackages))

		packages = append(packages, dpkgPackages...)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "error opening %v", dpkgPath)
	}

	_, err = os.Stat(homebrewCellarPath)
	if err == nil {
		foundPackageManager = true

		homebrewPackages, err := listBrewPackages()
		if err != nil {
			return nil, errors.Wrap(err, "error getting Homebrew packages")
		}
		ms.log.Debugf("Homebrew packages: %v", len(homebrewPackages))

		packages = append(packages, homebrewPackages...)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "error opening %v", homebrewCellarPath)
	}

	if !foundPackageManager && !ms.suppressNoPackageWarnings {
		ms.log.Warnf("No supported package managers found. None of %v, %v, %v exist.",
			rpmPath, dpkgPath, homebrewCellarPath)

		// Only warn once at the start of Auditbeat.
		ms.suppressNoPackageWarnings = true
	}

	return packages, nil
}

func listDebPackages() ([]*Package, error) {
	dpkgStatusFile := filepath.Join(dpkgPath, "status")

	file, err := os.Open(dpkgStatusFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening %s", dpkgStatusFile)
	}
	defer file.Close()

	var packages []*Package
	var skipPackage bool
	var pkg *Package
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			// empty line signals new package
			if !skipPackage {
				packages = append(packages, pkg)
			}
			skipPackage = false
			pkg = nil
			continue
		} else if skipPackage {
			// Skipping this package - read on.
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

		if pkg == nil {
			pkg = &Package{}
		}

		switch strings.ToLower(words[0]) {
		case "package":
			pkg.Name = value
		case "status":
			if strings.HasPrefix(value, "deinstall ok") {
				// Package was removed but not purged. We report both cases as removed.
				skipPackage = true
			}
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
		case "homepage":
			pkg.URL = value
		default:
			continue
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "error scanning file %v", dpkgStatusFile)
	}

	// Append last package if file ends without newline
	if pkg != nil && !skipPackage {
		packages = append(packages, pkg)
	}

	return packages, nil
}
