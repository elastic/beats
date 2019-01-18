package index

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/libbeat/ilm"

	"github.com/elastic/go-ucfg/yaml"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigs_ValidateConfig(t *testing.T) {
	testdata := []struct {
		cfg     common.MapStr
		indices Configs
		err     string
		name    string
	}{
		{name: "invalid", cfg: common.MapStr{"name": "my-index", "condition": common.MapStr{"when": ""}}, indices: Configs{}, err: "exactly one indices option is required"},
		{name: "valid", cfg: common.MapStr{"name": "my-index"}, indices: Configs{}, err: "exactly one indices option is required"},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(td.cfg)
			require.NoError(t, err)
			var tmp Configs
			err = cfg.Unpack(&tmp)
			require.NoError(t, err)

			err = tmp.Validate()
			if td.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, td.indices, tmp)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.err, fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
		})
	}
}

func TestConfigs_CompatibleIndexCfg(t *testing.T) {
	//load `indices` config section
	cfg, err := yaml.NewConfigWithFile("./testdata/beat.yml")
	require.NoError(t, err)
	cfg, err = cfg.Child("indices", -1)
	require.NoError(t, err)
	var indicesCfg Configs
	err = cfg.Unpack(&indicesCfg)
	require.NoError(t, err)

	//check compatibility mode
	index, indices, err := indicesCfg.CompatibleIndexCfg(&ESNoop{})
	require.NoError(t, err)

	var deprIndices []common.MapStr
	err = indices.Unpack(&deprIndices)
	require.NoError(t, err)

	assert.Equal(t, "metricbeat-%{[agent.version]}-%{+yyyy.MM.dd}", index)
	assert.Equal(t, 1, len(deprIndices))
	assert.Equal(t, "metricbeat-load-%{[agent.version]}-%{+yyyy.MM.dd}", deprIndices[0]["index"])
	when := map[string]interface{}{"contains": map[string]interface{}{"metricset": map[string]interface{}{"name": "load"}}}
	assert.Equal(t, when, deprIndices[0]["when"])
}

func TestConfigs_DefaultConfig(t *testing.T) {
	templateName := "%{[agent.name]}-%{[agent.version]}"
	tmplCfg := template.DefaultTemplateCfg()
	tmplCfg.Name = templateName
	tmplCfg.Pattern = fmt.Sprintf("%s*", templateName)
	assert.Equal(t, tmplCfg, DefaultConfig.Template)

	ilmCfg := ilm.DefaultILMConfig()
	ilmCfg.RolloverAlias = fmt.Sprintf("%s-%s", templateName, ilm.DefaultPattern)
	assert.Equal(t, ilmCfg, DefaultConfig.ILM)

}

type ESNoop struct{}

func (es *ESNoop) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}
func (es *ESNoop) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	return 200, []byte{}, nil
}
func (es *ESNoop) GetVersion() common.Version { return *common.MustNewVersion("7.0.0") }
