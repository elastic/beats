// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && cgo
// +build linux,cgo

package pkg

import (
	"debug/elf"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/coreos/pkg/dlopen"
)

/*
#include <stdio.h>
#include <stdlib.h>

#include <rpm/rpmlib.h>
#include <rpm/header.h>
#include <rpm/rpmts.h>
#include <rpm/rpmdb.h>

rpmts
my_rpmtsCreate(void *f) {
  rpmts (*rpmtsCreate)();
  rpmtsCreate = (rpmts (*)())f;

  return rpmtsCreate();
}

void
my_rpmFreeMacros(void *f) {
	void (*rpmFreeMacros)(void *);
	rpmFreeMacros = (void(*)(void*))f;
    rpmFreeMacros(NULL);
}

void
my_rpmFreeRpmrc(void *f) {
	void (*rpmFreeRpmrc)(void);
	rpmFreeRpmrc = (void (*)(void))f;
	rpmFreeRpmrc();
}

int
my_rpmReadConfigFiles(void *f) {
  int (*rpmReadConfigFiles)(const char*, const char*);
  rpmReadConfigFiles = (int (*)(const char*, const char*))f;
  return rpmReadConfigFiles(NULL, NULL);
}

rpmdbMatchIterator
my_rpmtsInitIterator(void *f, rpmts ts) {
  rpmdbMatchIterator (*rpmtsInitIterator)(const rpmts, rpmTag, const void*, size_t);
  rpmtsInitIterator = (rpmdbMatchIterator (*)(const rpmts, rpmTag, const void*, size_t))f;

  return rpmtsInitIterator(ts, RPMDBI_PACKAGES, NULL, 0);
}

Header
my_rpmdbNextIterator(void *f, rpmdbMatchIterator mi) {
  Header (*rpmdbNextIterator)(rpmdbMatchIterator);
  rpmdbNextIterator = (Header (*)(rpmdbMatchIterator))f;

  return rpmdbNextIterator(mi);
}

Header
my_headerLink(void *f, Header h) {
  Header (*headerLink)(Header);
  headerLink = (Header (*)(Header))f;

  return headerLink(h);
}

// Note: Using int32_t instead of rpmTag/rpmTagVal in definitions
// to make it work on CentOS 6.x, 7.x, and Fedora 29.
const char *
my_headerGetString(void *f, Header h, int32_t tag) {
  const char * (*headerGetString)(Header, int32_t);
  headerGetString = (const char * (*)(Header, int32_t))f;

  return headerGetString(h, tag);
}

// Note: Using int32_t instead of rpmTag/rpmTagVal in definitions
// to make it work on CentOS 6.x, 7.x, and Fedora 29.
uint64_t
my_headerGetNumber(void *f, Header h, int32_t tag) {
  uint64_t (*headerGetNumber)(Header, int32_t);
  headerGetNumber = (uint64_t (*)(Header, int32_t))f;

  return headerGetNumber(h, tag);
}

void
my_headerFree(void *f, Header h) {
  Header (*headerFree)(Header);
  headerFree = (Header (*)(Header))f;

  headerFree(h);
}

void
my_rpmdbFreeIterator(void *f, rpmdbMatchIterator mi) {
  rpmdbMatchIterator (*rpmdbFreeIterator)(rpmdbMatchIterator);
  rpmdbFreeIterator = (rpmdbMatchIterator (*)(rpmdbMatchIterator))f;

  rpmdbFreeIterator(mi);
}

void
my_rpmtsFree(void *f, rpmts ts) {
  rpmts (*rpmtsFree)(rpmts);
  rpmtsFree = (rpmts (*)(rpmts))f;

  rpmtsFree(ts);
}

// By default, librpm is going to trap various UNIX signals including SIGINT and SIGTERM
// which will prevent Beats from shutting down correctly.
//
// This disables that behavior by nullifying rpmsqEnable. We should be very dilligent in
// cleaning up in our use of librpm.
//
// More recent versions of librpm have a new function rpmsqSetInterruptSafety()
// to do this, see below.
//
// See also:
// - librpm traps signals and calls exit(1) to terminate the whole process incl. our Go code: https://github.com/rpm-software-management/rpm/blob/rpm-4.11.3-release/lib/rpmdb.c#L640
// - has caused problems for gdb before, they also nullify rpmsqEnable: https://bugzilla.redhat.com/show_bug.cgi?id=643031
// - the new rpmsqSetInterruptSafety(), unfortunately only available in librpm>=4.14.0 (CentOS 7 has 4.11.3): https://github.com/rpm-software-management/rpm/commit/56f49d7f5af7c1c8a3eb478431356195adbfdd25
extern int rpmsqEnable (int signum, void *handler);
int
rpmsqEnable (int signum, void *handler)
{
  return 0;
}

void
my_rpmsqSetInterruptSafety(void *f, int on) {
	void (*rpmsqSetInterruptSafety)(int);
	rpmsqSetInterruptSafety = (void (*)(int))f;

	rpmsqSetInterruptSafety(on);
}
*/
import "C"

