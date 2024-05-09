package status

// Status describes the current status of the beat.
type Status int

//go:generate stringer -type=Status
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
	// Stopped is status describing application is stopped.
	Stopped
)

// StatusReporter provides a method to update current status of a unit.
type StatusReporter interface {
	// UpdateStatus updates the status of the unit.
	UpdateStatus(status Status, msg string)
}

// WithStatusReporter provides a method to set a status reporter
type WithStatusReporter interface {
	// SetStatusReporter sets the status reporter
	SetStatusReporter(reporter StatusReporter)
}
