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
	"github.com/elastic/beats/x-pack/auditbeat/module/system"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	moduleName    = "system"
	metricsetName = "package"
	namespace     = "system.audit.package"

	redhat = "redhat"
	suse   = "suse"
	debian = "debian"
	darwin = "darwin"

	dpkgStatusFile     = "/var/lib/dpkg/status"
	homebrewCellarPath = "/usr/local/Cellar"

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
	Error       error
}

// Hash creates a hash for Package.
func (pkg Package) Hash() uint64 {
	h := xxhash.New64()
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

	osInfo, err := getOS()
	if err != nil {
		return nil, errors.Wrap(err, "error determining operating system")
	}
	ms.osFamily = osInfo.Family
	switch osInfo.Family {
	case redhat, suse:
		// ok
	case debian:
		if _, err := os.Stat(dpkgStatusFile); err != nil {
			return nil, errors.Wrapf(err, "error looking up %s", dpkgStatusFile)
		}
	case darwin:
		if _, err := os.Stat(homebrewCellarPath); err != nil {
			return nil, errors.Wrapf(err, "error looking up %s - is Homebrew installed?", homebrewCellarPath)
		}
	default:
		return nil, fmt.Errorf("this metricset does not support OS family %v", osInfo.Family)
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
	packages, err := getPackages(ms.osFamily)
	if err != nil {
		return errors.Wrap(err, "failed to get packages")
	}
	ms.log.Debugf("Found %v packages", len(packages))

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

	event.MetricSetFields.Put("entity_id", pkg.entityID(ms.HostID()))

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

func getPackages(osFamily string) (packages []*Package, err error) {
	switch osFamily {
	case redhat, suse:
		packages, err = listRPMPackages()
		if err != nil {
			err = errors.Wrap(err, "error getting RPM packages")
		}
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
	file, err := os.Open(dpkgStatusFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening %s", dpkgStatusFile)
	}
	defer file.Close()

	var packages []*Package
	var skipPackage bool
	pkg := &Package{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			// empty line signals new package
			if !skipPackage {
				packages = append(packages, pkg)
			}
			skipPackage = false
			pkg = &Package{}
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
		default:
			continue
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "error scanning file %v", dpkgStatusFile)
	}
	return packages, nil
}
