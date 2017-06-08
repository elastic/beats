package add_kubernetes_metadata

type ObjectMeta struct {
	Annotations       map[string]string `json:"annotations"`
	CreationTimestamp string            `json:"creationTimestamp"`
	DeletionTimestamp string            `json:"deletionTimestamp"`
	GenerateName      string            `json:"generateName"`
	Labels            map[string]string `json:"labels"`
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	OwnerReferences   []struct {
		APIVersion string `json:"apiVersion"`
		Controller bool   `json:"controller"`
		Kind       string `json:"kind"`
		Name       string `json:"name"`
		UID        string `json:"uid"`
	} `json:"ownerReferences"`
	ResourceVersion string `json:"resourceVersion"`
	SelfLink        string `json:"selfLink"`
	UID             string `json:"uid"`
}

type Container struct {
	Image                  string          `json:"image"`
	ImagePullPolicy        string          `json:"imagePullPolicy"`
	Name                   string          `json:"name"`
	Ports                  []ContainerPort `json:"ports"`
	Resources              struct{}        `json:"resources"`
	TerminationMessagePath string          `json:"terminationMessagePath"`
	VolumeMounts           []struct {
		MountPath string `json:"mountPath"`
		Name      string `json:"name"`
		ReadOnly  bool   `json:"readOnly"`
	} `json:"volumeMounts"`
}

type ContainerPort struct {
	Name          string `json:"name"`
	ContainerPort int64  `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type PodSpec struct {
	Containers                    []Container `json:"containers"`
	DNSPolicy                     string      `json:"dnsPolicy"`
	NodeName                      string      `json:"nodeName"`
	RestartPolicy                 string      `json:"restartPolicy"`
	SecurityContext               struct{}    `json:"securityContext"`
	ServiceAccount                string      `json:"serviceAccount"`
	ServiceAccountName            string      `json:"serviceAccountName"`
	TerminationGracePeriodSeconds int64       `json:"terminationGracePeriodSeconds"`
}

type PodStatusCondition struct {
	LastProbeTime      interface{} `json:"lastProbeTime"`
	LastTransitionTime string      `json:"lastTransitionTime"`
	Status             string      `json:"status"`
	Type               string      `json:"type"`
}

type PodContainerStatus struct {
	ContainerID string `json:"containerID"`
	Image       string `json:"image"`
	ImageID     string `json:"imageID"`
	LastState   struct {
		Terminated struct {
			ContainerID string `json:"containerID"`
			ExitCode    int64  `json:"exitCode"`
			FinishedAt  string `json:"finishedAt"`
			Reason      string `json:"reason"`
			StartedAt   string `json:"startedAt"`
		} `json:"terminated"`
	} `json:"lastState"`
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int64  `json:"restartCount"`
	State        struct {
		Running struct {
			StartedAt string `json:"startedAt"`
		} `json:"running"`
	} `json:"state"`
}

type PodStatus struct {
	Conditions        []PodStatusCondition `json:"conditions"`
	ContainerStatuses []PodContainerStatus `json:"containerStatuses"`
	HostIP            string               `json:"hostIP"`
	Phase             string               `json:"phase"`
	PodIP             string               `json:"podIP"`
	StartTime         string               `json:"startTime"`
}

type Pod struct {
	APIVersion string     `json:"apiVersion"`
	Kind       string     `json:"kind"`
	Metadata   ObjectMeta `json:"metadata"`
	Spec       PodSpec    `json:"spec"`
	Status     PodStatus  `json:"status"`
}
