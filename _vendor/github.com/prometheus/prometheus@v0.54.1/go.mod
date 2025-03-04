module github.com/prometheus/prometheus

go 1.21.0

toolchain go1.22.5

require (
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/gogo/protobuf v1.3.2
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc
	github.com/prometheus/common v0.55.0
	golang.org/x/text v0.16.0
)

require (
	github.com/prometheus/client_model v0.6.1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace (
	k8s.io/klog => github.com/simonpasquier/klog-gokit v0.3.0
	k8s.io/klog/v2 => github.com/simonpasquier/klog-gokit/v3 v3.3.0
)

// Exclude linodego v1.0.0 as it is no longer published on github.
exclude github.com/linode/linodego v1.0.0

// Exclude grpc v1.30.0 because of breaking changes. See #7621.
exclude (
	github.com/grpc-ecosystem/grpc-gateway v1.14.7
	google.golang.org/api v0.30.0
)
