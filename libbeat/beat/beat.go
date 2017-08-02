package beat

import (
	"github.com/elastic/beats/libbeat/common"
)

// Creator initializes and configures a new Beater instance used to execute
// the beat its run-loop.
type Creator func(*Beat, *common.Config) (Beater, error)

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
	Info      common.BeatInfo // beat metadata.
	Publisher Pipeline        // Publisher pipeline

	SetupMLCallback SetupMLCallback // setup callback for ML job configs
	InSetupCmd      bool            // this is set to true when the `setup` command is called

	// XXX: remove Config from public interface
	Config *BeatConfig // Common Beat configuration data.

	BeatConfig *common.Config // The beats it's own configuration section
}

// BeatConfig struct contains the basic configuration of every beat
type BeatConfig struct {
	// output/publishing related configurations
	Output common.ConfigNamespace `config:"output"`

	// 'setup' configurations
	Dashboards *common.Config `config:"setup.dashboards"`
	Template   *common.Config `config:"setup.template"`
	Kibana     *common.Config `config:"setup.kibana"`
}

// SetupMLCallback can be used by the Beat to register MachineLearning configurations
// for the enabled modules.
type SetupMLCallback func(*Beat) error
