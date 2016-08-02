package exec

import (
	"fmt"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
)

func loadCredentials(username string) (*syscall.Credential, error) {
	if username == "" {
		return nil, nil
	}

	err := fmt.Errorf("runnning commands as user ('%v') not supported on this platform.", username)
	logp.Err("Loading credential failed: %v", err)
	return nil, err
}
