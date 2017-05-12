package harvester

import uuid "github.com/satori/go.uuid"

type Harvester interface {
	ID() uuid.UUID
	Start()
	Stop()
}
