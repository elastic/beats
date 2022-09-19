package move_fields

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

func TestMoveFields(t *testing.T) {
	cases := []struct {
		in, except common.MapStr
		p          *moveFields
	}{
		{
			common.MapStr{"app": common.MapStr{"version": 1, "method": "2"}, "other": 3},
			common.MapStr{"app": common.MapStr{"method": "2"}, "rpc": common.MapStr{"version": 1}, "other": 3},
			&moveFields{config: moveFieldsConfig{
				ParentPath: "app",
				From:       nil,
				To:         "rpc.",
				Exclude:    []string{"method"},
				excludeMap: map[string]bool{"method": true},
			}},
		},
		{
			common.MapStr{"app": common.MapStr{"version": 1, "method": "2"}, "other": 3},
			common.MapStr{"app": common.MapStr{}, "rpc": common.MapStr{"method": "2", "version": 1}, "other": 3},
			&moveFields{config: moveFieldsConfig{
				ParentPath: "app",
				From:       nil,
				To:         "rpc.",
				Exclude:    nil,
				excludeMap: nil,
			}},
		},
		{
			common.MapStr{"app": common.MapStr{"version": 1, "method": "2"}, "other": 3},
			common.MapStr{"app": common.MapStr{}, "rpc_method": "2", "rpc_version": 1, "other": 3},
			&moveFields{config: moveFieldsConfig{
				ParentPath: "app",
				From:       nil,
				To:         "rpc_",
				Exclude:    nil,
				excludeMap: nil,
			}},
		},
		{
			common.MapStr{"app.version": 1, "other": 3},
			common.MapStr{"app": common.MapStr{"version": 1}, "other": 3},
			&moveFields{config: moveFieldsConfig{
				ParentPath: "",
				From:       []string{"app.version"},
				To:         "",
				Exclude:    nil,
				excludeMap: nil,
			}},
		},
		{
			common.MapStr{"app": common.MapStr{"version": 1, "method": "2"}, "other": 3},
			common.MapStr{"my_prefix_app": common.MapStr{"version": 1, "method": "2"}, "my_prefix_other": 3},
			&moveFields{config: moveFieldsConfig{
				ParentPath: "",
				From:       nil,
				To:         "my_prefix_",
				Exclude:    nil,
				excludeMap: nil,
			}},
		},
	}

	for idx, c := range cases {
		evt := &beat.Event{Fields: c.in.Clone()}
		out, err := c.p.Run(evt)
		if err != nil {
			t.Fatal(err)
		}
		except, output := c.except.String(), out.Fields.String()
		if !reflect.DeepEqual(c.except, out.Fields) {
			t.Fatalf("move field test case failed, out: %s, except: %s, index: %d\n",
				output, except, idx)
		}
	}
}
