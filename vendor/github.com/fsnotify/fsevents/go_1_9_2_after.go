// +build darwin,go1.9.2

package fsevents

/*
#include <CoreServices/CoreServices.h>
*/
import "C"

// eventIDSinceNow is a sentinel to begin watching events "since now".
const eventIDSinceNow = uint64(C.kFSEventStreamEventIdSinceNow)
