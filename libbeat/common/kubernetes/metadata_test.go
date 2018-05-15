package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestPodMetadataDeDot(t *testing.T) {
	tests := []struct {
		pod     *Pod
		meta    common.MapStr
		metaGen MetaGenerator
	}{
		{
			pod: &Pod{
				Metadata: ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
			},
			meta: common.MapStr{
				"pod":       common.MapStr{"name": ""},
				"namespace": "",
				"node":      common.MapStr{"name": ""},
				"labels":    common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}},
			},
			metaGen: NewMetaGenerator(nil, nil, nil, false),
		},
		{
			pod: &Pod{
				Metadata: ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
					UID:    "005f3b90-4b9d-12f8-acf0-31020a840133",
				},
			},
			meta: common.MapStr{
				"pod":       common.MapStr{"name": "", "uid": "005f3b90-4b9d-12f8-acf0-31020a840133"},
				"namespace": "",
				"node":      common.MapStr{"name": ""},
				"labels":    common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}},
			},
			metaGen: NewMetaGenerator(nil, nil, nil, true),
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.metaGen.PodMetadata(test.pod), test.meta)
	}
}
