package elasticsearch

import (
	_ "embed"
	"testing"

	"github.com/elastic/elastic-agent-libs/config"
)

//go:embed testdata/filebeat.yml
var beatYAMLCfg string

func TestToOtelConfig(t *testing.T) {
	beatCfg := config.MustNewConfigFrom(beatYAMLCfg)

	otelCfg, err := ToOtelConfig(beatCfg)
	if err != nil {
		t.Fatalf("could not convert Beat config to OTel elasicsearch exporter: %s", err)
	}

	got, want := string(otelCfg.Authentication.Password), "password"
	if got != want {
		t.Errorf("expecting password to be 'password', got '%s' instead", got)
	}

	got, want = otelCfg.Authentication.User, "elastic-cloud"
	if got != want {
		t.Errorf("expecting User %q, got %q", want, got)
	}
}
