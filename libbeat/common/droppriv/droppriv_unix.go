// +build !windows

package droppriv

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
)

type RunOptions struct {
	UID *int
	GID *int
}

func DropPrivileges(config RunOptions) error {
	var err error

	if config.UID == nil {
		// not found, no dropping privileges but no err
		return nil
	}

	if config.GID == nil {
		return errors.New("GID must be specified for dropping privileges")
	}

	logp.Info("Switching to user: %d.%d", config.UID, config.GID)

	if err = syscall.Setgid(*config.GID); err != nil {
		return fmt.Errorf("setgid: %s", err.Error())
	}

	if err = syscall.Setuid(*config.UID); err != nil {
		return fmt.Errorf("setuid: %s", err.Error())
	}

	return nil
}
