package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestPodMetadataDeDot(t *testing.T) {
	tests := []struct {
		pod  *Pod
		meta common.MapStr
	}{
		{
			pod: &Pod{
				Metadata: ObjectMeta{
					Labels: map[string]string{"a.key": "foo", "a": "bar"},
				},
			},
			meta: common.MapStr{"labels": common.MapStr{"a": common.MapStr{"value": "bar", "key": "foo"}}},
		},
	}

	for _, test := range tests {
		assert.Equal(t, NewMetaGenerator(nil, nil, nil).PodMetadata(test.pod)["labels"], test.meta["labels"])
	}
}
