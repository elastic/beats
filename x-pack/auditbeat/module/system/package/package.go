// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gofrs/uuid"
	"github.com/joeshaw/multierror"
	"go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/auditbeat/datastore"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/cache"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	metricsetName = "package"
	namespace     = "system.audit.package"

	bucketNameV2            = "package.v2"
	bucketKeyPackages       = "packages"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

var (
	rpmPath            = "/var/lib/rpm"
	dpkgPath           = "/var/lib/dpkg"
	homebrewCellarPath = []string{"/usr/local/Cellar", "/opt/homebrew/Cellar"}
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

func (action eventAction) Type() string {
	switch action {
	case eventActionExistingPackage:
		return "info"
	case eventActionPackageInstalled:
		return "installation"
	case eventActionPackageRemoved:
		return "deletion"
	case eventActionPackageUpdated:
		return "change"
	default:
		return "info"
	}
}

func init() {
	mb.Registry.MustAddMetricSet(system.ModuleName, metricsetName, New,
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
	Type        string

	error error
}

// Hash creates a hash for Package.
//
//nolint:errcheck // Writing to the hash never returns an error.
func (pkg Package) Hash() uint64 {
	h := xxhash.New()
	h.WriteString(pkg.Name)
	h.WriteString(pkg.Version)
	h.WriteString(pkg.Release)
	binary.Write(h, binary.LittleEndian, pkg.Size)
	return h.Sum64()
}

func (pkg Package) toMapStr() (mapstr.M, mapstr.M) {
	nonECS := mapstr.M{
		"name":    pkg.Name,
		"version": pkg.Version,
	}
	ecs := mapstr.M{
		"name":    pkg.Name,
		"version": pkg.Version,
	}

	if pkg.Release != "" {
		nonECS["release"] = pkg.Release
	}

	if pkg.Arch != "" {
		nonECS["arch"] = pkg.Arch
		ecs["architecture"] = pkg.License
	}

	if pkg.License != "" {
		nonECS["license"] = pkg.License
		ecs["license"] = pkg.License
	}

	if !pkg.InstallTime.IsZero() {
		nonECS["installtime"] = pkg.InstallTime
		ecs["installed"] = pkg.InstallTime
	}

	if pkg.Size != 0 {
		nonECS["size"] = pkg.Size
		ecs["size"] = pkg.Size
	}

	if pkg.Summary != "" {
		nonECS["summary"] = pkg.Summary
		ecs["description"] = pkg.Summary
	}

	if pkg.URL != "" {
		nonECS["url"] = pkg.URL
		ecs["reference"] = pkg.URL
	}

	if pkg.Type != "" {
		ecs["type"] = pkg.Type
	}

	return nonECS, ecs
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
	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %v/%v config: %w", system.ModuleName, metricsetName, err)
	}

	if err := datastore.Update(migrateDatastoreSchema); err != nil {
		return nil, fmt.Errorf("datastore schema migration failed: %w", err)
	}

	bucket, err := datastore.OpenBucket(bucketNameV2)
	if err != nil {
		return nil, fmt.Errorf("failed to open persistent datastore: %w", err)
	}

	ms := &MetricSet{
		SystemMetricSet: system.NewSystemMetricSet(base),
		config:          config,
		log:             logp.NewLogger(metricsetName),
		cache:           cache.New(),
		bucket:          bucket,
	}

	ms.lastState, err = loadStateTimestamp(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to load state timestamp from bucket %v: %w", bucketNameV2, err)
	}
	if !ms.lastState.IsZero() {
		ms.log.Debugf("Last state was sent at %v. Next state update by %v.", ms.lastState, ms.lastState.Add(ms.config.effectiveStatePeriod()))
	} else {
		ms.log.Debug("No state timestamp found.")
	}

	packages, err := loadPackages(ms.bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to load persisted package metadata from disk: %w", err)
	}
	ms.log.Debugf("Loaded %d packages from disk", len(packages))

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
		return fmt.Errorf("failed to get packages: %w", err)
	}

	stateID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("error generating state ID: %w", err)
	}

	for _, pkg := range packages {
		event := ms.packageEvent(pkg, eventTypeState, eventActionExistingPackage)
		event.RootFields.Put("event.id", stateID.String()) //nolint:errcheck // This will not return an error as long as 'event' remains as a map.
		report.Event(event)
	}

	// This will initialize the cache with the current packages
	ms.cache.DiffAndUpdateCache(convertToCacheable(packages))

	if err = storeStateTimestamp(ms.bucket, ms.lastState); err != nil {
		return fmt.Errorf("error persisting state timestamp: %w", err)
	}
	return storePackages(ms.bucket, packages)
}

