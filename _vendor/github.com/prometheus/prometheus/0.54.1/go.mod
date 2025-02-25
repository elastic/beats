module github.com/prometheus/prometheus

go 1.22.7

toolchain go1.23.4

require (
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc
	github.com/prometheus/common v0.62.0
	golang.org/x/text v0.21.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	google.golang.org/protobuf v1.36.4 // indirect
)

// Exclude linodego v1.0.0 as it is no longer published on github.
exclude github.com/linode/linodego v1.0.0

// Exclude grpc v1.30.0 because of breaking changes. See #7621.
exclude (
	github.com/grpc-ecosystem/grpc-gateway v1.14.7
	google.golang.org/api v0.30.0
)

// Pin until https://github.com/fsnotify/fsnotify/issues/656 is resolved.
replace github.com/fsnotify/fsnotify v1.8.0 => github.com/fsnotify/fsnotify v1.7.0
