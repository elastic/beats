// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package socket

import (
	"reflect"
	"time"
)

// Config defines this metricset's configuration options.
type Config struct {
	// TraceFSPath holds a custom path to tracefs (or debugfs' tracing dir).
	// If unset (default), the first available path is used:
	// 		- /sys/kernel/tracing (tracefs, 4.x+)
	//		- /sys/kernel/debug/tracing (debugfs, 2.6+)
	TraceFSPath *string `config:"socket.tracefs_path"`

	// PerfQueueSize defines how many tracing events can be queued.
	PerfQueueSize int `config:"socket.perf_queue_size,min=1"`

	// LostQueueSize specifies how many lost-event notifications can be queued.
	LostQueueSize int `config:"socket.lost_queue_size,min=1"`

	// ErrQueueSize defines the size of the error queue. A single error is fatal.
	ErrQueueSize int `config:"socket.err_queue_size,min=1"`

	// RingSizeExp configures the exponent size for the per-cpu ring buffer used
	// by the kernel to pass tracing events.
	// The actual size is 2**exponent memory pages, per CPU.
	RingSizeExp int `config:"socket.ring_size_exponent,min=1"`

	// FlowInactiveTimeout determines how long a flow has to be inactive to be
	// considered closed.
	FlowInactiveTimeout time.Duration `config:"socket.flow_inactive_timeout"`

	// SocketInactiveTimeout determines how long a socket has to be inactive to be
	// considered terminated or closed.
	SocketInactiveTimeout time.Duration `config:"socket.socket_inactive_timeout"`

	// FlowTerminationTimeout determines how long to wait after a flow has been
	// closed for out of order packets. With TCP, some packets can be received
	// shortly after a socket is closed. If set too low, additional flows will
	// be generated for those packets.
	FlowTerminationTimeout time.Duration `config:"socket.flow_termination_timeout"`

	// ClockMaxDrift defines the maximum difference between the kernel internal
	// clock (boot time) and our reference time used to timestamp events. Once
	// this max drift is exceeded, the reference time is adjusted.
	// This clock has been observed to drift from usermode clocks up to 0.15ms/s
	ClockMaxDrift time.Duration `config:"socket.clock_max_drift,positive"`

	// ClockSyncPeriod determines how often clock synchronization events are
	// generated to measure the drift between the kernel clock and our reference
	ClockSyncPeriod time.Duration `config:"socket.clock_sync_period,positive"`

	// GuessTimeout is the maximum time an individual guess is allowed to run.
	GuessTimeout time.Duration `config:"socket.guess_timeout,positive"`

	// DevelopmentMode is an undocumented flag to ignore SSH traffic so that the
	// dataset can be run with debug output without creating a feedback loop.
	DevelopmentMode bool `config:"socket.development_mode"`

	// EnableIPv6 allows to control IPv6 support. When unset (default) IPv6
	// will be automatically detected on runtime.
	EnableIPv6 *bool `config:"socket.enable_ipv6"`
}

// Validate validates the socket metricset config.
func (c *Config) Validate() error {
	return nil
}

// Equals compares two Config objects
func (c *Config) Equals(other Config) bool {
	// reflect.DeepEquals() doesn't compare pointed-to values, so strip
	// all pointers and then compare them manually.
	simpler := [2]Config{*c, other}
	for idx := range simpler {
		simpler[idx].EnableIPv6 = nil
		simpler[idx].TraceFSPath = nil
	}
	return reflect.DeepEqual(simpler[0], simpler[1]) &&
		(c.EnableIPv6 == nil) == (other.EnableIPv6 == nil) &&
		(c.EnableIPv6 == nil || *c.EnableIPv6 == *other.EnableIPv6) &&
		(c.TraceFSPath == nil) == (other.TraceFSPath == nil) &&
		(c.TraceFSPath == nil || *c.TraceFSPath == *other.TraceFSPath)
}

var defaultConfig = Config{
	PerfQueueSize:          4096,
	LostQueueSize:          128,
	ErrQueueSize:           1,
	RingSizeExp:            7,
	FlowInactiveTimeout:    30 * time.Second,
	SocketInactiveTimeout:  60 * time.Second,
	FlowTerminationTimeout: 5 * time.Second,
	ClockMaxDrift:          100 * time.Millisecond,
	ClockSyncPeriod:        10 * time.Second,
	GuessTimeout:           15 * time.Second,
}
