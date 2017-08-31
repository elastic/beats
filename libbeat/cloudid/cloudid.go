// package cloudid contains functions for parsing the cloud.id and cloud.auth
// settings and modifying the configuration to take them into account.
package cloudid

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// OverwriteSettings modifies the received config object by overwriting the
// output.elasticsearch.hosts, output.elasticsearch.username, output.elasticsearch.password,
// setup.kibana.host settings based on values derived from the cloud.id and cloud.auth
// settings.
func OverwriteSettings(cfg *common.Config) error {

	cloudID, _ := cfg.String("cloud.id", -1)
	cloudAuth, _ := cfg.String("cloud.auth", -1)

	if cloudID == "" && cloudAuth == "" {
		// nothing to hack
		return nil
	}

	logp.Debug("cloudid", "cloud.id: %s, cloud.auth: %s", cloudID, cloudAuth)
	if cloudID == "" {
		return errors.New("cloud.auth specified but cloud.id is empty. Please specify both.")
	}

	// cloudID overwrites
	esURL, kibanaURL, err := decodeCloudID(cloudID)
	if err != nil {
		return errors.Errorf("Error decoding cloud.id: %v", err)
	}

	logp.Info("Setting Elasticsearch and Kibana URLs based on the cloud id: output.elasticsearch.hosts=%s and setup.kibana.host=%s", esURL, kibanaURL)

	esURLConfig, err := common.NewConfigFrom([]string{esURL})
	if err != nil {
		return err
	}

	// Before enabling the ES output, check that no other output is enabled
	tmp := struct {
		Output common.ConfigNamespace `config:"output"`
	}{}
	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}
	if out := tmp.Output; out.IsSet() && out.Name() != "elasticsearch" {
		return errors.Errorf("The cloud.id setting enables the Elasticsearch output, but you already have the %s output enabled in the config", out.Name())
	}

	err = cfg.SetChild("output.elasticsearch.hosts", -1, esURLConfig)
	if err != nil {
		return err
	}

	err = cfg.SetString("setup.kibana.host", -1, kibanaURL)
	if err != nil {
		return err
	}

	if cloudAuth != "" {
		// cloudAuth overwrites
		username, password, err := decodeCloudAuth(cloudAuth)
		if err != nil {
			return err
		}

		err = cfg.SetString("output.elasticsearch.username", -1, username)
		if err != nil {
			return err
		}

		err = cfg.SetString("output.elasticsearch.password", -1, password)
		if err != nil {
			return err
		}
	}

	return nil
}

// decodeCloudID decodes the cloud.id into elasticsearch-URL and kibana-URL
func decodeCloudID(cloudID string) (string, string, error) {

	// 1. Ignore anything before `:`.
	idx := strings.LastIndex(cloudID, ":")
	if idx >= 0 {
		cloudID = cloudID[idx+1:]
	}

	// 2. base64 decode
	decoded, err := base64.StdEncoding.DecodeString(cloudID)
	if err != nil {
		return "", "", errors.Wrapf(err, "base64 decoding failed on %s", cloudID)
	}

	// 3. separate based on `$`
	words := strings.Split(string(decoded), "$")
	if len(words) < 3 {
		return "", "", errors.Errorf("Expected at least 3 parts in %s", string(decoded))
	}

	// 4. form the URLs
	esURL := url.URL{Scheme: "https", Host: fmt.Sprintf("%s.%s:443", words[1], words[0])}
	kibanaURL := url.URL{Scheme: "https", Host: fmt.Sprintf("%s.%s:443", words[2], words[0])}

	return esURL.String(), kibanaURL.String(), nil
}

// decodeCloudAuth splits the cloud.auth into username and password.
func decodeCloudAuth(cloudAuth string) (string, string, error) {

	idx := strings.Index(cloudAuth, ":")
	if idx < 0 {
		return "", "", errors.New("cloud.auth setting doesn't contain `:` to split between username and password")
	}

	return cloudAuth[0:idx], cloudAuth[idx+1:], nil
}
