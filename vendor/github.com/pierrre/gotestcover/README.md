# Go test cover with multiple packages support

## Deprecated
Just use this script instead:
```
echo 'mode: atomic' > coverage.txt && go list ./... | xargs -n1 -I{} sh -c 'go test -covermode=atomic -coverprofile=coverage.tmp {} && tail -n +2 coverage.tmp >> coverage.txt' && rm coverage.tmp
```
It's easier to customize, gives you better control, and doesn't require to download a third-party tool.

The repository will remain, but I will not update it anymore.
If you want to add new features, create a new fork.

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
