## How to add errorformat for new tool

### 1) Download errorformat CLI tool for debugging (optional)

```
go get -u github.com/reviewdog/errorformat/cmd/errorformat
```

### 2) Write errorformat for the target output
- errorformat doc: http://vimdoc.sourceforge.net/htmldoc/quickfix.html#errorformat

Note that https://github.com/reviewdog/errorformat doesn't support Vim regex, and `efm-%>` feature (currently).
Other syntax are supported.

#### Example (add errorformat for golint)

Prepare output of golint.

```
$ golint ./... > golint.in
```

##### golint.in

```
golint.new.go:3:5: exported var V should have comment or be unexported
golint.new.go:5:5: exported var NewError1 should have comment or be unexported
golint.new.go:7:1: comment on exported function F should be of the form "F ..."
golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
```

Write errorformat for it.

```
$ errorformat "%f:%l:%c: %m" < golint.in
golint.new.go|3 col 5| exported var V should have comment or be unexported
golint.new.go|5 col 5| exported var NewError1 should have comment or be unexported
golint.new.go|7 col 1| comment on exported function F should be of the form "F ..."
golint.new.go|11 col 1| comment on exported function F2 should be of the form "F2 ..."
```

### 3) Add errorformat with test
Add errorformat in `fmts/{lang}.go`, where `{lang}` is target programming language (or filetype) of the command.

#### fmts/go.go

```go
func init() {
	const lang = "go"

  // ...

	register(&Fmt{
		Name: "golint",
		Errorformat: []string{
			`%f:%l:%c: %m`,
		},
		Description: "linter for Go source code",
		URL:         "https://github.com/golang/lint",
		Language:    lang,
	})

  // ...
}
```

Required fields are self descriptive. See https://godoc.org/github.com/reviewdog/errorformat/fmts#Fmt

#### fmts/testdata/golint.in

Add input file in `fmts/testdata/{name}.in`

```
golint.new.go:3:5: exported var V should have comment or be unexported
golint.new.go:5:5: exported var NewError1 should have comment or be unexported
golint.new.go:7:1: comment on exported function F should be of the form "F ..."
golint.new.go:11:1: comment on exported function F2 should be of the form "F2 ..."
```

I also recommend to add resource code to reproduce this input file in `fmts/testdata/resources/{lang}/{name}`

#### fmts/testdata/golint.ok

Add ok file in `fmts/testdata/{name}.ok`

```
golint.new.go|3 col 5| exported var V should have comment or be unexported
golint.new.go|5 col 5| exported var NewError1 should have comment or be unexported
golint.new.go|7 col 1| comment on exported function F should be of the form "F ..."
golint.new.go|11 col 1| comment on exported function F2 should be of the form "F2 ..."
```

You can run test by `go test ./...`

### 4) go generate ./...

Run `go generate ./...` to update document file.

#### 5) Open Pull-Request!
