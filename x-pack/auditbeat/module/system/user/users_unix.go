// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

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
		users = append(users, &User{
			name:     C.GoString(passwd.pw_name),
			passwd:   C.GoString(passwd.pw_passwd),
			uid:      uint32(passwd.pw_uid),
			gid:      uint32(passwd.pw_gid),
			userInfo: C.GoString(passwd.pw_gecos),
			dir:      C.GoString(passwd.pw_dir),
			shell:    C.GoString(passwd.pw_shell),
		})
	}

	return users, nil
}
