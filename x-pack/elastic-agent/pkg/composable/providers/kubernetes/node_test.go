// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
)

func TestGenerateNodeData(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	node := &kubernetes.Node{
		ObjectMeta: kubernetes.ObjectMeta{
			Name: "testnode",
			UID:  types.UID(uid),
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"baz": "ban",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{{Type: v1.NodeReady, Status: v1.ConditionTrue}},
			Addresses:  []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
		},
	}

	data := generateNodeData(node, &Config{})

	mapping := map[string]interface{}{
		"node": map[string]interface{}{
			"uid":  string(node.GetUID()),
			"name": node.GetName(),
			"labels": common.MapStr{
				"foo": "bar",
			},
			"annotations": common.MapStr{
				"baz": "ban",
			},
			"ip": "node1",
		},
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes",
			},
		},
	}

	assert.Equal(t, node, data.node)
	assert.Equal(t, mapping, data.mapping)
	assert.Equal(t, processors, data.processors)
}
