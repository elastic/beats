// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/kibana"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

const (
	defaultKibanaHost = "http://kibana:5601"
	defaultESHost     = "http://elasticsearch:9200"
	defaultUsername   = "elastic"
	defaultPassword   = "changeme"
	defaultTokenName  = "Default"

	requestRetrySleep = 1 * time.Second // sleep 1 sec between retries for HTTP requests
	maxRequestRetries = 30              // maximum number of retries for HTTP requests
)

var (
	// Used to strip the appended ({uuid}) from the name of an enrollment token. This makes much easier for
	// a container to reference a token by name, without having to know what the generated UUID is for that name.
	tokenNameStrip = regexp.MustCompile(`\s\([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\)$`)
)

func newContainerCommand(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	return &cobra.Command{
		Hidden: true, // not exposed over help; used by container entrypoint only
		Use:    "container",
		Short:  "Bootstrap Elastic Agent to run inside of a container",
		Long: `This should only be used as an entrypoint for a container. This will prepare the Elastic Agent using
environment variables to run inside of the container.

The following actions are possible and grouped based on the actions.

* Preparing Kibana for Fleet
  This prepares the Fleet plugin that exists inside of Kibana. This must either be enabled here or done externally
  before Fleet Server will actually successfully start.

  KIBANA_FLEET_SETUP - set to 1 enables this setup
  KIBANA_FLEET_HOST - kibana host to enable Fleet on [$KIBANA_HOST]
  KIBANA_FLEET_USERNAME - kibana username to enable Fleet [$KIBANA_USERNAME]
  KIBANA_FLEET_PASSWORD - kibana password to enable Fleet [$KIBANA_PASSWORD]
  KIBANA_FLEET_CA - path to certificate authority to use with communicate with Kibana [$KIBANA_CA]

* Bootstrapping Fleet Server
  This bootstraps the Fleet Server to be run by this Elastic Agent. At least one Fleet Server is required in a Fleet
  deployment for other Elastic Agent to bootstrap.

  FLEET_SERVER_ENABLE - set to 1 enables bootstrapping of Fleet Server (forces FLEET_ENROLL enabled)
  FLEET_SERVER_ELASTICSEARCH_HOST - elasticsearch host for Fleet Server to communicate with [$ELASTICSEARCH_HOST]
  FLEET_SERVER_ELASTICSEARCH_USERNAME - elasticsearch username for Fleet Server [$ELASTICSEARCH_USERNAME]
  FLEET_SERVER_ELASTICSEARCH_PASSWORD - elasticsearch password for Fleet Server [$ELASTICSEARCH_PASSWORD]
  FLEET_SERVER_ELASTICSEARCH_CA - path to certificate authority to use with communicate with elasticsearch [$ELASTICSEARCH_CA]
  FLEET_SERVER_POLICY_NAME - name of policy for the Fleet Server to use for itself [$FLEET_TOKEN_POLICY_NAME]
  FLEET_SERVER_POLICY_ID - policy ID for Fleet Server to use for itself ("Default Fleet Server policy" used when undefined)
  FLEET_SERVER_HOST - binding host for Fleet Server HTTP (overrides the policy)
  FLEET_SERVER_PORT - binding port for Fleet Server HTTP (overrides the policy)
  FLEET_SERVER_CERT - path to certificate to use for HTTPS endpoint
  FLEET_SERVER_CERT_KEY - path to private key for certificate to use for HTTPS endpoint
  FLEET_SERVER_INSECURE_HTTP - expose Fleet Server over HTTP (not recommended; insecure)

* Elastic Agent Fleet Enrollment
  This enrolls the Elastic Agent into a Fleet Server. It is also possible to have this create a new enrollment token
  for this specific Elastic Agent.

  FLEET_ENROLL - set to 1 for enrollment to occur
  FLEET_URL - URL of the Fleet Server to enroll into
  FLEET_ENROLLMENT_TOKEN - token to use for enrollment
  FLEET_TOKEN_NAME - token name to use for fetching token from Kibana
  FLEET_TOKEN_POLICY_NAME - token policy name to use for fetching token from Kibana
  FLEET_CA - path to certificate authority to use with communicate with Fleet Server [$KIBANA_CA]
  FLEET_INSECURE - communicate with Fleet with either insecure HTTP or un-verified HTTPS
  KIBANA_FLEET_HOST - kibana host to enable create enrollment token on [$KIBANA_HOST]
  KIBANA_FLEET_USERNAME - kibana username to create enrollment token [$KIBANA_USERNAME]
  KIBANA_FLEET_PASSWORD - kibana password to create enrollment token [$KIBANA_PASSWORD]

The following environment variables are provided as a convenience to prevent a large number of environment variable to
be used when the same credentials will be used across all the possible actions above.

  ELASTICSEARCH_HOST - elasticsearch host [http://elasticsearch:9200]
  ELASTICSEARCH_USERNAME - elasticsearch username [elastic]
  ELASTICSEARCH_PASSWORD - elasticsearch password [changeme]
  ELASTICSEARCH_CA - path to certificate authority to use with communicate with elasticsearch
  KIBANA_HOST - kibana host [http://kibana:5601]
  KIBANA_USERNAME - kibana username [$ELASTICSEARCH_USERNAME]
  KIBANA_PASSWORD - kibana password [$ELASTICSEARCH_PASSWORD]
  KIBANA_CA - path to certificate authority to use with communicate with Kibana [$ELASTICSEARCH_CA]

By default when this command starts it will check for an existing fleet.yml. If that file already exists then
all the above actions will be skipped, because the Elastic Agent has already been enrolled. To ensure that enrollment
occurs on every start of the container set FLEET_FORCE to 1.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := containerCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func containerCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	var err error
	var client *kibana.Client
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	_, err = os.Stat(paths.AgentConfigFile())
	if !os.IsNotExist(err) && !envBool("FLEET_FORCE") {
		// already enrolled, just run the standard run
		return run(flags, streams)
	}

	// Remove FLEET_SETUP in 8.x
	// The FLEET_SETUP environment variable boolean is a fallback to the old name. The name was updated to
	// reflect that its setting up Fleet in Kibana versus setting up Fleet Server.
	if envBool("KIBANA_FLEET_SETUP", "FLEET_SETUP") {
		client, err = kibanaClient()
		if err != nil {
			return err
		}
		fmt.Fprintf(streams.Out, "Performing setup of Fleet in Kibana\n")
		err = kibanaSetup(client, streams)
		if err != nil {
			return err
		}
	}
	if envBool("FLEET_ENROLL", "FLEET_SERVER_ENABLE") {
		if client == nil {
			client, err = kibanaClient()
			if err != nil {
				return err
			}
		}
		var policy *kibanaPolicy
		token := envWithDefault("", "FLEET_ENROLLMENT_TOKEN")
		if token == "" {
			policy, err = kibanaFetchPolicy(client, streams)
			if err != nil {
				return err
			}
			token, err = kibanaFetchToken(client, policy, streams)
			if err != nil {
				return err
			}
		}
		policyID := ""
		if policy != nil {
			policyID = policy.ID
		}
		cmdArgs, err := buildEnrollArgs(token, policyID)
		if err != nil {
			return err
		}
		enroll := exec.Command(executable, cmdArgs...)
		enroll.Stdout = os.Stdout
		enroll.Stderr = os.Stderr
		err = enroll.Start()
		if err != nil {
			return errors.New("failed to execute enrollment command", err)
		}
		err = enroll.Wait()
		if err != nil {
			return errors.New("enrollment failed", err)
		}
	}

	return run(flags, streams)
}

func buildEnrollArgs(token string, policyID string) ([]string, error) {
	args := []string{"enroll", "-f"}
	if envBool("FLEET_SERVER_ENABLE") {
		connStr, err := buildFleetServerConnStr()
		if err != nil {
			return nil, err
		}
		args = append(args, "--fleet-server", connStr)
		if policyID == "" {
			policyID = envWithDefault("", "FLEET_SERVER_POLICY_ID")
		}
		if policyID != "" {
			args = append(args, "--fleet-server-policy", policyID)
		}
		ca := envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA", "ELASTICSEARCH_CA")
		if ca != "" {
			args = append(args, "--fleet-server-elasticsearch-ca", ca)
		}
		host := envWithDefault("", "FLEET_SERVER_HOST")
		if host != "" {
			args = append(args, "--fleet-server-host", host)
		}
		port := envWithDefault("", "FLEET_SERVER_PORT")
		if port != "" {
			args = append(args, "--fleet-server-port", port)
		}
		cert := envWithDefault("", "FLEET_SERVER_CERT")
		if cert != "" {
			args = append(args, "--fleet-server-cert", cert)
		}
		certKey := envWithDefault("", "FLEET_SERVER_CERT_KEY")
		if certKey != "" {
			args = append(args, "--fleet-server-cert-key", certKey)
		}
		if envBool("FLEET_SERVER_INSECURE_HTTP") {
			args = append(args, "--fleet-server-insecure-http")
			args = append(args, "--insecure")
		}
	} else {
		url := envWithDefault("", "FLEET_URL")
		if url == "" {
			return nil, errors.New("FLEET_URL is required when FLEET_ENROLL is true without FLEET_SERVER_ENABLE")
		}
		args = append(args, "--url", url)
		if envBool("FLEET_INSECURE") {
			args = append(args, "--insecure")
		}
		ca := envWithDefault("", "FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA")
		if ca != "" {
			args = append(args, "--certificate-authorities", ca)
		}
	}
	args = append(args, "--enrollment-token", token)
	return args, nil
}

func buildFleetServerConnStr() (string, error) {
	host := envWithDefault(defaultESHost, "FLEET_SERVER_ELASTICSEARCH_HOST", "ELASTICSEARCH_HOST")
	username := envWithDefault(defaultUsername, "FLEET_SERVER_ELASTICSEARCH_USERNAME", "ELASTICSEARCH_USERNAME")
	password := envWithDefault(defaultPassword, "FLEET_SERVER_ELASTICSEARCH_PASSWORD", "ELASTICSEARCH_PASSWORD")
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	path := ""
	if u.Path != "" {
		path += "/" + strings.TrimLeft(u.Path, "/")
	}
	return fmt.Sprintf("%s://%s:%s@%s%s", u.Scheme, username, password, u.Host, path), nil
}

func kibanaSetup(client *kibana.Client, streams *cli.IOStreams) error {
	err := performPOST(client, "/api/fleet/setup", streams.Err, "Kibana Fleet setup")
	if err != nil {
		return err
	}
	err = performPOST(client, "/api/fleet/agents/setup", streams.Err, "Kibana Fleet Agents setup")
	if err != nil {
		return err
	}
	return nil
}

func kibanaFetchPolicy(client *kibana.Client, streams *cli.IOStreams) (*kibanaPolicy, error) {
	var policies kibanaPolicies
	err := performGET(client, "/api/fleet/agent_policies", &policies, streams.Err, "Kibana fetch policy")
	if err != nil {
		return nil, err
	}
	return findPolicy(policies.Items)
}

func kibanaFetchToken(client *kibana.Client, policy *kibanaPolicy, streams *cli.IOStreams) (string, error) {
	var keys kibanaAPIKeys
	err := performGET(client, "/api/fleet/enrollment-api-keys", &keys, streams.Err, "Kibana fetch token")
	if err != nil {
		return "", err
	}
	key, err := findKey(keys.List, policy)
	if err != nil {
		return "", err
	}
	var keyDetail kibanaAPIKeyDetail
	err = performGET(client, fmt.Sprintf("/api/fleet/enrollment-api-keys/%s", key.ID), &keyDetail, streams.Err, "Kibana fetch token detail")
	if err != nil {
		return "", err
	}
	return keyDetail.Item.APIKey, nil
}

func kibanaClient() (*kibana.Client, error) {
	host := envWithDefault(defaultKibanaHost, "KIBANA_FLEET_HOST", "KIBANA_HOST")
	username := envWithDefault(defaultUsername, "KIBANA_FLEET_USERNAME", "KIBANA_USERNAME", "ELASTICSEARCH_USERNAME")
	password := envWithDefault(defaultPassword, "KIBANA_FLEET_PASSWORD", "KIBANA_PASSWORD", "ELASTICSEARCH_PASSWORD")

	var tls *tlscommon.Config
	ca := envWithDefault("", "KIBANA_FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA")
	if ca != "" {
		tls = &tlscommon.Config{
			CAs: []string{ca},
		}
	}
	return kibana.NewClientWithConfig(&kibana.ClientConfig{
		Host:          host,
		Username:      username,
		Password:      password,
		IgnoreVersion: true,
		TLS:           tls,
	})
}

func findPolicy(policies []kibanaPolicy) (*kibanaPolicy, error) {
	fleetServerEnabled := envBool("FLEET_SERVER_ENABLE")
	policyName := envWithDefault("", "FLEET_TOKEN_POLICY_NAME")
	if fleetServerEnabled {
		policyName = envWithDefault("", "FLEET_SERVER_POLICY_NAME", "FLEET_TOKEN_POLICY_NAME")
	}
	for _, policy := range policies {
		if policy.Status != "active" {
			continue
		}
		if policyName != "" {
			if policyName == policy.Name {
				return &policy, nil
			}
		} else if fleetServerEnabled {
			if policy.IsDefaultFleetServer {
				return &policy, nil
			}
		} else {
			if policy.IsDefault {
				return &policy, nil
			}
		}
	}
	return nil, fmt.Errorf(`unable to find policy named "%s"`, policyName)
}

func findKey(keys []kibanaAPIKey, policy *kibanaPolicy) (*kibanaAPIKey, error) {
	tokenName := envWithDefault(defaultTokenName, "FLEET_TOKEN_NAME")
	for _, key := range keys {
		name := strings.TrimSpace(tokenNameStrip.ReplaceAllString(key.Name, ""))
		if name == tokenName && key.PolicyID == policy.ID {
			return &key, nil
		}
	}
	return nil, fmt.Errorf(`unable to find enrollment token named "%s" in policy "%s"`, tokenName, policy.Name)
}

func envWithDefault(def string, keys ...string) string {
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok {
			return val
		}
	}
	return def
}

func envBool(keys ...string) bool {
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok && isTrue(val) {
			return true
		}
	}
	return false
}

func isTrue(val string) bool {
	trueVals := []string{"1", "true", "yes", "y"}
	val = strings.ToLower(val)
	for _, v := range trueVals {
		if val == v {
			return true
		}
	}
	return false
}

func performGET(client *kibana.Client, path string, response interface{}, writer io.Writer, msg string) error {
	var lastErr error
	for i := 0; i < maxRequestRetries; i++ {
		code, result, err := client.Connection.Request("GET", path, nil, nil, nil)
		if err != nil || code != 200 {
			err = fmt.Errorf("http GET request to %s%s fails: %v. Response: %s",
				client.Connection.URL, path, err, truncateString(result))
			fmt.Fprintf(writer, "%s failed: %s\n", msg, err)
			<-time.After(requestRetrySleep)
			continue
		}
		if response == nil {
			return nil
		}
		return json.Unmarshal(result, response)
	}
	return lastErr
}

func performPOST(client *kibana.Client, path string, writer io.Writer, msg string) error {
	var lastErr error
	for i := 0; i < maxRequestRetries; i++ {
		code, result, err := client.Connection.Request("POST", path, nil, nil, nil)
		if err != nil || code >= 400 {
			err = fmt.Errorf("http POST request to %s%s fails: %v. Response: %s",
				client.Connection.URL, path, err, truncateString(result))
			lastErr = err
			fmt.Fprintf(writer, "%s failed: %s\n", msg, err)
			<-time.After(requestRetrySleep)
			continue
		}
		return nil
	}
	return lastErr
}

func truncateString(b []byte) string {
	const maxLength = 250
	runes := bytes.Runes(b)
	if len(runes) > maxLength {
		runes = append(runes[:maxLength], []rune("... (truncated)")...)
	}

	return strings.Replace(string(runes), "\n", " ", -1)
}

type kibanaPolicy struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Status               string `json:"status"`
	IsDefault            bool   `json:"is_default"`
	IsDefaultFleetServer bool   `json:"is_default_fleet_server"`
}

type kibanaPolicies struct {
	Items []kibanaPolicy `json:"items"`
}

type kibanaAPIKey struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	PolicyID string `json:"policy_id"`
	APIKey   string `json:"api_key"`
}

type kibanaAPIKeys struct {
	List []kibanaAPIKey `json:"list"`
}

type kibanaAPIKeyDetail struct {
	Item kibanaAPIKey `json:"item"`
}
