// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package tracing

import (
	"strings"
)

const (
	kprobeCfgFile = "kprobe_events"
	uprobeCfgFile = "uprobe_events"
)

// ProbeType represents the type of a probe.
type ProbeType uint8

const (
	// TypeKProbe represents a KProbe.
	TypeKProbe ProbeType = iota

	// TypeKRetProbe represents a regular KRetProbe.
	TypeKRetProbe

	// TypeUProbe represents a UProbe.
	TypeUProbe

	// TypeURetProbe represents a URetProbe.
	TypeURetProbe
)

type probeSettings struct {
	defaultGroup string
	cfgFile      string
	prefix       byte
}

var probeCfg = map[ProbeType]probeSettings{
	TypeKProbe:    {"kprobes", kprobeCfgFile, 'p'},
	TypeKRetProbe: {"kprobes", kprobeCfgFile, 'r'},
	TypeUProbe:    {"uprobes", uprobeCfgFile, 'p'},
	TypeURetProbe: {"uprobes", uprobeCfgFile, 'r'},
}

var probeFileInfo = make(map[string]map[byte]ProbeType, 2)

func init() {
	for typ, cfg := range probeCfg {
		m := probeFileInfo[cfg.cfgFile]
		if m == nil {
			m = make(map[byte]ProbeType, 2)
			probeFileInfo[cfg.cfgFile] = m
		}
		m[cfg.prefix] = typ
	}
}

// Probe represents a probe ([KU](Ret)?Probe).
type Probe struct {
	// Type tells whether this is a kprobe, kretprobe, uprobe or uretprobe.
	Type ProbeType

	// Group is the probe's group. If left unset, it will be automatically
	// set to "kprobes" or "uprobes", depending on type. This affects where
	// the probe configuration resides inside `tracefs`:
	// /sys/kernel/tracing/events/<group>/<name>
	Group string

	// Name is the name given to this probe. If left empty (not recommended),
	// the kernel will give it a name based on Address. Then it will be
	// necessary to list the installed probes and figure out which one it is,
	// so it can be used with LoadProbeDescription.
	Name string

	// Address is the function name or address where the probe will be installed.
	// According to the docs:
	// - `[MOD:]SYM[+offs]|MEMADDR` for KProbes.
	// - `PATH:OFFSET` for UProbes.
	Address string

	// Fetchargs is the string of arguments that will be fetched when the probe
	// is hit.
	Fetchargs string

	// Filter is a filter expression to apply to this probe.
	Filter string
}

// String converts this probe to the textual representation expected by the Kernel.
func (p *Probe) String() string {
	var builder strings.Builder
	builder.WriteByte(p.settings().prefix)
	builder.WriteByte(':')
	if len(p.Group) > 0 {
		builder.WriteString(p.Group)
		builder.WriteByte('/')
	}
	builder.WriteString(p.Name)
	builder.WriteByte(' ')
	builder.WriteString(p.Address)
	builder.WriteByte(' ')
	builder.WriteString(p.Fetchargs)
	return builder.String()
}

func (p *Probe) settings() probeSettings {
	return probeCfg[p.Type]
}

// RemoveString converts this probe to the textual representation needed to
// remove the probe.
func (p *Probe) RemoveString() string {
	var builder strings.Builder
	builder.WriteString("-:")
	if len(p.Group) > 0 {
		builder.WriteString(p.Group)
		builder.WriteByte('/')
	}
	builder.WriteString(p.Name)
	return builder.String()
}

// EffectiveGroup is the actual group used to access this kprobe inside debugfs.
// It is the group given when setting the probe, or "kprobes" if unset.
func (p *Probe) EffectiveGroup() string {
	if len(p.Group) > 0 {
		return p.Group
	}
	return p.settings().defaultGroup
}
