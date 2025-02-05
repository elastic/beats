package monitoring

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	monitoring2 "github.com/elastic/elastic-agent-libs/monitoring"
)

const (
	RegistryNameInternalInputs = "internal.inputs"
)

func BeatInternalInputsRegistry(beatInfo beat.Info) *monitoring2.Registry {
	internalReg := beatInfo.Monitoring.Namespace.
		GetRegistry().
		GetRegistry(RegistryNameInternalInputs)
	if internalReg == nil {
		internalReg = beatInfo.Monitoring.Namespace.
			GetRegistry().
			NewRegistry(RegistryNameInternalInputs)
	}
	return internalReg
}
