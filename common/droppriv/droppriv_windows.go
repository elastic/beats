package droppriv

import (
	"errors"

	"github.com/elastic/packetbeat/common"
)

func DropPrivileges(cfg common.Config) error {

	if !cfg.Meta.IsDefined("runoptions", "uid") {
		// not found, no dropping privileges but no err
		return nil
	}

	return errors.New("Dropping privileges is not supported on Windows")
}
