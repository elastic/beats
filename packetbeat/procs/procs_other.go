// +build !linux,!windows

package procs

// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
// that owns them.
func (proc *ProcessesWatcher) GetLocalPortToPIDMapping() (ports map[uint16]int, err error) {
	return nil, nil
}