// Constants in sync with /usr/include/rpm/rpmtag.h
const (
	RPMTAG_NAME        = 1000
	RPMTAG_VERSION     = 1001
	RPMTAG_RELEASE     = 1002
	RPMTAG_SUMMARY     = 1004
	RPMTAG_LICENSE     = 1014
	RPMTAG_URL         = 1020
	RPMTAG_ARCH        = 1022
	RPMTAG_SIZE        = 1009
	RPMTAG_INSTALLTIME = 1008
)

var openedLibrpm *librpm

// closeDataset performs cleanup when the dataset is closed.
func closeDataset() error {
	if openedLibrpm != nil {
		err := openedLibrpm.close()
		openedLibrpm = nil
		return err
	}

	return nil
}

type librpm struct {
	handle *dlopen.LibHandle

	rpmtsCreate             unsafe.Pointer
	rpmReadConfigFiles      unsafe.Pointer
	rpmtsInitIterator       unsafe.Pointer
	rpmdbNextIterator       unsafe.Pointer
	headerLink              unsafe.Pointer
	headerGetString         unsafe.Pointer
	headerGetNumber         unsafe.Pointer
	headerFree              unsafe.Pointer
	rpmdbFreeIterator       unsafe.Pointer
	rpmtsFree               unsafe.Pointer
	rpmsqSetInterruptSafety unsafe.Pointer
	rpmFreeRpmrc            unsafe.Pointer
	rpmFreeMacros           unsafe.Pointer
}

func (lib *librpm) close() error {
	if lib.handle != nil {
		return lib.handle.Close()
	}

	return nil
}

