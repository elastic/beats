package add_data_stream_index

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddDataStreamIndex(t *testing.T) {
	simpleDs := DataStream{
		"myns",
		"myds",
		"mytype",
	}
	tests := []struct {
		name    string
		ds  DataStream
		event *beat.Event
		want    string
		wantErr bool
	}{
		{
			"simple",
			simpleDs,
			&beat.Event{},
			"mytype-myds-myns",
			false,
		},
		{
			"existing meta",
			simpleDs,
			&beat.Event{Meta: common.MapStr{}},
			"mytype-myds-myns",
			false,
		},
		{
			"custom ds",
			simpleDs,
			&beat.Event{Meta: common.MapStr{
				FieldMetaCustomDataset: "custom-ds",
			}},
			"mytype-custom-ds-myns",
			false,
		},
		{
			"defaults ds/ns",
			DataStream{
				Type: "mytype",
			},
			&beat.Event{},
			"mytype-generic-default",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.ds)
			got, err := p.Run(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got.Meta[events.FieldMetaRawIndex])
		})
	}
}
