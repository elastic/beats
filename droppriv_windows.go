package main

import (
	"errors"
	"packetbeat/config"
)

func DropPrivileges() error {

	if !config.ConfigMeta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	return errors.New("Dropping privileges is not supported on Windows")
}
