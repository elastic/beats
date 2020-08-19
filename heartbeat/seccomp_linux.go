package main

import (
	"github.com/elastic/beats/v7/libbeat/common/seccomp"
	"runtime"
)

func init() {
	switch runtime.GOARCH {
	case "amd64", "386":
		if err := seccomp.ModifyDefaultPolicy(seccomp.AddSyscall, "execve"); err != nil {
			panic(err)
		}
	case "arm":
		// TODO: Figure out how to allow execve here
	}
}
