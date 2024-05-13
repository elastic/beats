// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"os/user"
	"sync"
)

type cval struct {
	name  string
	found bool
}

type namesCache struct {
	mutex  sync.RWMutex
	users  map[string]cval
	groups map[string]cval
}

// newNamesCache will return a new namesCache, which can be used to get mappings
// of user and group IDs to names.
func newNamesCache() *namesCache {
	u := namesCache{
		users:  make(map[string]cval),
		groups: make(map[string]cval),
	}
	return &u
}

// getUserName will return the name associated with the user ID, if it exists
func (u *namesCache) getUserName(id string) (string, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	val, ok := u.users[id]
	if ok {
		return val.name, val.found
	}
	user, err := user.LookupId(id)
	cval := cval{}
	if err != nil {
		cval.name = ""
		cval.found = false
	} else {
		cval.name = user.Username
		cval.found = true
	}
	return cval.name, cval.found
}

// getGroupName will return the name associated with the group ID, if it exists
func (u *namesCache) getGroupName(id string) (string, bool) {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	val, ok := u.groups[id]
	if ok {
		return val.name, val.found
	}
	group, err := user.LookupGroupId(id)
	cval := cval{}
	if err != nil {
		cval.name = ""
		cval.found = false
	} else {
		cval.name = group.Name
		cval.found = true
	}
	return cval.name, cval.found
}
