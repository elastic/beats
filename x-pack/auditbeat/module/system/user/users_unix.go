// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows,cgo

package user

// #include <sys/types.h>
// #include <pwd.h>
import "C"

import (
	"github.com/pkg/errors"
)

// GetUsers retrieves a list of users using getpwent(3).
func GetUsers() (users []*User, err error) {
	C.setpwent()
	defer C.endpwent()

	for passwd, err := C.getpwent(); passwd != nil; passwd, err = C.getpwent() {
		if err != nil {
			return nil, errors.Wrap(err, "error getting user")
		}

		// passwd is C.struct_passwd
		user := &User{
			Name:     C.GoString(passwd.pw_name),
			Passwd:   C.GoString(passwd.pw_passwd),
			UID:      uint32(passwd.pw_uid),
			GID:      uint32(passwd.pw_gid),
			UserInfo: C.GoString(passwd.pw_gecos),
			Dir:      C.GoString(passwd.pw_dir),
			Shell:    C.GoString(passwd.pw_shell),
		}

		switch C.GoString(passwd.pw_passwd) {
		case "x":
			user.Passwd = "shadow_password"
		case "*":
			user.Passwd = "login_disabled"
		case "":
			user.Passwd = "no_password"
		default:
			user.Passwd = "<redacted>"
		}

		users = append(users, user)
	}

	return users, nil
}
