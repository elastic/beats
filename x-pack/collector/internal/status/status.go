// Package status provides internal status reporting. Subsystems update their
// status via status.Reporter.
package status

//go:generate godocdown -plain=false -output Readme.md

// Reporter is used by subsystems to report their current state. If a system
// itself has multiple subsystems, it might want to create a separate reporter
// per subsystem in order to merge status states into one common state.
type Reporter interface {
	UpdateStatus(s Status, reason string)
}

// TODO: create special reporter type merges its state from child reporters. It
// must be possible to add/remove children dynamically.

// Status describes the current status of the beat.
type Status int

// State stores a status state. The state can be updated via Update.
type State struct {
	Status  Status
	Message string
}

const (
	// Unknown is initial status when none has been reported.
	Unknown Status = iota
	// Starting is status describing application is starting.
	Starting
	// Configuring is status describing application is configuring.
	Configuring
	// Running is status describing application is running.
	Running
	// Degraded is status describing application is degraded.
	Degraded
	// Failed is status describing application is failed. This status should
	// only be used in the case the beat should stop running as the failure
	// cannot be recovered.
	Failed
	// Stopping is status describing application is stopping.
	Stopping
)

// Update modifies the stored status or status message. It returns true if the
// new status differs from the old status.
func (state *State) Update(s Status, reason string) bool {
	if state.Status != s || state.Message != reason {
		state.Status = s
		state.Message = reason
		return true
	}
	return false
}
