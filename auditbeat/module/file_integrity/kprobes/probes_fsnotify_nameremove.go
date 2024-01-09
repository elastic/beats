package kprobes

import (
	"errors"
	"fmt"
	tkbtf "github.com/elastic/tk-btf"
)

type fsNotifyNameRemoveSymbol struct {
	symbolName string
}

func loadFsNotifyNameRemoveSymbol(s *probeManager) error {

	symbolInfo, err := s.getSymbolInfoRuntime("fsnotify_nameremove")
	if err != nil {
		if errors.Is(err, ErrSymbolNotFound) {
			s.buildChecks = append(s.buildChecks, func(spec *tkbtf.Spec) bool {
				return !spec.ContainsSymbol(symbolInfo.symbolName)
			})
			return nil
		}
		return err
	}

	if symbolInfo.isOptimised {
		return fmt.Errorf("symbol %s is optimised", symbolInfo.symbolName)
	}

	s.buildChecks = append(s.buildChecks, func(spec *tkbtf.Spec) bool {
		return spec.ContainsSymbol(symbolInfo.symbolName)
	})

	s.symbols = append(s.symbols, &fsNotifyNameRemoveSymbol{
		symbolName: symbolInfo.symbolName,
	})

	return nil
}

func (f *fsNotifyNameRemoveSymbol) buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error) {
	allocFunc := allocDeleteProbeEvent

	probe := tkbtf.NewKProbe().AddFetchArgs(
		tkbtf.NewFetchArg("mid", "u32").FuncParamWithName("isdir"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithName("dentry", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithName("dentry", "d_parent", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithName("dentry", "d_name", "name"),
	)

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

func (f *fsNotifyNameRemoveSymbol) onErr(_ error) bool {
	return false
}
