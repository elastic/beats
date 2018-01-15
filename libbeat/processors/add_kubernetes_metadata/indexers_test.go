package add_kubernetes_metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

var metagen = kubernetes.NewMetaGenerator([]string{}, []string{}, []string{})

func TestPodIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	podIndexer, err := NewPodNameIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := kubernetes.Pod{
		Metadata: kubernetes.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: kubernetes.PodSpec{
			NodeName: "testnode",
		},
	}

	indexers := podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, fmt.Sprintf("%s/%s", ns, podName))

	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
		},
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
		"node": common.MapStr{
			"name": "testnode",
		},
	}

	assert.Equal(t, expected.String(), indexers[0].Data.String())

	indices := podIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], fmt.Sprintf("%s/%s", ns, podName))
}

func TestContainerIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	conIndexer, err := NewContainerIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	container := "container"
	initContainer := "initcontainer"

	pod := kubernetes.Pod{
		Metadata: kubernetes.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Status: kubernetes.PodStatus{
			ContainerStatuses:     make([]kubernetes.PodContainerStatus, 0),
			InitContainerStatuses: make([]kubernetes.PodContainerStatus, 0),
		},
	}

	indexers := conIndexer.GetMetadata(&pod)
	indices := conIndexer.GetIndexes(&pod)
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
		"node": common.MapStr{
			"name": "testnode",
		},
	}
	pod.Spec.NodeName = "testnode"
	pod.Status.ContainerStatuses = []kubernetes.PodContainerStatus{
		{
			Name:        container,
			ContainerID: "docker://abcde",
		},
	}
	pod.Status.InitContainerStatuses = []kubernetes.PodContainerStatus{
		{
			Name:        initContainer,
			ContainerID: "docker://fghij",
		},
	}

	indexers = conIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 2)
	assert.Equal(t, indexers[0].Index, "abcde")
	assert.Equal(t, indexers[1].Index, "fghij")

	indices = conIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 2)
	assert.Equal(t, indices[0], "abcde")
	assert.Equal(t, indices[1], "fghij")

	expected["container"] = common.MapStr{
		"name": container,
	}
	assert.Equal(t, expected.String(), indexers[0].Data.String())

	expected["container"] = common.MapStr{
		"name": initContainer,
	}
	assert.Equal(t, expected.String(), indexers[1].Data.String())
}

func TestFilteredGenMeta(t *testing.T) {
	var testConfig = common.NewConfig()

	podIndexer, err := NewPodNameIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := kubernetes.Pod{
		Metadata: kubernetes.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"foo": "bar",
				"x":   "y",
			},
			Annotations: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: kubernetes.PodSpec{},
	}

	indexers := podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)

	rawLabels, _ := indexers[0].Data["labels"]
	assert.NotNil(t, rawLabels)

	labelMap, ok := rawLabels.(common.MapStr)
	assert.Equal(t, ok, true)
	assert.Equal(t, len(labelMap), 2)

	rawAnnotations := indexers[0].Data["annotations"]
	assert.Nil(t, rawAnnotations)

	filteredGen := kubernetes.NewMetaGenerator([]string{"a"}, []string{"foo"}, []string{})
	podIndexer, err = NewPodNameIndexer(*testConfig, filteredGen)
	assert.Nil(t, err)

	indexers = podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)

	rawLabels, _ = indexers[0].Data["labels"]
	assert.NotNil(t, rawLabels)

	labelMap, ok = rawLabels.(common.MapStr)
	assert.Equal(t, ok, true)
	assert.Equal(t, len(labelMap), 1)

	ok, _ = labelMap.HasKey("foo")
	assert.Equal(t, ok, true)

	rawAnnotations = indexers[0].Data["annotations"]
	assert.NotNil(t, rawAnnotations)
	annotationsMap, ok := rawAnnotations.(common.MapStr)

	assert.Equal(t, ok, true)
	assert.Equal(t, len(annotationsMap), 1)

	ok, _ = annotationsMap.HasKey("a")
	assert.Equal(t, ok, true)
}

func TestFilteredGenMetaExclusion(t *testing.T) {
	var testConfig = common.NewConfig()

	filteredGen := kubernetes.NewMetaGenerator([]string{}, []string{}, []string{"x"})
	podIndexer, err := NewPodNameIndexer(*testConfig, filteredGen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := kubernetes.Pod{
		Metadata: kubernetes.ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"foo": "bar",
				"x":   "y",
			},
			Annotations: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: kubernetes.PodSpec{},
	}

	assert.Nil(t, err)

	indexers := podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)

	rawLabels, _ := indexers[0].Data["labels"]
	assert.NotNil(t, rawLabels)

	labelMap, ok := rawLabels.(common.MapStr)
	assert.Equal(t, ok, true)
	assert.Equal(t, len(labelMap), 1)

	ok, _ = labelMap.HasKey("foo")
	assert.Equal(t, ok, true)

	ok, _ = labelMap.HasKey("x")
	assert.Equal(t, ok, false)
}

func TestIpPortIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	ipIndexer, err := NewIPPortIndexer(*testConfig, metagen)
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

	assert.Equal(t, 1, len(indexers))
	assert.Equal(t, 1, len(indices))
	assert.Equal(t, ip, indices[0])
	assert.Equal(t, ip, indexers[0].Index)

	// Meta doesn't have container info
	_, err = indexers[0].Data.GetValue("kubernetes.container.name")
	assert.NotNil(t, err)

	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
		},
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
		"node": common.MapStr{
			"name": "testnode",
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
	pod.Spec.NodeName = "testnode"

	indexers = ipIndexer.GetMetadata(&pod)
	assert.Equal(t, 2, len(indexers))
	assert.Equal(t, ip, indexers[0].Index)
	assert.Equal(t, fmt.Sprintf("%s:%d", ip, port), indexers[1].Index)

	indices = ipIndexer.GetIndexes(&pod)
	assert.Equal(t, 2, len(indices))
	assert.Equal(t, ip, indices[0])
	assert.Equal(t, fmt.Sprintf("%s:%d", ip, port), indices[1])

	assert.Equal(t, expected.String(), indexers[0].Data.String())
	expected["container"] = common.MapStr{"name": container}
	assert.Equal(t, expected.String(), indexers[1].Data.String())
}
