package cloudid

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		cloudID           string
		expectedEsURL     string
		expectedKibanaURL string
	}{
		{
			cloudID:           "staging:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw==",
			expectedEsURL:     "https://cec6f261a74bf24ce33bb8811b84294f.us-east-1.aws.found.io:443",
			expectedKibanaURL: "https://c6c2ca6d042249af0cc7d7a9e9625743.us-east-1.aws.found.io:443",
		},
		{
			cloudID:           "dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw==",
			expectedEsURL:     "https://cec6f261a74bf24ce33bb8811b84294f.us-east-1.aws.found.io:443",
			expectedKibanaURL: "https://c6c2ca6d042249af0cc7d7a9e9625743.us-east-1.aws.found.io:443",
		},
		{
			cloudID:           ":dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw==",
			expectedEsURL:     "https://cec6f261a74bf24ce33bb8811b84294f.us-east-1.aws.found.io:443",
			expectedKibanaURL: "https://c6c2ca6d042249af0cc7d7a9e9625743.us-east-1.aws.found.io:443",
		},
		{
			cloudID:           "gcp-cluster:dXMtY2VudHJhbDEuZ2NwLmNsb3VkLmVzLmlvJDhhMDI4M2FmMDQxZjE5NWY3NzI5YmMwNGM2NmEwZmNlJDBjZDVjZDU2OGVlYmU1M2M4OWViN2NhZTViYWM4YjM3",
			expectedEsURL:     "https://8a0283af041f195f7729bc04c66a0fce.us-central1.gcp.cloud.es.io:443",
			expectedKibanaURL: "https://0cd5cd568eebe53c89eb7cae5bac8b37.us-central1.gcp.cloud.es.io:443",
		},
	}

	for _, test := range tests {
		esURL, kbURL, err := decodeCloudID(test.cloudID)
		assert.NoError(t, err, test.cloudID)

		assert.Equal(t, esURL, test.expectedEsURL, test.cloudID)
		assert.Equal(t, kbURL, test.expectedKibanaURL, test.cloudID)
	}
}

func TestDecodeError(t *testing.T) {
	tests := []struct {
		cloudID  string
		errorMsg string
	}{
		{
			cloudID:  "staging:garbagedXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw==",
			errorMsg: "base64 decoding failed",
		},
		{
			cloudID:  "dXMtY2VudHJhbDEuZ2NwLmNsb3VkLmVzLmlvJDhhMDI4M2FmMDQxZjE5NWY3NzI5YmMwNGM2NmEwZg==",
			errorMsg: "Expected at least 3 parts",
		},
	}

	for _, test := range tests {
		_, _, err := decodeCloudID(test.cloudID)
		assert.Error(t, err, test.cloudID)
		assert.Contains(t, err.Error(), test.errorMsg, test.cloudID)
	}
}

func TestOverwriteSettings(t *testing.T) {
	tests := []struct {
		name   string
		inCfg  map[string]interface{}
		outCfg map[string]interface{}
	}{
		{
			name: "No cloud-id specified, nothing should change",
			inCfg: map[string]interface{}{
				"output.elasticsearch.hosts": "localhost:9200",
			},
			outCfg: map[string]interface{}{
				"output.elasticsearch.hosts": "localhost:9200",
			},
		},
		{
			name: "cloudid realistic example",
			inCfg: map[string]interface{}{
				"output.elasticsearch.hosts": "localhost:9200",
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
				"cloud.auth":                 "elastic:changeme",
			},
			outCfg: map[string]interface{}{
				"output.elasticsearch.hosts":    []interface{}{"https://249f3af1f4eee24a84e3b401e68a1b2a.us-east-1.aws.found.io:443"},
				"output.elasticsearch.username": "elastic",
				"output.elasticsearch.password": "changeme",
				"setup.kibana.host":             "https://d4ac7559d4674b7c91abe10856d84304.us-east-1.aws.found.io:443",
				"cloud.id":                      "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
				"cloud.auth":                    "elastic:changeme",
			},
		},
		{
			name: "only cloudid specified",
			inCfg: map[string]interface{}{
				"output.elasticsearch.hosts": "localhost:9200",
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
			outCfg: map[string]interface{}{
				"output.elasticsearch.hosts": []interface{}{"https://249f3af1f4eee24a84e3b401e68a1b2a.us-east-1.aws.found.io:443"},
				"setup.kibana.host":          "https://d4ac7559d4674b7c91abe10856d84304.us-east-1.aws.found.io:443",
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
		},
		{
			name: "no output defined",
			inCfg: map[string]interface{}{
				"cloud.id": "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
			outCfg: map[string]interface{}{
				"output.elasticsearch.hosts": []interface{}{"https://249f3af1f4eee24a84e3b401e68a1b2a.us-east-1.aws.found.io:443"},
				"setup.kibana.host":          "https://d4ac7559d4674b7c91abe10856d84304.us-east-1.aws.found.io:443",
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
		},
		{
			name: "multiple hosts to overwrite",
			inCfg: map[string]interface{}{
				"output.elasticsearch.hosts": []string{"localhost:9200", "test", "test1"},
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
			outCfg: map[string]interface{}{
				"output.elasticsearch.hosts": []interface{}{"https://249f3af1f4eee24a84e3b401e68a1b2a.us-east-1.aws.found.io:443"},
				"setup.kibana.host":          "https://d4ac7559d4674b7c91abe10856d84304.us-east-1.aws.found.io:443",
				"cloud.id":                   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
			},
		},
	}

	for _, test := range tests {
		t.Logf("Executing test: %s", test.name)

		cfg, err := common.NewConfigFrom(test.inCfg)
		assert.NoError(t, err)

		err = OverwriteSettings(cfg)
		assert.NoError(t, err)

		var res map[string]interface{}
		err = cfg.Unpack(&res)
		assert.NoError(t, err)

		var expected map[string]interface{}
		expectedCfg, err := common.NewConfigFrom(test.outCfg)
		assert.NoError(t, err)
		err = expectedCfg.Unpack(&expected)
		assert.NoError(t, err)

		assert.Equal(t, res, expected)
	}
}

func TestOverwriteErrors(t *testing.T) {
	tests := []struct {
		name   string
		inCfg  map[string]interface{}
		errMsg string
	}{
		{
			name: "cloud.auth specified but cloud.id not",
			inCfg: map[string]interface{}{
				"cloud.auth": "elastic:changeme",
			},
			errMsg: "cloud.auth specified but cloud.id is empty",
		},
		{
			name: "invalid cloud.id",
			inCfg: map[string]interface{}{
				"cloud.id": "blah",
			},
			errMsg: "Error decoding cloud.id",
		},
		{
			name: "invalid cloud.auth",
			inCfg: map[string]interface{}{
				"cloud.id":   "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
				"cloud.auth": "blah",
			},
			errMsg: "cloud.auth setting doesn't contain `:`",
		},
		{
			name: "logstash output enabled",
			inCfg: map[string]interface{}{
				"cloud.id":              "cloudidtest:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyQyNDlmM2FmMWY0ZWVlMjRhODRlM2I0MDFlNjhhMWIyYSRkNGFjNzU1OWQ0Njc0YjdjOTFhYmUxMDg1NmQ4NDMwNA==",
				"output.logstash.hosts": "localhost:544",
			},
			errMsg: "The cloud.id setting enables the Elasticsearch output, but you already have the logstash output enabled",
		},
	}

	for _, test := range tests {
		t.Logf("Executing test: %s", test.name)

		cfg, err := common.NewConfigFrom(test.inCfg)
		assert.NoError(t, err)

		err = OverwriteSettings(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), test.errMsg)
	}
}
