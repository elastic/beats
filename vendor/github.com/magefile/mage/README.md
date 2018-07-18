<p align="center"><img src="https://user-images.githubusercontent.com/3185864/32058716-5ee9b512-ba38-11e7-978a-287eb2a62743.png"/></p>

## About [![Build Status](https://travis-ci.org/magefile/mage.svg?branch=master)](https://travis-ci.org/magefile/mage)

Mage is a make/rake-like build tool using Go.  You write plain-old go functions,
and Mage automatically uses them as Makefile-like runnable targets.

## Installation

Mage has no dependencies outside the Go standard library, and builds with Go 1.7
and above (possibly even lower versions, but they're not regularly tested). 

Install mage by running 

```
go get -u -d github.com/magefile/mage
cd $GOPATH/src/github.com/magefile/mage
go run bootstrap.go
```

This will download the code into your GOPATH, and then run the bootstrap script
to build mage with version infomation embedded in it.  A normal `go get`
(without -d) will build the binary correctly, but no version info will be
embedded.  If you've done this, no worries, just go to
$GOPATH/src/github.com/magefile/mage and run `mage install` or `go run
bootstrap.go` and a new binary will be created with the correct version
information.

The mage binary will be created in your $GOPATH/bin directory.

You may also install a binary release from our
[releases](https://github.com/magefile/mage/releases) page. 

## Demo

[![Mage Demo](https://img.youtube.com/vi/GOqbD0lF-iA/maxresdefault.jpg)](https://www.youtube.com/watch?v=GOqbD0lF-iA)

## Discussion

Join the `#mage` channel on [gophers slack](https://gophers.slack.com/messages/general/) for discussion of usage, development, etc.

# Documentation

see [magefile.org](https://magefile.org) for full docs

see [godoc.org/github.com/magefile/mage/mage](https://godoc.org/github.com/magefile/mage/mage) for how to use mage as a library.

# Why?

Makefiles are hard to read and hard to write.  Mostly because makefiles are essentially fancy bash scripts with significant white space and additional make-related syntax.

Mage lets you have multiple magefiles, name your magefiles whatever you
want, and they're easy to customize for multiple operating systems.  Mage has no
dependencies (aside from go) and runs just fine on all major operating systems, whereas make generally uses bash which is not well supported on Windows.
Go is superior to bash for any non-trivial task involving branching, looping, anything that's not just straight line execution of commands.  And if your project is written in Go, why introduce another
language as idiosyncratic as bash?  Why not use the language your contributors
are already comfortable with?

# TODO

* File conversion tasks
