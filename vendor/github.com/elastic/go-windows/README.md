# go-windows

[![Build Status](http://img.shields.io/travis/elastic/go-windows.svg?style=flat-square)][travis]
[![Build status](https://ci.appveyor.com/api/projects/status/remqhuw0jjguygc3/branch/master?svg=true)][appveyor]
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[travis]:   http://travis-ci.org/elastic/go-windows
[appveyor]: https://ci.appveyor.com/project/elastic-beats/go-windows/branch/master
[godocs]:   http://godoc.org/github.com/elastic/go-windows

go-windows is a library for Go (golang) that provides wrappers to various
Windows APIs that are not covered by the stdlib or by
[golang.org/x/sys/windows](https://godoc.org/golang.org/x/sys/windows).

Goals / Features

- Does not use cgo.
- Provide abstractions to make using the APIs easier.
