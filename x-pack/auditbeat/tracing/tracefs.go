// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package tracing

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/joeshaw/multierror"
)

const (
	debugFSTracingPath = "/sys/kernel/debug/tracing"
	traceFSPath        = "/sys/kernel/tracing"
)

var (
	// p[:[GRP/]EVENT] [MOD:]SYM[+offs]|MEMADDR [FETCHARGS] : Set a probe
	// r[MAXACTIVE][:[GRP/]EVENT] [MOD:]SYM[+0] [FETCHARGS] : Set a return probe
	kprobeRegexp *regexp.Regexp = regexp.MustCompile("^([pr])[0-9]*:(?:([^/ ]*)/)?([^/ ]+) ([^ ]+) ?(.*)")

	formatRegexp *regexp.Regexp = regexp.MustCompile("\\s+([^:]+):([^;]*);")
)

// TraceFS is an accessor to manage event tracing via tracefs or debugfs.
type TraceFS struct {
	basePath string
}

// NewTraceFS creates a new accessor for the event tracing feature.
// It autodetects a tracefs mounted on /sys/kernel/tracing or via
// debugfs at /sys/kernel/debug/tracing.
func NewTraceFS() (*TraceFS, error) {
	var errs multierror.Errors
	ptr, err := NewTraceFSWithPath(traceFSPath)
	if err != nil {
		errs = append(errs, err)
		ptr, err = NewTraceFSWithPath(debugFSTracingPath)
		if err != nil {
			errs = append(errs, err)
		} else {
			errs = nil
		}
	}
	return ptr, errs.Err()
}

// NewTraceFSWithPath creates a new accessor for the event tracing feature
// at the given path.
func NewTraceFSWithPath(path string) (*TraceFS, error) {
	if _, err := os.Stat(filepath.Join(path, kprobeCfgFile)); err != nil {
		return nil, err
	}
	return &TraceFS{basePath: path}, nil
}

// IsTraceFSAvailableAt returns nil if the path passed is a mounted tracefs
// or debugfs that supports KProbes. Otherwise returns an error.
func IsTraceFSAvailableAt(path string) error {
	_, err := os.Stat(filepath.Join(path, kprobeCfgFile))
	return err
}

// IsTraceFSAvailable returns nil if a tracefs or debugfs supporting KProbes
// is available at the well-known paths. Otherwise returns an error.
func IsTraceFSAvailable() (err error) {
	for _, path := range []string{traceFSPath, debugFSTracingPath} {
		if err = IsTraceFSAvailableAt(path); err == nil {
			break
		}
	}
	return
}

// ListKProbes lists the currently installed kprobes / kretprobes
func (dfs *TraceFS) ListKProbes() (kprobes []Probe, err error) {
	return dfs.listProbes(kprobeCfgFile)
}

// ListUProbes lists the currently installed uprobes / uretprobes
func (dfs *TraceFS) ListUProbes() (uprobes []Probe, err error) {
	return dfs.listProbes(uprobeCfgFile)
}

func (dfs *TraceFS) listProbes(filename string) (probes []Probe, err error) {
	mapping, ok := probeFileInfo[filename]
	if !ok {
		return nil, fmt.Errorf("unknown probe events file: %s", filename)
	}
	file, err := os.Open(filepath.Join(dfs.basePath, filename))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if matches := kprobeRegexp.FindStringSubmatch(scanner.Text()); len(matches) == 6 {
			typ, ok := mapping[matches[1][0]]
			if !ok {
				return nil, fmt.Errorf("no mapping for probe of type '%c' in file %s", matches[1][0], filename)
			}
			probes = append(probes, Probe{
				Type:      typ,
				Group:     matches[2],
				Name:      matches[3],
				Address:   matches[4],
				Fetchargs: matches[5],
			})
		}
	}
	return probes, nil
}

// AddKProbe installs a new kprobe/kretprobe.
func (dfs *TraceFS) AddKProbe(probe Probe) (err error) {
	return dfs.appendToFile(kprobeCfgFile, probe.String())
}

// RemoveKProbe removes an installed kprobe/kretprobe.
func (dfs *TraceFS) RemoveKProbe(probe Probe) error {
	return dfs.appendToFile(kprobeCfgFile, probe.RemoveString())
}

// AddUProbe installs a new uprobe/uretprobe.
func (dfs *TraceFS) AddUProbe(probe Probe) error {
	return dfs.appendToFile(uprobeCfgFile, probe.String())
}

// RemoveUProbe removes an installed uprobe/uretprobe.
func (dfs *TraceFS) RemoveUProbe(probe Probe) error {
	return dfs.appendToFile(uprobeCfgFile, probe.RemoveString())
}

// RemoveAllKProbes removes all installed kprobes and kretprobes.
func (dfs *TraceFS) RemoveAllKProbes() error {
	return dfs.removeAllProbes(kprobeCfgFile)
}

// RemoveAllUProbes removes all installed uprobes and uretprobes.
func (dfs *TraceFS) RemoveAllUProbes() error {
	return dfs.removeAllProbes(uprobeCfgFile)
}

func (dfs *TraceFS) removeAllProbes(filename string) error {
	file, err := os.OpenFile(filepath.Join(dfs.basePath, filename), os.O_WRONLY|os.O_TRUNC|os.O_SYNC, 0)
	if err != nil {
		return err
	}
	return file.Close()
}

