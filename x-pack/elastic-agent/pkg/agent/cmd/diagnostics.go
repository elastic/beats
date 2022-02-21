// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config/operations"
)

var diagOutputs = map[string]outputter{
	"human": humanDiagnosticsOutput,
	"json":  jsonOutput,
	"yaml":  yamlOutput,
}

// DiagnosticsInfo a struct to track all information related to diagnostics for the agent.
type DiagnosticsInfo struct {
	ProcMeta     []client.ProcMeta
	AgentVersion client.Version
}

// AgentConfig tracks all configuration that the agent uses, local files, rendered policies, beat inputs etc.
type AgentConfig struct {
	ConfigLocal    *configuration.Configuration
	ConfigRendered map[string]interface{}
}

func newDiagnosticsCommand(s []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Gather diagnostics information from the elastic-agent and running processes.",
		Long:  "Gather diagnostics information from the elastic-agent and running processes.",
		Run: func(c *cobra.Command, args []string) {
			if err := diagnosticCmd(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n%s\n", err, troubleshootMessage())
				os.Exit(1)
			}
		},
	}

	cmd.Flags().String("output", "human", "Output the diagnostics information in either human, json, or yaml (default: human)")
	cmd.AddCommand(newDiagnosticsCollectCommandWithArgs(s, streams))

	return cmd
}

func newDiagnosticsCollectCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect diagnostics information from the elastic-agent and write it to a zip archive.",
		Long:  "Collect diagnostics information from the elastic-agent and write it to a zip archive.\nNote that any credentials will appear in plain text.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			file, _ := c.Flags().GetString("file")

			if file == "" {
				ts := time.Now().UTC()
				file = "elastic-agent-diagnostics-" + ts.Format("2006-01-02T15-04-05Z07-00") + ".zip" // RFC3339 format that replaces : with -, so it will work on Windows
			}

			output, _ := c.Flags().GetString("output")
			switch output {
			case "yaml":
			case "json":
			default:
				return fmt.Errorf("unsupported output: %s", output)
			}

			return diagnosticsCollectCmd(streams, file, output)
		},
	}

	cmd.Flags().StringP("file", "f", "", "name of the output diagnostics zip archive")
	cmd.Flags().String("output", "yaml", "Output the collected information in either json, or yaml (default: yaml)") // replace output flag with different options

	return cmd
}

func diagnosticCmd(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	err := tryContainerLoadPaths()
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	outputFunc, ok := diagOutputs[output]
	if !ok {
		return fmt.Errorf("unsupported output: %s", output)
	}

	ctx := handleSignal(context.Background())
	innerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	diag, err := getDiagnostics(innerCtx)
	if err == context.DeadlineExceeded {
		return errors.New("timed out after 30 seconds trying to connect to Elastic Agent daemon")
	} else if err == context.Canceled {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to communicate with Elastic Agent daemon: %s", err)
	}

	return outputFunc(streams.Out, diag)
}

func diagnosticsCollectCmd(streams *cli.IOStreams, fileName, outputFormat string) error {
	err := tryContainerLoadPaths()
	if err != nil {
		return err
	}

	ctx := handleSignal(context.Background())
	innerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	diag, err := getDiagnostics(innerCtx)
	if err == context.DeadlineExceeded {
		return errors.New("timed out after 30 seconds trying to connect to Elastic Agent daemon")
	} else if err == context.Canceled {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to communicate with Elastic Agent daemon: %w", err)
	}

	cfg, err := gatherConfig()
	if err != nil {
		return fmt.Errorf("unable to gather config data: %w", err)
	}

	err = createZip(fileName, outputFormat, diag, cfg)
	if err != nil {
		return fmt.Errorf("unable to create archive %q: %w", fileName, err)
	}
	fmt.Fprintf(streams.Out, "Created diagnostics archive %q\n", fileName)
	fmt.Fprintln(streams.Out, "***** WARNING *****\nCreated archive may contain plain text credentials.\nEnsure that files in archive are redacted before sharing.\n*******************")
	return nil
}

func getDiagnostics(ctx context.Context) (DiagnosticsInfo, error) {
	daemon := client.New()
	diag := DiagnosticsInfo{}
	err := daemon.Connect(ctx)
	if err != nil {
		return DiagnosticsInfo{}, err
	}
	defer daemon.Disconnect()

	bv, err := daemon.ProcMeta(ctx)
	if err != nil {
		return DiagnosticsInfo{}, err
	}
	diag.ProcMeta = bv

	version, err := daemon.Version(ctx)
	if err != nil {
		return diag, err
	}
	diag.AgentVersion = version

	return diag, nil
}

func humanDiagnosticsOutput(w io.Writer, obj interface{}) error {
	diag, ok := obj.(DiagnosticsInfo)
	if !ok {
		return fmt.Errorf("unable to cast %T as DiagnosticsInfo", obj)
	}
	return outputDiagnostics(w, diag)
}

