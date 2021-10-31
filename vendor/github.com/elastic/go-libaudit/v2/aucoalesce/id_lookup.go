// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package aucoalesce

import (
	"math"
	"os/user"
	"strings"
	"sync"
	"time"
)

const cacheTimeout = time.Minute

var (
	userLookup  = NewUserCache(cacheTimeout)
	groupLookup = NewGroupCache(cacheTimeout)

	// noExpiration = time.Unix(math.MaxInt64, 0)
	// The above breaks time.Before and time.After due to overflows.
	// See https://stackoverflow.com/questions/25065055/what-is-the-maximum-time-time-in-go
	//
	// Safe alternative:
	noExpiration = time.Unix(0, 0).Add(math.MaxInt64 - 1)
)

type stringItem struct {
	timeout time.Time
	value   string
}

func (i *stringItem) isExpired() bool {
	return time.Now().After(i.timeout)
}

// EntityCache is a cache of IDs and usernames.
type EntityCache struct {
	byID, byName stringCache
}

// NewUserCache returns a new EntityCache to resolve users. EntityCache is thread-safe.
func NewUserCache(expiration time.Duration) *EntityCache {
	return &EntityCache{
		byID: stringCache{
			expiration: expiration,
			data: map[string]stringItem{
				"0": {timeout: noExpiration, value: "root"},
			},
			lookupFn: func(s string) string {
				user, err := user.LookupId(s)
				if err != nil {
					return ""
				}
				return user.Username
			},
		},
		byName: stringCache{
			expiration: expiration,
			data: map[string]stringItem{
				"root": {timeout: noExpiration, value: "0"},
			},
			lookupFn: func(s string) string {
				user, err := user.Lookup(s)
				if err != nil {
					return ""
				}
				return user.Uid
			},
		},
	}
}

// LookupID looks up an UID/GID and returns the user/group name associated with it. If
// no name could be found an empty string is returned. The value will be
// cached for a minute.
func (c *EntityCache) LookupID(uid string) string {
	return c.byID.lookup(uid)
}

// LookupName looks up an user/group name and returns the ID associated with it. If
// no ID could be found an empty string is returned. The value will be
// cached for a minute. This requires cgo on Linux.
func (c *EntityCache) LookupName(name string) string {
	return c.byName.lookup(name)
}

// NewGroupCache returns a new EntityCache to resolve groups. EntityCache is thread-safe.
func NewGroupCache(expiration time.Duration) *EntityCache {
	return &EntityCache{
		byID: stringCache{
			expiration: expiration,
			data: map[string]stringItem{
				"0": {timeout: noExpiration, value: "root"},
			},
			lookupFn: func(s string) string {
				grp, err := user.LookupGroupId(s)
				if err != nil {
					return ""
				}
				return grp.Name
			},
		},
		byName: stringCache{
			expiration: expiration,
			data: map[string]stringItem{
				"root": {timeout: noExpiration, value: "0"},
			},
			lookupFn: func(s string) string {
				grp, err := user.LookupGroup(s)
				if err != nil {
					return ""
				}
				return grp.Gid
			},
		},
	}
}

// ResolveIDs translates all uid and gid values to their associated names.
// Prior to Go 1.9 this requires cgo on Linux. UID and GID values are cached
// for 60 seconds from the time they are read.
func ResolveIDs(event *Event) {
	ResolveIDsFromCaches(event, userLookup, groupLookup)
}

// ResolveIDsFromCaches translates all uid and gid values to their associated
// names using the provided caches. Prior to Go 1.9 this requires cgo on Linux.
func ResolveIDsFromCaches(event *Event, users, groups *EntityCache) {
	// Actor
	if v := users.LookupID(event.Summary.Actor.Primary); v != "" {
		event.Summary.Actor.Primary = v
	}
	if v := users.LookupID(event.Summary.Actor.Secondary); v != "" {
		event.Summary.Actor.Secondary = v
	}

	// User
	names := map[string]string{}
	for key, id := range event.User.IDs {
		if strings.HasSuffix(key, "uid") {
			if v := users.LookupID(id); v != "" {
				names[key] = v
			}
		} else if strings.HasSuffix(key, "gid") {
			if v := groups.LookupID(id); v != "" {
				names[key] = v
			}
		}
	}
	if len(names) > 0 {
		event.User.Names = names
	}

	// File owner/group
	if event.File != nil {
		if event.File.UID != "" {
			event.File.Owner = users.LookupID(event.File.UID)
		}
		if event.File.GID != "" {
			event.File.Group = groups.LookupID(event.File.GID)
		}
	}

	// ECS User and groups
	event.ECS.User.lookup(users)
	event.ECS.Group.lookup(groups)
}

// HardcodeUsers is useful for injecting values for testing.
func HardcodeUsers(users ...user.User) {
	for _, usr := range users {
		userLookup.byID.hardcode(usr.Uid, usr.Username)
		userLookup.byName.hardcode(usr.Username, usr.Uid)
	}
}

// HardcodeGroups is useful for injecting values for testing.
func HardcodeGroups(groups ...user.Group) {
	for _, grp := range groups {
		groupLookup.byID.hardcode(grp.Gid, grp.Name)
		groupLookup.byName.hardcode(grp.Name, grp.Gid)
	}
}

type stringCache struct {
	mutex      sync.Mutex
	expiration time.Duration
	data       map[string]stringItem
	lookupFn   func(string) string
}

func (c *stringCache) lookup(key string) string {
	if key == "" || key == "unset" {
		return ""
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, found := c.data[key]; found && !item.isExpired() {
		return item.value
	}

	// Cache the result (even on error).
	resolved := c.lookupFn(key)
	c.data[key] = stringItem{timeout: time.Now().Add(c.expiration), value: resolved}
	return resolved
}

func (c *stringCache) hardcode(key, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = stringItem{
		timeout: noExpiration,
		value:   value,
	}
}
