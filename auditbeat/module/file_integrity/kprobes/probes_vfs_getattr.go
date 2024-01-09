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
