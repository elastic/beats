package exec

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
)

func newProcAttributes(config execRunnerConfig) (*syscall.SysProcAttr, error) {
	if user := config.User; user != "" {
		err := fmt.Errorf(
			"runnning commands as user ('%v') not supported on this platform.",
			user)
		logp.Err("Reading exec config failed: %v", err)
		return nil, err
	}

	if config.Chroot != "" {
		err := errors.New("Chroot not supported on this platform")
		logp.Err("Reading exec config failed: %v", err)
		return nil, err
	}

	return &syscall.SysProcAttr{
		HideWindow: true,
	}, nil
}
