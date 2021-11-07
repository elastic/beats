// +build darwin,!go1.9.2

package fsevents

// Prior to Go 1.9.2, converting C.kFSEventStreamEventIdSinceNow to a uint64
// results in the error: "constant -1 overflows uint64".
// Related Go issue: https://github.com/golang/go/issues/21708

// Hardcoding the value here from FSEvents.h:
//   kFSEventStreamEventIdSinceNow = 0xFFFFFFFFFFFFFFFFULL

// eventIDSinceNow is a sentinel to begin watching events "since now".
const eventIDSinceNow = uint64(0xFFFFFFFFFFFFFFFF)
