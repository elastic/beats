// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package build gathers information about Go packages.
//
// Go Path
//
// The Go path is a list of directory trees containing Go source code.
// It is consulted to resolve imports that cannot be found in the standard
// Go tree. The default path is the value of the GOPATH environment
// variable, interpreted as a path list appropriate to the operating system
// (on Unix, the variable is a colon-separated string;
// on Windows, a semicolon-separated string;
// on Plan 9, a list).
//
// Each directory listed in the Go path must have a prescribed structure:
//
// The src/ directory holds source code. The path below 'src' determines
// the import path or executable name.
//
// The pkg/ directory holds installed package objects.
// As in the Go tree, each target operating system and
// architecture pair has its own subdirectory of pkg
// (pkg/GOOS_GOARCH).
//
// If DIR is a directory listed in the Go path, a package with
// source in DIR/src/foo/bar can be imported as "foo/bar" and
// has its compiled form installed to "DIR/pkg/GOOS_GOARCH/foo/bar.a"
// (or, for gccgo, "DIR/pkg/gccgo/foo/libbar.a").
//
// The bin/ directory holds compiled commands.
// Each command is named for its source directory, but only
// using the final element, not the entire path. That is, the
// command with source in DIR/src/foo/quux is installed into
// DIR/bin/quux, not DIR/bin/foo/quux. The foo/ is stripped
// so that you can add DIR/bin to your PATH to get at the
// installed commands.
//
// Here's an example directory layout:
//
//	GOPATH=/home/user/gocode
//
//	/home/user/gocode/
//	    src/
//	        foo/
//	            bar/               (go code in package bar)
//	                x.go
//	            quux/              (go code in package main)
//	                y.go
//	    bin/
//	        quux                   (installed command)
//	    pkg/
//	        linux_amd64/
//	            foo/
//	                bar.a          (installed package object)
//
// Build Constraints
//
// A build constraint, also known as a build tag, is a line comment that begins
//
//	// +build
//
// that lists the conditions under which a file should be included in the package.
// Constraints may appear in any kind of source file (not just Go), but
// they must appear near the top of the file, preceded
// only by blank lines and other line comments. These rules mean that in Go
// files a build constraint must appear before the package clause.
//
// To distinguish build constraints from package documentation, a series of
// build constraints must be followed by a blank line.
//
// A build constraint is evaluated as the OR of space-separated options;
// each option evaluates as the AND of its comma-separated terms;
// and each term is an alphanumeric word or, preceded by !, its negation.
// That is, the build constraint:
//
//	// +build linux,386 darwin,!cgo
//
// corresponds to the boolean formula:
//
//	(linux AND 386) OR (darwin AND (NOT cgo))
//
// A file may have multiple build constraints. The overall constraint is the AND
// of the individual constraints. That is, the build constraints:
//
//	// +build linux darwin
//	// +build 386
//
// corresponds to the boolean formula:
//
//	(linux OR darwin) AND 386
//
// During a particular build, the following words are satisfied:
//
//	- the target operating system, as spelled by runtime.GOOS
//	- the target architecture, as spelled by runtime.GOARCH
//	- the compiler being used, either "gc" or "gccgo"
//	- "cgo", if ctxt.CgoEnabled is true
//	- "go1.1", from Go version 1.1 onward
//	- "go1.2", from Go version 1.2 onward
//	- "go1.3", from Go version 1.3 onward
//	- "go1.4", from Go version 1.4 onward
//	- "go1.5", from Go version 1.5 onward
//	- "go1.6", from Go version 1.6 onward
//	- "go1.7", from Go version 1.7 onward
//	- "go1.8", from Go version 1.8 onward
//	- "go1.9", from Go version 1.9 onward
//	- any additional words listed in ctxt.BuildTags
//
// If a file's name, after stripping the extension and a possible _test suffix,
// matches any of the following patterns:
//	*_GOOS
// 	*_GOARCH
// 	*_GOOS_GOARCH
// (example: source_windows_amd64.go) where GOOS and GOARCH represent
// any known operating system and architecture values respectively, then
// the file is considered to have an implicit build constraint requiring
// those terms (in addition to any explicit constraints in the file).
//
// To keep a file from being considered for the build:
//
//	// +build ignore
//
// (any other unsatisfied word will work as well, but ``ignore'' is conventional.)
//
// To build a file only when using cgo, and only on Linux and OS X:
//
//	// +build linux,cgo darwin,cgo
//
// Such a file is usually paired with another file implementing the
// default functionality for other systems, which in this case would
// carry the constraint:
//
//	// +build !linux,!darwin !cgo
//
// Naming a file dns_windows.go will cause it to be included only when
// building the package for Windows; similarly, math_386.s will be included
// only when building the package for 32-bit x86.
//
// Using GOOS=android matches build tags and files as for GOOS=linux
// in addition to android tags and files.
//
// Binary-Only Packages
//
// It is possible to distribute packages in binary form without including the
// source code used for compiling the package. To do this, the package must
// be distributed with a source file not excluded by build constraints and
// containing a "//go:binary-only-package" comment.
// Like a build constraint, this comment must appear near the top of the file,
// preceded only by blank lines and other line comments and with a blank line
// following the comment, to separate it from the package documentation.
// Unlike build constraints, this comment is only recognized in non-test
// Go source files.
//
// The minimal source code for a binary-only package is therefore:
//
//	//go:binary-only-package
//
//	package mypkg
//
// The source code may include additional Go code. That code is never compiled
// but will be processed by tools like godoc and might be useful as end-user
// documentation.
//
package build
