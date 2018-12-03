// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package user

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"syscall"
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
)

const (
	moduleName    = "system"
	metricsetName = "user"

	bucketName              = "user.v1"
	bucketKeyUsers          = "users"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"

	eventActionExistingUser    = "existing_user"
	eventActionUserAdded       = "user_added"
	eventActionUserRemoved     = "user_removed"
	eventActionUserChanged     = "user_changed"
	eventActionPasswordChanged = "password_changed"
)

// User represents a user. Fields according to getpwent(3).
type User struct {
	Name             string
	PasswordType     string
	PasswordChanged  time.Time
	PasswordHashHash []byte
	UID              uint32
	GID              uint32
	Groups           []Group
	UserInfo         string
	Dir              string
	Shell            string
	Action           string
}

// Group contains information about a group.
type Group struct {
	Name string
	GID  uint32
}

// Hash creates a hash for User.
func (user User) Hash() uint64 {
	h := xxhash.New64()
	// Use everything except userInfo
	h.WriteString(user.Name)
	h.WriteString(user.PasswordType)
	h.WriteString(user.PasswordChanged.String())
	h.Write(user.PasswordHashHash)
	h.WriteString(strconv.Itoa(int(user.UID)))
	h.WriteString(strconv.Itoa(int(user.GID)))
	h.WriteString(user.Dir)
	h.WriteString(user.Shell)

	for _, group := range user.Groups {
		h.WriteString(group.Name)
		h.WriteString(strconv.Itoa(int(group.GID)))
	}

	return h.Sum64()
}

func (user User) toMapStr() common.MapStr {
	evt := common.MapStr{
		"name": user.Name,
		"password": common.MapStr{
			"type": user.PasswordType,
		},
		"uid":   user.UID,
		"gid":   user.GID,
		"dir":   user.Dir,
		"shell": user.Shell,
	}

	if user.UserInfo != "" {
		evt.Put("user_information", user.UserInfo)
	}

	if !user.PasswordChanged.IsZero() {
		evt.Put("password.last_changed", user.PasswordChanged)
	}

	if len(user.Groups) > 0 {
		var groupMapStr []common.MapStr
		for _, group := range user.Groups {
			groupMapStr = append(groupMapStr, common.MapStr{
				"name": group.Name,
				"gid":  group.GID,
			})
		}
		evt.Put("group", groupMapStr)
	}

	return evt
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about a system's users.
type MetricSet struct {
	mb.BaseMetricSet
	config     Config
	log        *logp.Logger
	cache      *cache.Cache
	bucket     datastore.Bucket
	lastState  time.Time
	lastChange time.Time
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("the %v/%v dataset is only supported on Linux", moduleName, metricsetName)
	}

	config := defaultConfig
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

	// Load from disk: Users
	users, err := ms.restoreUsersFromDisk()
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore users from disk")
	}
	ms.log.Debugf("Restored %d users from disk", len(users))

	ms.cache.DiffAndUpdateCache(convertToCacheable(users))

	return ms, nil
}

// Close cleans up the MetricSet when it finishes.
func (ms *MetricSet) Close() error {
	if ms.bucket != nil {
		return ms.bucket.Close()
	}
	return nil
}

// Fetch collects the user information. It is invoked periodically.
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

// reportState reports all existing users on the system.
func (ms *MetricSet) reportState(report mb.ReporterV2) error {
	ms.lastState = time.Now()

	users, err := GetUsers()
	if err != nil {
		return errors.Wrap(err, "failed to get users")
	}
	ms.log.Debugf("Found %v users", len(users))

	stateID, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "error generating state ID")
	}
	for _, user := range users {
		event := userEvent(user, eventTypeState, eventActionExistingUser)
		event.RootFields.Put("event.id", stateID.String())
		report.Event(event)
	}

	if ms.cache != nil {
		// This will initialize the cache with the current processes
		ms.cache.DiffAndUpdateCache(convertToCacheable(users))
	}

	// Save time so we know when to send the state again (config.StatePeriod)
	timeBytes, err := ms.lastState.MarshalBinary()
	if err != nil {
		return err
	}
	err = ms.bucket.Store(bucketKeyStateTimestamp, timeBytes)
	if err != nil {
		return errors.Wrap(err, "error writing state timestamp to disk")
	}

	return ms.saveUsersToDisk(users)
}

