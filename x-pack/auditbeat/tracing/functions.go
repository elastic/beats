// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package tracing

import (
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
)

func (e *Engine) isKernelFunctionAvailable(name string, tracingFns common.StringSet) bool {
	if tracingFns.Count() != 0 {
		return tracingFns.Has(name)
	}
	defer e.uninstallProbes()
	checkProbe := ProbeDef{
		Probe: Probe{
			Name:      "check_" + name,
			Address:   name,
			Fetchargs: "%ax:u64", // dump decoder needs it.
		},
		Decoder: NewDumpDecoder,
	}
	_, _, err := e.installProbe(checkProbe)
	return err == nil
}

func loadTracingFunctions(tfs *TraceFS) (common.StringSet, error) {
	fnList, err := tfs.AvailableFilterFunctions()
	if err != nil {
		return nil, err
	}
	// This uses make() instead of common.MakeStringSet() because the later
	// doesn't allow to create empty sets.
	functions := common.StringSet(make(map[string]struct{}, len(fnList)))
	for _, fn := range fnList {
		// Strip the module name (if any)
		end := strings.IndexByte(fn, ' ')
		if end == -1 {
			end = len(fn)
		}
		functions.Add(fn[:end])
	}
	return functions, nil
}
