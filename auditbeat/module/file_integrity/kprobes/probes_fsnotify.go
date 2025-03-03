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

type fsNotifySymbol struct {
	symbolName        string
	inodeProbeFilter  string
	dentryProbeFilter string
	pathProbeFilter   string
	lastOnErr         error
	seenSpecs         map[*tkbtf.Spec]struct{}
}

func loadFsNotifySymbol(s *probeManager) error {
	symbolInfo, err := s.getSymbolInfoRuntime("fsnotify")
	if err != nil {
		return err
	}

	if symbolInfo.isOptimised {
		return fmt.Errorf("symbol %s is optimised", symbolInfo.symbolName)
	}

	s.buildChecks = append(s.buildChecks, func(spec *tkbtf.Spec) bool {
		return spec.ContainsSymbol(symbolInfo.symbolName)
	})

	// default filters for all three fsnotify probes enable mask_create, mask_delete, mask_attrib, mask_modify,
	// mask_moved_to, and mask_moved_from events.
	s.symbols = append(s.symbols, &fsNotifySymbol{
		symbolName: symbolInfo.symbolName,
	})

	return nil
}

func (f *fsNotifySymbol) buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error) {
	allocFunc := allocProbeEvent

	_, seen := f.seenSpecs[spec]
	if !seen {

		if f.seenSpecs == nil {
			f.seenSpecs = make(map[*tkbtf.Spec]struct{})
		}

		f.lastOnErr = nil
		// reset probe filters for each new spec
		// this probes shouldn't cause any ErrVerifyOverlappingEvents or ErrVerifyMissingEvents
		// for linux kernel versions linux 5.17+, thus we start from here. To see how we handle all
		// linux kernels filter variation check OnErr() method.
		f.seenSpecs[spec] = struct{}{}
		f.pathProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==1"
		f.inodeProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==2 && nptr!=0"
		f.dentryProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==3"
	}

	pathProbe := tkbtf.NewKProbe().SetRef("fsnotify_path").AddFetchArgs(
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_parent", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("mc", tkbtf.BitFieldTypeMask(fsEventCreate)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("md", tkbtf.BitFieldTypeMask(fsEventDelete)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("ma", tkbtf.BitFieldTypeMask(fsEventAttrib)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mm", tkbtf.BitFieldTypeMask(fsEventModify)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mid", tkbtf.BitFieldTypeMask(fsEventIsDir)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmt", tkbtf.BitFieldTypeMask(fsEventMovedTo)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmf", tkbtf.BitFieldTypeMask(fsEventMovedFrom)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("dt", "s32").FuncParamWithName("data_type").FuncParamWithName("data_is"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithCustomType("data", tkbtf.WrapPointer, "path", "dentry", "d_name", "name"),
	).SetFilter(f.pathProbeFilter)

	inodeProbe := tkbtf.NewKProbe().SetRef("fsnotify_inode").AddFetchArgs(
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithName("dir", "i_ino").FuncParamWithName("to_tell", "i_ino"),
		tkbtf.NewFetchArg("mc", tkbtf.BitFieldTypeMask(fsEventCreate)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("md", tkbtf.BitFieldTypeMask(fsEventDelete)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("ma", tkbtf.BitFieldTypeMask(fsEventAttrib)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mm", tkbtf.BitFieldTypeMask(fsEventModify)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mid", tkbtf.BitFieldTypeMask(fsEventIsDir)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmt", tkbtf.BitFieldTypeMask(fsEventMovedTo)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmf", tkbtf.BitFieldTypeMask(fsEventMovedFrom)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("nptr", "u64").FuncParamWithName("file_name"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithCustomType("data", tkbtf.WrapPointer, "inode", "i_ino"),
		tkbtf.NewFetchArg("dt", "s32").FuncParamWithName("data_type").FuncParamWithName("data_is"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("dir", "i_sb", "s_dev").FuncParamWithName("to_tell", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("dir", "i_sb", "s_dev").FuncParamWithName("to_tell", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithName("file_name", "name").FuncParamWithName("file_name"),
	).SetFilter(f.inodeProbeFilter)

	dentryProbe := tkbtf.NewKProbe().SetRef("fsnotify_dentry").AddFetchArgs(
		tkbtf.NewFetchArg("mc", tkbtf.BitFieldTypeMask(fsEventCreate)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("md", tkbtf.BitFieldTypeMask(fsEventDelete)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("ma", tkbtf.BitFieldTypeMask(fsEventAttrib)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mm", tkbtf.BitFieldTypeMask(fsEventModify)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mid", tkbtf.BitFieldTypeMask(fsEventIsDir)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmt", tkbtf.BitFieldTypeMask(fsEventMovedTo)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmf", tkbtf.BitFieldTypeMask(fsEventMovedFrom)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("dt", "s32").FuncParamWithName("data_type").FuncParamWithName("data_is"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_parent", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithCustomType("data", tkbtf.WrapPointer, "dentry", "d_name", "name"),
	).SetFilter(f.dentryProbeFilter)

	btfSymbol := tkbtf.NewSymbol(f.symbolName).AddProbes(
		inodeProbe,
		dentryProbe,
		pathProbe,
	)

	if err := spec.BuildSymbol(btfSymbol); err != nil {
		return nil, err
	}

	return []*probeWithAllocFunc{
		{
			probe:      inodeProbe,
			allocateFn: allocFunc,
		},
		{
			probe:      dentryProbe,
			allocateFn: allocFunc,
		},
		{
			probe:      pathProbe,
			allocateFn: allocFunc,
		},
	}, nil
}

func (f *fsNotifySymbol) onErr(err error) bool {
	if f.lastOnErr != nil && errors.Is(err, f.lastOnErr) {
		return false
	}

	f.lastOnErr = err

	switch {
	case errors.Is(err, ErrVerifyOverlappingEvents):

		// on ErrVerifyOverlappingEvents for linux kernel versions < 5.7 the __fsnotify_parent
		// probe is capturing and sending the modify events as well, thus disable them for
		// fsnotify and return true to signal a retry.
		f.inodeProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==2 && nptr!=0"
		f.dentryProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==3"
		f.pathProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==1"

		return true
	case errors.Is(err, ErrVerifyMissingEvents):

		// on ErrVerifyMissingEvents for linux kernel versions 5.10 - 5.16 the __fsnotify_parent
		// probe is not capturing and sending the modify attributes events for directories, thus
		// we adjust the filters to allow them flowing through fsnotify and return true to signal
		// a retry.
		f.pathProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==1"
		f.inodeProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==2 && (nptr!=0 || (mid==1 && ma==1))"
		f.dentryProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==3"

		return true
	default:
		return false
	}
}
