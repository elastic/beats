package kernel

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	data := `
kernel.audit_rules: |
  # Comments and empty lines are ignored.
  -w /etc/passwd -p wa -k auth

  -a always,exit -F arch=b64 -S execve -k exec`

	config, err := parseConfig(t, data)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := config.rules()
	if err != nil {
		t.Fatal()
	}
	assert.EqualValues(t, []string{
		"-w /etc/passwd -p wa -k auth",
		"-a always,exit -F arch=b64 -S execve -k exec",
	}, commands(rules))
}

func TestConfigValidateWithError(t *testing.T) {
	data := `
kernel.audit_rules: |
  -x bad -F flag
  -a always,exit -w /etc/passwd
  -a always,exit -F arch=b64 -S fake -k exec`

	_, err := parseConfig(t, data)
	if err == nil {
		t.Fatal("expected error")
	}
	t.Log(err)
}

func TestConfigValidateWithDuplicates(t *testing.T) {
	data := `
kernel.audit_rules: |
  -w /etc/passwd -p rwxa -k auth
  -w /etc/passwd -k auth`

	_, err := parseConfig(t, data)
	if err == nil {
		t.Fatal("expected error")
	}
	t.Log(err)
}

func TestConfigValidateFailureMode(t *testing.T) {
	config := defaultConfig
	config.FailureMode = "boom"
	err := config.Validate()
	assert.Error(t, err)
	t.Log(err)
}

func parseConfig(t testing.TB, yaml string) (Config, error) {
	c, err := common.NewConfigWithYAML([]byte(yaml), "")
	if err != nil {
		t.Fatal(err)
	}

	config := defaultConfig
	err = c.Unpack(&config)
	return config, err
}

func commands(rules []auditRule) []string {
	var cmds []string
	for _, r := range rules {
		cmds = append(cmds, r.flags)
	}
	return cmds
}
