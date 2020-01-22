# Go test cover with multiple packages support

## Features
- Coverage profile with multiple packages (`go test` doesn't support that)

## Install
`go get github.com/pierrre/gotestcover`

## Usage
```sh
gotestcover -coverprofile=cover.out mypackage
go tool cover -html=cover.out -o=cover.html
```

Run on multiple package with:
- `package1 package2`
- `package/...`

Some `go test / build` flags are available.
