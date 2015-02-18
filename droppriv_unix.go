// +build !windows

package main

import (
	"packetbeat/logp"
	"syscall"
)

func DropPrivileges() error {
	var err error

	if !_ConfigMeta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	if !_ConfigMeta.IsDefined("runoptions", "gid") {
		return MsgError("GID must be specified for dropping privileges")
	}

	logp.Info("Switching to user: %d.%d", _Config.RunOptions.Uid, _Config.RunOptions.Gid)

	if err = syscall.Setgid(_Config.RunOptions.Gid); err != nil {
		return MsgError("setgid: %s", err.Error())
	}

	if err = syscall.Setuid(_Config.RunOptions.Uid); err != nil {
		return MsgError("setuid: %s", err.Error())
	}

	return nil
}
