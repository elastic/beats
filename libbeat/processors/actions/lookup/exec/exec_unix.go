// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package exec

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

func newProcAttributes(config execRunnerConfig) (*syscall.SysProcAttr, error) {
	creds, err := loadCredentials(config.User)
	if err != nil {
		return nil, err
	}

	return &syscall.SysProcAttr{
		Chroot:     config.Chroot,
		Credential: creds,
	}, nil
}

func loadCredentials(username string) (*syscall.Credential, error) {
	if username == "" {
		return nil, nil
	}

	u, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uid %v", uid)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gid %v", uid)
	}

	return &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}, nil
}