// reportChanges detects and reports any changes to users on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	currentTime := time.Now()
	changed, err := haveFilesChanged(ms.lastChange)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	ms.lastChange = currentTime

	users, err := GetUsers()
	if err != nil {
		return errors.Wrap(err, "failed to get users")
	}
	ms.log.Debugf("Found %v users", len(users))

	newInCache, missingFromCache := ms.cache.DiffAndUpdateCache(convertToCacheable(users))

	if len(newInCache) > 0 && len(missingFromCache) > 0 {
		// Check for changes to users
		missingUserMap := make(map[uint32](*User))
		for _, missingUser := range missingFromCache {
			missingUserMap[missingUser.(*User).UID] = missingUser.(*User)
		}

		for _, userFromCache := range newInCache {
			newUser := userFromCache.(*User)
			matchingMissingUser, found := missingUserMap[newUser.UID]

			if found {
				// Report password change separately
				if newUser.PasswordChanged.Before(matchingMissingUser.PasswordChanged) ||
					!bytes.Equal(newUser.PasswordHashHash, matchingMissingUser.PasswordHashHash) ||
					newUser.PasswordType != matchingMissingUser.PasswordType {
					report.Event(userEvent(newUser, eventTypeEvent, eventActionPasswordChanged))
				}

				// Hack to check if only the password changed
				matchingMissingUser.PasswordChanged = newUser.PasswordChanged
				matchingMissingUser.PasswordHashHash = newUser.PasswordHashHash
				matchingMissingUser.PasswordType = newUser.PasswordType
				if newUser.Hash() != matchingMissingUser.Hash() {
					report.Event(userEvent(newUser, eventTypeEvent, eventActionUserChanged))
				}

				delete(missingUserMap, matchingMissingUser.UID)
			} else {
				report.Event(userEvent(newUser, eventTypeEvent, eventActionUserAdded))
			}
		}

		for _, missingUser := range missingUserMap {
			report.Event(userEvent(missingUser, eventTypeEvent, eventActionUserRemoved))
		}
	} else {
		// No changes to users
		for _, user := range newInCache {
			report.Event(userEvent(user.(*User), eventTypeEvent, eventActionUserAdded))
		}

		for _, user := range missingFromCache {
			report.Event(userEvent(user.(*User), eventTypeEvent, eventActionUserRemoved))
		}
	}

	if len(newInCache) > 0 || len(missingFromCache) > 0 {
		return ms.saveUsersToDisk(users)
	}

	return nil
}

func userEvent(user *User, eventType string, eventAction string) mb.Event {
	return mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"type":   eventType,
				"action": eventAction,
			},
			"user": common.MapStr{
				"id":   user.UID,
				"name": user.Name,
			},
		},
		MetricSetFields: user.toMapStr(),
	}
}

func convertToCacheable(users []*User) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(users))

	for _, u := range users {
		c = append(c, u)
	}

	return c
}

// restoreUsersFromDisk loads the user cache from disk.
func (ms *MetricSet) restoreUsersFromDisk() (users []*User, err error) {
	var decoder *gob.Decoder
	err = ms.bucket.Load(bucketKeyUsers, func(blob []byte) error {
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
			user := new(User)
			err = decoder.Decode(user)
			if err == nil {
				users = append(users, user)
			} else if err == io.EOF {
				// Read all users
				break
			} else {
				return nil, errors.Wrap(err, "error decoding users")
			}
		}
	}

	return users, nil
}

// Save user cache to disk.
func (ms *MetricSet) saveUsersToDisk(users []*User) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, user := range users {
		err := encoder.Encode(*user)
		if err != nil {
			return errors.Wrap(err, "error encoding users")
		}
	}

	err := ms.bucket.Store(bucketKeyUsers, buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing users to disk")
	}
	return nil
}

// haveFilesChanged checks if any of the relevant files (/etc/passwd, /etc/shadow, /etc/group)
// have changed.
func haveFilesChanged(since time.Time) (bool, error) {
	const passwdFile = "/etc/passwd"
	const shadowFile = "/etc/shadow"
	const groupFile = "/etc/group"

	var stats syscall.Stat_t
	if err := syscall.Stat(passwdFile, &stats); err != nil {
		return true, errors.Wrapf(err, "failed to stat %v", passwdFile)
	}
	if since.Before(time.Unix(stats.Ctim.Sec, stats.Ctim.Nsec)) {
		return true, nil
	}

	if err := syscall.Stat(shadowFile, &stats); err != nil {
		return true, errors.Wrapf(err, "failed to stat %v", shadowFile)
	}
	if since.Before(time.Unix(stats.Ctim.Sec, stats.Ctim.Nsec)) {
		return true, nil
	}

	if err := syscall.Stat(groupFile, &stats); err != nil {
		return true, errors.Wrapf(err, "failed to stat %v", groupFile)
	}
	if since.Before(time.Unix(stats.Ctim.Sec, stats.Ctim.Nsec)) {
		return true, nil
	}

	return false, nil
}
