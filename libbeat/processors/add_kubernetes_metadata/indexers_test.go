// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package add_kubernetes_metadata

import (
	"fmt"
	"testing"

	v1 "github.com/ericchiang/k8s/apis/core/v1"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

var metagen, _ = kubernetes.NewMetaGenerator(common.NewConfig())

func TestPodIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	podIndexer, err := NewPodNameIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	ns := "testns"
	nodeName := "testnode"
	pod := kubernetes.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Uid:       &uid,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: &v1.PodSpec{
			NodeName: &nodeName,
		},
	}

	indexers := podIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, fmt.Sprintf("%s/%s", ns, podName))

	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
			"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
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

func TestPodUIDIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	metaGenWithPodUID, err := kubernetes.NewMetaGenerator(common.NewConfig())
	assert.Nil(t, err)

	podUIDIndexer, err := NewPodUIDIndexer(*testConfig, metaGenWithPodUID)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	nodeName := "testnode"
	pod := kubernetes.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Uid:       &uid,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: &v1.PodSpec{
			NodeName: &nodeName,
		},
	}

	indexers := podUIDIndexer.GetMetadata(&pod)
	assert.Equal(t, len(indexers), 1)
	assert.Equal(t, indexers[0].Index, uid)

	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
			"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
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

	indices := podUIDIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indices), 1)
	assert.Equal(t, indices[0], uid)
}

func TestContainerIndexer(t *testing.T) {
	var testConfig = common.NewConfig()

	conIndexer, err := NewContainerIndexer(*testConfig, metagen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	container := "container"
	containerImage := "containerimage"
	initContainerImage := "initcontainerimage"
	initContainer := "initcontainer"
	nodeName := "testnode"

	pod := kubernetes.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Uid:       &uid,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Status: &v1.PodStatus{
			ContainerStatuses:     make([]*kubernetes.PodContainerStatus, 0),
			InitContainerStatuses: make([]*kubernetes.PodContainerStatus, 0),
		},
		Spec: &v1.PodSpec{},
	}

	indexers := conIndexer.GetMetadata(&pod)
	indices := conIndexer.GetIndexes(&pod)
	assert.Equal(t, len(indexers), 0)
	assert.Equal(t, len(indices), 0)
	expected := common.MapStr{
		"pod": common.MapStr{
			"name": "testpod",
			"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
		},
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
		"node": common.MapStr{
			"name": "testnode",
		},
	}
	container1 := "docker://abcde"
	pod.Spec.NodeName = &nodeName
	pod.Status.ContainerStatuses = []*kubernetes.PodContainerStatus{
		{
			Name:        &container,
			Image:       &containerImage,
			ContainerID: &container1,
		},
	}
	container2 := "docker://fghij"
	pod.Status.InitContainerStatuses = []*kubernetes.PodContainerStatus{
		{
			Name:        &initContainer,
			Image:       &initContainerImage,
			ContainerID: &container2,
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
		"name":  container,
		"image": containerImage,
	}
	assert.Equal(t, expected.String(), indexers[0].Data.String())

	expected["container"] = common.MapStr{
		"name":  initContainer,
		"image": initContainerImage,
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
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Labels: map[string]string{
				"foo": "bar",
				"x":   "y",
			},
			Annotations: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: &v1.PodSpec{},
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

	config, err := common.NewConfigFrom(map[string]interface{}{
		"include_annotations": []string{"a"},
		"include_labels":      []string{"foo"},
	})
	assert.Nil(t, err)

	filteredGen, err := kubernetes.NewMetaGenerator(config)
	assert.Nil(t, err)

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

	config, err := common.NewConfigFrom(map[string]interface{}{
		"exclude_labels": []string{"x"},
	})
	assert.Nil(t, err)

	filteredGen, err := kubernetes.NewMetaGenerator(config)
	assert.Nil(t, err)

	podIndexer, err := NewPodNameIndexer(*testConfig, filteredGen)
	assert.Nil(t, err)

	podName := "testpod"
	ns := "testns"
	pod := kubernetes.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Namespace: &ns,
			Labels: map[string]string{
				"foo": "bar",
				"x":   "y",
			},
			Annotations: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		Spec: &v1.PodSpec{},
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
	uid := "005f3b90-4b9d-12f8-acf0-31020a840133"
	container := "container"
	containerImage := "containerimage"
	ip := "1.2.3.4"
	port := int32(80)
	pod := kubernetes.Pod{
		Metadata: &metav1.ObjectMeta{
			Name:      &podName,
			Uid:       &uid,
			Namespace: &ns,
			Labels: map[string]string{
				"labelkey": "labelvalue",
			},
		},
		Spec: &v1.PodSpec{
			Containers: make([]*kubernetes.Container, 0),
		},

		Status: &v1.PodStatus{
			PodIP: &ip,
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
			"uid":  "005f3b90-4b9d-12f8-acf0-31020a840133",
		},
		"namespace": "testns",
		"labels": common.MapStr{
			"labelkey": "labelvalue",
		},
		"node": common.MapStr{
			"name": "testnode",
		},
	}

	pod.Spec.Containers = []*v1.Container{
		{
			Name:  &container,
			Image: &containerImage,
			Ports: []*v1.ContainerPort{
				{
					Name:          &container,
					ContainerPort: &port,
				},
			},
		},
	}

	nodeName := "testnode"
	pod.Spec.NodeName = &nodeName

	indexers = ipIndexer.GetMetadata(&pod)
	assert.Equal(t, 2, len(indexers))
	assert.Equal(t, ip, indexers[0].Index)
	assert.Equal(t, fmt.Sprintf("%s:%d", ip, port), indexers[1].Index)

	indices = ipIndexer.GetIndexes(&pod)
	assert.Equal(t, 2, len(indices))
	assert.Equal(t, ip, indices[0])
	assert.Equal(t, fmt.Sprintf("%s:%d", ip, port), indices[1])

	assert.Equal(t, expected.String(), indexers[0].Data.String())
	expected["container"] = common.MapStr{"name": container, "image": containerImage}
	assert.Equal(t, expected.String(), indexers[1].Data.String())
}
