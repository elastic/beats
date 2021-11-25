//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"context"
	"path"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestInputParsers(t *testing.T) {
	inputParsersExpected := []string{"1st line\n2nd line\n3rd line", "4th line\n5th line\n6th line"}
	env := newInputTestingEnvironment(t)

	inp := env.mustCreateInput(common.MapStr{
		"paths":           []string{path.Join("testdata", "input-multiline-parser.journal")},
		"include_matches": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
		"parsers": []common.MapStr{
			{
				"multiline": common.MapStr{
					"type":        "count",
					"count_lines": 3,
				},
			},
		},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	env.startInput(ctx, inp)
	env.waitUntilEventCount(len(inputParsersExpected))

	for idx, event := range env.pipeline.clients[0].GetEvents() {
		if got, expected := event.Fields["message"], inputParsersExpected[idx]; got != expected {
			t.Errorf("expecting event message %q, got %q", expected, got)
		}
	}

	cancelInput()
}
