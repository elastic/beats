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

* Preparing Kibana for Fleet
  This prepares the Fleet plugin that exists inside of Kibana. This must either be enabled here or done externally
  before Fleet Server will actually successfully start.

  KIBANA_FLEET_SETUP - set to 1 enables this setup
  KIBANA_FLEET_HOST - kibana host to enable Fleet on [$KIBANA_HOST]
  KIBANA_FLEET_USERNAME - kibana username to enable Fleet [$KIBANA_USERNAME]
  KIBANA_FLEET_PASSWORD - kibana password to enable Fleet [$KIBANA_PASSWORD]
  KIBANA_FLEET_CA - path to certificate authority to use with communicate with Kibana [$KIBANA_CA]
  KIBANA_REQUEST_RETRY_SLEEP - specifies sleep duration taken when agent performs a request to kibana [default 1s]
  KIBANA_REQUEST_RETRY_COUNT - specifies number of retries agent performs when executing a request to kibana [default 30]

* Bootstrapping Fleet Server
  This bootstraps the Fleet Server to be run by this Elastic Agent. At least one Fleet Server is required in a Fleet
  deployment for other Elastic Agent to bootstrap.

  FLEET_SERVER_ENABLE - set to 1 enables bootstrapping of Fleet Server (forces FLEET_ENROLL enabled)
  FLEET_SERVER_ELASTICSEARCH_HOST - elasticsearch host for Fleet Server to communicate with [$ELASTICSEARCH_HOST]
  FLEET_SERVER_ELASTICSEARCH_USERNAME - elasticsearch username for Fleet Server [$ELASTICSEARCH_USERNAME]
  FLEET_SERVER_ELASTICSEARCH_PASSWORD - elasticsearch password for Fleet Server [$ELASTICSEARCH_PASSWORD]
  FLEET_SERVER_ELASTICSEARCH_CA - path to certificate authority to use with communicate with elasticsearch [$ELASTICSEARCH_CA]
  FLEET_SERVER_SERVICE_TOKEN - service token to use for communication with elasticsearch
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
			if err := logContainerCmd(streams, c); err != nil {
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
	if err := setPaths(); err != nil {
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

	if cfg.Kibana.Fleet.Setup {
		client, err = kibanaClient(cfg.Kibana)
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
				client, err = kibanaClient(cfg.Kibana)
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
		policyID := ""
		if policy != nil {
			policyID = policy.ID
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

func buildEnrollArgs(cfg setupConfig, token string, policyID string) ([]string, error) {
	args := []string{
		"enroll", "-f",
		"-c", paths.ConfigFile(),
		"--path.home", paths.Top(), // --path.home actually maps to paths.Top()
		"--path.config", paths.Config(),
		"--path.logs", paths.Logs(),
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
		if policyID == "" {
			policyID = cfg.FleetServer.PolicyID
		}
		if policyID != "" {
			args = append(args, "--fleet-server-policy", policyID)
		}
		if cfg.FleetServer.Elasticsearch.CA != "" {
			args = append(args, "--fleet-server-es-ca", cfg.FleetServer.Elasticsearch.CA)
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
		if cfg.Fleet.URL != "" {
			args = append(args, "--url", cfg.Fleet.URL)
		}
		if cfg.FleetServer.InsecureHTTP {
			args = append(args, "--fleet-server-insecure-http")
		}
		if cfg.FleetServer.InsecureHTTP || cfg.Fleet.Insecure {
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
	if token != "" {
		args = append(args, "--enrollment-token", token)
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
	if cfg.Elasticsearch.ServiceToken != "" {
		return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path), nil
	}
	return fmt.Sprintf("%s://%s:%s@%s%s", u.Scheme, cfg.Elasticsearch.Username, cfg.Elasticsearch.Password, u.Host, path), nil
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
	return findPolicy(cfg, policies.Items)
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
	// Extract the ingest pipeline definition to the HOME_DIR
	if home := os.Getenv("HOME_PATH"); home != "" {
		if err := syncDir(filepath.Join(apmDir, "ingest"), filepath.Join(home, "ingest")); err != nil {
			return nil, fmt.Errorf("syncing APM ingest directory to HOME_PATH(%s) failed: %s", home, err)
		}
	}
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
	addEnv("--path.home", "HOME_PATH")
	addEnv("--path.config", "CONFIG_PATH")
	addEnv("--path.data", "DATA_PATH")
	addEnv("--path.logs", "LOGS_PATH")
	addEnv("--httpprof", "HTTPPROF")
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

func setPaths() error {
	statePath := envWithDefault(defaultStateDirectory, "STATE_PATH")
	if statePath == "" {
		return errors.New("STATE_PATH cannot be set to an empty string")
	}
	topPath := filepath.Join(statePath, "data")
	configPath := envWithDefault("", "CONFIG_PATH")
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
	srcDownloads := filepath.Join(paths.Home(), "downloads")
	destDownloads := filepath.Join(statePath, "data", "downloads")
	if err := syncDir(srcDownloads, destDownloads); err != nil {
		return fmt.Errorf("syncing download directory to STATE_PATH(%s) failed: %s", statePath, err)
	}
	paths.SetTop(topPath)
	paths.SetConfig(configPath)
	// when custom top path is provided the home directory is not versioned
	paths.SetVersionHome(false)
	// set LOGS_PATH is given
	if logsPath := envWithDefault("", "LOGS_PATH"); logsPath != "" {
		paths.SetLogs(logsPath)
		// ensure that the logs directory exists
		if err := os.MkdirAll(filepath.Join(logsPath), 0755); err != nil {
			return fmt.Errorf("preparing LOGS_PATH(%s) failed: %s", logsPath, err)
		}
	}
	return nil
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
	CA           string `config:"ca"`
	Host         string `config:"host"`
	Username     string `config:"username"`
	Password     string `config:"password"`
	ServiceToken string `config:"service_token"`
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
	CertKey       string              `config:"cert_key"`
	Elasticsearch elasticsearchConfig `config:"elasticsearch"`
	Enable        bool                `config:"enable"`
	Host          string              `config:"host"`
	InsecureHTTP  bool                `config:"insecure_http"`
	PolicyID      string              `config:"policy_id"`
	PolicyName    string              `config:"policy_name"`
	Port          string              `config:"port"`
}

type kibanaConfig struct {
	Fleet              kibanaFleetConfig `config:"fleet"`
	RetrySleepDuration time.Duration     `config:"retry_sleep_duration"`
	RetryMaxCount      int               `config:"retry_max_count"`
}

type kibanaFleetConfig struct {
	CA       string `config:"ca"`
	Host     string `config:"host"`
	Password string `config:"password"`
	Setup    bool   `config:"setup"`
	Username string `config:"username"`
}

func defaultAccessConfig() (setupConfig, error) {
	retrySleepDuration, err := envDurationWithDefault(defaultRequestRetrySleep, requestRetrySleepEnv)
	if err != nil {
		return setupConfig{}, err
	}

	retryMaxCount, err := envIntWithDefault(defaultMaxRequestRetries, maxRequestRetriesEnv)
	if err != nil {
		return setupConfig{}, err
	}

	cfg := setupConfig{
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
				Host:         envWithDefault("http://elasticsearch:9200", "FLEET_SERVER_ELASTICSEARCH_HOST", "ELASTICSEARCH_HOST"),
				Username:     envWithDefault("elastic", "FLEET_SERVER_ELASTICSEARCH_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password:     envWithDefault("changeme", "FLEET_SERVER_ELASTICSEARCH_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				ServiceToken: envWithDefault("", "FLEET_SERVER_SERVICE_TOKEN"),
				CA:           envWithDefault("", "FLEET_SERVER_ELASTICSEARCH_CA", "ELASTICSEARCH_CA"),
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
				Setup:    envBool("KIBANA_FLEET_SETUP", "FLEET_SETUP"),
				Host:     envWithDefault("http://kibana:5601", "KIBANA_FLEET_HOST", "KIBANA_HOST"),
				Username: envWithDefault("elastic", "KIBANA_FLEET_USERNAME", "KIBANA_USERNAME", "ELASTICSEARCH_USERNAME"),
				Password: envWithDefault("changeme", "KIBANA_FLEET_PASSWORD", "KIBANA_PASSWORD", "ELASTICSEARCH_PASSWORD"),
				CA:       envWithDefault("", "KIBANA_FLEET_CA", "KIBANA_CA", "ELASTICSEARCH_CA"),
			},
			RetrySleepDuration: retrySleepDuration,
			RetryMaxCount:      retryMaxCount,
		},
	}
	return cfg, nil
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
