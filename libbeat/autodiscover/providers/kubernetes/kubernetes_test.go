package kubernetes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/kubernetes"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		event  bus.Event
		result bus.Event
	}{
		// Empty events should return empty hints
		{
			event:  bus.Event{},
			result: bus.Event{},
		},
		// Only kubernetes payload must return only kubernetes as part of the hint
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "foobar",
					},
				},
			},
		},
		// Kubernetes payload with container info must be bubbled to top level
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "rkt",
					},
				},
				"container": common.MapStr{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "rkt",
				},
			},
		},
		// Scenarios being tested:
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			event: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern": "^test",
						"co.elastic.metrics/module":         "prometheus",
						"co.elastic.metrics/period":         "10s",
						"co.elastic.metrics.foobar/period":  "15s",
						"not.to.include":                    "true",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
			},
			result: bus.Event{
				"kubernetes": common.MapStr{
					"annotations": getNestedAnnotations(common.MapStr{
						"co.elastic.logs/multiline.pattern": "^test",
						"co.elastic.metrics/module":         "prometheus",
						"not.to.include":                    "true",
						"co.elastic.metrics/period":         "10s",
						"co.elastic.metrics.foobar/period":  "15s",
					}),
					"container": common.MapStr{
						"name":    "foobar",
						"id":      "abc",
						"runtime": "docker",
					},
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
						},
					},
					"metrics": common.MapStr{
						"module": "prometheus",
						"period": "15s",
					},
				},
				"container": common.MapStr{
					"name":    "foobar",
					"id":      "abc",
					"runtime": "docker",
				},
			},
		},
	}

	cfg := defaultConfig()

	p := Provider{
		config: cfg,
	}
	for _, test := range tests {
		assert.Equal(t, p.generateHints(test.event), test.result)
	}
}

func TestEmitEvent(t *testing.T) {
	tests := []struct {
		Message  string
		Flag     string
		Pod      *kubernetes.Pod
		Expected bus.Event
	}{
		{
			Message: "Test common pod start",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				Metadata: kubernetes.ObjectMeta{
					Name:        "filebeat",
					Namespace:   "default",
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Status: kubernetes.PodStatus{
					PodIP: "127.0.0.1",
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        "filebeat",
							ContainerID: "docker://foobar",
						},
					},
				},
				Spec: kubernetes.PodSpec{
					NodeName: "node",
					Containers: []kubernetes.Container{
						{
							Image: "elastic/filebeat:6.3.0",
							Name:  "filebeat",
						},
					},
				},
			},
			Expected: bus.Event{
				"start": true,
				"host":  "127.0.0.1",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"id":      "foobar",
						"name":    "filebeat",
						"image":   "elastic/filebeat:6.3.0",
						"runtime": "docker",
					},
					"pod": common.MapStr{
						"name": "filebeat",
					},
					"node": common.MapStr{
						"name": "node",
					},
					"namespace":   "default",
					"annotations": common.MapStr{},
				},
				"meta": common.MapStr{
					"kubernetes": common.MapStr{
						"namespace": "default",
						"container": common.MapStr{
							"name": "filebeat",
						}, "pod": common.MapStr{
							"name": "filebeat",
						}, "node": common.MapStr{
							"name": "node",
						},
					},
				},
			},
		},
		{
			Message: "Test pod without host",
			Flag:    "start",
			Pod: &kubernetes.Pod{
				Metadata: kubernetes.ObjectMeta{
					Name:        "filebeat",
					Namespace:   "default",
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Status: kubernetes.PodStatus{
					ContainerStatuses: []kubernetes.PodContainerStatus{
						{
							Name:        "filebeat",
							ContainerID: "docker://foobar",
						},
					},
				},
				Spec: kubernetes.PodSpec{
					NodeName: "node",
					Containers: []kubernetes.Container{
						{
							Image: "elastic/filebeat:6.3.0",
							Name:  "filebeat",
						},
					},
				},
			},
			Expected: nil,
		},
	}

	for _, test := range tests {
		mapper, err := template.NewConfigMapper(nil)
		if err != nil {
			t.Fatal(err)
		}

		metaGen := kubernetes.NewMetaGenerator(nil, nil, nil)
		p := &Provider{
			config:    defaultConfig(),
			bus:       bus.New("test"),
			metagen:   metaGen,
			templates: mapper,
		}

		listener := p.bus.Subscribe()

		p.emit(test.Pod, test.Flag)

		select {
		case event := <-listener.Events():
			assert.Equal(t, test.Expected, event)
		case <-time.After(2 * time.Second):
			if test.Expected != nil {
				t.Fatal("Timeout while waiting for event")
			}
		}
	}
}

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}
