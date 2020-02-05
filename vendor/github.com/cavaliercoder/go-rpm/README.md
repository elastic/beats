# go-rpm [![GoDoc](https://godoc.org/github.com/cavaliercoder/go-rpm?status.svg)](https://godoc.org/github.com/cavaliercoder/go-rpm) [![Build Status](https://travis-ci.org/cavaliercoder/go-rpm.svg?branch=master)](https://travis-ci.org/cavaliercoder/go-rpm) [![Go Report Card](https://goreportcard.com/badge/github.com/cavaliercoder/go-rpm)](https://goreportcard.com/report/github.com/cavaliercoder/go-rpm)

A native implementation of the RPM file specification in Go.

	$ go get github.com/cavaliercoder/go-rpm


The go-rpm package aims to enable cross-platform tooling for yum/dnf/rpm
written in Go (E.g. [y10k](https://github.com/cavaliercoder/y10k)).

Initial goals include like-for-like implementation of existing rpm ecosystem
features such as:

* Reading of modern and legacy rpm package file formats
* Reading, creating and updating modern and legacy yum repository metadata
* Reading of the rpm database

```go
package main

import (
	"fmt"
	"github.com/cavaliercoder/go-rpm"
)

func main() {
	p, err := rpm.OpenPackageFile("golang-1.6.3-2.el7.rpm")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Loaded package: %v - %s\n", p, p.Summary())

	// Output: golang-0:1.6.3-2.el7.x86_64 - The Go Programming Language
}
```

## Tools

This package also includes two tools `rpmdump` and `rpminfo`.

The code for both tools demonstrates some use-cases of this package. They are
both also useful for interrogating RPM packages on any platform.

```
$ rpminfo golang-1.6.3-2.el7.x86_64.rpm
Name        : golang
Version     : 1.6.3
Release     : 2.el7
Architecture: x86_64
Group       : Unspecified
Size        : 11809071
License     : BSD and Public Domain
Signature   : RSA/SHA256, Sun Nov 20 18:01:16 2016, Key ID 24c6a8a7f4a80eb5
Source RPM  : golang-1.6.3-2.el7.src.rpm
Build Date  : Tue Nov 15 12:20:30 2016
Build Host  : c1bm.rdu2.centos.org
Packager    : CentOS BuildSystem <http://bugs.centos.org>
Vendor      : CentOS
URL         : http://golang.org/
Summary     : The Go Programming Language
Description :
The Go Programming Language.
```
