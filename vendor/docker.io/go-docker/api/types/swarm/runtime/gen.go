//go:generate protoc -I . --gogofast_out=import_path=docker.io/go-docker/api/types/swarm/runtime:. plugin.proto

package runtime
