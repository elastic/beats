package v1beta1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("policy", "v1beta1", "poddisruptionbudgets", true, &PodDisruptionBudget{})

	k8s.RegisterList("policy", "v1beta1", "poddisruptionbudgets", true, &PodDisruptionBudgetList{})
}
