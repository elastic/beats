// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package user

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/OneOfOne/xxhash"
	"github.com/gofrs/uuid"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
	"github.com/elastic/beats/x-pack/auditbeat/module/system"
)

const (
	moduleName    = "system"
	metricsetName = "user"
	namespace     = "system.audit.user"

	passwdFile = "/etc/passwd"
	groupFile  = "/etc/group"
	shadowFile = "/etc/shadow"

	bucketName              = "user.v1"
	bucketKeyUsers          = "users"
	bucketKeyStateTimestamp = "state_timestamp"

	eventTypeState = "state"
	eventTypeEvent = "event"
)

type eventAction uint8

const (
	eventActionExistingUser eventAction = iota
	eventActionUserAdded
	eventActionUserRemoved
	eventActionUserChanged
	eventActionPasswordChanged
)

func (action eventAction) String() string {
	switch action {
	case eventActionExistingUser:
		return "existing_user"
	case eventActionUserAdded:
		return "user_added"
	case eventActionUserRemoved:
		return "user_removed"
	case eventActionUserChanged:
		return "user_changed"
	case eventActionPasswordChanged:
		return "password_changed"
	default:
		return ""
	}
}

type passwordType uint8

const (
	detectionDisabled passwordType = iota
	shadowPassword
	passwordDisabled
	noPassword
	cryptPassword
)

func (t passwordType) String() string {
	switch t {
	case shadowPassword:
		return "shadow_password"
	case passwordDisabled:
		return "password_disabled"
	case noPassword:
		return "no_password"
	case cryptPassword:
		return "crypt_password"
	default:
		return ""
	}
}

// User represents a user. Fields according to getpwent(3).
type User struct {
	Name             string
	PasswordType     passwordType
	PasswordChanged  time.Time
	PasswordHashHash []byte
	UID              string
	GID              string
	Groups           []*user.Group
	UserInfo         string
	Dir              string
	Shell            string
	Action           string
}

// Hash creates a hash for User.
func (user User) Hash() uint64 {
	h := xxhash.New64()
	// Use everything except userInfo
	h.WriteString(user.Name)
	binary.Write(h, binary.BigEndian, uint8(user.PasswordType))
	h.WriteString(user.PasswordChanged.String())
	h.Write(user.PasswordHashHash)
	h.WriteString(user.UID)
	h.WriteString(user.GID)
	h.WriteString(user.Dir)
	h.WriteString(user.Shell)

	for _, group := range user.Groups {
		h.WriteString(group.Name)
		h.WriteString(group.Gid)
	}

	return h.Sum64()
}

func (user User) toMapStr() common.MapStr {
	evt := common.MapStr{
		"name":  user.Name,
		"uid":   user.UID,
		"gid":   user.GID,
		"dir":   user.Dir,
		"shell": user.Shell,
	}

	if user.UserInfo != "" {
		evt.Put("user_information", user.UserInfo)
	}

	if user.PasswordType != detectionDisabled {
		evt.Put("password.type", user.PasswordType.String())
	}

	if !user.PasswordChanged.IsZero() {
		evt.Put("password.last_changed", user.PasswordChanged)
	}

	if len(user.Groups) > 0 {
		var groupMapStr []common.MapStr
		for _, group := range user.Groups {
			groupMapStr = append(groupMapStr, common.MapStr{
				"name": group.Name,
				"gid":  group.Gid,
			})
		}
		evt.Put("group", groupMapStr)
	}

	return evt
}

// entityID creates an ID that uniquely identifies this user across machines.
func (u User) entityID(hostID string) string {
	h := system.NewEntityHash()
	h.Write([]byte(hostID))
	h.Write([]byte(u.Name))
	h.Write([]byte(u.UID))
	return h.Sum()
}

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithNamespace(namespace),
	)
}

// MetricSet collects data about a system's users.
type MetricSet struct {
	system.SystemMetricSet
	config    config
	log       *logp.Logger
	cache     *cache.Cache
	bucket    datastore.Bucket
	lastState time.Time
	userFiles []string
	lastRead  time.Time
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The %v/%v dataset is beta", moduleName, metricsetName)
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("the %v/%v dataset is only supported on Linux", moduleName, metricsetName)
	}

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

	if ms.config.DetectPasswordChanges {
		ms.userFiles = []string{passwdFile, groupFile, shadowFile}
	} else {
		ms.userFiles = []string{passwdFile, groupFile}
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
	var errs multierror.Errors
	ms.lastState = time.Now()

	users, err := GetUsers(ms.config.DetectPasswordChanges)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "error while getting users"))
	}

	ms.log.Debugf("Found %v users", len(users))
	if len(users) > 0 {
		stateID, err := uuid.NewV4()
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error generating state ID"))
		}

		for _, user := range users {
			event := ms.userEvent(user, eventTypeState, eventActionExistingUser)
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
			errs = append(errs, err)
		} else {
			err = ms.bucket.Store(bucketKeyStateTimestamp, timeBytes)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "error writing state timestamp to disk"))
			}
		}

		err = ms.saveUsersToDisk(users)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs.Err()
}

