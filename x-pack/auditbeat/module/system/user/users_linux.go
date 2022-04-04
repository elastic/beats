// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && cgo
// +build linux,cgo

package user

// #include <errno.h>
// #include <sys/types.h>
// #include <pwd.h>
// #include <shadow.h>
//
// void clearErrno() {
//      errno = 0;
// }
import "C"

import (
	"crypto/sha512"
	"fmt"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
)

var epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

// GetUsers retrieves a list of users using information from
// /etc/passwd, /etc/group, and - if configured - /etc/shadow.
func GetUsers(readPasswords bool) ([]*User, error) {
	var errs multierror.Errors

	// We are using a number of thread sensitive C functions in
	// this file, most importantly setpwent/getpwent/endpwent and
	// setspent/getspent/endspent. And we set errno (which is thread-local).
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	users, err := readPasswdFile(readPasswords)
	if err != nil {
		errs = append(errs, err)
	}

	if len(users) > 0 {
		err = enrichWithGroups(users)
		if err != nil {
			errs = append(errs, err)
		}

		if readPasswords {
			err = enrichWithShadow(users)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	return users, errs.Err()
}

func readPasswdFile(readPasswords bool) ([]*User, error) {
	var users []*User

	C.setpwent()
	defer C.endpwent()

	for {
		// Setting errno to 0 before calling getpwent().
		// See return value section of getpwent(3).
		C.clearErrno()

		passwd, err := C.getpwent()

		if passwd == nil {
			// getpwent() can return ENOENT even when there is no error,
			// see https://github.com/systemd/systemd/issues/9585.
			if err != nil && err != syscall.ENOENT {
				return users, fmt.Errorf("error getting user: %w", err)
			}

			// No more entries
			break
		}

		// passwd is C.struct_passwd
		user := &User{
			Name:     C.GoString(passwd.pw_name),
			UID:      strconv.Itoa(int(passwd.pw_uid)),
			GID:      strconv.Itoa(int(passwd.pw_gid)),
			UserInfo: C.GoString(passwd.pw_gecos),
			Dir:      C.GoString(passwd.pw_dir),
			Shell:    C.GoString(passwd.pw_shell),
		}

		if readPasswords {
			switch C.GoString(passwd.pw_passwd) {
			case "x":
				user.PasswordType = shadowPassword
			case "*":
				user.PasswordType = passwordDisabled
			case "":
				user.PasswordType = noPassword
			default:
				user.PasswordType = cryptPassword
				user.PasswordHashHash = multiRoundHash(C.GoString(passwd.pw_passwd))
			}
		} else {
			user.PasswordType = detectionDisabled
		}

		users = append(users, user)
	}

	return users, nil
}

func enrichWithGroups(users []*User) error {
	gidCache := make(map[string]*user.Group, len(users))

	for _, u := range users {
		goUser := user.User{
			Uid:      u.UID,
			Gid:      u.GID,
			Username: u.Name,
		}

		groupIds, err := goUser.GroupIds()
		if err != nil {
			return fmt.Errorf("error getting group IDs for user %v (UID: %v): %w", u.Name, u.UID, err)
		}

		for _, gid := range groupIds {
			group, found := gidCache[gid]
			if !found {
				group, err = user.LookupGroupId(gid)
				if err != nil {
					return fmt.Errorf("error looking up group ID %v for user %v (UID: %v): %w", gid, u.Name, u.UID, err)
				}
				gidCache[gid] = group
			}

			u.Groups = append(u.Groups, group)
		}
	}

	return nil
}

func enrichWithShadow(users []*User) error {
	shadowEntries, err := readShadowFile()
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.PasswordType == shadowPassword {
			shadow, found := shadowEntries[user.Name]
			if found {
				user.PasswordChanged = shadow.LastChanged

				if shadow.Password == "" {
					user.PasswordType = noPassword
				} else if strings.HasPrefix(shadow.Password, "!") || strings.HasPrefix(shadow.Password, "*") {
					user.PasswordType = passwordDisabled
				} else {
					user.PasswordHashHash = multiRoundHash(shadow.Password)
				}
			}
		}
	}

	return nil
}

// shadowFileEntry represents an entry in /etc/shadow. See getspnam(3) for details.
type shadowFileEntry struct {
	LastChanged time.Time
	Password    string
}

// readShadowFile reads /etc/shadow and returns a map of the entries keyed to user's names.
func readShadowFile() (map[string]shadowFileEntry, error) {
	C.setspent()
	defer C.endspent()

	shadowEntries := make(map[string]shadowFileEntry)

	for {
		// While getspnam(3) does not explicitly call out the need for setting errno to 0
		// as getpwent(3) does, at least glibc uses the same code for both, and so it
		// probably makes sense to do the same for both.
		C.clearErrno()

		spwd, err := C.getspent()

		if spwd == nil {
			if err != nil {
				return shadowEntries, fmt.Errorf("error while reading shadow file: %w", err)
			}

			// No more entries
			break
		}

		shadow := shadowFileEntry{
			// sp_lstchg is in days since Jan 1, 1970.
			LastChanged: epoch.AddDate(0, 0, int(spwd.sp_lstchg)),

			// The password hash is never output to Elasticsearch or any other output,
			// but a hash of the hash is persisted to disk in the beat.db file.
			Password: C.GoString(spwd.sp_pwdp),
		}

		shadowEntries[C.GoString(spwd.sp_namp)] = shadow
	}

	return shadowEntries, nil
}

// multiRoundHash performs 10 rounds of SHA-512 hashing.
func multiRoundHash(s string) []byte {
	hash := sha512.Sum512([]byte(s))
	for i := 0; i < 9; i++ {
		hash = sha512.Sum512(hash[:])
	}
	return hash[:]
}
