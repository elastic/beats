package droppriv

import (
	"errors"

	"github.com/BurntSushi/toml"
)

type RunOptions struct {
	Uid int
	Gid int
}

func DropPrivileges(config RunOptions, configMeta toml.MetaData) error {

	if !configMeta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	return errors.New("Dropping privileges is not supported on Windows")
}
