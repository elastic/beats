package kubernetes

import (
	"github.com/elastic/beats/libbeat/common"
	corev1 "github.com/ericchiang/k8s/api/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPodIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	podIndexer, err := NewPodNameIndexer(*testConfig)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := corev1.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: &corev1.PodSpec{},
	}

	indexers := podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, podName)

	expected := common.MapStr{
		"pod":       "testpod",
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
	}

	assert.Equal(t, expected.String(), indexers[0].Data.String())

	indices := podIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], podName)
}

func TestContainerIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	conIndexer, err := NewContainerIndexer(*testConfig)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	container := "container"

	pod := corev1.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Status: &corev1.PodStatus{
			ContainerStatuses: make([]*corev1.ContainerStatus, 0),
		},
	}

	indexers := conIndexer.GetMetadata(&pod)
	indices := conIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indexers), 0)
	assert.Equal(t, len(indices), 0)
	expected := common.MapStr{
		"pod":       "testpod",
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
	}

	cid := "docker://abcde"

	pod.Status.ContainerStatuses = []*corev1.ContainerStatus{
		{
			Name:        &container,
			ContainerID: &cid,
		},
	}
	expected["container"] = container

	indexers = conIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, "abcde")

	indices = conIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], "abcde")

	assert.Equal(t, expected.String(), indexers[0].Data.String())
}

func TestFieldMatcher(t *testing.T) {
	testCfg := map[string]interface{}{
		"lookup_fields": []string{},
	}
	fieldCfg, err := common.NewConfigFrom(testCfg)

	assert.Nil(t, err)
	matcher, err := NewFieldMatcher(*fieldCfg)
	assert.NotNil(t, err)

	testCfg["lookup_fields"] = "foo"
	fieldCfg, _ = common.NewConfigFrom(testCfg)

	matcher, err = NewFieldMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	input := common.MapStr{
		"foo": "bar",
	}

	out := matcher.MetadataIndex(input)
	assert.Equal(t, out, "bar")

	nonMatchInput := common.MapStr{
		"not": "match",
	}

	out = matcher.MetadataIndex(nonMatchInput)
	assert.Equal(t, out, "")
}
