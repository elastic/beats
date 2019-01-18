package ilm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/version"
)

func TestESLoader_LoadPolicy(t *testing.T) {
	testdata := []struct {
		cfg    Config
		client ESClient

		loaded bool
		err    string

		name string
	}{
		//ilm not supported by client
		{name: "ilm auto and not supported", loaded: false, cfg: Config{Enabled: ModeAuto}, client: &ES65{}},
		{name: "ilm enabled and not supported", loaded: false, cfg: Config{Enabled: ModeEnabled}, client: &ES65{}, err: "ILM set to `true`"},
		{name: "ilm auto no client", loaded: false, cfg: Config{Enabled: ModeAuto}},
		{name: "ilm disabled no client", loaded: false, cfg: Config{Enabled: ModeDisabled}},
		{name: "ilm enabled and not supported", loaded: false, cfg: Config{Enabled: ModeEnabled}, client: &ES65{}, err: "ILM set to `true`"},

		//ilm supported by client
		{name: "ilm disabled", loaded: false, client: &ES66{},
			cfg: Config{Enabled: ModeDisabled, RolloverAlias: "testbeat", Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "ilm auto", loaded: true, client: &ES66{},
			cfg: Config{Enabled: ModeAuto, RolloverAlias: "testbeat-%{[observer.name]}-%{[observer.version]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "ilm enabled", loaded: true, client: &ES66{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[agent.name]}-%{[agent.version]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}}},

		{name: "ilm rollover_alias invalid", loaded: false, client: &ES66{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[agent.id]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}},
			err: "key not found"},
		{name: "ilm policy invalid", loaded: false, client: &ES66{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[beat.name]}-%{[beat.version]}",
				Policy: PolicyCfg{Name: "invalid"}},
			err: "no ILM policy found"},
		{name: "ilm policy could not be created", loaded: false, client: &ES66Error{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[agent.name]}-%{[agent.version]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}},
			err: "could not be created"}, //error only raised from ES loader
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			info := beat.Info{Version: version.GetDefaultVersion(), IndexPrefix: "testbeat"}
			loader, err := NewESLoader(td.client, info)
			require.NoError(t, err)

			loaded, err := loader.LoadPolicy(td.cfg)
			assert.Equal(t, td.loaded, loaded)
			if td.err == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.err, fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
		})
	}
}

func TestESLoader_LoadWriteAlias(t *testing.T) {
	testdata := []struct {
		cfg    Config
		client ESClient

		loaded bool
		err    string

		name string
	}{
		//ilm not supported by client
		{name: "ilm auto and not supported", loaded: false, cfg: Config{Enabled: ModeAuto}, client: &ES65{}},
		{name: "ilm enabled and not supported", loaded: false, cfg: Config{Enabled: ModeEnabled}, client: &ES65{}, err: "ILM set to `true`"},
		{name: "ilm auto no client", loaded: false, cfg: Config{Enabled: ModeAuto}},
		{name: "ilm disabled no client", loaded: false, cfg: Config{Enabled: ModeDisabled}},
		{name: "ilm enabled and not supported", loaded: false, cfg: Config{Enabled: ModeEnabled}, client: &ES65{}, err: "ILM set to `true`"},

		//ilm supported by client
		{name: "ilm disabled", loaded: false, client: &ES66{},
			cfg: Config{Enabled: ModeDisabled, RolloverAlias: "testbeat", Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "ilm auto", loaded: true, client: &ES66{},
			cfg: Config{Enabled: ModeAuto, RolloverAlias: "testbeat-%{[observer.name]}-%{[observer.version]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}}},
		{name: "ilm enabled", loaded: true, client: &ES66{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[agent.name]}-%{[agent.version]}",
				Policy: PolicyCfg{Name: DefaultPolicyName}}},

		{name: "ilm rollover_alias invalid", loaded: false, client: &ES66{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[agent.id]}"},
			err: "key not found"},
		{name: "ilm rollover_alias exists", loaded: false, client: &ES66Error{},
			cfg: Config{Enabled: ModeEnabled, RolloverAlias: "testbeat-%{[beat.name]}-%{[beat.version]}"}},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			info := beat.Info{Version: version.GetDefaultVersion(), IndexPrefix: "testbeat"}
			loader, err := NewESLoader(td.client, info)
			require.NoError(t, err)

			loaded, err := loader.LoadWriteAlias(td.cfg)
			fmt.Println(err)
			assert.Equal(t, td.loaded, loaded)
			if td.err == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.err, fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
		})
	}
}

func TestStdoutLoader_LoadPolicy(t *testing.T) {
	testdata := []struct {
		cfg Config

		printed    bool
		err        string
		policyName string

		name string
	}{
		{name: "ilm auto", printed: true, cfg: Config{Enabled: ModeAuto}, policyName: DefaultPolicyName},
		{name: "ilm enabled", printed: true, cfg: Config{Enabled: ModeEnabled, Policy: PolicyCfg{Name: "deleteAfter10Days"}},
			policyName: "deleteAfter10Days"},
		{name: "ilm disabled", printed: false, cfg: Config{Enabled: ModeDisabled}},
		{name: "ilm policy invalid", printed: false, cfg: Config{
			Enabled:       ModeEnabled,
			RolloverAlias: "testbeat-%{[beat.name]}-%{[beat.version]}",
			Policy:        PolicyCfg{Name: "invalid"}},
			err: "no ILM policy found"},
	}
	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {

			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			defer func() {
				w.Close()
				os.Stdout = old

				var buf bytes.Buffer
				io.Copy(&buf, r)
				s := buf.String()
				registrationStr := fmt.Sprintf("Register policy at `/_ilm/policy/%s", td.policyName)
				if td.printed {
					assert.Contains(t, s, registrationStr)
				} else {
					assert.NotContains(t, s, registrationStr)
				}
			}()

			info := beat.Info{Version: "7.0.0", IndexPrefix: "testbeat"}
			loader, err := NewStdoutLoader(info)
			require.NoError(t, err)

			loaded, err := loader.LoadPolicy(td.cfg)
			assert.Equal(t, td.printed, loaded)
			if td.err == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.err, fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}

		})
	}
}

type ES65 struct{}

func (es *ES65) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}
func (es *ES65) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	return 200, []byte{}, nil
}
func (es *ES65) GetVersion() common.Version { return *common.MustNewVersion("6.5.0") }

type ES66 struct{}

func (es *ES66) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}
func (es *ES66) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	if method == "HEAD" {
		return 404, nil, nil
	}
	b := []byte(`{"features":{"ilm":{"enabled":true,"available":true}}}`)
	return 200, b, nil
}
func (es *ES66) GetVersion() common.Version { return *common.MustNewVersion("6.6.0") }

type ES66Disabled struct{}

func (es *ES66Disabled) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}
func (es *ES66Disabled) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	return 400, []byte{}, nil
}
func (es *ES66Disabled) GetVersion() common.Version { return *common.MustNewVersion("6.6.0") }

type ES66Error struct{}

func (es *ES66Error) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}
func (es *ES66Error) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	if strings.HasPrefix(path, "/_ilm/policy") {
		return 404, []byte{}, errors.New("policy could not be created")
	}

	b := []byte(`{"features":{"ilm":{"enabled":true,"available":true}}}`)
	return 200, b, nil
}

func (es *ES66Error) GetVersion() common.Version { return *common.MustNewVersion("6.6.0") }
