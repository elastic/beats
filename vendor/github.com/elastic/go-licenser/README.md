# Go Licenser [![Build Status](https://travis-ci.org/elastic/go-licenser.svg?branch=master)](https://travis-ci.org/elastic/go-licenser)

Small zero dependency license header checker for source files. The aim of this project is to provide a common
binary that can be used to ensure that code source files contain a license header. It's unlikely that this project
is useful outside of Elastic **_at the current stage_**, but the `licensing` package can be used as a building block.

## Supported Licenses

* Apache 2.0
* Elastic

## Supported languages

* Go

## Installing

```
go get -u github.com/elastic/go-licenser
```

## Usage

```
Usage: go-licenser [flags] [path]

  go-licenser walks the specified path recursiely and appends a license Header if the current
  header doesn't match the one found in the file.

Options:

  -d	skips rewriting files and returns exitcode 1 if any discrepancies are found.
  -exclude value
    	path to exclude (can be specified multiple times).
  -ext string
    	sets the file extension to scan for. (default ".go")
  -license string
    	sets the license type to check: ASL2, Elastic (default "ASL2")
  -version
    	prints out the binary version.
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

