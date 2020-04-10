// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin

package app

import (
	"os"
	"os/user"
	"strconv"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
)

func getUserGroup(spec ProcessSpec) (int, int, error) {
	if spec.User.Uid == "" && spec.Group.Gid == "" {
		// use own level
		return os.Geteuid(), os.Getegid(), nil
	}

	// check if user/group exists
	usedUID := spec.User.Uid
	userGID := ""
	if u, err := user.LookupId(spec.User.Uid); err != nil {
		u, err := user.Lookup(spec.User.Name)
		if err != nil {
			return 0, 0, err
		}
		usedUID = u.Uid
		userGID = u.Gid
	} else {
		userGID = u.Gid
	}

	usedGID := spec.Group.Gid
	if spec.Group.Gid != "" || spec.Group.Name != "" {
		if _, err := user.LookupGroupId(spec.Group.Gid); err != nil {
			g, err := user.LookupGroup(spec.Group.Name)
			if err != nil {
				return 0, 0, err
			}

			usedGID = g.Gid
		}
	} else {
		// if group is not specified and user is found, use users group
		usedGID = userGID
	}

	uid, err := strconv.Atoi(usedUID)
	if err != nil {
		return 0, 0, errors.New(err, "invalid user")
	}

	gid, _ := strconv.Atoi(usedGID)
	if err != nil {
		return 0, 0, errors.New(err, "invalid group")
	}

	return uid, gid, nil
}
