// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/kibana"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install/tar"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	requestRetrySleepEnv     = "KIBANA_REQUEST_RETRY_SLEEP"
	maxRequestRetriesEnv     = "KIBANA_REQUEST_RETRY_COUNT"
	defaultRequestRetrySleep = "1s"                             // sleep 1 sec between retries for HTTP requests
	defaultMaxRequestRetries = "30"                             // maximum number of retries for HTTP requests
	defaultStateDirectory    = "/usr/share/elastic-agent/state" // directory that will hold the state data
)

var (
	// Used to strip the appended ({uuid}) from the name of an enrollment token. This makes much easier for
	// a container to reference a token by name, without having to know what the generated UUID is for that name.
	tokenNameStrip = regexp.MustCompile(`\s\([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\)$`)
)

func newContainerCommand(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := cobra.Command{
		Hidden: true, // not exposed over help; used by container entrypoint only
		Use:    "container",
		Short:  "Bootstrap Elastic Agent to run inside of a container",
		Long: `This should only be used as an entrypoint for a container. This will prepare the Elastic Agent using
environment variables to run inside of the container.

The following actions are possible and grouped based on the actions.

* Elastic Agent Fleet Enrollment
  This enrolls the Elastic Agent into a Fleet Server. It is also possible to have this create a new enrollment token
  for this specific Elastic Agent.

  FLEET_ENROLL - set to 1 for enrollment into fleet-server. If not set, Elastic Agent is run in standalone mode.
  FLEET_URL - URL of the Fleet Server to enroll into
  FLEET_ENROLLMENT_TOKEN - token to use for enrollment. This is not needed in case FLEET_SERVER_ENABLED and FLEET_ENROLL is set. Then the token is fetched from Kibana.
  FLEET_CA - path to certificate authority to use with communicate with Fleet Server [$KIBANA_CA]
  FLEET_INSECURE - communicate with Fleet with either insecure HTTP or unverified HTTPS

  The following vars are need in the scenario that Elastic Agent should automatically fetch its own token.

  KIBANA_FLEET_HOST - kibana host to enable create enrollment token on [$KIBANA_HOST]
  FLEET_TOKEN_NAME - token name to use for fetching token from Kibana. This requires Kibana configs to be set.
  FLEET_TOKEN_POLICY_NAME - token policy name to use for fetching token from Kibana. This requires Kibana configs to be set.

* Bootstrapping Fleet Server
  This bootstraps the Fleet Server to be run by this Elastic Agent. At least one Fleet Server is required in a Fleet
  deployment for other Elastic Agent to bootstrap. In case the Elastic Agent is run without fleet-server. These variables
  are not needed.

  If FLEET_SERVER_ENABLE and FLEET_ENROLL is set but no FLEET_ENROLLMENT_TOKEN, the token is automatically fetched from Kibana.

  FLEET_SERVER_ENABLE - set to 1 enables bootstrapping of Fleet Server inside Elastic Agent (forces FLEET_ENROLL enabled)
  FLEET_SERVER_ELASTICSEARCH_HOST - elasticsearch host for Fleet Server to communicate with [$ELASTICSEARCH_HOST]
  FLEET_SERVER_ELASTICSEARCH_CA - path to certificate authority to use with communicate with elasticsearch [$ELASTICSEARCH_CA]
  FLEET_SERVER_ELASTICSEARCH_CA_TRUSTED_FINGERPRINT - The sha-256 fingerprint value of the certificate authority to trust
  FLEET_SERVER_ELASTICSEARCH_INSECURE - disables cert validation for communication with Elasticsearch
  FLEET_SERVER_SERVICE_TOKEN - service token to use for communication with elasticsearch
  FLEET_SERVER_POLICY_ID - policy ID for Fleet Server to use for itself ("Default Fleet Server policy" used when undefined)
  FLEET_SERVER_HOST - binding host for Fleet Server HTTP (overrides the policy). By default this is 0.0.0.0.
  FLEET_SERVER_PORT - binding port for Fleet Server HTTP (overrides the policy)
  FLEET_SERVER_CERT - path to certificate to use for HTTPS endpoint
  FLEET_SERVER_CERT_KEY - path to private key for certificate to use for HTTPS endpoint
  FLEET_SERVER_INSECURE_HTTP - expose Fleet Server over HTTP (not recommended; insecure)

* Preparing Kibana for Fleet
  This prepares the Fleet plugin that exists inside of Kibana. This must either be enabled here or done externally
  before Fleet Server will actually successfully start. All the Kibana variables are not needed in case Elastic Agent
  should not setup Fleet. To manually trigger KIBANA_FLEET_SETUP navigate to Kibana -> Fleet -> Agents and enabled it.

  KIBANA_FLEET_SETUP - set to 1 enables the setup of Fleet in Kibana by Elastic Agent. This was previously FLEET_SETUP.
  KIBANA_FLEET_HOST - Kibana host accessible from fleet-server. [$KIBANA_HOST]
  KIBANA_FLEET_USERNAME - kibana username to service token [$KIBANA_USERNAME]
  KIBANA_FLEET_PASSWORD - kibana password to service token [$KIBANA_PASSWORD]
  KIBANA_FLEET_CA - path to certificate authority to use with communicate with Kibana [$KIBANA_CA]
  KIBANA_REQUEST_RETRY_SLEEP - specifies sleep duration taken when agent performs a request to kibana [default 1s]
  KIBANA_REQUEST_RETRY_COUNT - specifies number of retries agent performs when executing a request to kibana [default 30]

The following environment variables are provided as a convenience to prevent a large number of environment variable to
be used when the same credentials will be used across all the possible actions above.

  ELASTICSEARCH_HOST - elasticsearch host [http://elasticsearch:9200]
  ELASTICSEARCH_USERNAME - elasticsearch username [elastic]
  ELASTICSEARCH_PASSWORD - elasticsearch password [changeme]
  ELASTICSEARCH_CA - path to certificate authority to use with communicate with elasticsearch
  KIBANA_HOST - kibana host [http://kibana:5601]
  KIBANA_FLEET_USERNAME - kibana username to enable Fleet [$ELASTICSEARCH_USERNAME]
  KIBANA_FLEET_PASSWORD - kibana password to enable Fleet [$ELASTICSEARCH_PASSWORD]
  KIBANA_CA - path to certificate authority to use with communicate with Kibana [$ELASTICSEARCH_CA]


By default when this command starts it will check for an existing fleet.yml. If that file already exists then
all the above actions will be skipped, because the Elastic Agent has already been enrolled. To ensure that enrollment
occurs on every start of the container set FLEET_FORCE to 1.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := logContainerCmd(streams, c); err != nil {
				logError(streams, err)
				os.Exit(1)
			}
		},
	}
	return &cmd
}

func logError(streams *cli.IOStreams, err error) {
	fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
}

func logInfo(streams *cli.IOStreams, a ...interface{}) {
	fmt.Fprintln(streams.Out, a...)
}

func logContainerCmd(streams *cli.IOStreams, cmd *cobra.Command) error {
	logsPath := envWithDefault("", "LOGS_PATH")
	if logsPath != "" {
		// log this entire command to a file as well as to the passed streams
		if err := os.MkdirAll(logsPath, 0755); err != nil {
			return fmt.Errorf("preparing LOGS_PATH(%s) failed: %s", logsPath, err)
		}
		logPath := filepath.Join(logsPath, "elastic-agent-startup.log")
		w, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("opening startup log(%s) failed: %s", logPath, err)
		}
		defer w.Close()
		streams.Out = io.MultiWriter(streams.Out, w)
		streams.Err = io.MultiWriter(streams.Out, w)
	}
	return containerCmd(streams, cmd)
}

func containerCmd(streams *cli.IOStreams, cmd *cobra.Command) error {
	// set paths early so all action below use the defined paths
	if err := setPaths("", "", "", true); err != nil {
		return err
	}

	elasticCloud := envBool("ELASTIC_AGENT_CLOUD")
	// if not in cloud mode, always run the agent
	runAgent := !elasticCloud
	// create access configuration from ENV and config files
	cfg, err := defaultAccessConfig()
	if err != nil {
		return err
	}

	for _, f := range []string{"fleet-setup.yml", "credentials.yml"} {
		c, err := config.LoadFile(filepath.Join(paths.Config(), f))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("parsing config file(%s): %s", f, err)
		}
		if c != nil {
			err = c.Unpack(&cfg)
			if err != nil {
				return fmt.Errorf("unpacking config file(%s): %s", f, err)
			}
			// if in elastic cloud mode, only run the agent when configured
			runAgent = true
		}
	}

	// start apm-server legacy process when in cloud mode
	var wg sync.WaitGroup
	var apmProc *process.Info
	apmPath := os.Getenv("APM_SERVER_PATH")
	if elasticCloud {
		logInfo(streams, "Starting in elastic cloud mode")
		if elasticCloud && apmPath != "" {
			// run legacy APM Server as a daemon; send termination signal
			// to the main process if the daemon is stopped
			mainProc, err := os.FindProcess(os.Getpid())
			if err != nil {
				return errors.New(err, "finding current process")
			}
			if apmProc, err = runLegacyAPMServer(streams, apmPath); err != nil {
				return errors.New(err, "starting legacy apm-server")
			}
			wg.Add(1) // apm-server legacy process
			logInfo(streams, "Legacy apm-server daemon started.")
			go func() {
				if err := func() error {
					apmProcState, err := apmProc.Process.Wait()
					if err != nil {
						return err
					}
					if apmProcState.ExitCode() != 0 {
						return fmt.Errorf("apm-server process exited with %d", apmProcState.ExitCode())
					}
					return nil
				}(); err != nil {
					logError(streams, err)
				}

				wg.Done()
				// sending kill signal to current process (elastic-agent)
				logInfo(streams, "Initiate shutdown elastic-agent.")
				mainProc.Signal(syscall.SIGTERM)
			}()

			defer func() {
				if apmProc != nil {
					apmProc.Stop()
					logInfo(streams, "Initiate shutdown legacy apm-server.")
				}
			}()
		}
	}

	if runAgent {
		// run the main elastic-agent container command
		err = runContainerCmd(streams, cmd, cfg)
	}
	// wait until APM Server shut down
	wg.Wait()
	return err
}

func runContainerCmd(streams *cli.IOStreams, cmd *cobra.Command, cfg setupConfig) error {
	var err error
	var client *kibana.Client
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	_, err = os.Stat(paths.AgentConfigFile())
	if !os.IsNotExist(err) && !cfg.Fleet.Force {
		// already enrolled, just run the standard run
		return run(streams, logToStderr)
	}

	if cfg.Kibana.Fleet.Setup || cfg.FleetServer.Enable {
		err = ensureServiceToken(streams, &cfg)
		if err != nil {
			return err
		}
	}
	if cfg.Kibana.Fleet.Setup {
		client, err = kibanaClient(cfg.Kibana, cfg.Kibana.Headers)
		if err != nil {
			return err
		}

		logInfo(streams, "Performing setup of Fleet in Kibana\n")
		err = kibanaSetup(cfg, client, streams)
		if err != nil {
			return err
		}
	}
	if cfg.Fleet.Enroll {
		var policy *kibanaPolicy
		token := cfg.Fleet.EnrollmentToken
		if token == "" && !cfg.FleetServer.Enable {
			if client == nil {
				client, err = kibanaClient(cfg.Kibana, cfg.Kibana.Headers)
				if err != nil {
					return err
				}
			}
			policy, err = kibanaFetchPolicy(cfg, client, streams)
			if err != nil {
				return err
			}
			token, err = kibanaFetchToken(cfg, client, policy, streams, cfg.Fleet.TokenName)
			if err != nil {
				return err
			}
		}
		policyID := cfg.FleetServer.PolicyID
		if policy != nil {
			policyID = policy.ID
		}
		if policyID != "" {
			logInfo(streams, "Policy selected for enrollment: ", policyID)
		}

		cmdArgs, err := buildEnrollArgs(cfg, token, policyID)
		if err != nil {
			return err
		}
		enroll := exec.Command(executable, cmdArgs...)
		enroll.Stdout = streams.Out
		enroll.Stderr = streams.Err
		err = enroll.Start()
		if err != nil {
			return errors.New("failed to execute enrollment command", err)
		}
		err = enroll.Wait()
		if err != nil {
			return errors.New("enrollment failed", err)
		}
	}

	return run(streams, logToStderr)
}

// TokenResp is used to decode a response for generating a service token
type TokenResp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ensureServiceToken will ensure that the cfg specified has the service_token attributes filled.
//
// If no token is specified it will use the elasticsearch username/password to request a new token from Kibana
func ensureServiceToken(streams *cli.IOStreams, cfg *setupConfig) error {
	// There's already a service token
	if cfg.Kibana.Fleet.ServiceToken != "" || cfg.FleetServer.Elasticsearch.ServiceToken != "" {
		return nil
	}
	if cfg.Kibana.Fleet.Username == "" || cfg.Kibana.Fleet.Password == "" {
		return fmt.Errorf("username/password must be provided to retrieve service token")
	}

	logInfo(streams, "Requesting service_token from Kibana.")

	// Client is not passed in to this function because this function will use username/password and then
	// all the following clients will use the created service token.
	client, err := kibanaClient(cfg.Kibana, cfg.Kibana.Headers)
	if err != nil {
		return err
	}
	code, r, err := client.Connection.Request("POST", "/api/fleet/service-tokens", nil, nil, nil)
	if err != nil {
		return fmt.Errorf("request to get security token from Kibana failed: %w", err)
	}
	if code >= 400 {
		return fmt.Errorf("request to get security token from Kibana failed with status %d, body: %s", code, string(r))
	}
	t := TokenResp{}
	err = json.Unmarshal(r, &t)
	if err != nil {
		return fmt.Errorf("unable to decode response: %w", err)
	}
	logInfo(streams, "Created service_token named:", t.Name)
	cfg.Kibana.Fleet.ServiceToken = t.Value
	cfg.FleetServer.Elasticsearch.ServiceToken = t.Value
	return nil
}

func buildEnrollArgs(cfg setupConfig, token string, policyID string) ([]string, error) {
	args := []string{
		"enroll", "-f",
		"-c", paths.ConfigFile(),
		"--path.home", paths.Top(), // --path.home actually maps to paths.Top()
		"--path.config", paths.Config(),
		"--path.logs", paths.Logs(),
	}
	if paths.Downloads() != "" {
		args = append(args, "--path.downloads", paths.Downloads())
	}
	if paths.Install() != "" {
		args = append(args, "--path.install", paths.Install())
	}
	if !paths.IsVersionHome() {
		args = append(args, "--path.home.unversioned")
	}
	if cfg.FleetServer.Enable {
		connStr, err := buildFleetServerConnStr(cfg.FleetServer)
		if err != nil {
			return nil, err
		}
		args = append(args, "--fleet-server-es", connStr)
		if cfg.FleetServer.Elasticsearch.ServiceToken != "" {
			args = append(args, "--fleet-server-service-token", cfg.FleetServer.Elasticsearch.ServiceToken)
		}
		if policyID != "" {
			args = append(args, "--fleet-server-policy", policyID)
		}
		if cfg.FleetServer.Elasticsearch.CA != "" {
			args = append(args, "--fleet-server-es-ca", cfg.FleetServer.Elasticsearch.CA)
		}
		if cfg.FleetServer.Elasticsearch.CATrustedFingerprint != "" {
			args = append(args, "--fleet-server-es-ca-trusted-fingerprint", cfg.FleetServer.Elasticsearch.CATrustedFingerprint)
		}
		if cfg.FleetServer.Host != "" {
			args = append(args, "--fleet-server-host", cfg.FleetServer.Host)
		}
		if cfg.FleetServer.Port != "" {
			args = append(args, "--fleet-server-port", cfg.FleetServer.Port)
		}
		if cfg.FleetServer.Cert != "" {
			args = append(args, "--fleet-server-cert", cfg.FleetServer.Cert)
		}
		if cfg.FleetServer.CertKey != "" {
			args = append(args, "--fleet-server-cert-key", cfg.FleetServer.CertKey)
		}

		for k, v := range cfg.FleetServer.Headers {
			args = append(args, "--header", k+"="+v)
		}

		if cfg.Fleet.URL != "" {
			args = append(args, "--url", cfg.Fleet.URL)
		}
		if cfg.FleetServer.InsecureHTTP {
			args = append(args, "--fleet-server-insecure-http")
		}
		if cfg.FleetServer.InsecureHTTP || cfg.Fleet.Insecure {
			args = append(args, "--insecure")
		}
		if cfg.FleetServer.Elasticsearch.Insecure {
			args = append(args, "--fleet-server-es-insecure")
		}
		if cfg.FleetServer.Timeout != 0 {
			args = append(args, "--fleet-server-timeout")
			args = append(args, cfg.FleetServer.Timeout.String())
		}
	} else {
		if cfg.Fleet.URL == "" {
			return nil, errors.New("FLEET_URL is required when FLEET_ENROLL is true without FLEET_SERVER_ENABLE")
		}
		args = append(args, "--url", cfg.Fleet.URL)
		if cfg.Fleet.Insecure {
			args = append(args, "--insecure")
		}
	}
	if cfg.Fleet.CA != "" {
		args = append(args, "--certificate-authorities", cfg.Fleet.CA)
	}
	if token != "" {
		args = append(args, "--enrollment-token", token)
	}
	if cfg.Fleet.DaemonTimeout != 0 {
		args = append(args, "--daemon-timeout")
		args = append(args, cfg.Fleet.DaemonTimeout.String())
	}
	return args, nil
}

func buildFleetServerConnStr(cfg fleetServerConfig) (string, error) {
	u, err := url.Parse(cfg.Elasticsearch.Host)
	if err != nil {
		return "", err
	}
	path := ""
	if u.Path != "" {
		path += "/" + strings.TrimLeft(u.Path, "/")
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path), nil
}

func kibanaSetup(cfg setupConfig, client *kibana.Client, streams *cli.IOStreams) error {
	err := performPOST(cfg, client, "/api/fleet/setup", streams.Err, "Kibana Fleet setup")
	if err != nil {
		return err
	}
	err = performPOST(cfg, client, "/api/fleet/agents/setup", streams.Err, "Kibana Fleet Agents setup")
	if err != nil {
		return err
	}
	return nil
}

func kibanaFetchPolicy(cfg setupConfig, client *kibana.Client, streams *cli.IOStreams) (*kibanaPolicy, error) {
	var policies kibanaPolicies
	err := performGET(cfg, client, "/api/fleet/agent_policies", &policies, streams.Err, "Kibana fetch policy")
	if err != nil {
		return nil, err
	}
	packagePolicies, err := kibanaFetchPackagePolicies(cfg, client, streams)
	if err != nil {
		return nil, err
	}
	return findPolicy(cfg, policies.Items, packagePolicies)
}

func kibanaFetchPackagePolicies(cfg setupConfig, client *kibana.Client, streams *cli.IOStreams) (*packagePolicyResponse, error) {
	var packagePolicies kibanaPackagePolicies
	err := performGET(cfg, client, "/api/fleet/package_policies", &packagePolicies, streams.Err, "Kibana fetch package policies")
	if err != nil {
		return nil, err
	}
	return separatePackagePolicies(&packagePolicies), nil
}

func separatePackagePolicies(packagePolicies *kibanaPackagePolicies) *packagePolicyResponse {
	result := packagePolicyResponse{
		Fleet:    make(map[string]struct{}),
		NonFleet: make(map[string]struct{}),
	}
	for _, packagePolicy := range packagePolicies.Items {
		policyID := packagePolicy.PolicyID
		if packagePolicy.Package.Name == "fleet_server" {
			// if we have previously marked a policy as unmanaged, clear that marking
			if _, ok := result.NonFleet[policyID]; ok {
				delete(result.NonFleet, policyID)
			}

			result.Fleet[policyID] = struct{}{}
		} else {
			// only mark new policies as unmanaged
			if _, ok := result.Fleet[policyID]; !ok {
				result.NonFleet[policyID] = struct{}{}
			}
		}
	}
	return &result
}

func kibanaFetchToken(cfg setupConfig, client *kibana.Client, policy *kibanaPolicy, streams *cli.IOStreams, tokenName string) (string, error) {
	var keys kibanaAPIKeys
	err := performGET(cfg, client, "/api/fleet/enrollment-api-keys", &keys, streams.Err, "Kibana fetch token")
	if err != nil {
		return "", err
	}
	key, err := findKey(keys.List, policy, tokenName)
	if err != nil {
		return "", err
	}
	var keyDetail kibanaAPIKeyDetail
	err = performGET(cfg, client, fmt.Sprintf("/api/fleet/enrollment-api-keys/%s", key.ID), &keyDetail, streams.Err, "Kibana fetch token detail")
	if err != nil {
		return "", err
	}
	return keyDetail.Item.APIKey, nil
}

func kibanaClient(cfg kibanaConfig, headers map[string]string) (*kibana.Client, error) {
	var tls *tlscommon.Config
	if cfg.Fleet.CA != "" {
		tls = &tlscommon.Config{
			CAs: []string{cfg.Fleet.CA},
		}
	}

	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.TLS = tls

	return kibana.NewClientWithConfigDefault(&kibana.ClientConfig{
		Host:          cfg.Fleet.Host,
		Username:      cfg.Fleet.Username,
		Password:      cfg.Fleet.Password,
		ServiceToken:  cfg.Fleet.ServiceToken,
		IgnoreVersion: true,
		Transport:     transport,
		Headers:       headers,
	}, 0, "Elastic-Agent")
}

func findPolicy(cfg setupConfig, policies []kibanaPolicy, packagePolicies *packagePolicyResponse) (*kibanaPolicy, error) {
	policyID := ""
	policyName := cfg.Fleet.TokenPolicyName
	if cfg.FleetServer.Enable {
		policyID = cfg.FleetServer.PolicyID
	}
	var fallbackPolicy *kibanaPolicy
	for _, policy := range policies {
		if policyID != "" {
			if policyID == policy.ID {
				return &policy, nil
			}
		} else if policyName != "" {
			if policyName == policy.Name {
				return &policy, nil
			}
		} else if cfg.FleetServer.Enable {
			if _, ok := packagePolicies.Fleet[policy.ID]; ok && fallbackPolicy == nil {
				fallbackPolicy = &kibanaPolicy{}
				*fallbackPolicy = policy
			}
			if policy.ID == cfg.FleetServer.DefaultPolicyID {
				return &policy, nil
			}
		} else {
			if _, ok := packagePolicies.NonFleet[policy.ID]; ok {
				return &policy, nil
			}
		}
	}

	if fallbackPolicy != nil {
		return fallbackPolicy, nil
	}
	return nil, fmt.Errorf(`unable to find policy named "%s"`, policyName)
}

func findKey(keys []kibanaAPIKey, policy *kibanaPolicy, tokenName string) (*kibanaAPIKey, error) {
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

func envTimeout(keys ...string) time.Duration {
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok {
			dur, err := time.ParseDuration(val)
			if err == nil {
				return dur
			}
		}
	}
	return 0
}

func envMap(key string) map[string]string {
	m := make(map[string]string)
	prefix := key + "="
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		envVal := strings.TrimPrefix(env, prefix)

		keyValue := strings.SplitN(envVal, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		m[keyValue[0]] = keyValue[1]
	}

	return m
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

func performGET(cfg setupConfig, client *kibana.Client, path string, response interface{}, writer io.Writer, msg string) error {
	var lastErr error
	for i := 0; i < cfg.Kibana.RetryMaxCount; i++ {
		code, result, err := client.Connection.Request("GET", path, nil, nil, nil)
		if err != nil || code != 200 {
			err = fmt.Errorf("http GET request to %s%s fails: %v. Response: %s",
				client.Connection.URL, path, err, truncateString(result))
			fmt.Fprintf(writer, "%s failed: %s\n", msg, err)
			<-time.After(cfg.Kibana.RetrySleepDuration)
			continue
		}
		if response == nil {
			return nil
		}
		return json.Unmarshal(result, response)
	}
	return lastErr
}

func performPOST(cfg setupConfig, client *kibana.Client, path string, writer io.Writer, msg string) error {
	var lastErr error
	for i := 0; i < cfg.Kibana.RetryMaxCount; i++ {
		code, result, err := client.Connection.Request("POST", path, nil, nil, nil)
		if err != nil || code >= 400 {
			err = fmt.Errorf("http POST request to %s%s fails: %v. Response: %s",
				client.Connection.URL, path, err, truncateString(result))
			lastErr = err
			fmt.Fprintf(writer, "%s failed: %s\n", msg, err)
			<-time.After(cfg.Kibana.RetrySleepDuration)
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

// runLegacyAPMServer extracts the bundled apm-server from elastic-agent
// to path and runs it with args.
func runLegacyAPMServer(streams *cli.IOStreams, path string) (*process.Info, error) {
	name := "apm-server"
	logInfo(streams, "Preparing apm-server for legacy mode.")
	cfg := artifact.DefaultConfig()

	logInfo(streams, fmt.Sprintf("Extracting apm-server into install directory %s.", path))
	installer, err := tar.NewInstaller(cfg)
	if err != nil {
		return nil, errors.New(err, "creating installer")
	}
	spec := program.Spec{Name: name, Cmd: name, Artifact: name}
	version := release.Version()
	if release.Snapshot() {
		version = fmt.Sprintf("%s-SNAPSHOT", version)
	}
	// Extract the bundled apm-server into the APM_SERVER_PATH
	if err := installer.Install(context.Background(), spec, version, path); err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("installing %s (%s) from %s to %s", spec.Name, version, cfg.TargetDirectory, path))
	}
	// Get the apm-server directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("reading directory %s", path))
	}
	if len(files) != 1 || !files[0].IsDir() {
		return nil, errors.New("expected one directory")
	}
	apmDir := filepath.Join(path, files[0].Name())
	// Start apm-server process respecting path ENVs
	apmBinary := filepath.Join(apmDir, spec.Cmd)
	log, err := logger.New("apm-server", false)
	if err != nil {
		return nil, err
	}
	// add APM Server specific configuration
	var args []string
	addEnv := func(arg, env string) {
		if v := os.Getenv(env); v != "" {
			args = append(args, arg, v)
		}
	}
	addSettingEnv := func(arg, env string) {
		if v := os.Getenv(env); v != "" {
			args = append(args, "-E", fmt.Sprintf("%v=%v", arg, v))
		}
	}

	addEnv("--path.home", "HOME_PATH")
	addEnv("--path.config", "CONFIG_PATH")
	addEnv("--path.data", "DATA_PATH")
	addEnv("--path.logs", "LOGS_PATH")
	addEnv("--httpprof", "HTTPPROF")
	addSettingEnv("gc_percent", "APMSERVER_GOGC")
	logInfo(streams, "Starting legacy apm-server daemon as a subprocess.")
	return process.Start(log, apmBinary, nil, os.Geteuid(), os.Getegid(), args)
}

func logToStderr(cfg *configuration.Configuration) {
	logsPath := envWithDefault("", "LOGS_PATH")
	if logsPath == "" {
		// when no LOGS_PATH defined the container should log to stderr
		cfg.Settings.LoggingConfig.ToStderr = true
		cfg.Settings.LoggingConfig.ToFiles = false
	}
}

func setPaths(statePath, configPath, logsPath string, writePaths bool) error {
	statePath = envWithDefault(statePath, "STATE_PATH")
	if statePath == "" {
		statePath = defaultStateDirectory
	}
	topPath := filepath.Join(statePath, "data")
	configPath = envWithDefault(configPath, "CONFIG_PATH")
	if configPath == "" {
		configPath = statePath
	}
	// ensure that the directory and sub-directory data exists
	if err := os.MkdirAll(topPath, 0755); err != nil {
		return fmt.Errorf("preparing STATE_PATH(%s) failed: %s", statePath, err)
	}
	// ensure that the elastic-agent.yml exists in the state directory or if given in the config directory
	baseConfig := filepath.Join(configPath, paths.DefaultConfigName)
	if _, err := os.Stat(baseConfig); os.IsNotExist(err) {
		if err := copyFile(baseConfig, paths.ConfigFile(), 0); err != nil {
			return err
		}
	}
	// sync the downloads to the data directory
	destDownloads := filepath.Join(statePath, "data", "downloads")
	if err := syncDir(paths.Downloads(), destDownloads); err != nil {
		return fmt.Errorf("syncing download directory to STATE_PATH(%s) failed: %s", statePath, err)
	}
	originalInstall := paths.Install()
	originalTop := paths.Top()
	paths.SetTop(topPath)
	paths.SetConfig(configPath)
	// when custom top path is provided the home directory is not versioned
	paths.SetVersionHome(false)
	// install path stays on container default mount (otherwise a bind mounted directory could have noexec set)
	paths.SetInstall(originalInstall)
	// set LOGS_PATH is given
	logsPath = envWithDefault(logsPath, "LOGS_PATH")
	if logsPath != "" {
		paths.SetLogs(logsPath)
		// ensure that the logs directory exists
		if err := os.MkdirAll(filepath.Join(logsPath), 0755); err != nil {
			return fmt.Errorf("preparing LOGS_PATH(%s) failed: %s", logsPath, err)
		}
	}
	// persist the paths so other commands in the container will use the correct paths
	if writePaths {
		if err := writeContainerPaths(originalTop, statePath, configPath, logsPath); err != nil {
			return err
		}
	}
	return nil
}

type containerPaths struct {
	StatePath  string `config:"state_path" yaml:"state_path"`
	ConfigPath string `config:"state_path" yaml:"config_path,omitempty"`
	LogsPath   string `config:"state_path" yaml:"logs_path,omitempty"`
}

func writeContainerPaths(original, statePath, configPath, logsPath string) error {
	pathFile := filepath.Join(original, "container-paths.yml")
	fp, err := os.Create(pathFile)
	if err != nil {
		return fmt.Errorf("failed creating %s: %s", pathFile, err)
	}
	b, err := yaml.Marshal(containerPaths{
		StatePath:  statePath,
		ConfigPath: configPath,
		LogsPath:   logsPath,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal for %s: %s", pathFile, err)
	}
	_, err = fp.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write %s: %s", pathFile, err)
	}
	return nil
}

func tryContainerLoadPaths() error {
	pathFile := filepath.Join(paths.Top(), "container-paths.yml")
	_, err := os.Stat(pathFile)
	if os.IsNotExist(err) {
		// no container-paths.yml file exists, so nothing to do
		return nil
	}
	cfg, err := config.LoadFile(pathFile)
	if err != nil {
		return fmt.Errorf("failed to load %s: %s", pathFile, err)
	}
	var paths containerPaths
	err = cfg.Unpack(&paths)
	if err != nil {
		return fmt.Errorf("failed to unpack %s: %s", pathFile, err)
	}
	return setPaths(paths.StatePath, paths.ConfigPath, paths.LogsPath, false)
}

func syncDir(src string, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath := strings.TrimPrefix(path, src)
		if info.IsDir() {
			err = os.MkdirAll(filepath.Join(dest, relativePath), info.Mode())
			if err != nil {
				return err
			}
			return nil
		}
		return copyFile(filepath.Join(dest, relativePath), path, info.Mode())
	})
}

func copyFile(destPath string, srcPath string, mode os.FileMode) error {
	// if mode is unset; set to the same as the source file
	if mode == 0 {
		info, err := os.Stat(srcPath)
		if err == nil {
			// ignoring error because; os.Open will also error if the file cannot be stat'd
			mode = info.Mode()
		}
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dest, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	return err
}

type kibanaPackage struct {
	Name string `json:"name"`
}

type packagePolicyResponse struct {
	Fleet    map[string]struct{}
	NonFleet map[string]struct{}
}

type kibanaPackagePolicy struct {
	PolicyID string        `json:"policy_id"`
	Package  kibanaPackage `json:"package"`
}

type kibanaPackagePolicies struct {
	Items []kibanaPackagePolicy `json:"items"`
}

type kibanaPolicy struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	PackagePolicies []string `json:"package_policies"`
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

func envDurationWithDefault(defVal string, keys ...string) (time.Duration, error) {
	valStr := defVal
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok {
			valStr = val
			break
		}
	}

	return time.ParseDuration(valStr)
}

func envIntWithDefault(defVal string, keys ...string) (int, error) {
	valStr := defVal
	for _, key := range keys {
		val, ok := os.LookupEnv(key)
		if ok {
			valStr = val
			break
		}
	}

	return strconv.Atoi(valStr)
}
