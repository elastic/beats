package logv2

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/input/filestream"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestDirectTranslate(t *testing.T) {
	src := config.MustNewConfigFrom(map[string]any{
		"id":    "foo-id",
		"paths": []string{"foo", "bar"},
	})

	newCfg, err := translateCfg(src)
	if err != nil {
		t.Fatalf("cannot translate config: %s", err)
	}

	str := config.DebugString(newCfg, false)
	t.Log(str)

	fsCfg := filestream.Config{}
	if err := newCfg.Unpack(&fsCfg); err != nil {
		t.Fatalf("cannot unpack translated config: %s", err)
	}
}