// reportChanges detects and reports any changes to installed packages on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	packages, err := ms.getPackages()
	if err != nil {
		return fmt.Errorf("failed to get packages: %w", err)
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
		return storePackages(ms.bucket, packages)
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
	pkgFields, ecsPkgFields := pkg.toMapStr()
	event := mb.Event{
		RootFields: mapstr.M{
			"event": mapstr.M{
				"kind":     eventType,
				"category": []string{"package"},
				"type":     []string{action.Type()},
				"action":   action.String(),
			},
			"package": ecsPkgFields,
			"message": packageMessage(pkg, action),
		},
		MetricSetFields: pkgFields,
	}

	if ms.HostID() != "" {
		event.MetricSetFields["entity_id"] = pkg.entityID(ms.HostID())
	}

	if pkg.error != nil {
		event.RootFields["error"] = mapstr.M{
			"message": pkg.error.Error(),
		}
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

// loadStateTimestamp loads state timestamp from a bucket. This is the time
// when all package state was last emitted as events.
func loadStateTimestamp(bucket datastore.Bucket) (time.Time, error) {
	var stateTimestamp time.Time
	err := bucket.Load(bucketKeyStateTimestamp, func(blob []byte) error {
		if len(blob) > 0 {
			return stateTimestamp.UnmarshalBinary(blob)
		}
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}

	return stateTimestamp, nil
}

// storeStateTimestamp stores the timestamp of the last state update to
// the given datastore bucket.
func storeStateTimestamp(bucket datastore.Bucket, t time.Time) error {
	data, err := t.MarshalBinary()
	if err != nil {
		return err
	}

	if err = bucket.Store(bucketKeyStateTimestamp, data); err != nil {
		return fmt.Errorf("error writing state timestamp to disk: %w", err)
	}
	return nil
}

// loadPackages loads the persisted packages from the given datastore bucket.
func loadPackages(bucket datastore.Bucket) (packages []*Package, err error) {
	var data []byte
	err = bucket.Load(bucketKeyPackages, func(blob []byte) error {
		data = blob
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		packages, err = decodePackagesFromContainer(data)
		if err != nil {
			return nil, err
		}
	}

	return packages, nil
}

// storePackages stores packages to the given datastore bucket.
func storePackages(bucket datastore.Bucket, packages []*Package) error {
	builder, release := fbGetBuilder()
	defer release()

	if err := bucket.Store(bucketKeyPackages, encodePackages(builder, packages)); err != nil {
		return fmt.Errorf("error persisting packages to datastore: %w", err)
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
			return nil, fmt.Errorf("error getting RPM packages: %w", err)
		}
		ms.log.Debugf("RPM packages: %v", len(rpmPackages))

		packages = append(packages, rpmPackages...)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error opening %v: %w", rpmPath, err)
	}

	_, err = os.Stat(dpkgPath)
	if err == nil {
		foundPackageManager = true

		dpkgPackages, err := ms.listDebPackages()
		if err != nil {
			return nil, fmt.Errorf("error getting DEB packages: %w", err)
		}
		ms.log.Debugf("DEB packages: %v", len(dpkgPackages))

		packages = append(packages, dpkgPackages...)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error opening %v: %w", dpkgPath, err)
	}

	for _, path := range homebrewCellarPath {
		_, err = os.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("error opening %v: %w", path, err)
		}
		foundPackageManager = true
		homebrewPackages, err := listBrewPackages(path)
		if err != nil {
			return nil, fmt.Errorf("error getting Homebrew packages: %w", err)
		}
		ms.log.Debugf("Homebrew packages: %v", len(homebrewPackages))
		packages = append(packages, homebrewPackages...)
		break
	}

	if !foundPackageManager && !ms.suppressNoPackageWarnings {
		ms.log.Warnf("No supported package managers found. None of %v, %v, %v exist.",
			rpmPath, dpkgPath, strings.Join(homebrewCellarPath, ","))

		// Only warn once at the start of Auditbeat.
		ms.suppressNoPackageWarnings = true
	}

	return packages, nil
}

func (ms *MetricSet) listDebPackages() ([]*Package, error) {
	dpkgStatusFile := filepath.Join(dpkgPath, "status")

	file, err := os.Open(dpkgStatusFile)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %w", dpkgStatusFile, err)
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
			pkg = &Package{
				Type: "dpkg",
			}
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
			pkg.Size, err = parseDpkgInstalledSize(value)
			if err != nil {
				// If installed size is invalid, log a warning but still
				// report the package with size=0.
				ms.log.Warnw("Failed parsing installed size",
					"package", pkg.Name,
					"Installed-Size", value,
					"Error", err)
			}
		case "homepage":
			pkg.URL = value
		default:
			continue
		}
	}

	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %v: %w", dpkgStatusFile, err)
	}

	// Append last package if file ends without newline
	if pkg != nil && !skipPackage {
		packages = append(packages, pkg)
	}

	return packages, nil
}

