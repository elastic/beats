// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package user

import (
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/OneOfOne/xxhash"

	"github.com/elastic/beats/auditbeat/datastore"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/auditbeat/cache"
)

const (
	moduleName    = "system"
	metricsetName = "user"
	bucketName    = "user.v1"

	eventTypeSnapshot = "snapshot"
	eventTypeChange   = "change"

	eventTypeDetailUserExists  = "user_exists"
	eventTypeDetailUserAdded   = "user_added"
	eventTypeDetailUserRemoved = "user_removed"
	eventTypeDetailUserChanged = "user_changed"
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
	name     string
	passwd   string
	uid      uint32
	gid      uint32
	userInfo string
	dir      string
	shell    string
}

// Hash creates a hash for User.
func (user User) Hash() uint64 {
	h := xxhash.New64()
	// Use everything except userInfo
	h.WriteString(user.name)
	h.WriteString(user.passwd)
	h.WriteString(strconv.Itoa(int(user.uid)))
	h.WriteString(strconv.Itoa(int(user.gid)))
	h.WriteString(user.dir)
	h.WriteString(user.shell)
	return h.Sum64()
}

func (user User) toMapStr() common.MapStr {
	evt := common.MapStr{
		"name":   user.name,
		"passwd": user.passwd,
		"uid":    user.uid,
		"gid":    user.gid,
		"dir":    user.dir,
		"shell":  user.shell,
	}

	if user.userInfo != "" {
		evt.Put("user_information", user.userInfo)
	}

	return evt
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The %v/%v dataset is experimental", moduleName, metricsetName)

	bucket, err := datastore.OpenBucket(bucketName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open persistent datastore")
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger(moduleName),
		cache:         cache.New(),
		bucket:        bucket,
	}, nil
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
					"event.type":        eventTypeChange,
					"event.type_detail": eventTypeDetailUserAdded,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		for _, user := range removed {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":        eventTypeChange,
					"event.type_detail": eventTypeDetailUserRemoved,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		for _, user := range changed {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":        eventTypeChange,
					"event.type_detail": eventTypeDetailUserChanged,
				},
				MetricSetFields: user.toMapStr(),
			})
		}
	} else {
		// Report all existing users
		for _, user := range users {
			report.Event(mb.Event{
				RootFields: common.MapStr{
					"event.type":        eventTypeSnapshot,
					"event.type_detail": eventTypeDetailUserExists,
				},
				MetricSetFields: user.toMapStr(),
			})
		}

		if ms.cache != nil {
			// This will initialize the cache with the current processes
			ms.cache.DiffAndUpdateCache(convertToCacheable(users))
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
			missingUserMap[missingUser.(*User).uid] = missingUser.(*User)
		}

		for _, newUser := range newInCache {
			matchingMissingUser, found := missingUserMap[newUser.(*User).uid]

			if found {
				changed = append(changed, newUser.(*User))
				delete(missingUserMap, matchingMissingUser.uid)
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
