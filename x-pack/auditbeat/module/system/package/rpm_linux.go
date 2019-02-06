// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,cgo

package pkg

import (
	"fmt"
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

int
my_headerGetEntry(void *f, Header h, rpm_tag_t tag, char **p) {
  int (*headerGetEntry)(Header, rpm_tag_t, rpm_tagtype_t*, rpm_data_t*, rpm_count_t*);
  headerGetEntry = (int (*)(Header, rpm_tag_t, rpm_tagtype_t*, rpm_data_t*, rpm_count_t*))f;

  return headerGetEntry(h, tag, NULL, (void**)p, NULL);
}

int
my_headerGetEntryInt(void *f, Header h, rpm_tag_t tag, int **p) {
  int (*headerGetEntry)(Header, rpm_tag_t, rpm_tagtype_t*, rpm_data_t*, rpm_count_t*);
  headerGetEntry = (int (*)(Header, rpm_tag_t, rpm_tagtype_t*, rpm_data_t*, rpm_count_t*))f;

  return headerGetEntry(h, tag, NULL, (void**)p, NULL);
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
}*/
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

type cFunctions struct {
	rpmtsCreate        unsafe.Pointer
	rpmReadConfigFiles unsafe.Pointer
	rpmtsInitIterator  unsafe.Pointer
	rpmdbNextIterator  unsafe.Pointer
	headerLink         unsafe.Pointer
	headerGetEntry     unsafe.Pointer
	headerFree         unsafe.Pointer
	rpmdbFreeIterator  unsafe.Pointer
	rpmtsFree          unsafe.Pointer
}

var cFun *cFunctions

func dlopenCFunctions() (*cFunctions, error) {
	var librpmNames = []string{
		"/usr/lib64/librpm.so",
	}
	var cFun cFunctions

	librpm, err := dlopen.GetHandle(librpmNames)
	if err != nil {
		return nil, err
	}

	cFun.rpmtsCreate, err = librpm.GetSymbolPointer("rpmtsCreate")
	if err != nil {
		return nil, err
	}

	cFun.rpmReadConfigFiles, err = librpm.GetSymbolPointer("rpmReadConfigFiles")
	if err != nil {
		return nil, err
	}

	cFun.rpmtsInitIterator, err = librpm.GetSymbolPointer("rpmtsInitIterator")
	if err != nil {
		return nil, err
	}

	cFun.rpmdbNextIterator, err = librpm.GetSymbolPointer("rpmdbNextIterator")
	if err != nil {
		return nil, err
	}

	cFun.headerLink, err = librpm.GetSymbolPointer("headerLink")
	if err != nil {
		return nil, err
	}

	cFun.headerGetEntry, err = librpm.GetSymbolPointer("headerGetEntry")
	if err != nil {
		return nil, err
	}

	cFun.headerFree, err = librpm.GetSymbolPointer("headerFree")
	if err != nil {
		return nil, err
	}

	cFun.rpmdbFreeIterator, err = librpm.GetSymbolPointer("rpmdbFreeIterator")
	if err != nil {
		return nil, err
	}

	cFun.rpmtsFree, err = librpm.GetSymbolPointer("rpmtsFree")
	if err != nil {
		return nil, err
	}

	return &cFun, nil
}

func listRPMPackages() ([]*Package, error) {
	if cFun == nil {
		var err error
		cFun, err = dlopenCFunctions()
		if err != nil {
			return nil, err
		}
	}

	rpmts := C.my_rpmtsCreate(cFun.rpmtsCreate)
	if rpmts == nil {
		return nil, fmt.Errorf("Failed to get rpmts")
	}
	defer C.my_rpmtsFree(cFun.rpmtsFree, rpmts)
	res := C.my_rpmReadConfigFiles(cFun.rpmReadConfigFiles)
	if int(res) != 0 {
		return nil, fmt.Errorf("Error: %d", int(res))
	}

	mi := C.my_rpmtsInitIterator(cFun.rpmtsInitIterator, rpmts)
	if mi == nil {
		return nil, fmt.Errorf("Failed to get match iterator")
	}
	defer C.my_rpmdbFreeIterator(cFun.rpmdbFreeIterator, mi)

	var packages []*Package
	for header := C.my_rpmdbNextIterator(cFun.rpmdbNextIterator, mi); header != nil; header = C.my_rpmdbNextIterator(cFun.rpmdbNextIterator, mi) {

		pkg, err := packageFromHeader(header, cFun)
		if err != nil {
			return nil, err
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

func packageFromHeader(header C.Header, cFun *cFunctions) (*Package, error) {

	header = C.my_headerLink(cFun.headerLink, header)
	if header == nil {
		return nil, fmt.Errorf("Error calling headerLink")
	}
	defer C.my_headerFree(cFun.headerFree, header)

	pkg := Package{}

	var name *C.char
	res := C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_NAME, &name)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(name): %d", res)
	}
	pkg.Name = C.GoString(name)

	var version *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_VERSION, &version)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(version): %d", res)
	}
	pkg.Version = C.GoString(version)

	var release *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_RELEASE, &release)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(release): %d", res)
	}
	pkg.Release = C.GoString(release)

	var license *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_LICENSE, &license)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(license): %d", res)
	}
	pkg.License = C.GoString(license)

	var arch *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_ARCH, &arch)
	if res == 1 { // not always successful
		pkg.Arch = C.GoString(arch)
	}

	var url *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_URL, &url)
	if res == 1 { // not always successful
		pkg.URL = C.GoString(url)
	}

	var summary *C.char
	res = C.my_headerGetEntry(cFun.headerGetEntry, header, RPMTAG_SUMMARY, &summary)
	if res == 1 { // not always successful
		pkg.Summary = C.GoString(summary)
	}

	var size *C.int
	res = C.my_headerGetEntryInt(cFun.headerGetEntry, header, RPMTAG_SIZE, &size)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(size): %d", res)
	}
	pkg.Size = uint64(*size)

	var installTime *C.int
	res = C.my_headerGetEntryInt(cFun.headerGetEntry, header, RPMTAG_INSTALLTIME, &installTime)
	if res != 1 {
		return nil, fmt.Errorf("Failed to call headerGetEntry(installTime): %d", res)
	}
	pkg.InstallTime = time.Unix(int64(*installTime), 0)

	return &pkg, nil
}
