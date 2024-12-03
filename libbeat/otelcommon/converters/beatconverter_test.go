package converters

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestESConverter(t *testing.T) {
	expectedCfgMap, _ := confmaptest.LoadConf("otel.yaml")
	c := converter{}

	fmt.Println(c.Convert(context.Background(), expectedCfgMap))
	s, _ := json.MarshalIndent(expectedCfgMap.ToStringMap(), "", " ")
	fmt.Println(string(s))
}
