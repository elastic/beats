package add_process_metadata

import (
	"fmt"
	"strings"
)

type MetadataType uint32

const (
	ContainerID MetadataType = iota + 1
	Cgroups
)

var metadataTypeNames = map[MetadataType]string{
	ContainerID: "container_id",
	Cgroups:     "cgroups",
}

func (t MetadataType) String() string {
	name, found := metadataTypeNames[t]
	if found {
		return name
	}
	return fmt.Sprintf("unknown (%d)", t)
}

func (t *MetadataType) Unpack(s string) error {
	s = strings.ToLower(s)
	for typ, name := range metadataTypeNames {
		if s == name {
			*t = typ
			return nil
		}
	}
	return fmt.Errorf("invalid metadata type '%v'", s)
}

type Config struct {
	// Metadata specifies what types of process information to capture.
	Metadata []MetadataType `config:"metadata_types" validate:"required"`

	// PIDFields specifies the field names that contain process IDs.
	PIDFields []string `config:"pid_fields" validate:"required"`

	// Targets
	ContainerIDTarget string `config:"target.container_id"` // Target field for the container ID.
	CgroupsTarget     string `config:"target.cgroups"`      // Target field for cgroup subsystems and paths.
}

var defaultConfig = Config{
	Metadata:          []MetadataType{ContainerID},
	ContainerIDTarget: "docker.container.id",
	CgroupsTarget:     "process.cgroups",
}