func outputDiagnostics(w io.Writer, d DiagnosticsInfo) error {
	tw := tabwriter.NewWriter(w, 4, 1, 2, ' ', 0)
	fmt.Fprintf(tw, "elastic-agent\tversion: %s\n", d.AgentVersion.Version)
	fmt.Fprintf(tw, "\tbuild_commit: %s\tbuild_time: %s\tsnapshot_build: %v\n", d.AgentVersion.Commit, d.AgentVersion.BuildTime, d.AgentVersion.Snapshot)
	if len(d.ProcMeta) == 0 {
		fmt.Fprintf(tw, "Applications: (none)\n")
	} else {
		fmt.Fprintf(tw, "Applications:\n")
		for _, app := range d.ProcMeta {
			fmt.Fprintf(tw, "  *\tname: %s\troute_key: %s\n", app.Name, app.RouteKey)
			if app.Error != "" {
				fmt.Fprintf(tw, "\terror: %s\n", app.Error)
			} else {
				fmt.Fprintf(tw, "\tprocess: %s\tid: %s\tephemeral_id: %s\telastic_license: %v\n", app.Process, app.ID, app.EphemeralID, app.ElasticLicensed)
				fmt.Fprintf(tw, "\tversion: %s\tcommit: %s\tbuild_time: %s\tbinary_arch: %v\n", app.Version, app.BuildCommit, app.BuildTime, app.BinaryArchitecture)
				fmt.Fprintf(tw, "\thostname: %s\tusername: %s\tuser_id: %s\tuser_gid: %s\n", app.Hostname, app.Username, app.UserID, app.UserGID)
			}

		}
	}
	tw.Flush()
	return nil
}

func gatherConfig() (AgentConfig, error) {
	cfg := AgentConfig{}
	localCFG, err := loadConfig(nil)
	if err != nil {
		return cfg, err
	}
	cfg.ConfigLocal = localCFG

	renderedCFG, err := operations.LoadFullAgentConfig(paths.ConfigFile(), true)
	if err != nil {
		return cfg, err
	}
	// Must force *config.Config to map[string]interface{} in order to write to a file.
	mapCFG, err := renderedCFG.ToMapStr()
	if err != nil {
		return cfg, err
	}

	for k, _ := range mapCFG {
		if strings.Contains(k, "api_key") {
			mapCFG[k] = "REDACTED"
		}
	}
	cfg.ConfigRendered = mapCFG

	return cfg, nil
}

// createZip creates a zip archive with the passed fileName.
//
// The passed DiagnosticsInfo and AgentConfig data is written in the specified output format.
// Any local log files are collected and copied into the archive.
func createZip(fileName, outputFormat string, diag DiagnosticsInfo, cfg AgentConfig) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)

	zf, err := zw.Create("meta/")
	if err != nil {
		return closeHandlers(err, zw, f)
	}

	zf, err = zw.Create("meta/elastic-agent-version." + outputFormat)
	if err != nil {
		return closeHandlers(err, zw, f)
	}
	if err := writeFile(zf, outputFormat, diag.AgentVersion); err != nil {
		return closeHandlers(err, zw, f)
	}

	for _, m := range diag.ProcMeta {
		zf, err = zw.Create("meta/" + m.Name + "-" + m.RouteKey + "." + outputFormat)
		if err != nil {
			return closeHandlers(err, zw, f)
		}

		if err := writeFile(zf, outputFormat, m); err != nil {
			return closeHandlers(err, zw, f)
		}
	}

	zf, err = zw.Create("config/")
	if err != nil {
		return closeHandlers(err, zw, f)
	}

	zf, err = zw.Create("config/elastic-agent-local." + outputFormat)
	if err != nil {
		return closeHandlers(err, zw, f)
	}
	if err := writeFile(zf, outputFormat, cfg.ConfigLocal); err != nil {
		return closeHandlers(err, zw, f)
	}

	zf, err = zw.Create("config/elastic-agent-policy." + outputFormat)
	if err != nil {
		return closeHandlers(err, zw, f)
	}
	if err := writeFile(zf, outputFormat, cfg.ConfigRendered); err != nil {
		return closeHandlers(err, zw, f)
	}

	if err := zipLogs(zw); err != nil {
		return closeHandlers(err, zw, f)
	}

	return closeHandlers(nil, zw, f)
}

// zipLogs walks paths.Logs() and copies the file structure into zw in "logs/"
func zipLogs(zw *zip.Writer) error {
	_, err := zw.Create("logs/")
	if err != nil {
		return err
	}

	// using Data() + "/logs", for some reason default paths/Logs() is the home dir...
	logPath := filepath.Join(paths.Home(), "logs") + string(filepath.Separator)
	return filepath.WalkDir(logPath, func(path string, d fs.DirEntry, fErr error) error {
		if stderrors.Is(fErr, fs.ErrNotExist) {
			return nil
		}
		if fErr != nil {
			return fmt.Errorf("unable to walk log dir: %w", fErr)
		}

		name := strings.TrimPrefix(path, logPath)
		if name == "" {
			return nil
		}

		if d.IsDir() {
			_, err := zw.Create("logs/" + name + "/")
			if err != nil {
				return fmt.Errorf("unable to create log directory in archive: %w", err)
			}
			return nil
		}

		lf, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open log file: %w", err)
		}
		zf, err := zw.Create("logs/" + name)
		if err != nil {
			return closeHandlers(fmt.Errorf("unable to create log file in archive: %w", err), lf)
		}
		_, err = io.Copy(zf, lf)
		if err != nil {
			return closeHandlers(fmt.Errorf("log file copy failed: %w", err), lf)
		}

		return lf.Close()
	})
}

// writeFile writes json or yaml data from the interface to the writer.
func writeFile(w io.Writer, outputFormat string, v interface{}) error {
	if outputFormat == "json" {
		je := json.NewEncoder(w)
		je.SetIndent("", "  ")
		return je.Encode(v)
	}
	ye := yaml.NewEncoder(w)
	err := ye.Encode(v)
	return closeHandlers(err, ye)
}

// closeHandlers will close all passed closers attaching any errors to the passed err and returning the result
func closeHandlers(err error, closers ...io.Closer) error {
	var mErr *multierror.Error
	mErr = multierror.Append(mErr, err)
	for _, c := range closers {
		if inErr := c.Close(); inErr != nil {
			mErr = multierror.Append(mErr, inErr)
		}
	}
	return mErr.ErrorOrNil()
}
