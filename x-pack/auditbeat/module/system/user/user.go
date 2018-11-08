// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package user

import (
	"bytes"
	"encoding/gob"
	"io"
	"runtime"
	"strconv"

	"github.com/pkg/errors"

	"github.com/OneOfOne/xxhash"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
)

const (
	moduleName    = "system"
	metricsetName = "user"
	bucketName    = "user.v1"

	eventTypeState = "state"
	eventTypeEvent = "event"

	eventActionUserExists  = "existing_user"
	eventActionUserAdded   = "user_added"
	eventActionUserRemoved = "user_removed"
	eventActionUserChanged = "user_changed"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet collects data about a system's users.
type MetricSet struct {
	mb.BaseMetricSet
	log    *logp.Logger
	cache  *cache.Cache
	bucket datastore.Bucket
}

// User represents a user. Fields according to getpwent(3).
type User struct {
	Name     string
	Passwd   string
	UID      uint32
	GID      uint32
	UserInfo string
	Dir      string
	Shell    string
}

// Hash creates a hash for User.
func (user User) Hash() uint64 {
	h := xxhash.New64()
	// Use everything except userInfo
	h.WriteString(user.Name)
	h.WriteString(user.Passwd)
	h.WriteString(strconv.Itoa(int(user.UID)))
	h.WriteString(strconv.Itoa(int(user.GID)))
	h.WriteString(user.Dir)
	h.WriteString(user.Shell)
	return h.Sum64()
}

func (user User) toMapStr() common.MapStr {
	evt := common.MapStr{
		"name":   user.Name,
		"passwd": user.Passwd,
		"uid":    user.UID,
		"gid":    user.GID,
		"dir":    user.Dir,
		"shell":  user.Shell,
	}

	if user.UserInfo != "" {
		evt.Put("user_information", user.UserInfo)
	}

	return evt
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)
	if runtime.GOOS == "windows" {
		return nil, errors.New("the %v/%v dataset is not supported on Windows", moduleName, metricsetName)
	}

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(metricsetName),
		cache:         cache.New(),
		bucket:        bucket,
	}

	users, err := ms.restoreUsersFromDisk()
	if err != nil {
		return nil, errors.Wrap(err, "failed to restore users from disk")
	}
	ms.log.Debugf("Restored %d users from disk", len(users))

	ms.log.Debugf("%v", users)
	ms.cache.DiffAndUpdateCache(convertToCacheable(users))

	return ms, nil
}

// restoreUsersFromDisk loads the user cache from disk.
func (ms *MetricSet) restoreUsersFromDisk() (users []*User, err error) {
	var decoder *gob.Decoder
	err = ms.bucket.Load("users", func(blob []byte) error {
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
				return nil, errors.Wrap(err, "decode error")
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

	err := ms.bucket.Store("users", buf.Bytes())
	if err != nil {
		return errors.Wrap(err, "error writing users to disk")
	}
	return nil
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
	users, err := GetUsers()
	if err != nil {
		errW := errors.Wrap(err, "Failed to get users")
		ms.log.Error(errW)
		report.Error(errW)
		return
	}

	if ms.cache != nil && !ms.cache.IsEmpty() {
		added, removed, changed := ms.compareUsers(users)

		for _, user := range added {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":   eventTypeEvent,
					"event.action": eventActionUserAdded,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		for _, user := range removed {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":   eventTypeEvent,
					"event.action": eventActionUserRemoved,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		for _, user := range changed {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":   eventTypeEvent,
					"event.action": eventActionUserChanged,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
			err := ms.saveUsersToDisk(users)
			if err != nil {
				ms.log.Error(err)
				report.Error(err)
			}
		}
	} else {
		// Report all existing users
		for _, user := range users {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":   eventTypeState,
					"event.action": eventActionUserExists,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		if ms.cache != nil {
			// This will initialize the cache with the current processes
			ms.cache.DiffAndUpdateCache(convertToCacheable(users))
		}

		err := ms.saveUsersToDisk(users)
		if err != nil {
			ms.log.Error(err)
			report.Error(err)
		}
	}
}

// compareUsers compares a new list of users with what is in the cache. It returns
// any users that were added, removed, or changed.
func (ms *MetricSet) compareUsers(users []*User) (added, removed, changed []*User) {
	newInCache, missingFromCache := ms.cache.DiffAndUpdateCache(convertToCacheable(users))

	if len(newInCache) > 0 && len(missingFromCache) > 0 {
		// Check for changes to users
		missingUserMap := make(map[uint32](*User))
		for _, missingUser := range missingFromCache {
			missingUserMap[missingUser.(*User).UID] = missingUser.(*User)
		}

		for _, newUser := range newInCache {
			matchingMissingUser, found := missingUserMap[newUser.(*User).UID]

			if found {
				changed = append(changed, newUser.(*User))
				delete(missingUserMap, matchingMissingUser.UID)
			} else {
				added = append(added, newUser.(*User))
			}
		}

		for _, missingUser := range missingUserMap {
			removed = append(removed, missingUser)
		}
	} else {
		// No changes to users
		for _, user := range newInCache {
			added = append(added, user.(*User))
		}

		for _, user := range missingFromCache {
			removed = append(removed, user.(*User))
		}
	}

	return
}

func convertToCacheable(users []*User) []cache.Cacheable {
	c := make([]cache.Cacheable, 0, len(users))

	for _, u := range users {
		c = append(c, u)
	}

	return c
}
