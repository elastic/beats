package common

import (
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func GetOrCreateRegistry(namespace *monitoring.Namespace, registryName string) *monitoring.Registry {
	if namespace == nil {
		return monitoring.GetNamespace(registryName).GetRegistry()
	}
	reg := namespace.GetRegistry().GetRegistry(registryName)
	if reg == nil {
		reg = namespace.GetRegistry().NewRegistry(registryName)
	}
	return reg
}
