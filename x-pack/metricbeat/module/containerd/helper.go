package containerd

import "github.com/elastic/beats/v7/libbeat/common"

// SetCIDandNamespace sets container.id ECS field and containerd.namespace module field
func SetCIDandNamespace(event common.MapStr) (common.MapStr, common.MapStr, string) {
	containerFields := common.MapStr{}
	moduleFields := common.MapStr{}
	rootFields := common.MapStr{}
	var cID string
	if containerID, ok := event["id"]; ok {
		cID = (containerID).(string)
		containerFields.Put("id", cID)
		event.Delete("id")
	}
	if len(containerFields) > 0 {
		rootFields.Put("container", containerFields)
	}
	if ns, ok := event["namespace"]; ok {
		moduleFields.Put("namespace", ns)
		event.Delete("namespace")
	}
	return rootFields, moduleFields, cID
}
