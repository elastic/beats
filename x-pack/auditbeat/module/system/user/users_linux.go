// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package user

// #include <sys/types.h>
// #include <pwd.h>
// #include <shadow.h>
import "C"

import (
	"github.com/pkg/errors"
	"time"
)

var (
	epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
)

// GetUsers retrieves a list of users using getpwent(3).
func GetUsers() (users []*User, err error) {
	shadowEntries, err := readShadowFile()
	if err != nil {
		return nil, errors.Wrap(err, "error getting password change times")
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

		shadow, found := shadowEntries[user.Name]
		if found {
			user.PasswordChanged = shadow.LastChanged
			user.PasswordHashExcerpt = shadow.PasswordHashExcerpt
		}

		users = append(users, user)
	}

	return users, nil
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
