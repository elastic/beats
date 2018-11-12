// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package user

// #include <sys/types.h>
// #include <pwd.h>
// #include <grp.h>
// #include <shadow.h>
import "C"

import (
	"github.com/pkg/errors"
	"time"
	"unsafe"
)

var (
	epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
)

// GetUsers retrieves a list of users using getpwent(3).
func GetUsers() (users []*User, err error) {
	gidToGroup, userToGroup, err := readGroupFile()
	if err != nil {
		return nil, err
	}

	shadowEntries, err := readShadowFile()
	if err != nil {
		return nil, err
	}

	C.setpwent()
	defer C.endpwent()

	for passwd, err := C.getpwent(); passwd != nil; passwd, err = C.getpwent() {
		if err != nil {
			return nil, errors.Wrap(err, "error getting user")
		}

		// passwd is C.struct_passwd
		user := &User{
			Name:     C.GoString(passwd.pw_name),
			UID:      uint32(passwd.pw_uid),
			GID:      uint32(passwd.pw_gid),
			UserInfo: C.GoString(passwd.pw_gecos),
			Dir:      C.GoString(passwd.pw_dir),
			Shell:    C.GoString(passwd.pw_shell),
		}

		switch C.GoString(passwd.pw_passwd) {
		case "x":
			user.PasswordType = "shadow_password"
		case "*":
			user.PasswordType = "login_disabled"
		case "":
			user.PasswordType = "no_password"
		default:
			user.PasswordType = "<redacted>"
		}

		primaryGroup, found := gidToGroup[user.GID]
		if found {
			user.Groups = append(user.Groups, primaryGroup)
		}

		secondaryGroups, found := userToGroup[user.Name]
		if found {
			user.Groups = append(user.Groups, secondaryGroups...)
		}

		shadow, found := shadowEntries[user.Name]
		if found {
			user.PasswordChanged = shadow.LastChanged
			user.PasswordHashExcerpt = shadow.PasswordHashExcerpt
		}

		users = append(users, user)
	}

	return users, nil
}

// readGroupFile reads /etc/group and returns two maps:
// The first maps group IDs to groups.
// The second maps group members (user names) to groups.
// See getgrent(3) for details of the structs.
func readGroupFile() (map[uint32]Group, map[string][]Group, error) {
	C.setgrent()
	defer C.endgrent()

	groupIDMap := make(map[uint32]Group)
	groupMemberMap := make(map[string][]Group)
	for cgroup, err := C.getgrent(); cgroup != nil; cgroup, err = C.getgrent() {
		if err != nil {
			return nil, nil, errors.Wrap(err, "error while reading group file")
		}

		groupName := C.GoString(cgroup.gr_name)
		gid := uint32(cgroup.gr_gid)

		group := Group{
			Name: groupName,
			GID:  gid,
		}

		groupIDMap[gid] = group

		/*
			group.gr_mem is a NULL-terminated array of pointers to user names (char **)
			which makes some pointer arithmetic necessary to read it.
		*/
		for i := 0; ; i++ {
			offset := (unsafe.Sizeof(unsafe.Pointer(*cgroup.gr_mem)) * uintptr(i))
			member := *(**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cgroup.gr_mem)) + offset))

			if member == nil {
				break
			}

			groupMember := C.GoString(member)
			groupMemberMap[groupMember] = append(groupMemberMap[groupMember], group)
		}
	}

	return groupIDMap, groupMemberMap, nil
}

// shadowFileEntry represents an entry in /etc/shadow. See getspnam(3) for details.
type shadowFileEntry struct {
	LastChanged         time.Time
	PasswordHashExcerpt string
}

// readShadowFile reads /etc/shadow and returns a map of the entries keyed to user's names.
func readShadowFile() (map[string]shadowFileEntry, error) {
	C.setspent()
	defer C.endspent()

	shadowEntries := make(map[string]shadowFileEntry)
	for spwd, err := C.getspent(); spwd != nil; spwd, err = C.getspent() {
		if err != nil {
			return nil, errors.Wrap(err, "error while reading shadow file")
		}

		shadow := shadowFileEntry{
			// sp_lstchg is in days since Jan 1, 1970.
			LastChanged: epoch.AddDate(0, 0, int(spwd.sp_lstchg)),
		}

		/*
			Make sure the full password hash is long (it should be) and then collect only 10 characters.
			This should be long enough to make collisions (where a new password hash shares these
			10 characters with an old password hash) extremely unlikely. In addition, we also check
			the day the password was last changed to detect a password change.

			The hash excerpt is not included in any event output, but it is persisted to the
			beat.db file on disk.
		*/
		passwordHash := C.GoString(spwd.sp_pwdp)
		if len(passwordHash) > 24 {
			shadow.PasswordHashExcerpt = passwordHash[len(passwordHash)-10:]
		}

		shadowEntries[C.GoString(spwd.sp_namp)] = shadow
	}

	return shadowEntries, nil
}
