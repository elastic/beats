package template

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	testdata := []struct {
		cfg      common.MapStr
		template Config
		err      string
		name     string
	}{
		{name: "invalid", cfg: nil, template: Config{}, err: "template configuration requires a name"},
		{name: "default config", cfg: common.MapStr{"name": "beat"}, template: Config{Enabled: true, Name: "beat", Pattern: "beat*"}},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(td.cfg)
			require.NoError(t, err)
			var tmp Config
			err = cfg.Unpack(&tmp)
			if td.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, td.template, tmp)
			} else if assert.Error(t, err) {
				assert.True(t, strings.Contains(err.Error(), td.err), fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
		})
	}

}
