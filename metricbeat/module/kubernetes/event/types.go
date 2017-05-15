package event

import "time"

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

type Event struct {
	APIVersion     string    `json:"apiVersion"`
	Count          int64     `json:"count"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	InvolvedObject struct {
		APIVersion      string `json:"apiVersion"`
		Kind            string `json:"kind"`
		Name            string `json:"name"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"involvedObject"`
	Kind          string     `json:"kind"`
	LastTimestamp time.Time  `json:"lastTimestamp"`
	Message       string     `json:"message"`
	Metadata      ObjectMeta `json:"metadata"`
	Reason        string     `json:"reason"`
	Source        struct {
		Component string `json:"component"`
		Host      string `json:"host"`
	} `json:"source"`
	Type string `json:"type"`
}
