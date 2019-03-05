package v1alpha1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("settings.k8s.io", "v1alpha1", "podpresets", true, &PodPreset{})

	k8s.RegisterList("settings.k8s.io", "v1alpha1", "podpresets", true, &PodPresetList{})
}
