// Package cgroup reads metrics and other tunable parameters associated with
// control groups, a Linux kernel feature for grouping tasks to track and limit
// resource usage.
//
// Terminology
//
// A cgroup is a collection of processes that are bound to a set of limits.
//
// A subsystem is a kernel component the modifies the behavior of processes
// in a cgroup.
package cgroup
