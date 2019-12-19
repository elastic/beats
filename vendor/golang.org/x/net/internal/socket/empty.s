// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

<<<<<<< HEAD:vendor/golang.org/x/net/internal/socket/empty.s
// +build darwin,go1.12

// This exists solely so we can linkname in symbols from syscall.
=======
// +build !linux,arm64

package cpu

func doinit() {}
>>>>>>> update golang.org/x/sys:vendor/golang.org/x/sys/cpu/cpu_other_arm64.go
