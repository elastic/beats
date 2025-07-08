// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beat

import (
	"github.com/elastic/beats/v7/libbeat/api"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/instrumentation"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/useragent"
)

// Creator initializes and configures a new Beater instance used to execute
// the beat's run-loop.
type Creator func(*Beat, *config.C) (Beater, error)

// Beater is the interface that must be implemented by every Beat. A Beater
// provides the main Run-loop and a Stop method to break the Run-loop.
// Instantiation and Configuration is normally provided by a Beat-`Creator`.
//
// Once the beat is fully configured, the Run() method is invoked. The
// Run()-method implements the beat its run-loop. Once the Run()-method returns,
// the beat shuts down.
//
// The Stop() method is invoked the first time (and only the first time) a
// shutdown signal is received. The Stop()-method normally will stop the Run()-loop,
// such that the beat can gracefully shutdown.
type Beater interface {
	// The main event loop. This method should block until signalled to stop by an
	// invocation of the Stop() method.
	Run(b *Beat) error

	// Stop is invoked to signal that the Run method should finish its execution.
	// It will be invoked at most once.
	Stop()
}

// Beat contains the basic beat data and the publisher client used to publish
// events.
type Beat struct {
	Info      Info     // beat metadata.
	Publisher Pipeline // Publisher pipeline

	Monitoring Monitoring

	InSetupCmd bool // this is set to true when the `setup` command is called

	OverwritePipelinesCallback OverwritePipelinesCallback // ingest pipeline loader callback
	// XXX: remove Config from public interface.
	//      It's currently used by filebeat modules to setup the Ingest Node
	//      pipeline and ML jobs.
	Config *BeatConfig // Common Beat configuration data.

	// OutputConfigReloader may be set by a Creator to watch for output config changes.
	//
	// This reloader is called in addition to libbeat's internal output reloader, which
	// is responsible for reconfiguring Publisher.
	OutputConfigReloader reload.Reloadable

	BeatConfig *config.C // The beat's own configuration section

	Fields []byte // Data from fields.yml

	Manager management.Manager // manager

	Keystore keystore.Keystore

	Instrumentation instrumentation.Instrumentation // instrumentation holds an APM agent for capturing and reporting traces

	API      *api.Server      // API server. This is nil unless the http endpoint is enabled.
	Registry *reload.Registry // input, & output registry for configuration manager, should be instantiated in NewBeat
}

func (beat *Beat) userAgentProduct() string {
	if beat.Info.Beat != "" {
		return beat.Info.Beat
	}
	return "Libbeat"
}

// fallbackUserAgent returns the user agent string for the beat.
func (beat *Beat) fallbackUserAgent() string {
	// if we're in fleet mode, construct some additional elements for the UA comment field
	comments := []string{}
	if beat.Manager != nil && beat.Manager.Enabled() {
		info := beat.Manager.AgentInfo()
		if info.ManagedMode == proto.AgentManagedMode_MANAGED {
			comments = append(comments, "Managed")
		} else if info.ManagedMode == proto.AgentManagedMode_STANDALONE {
			comments = append(comments, "Standalone")
		}

		if info.Unprivileged {
			comments = append(comments, "Unprivileged")
		}
	}

	return useragent.UserAgent(beat.userAgentProduct(), version.GetDefaultVersion(),
		version.Commit(), version.BuildTime().String(), comments...)
}

// generateUserAgent returns the user agent string for the beat.
func (beat *Beat) generateUserAgent() (string, error) {
	var mode useragent.AgentManagementMode

	info := beat.Manager.AgentInfo()
	switch info.ManagedMode {
	case proto.AgentManagedMode_MANAGED:
		mode = useragent.AgentManagementModeManaged
	case proto.AgentManagedMode_STANDALONE:
		mode = useragent.AgentManagementModeStandalone
	}

	privileged := useragent.AgentUnprivilegedModePrivileged
	if info.Unprivileged {
		privileged = useragent.AgentUnprivilegedModeUnprivileged
	}

	return useragent.UserAgentWithBeatTelemetry(beat.userAgentProduct(), version.GetDefaultVersion(),
		mode, privileged)
}

// GenerateUserAgent populates the UserAgent field on the beat.Info struct
func (beat *Beat) GenerateUserAgent() {
	ua, err := beat.generateUserAgent()
	if err != nil {
		ua = beat.fallbackUserAgent()
	}
	beat.Info.UserAgent = ua
}

// BeatConfig struct contains the basic configuration of every beat
type BeatConfig struct {
	// output/publishing related configurations
	Output config.Namespace `config:"output"`
}

// OverwritePipelinesCallback can be used by the Beat to register Ingest pipeline loader
// for the enabled modules.
type OverwritePipelinesCallback func(*config.C) error
