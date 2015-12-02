// +build !windows

package droppriv

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
)

type RunOptions struct {
	Uid *int
	Gid *int
}

func DropPrivileges(config RunOptions) error {
	var err error

	if config.Uid == nil {
		// not found, no dropping privileges but no err
		return nil
	}

	if config.Gid == nil {
		return errors.New("GID must be specified for dropping privileges")
	}

	logp.Info("Switching to user: %d.%d", config.Uid, config.Gid)

	if err = syscall.Setgid(*config.Gid); err != nil {
		return fmt.Errorf("setgid: %s", err.Error())
	}

	if err = syscall.Setuid(*config.Uid); err != nil {
		return fmt.Errorf("setuid: %s", err.Error())
	}

	return nil
}
