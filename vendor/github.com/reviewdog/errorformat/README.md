## errorformat - Vim 'errorformat' implementation in Go

[![Travis Build Status][travis-badge]](https://travis-ci.org/reviewdog/errorformat)
[![codecov][codecov-badge]](https://codecov.io/gh/haya14busa/reviewdog)
[![Go Report Card](https://goreportcard.com/badge/github.com/reviewdog/errorformat)](https://goreportcard.com/report/github.com/reviewdog/errorformat)
[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/reviewdog/errorformat?status.svg)](https://godoc.org/github.com/reviewdog/errorformat)

errorformat is Vim's quickfix [errorformat](https://vim-jp.org/vimdoc-en/quickfix.html#error-file-format) implementation in golang.

errorformat provides default errorformats for major tools.
You can see defined errorformats [here](https://godoc.org/github.com/reviewdog/errorformat/fmts).
Also, it's easy to [add new errorformat](fmts/README.md) in a similar way to Vim's errorformat.

Note that it's highly compatible with Vim implementation, but it doesn't support Vim regex.

### :arrow_forward: Playground :arrow_forward:

Try errorformat on [the Playground](https://reviewdog.github.io/errorformat-playground/)!

The playground uses [gopherjs](https://github.com/gopherjs/gopherjs) to try
errorformat implementation in Go with JavaScript :sparkles:

### Usage

```go
import "github.com/reviewdog/errorformat"
```

### Example 

#### Code:

```go
in := `
golint.new.go:3:5: exported var V should have comment or be unexported
golint.new.go:5:5: exported var NewError1 should have comment or be unexported
golint.new.go:7:1: comment on exported function F should be of the form "F ..."
golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
`
efm, _ := errorformat.NewErrorformat([]string{`%f:%l:%c: %m`, `%-G%.%#`})
s := efm.NewScanner(strings.NewReader(in))
for s.Scan() {
    fmt.Println(s.Entry())
}
```

#### Output:

```
golint.new.go|3 col 5| exported var V should have comment or be unexported
golint.new.go|5 col 5| exported var NewError1 should have comment or be unexported
golint.new.go|7 col 1| comment on exported function F should be of the form "F ..."
golint.new.go|11 col 1| comment on exported function F2 should be of the form "F2 ..."
```

### CLI tool

#### Installation

```
go get -u github.com/reviewdog/errorformat/cmd/errorformat
```

#### Usage

```
Usage: errorformat [flags] [errorformat ...]

errorformat reads compiler/linter/static analyzer result from STDIN, formats
them by given 'errorformat' (90% compatible with Vim's errorformat. :h
errorformat), and outputs formated result to STDOUT.

Example:
        $ echo '/path/to/file:14:28: error message\nfile2:3:4: msg' | errorformat "%f:%l:%c: %m"
        /path/to/file|14 col 28| error message
        file2|3 col 4| msg

        $ golint ./... | errorformat -name=golint

The -f flag specifies an alternate format for the entry, using the
syntax of package template.  The default output is equivalent to -f
'{{.String}}'. The struct being passed to the template is:

        type Entry struct {
                // name of a file
                Filename string
                // line number
                Lnum int
                // column number (first column is 1)
                Col int
                // true: "col" is visual column
                // false: "col" is byte index
                Vcol bool
                // error number
                Nr int
                // search pattern used to locate the error
                Pattern string
                // description of the error
                Text string
                // type of the error, 'E', '1', etc.
                Type rune
                // true: recognized error message
                Valid bool

                // Original error lines (often one line. more than one line for multi-line
                // errorformat. :h errorformat-multi-line)
                Lines []string
        }

Flags:
  -f string
        format template for -w=template (default "{{.String}}")
  -list
        list defined errorformats
  -name string
        defined errorformat name
  -w string
        writer format (template|checkstyle) (default "template")
```

```
$ cat testdata/sbt.in
[warn] /path/to/F1.scala:203: local val in method f is never used: (warning smaple 3)
[warn]         val x = 1
[warn]             ^
[warn] /path/to/F1.scala:204: local val in method f is never used: (warning smaple 2)
[warn]   val x = 2
[warn]       ^
[error] /path/to/F2.scala:1093: error: value ++ is not a member of Int
[error]     val x = 1 ++ 2
[error]               ^
[warn] /path/to/dir/F3.scala:83: local val in method f is never used
[warn]         val x = 4
[warn]             ^
[error] /path/to/dir/F3.scala:84: error: value ++ is not a member of Int
[error]         val x = 5 ++ 2
[error]                   ^
[warn] /path/to/dir/F3.scala:86: local val in method f is never used
[warn]         val x = 6
[warn]             ^
$ errorformat "%E[%t%.%+] %f:%l: error: %m" "%A[%t%.%+] %f:%l: %m" "%Z[%.%+] %p^" "%C[%.%+] %.%#" "%-G%.%#" < testdata/sbt.in
/path/to/F1.scala|203 col 13 warning| local val in method f is never used: (warning smaple 3)
/path/to/F1.scala|204 col 7 warning| local val in method f is never used: (warning smaple 2)
/path/to/F2.scala|1093 col 15 error| value &#43;&#43; is not a member of Int
/path/to/dir/F3.scala|83 col 13 warning| local val in method f is never used
/path/to/dir/F3.scala|84 col 19 error| value &#43;&#43; is not a member of Int
/path/to/dir/F3.scala|86 col 13 warning| local val in method f is never used
$ cat fmts/testdata/sbt.in | errorformat -name=sbt -w=checkstyle
<?xml version="1.0" encoding="UTF-8"?>
<checkstyle version="1.0">
  <file name="/home/haya14busa/src/github.com/reviewdog/errorformat/fmts/testdata/resources/scala/scalac.scala">
    <error column="3" line="6" message="missing argument list for method error in object Predef" severity="error"></error>
    <error column="15" line="4" message="private val in object F is never used" severity="warning"></error>
    <error column="15" line="5" message="private method in object F is never used" severity="warning"></error>
  </file>
</checkstyle>
```

### Use cases of 'errorformat' outside Vim
- [reviewdog/reviewdog - A code review dog who keeps your codebase healthy](https://github.com/haya14busa/reviewdog)
- [mattn/efm-langserver - General purpose Language Server](https://github.com/mattn/efm-langserver)

## :bird: Author
haya14busa (https://github.com/haya14busa)

<!-- From https://github.com/zchee/template -->
[travis-badge]: https://img.shields.io/travis/reviewdog/errorformat.svg?style=flat-square&label=%20Travis%20CI&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgdmlld0JveD0iNSA0IDI0IDI0Ij48cGF0aCBmaWxsPSIjREREIiBkPSJNMTEuMzkyKzkuMzc0aDQuMDk2djEzLjEyaC0xLjUzNnYyLjI0aDYuMDgwdi0yLjQ5NmgtMS45MnYtMTMuMDU2aDQuMzUydjEuOTJoMS45ODR2LTMuOTA0aC0xNS4yOTZ2My45MDRoMi4yNHpNMjkuMjYzKzIuNzE4aC0yNC44NDhjLTAuNDMzKzAtMC44MzIrMC4zMjEtMC44MzIrMC43NDl2MjQuODQ1YzArMC40MjgrMC4zOTgrMC43NzQrMC44MzIrMC43NzRoMjQuODQ4YzAuNDMzKzArMC43NTMtMC4zNDcrMC43NTMtMC43NzR2LTI0Ljg0NWMwLTAuNDI4LTAuMzE5LTAuNzQ5LTAuNzUzLTAuNzQ5ek0yNS43MjgrMTIuMzgyaC00LjU0NHYtMS45MmgtMS43OTJ2MTAuNDk2aDEuOTJ2NS4wNTZoLTguNjR2LTQuOGgxLjUzNnYtMTAuNTZoLTEuNTM2djEuNzI4aC00Ljh2LTYuNDY0aDE3Ljg1NnY2LjQ2NHoiLz48L3N2Zz4=
[circleci-badge]: https://img.shields.io/circleci/project/github/reviewdog/errorformat.svg?style=flat-square&label=%20%20CircleCI&logoWidth=16&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgdmlld0JveD0iMCAwIDIwMCAyMDAiPjxwYXRoIGZpbGw9IiNEREQiIGQ9Ik03NC43IDEwMGMwLTEzLjIgMTAuNy0yMy44IDIzLjgtMjMuOCAxMy4xIDAgMjMuOCAxMC43IDIzLjggMjMuOCAwIDEzLjEtMTAuNyAyMy44LTIzLjggMjMuOC0xMy4xIDAtMjMuOC0xMC43LTIzLjgtMjMuOHpNOTguNSAwQzUxLjggMCAxMi43IDMyIDEuNiA3NS4yYy0uMS4zLS4xLjYtLjEgMSAwIDIuNiAyLjEgNC44IDQuOCA0LjhoNDAuM2MxLjkgMCAzLjYtMS4xIDQuMy0yLjggOC4zLTE4IDI2LjUtMzAuNiA0Ny42LTMwLjYgMjguOSAwIDUyLjQgMjMuNSA1Mi40IDUyLjRzLTIzLjUgNTIuNC01Mi40IDUyLjRjLTIxLjEgMC0zOS4zLTEyLjUtNDcuNi0zMC42LS44LTEuNi0yLjQtMi44LTQuMy0yLjhINi4zYy0yLjYgMC00LjggMi4xLTQuOCA0LjggMCAuMy4xLjYuMSAxQzEyLjYgMTY4IDUxLjggMjAwIDk4LjUgMjAwYzU1LjIgMCAxMDAtNDQuOCAxMDAtMTAwUzE1My43IDAgOTguNSAweiIvPjwvc3ZnPg%3D%3D
[godoc-badge]: https://img.shields.io/badge/godoc-reference-4F73B3.svg?style=flat-square&label=%20godoc.org
[codecov-badge]: https://img.shields.io/codecov/c/github/reviewdog/errorformat.svg?style=flat-square&label=%20%20Codecov%2Eio&logo=data%3Aimage%2Fsvg%2Bxml%3Bcharset%3Dutf-8%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0MCIgaGVpZ2h0PSI0MCIgdmlld0JveD0iMCAwIDI1NiAyODEiPjxwYXRoIGZpbGw9IiNFRUUiIGQ9Ik0yMTguNTUxIDM3LjQxOUMxOTQuNDE2IDEzLjI4OSAxNjIuMzMgMCAxMjguMDk3IDAgNTcuNTM3LjA0Ny4wOTEgNTcuNTI3LjA0IDEyOC4xMjFMMCAxNDkuODEzbDE2Ljg1OS0xMS40OWMxMS40NjgtNy44MTQgMjQuNzUtMTEuOTQ0IDM4LjQxNy0xMS45NDQgNC4wNzkgMCA4LjE5OC4zNzMgMTIuMjQgMS4xMSAxMi43NDIgMi4zMiAyNC4xNjUgOC4wODkgMzMuNDE0IDE2Ljc1OCAyLjEyLTQuNjcgNC42MTQtOS4yMDkgNy41Ni0xMy41MzZhODguMDgxIDg4LjA4MSAwIDAgMSAzLjgwNS01LjE1Yy0xMS42NTItOS44NC0yNS42NDktMTYuNDYzLTQwLjkyNi0xOS4yNDVhOTAuMzUgOTAuMzUgMCAwIDAtMTYuMTItMS40NTkgODguMzc3IDg4LjM3NyAwIDAgMC0zMi4yOSA2LjA3YzguMzYtNTEuMjIyIDUyLjg1LTg5LjM3IDEwNS4yMy04OS40MDggMjguMzkyIDAgNTUuMDc4IDExLjA1MyA3NS4xNDkgMzEuMTE3IDE2LjAxMSAxNi4wMSAyNi4yNTQgMzYuMDMzIDI5Ljc4OCA1OC4xMTctMTAuMzI5LTQuMDM1LTIxLjIxMi02LjEtMzIuNDAzLTYuMTQ0bC0xLjU2OC0uMDA3YTkwLjk1NyA5MC45NTcgMCAwIDAtMy40MDEuMTExYy0xLjk1NS4xLTMuODk4LjI3Ny01LjgyMS41LS41NzQuMDYzLTEuMTM5LjE1My0xLjcwNy4yMzEtMS4zNzguMTg2LTIuNzUuMzk1LTQuMTA5LjYzOS0uNjAzLjExLTEuMjAzLjIzMS0xLjguMzUxYTkwLjUxNyA5MC41MTcgMCAwIDAtNC4xMTQuOTM3Yy0uNDkyLjEyNi0uOTgzLjI0My0xLjQ3LjM3NGE5MC4xODMgOTAuMTgzIDAgMCAwLTUuMDkgMS41MzhjLS4xLjAzNS0uMjA0LjA2My0uMzA0LjA5NmE4Ny41MzIgODcuNTMyIDAgMCAwLTExLjA1NyA0LjY0OWMtLjA5Ny4wNS0uMTkzLjEwMS0uMjkzLjE1MWE4Ni43IDg2LjcgMCAwIDAtNC45MTIgMi43MDFsLS4zOTguMjM4YTg2LjA5IDg2LjA5IDAgMCAwLTIyLjMwMiAxOS4yNTNjLS4yNjIuMzE4LS41MjQuNjM1LS43ODQuOTU4LTEuMzc2IDEuNzI1LTIuNzE4IDMuNDktMy45NzYgNS4zMzZhOTEuNDEyIDkxLjQxMiAwIDAgMC0zLjY3MiA1LjkxMyA5MC4yMzUgOTAuMjM1IDAgMCAwLTIuNDk2IDQuNjM4Yy0uMDQ0LjA5LS4wODkuMTc1LS4xMzMuMjY1YTg4Ljc4NiA4OC43ODYgMCAwIDAtNC42MzcgMTEuMjcybC0uMDAyLjAwOXYuMDA0YTg4LjAwNiA4OC4wMDYgMCAwIDAtNC41MDkgMjkuMzEzYy4wMDUuMzk3LjAwNS43OTQuMDE5IDEuMTkyLjAyMS43NzcuMDYgMS41NTcuMTA0IDIuMzM4YTk4LjY2IDk4LjY2IDAgMCAwIC4yODkgMy44MzRjLjA3OC44MDQuMTc0IDEuNjA2LjI3NSAyLjQxLjA2My41MTIuMTE5IDEuMDI2LjE5NSAxLjUzNGE5MC4xMSA5MC4xMSAwIDAgMCAuNjU4IDQuMDFjNC4zMzkgMjIuOTM4IDE3LjI2MSA0Mi45MzcgMzYuMzkgNTYuMzE2bDIuNDQ2IDEuNTY0LjAyLS4wNDhhODguNTcyIDg4LjU3MiAwIDAgMCAzNi4yMzIgMTMuNDVsMS43NDYuMjM2IDEyLjk3NC0yMC44MjItNC42NjQtLjEyN2MtMzUuODk4LS45ODUtNjUuMS0zMS4wMDMtNjUuMS02Ni45MTcgMC0zNS4zNDggMjcuNjI0LTY0LjcwMiA2Mi44NzYtNjYuODI5bDIuMjMtLjA4NWMxNC4yOTItLjM2MiAyOC4zNzIgMy44NTkgNDAuMzI1IDExLjk5N2wxNi43ODEgMTEuNDIxLjAzNi0yMS41OGMuMDI3LTM0LjIxOS0xMy4yNzItNjYuMzc5LTM3LjQ0OS05MC41NTQiLz48L3N2Zz4=
