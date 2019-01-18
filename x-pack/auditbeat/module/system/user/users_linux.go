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
	"crypto/sha512"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
)

// GetUsers retrieves a list of users using information from
// /etc/passwd, /etc/group, and - if configured - /etc/shadow.
func GetUsers(readPasswords bool) ([]*User, error) {
	users, err := readPasswdFile(readPasswords)
	if err != nil {
		return nil, err
	}

	err = enrichWithGroups(users)
	if err != nil {
		return nil, err
	}

	if readPasswords {
		err = enrichWithShadow(users)
		if err != nil {
			return nil, err
		}
	}

	return users, nil
}

func readPasswdFile(readPasswords bool) ([]*User, error) {
	var users []*User

	C.setpwent()
	defer C.endpwent()

	for passwd, err := C.getpwent(); passwd != nil; passwd, err = C.getpwent() {
		if err != nil {
			return nil, errors.Wrap(err, "error getting user")
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
			return errors.Wrapf(err, "error getting group IDs for user %v (UID: %v)", u.Name, u.UID)
		}

		for _, gid := range groupIds {
			group, found := gidCache[gid]
			if !found {
				group, err = user.LookupGroupId(gid)
				if err != nil {
					return errors.Wrapf(err, "error looking up group ID %v for user %v (UID: %v)", gid, u.Name, u.UID)
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
	for spwd, err := C.getspent(); spwd != nil; spwd, err = C.getspent() {
		if err != nil {
			return nil, errors.Wrap(err, "error while reading shadow file")
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