// getLibrpmNames determines the versions of librpm.so that are
// installed on a system.  rpm-devel rpm installs the librpm.so
// symbolic link to the correct version of librpm, but that isn't a
// required package.  rpm will install librpm.so.X, where X is the
// version number.  getLibrpmNames looks at the elf header for the rpm
// binary to determine what version of librpm.so it is linked against.
func getLibrpmNames() []string {
	rpmPaths := []string{
		"/usr/bin/rpm",
		"/bin/rpm",
	}
	libNames := []string{
		"librpm.so",
	}
	var rpmElf *elf.File
	var err error

	for _, path := range rpmPaths {
		rpmElf, err = elf.Open(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return libNames
	}

	impLibs, err := rpmElf.ImportedLibraries()
	if err != nil {
		return libNames
	}

	for _, lib := range impLibs {
		if strings.Contains(lib, "librpm.so") {
			libNames = append(libNames, lib)
		}
	}

	return libNames
}

func openLibrpm() (*librpm, error) {
	var librpm librpm
	var err error

	librpmNames := getLibrpmNames()

	librpm.handle, err = dlopen.GetHandle(librpmNames)
	if err != nil {
		return nil, fmt.Errorf("couldn't open %v: %v", librpmNames, err)
	}

	librpm.rpmtsCreate, err = librpm.handle.GetSymbolPointer("rpmtsCreate")
	if err != nil {
		return nil, err
	}

	librpm.rpmReadConfigFiles, err = librpm.handle.GetSymbolPointer("rpmReadConfigFiles")
	if err != nil {
		return nil, err
	}

	librpm.rpmtsInitIterator, err = librpm.handle.GetSymbolPointer("rpmtsInitIterator")
	if err != nil {
		return nil, err
	}

	librpm.rpmdbNextIterator, err = librpm.handle.GetSymbolPointer("rpmdbNextIterator")
	if err != nil {
		return nil, err
	}

	librpm.headerLink, err = librpm.handle.GetSymbolPointer("headerLink")
	if err != nil {
		return nil, err
	}

	librpm.headerGetString, err = librpm.handle.GetSymbolPointer("headerGetString")
	if err != nil {
		return nil, err
	}

	librpm.headerGetNumber, err = librpm.handle.GetSymbolPointer("headerGetNumber")
	if err != nil {
		return nil, err
	}

	librpm.headerFree, err = librpm.handle.GetSymbolPointer("headerFree")
	if err != nil {
		return nil, err
	}

	librpm.rpmdbFreeIterator, err = librpm.handle.GetSymbolPointer("rpmdbFreeIterator")
	if err != nil {
		return nil, err
	}

	librpm.rpmtsFree, err = librpm.handle.GetSymbolPointer("rpmtsFree")
	if err != nil {
		return nil, err
	}

	librpm.rpmFreeRpmrc, err = librpm.handle.GetSymbolPointer("rpmFreeRpmrc")
	if err != nil {
		return nil, err
	}

	// Only available in librpm>=4.13.0
	librpm.rpmsqSetInterruptSafety, err = librpm.handle.GetSymbolPointer("rpmsqSetInterruptSafety")
	// no error check

	// Only available in librpm>=4.6.0
	librpm.rpmFreeMacros, err = librpm.handle.GetSymbolPointer("rpmFreeMacros")
	// no error check

	return &librpm, nil
}

func listRPMPackages() ([]*Package, error) {
	// In newer versions, librpm is using the thread-local variable
	// `disableInterruptSafety` in rpmio/rpmsq.c to disable signal
	// traps. To make sure our settings remain in effect throughout
	// our function calls we have to lock the OS thread here, since
	// Golang can otherwise use any thread it likes for each C.* call.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if openedLibrpm == nil {
		var err error
		openedLibrpm, err = openLibrpm()
		if err != nil {
			return nil, err
		}
	}

	if openedLibrpm.rpmsqSetInterruptSafety != nil {
		C.my_rpmsqSetInterruptSafety(openedLibrpm.rpmsqSetInterruptSafety, 0)
	}

	rpmts := C.my_rpmtsCreate(openedLibrpm.rpmtsCreate)
	if rpmts == nil {
		return nil, fmt.Errorf("Failed to get rpmts")
	}
	defer C.my_rpmtsFree(openedLibrpm.rpmtsFree, rpmts)

	res := C.my_rpmReadConfigFiles(openedLibrpm.rpmReadConfigFiles)
	if int(res) != 0 {
		return nil, fmt.Errorf("Error: %d", int(res))
	}
	defer C.my_rpmFreeRpmrc(openedLibrpm.rpmFreeRpmrc)
	if openedLibrpm.rpmFreeMacros != nil {
		defer C.my_rpmFreeMacros(openedLibrpm.rpmFreeMacros)
	}

	mi := C.my_rpmtsInitIterator(openedLibrpm.rpmtsInitIterator, rpmts)
	if mi == nil {
		return nil, fmt.Errorf("Failed to get match iterator")
	}
	defer C.my_rpmdbFreeIterator(openedLibrpm.rpmdbFreeIterator, mi)

	var packages []*Package
	for header := C.my_rpmdbNextIterator(openedLibrpm.rpmdbNextIterator, mi); header != nil; header = C.my_rpmdbNextIterator(openedLibrpm.rpmdbNextIterator, mi) {

		pkg, err := packageFromHeader(header, openedLibrpm)
		if err != nil {
			return nil, err
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

func packageFromHeader(header C.Header, openedLibrpm *librpm) (*Package, error) {
	header = C.my_headerLink(openedLibrpm.headerLink, header)
	if header == nil {
		return nil, fmt.Errorf("Error calling headerLink")
	}
	defer C.my_headerFree(openedLibrpm.headerFree, header)

	pkg := Package{
		Type: "rpm",
	}

	name := C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_NAME)
	if name != nil {
		pkg.Name = C.GoString(name)
	} else {
		return nil, errors.New("Failed to get package name")
	}

	version := C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_VERSION)
	if version != nil {
		pkg.Version = C.GoString(version)
	} else {
		pkg.error = errors.New("failed to get package version")
	}

	pkg.Release = C.GoString(C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_RELEASE))
	pkg.License = C.GoString(C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_LICENSE))
	pkg.Arch = C.GoString(C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_ARCH))
	pkg.URL = C.GoString(C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_URL))
	pkg.Summary = C.GoString(C.my_headerGetString(openedLibrpm.headerGetString, header, RPMTAG_SUMMARY))

	pkg.Size = uint64(C.my_headerGetNumber(openedLibrpm.headerGetNumber, header, RPMTAG_SIZE))

	installTime := C.my_headerGetNumber(openedLibrpm.headerGetNumber, header, RPMTAG_INSTALLTIME)
	if installTime != 0 {
		pkg.InstallTime = time.Unix(int64(installTime), 0)
	}

	return &pkg, nil
}
