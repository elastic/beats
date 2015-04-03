// +build !windows

package droppriv

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/elastic/packetbeat/common"
	"github.com/elastic/packetbeat/logp"
)

type DropPrivConfig struct {
	RunOptions RunOptions
}

type RunOptions struct {
	Uid int
	Gid int
}

func DropPrivileges(cfg common.Config) error {
	var config DropPrivConfig

	err := common.DecodeConfig(cfg, &config)
	if err != nil {
		return err
	}

	if !cfg.Meta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	if !cfg.Meta.IsDefined("runoptions", "gid") {
		return errors.New("GID must be specified for dropping privileges")
	}

	logp.Info("Switching to user: %d.%d", config.RunOptions.Uid, config.RunOptions.Gid)

	if err = syscall.Setgid(config.RunOptions.Gid); err != nil {
		return fmt.Errorf("setgid: %s", err.Error())
	}

	if err = syscall.Setuid(config.RunOptions.Uid); err != nil {
		return fmt.Errorf("setuid: %s", err.Error())
	}

	return nil
}
