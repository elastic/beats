package beat

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"
)

// Beater interface that every beat must use
type Beater interface {
	Config(*Beat) error
	Setup(*Beat) error
	Run(*Beat) error
	Cleanup(*Beat) error
	Stop()
}

// FlagsHandler (optional) Beater extension for
// handling flags input on startup. The HandleFlags callback will
// be called after parsing the command line arguments and handling
// the '--help' or '--version' flags.
type FlagsHandler interface {
	HandleFlags(*Beat)
}

// Basic beat information
type Beat struct {
	Name    string
	Version string
	Config  *BeatConfig
	BT      Beater
	Events  publisher.Client
}

// Basic configuration of every beat
type BeatConfig struct {
	Output  map[string]outputs.MothershipConfig
	Logging logp.Logging
	Shipper publisher.ShipperConfig
}

var printVersion *bool

func init() {
	printVersion = flag.Bool("version", false, "Print version and exit")
}

// Initiates a new beat object
func NewBeat(name string, version string, bt Beater) *Beat {
	b := Beat{
		Version: version,
		Name:    name,
		BT:      bt,
	}

	return &b
}

// Initiates and runs a new beat object
func Run(name string, version string, bt Beater) *Beat {
	b := NewBeat(name, version, bt)

	// Additional command line args are used to overwrite config options
	b.CommandLineSetup()

	// Loads base config
	b.LoadConfig()

	// Configures beat
	err := bt.Config(b)
	if err != nil {
		logp.Critical("Config error: %v", err)
		os.Exit(1)
	}

	// Run beat. This calls first beater.Setup,
	// then beater.Run and beater.Cleanup in the end
	b.Run()

	return b
}

// Reads and parses the default command line params
// To set additional cmd line args use the beat.CmdLine type before calling the function
func (beat *Beat) CommandLineSetup() {

	// The -c flag is treated separately because it needs the Beat name
	err := cfgfile.ChangeDefaultCfgfileFlag(beat.Name)
	if err != nil {
		fmt.Printf("Failed to fix the -c flag: %v\n", err)
		os.Exit(1)
	}

	flag.Parse()

	if *printVersion {
		fmt.Printf("%s version %s (%s)\n", beat.Name, beat.Version, runtime.GOARCH)
		os.Exit(0)
	}

	// if beater implements CLIFlags for additional CLI handling, call it now
	if flagsHandler, ok := beat.BT.(FlagsHandler); ok {
		flagsHandler.HandleFlags(beat)
	}
}

// LoadConfig inits the config file and reads the default config information
// into Beat.Config. It exists the processes in case of errors.
func (b *Beat) LoadConfig() {

	err := cfgfile.Read(&b.Config, "")
	if err != nil {
		// logging not yet initialized, so using fmt.Printf
		fmt.Printf("Loading config file error: %v\n", err)
		os.Exit(1)
	}

	err = logp.Init(b.Name, &b.Config.Logging)
	if err != nil {
		fmt.Printf("Error initializing logging: %v\n", err)
		os.Exit(1)
	}

	// Disable stderr logging if requested by cmdline flag
	logp.SetStderr()

	logp.Debug("beat", "Initializing output plugins")

	if err := publisher.Publisher.Init(b.Name, b.Config.Output, b.Config.Shipper); err != nil {
		fmt.Printf("Error Initialising publisher: %v\n", err)
		logp.Critical(err.Error())
		os.Exit(1)
	}

	b.Events = publisher.Publisher.Client()

	logp.Info("Init Beat: %s; Version: %s", b.Name, b.Version)
}

// Run calls the beater Setup and Run methods. In case of errors
// during the setup phase, it exits the process.
func (b *Beat) Run() {

	// Setup beater object
	err := b.BT.Setup(b)
	if err != nil {
		logp.Critical("Setup returned an error: %v", err)
		os.Exit(1)
	}

	// Up to here was the initialization, now about running
	if cfgfile.IsTestConfig() {
		// all good, exit with 0
		os.Exit(0)
	}
	service.BeforeRun()

	// Callback is called if the processes is asked to stop.
	// This needs to be called before the main loop is started so that
	// it can register the signals that stop or query (on Windows) the loop.
	service.HandleSignals(b.BT.Stop)

	logp.Info("%s sucessfully setup. Start running.", b.Name)

	// Run beater specific stuff
	err = b.BT.Run(b)
	if err != nil {
		logp.Critical("Run returned an error: %v", err)
	}

	service.Cleanup()

	logp.Info("Cleaning up %s before shutting down.", b.Name)

	// Call beater cleanup function
	err = b.BT.Cleanup(b)
	if err != nil {
		logp.Err("Cleanup returned an error: %v", err)
	}
}

// Stop calls the beater Stop action.
func (beat *Beat) Stop() {
	beat.BT.Stop()
}
