// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package socket

import (
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
)

// baseTemplateVars contains the substitution variables useful to write KProbes
// in a portable fashion. During setup it will be populated with arch-dependent
// variables and guessed offsets.
var baseTemplateVars = common.MapStr{
	// Constants to make KProbes more readable
	"AF_INET":     2,
	"AF_INET6":    10,
	"IPPROTO_TCP": 6,
	"IPPROTO_UDP": 17,
	"SOCK_STREAM": 2,
	"TCP_CLOSED":  7,

	// Offset of the ith element on an array of pointers
	"POINTER_INDEX": func(index int) int {
		return int(unsafe.Sizeof(uintptr(0))) * index
	},
}

// These functions names vary between kernel versions. The first available one
// will be selected during setup.
var functionAlternatives = map[string][]string{
	"SYS_UNAME":         syscallAlternatives("newuname"),
	"SYS_EXECVE":        syscallAlternatives("execve"),
	"IP_LOCAL_OUT":      {"ip_local_out", "ip_local_out_sk"},
	"SYS_GETTIMEOFDAY":  syscallAlternatives("gettimeofday"),
	"RECV_UDP_DATAGRAM": {"__skb_recv_udp", "__skb_recv_datagram"},
}

func syscallAlternatives(syscall string) []string {
	return []string{
		"SyS_" + syscall,
		"sys_" + syscall,
		"__x64_sys_" + syscall,
	}
}

func LoadTracingFunctions(tfs *tracing.TraceFS) (common.StringSet, error) {
	fnList, err := tfs.AvailableFilterFunctions()
	if err != nil {
		return nil, err
	}
	// This uses make() instead of common.MakeStringSet() because the later
	// doesn't allow to create empty sets.
	functions := common.StringSet(make(map[string]struct{}, len(fnList)))
	for _, fn := range fnList {
		functions.Add(fn)
	}
	return functions, nil
}
