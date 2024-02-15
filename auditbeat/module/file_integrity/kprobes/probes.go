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

func newProbeManager(e executor) (*probeManager, error) {
	fs := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: getSymbolInfoRuntime,
	}

	if err := loadFsNotifySymbol(fs); err != nil {
		return nil, err
	}

	if err := loadFsNotifyParentSymbol(fs); err != nil {
		return nil, err
	}

	if err := loadFsNotifyNameRemoveSymbol(fs); err != nil {
		return nil, err
	}

	if err := loadVFSGetAttrSymbol(fs, e); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *probeManager) shouldBuild(spec *tkbtf.Spec) bool {
	for _, check := range fs.buildChecks {
		if !check(spec) {
			return false
		}
	}

	return true
}

func (fs *probeManager) build(spec *tkbtf.Spec) (map[tracing.Probe]tracing.AllocateFn, error) {
	trProbesMap := make(map[tracing.Probe]tracing.AllocateFn)

	for _, s := range fs.symbols {
		probesWithAlloc, err := s.buildProbes(spec)
		if err != nil {
			return nil, err
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

func (fs *probeManager) onErr(err error) bool {
	repeat := false
	for _, s := range fs.symbols {
		if s.onErr(err) {
			repeat = true
		}
	}

	return repeat
}