func (dfs *TraceFS) appendToFile(filename string, desc string) error {
	file, err := os.OpenFile(filepath.Join(dfs.basePath, filename), os.O_WRONLY|os.O_APPEND|os.O_SYNC, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(desc + "\n")
	return err
}

// FieldType describes the type of a field in a event tracing probe.
type FieldType uint8

const (
	// FieldTypeInteger describes a fixed-size integer field.
	FieldTypeInteger = iota

	// FieldTypeString describes a string field.
	FieldTypeString

	// FieldTypeMeta describes the metadata.
	FieldTypeMeta

	// FieldTypeRaw describes a field of raw bytes.
	FieldTypeRaw
)

// Field describes a field returned by a event tracing probe.
type Field struct {
	// Name is the name given to the field.
	Name string

	// Offset of the field inside the raw event.
	Offset int

	// Size in bytes of the serialised field: 1, 2, 4, 8 for fixed size integers
	// or 4 for strings.
	Size int

	// Signed tells whether an integer is signed (true) or unsigned (false).
	Signed bool

	// Type of field.
	Type FieldType
}

// ProbeFormat describes a Probe and the serialisation format used to
// encode its arguments into a tracing event.
type ProbeFormat struct {
	// ID is the numeric ID given to this kprobe/kretprobe by the kernel.
	ID int

	// Probe is the probe described by this format.
	Probe Probe

	// Fields is a description of the fields (fetchargs) set by this kprobe.
	Fields map[string]Field
}

var integerTypes = map[string]uint8{
	"char":  1,
	"s8":    1,
	"u8":    1,
	"short": 2,
	"s16":   2,
	"u16":   2,
	"int":   4,
	"s32":   4,
	"u32":   4,
	"long":  strconv.IntSize / 8,
	"s64":   8,
	"u64":   8,
}

// LoadProbeFormat returns the format used for serialisation of the given
// kprobe/kretprobe into a tracing event. The probe needs to be installed
// for the kernel to provide its format.
func (dfs *TraceFS) LoadProbeFormat(probe Probe) (format ProbeFormat, err error) {
	path := filepath.Join(dfs.basePath, "events", probe.EffectiveGroup(), probe.Name, "format")
	file, err := os.Open(path)
	if err != nil {
		return format, err
	}
	defer file.Close()
	format.Probe = probe
	format.Fields = make(map[string]Field)
	scanner := bufio.NewScanner(file)
	parseFormat := false
	for scanner.Scan() {
		line := scanner.Text()
		if !parseFormat {
			// Parse the header
			parts := strings.SplitN(line, ": ", 2)
			switch {
			case len(parts) == 2 && parts[0] == "ID":
				if format.ID, err = strconv.Atoi(parts[1]); err != nil {
					return format, err
				}
			case len(parts) == 1 && parts[0] == "format:":
				parseFormat = true
			}
		} else {
			// Parse the fields
			// Ends on the first line that doesn't start with a TAB
			if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
				break
			}

			// Find all "<key>:<value>;" matches
			// The actual format is:
			// "\tfield:%s %s;\toffset:%u;\tsize:%u;\tsigned:%d;\n"
			var f Field
			matches := formatRegexp.FindAllStringSubmatch(line, -1)
			if len(matches) != 4 {
				continue
			}

			for _, match := range matches {
				if len(match) != 3 {
					continue
				}
				key, value := match[1], match[2]
				switch key {
				case "field":
					fparts := strings.Split(value, " ")
					n := len(fparts)
					if n < 2 {
						return format, fmt.Errorf("bad format for kprobe '%s': `field` has no type: %s", probe.String(), value)
					}

					fparts, f.Name = fparts[:n-1], fparts[n-1]
					typeIdx, isDataLoc := -1, false

					for idx, part := range fparts {
						switch part {
						case "signed", "unsigned":
							// ignore
						case "__data_loc":
							isDataLoc = true
						default:
							if typeIdx != -1 {
								return format, fmt.Errorf("bad format for kprobe '%s': unknown parameter=`%s` in type=`%s`", probe.String(), part, value)
							}
							typeIdx = idx
						}
					}
					if typeIdx == -1 {
						return format, fmt.Errorf("bad format for kprobe '%s': type not found in `%s`", probe.String(), value)
					}
					intLen, isInt := integerTypes[fparts[typeIdx]]
					if isInt {
						f.Type = FieldTypeInteger
						f.Size = int(intLen)
					} else {
						if fparts[typeIdx] != "char[]" || !isDataLoc {
							return format, fmt.Errorf("bad format for kprobe '%s': unsupported type in `%s`", probe.String(), value)
						}
						f.Type = FieldTypeString
					}

				case "offset":
					f.Offset, err = strconv.Atoi(value)
					if err != nil {
						return format, err
					}

				case "size":
					prev := f.Size
					f.Size, err = strconv.Atoi(value)
					if err != nil {
						return format, err
					}
					if prev != 0 && prev != f.Size {
						return format, fmt.Errorf("bad format for kprobe '%s': int field length mismatch at `%s`", probe.String(), line)
					}

				case "signed":
					f.Signed = len(value) > 0 && value[0] == '1'
				}
			}
			if f.Type == FieldTypeString && f.Size != 4 {
				return format, fmt.Errorf("bad format for kprobe '%s': unexpected size for string in `%s`", probe.String(), line)
			}
			format.Fields[f.Name] = f
		}
	}
	return format, nil
}

// AvailableFilterFunctions returns a list of all the symbols that can be used
// in a KProbe's address.
func (dfs *TraceFS) AvailableFilterFunctions() (functions []string, err error) {
	path := filepath.Join(dfs.basePath, "available_filter_functions")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		functions = append(functions, scanner.Text())
	}
	return functions, nil
}
