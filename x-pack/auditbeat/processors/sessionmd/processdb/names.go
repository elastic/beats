// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package processdb

import (
	"os/user"
)

// getUserName will return the name associated with the user ID, if it exists
func getUserName(id string) (string, bool) {
	user, err := user.LookupId(id)
	if err != nil {
		return "", false
	}
	return user.Username, true
}

// getGroupName will return the name associated with the group ID, if it exists
func getGroupName(id string) (string, bool) {
	group, err := user.LookupGroupId(id)
	if err != nil {
		return "", false
	}
	return group.Name, true
}
