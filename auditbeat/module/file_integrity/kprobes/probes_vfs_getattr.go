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
	"errors"
	"fmt"

	tkbtf "github.com/elastic/tk-btf"
)

type vfsGetAttrSymbol struct {
	symbolName string
	filter     string
}

func loadVFSGetAttrSymbol(s *probeManager, e executor) error {
	// get the vfs_getattr_nosec symbol information
	symbolInfo, err := s.getSymbolInfoRuntime("vfs_getattr_nosec")
	if err != nil {
		if !errors.Is(err, ErrSymbolNotFound) {
			return err
		}

		// for older kernel versions use the vfs_getattr symbol
		symbolInfo, err = s.getSymbolInfoRuntime("vfs_getattr")
		if err != nil {
			return err
		}
	}

	// we do not support optimised symbols
	if symbolInfo.isOptimised {
		return fmt.Errorf("symbol %s is optimised", symbolInfo.symbolName)
	}

	s.buildChecks = append(s.buildChecks, func(spec *tkbtf.Spec) bool {
		return spec.ContainsSymbol(symbolInfo.symbolName)
	})

	s.symbols = append(s.symbols, &vfsGetAttrSymbol{
		symbolName: symbolInfo.symbolName,
		filter:     fmt.Sprintf("common_pid==%d", e.GetTID()),
	})

	return nil
}

func (f *vfsGetAttrSymbol) buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error) {
	allocFunc := allocMonitorProbeEvent

	probe := tkbtf.NewKProbe().AddFetchArgs(
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithName("path", "dentry", "d_parent", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithName("path", "dentry", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("path", "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("path", "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("path", "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("path", "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithName("path", "dentry", "d_name", "name"),
	).SetFilter(f.filter)

	btfSymbol := tkbtf.NewSymbol(f.symbolName).AddProbes(probe)

	if err := spec.BuildSymbol(btfSymbol); err != nil {
		return nil, err
	}

	return []*probeWithAllocFunc{
		{
			probe:      probe,
			allocateFn: allocFunc,
		},
	}, nil
}

func (f *vfsGetAttrSymbol) onErr(_ error) bool {
	return false
}
