package system

import "path/filepath"

// Resolver is an interface for HostFS resolvers. This is meant to be generic and (hopefully) future-proof way of dealing with a user-supplied root filesystem path.
// A resolver-style function serves two ends:
// 1) if we attempt to stop consumers from merely "saving off" a string, the underlying implementation can update hostfs values and pass the new paths along to consumers
// 2) This stops different bits of code from making different assumptions about what's in hostfs and otherwise treating the concept differently. It's easy to mix up "hostfs" and "procfs" and "sysfs" as concepts.
// A single resolver forces this logic to be a little more centralized.
type Resolver interface {
	// ResolveHostFS Resolves a path based on a user-set HostFS flag, in cases where a user wants to monitor an alternate filesystem root
	// If no user root has been set, it will return the input string
	ResolveHostFS(string) string
	// IsSet returns true if the user has set an alternate filesystem root
	IsSet() bool
}

// TestingResolver is a bare implementation of the resolver, for system tests that need a Resolver object or a test path for input files.
type TestingResolver struct {
	path  string
	isSet bool
}

// NewTestResolver returns a new resolver for internal testing, or other uses outside metricbeat modules.
func NewTestResolver(path string) TestingResolver {
	if path == "" || path == "/" {
		return TestingResolver{path: "/", isSet: false}
	}

	return TestingResolver{path: path, isSet: true}
}

func (t TestingResolver) ResolveHostFS(path string) string {
	return filepath.Join(t.path, path)
}

func (t TestingResolver) IsSet() bool {
	return t.isSet
}
