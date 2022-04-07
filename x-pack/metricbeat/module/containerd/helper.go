// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package containerd

import "github.com/elastic/beats/v8/libbeat/common"

// GetAndDeleteCid deletes and returns container id from an event
func GetAndDeleteCid(event common.MapStr) (cID string) {
	if containerID, ok := event["id"]; ok {
		cID = (containerID).(string)
		event.Delete("id")
	}
	return
}

// GetAndDeleteNamespace deletes and returns namespace from an event
func GetAndDeleteNamespace(event common.MapStr) (namespace string) {
	if ns, ok := event["namespace"]; ok {
		namespace = (ns).(string)
		event.Delete("namespace")
	}
	return
}
