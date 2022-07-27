package util

import "testing"

func TestSet(t *testing.T) {
	ns := "namespace"
	pod := "pod"
	container := "container"
	cuid := ContainerUID(ns, pod, container)

	metrics := MetricsStorage{
	}
	metrics[cuid] := MetricsEntry{
		NodeMemAllocatable: &Float64Metric{},
	}
	
}
