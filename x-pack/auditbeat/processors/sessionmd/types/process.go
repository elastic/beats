// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package types

import (
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// These fields contain information about a process.
// These fields can help you correlate metrics information with a process id/name from a log message.  The `process.pid` often stays in the metric itself and is copied to the global field for correlation.
type Process struct {
	// Unique identifier for the process.
	// The implementation of this is specified by the data source, but some examples of what could be used here are a process-generated UUID, Sysmon Process GUIDs, or a hash of some uniquely identifying components of a process.
	// Constructing a globally unique identifier is a common practice to mitigate PID reuse as well as to identify a specific process over time, across multiple monitored hosts.
	EntityID string `json:"entity_id,omitempty"`

	// Absolute path to the process executable.
	Executable string `json:"executable,omitempty"`

	// Process name.
	// Sometimes called program name or similar.
	Name string `json:"name,omitempty"`

	// The time the process started.
	Start *time.Time `json:"start,omitempty"`

	// The time the process ended.
	End *time.Time `json:"end,omitempty"`

	// The exit code of the process, if this is a termination event.
	// The field should be absent if there is no exit code for the event (e.g. process start).
	ExitCode int32 `json:"exit_code,omitempty"`

	// Whether the process is connected to an interactive shell.
	// Process interactivity is inferred from the processes file descriptors. If the character device for the controlling tty is the same as stdin and stderr for the process, the process is considered interactive.
	// Note: A non-interactive process can belong to an interactive session and is simply one that does not have open file descriptors reading the controlling TTY on FD 0 (stdin) or writing to the controlling TTY on FD 2 (stderr). A backgrounded process is still considered interactive if stdin and stderr are connected to the controlling TTY.
	Interactive *bool `json:"interactive,omitempty"`

	// The working directory of the process.
	WorkingDirectory string `json:"working_directory,omitempty"`

	// The effective user (euid).
	User struct {
		// Unique identifier of the user.
		ID string `json:"id,omitempty"`

		// Short name or login of the user.
		Name string `json:"name,omitempty"`
	} `json:"user,omitempty"`

	// The effective group (egid).
	Group struct {
		// Unique identifier for the group on the system/platform.
		ID string `json:"id,omitempty"`

		// Name of the group.
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`

	// Process id.
	PID uint32 `json:"pid,omitempty"`

	Vpid uint32 `json:"vpid,omitempty"`

	// Array of process arguments, starting with the absolute path to the executable.
	// May be filtered to protect sensitive information.
	Args []string `json:"args,omitempty"`

	// An array of previous executions for the process, including the initial fork. Only executable and args are set.
	Previous []struct {
		// Absolute path to the process executable.
		Executable string `json:"executable,omitempty"`

		// Array of process arguments, starting with the absolute path to the executable.
		// May be filtered to protect sensitive information.
		Args []string `json:"args,omitempty"`
	} `json:"previous,omitempty"`

	Thread struct {
		Capabilities struct {
			Permitted []string `json:"permitted,omitempty"`

			Effective []string `json:"effective,omitempty"`
		} `json:"capabilities,omitempty"`
	} `json:"thread,omitempty"`

	// Information about the controlling TTY device.
	// If set, the process belongs to an interactive session.
	TTY struct {
		CharDevice struct {
			Major uint16 `json:"major,omitempty"`
			Minor uint16 `json:"minor,omitempty"`
		} `json:"char_device,omitempty"`
	} `json:"tty,omitempty"`

	// Information about the parent process.
	Parent struct {
		// Unique identifier for the process.
		// The implementation of this is specified by the data source, but some examples of what could be used here are a process-generated UUID, Sysmon Process GUIDs, or a hash of some uniquely identifying components of a process.
		// Constructing a globally unique identifier is a common practice to mitigate PID reuse as well as to identify a specific process over time, across multiple monitored hosts.
		EntityID string `json:"entity_id,omitempty"`

		// Absolute path to the process executable.
		Executable string `json:"executable,omitempty"`

		// Whether the process is connected to an interactive shell.
		// Process interactivity is inferred from the processes file descriptors. If the character device for the controlling tty is the same as stdin and stderr for the process, the process is considered interactive.
		// Note: A non-interactive process can belong to an interactive session and is simply one that does not have open file descriptors reading the controlling TTY on FD 0 (stdin) or writing to the controlling TTY on FD 2 (stderr). A backgrounded process is still considered interactive if stdin and stderr are connected to the controlling TTY.
		Interactive *bool `json:"interactive,omitempty"`

		// Process name.
		// Sometimes called program name or similar.
		Name string `json:"name,omitempty"`

		// The time the process started.
		Start *time.Time `json:"start,omitempty"`

		// The working directory of the process.
		WorkingDirectory string `json:"working_directory,omitempty"`

		// The effective user (euid).
		User struct {
			// Unique identifier of the user.
			ID string `json:"id,omitempty"`

			// Short name or login of the user.
			Name string `json:"name,omitempty"`
		} `json:"user,omitempty"`

		// The effective group (egid).
		Group struct {
			// Unique identifier for the group on the system/platform.
			ID string `json:"id,omitempty"`

			// Name of the group.
			Name string `json:"name,omitempty"`
		} `json:"group,omitempty"`

		// Process id.
		PID uint32 `json:"pid,omitempty"`

		// Array of process arguments, starting with the absolute path to the executable.
		// May be filtered to protect sensitive information.
		Args []string `json:"args,omitempty"`

		Thread struct {
			Capabilities struct {
				Permitted []string `json:"permitted,omitempty"`

				Effective []string `json:"effective,omitempty"`
			} `json:"capabilities,omitempty"`
		} `json:"thread,omitempty"`
	} `json:"parent,omitempty"`

	// Information about the process group leader. In some cases this may be the same as the top level process.
	GroupLeader struct {
		// Unique identifier for the process.
		// The implementation of this is specified by the data source, but some examples of what could be used here are a process-generated UUID, Sysmon Process GUIDs, or a hash of some uniquely identifying components of a process.
		// Constructing a globally unique identifier is a common practice to mitigate PID reuse as well as to identify a specific process over time, across multiple monitored hosts.
		EntityID string `json:"entity_id,omitempty"`

		// Absolute path to the process executable.
		Executable string `json:"executable,omitempty"`

		// Whether the process is connected to an interactive shell.
		// Process interactivity is inferred from the processes file descriptors. If the character device for the controlling tty is the same as stdin and stderr for the process, the process is considered interactive.
		// Note: A non-interactive process can belong to an interactive session and is simply one that does not have open file descriptors reading the controlling TTY on FD 0 (stdin) or writing to the controlling TTY on FD 2 (stderr). A backgrounded process is still considered interactive if stdin and stderr are connected to the controlling TTY.
		Interactive *bool `json:"interactive,omitempty"`

		// Process name.
		// Sometimes called program name or similar.
		Name string `json:"name,omitempty"`

		// The time the process started.
		Start *time.Time `json:"start,omitempty"`

		// The working directory of the process.
		WorkingDirectory string `json:"working_directory,omitempty"`

		// The effective user (euid).
		User struct {
			// Unique identifier of the user.
			ID string `json:"id,omitempty"`

			// Short name or login of the user.
			Name string `json:"name,omitempty"`
		} `json:"user,omitempty"`

		// The effective group (egid).
		Group struct {
			// Unique identifier for the group on the system/platform.
			ID string `json:"id,omitempty"`

			// Name of the group.
			Name string `json:"name,omitempty"`
		} `json:"group,omitempty"`

		// Process id.
		PID uint32 `json:"pid,omitempty"`

		// Array of process arguments, starting with the absolute path to the executable.
		// May be filtered to protect sensitive information.
		Args []string `json:"args,omitempty"`

		// This boolean is used to identify if a leader process is the same as the top level process.
		// For example, if `process.group_leader.same_as_process = true`, it means the process event in question is the leader of its process group. Details under `process.*` like `pid` would be the same under `process.group_leader.*` The same applies for both `process.session_leader` and `process.entry_leader`.
		// This field exists to the benefit of EQL and other rule engines since it's not possible to compare equality between two fields in a single document. e.g `process.entity_id` = `process.group_leader.entity_id` (top level process is the process group leader) OR `process.entity_id` = `process.entry_leader.entity_id` (top level process is the entry session leader)
		// Instead these rules could be written like: `process.group_leader.same_as_process: true` OR `process.entry_leader.same_as_process: true`
		// Note: This field is only set on `process.entry_leader`, `process.session_leader` and `process.group_leader`.
		SameAsProcess *bool `json:"same_as_process,omitempty"`
	} `json:"group_leader,omitempty"`

	// Often the same as entry_leader. When it differs, it represents a session started within another session. e.g. using tmux
	SessionLeader struct {
		// Unique identifier for the process.
		// The implementation of this is specified by the data source, but some examples of what could be used here are a process-generated UUID, Sysmon Process GUIDs, or a hash of some uniquely identifying components of a process.
		// Constructing a globally unique identifier is a common practice to mitigate PID reuse as well as to identify a specific process over time, across multiple monitored hosts.
		EntityID string `json:"entity_id,omitempty"`

		// Absolute path to the process executable.
		Executable string `json:"executable,omitempty"`

		// Whether the process is connected to an interactive shell.
		// Process interactivity is inferred from the processes file descriptors. If the character device for the controlling tty is the same as stdin and stderr for the process, the process is considered interactive.
		// Note: A non-interactive process can belong to an interactive session and is simply one that does not have open file descriptors reading the controlling TTY on FD 0 (stdin) or writing to the controlling TTY on FD 2 (stderr). A backgrounded process is still considered interactive if stdin and stderr are connected to the controlling TTY.
		Interactive *bool `json:"interactive,omitempty"`

		// Process name.
		// Sometimes called program name or similar.
		Name string `json:"name,omitempty"`

		// The time the process started.
		Start *time.Time `json:"start,omitempty"`

		// The working directory of the process.
		WorkingDirectory string `json:"working_directory,omitempty"`

		// The effective user (euid).
		User struct {
			// Unique identifier of the user.
			ID string `json:"id,omitempty"`

			// Short name or login of the user.
			Name string `json:"name,omitempty"`
		} `json:"user,omitempty"`

		// The effective group (egid).
		Group struct {
			// Unique identifier for the group on the system/platform.
			ID string `json:"id,omitempty"`

			// Name of the group.
			Name string `json:"name,omitempty"`
		} `json:"group,omitempty"`

		// Process id.
		PID uint32 `json:"pid,omitempty"`

		// Array of process arguments, starting with the absolute path to the executable.
		// May be filtered to protect sensitive information.
		Args []string `json:"args,omitempty"`

		// This boolean is used to identify if a leader process is the same as the top level process.
		// For example, if `process.group_leader.same_as_process = true`, it means the process event in question is the leader of its process group. Details under `process.*` like `pid` would be the same under `process.group_leader.*` The same applies for both `process.session_leader` and `process.entry_leader`.
		// This field exists to the benefit of EQL and other rule engines since it's not possible to compare equality between two fields in a single document. e.g `process.entity_id` = `process.group_leader.entity_id` (top level process is the process group leader) OR `process.entity_id` = `process.entry_leader.entity_id` (top level process is the entry session leader)
		// Instead these rules could be written like: `process.group_leader.same_as_process: true` OR `process.entry_leader.same_as_process: true`
		// Note: This field is only set on `process.entry_leader`, `process.session_leader` and `process.group_leader`.
		SameAsProcess *bool `json:"same_as_process,omitempty"`
	} `json:"session_leader,omitempty"`

	// First process from terminal or remote access via SSH, SSM, etc OR a service directly started by the init process.
	EntryLeader struct {
		// Unique identifier for the process.
		// The implementation of this is specified by the data source, but some examples of what could be used here are a process-generated UUID, Sysmon Process GUIDs, or a hash of some uniquely identifying components of a process.
		// Constructing a globally unique identifier is a common practice to mitigate PID reuse as well as to identify a specific process over time, across multiple monitored hosts.
		EntityID string `json:"entity_id,omitempty"`

		// Absolute path to the process executable.
		Executable string `json:"executable,omitempty"`

		// Whether the process is connected to an interactive shell.
		// Process interactivity is inferred from the processes file descriptors. If the character device for the controlling tty is the same as stdin and stderr for the process, the process is considered interactive.
		// Note: A non-interactive process can belong to an interactive session and is simply one that does not have open file descriptors reading the controlling TTY on FD 0 (stdin) or writing to the controlling TTY on FD 2 (stderr). A backgrounded process is still considered interactive if stdin and stderr are connected to the controlling TTY.
		Interactive *bool `json:"interactive,omitempty"`

		// Process name.
		// Sometimes called program name or similar.
		Name string `json:"name,omitempty"`

		// The time the process started.
		Start *time.Time `json:"start,omitempty"`

		// The working directory of the process.
		WorkingDirectory string `json:"working_directory,omitempty"`

		EntryMeta struct {
			// The entry type for the entry session leader. Values include: init(e.g systemd), sshd, ssm, kubelet, teleport, terminal, console
			// Note: This field is only set on process.session_leader.
			Type string `json:"type,omitempty"`
		} `json:"entry_meta,omitempty"`

		// The effective user (euid).
		User struct {
			// Unique identifier of the user.
			ID string `json:"id,omitempty"`

			// Short name or login of the user.
			Name string `json:"name,omitempty"`
		} `json:"user,omitempty"`

		// The effective group (egid).
		Group struct {
			// Unique identifier for the group on the system/platform.
			ID string `json:"id,omitempty"`

			// Name of the group.
			Name string `json:"name,omitempty"`
		} `json:"group,omitempty"`

		// Process id.
		PID uint32 `json:"pid,omitempty"`

		// Array of process arguments, starting with the absolute path to the executable.
		// May be filtered to protect sensitive information.
		Args []string `json:"args,omitempty"`

		// This boolean is used to identify if a leader process is the same as the top level process.
		// For example, if `process.group_leader.same_as_process = true`, it means the process event in question is the leader of its process group. Details under `process.*` like `pid` would be the same under `process.group_leader.*` The same applies for both `process.session_leader` and `process.entry_leader`.
		// This field exists to the benefit of EQL and other rule engines since it's not possible to compare equality between two fields in a single document. e.g `process.entity_id` = `process.group_leader.entity_id` (top level process is the process group leader) OR `process.entity_id` = `process.entry_leader.entity_id` (top level process is the entry session leader)
		// Instead these rules could be written like: `process.group_leader.same_as_process: true` OR `process.entry_leader.same_as_process: true`
		// Note: This field is only set on `process.entry_leader`, `process.session_leader` and `process.group_leader`.
		SameAsProcess *bool `json:"same_as_process,omitempty"`
	} `json:"entry_leader,omitempty"`
}

func (p *Process) ToMap() mapstr.M {
	process := mapstr.M{
		"entity_id":         p.EntityID,
		"executable":        p.Executable,
		"name":              p.Name,
		"exit_code":         p.ExitCode,
		"interactive":       p.Interactive,
		"working_directory": p.WorkingDirectory,
		"user": mapstr.M{
			"id":   p.User.ID,
			"name": p.User.Name,
		},
		"group": mapstr.M{
			"id":   p.Group.ID,
			"name": p.Group.Name,
		},
		"pid":  p.PID,
		"vpid": p.Vpid,
		"args": p.Args,
		"thread": mapstr.M{
			"capabilities": mapstr.M{
				"permitted": p.Thread.Capabilities.Permitted,
				"effective": p.Thread.Capabilities.Effective,
			},
		},
		"tty": mapstr.M{
			"char_device": mapstr.M{
				"major": p.TTY.CharDevice.Major,
				"minor": p.TTY.CharDevice.Minor,
			},
		},
		"parent": mapstr.M{
			"entity_id":         p.Parent.EntityID,
			"executable":        p.Parent.Executable,
			"name":              p.Parent.Name,
			"interactive":       p.Parent.Interactive,
			"working_directory": p.Parent.WorkingDirectory,
			"user": mapstr.M{
				"id":   p.Parent.User.ID,
				"name": p.Parent.User.Name,
			},
			"group": mapstr.M{
				"id":   p.Parent.Group.ID,
				"name": p.Parent.Group.Name,
			},
			"pid":  p.Parent.PID,
			"args": p.Parent.Args,
			"thread": mapstr.M{
				"capabilities": mapstr.M{
					"permitted": p.Parent.Thread.Capabilities.Permitted,
					"effective": p.Parent.Thread.Capabilities.Effective,
				},
			},
		},
		"group_leader": mapstr.M{
			"entity_id":         p.GroupLeader.EntityID,
			"executable":        p.GroupLeader.Executable,
			"name":              p.GroupLeader.Name,
			"interactive":       p.GroupLeader.Interactive,
			"working_directory": p.GroupLeader.WorkingDirectory,
			"user": mapstr.M{
				"id":   p.GroupLeader.User.ID,
				"name": p.GroupLeader.User.Name,
			},
			"group": mapstr.M{
				"id":   p.GroupLeader.Group.ID,
				"name": p.GroupLeader.Group.Name,
			},
			"pid":             p.GroupLeader.PID,
			"args":            p.GroupLeader.Args,
			"same_as_process": p.GroupLeader.SameAsProcess,
		},
		"session_leader": mapstr.M{
			"entity_id":         p.SessionLeader.EntityID,
			"executable":        p.SessionLeader.Executable,
			"name":              p.SessionLeader.Name,
			"interactive":       p.SessionLeader.Interactive,
			"working_directory": p.SessionLeader.WorkingDirectory,
			"user": mapstr.M{
				"id":   p.SessionLeader.User.ID,
				"name": p.SessionLeader.User.Name,
			},
			"group": mapstr.M{
				"id":   p.SessionLeader.Group.ID,
				"name": p.SessionLeader.Group.Name,
			},
			"pid":             p.SessionLeader.PID,
			"args":            p.SessionLeader.Args,
			"same_as_process": p.SessionLeader.SameAsProcess,
		},
		"entry_leader": mapstr.M{
			"entity_id":         p.EntryLeader.EntityID,
			"executable":        p.EntryLeader.Executable,
			"name":              p.EntryLeader.Name,
			"interactive":       p.EntryLeader.Interactive,
			"working_directory": p.EntryLeader.WorkingDirectory,
			"entry_meta": mapstr.M{
				"type": p.EntryLeader.EntryMeta.Type,
			},
			"user": mapstr.M{
				"id":   p.EntryLeader.User.ID,
				"name": p.EntryLeader.User.Name,
			},
			"group": mapstr.M{
				"id":   p.EntryLeader.Group.ID,
				"name": p.EntryLeader.Group.Name,
			},
			"pid":             p.EntryLeader.PID,
			"args":            p.EntryLeader.Args,
			"same_as_process": p.EntryLeader.SameAsProcess,
		},
	}

	// nil timestamps will cause a panic within the publisher, only add the mapping if it exists
	if p.Start != nil {
		process.Put("start", p.Start)
	}
	if p.Parent.Start != nil {
		process.Put("parent.start", p.Parent.Start)
	}
	if p.GroupLeader.Start != nil {
		process.Put("group_leader.start", p.GroupLeader.Start)
	}
	if p.SessionLeader.Start != nil {
		process.Put("session_leader.start", p.SessionLeader.Start)
	}
	if p.EntryLeader.Start != nil {
		process.Put("entry_leader.start", p.EntryLeader.Start)
	}

	return process
}
