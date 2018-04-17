// +build !linux

package process

import "os"

// GetSelfPid returns the PID for this process
func GetSelfPid() (int, error) {
	return os.Getpid(), nil
}
