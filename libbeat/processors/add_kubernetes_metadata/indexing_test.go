package add_kubernetes_metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

var metagen = &GenDefaultMeta{}

func TestPodIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	podIndexer, err := NewPodNameIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := Pod{
		Metadata: ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: PodSpec{},
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

	pod := Pod{
		Metadata: ObjectMeta{
			Name:      podName,
			Namespace: ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Status: PodStatus{
			ContainerStatuses:     make([]PodContainerStatus, 0),
			InitContainerStatuses: make([]PodContainerStatus, 0),
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
	}

	pod.Status.ContainerStatuses = []PodContainerStatus{
		{
			Name:        container,
			ContainerID: "docker://abcde",
		},
	}
	pod.Status.InitContainerStatuses = []PodContainerStatus{
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

func TestFilteredGenMeta(t *testing.T) {
	var testConfig = common.NewConfig()

	filteredGen := &GenDefaultMeta{}
	podIndexer, err := NewPodNameIndexer(*testConfig, filteredGen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := Pod{
		Metadata: ObjectMeta{
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
		Spec: PodSpec{},
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

	filteredGen.labels = []string{"foo"}
	filteredGen.annotations = []string{"a"}

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

func TestFieldFormatMatcher(t *testing.T) {
	testCfg := map[string]interface{}{}
	fieldCfg, err := common.NewConfigFrom(testCfg)

	assert.Nil(t, err)
	matcher, err := NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, err)

	testCfg["format"] = `%{[namespace]}/%{[pod]}`
	fieldCfg, _ = common.NewConfigFrom(testCfg)

	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	event := common.MapStr{
		"namespace": "foo",
		"pod":       "bar",
	}

	out := matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)

	event = common.MapStr{
		"foo": "bar",
	}
	out = matcher.MetadataIndex(event)
	assert.Empty(t, out)

	testCfg["format"] = `%{[dimensions.namespace]}/%{[dimensions.pod]}`
	fieldCfg, _ = common.NewConfigFrom(testCfg)
	matcher, err = NewFieldFormatMatcher(*fieldCfg)
	assert.NotNil(t, matcher)
	assert.Nil(t, err)

	event = common.MapStr{
		"dimensions": common.MapStr{
			"pod":       "bar",
			"namespace": "foo",
		},
	}

	out = matcher.MetadataIndex(event)
	assert.Equal(t, "foo/bar", out)
}

func TestFilteredGenMetaExclusion(t *testing.T) {
	var testConfig = common.NewConfig()

	filteredGen := &GenDefaultMeta{
		labelsExclude: []string{"x"},
	}
	podIndexer, err := NewPodNameIndexer(*testConfig, filteredGen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := Pod{
		Metadata: ObjectMeta{
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
		Spec: PodSpec{},
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
