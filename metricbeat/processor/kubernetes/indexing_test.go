package kubernetes

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors/kubernetes"
	"github.com/stretchr/testify/assert"
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
	port := int64(80)
	pod := kubernetes.Pod{
		Metadata: kubernetes.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: kubernetes.PodSpec{
			Containers: make([]kubernetes.Container, 0),
		},

		Status: kubernetes.PodStatus{
			PodIP: ip,
		},
	}

	indexers := ipIndexer.GetMetadata(&pod)
	indices := ipIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indexers), 0)
	assert.Equal(t, len(indices), 0)
	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
		},
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
	}

	pod.Spec.Containers = []kubernetes.Container{
		{
			Name: container,
			Ports: []kubernetes.ContainerPort{
				{
					Name:          container,
					ContainerPort: port,
				},
			},
		},
	}
	expected["container"] = common.MapStr{"name": container}

	indexers = ipIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, fmt.Sprintf("%s:%d", ip, port))

	indices = ipIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], fmt.Sprintf("%s:%d", ip, port))

	assert.Equal(t, expected.String(), indexers[0].Data.String())
}
