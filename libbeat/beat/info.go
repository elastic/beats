package beat

import "github.com/satori/go.uuid"

// Info stores a beats instance meta data.
type Info struct {
	Beat        string    // The actual beat's name
	IndexPrefix string    // The beat's index prefix in Elasticsearch.
	Version     string    // The beat version. Defaults to the libbeat version when an implementation does not set a version
	Name        string    // configured beat name
	Hostname    string    // hostname
	UUID        uuid.UUID // ID assigned to beat instance
}
