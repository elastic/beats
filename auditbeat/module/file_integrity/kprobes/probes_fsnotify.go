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
	"math"

	tkbtf "github.com/elastic/tk-btf"

	"github.com/cilium/ebpf/btf"
)

type fsNotifySymbol struct {
	symbolName        string
	inodeProbeFilter  string
	dentryProbeFilter string
	pathProbeFilter   string
	lastOnErr         error
	seenSpecs         map[*tkbtf.Spec]struct{}
}

func loadFsNotifySymbol(probeMgr *probeManager) error {
	symbolInfo, err := probeMgr.getSymbolInfoRuntime("fsnotify")
	if err != nil {
		return fmt.Errorf("fsnotify symbol does not exist: %w", err)
	}

	if symbolInfo.isOptimised {
		return fmt.Errorf("symbol %s is optimised", symbolInfo.symbolName)
	}

	probeMgr.buildChecks = append(probeMgr.buildChecks, func(spec *tkbtf.Spec) bool {
		return spec.ContainsSymbol(symbolInfo.symbolName)
	})

	// default filters for all three fsnotify probes enable mask_create, mask_delete, mask_attrib, mask_modify,
	// mask_moved_to, and mask_moved_from events.
	probeMgr.symbols = append(probeMgr.symbols, &fsNotifySymbol{
		symbolName: symbolInfo.symbolName,
	})

	return nil
}

// setKprobeFiltersFromBTF fetches the enum values of the fsnotify_data_type enum from the BTF,
// and uses them to construct the probe filters, so that each of the three fsnotify probes only forwards
// events from the matching data type.
func (sym *fsNotifySymbol) setKprobeFiltersFromBTF(spec *tkbtf.Spec) error {
	types, err := spec.AnyTypesByName("fsnotify_data_type")
	if err != nil {
		// Kernels pre-5.7 do not have the fsnotify_data_type enum and instead code the data types as #define statements.
		// These do not show up in the BTF output, and thus we need a manual fallback.
		if errors.Is(err, btf.ErrNotFound) {
			// TODO: log
			sym.setKprobeFilters(1, 2, 3)
			return nil
		}
		return fmt.Errorf("error fetching fsnotify_data_type from BTF: %w", err)
	}
	btfEnum := &btf.Enum{}
	found := false

	for _, foundType := range types {
		btfEnum, found = foundType.(*btf.Enum)
		if found {
			break
		}
	}

	if !found || btfEnum == nil {
		return fmt.Errorf("fsnotify_data_type not an enum, this may be a kernel support issue")
	}

	var dentry, path, inode uint64 = math.MaxUint64, math.MaxUint64, math.MaxUint64
	for _, enumType := range btfEnum.Values {
		switch enumType.Name {
		case "FSNOTIFY_EVENT_PATH":
			path = enumType.Value
		case "FSNOTIFY_EVENT_INODE":
			inode = enumType.Value
		case "FSNOTIFY_EVENT_DENTRY":
			dentry = enumType.Value
		}
	}

	if dentry == math.MaxUint64 || inode == math.MaxUint64 || path == math.MaxUint64 {
		return fmt.Errorf("values missing from fsnotify_data_type struct, may be a kernel support issue: %v", btfEnum.Values)
	}

	sym.setKprobeFilters(path, inode, dentry)
	return nil
}

func (sym *fsNotifySymbol) setKprobeFilters(eventPath uint64, eventInode uint64, eventDentry uint64) {
	sym.pathProbeFilter = fmt.Sprintf("(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==%d", eventPath)
	sym.inodeProbeFilter = fmt.Sprintf("(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==%d && nptr!=0", eventInode)
	sym.dentryProbeFilter = fmt.Sprintf("(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==%d", eventDentry)
}

func (sym *fsNotifySymbol) buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error) {
	allocFunc := allocProbeEvent

	_, seen := sym.seenSpecs[spec]
	if !seen {

		if sym.seenSpecs == nil {
			sym.seenSpecs = make(map[*tkbtf.Spec]struct{})
		}

		sym.lastOnErr = nil
		// reset probe filters for each new spec
		// this probes shouldn't cause any ErrVerifyOverlappingEvents or ErrVerifyMissingEvents
		// for linux kernel versions linux 5.17+, thus we start from here. To see how we handle all
		// linux kernels filter variation check OnErr() method.
		sym.seenSpecs[spec] = struct{}{}
		// ***************** TO WHOEVER IS DEBUGGING THIS CODE IN THE FUTURE: *****************
		// the kprobe filters work by giving each of the three fsnotify kprobes (one each for dentry,
		// inode and path-based events) filtering rules based on the presence
		// of mask values, and the value of the fsnotify() data_type field.
		// If for some reason those mask/data_type fields become invalid (changes in the kernel, kprobe bug on our end)
		// the events that get get through the filters might look strange or corrupted with no apparent pattern.
		// Your first debugging step should be to disable these filters so you can see events as they arrive.
		// NOTE: in the future we may want to investigate alternatives to these filters (doing the filtering in userland, etc).
		err := sym.setKprobeFiltersFromBTF(spec)
		if err != nil {
			return nil, fmt.Errorf("error creating kprobe filters from BTF: %w", err)
		}
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
	).SetFilter(sym.pathProbeFilter)

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
	).SetFilter(sym.inodeProbeFilter)

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
	).SetFilter(sym.dentryProbeFilter)

	btfSymbol := tkbtf.NewSymbol(sym.symbolName).AddProbes(
		inodeProbe,
		dentryProbe,
		pathProbe,
	)

	if err := spec.BuildSymbol(btfSymbol); err != nil {
		return nil, fmt.Errorf("error building fsnotify probes: %w", err)
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

func (sym *fsNotifySymbol) onErr(err error) bool {
	if sym.lastOnErr != nil && errors.Is(err, sym.lastOnErr) {
		return false
	}

	sym.lastOnErr = err

	switch {
	case errors.Is(err, ErrVerifyOverlappingEvents):

		// on ErrVerifyOverlappingEvents for linux kernel versions < 5.7 the __fsnotify_parent
		// probe is capturing and sending the modify events as well, thus disable them for
		// fsnotify and return true to signal a retry.
		sym.inodeProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==2 && nptr!=0"
		sym.dentryProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==3"
		sym.pathProbeFilter = "(mc==1 || md==1 || ma==1 || mmt==1 || mmf==1) && dt==1"

		return true
	case errors.Is(err, ErrVerifyMissingEvents):

		// on ErrVerifyMissingEvents for linux kernel versions 5.10 - 5.16 the __fsnotify_parent
		// probe is not capturing and sending the modify attributes events for directories, thus
		// we adjust the filters to allow them flowing through fsnotify and return true to signal
		// a retry.
		sym.pathProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==1"
		sym.inodeProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==2 && (nptr!=0 || (mid==1 && ma==1))"
		sym.dentryProbeFilter = "(mc==1 || md==1 || ma==1 || mm==1 || mmt==1 || mmf==1) && dt==3"

		return true
	default:
		return false
	}
}
