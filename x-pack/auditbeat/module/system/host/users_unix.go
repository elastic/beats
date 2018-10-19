// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package host

// #include <sys/types.h>
// #include <pwd.h>
import "C"

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// User represents a user. Fields according to getpwent(3).
type User struct {
	name   string
	passwd string
	uid    uint32
	gid    uint32
	gecos  string
	dir    string
	shell  string
}

func (user User) toMapStr() common.MapStr {
	return common.MapStr{
		"name":   user.name,
		"passwd": user.passwd,
		"uid":    user.uid,
		"gid":    user.gid,
		"gecos":  user.gecos,
		"dir":    user.dir,
		"shell":  user.shell,
	}
}

// GetUsers retrieves a list of users using getpwent(3).
func GetUsers() (users []User, err error) {
	C.setpwent()
	defer C.endpwent()

	for passwd, err := C.getpwent(); passwd != nil; passwd, err = C.getpwent() {
		if err != nil {
			return nil, errors.Wrap(err, "Error getting user")
		}

		// passwd is C.struct_passwd
		users = append(users, User{
			name:   C.GoString(passwd.pw_name),
			passwd: C.GoString(passwd.pw_passwd),
			uid:    uint32(passwd.pw_uid),
			gid:    uint32(passwd.pw_gid),
			gecos:  C.GoString(passwd.pw_gecos),
			dir:    C.GoString(passwd.pw_dir),
			shell:  C.GoString(passwd.pw_shell),
		})
	}

	return users, err
}
