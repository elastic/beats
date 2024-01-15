package kprobes

import (
	"errors"
	"fmt"

	tkbtf "github.com/elastic/tk-btf"
)

type fsNotifyParentSymbol struct {
	symbolName string
	filter     string
}

func loadFsNotifyParentSymbol(s *probeManager) error {

	symbolInfo, err := s.getSymbolInfoRuntime("__fsnotify_parent")
	if err != nil {
		if !errors.Is(err, ErrSymbolNotFound) {
			return err
		}

		symbolInfo, err = s.getSymbolInfoRuntime("fsnotify_parent")
		if err != nil {
			return err
		}
	}

	if symbolInfo.isOptimised {
		return fmt.Errorf("symbol %s is optimised", symbolInfo.symbolName)
	}

	s.buildChecks = append(s.buildChecks, func(spec *tkbtf.Spec) bool {
		return spec.ContainsSymbol(symbolInfo.symbolName)
	})

	s.symbols = append(s.symbols, &fsNotifyParentSymbol{
		symbolName: symbolInfo.symbolName,
		filter:     "(mc==1 || md==1 || ma==1 || mm==1)",
	})

	return nil
}

func (f *fsNotifyParentSymbol) buildProbes(spec *tkbtf.Spec) ([]*probeWithAllocFunc, error) {
	allocFunc := allocProbeEvent

	probe := tkbtf.NewKProbe().AddFetchArgs(
		tkbtf.NewFetchArg("mc", tkbtf.BitFieldTypeMask(fsEventCreate)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("md", tkbtf.BitFieldTypeMask(fsEventDelete)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("ma", tkbtf.BitFieldTypeMask(fsEventAttrib)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mm", tkbtf.BitFieldTypeMask(fsEventModify)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mid", tkbtf.BitFieldTypeMask(fsEventIsDir)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmt", tkbtf.BitFieldTypeMask(fsEventMovedTo)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("mmf", tkbtf.BitFieldTypeMask(fsEventMovedFrom)).FuncParamWithName("mask"),
		tkbtf.NewFetchArg("fi", "u64").FuncParamWithName("dentry", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("fdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("dentry", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pi", "u64").FuncParamWithName("dentry", "d_parent", "d_inode", "i_ino"),
		tkbtf.NewFetchArg("pdmj", tkbtf.BitFieldTypeMask(devMajor)).FuncParamWithName("dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("pdmn", tkbtf.BitFieldTypeMask(devMinor)).FuncParamWithName("dentry", "d_parent", "d_inode", "i_sb", "s_dev"),
		tkbtf.NewFetchArg("fn", "string").FuncParamWithName("dentry", "d_name", "name"),
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

func (f *fsNotifyParentSymbol) onErr(_ error) bool {
	return false
}