// reportChanges detects and reports any changes to users on this system since the last call.
func (ms *MetricSet) reportChanges(report mb.ReporterV2) error {
	var errs multierror.Errors
	currentTime := time.Now()

	// If this is not the first call to Fetch/reportChanges,
	// check if files have changed since the last time before going any further.
	if !ms.lastRead.IsZero() {
		changed, err := ms.haveFilesChanged()
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
	}
	ms.lastRead = currentTime

	users, err := GetUsers(ms.config.DetectPasswordChanges)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "error while getting users"))
	}
	ms.log.Debugf("Found %v users", len(users))

	if len(users) > 0 {
		newInCache, missingFromCache := ms.cache.DiffAndUpdateCache(convertToCacheable(users))

		if len(newInCache) > 0 && len(missingFromCache) > 0 {
			// Check for changes to users
			missingUserMap := make(map[string](*User))
			for _, missingUser := range missingFromCache {
				missingUserMap[missingUser.(*User).UID] = missingUser.(*User)
			}

			for _, userFromCache := range newInCache {
				newUser := userFromCache.(*User)
				oldUser, found := missingUserMap[newUser.UID]

				if found {
					// Report password change separately
					if ms.config.DetectPasswordChanges && newUser.PasswordType != detectionDisabled &&
						oldUser.PasswordType != detectionDisabled {

						passwordChanged := newUser.PasswordChanged.Before(oldUser.PasswordChanged) ||
							!bytes.Equal(newUser.PasswordHashHash, oldUser.PasswordHashHash) ||
							newUser.PasswordType != oldUser.PasswordType

						if passwordChanged {
							report.Event(ms.userEvent(newUser, eventTypeEvent, eventActionPasswordChanged))
						}
					}

					// Hack to check if only the password changed
					oldUser.PasswordChanged = newUser.PasswordChanged
					oldUser.PasswordHashHash = newUser.PasswordHashHash
					oldUser.PasswordType = newUser.PasswordType
					if newUser.Hash() != oldUser.Hash() {
						report.Event(ms.userEvent(newUser, eventTypeEvent, eventActionUserChanged))
					}

					delete(missingUserMap, oldUser.UID)
				} else {
					report.Event(ms.userEvent(newUser, eventTypeEvent, eventActionUserAdded))
				}
			}

			for _, missingUser := range missingUserMap {
				report.Event(ms.userEvent(missingUser, eventTypeEvent, eventActionUserRemoved))
			}
		} else {
			// No changes to users
			for _, user := range newInCache {
				report.Event(ms.userEvent(user.(*User), eventTypeEvent, eventActionUserAdded))
			}

			for _, user := range missingFromCache {
				report.Event(ms.userEvent(user.(*User), eventTypeEvent, eventActionUserRemoved))
			}
		}

		if len(newInCache) > 0 || len(missingFromCache) > 0 {
			err = ms.saveUsersToDisk(users)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs.Err()
}

func (ms *MetricSet) userEvent(user *User, eventType string, action eventAction) mb.Event {
	return mb.Event{
		RootFields: common.MapStr{
			"event": common.MapStr{
				"kind":   eventType,
				"action": action.String(),
			},
			"user": common.MapStr{
				"entity_id": user.entityID(ms.HostID()),
				"id":        user.UID,
				"name":      user.Name,
			},
			"message": userMessage(user, action),
		},
		MetricSetFields: user.toMapStr(),
	}
}

func userMessage(user *User, action eventAction) string {
	var actionString string
	switch action {
	case eventActionExistingUser:
		actionString = "Existing"
	case eventActionUserAdded:
		actionString = "New"
	case eventActionUserRemoved:
		actionString = "Removed"
	case eventActionUserChanged:
		actionString = "Changed"
	case eventActionPasswordChanged:
		actionString = "Password changed for"
	}

	return fmt.Sprintf("%v user %v (UID: %v, Groups: %v)",
		actionString, user.Name, user.UID, fmtGroups(user.Groups))
}

func fmtGroups(groups []*user.Group) string {
	var b strings.Builder

	if len(groups) > 0 {
		b.WriteString(groups[0].Name)
		for _, group := range groups[1:] {
			b.WriteString(",")
			b.WriteString(group.Name)
		}
	}

	return b.String()
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

// haveFilesChanged checks if the ctime of any of the user files has changed.
func (ms *MetricSet) haveFilesChanged() (bool, error) {
	var stats syscall.Stat_t
	for _, path := range ms.userFiles {
		if err := syscall.Stat(path, &stats); err != nil {
			return true, errors.Wrapf(err, "failed to stat %v", path)
		}

		ctime := time.Unix(int64(stats.Ctim.Sec), int64(stats.Ctim.Nsec))
		if ms.lastRead.Before(ctime) {
			ms.log.Debugf("File changed: %v (lastRead=%v, ctime=%v)", path, ms.lastRead, ctime)

			return true, nil
		}
	}

	return false, nil
}
