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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

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
	requestRetrySleep = 1 * time.Second // sleep 1 sec between retries for HTTP requests
	maxRequestRetries = 30              // maximum number of retries for HTTP requests
)

var (
	// Used to strip the appended ({uuid}) from the name of an enrollment token. This makes much easier for
	// a container to reference a token by name, without having to know what the generated UUID is for that name.
	tokenNameStrip = regexp.MustCompile(`\s\([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\)$`)
)

func newContainerCommand(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := cobra.Command{
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
			var err error
			if _, cloud := os.LookupEnv("ELASTIC_AGENT_CLOUD"); cloud {
				err = containerCloudCmd(streams, c, flags, args)
			} else {
				err = containerCmd(streams, c, flags, defaultAccessConfig())
			}
			if err != nil {
				logError(streams, err)
				os.Exit(1)
			}
		},
	}
	return &cmd
}

func logError(streams *cli.IOStreams, err error) {
	fmt.Fprintf(streams.Err, "Error: %v\n", err)
}

func logInfo(streams *cli.IOStreams, msg string) {
	fmt.Fprintln(streams.Out, msg)
}

func containerCloudCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	logInfo(streams, "Elastic Agent container in cloud mode")
	// sync main process and apm-server legacy process
	var wg sync.WaitGroup
	wg.Add(1) // main process (always running)
	mainProc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return errors.New(err, "finding current process")
	}
	var apmProc *process.Info
	// run legacy APM Server as a daemon; send termination signal
	// to the main process if the daemon is stopped
	apmPath := os.Getenv("APM_SERVER_PATH")
	if apmPath != "" {
		apmProc, err = runLegacyAPMServer(streams, apmPath, args)
		if err != nil {
			return errors.New(err, "starting legacy apm-server")
		}
		logInfo(streams, "Legacy apm-server daemon started.")
		wg.Add(1) // apm-server legacy process
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
	}
	// run Elastic Agent; send termination signal to the
	// legacy apm-server process if stopped
	go func() {
		if err := func() error {
			// create configuration for Elastic Agent
			cfg := defaultAccessConfig()
			if err := readYaml(filepath.Join(paths.Config(), "fleet-setup.yml"), &cfg); err != nil {
				return errors.New(err, "parsing fleet-setup.yml")
			}
			if err := readYaml(filepath.Join(paths.Config(), "credentials.yml"), &cfg); err != nil {
				return errors.New(err, "parsing credentials.yml")
			}
			return containerCmd(streams, cmd, flags, cfg)
		}(); err != nil {
			logError(streams, err)
		}
		wg.Done()
		// sending kill signal to APM Server
		if apmProc != nil {
			apmProc.Stop()
			logInfo(streams, "Initiate shutdown legacy apm-server.")
		}
	}()
	wg.Wait()
	return nil
}

func containerCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, cfg setupConfig) error {
	var err error
	var client *kibana.Client
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	_, err = os.Stat(paths.AgentConfigFile())
	if !os.IsNotExist(err) && !cfg.Fleet.Force {
		// already enrolled, just run the standard run
		return run(flags, streams)
	}

	if cfg.Kibana.Fleet.Setup {
		client, err = kibanaClient(cfg.Kibana)
		if err != nil {
			return err
		}
		fmt.Fprintf(streams.Out, "Performing setup of Fleet in Kibana\n")
		err = kibanaSetup(client, streams)
		if err != nil {
			return err
		}
	}
	if cfg.Fleet.Enroll {
		if client == nil {
			client, err = kibanaClient(cfg.Kibana)
			if err != nil {
				return err
			}
		}
		var policy *kibanaPolicy
		token := cfg.Fleet.EnrollmentToken
		if token == "" {
			policy, err = kibanaFetchPolicy(client, cfg, streams)
			if err != nil {
				return err
			}
			token, err = kibanaFetchToken(client, policy, streams, cfg.Fleet.TokenName)
			if err != nil {
				return err
			}
		}
		policyID := ""
		if policy != nil {
			policyID = policy.ID
		}
		cmdArgs, err := buildEnrollArgs(cfg, token, policyID)
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

func buildEnrollArgs(cfg setupConfig, token string, policyID string) ([]string, error) {
	args := []string{"enroll", "-f"}
	if cfg.FleetServer.Enable {
		connStr, err := buildFleetServerConnStr(cfg.FleetServer)
		if err != nil {
			return nil, err
		}
		args = append(args, "--fleet-server", connStr)
		if policyID == "" {
			policyID = cfg.FleetServer.PolicyID
		}
		if policyID != "" {
			args = append(args, "--fleet-server-policy", policyID)
		}
		if cfg.FleetServer.Elasticsearch.CA != "" {
			args = append(args, "--fleet-server-elasticsearch-ca", cfg.FleetServer.Elasticsearch.CA)
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
		if cfg.FleetServer.InsecureHTTP {
			args = append(args, "--fleet-server-insecure-http")
			args = append(args, "--insecure")
		}
	} else {
		if cfg.Fleet.URL == "" {
			return nil, errors.New("FLEET_URL is required when FLEET_ENROLL is true without FLEET_SERVER_ENABLE")
		}
		args = append(args, "--url", cfg.Fleet.URL)
		if cfg.Fleet.Insecure {
			args = append(args, "--insecure")
		}
		if cfg.Fleet.CA != "" {
			args = append(args, "--certificate-authorities", cfg.Fleet.CA)
		}
	}
	return append(args, "--enrollment-token", token), nil
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
	return fmt.Sprintf("%s://%s:%s@%s%s", u.Scheme, cfg.Elasticsearch.Username, cfg.Elasticsearch.Password, u.Host, path), nil
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

func kibanaFetchPolicy(client *kibana.Client, cfg setupConfig, streams *cli.IOStreams) (*kibanaPolicy, error) {
	var policies kibanaPolicies
	err := performGET(client, "/api/fleet/agent_policies", &policies, streams.Err, "Kibana fetch policy")
	if err != nil {
		return nil, err
	}
	return findPolicy(cfg, policies.Items)
}

func kibanaFetchToken(client *kibana.Client, policy *kibanaPolicy, streams *cli.IOStreams, tokenName string) (string, error) {
	var keys kibanaAPIKeys
	err := performGET(client, "/api/fleet/enrollment-api-keys", &keys, streams.Err, "Kibana fetch token")
	if err != nil {
		return "", err
	}
	key, err := findKey(keys.List, policy, tokenName)
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

func kibanaClient(cfg kibanaConfig) (*kibana.Client, error) {
	var tls *tlscommon.Config
	if cfg.Fleet.CA != "" {
		tls = &tlscommon.Config{
			CAs: []string{cfg.Fleet.CA},
		}
	}
	return kibana.NewClientWithConfig(&kibana.ClientConfig{
		Host:          cfg.Fleet.Host,
		Username:      cfg.Fleet.Username,
		Password:      cfg.Fleet.Password,
		IgnoreVersion: true,
		TLS:           tls,
	})
}

func findPolicy(cfg setupConfig, policies []kibanaPolicy) (*kibanaPolicy, error) {
	policyName := cfg.Fleet.TokenPolicyName
	if cfg.FleetServer.Enable {
		policyName = cfg.FleetServer.PolicyName
	}
	for _, policy := range policies {
		if policy.Status != "active" {
			continue
		}
		if policyName != "" {
			if policyName == policy.Name {
				return &policy, nil
			}
		} else if cfg.FleetServer.Enable {
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

// runLegacyAPMServer extracts the bundled apm-server from elastic-agent
// to path and runs it with args.
func runLegacyAPMServer(streams *cli.IOStreams, path string, args []string) (*process.Info, error) {
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
	// Extract the bundled apm-server binary into the APM_SERVER_PATH
	if err := installer.Install(context.Background(), spec, version, path); err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("installing %s (%s) from %s to %s", spec.Name, version, cfg.TargetDirectory, path))
	}

	// Start apm-server process respecting args
	logInfo(streams, "Starting legacy apm-server daemon as a subprocess.")
	pattern := filepath.Join(path, fmt.Sprintf("%s-%s-%s*", spec.Cmd, version, cfg.OS()), spec.Cmd)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("searching apm-server in %s", pattern))
	}
	if len(files) != 1 {
		return nil, errors.New("multiple apm-server versions installed")
	}
	f, err := filepath.Abs(files[0])
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("absPath for %s", files[0]))
	}
	log, err := logger.New("apm-server")
	if err != nil {
		return nil, err
	}
	// add APM Server specific configuration
	addEnv := func(arg, env string) {
		if v := os.Getenv(env); v != "" {
			args = append(args, arg, v)
		}
	}
	addEnv("--path.config", "APM_SERVER_CONFIG_PATH")
	addEnv("--path.data", "APM_SERVER_DATA_PATH")
	addEnv("--path.logs", "APM_SERVER_LOGS_PATH")
	addEnv("--httpprof", "APM_SERVER_HTTPPROF")
	return process.Start(log, f, nil, os.Geteuid(), os.Getegid(), args...)
}

func readYaml(f string, cfg *setupConfig) error {
	c, err := config.LoadFile(f)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return c.Unpack(cfg)
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

// setup configuration

type setupConfig struct {
	Fleet       fleetConfig       `config:"fleet"`
	FleetServer fleetServerConfig `config:"fleet_server"`
	Kibana      kibanaConfig      `config:"kibana"`
}

type elasticsearchConfig struct {
	CA       string `config:"ca"`
	Host     string `config:"host"`
	Username string `config:"username"`
	Password string `config:"password"`
}

type fleetConfig struct {
	CA              string `config:"ca"`
	Enroll          bool   `config:"enroll"`
	EnrollmentToken string `config:"enrollment_token"`
	Force           bool   `config:"force"`
	Insecure        bool   `config:"insecure"`
	TokenName       string `config:"token_name"`
	TokenPolicyName string `config:"token_policy_name"`
	URL             string `config:"url"`
}

type fleetServerConfig struct {
	Cert          string              `config:"cert"`
	CertKey       string              `config:"certKey"`
	Elasticsearch elasticsearchConfig `config:"elasticsearch"`
	Enable        bool                `config:"enable"`
	Host          string              `config:"host"`
	InsecureHTTP  bool                `config:"insecure_http"`
	PolicyID      string              `config:"policy_id"`
	PolicyName    string              `config:"policy_name"`
	Port          string              `config:"port"`
}

type kibanaConfig struct {
	Fleet kibanaFleetConfig `config:"fleet"`
}

type kibanaFleetConfig struct {
	CA       string `config:"ca"`
	Host     string `config:"host"`
	Password string `config:"password"`
	Setup    bool   `config:"setup"`
	Username string `config:"username"`
}

func defaultAccessConfig() setupConfig {
	return setupConfig{
		Fleet: fleetConfig{
			CA:              envWithDefault("", "FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			Enroll:          envBool("FLEET_ENROLL", "FLEET_SERVER_ENABLE"),
			EnrollmentToken: envWithDefault("", "FLEET_ENROLLMENT_TOKEN"),
			Force:           envBool("FLEET_FORCE"),
			Insecure:        envBool("FLEET_INSECURE"),
			TokenName:       envWithDefault("Default", "FLEET_TOKEN_NAME"),
			TokenPolicyName: envWithDefault("", "FLEET_TOKEN_POLICY_NAME"),
			URL:             envWithDefault("", "FLEET_URL"),
		},
		FleetServer: fleetServerConfig{
			Cert:    envWithDefault("", "FLEET_SERVER_CERT"),
			CertKey: envWithDefault("", "FLEET_SERVER_CERT_KEY"),
			Elasticsearch: elasticsearchConfig{
				Host:     envWithDefault("http://elasticsearch:9200", "FLEET_SERVER_ELASTICSEARCH_HOST", "ELASTICSEARCH_HOST"),
				Username: envWithDefault("elastic", "FLEET_SERVER_ELASTICSEARCH_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password: envWithDefault("changeme", "FLEET_SERVER_ELASTICSEARCH_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				CA:       envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA", "ELASTICSEARCH_CA"),
			},
			Enable:       envBool("FLEET_SERVER_ENABLE"),
			Host:         envWithDefault("", "FLEET_SERVER_HOST"),
			InsecureHTTP: envBool("FLEET_SERVER_INSECURE_HTTP"),
			PolicyID:     envWithDefault("", "FLEET_SERVER_POLICY_ID"),
			PolicyName:   envWithDefault("", "FLEET_SERVER_POLICY_NAME", "FLEET_TOKEN_POLICY_NAME"),
			Port:         envWithDefault("", "FLEET_SERVER_PORT"),
		},
		Kibana: kibanaConfig{
			Fleet: kibanaFleetConfig{
				// Remove FLEET_SETUP in 8.x
				// The FLEET_SETUP environment variable boolean is a fallback to the old name. The name was updated to
				// reflect that its setting up Fleet in Kibana versus setting up Fleet Server.
				Setup: envBool("KIBANA_FLEET_SETUP", "FLEET_SETUP"),
				Host:  envWithDefault("http://kibana:5601", "KIBANA_FLEET_HOST", "KIBANA_HOST"),
				//TODO(simitt): check why ELASTICSEARCH values are used here?
				Username: envWithDefault("elastic", "KIBANA_FLEET_USERNAME", "KIBANA_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password: envWithDefault("changeme", "KIBANA_FLEET_PASSWORD", "KIBANA_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				CA:       envWithDefault("", "KIBANA_FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			},
		},
	}
}
