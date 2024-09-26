package kafka

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestToOTelConfig(t *testing.T) {
	tcs := []struct {
		name     string
		cfgPath  string
		wantPath string
		checkErr func(*testing.T, error)
	}{
		{
			name:     "plain auth",
			cfgPath:  "testdata/plain-beats.yaml",
			wantPath: "testdata/plain-otel.yaml",
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "unexpected error")
			},
		},
		{
			name:     "plain sasl auth",
			cfgPath:  "testdata/plain-sasl-beats.yaml",
			wantPath: "testdata/plain-sasl-otel.yaml",
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "unexpected error")
			},
		},
		{
			name:     "plain kerberos auth",
			cfgPath:  "testdata/plain-kerberos-beats.yaml",
			wantPath: "testdata/plain-kerberos-otel.yaml",
			checkErr: func(t *testing.T, err error) {
				require.NoError(t, err, "unexpected error")
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfgFile, err := os.ReadFile(tc.cfgPath)
			require.NoError(t, err, "could not open config file")
			want, err := os.ReadFile(tc.wantPath)

			cfg := config.MustNewConfigFrom(string(cfgFile))

			got, err := ToOTelConfig(cfg)
			gotYAML, err := yaml.Marshal(got)
			require.NoError(t, err, "failed to marshal OTel config to YAML")

			assert.Equal(t, string(want), string(gotYAML))
		})
	}
}

func TestExtractSingleTopic(t *testing.T) {
	testCases := []struct {
		name      string
		tmpl      string
		want      string
		assertErr func(t *testing.T, err error)
	}{
		{
			name:      "Valid template with single attribute",
			tmpl:      "%{[some_field]}",
			want:      "some_field",
			assertErr: nil,
		},
		{
			name:      "Valid template with single attribute with subfield",
			tmpl:      "%{[some_field.subfield]}",
			want:      "some_field.subfield",
			assertErr: nil,
		},
		{
			name:      "Constant template",
			tmpl:      "constant_topic",
			want:      "constant_topic",
			assertErr: nil,
		},
		{
			name: "Template with multiple attributes",
			tmpl: "%{[.field1]}-%{[.field2]}",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "only one attribute supported")
			}},
		{
			name: "Template not just an attribute",
			tmpl: "prefix-%{[.field]}",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "topic template is more than just a event attribute")
			}},
		{
			name: "Invalid template",
			tmpl: "%{[.invalid_syntax",
			want: "",
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to compile topic template")
			}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := extractSingleTopic(tc.tmpl)
			if tc.assertErr == nil {
				tc.assertErr = func(t *testing.T, err error) {
					assert.NoError(t, err, "unexpected error")
				}
			}

			tc.assertErr(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
