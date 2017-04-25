package kubernetes

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors/annotate/kubernetes"
	corev1 "github.com/ericchiang/k8s/api/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

var metagen = &kubernetes.GenDefaultMeta{}

func TestIpPortIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	ipIndexer, err := newIpPortIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	container := "container"
	ip := "1.2.3.4"
	port := int32(80)
	pod := corev1.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: &corev1.PodSpec{
			Containers: make([]*corev1.Container, 0),
		},

		Status: &corev1.PodStatus{
			PodIP: &ip,
		},
	}

	indexers := ipIndexer.GetMetadata(&pod)
	indices := ipIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indexers), 0)
	assert.Equal(t, len(indices), 0)
	expected := common.MapStr{
		"pod":       "testpod",
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
	}

	pod.Spec.Containers = []*corev1.Container{
		{
			Name: &container,
			Ports: []*corev1.ContainerPort{
				{
					Name:          &container,
					ContainerPort: &port,
				},
			},
		},
	}
	expected["container"] = container

	indexers = ipIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, fmt.Sprintf("%s:%d", ip, port))

	indices = ipIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], fmt.Sprintf("%s:%d", ip, port))

	assert.Equal(t, expected.String(), indexers[0].Data.String())
}
