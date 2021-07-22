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

func TestGenerateServiceData(t *testing.T) {
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	service := &kubernetes.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testsvc",
			UID:       types.UID(uid),
			Namespace: "testns",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"baz": "ban",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "1.2.3.4",
			Selector: map[string]string{
				"app":   "istiod",
				"istio": "pilot",
			},
		},
	}

	data := generateServiceData(service, &Config{})

	mapping := map[string]interface{}{
		"service": map[string]interface{}{
			"uid":  string(service.GetUID()),
			"name": service.GetName(),
			"labels": common.MapStr{
				"foo": "bar",
			},
			"annotations": common.MapStr{
				"baz": "ban",
			},
			"ip": service.Spec.ClusterIP,
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

	assert.Equal(t, service, data.service)
	assert.Equal(t, mapping, data.mapping)
	assert.Equal(t, processors, data.processors)
}
