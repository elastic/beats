package v1beta1

import "github.com/ericchiang/k8s"

func init() {
	k8s.Register("apps", "v1beta1", "controllerrevisions", true, &ControllerRevision{})
	k8s.Register("apps", "v1beta1", "deployments", true, &Deployment{})
	k8s.Register("apps", "v1beta1", "statefulsets", true, &StatefulSet{})

	k8s.RegisterList("apps", "v1beta1", "controllerrevisions", true, &ControllerRevisionList{})
	k8s.RegisterList("apps", "v1beta1", "deployments", true, &DeploymentList{})
	k8s.RegisterList("apps", "v1beta1", "statefulsets", true, &StatefulSetList{})
}
