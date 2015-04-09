package droppriv

import "errors"

type RunOptions struct {
	Uid *int
	Gid *int
}

func DropPrivileges(config RunOptions) error {

	if config.Uid == nil {
		// not found, no dropping privileges but no err
		return nil
	}

	return errors.New("Dropping privileges is not supported on Windows")
}
