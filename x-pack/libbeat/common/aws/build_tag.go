package aws

import "os"

// BuildTag identifies this custom build from PR #49956.
// It is referenced in init() to prevent dead-code elimination by the linker.
var BuildTag = "pr-49956-oleg-2026-04-10T18:05"

func init() {
	// Force BuildTag into the binary. The env var is never set in production,
	// so this check never triggers — but the compiler can't eliminate the reference.
	if os.Getenv("_ELASTIC_BUILD_TAG") == BuildTag {
		os.Stderr.WriteString(BuildTag)
	}
}