func parseDpkgInstalledSize(value string) (size uint64, err error) {
	// Installed-Size is an integer (KiB).
	if size, err = strconv.ParseUint(value, 10, 64); err == nil {
		return size, nil
	}

	// Some rare third-party packages contain a unit at the end. This is ignored
	// by dpkg tools. Try to parse to return a value as close as possible
	// to what the package maintainer meant.
	end := len(value)
	for idx, chr := range value {
		if chr < '0' || chr > '9' {
			end = idx
			break
		}
	}
	multiplier := uint64(1)
	if end < len(value) {
		switch value[end] {
		case 'm', 'M':
			multiplier = 1024
		case 'g', 'G':
			multiplier = 1024 * 1024
		}
	}

	size, err = strconv.ParseUint(value[:end], 10, 64)
	return size * multiplier, err
}

// packageV1 is the struct used in packages.v1.
// Do not modify this struct because this must remain the same as what
// was used in earlier Auditbeat releases.
type packageV1 struct {
	Name        string
	Version     string
	Release     string
	Arch        string
	License     string
	InstallTime time.Time
	Size        uint64
	Summary     string
	URL         string
	Type        string

	//nolint:unused // This field is unused, but we are keeping this struct as is.
	error error
}

// migrateDatastoreSchema migrates the contents of the data store to the latest
// schema. This allows users of earlier versions of Auditbeat to upgrade to
// new versions while maintaining existing state. This handles migrating data
// from the package.v1 bucket into package.v2.
//
// It performs the migration entirely within the given write transaction such
// that if any problems occur the changes are rolled back.
func migrateDatastoreSchema(tx *bbolt.Tx) error {
	const bucketNameV1 = "package.v1"

	v2Bucket := tx.Bucket([]byte(bucketNameV2))
	if v2Bucket != nil {
		// Already exists. No need to migrate.
		return nil
	}

	v1Bucket := tx.Bucket([]byte(bucketNameV1))
	if v1Bucket == nil {
		// No old data to migrate.
		return nil
	}

	log := logp.NewLogger(metricsetName)
	log.Debugf("Migrating data from %v to %v bucket.", bucketNameV1, bucketNameV2)

	var timestampGob []byte
	if timestampGob = v1Bucket.Get([]byte(bucketKeyStateTimestamp)); len(timestampGob) == 0 {
		return fmt.Errorf("error migrating %v data: no timestamp found", bucketNameV1)
	}

	var packages []*Package
	if data := v1Bucket.Get([]byte(bucketKeyPackages)); len(data) > 0 {
		dec := gob.NewDecoder(bytes.NewReader(data))

		for {
			var pkgV1 packageV1
			if err := dec.Decode(&pkgV1); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("error migrating %v data: failed decoding packages: %w", bucketNameV1, err)
			}
			packages = append(packages, &Package{
				Name:        pkgV1.Name,
				Version:     pkgV1.Version,
				Release:     pkgV1.Release,
				Arch:        pkgV1.Arch,
				License:     pkgV1.License,
				InstallTime: pkgV1.InstallTime,
				Size:        pkgV1.Size,
				Summary:     pkgV1.Summary,
				URL:         pkgV1.URL,
				Type:        pkgV1.Type,
			})
		}
	}

	v2Bucket, err := tx.CreateBucketIfNotExists([]byte(bucketNameV2))
	if err != nil {
		return fmt.Errorf("error migrating data: failed to create %v bucket: %w", bucketNameV2, err)
	}

	if err = v2Bucket.Put([]byte(bucketKeyStateTimestamp), timestampGob); err != nil {
		return fmt.Errorf("error migrating data: failed to write %v to %v bucket: %w", bucketKeyStateTimestamp, bucketNameV2, err)
	}

	builder, release := fbGetBuilder()
	defer release()
	if err = v2Bucket.Put([]byte(bucketKeyPackages), encodePackages(builder, packages)); err != nil {
		return fmt.Errorf("error migrating data: failed to write %v to %v bucket: %w", bucketKeyPackages, bucketNameV2, err)
	}

	if err = tx.DeleteBucket([]byte(bucketNameV1)); err != nil {
		return fmt.Errorf("error migrating data: failed to delete %v bucket: %w", bucketNameV1, err)
	}

	log.Debugf("Completed migrating data from %v to %v bucket. Moved %d packages.", bucketNameV1, bucketNameV2, len(packages))
	return nil
}
