package common

import "github.com/satori/go.uuid"

// BeatInfo stores a beats instance meta data.
type BeatInfo struct {
	Beat     string    // The actual beat its name
	Version  string    // The beat version. Defaults to the libbeat version when an implementation does not set a version
	Name     string    // configured beat name
	Hostname string    // hostname
	UUID     uuid.UUID // ID assigned to beat instance
}
