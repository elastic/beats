// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package kprobes

import (
	"fmt"

	"github.com/elastic/beats/v7/auditbeat/tracing"

	tkbtf "github.com/elastic/tk-btf"

	"golang.org/x/sys/unix"
)

const (
	fsEventModify    = uint32(unix.IN_MODIFY)
	fsEventAttrib    = uint32(unix.IN_ATTRIB)
	fsEventMovedFrom = uint32(unix.IN_MOVED_FROM)
	fsEventMovedTo   = uint32(unix.IN_MOVED_TO)
	fsEventCreate    = uint32(unix.IN_CREATE)
	fsEventDelete    = uint32(unix.IN_DELETE)
	fsEventIsDir     = uint32(unix.IN_ISDIR)
)

const (
	devMajor = uint32(0xFFF00000)
	devMinor = uint32(0x3FF)
)

type probeWithAllocFunc struct {
	probe      *tkbtf.Probe
	allocateFn func() any
}

type shouldBuildCheck func(spec *tkbtf.Spec) bool

type symbol interface {
	buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error)

	onErr(err error) bool
}

type probeManager struct {
	symbols              []symbol
	buildChecks          []shouldBuildCheck
	getSymbolInfoRuntime func(symbolName string) (runtimeSymbolInfo, error)
}

func newProbeManager(exec executor) (*probeManager, error) {
	probeMgr := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: getSymbolInfoRuntime,
	}

	if err := loadFsNotifySymbol(probeMgr); err != nil {
		return nil, fmt.Errorf("error loading fsnotify symbol: %w", err)
	}

	if err := loadFsNotifyParentSymbol(probeMgr); err != nil {
		return nil, fmt.Errorf("error loading fsnotify parent symbol: %w", err)
	}

	if err := loadFsNotifyNameRemoveSymbol(probeMgr); err != nil {
		return nil, fmt.Errorf("error loading fsnotify_nameremove symbol: %w", err)
	}

	if err := loadVFSGetAttrSymbol(probeMgr, exec); err != nil {
		return nil, fmt.Errorf("error loading vfs_getattr_nosec symbol: %w", err)
	}

	return probeMgr, nil
}

func (probeMgr *probeManager) shouldBuild(spec *tkbtf.Spec) bool {
	for _, check := range probeMgr.buildChecks {
		if !check(spec) {
			return false
		}
	}

	return true
}

func (probeMgr *probeManager) build(spec *tkbtf.Spec) (map[tracing.Probe]tracing.AllocateFn, error) {
	trProbesMap := make(map[tracing.Probe]tracing.AllocateFn)

	for _, sym := range probeMgr.symbols {
		probesWithAlloc, err := sym.buildProbes(spec)
		if err != nil {
			return nil, fmt.Errorf("error building probe: %w", err)
		}

		for _, p := range probesWithAlloc {
			trProbe := tracing.Probe{
				Group:     "auditbeat_fim",
				Name:      p.probe.GetID(),
				Address:   p.probe.GetSymbolName(),
				Fetchargs: p.probe.GetTracingEventProbe(),
				Filter:    p.probe.GetTracingEventFilter(),
			}
			switch p.probe.GetType() {
			case tkbtf.ProbeTypeKRetProbe:
				trProbe.Type = tracing.TypeKRetProbe
			default:
				trProbe.Type = tracing.TypeKProbe
			}
			trProbesMap[trProbe] = p.allocateFn
		}
	}

	return trProbesMap, nil
}

func (probeMgr *probeManager) onErr(err error) bool {
	repeat := false
	for _, s := range probeMgr.symbols {
		if s.onErr(err) {
			repeat = true
		}
	}

	return repeat
}
