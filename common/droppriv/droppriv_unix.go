// +build !windows

package droppriv

import (
	"errors"
	"fmt"
	"packetbeat/config"
	"packetbeat/logp"
	"syscall"
)

func DropPrivileges() error {
	var err error

	if !config.ConfigMeta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	if !config.ConfigMeta.IsDefined("runoptions", "gid") {
		return errors.New("GID must be specified for dropping privileges")
	}

	logp.Info("Switching to user: %d.%d", config.ConfigSingleton.RunOptions.Uid, config.ConfigSingleton.RunOptions.Gid)

	if err = syscall.Setgid(config.ConfigSingleton.RunOptions.Gid); err != nil {
		return fmt.Errorf("setgid: %s", err.Error())
	}

	if err = syscall.Setuid(config.ConfigSingleton.RunOptions.Uid); err != nil {
		return fmt.Errorf("setuid: %s", err.Error())
	}

	return nil
}
