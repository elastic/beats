package harvester

import (
	uuid "github.com/satori/go.uuid"
)

// Harvester contains all methods which must be supported by each harvester
// so the registry can be used by the input
type Harvester interface {
	ID() uuid.UUID
	Run() error
	Stop()
}
