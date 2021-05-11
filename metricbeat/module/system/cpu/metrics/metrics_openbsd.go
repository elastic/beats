// +build openbsd

package metrics

/*
#include <sys/param.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <sys/sched.h>
#include <sys/swap.h>
#include <stdlib.h>
#include <unistd.h>
*/
import "C"

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

func Get(_ string) error {
	load := [C.CPUSTATES]C.long{C.CP_USER, C.CP_NICE, C.CP_SYS, C.CP_INTR, C.CP_IDLE}

	mib := [2]int32{C.CTL_KERN, C.KERN_CPTIME}
	n := uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}

	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&load)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return nil
	}
	self := cpu{}

	self.User = uint64(load[0])
	self.Nice = uint64(load[1])
	self.Sys = uint64(load[2])
	self.Irq = uint64(load[3])
	self.Idle = uint64(load[4])

	return nil
}
