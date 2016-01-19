/*

Beat provides the basic environment for each beat.

Each beat implementation has to implement the beater interface.


# Start / Stop / Exit a Beat

A beat is start by calling the Run(name string, version string, bt Beater) function an passing the beater object.
This will create new beat and will Start the beat in its own go process. The Run function is blocked until
the Beat.exit channel is closed. This can be done through calling Beat.Exit(). This happens for example when CTRL-C
is pressed.

A beat can be stopped and started again through beat.Stop and beat.Start. When starting a beat again, it is important to
run it again in it's own go process. To allow a beat to be properly reastarted, it is important that Beater.Stop() properly
closes all channels and go processes.

In case a beat should not run as a long running process, the beater implementation must make sure to call Beat.Exit()
when the task is completed to stop the beat.

*/
package beat

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/service"

	"github.com/satori/go.uuid"
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
	UUID    uuid.UUID

	exit chan struct{}
}

// Basic configuration of every beat
type BeatConfig struct {
	Output  map[string]outputs.MothershipConfig
	Logging logp.Logging
	Shipper publisher.ShipperConfig
}

var printVersion *bool

// Channel that is closed as soon as the beat should exit
func init() {
	printVersion = flag.Bool("version", false, "Print version and exit")
}

// Initiates a new beat object
func NewBeat(name string, version string, bt Beater) *Beat {
	if version == "" {
		version = defaultBeatVersion
	}
	b := Beat{
		Version: version,
		Name:    name,
		BT:      bt,
		UUID:    uuid.NewV4(),

		exit: make(chan struct{}),
	}

	return &b
}

// Initiates and runs a new beat object
func Run(name string, version string, bt Beater) {

	b := NewBeat(name, version, bt)

	// Runs beat inside a go process
	go func() {
		err := b.Start()

		if err != nil {
			// TODO: detect if logging was already fully setup or not
			fmt.Printf("Start error: %v\n", err)
			logp.Critical("Start error: %v", err)
			os.Exit(1)
		}

		// If start finishes, exit has to be called. This requires start to be blocking
		// which is currently the default.
		b.Exit()
	}()

	// Waits until beats channel is closed
	select {
	case <-b.exit:
		b.Stop()
		logp.Info("Exit beat completed")
		return
	}
}

// Start starts the Beat by parsing and interpreting the command line flags,
// loading and parsing the configuration file, and running the Beat. This
// method blocks until the Beat exits. If an error occurs while initializing
// or running the Beat it will be returned.
func (b *Beat) Start() error {
	// Additional command line args are used to overwrite config options
	err, exit := b.CommandLineSetup()
	if err != nil {
		return err
	}

	if exit {
		return nil
	}

	// Loads base config
	err = b.LoadConfig()
	if err != nil {
		return err
	}

	// Configures beat
	err = b.BT.Config(b)
	if err != nil {
		return err
	}

	// Run beat. This calls first beater.Setup,
	// then beater.Run and beater.Cleanup in the end
	return b.Run()
}

// Reads and parses the default command line params
// To set additional cmd line args use the beat.CmdLine type before calling the function
// The second return param is to detect if system should exit. True if should exit
// Exit can also be without error
func (beat *Beat) CommandLineSetup() (error, bool) {

	// The -c flag is treated separately because it needs the Beat name
	err := cfgfile.ChangeDefaultCfgfileFlag(beat.Name)
	if err != nil {
		return fmt.Errorf("failed to fix the -c flag: %v\n", err), true
	}

	flag.Parse()

	if *printVersion {
		fmt.Printf("%s version %s (%s)\n", beat.Name, beat.Version, runtime.GOARCH)
		return nil, true
	}

	// if beater implements CLIFlags for additional CLI handling, call it now
	if flagsHandler, ok := beat.BT.(FlagsHandler); ok {
		flagsHandler.HandleFlags(beat)
	}

	return nil, false
}

// LoadConfig inits the config file and reads the default config information
// into Beat.Config. It exists the processes in case of errors.
func (b *Beat) LoadConfig() error {

	err := cfgfile.Read(&b.Config, "")
	if err != nil {
		return fmt.Errorf("loading config file error: %v\n", err)
	}

	err = logp.Init(b.Name, &b.Config.Logging)
	if err != nil {
		return fmt.Errorf("error initializing logging: %v\n", err)
	}

	// Disable stderr logging if requested by cmdline flag
	logp.SetStderr()

	logp.Debug("beat", "Initializing output plugins")

	pub, err := publisher.New(b.Name, b.Config.Output, b.Config.Shipper)
	if err != nil {
		return fmt.Errorf("error Initialising publisher: %v\n", err)
	}

	b.Events = pub.Client()

	logp.Info("Init Beat: %s; Version: %s", b.Name, b.Version)

	return nil
}

// Run calls the beater Setup and Run methods. In case of errors
// during the setup phase, it exits the process.
func (b *Beat) Run() error {

	// Setup beater object
	err := b.BT.Setup(b)
	if err != nil {
		return fmt.Errorf("setup returned an error: %v", err)
	}

	// Up to here was the initialization, now about running
	if cfgfile.IsTestConfig() {
		logp.Info("Testing configuration file")
		// all good, exit
		return nil
	}
	service.BeforeRun()

	// Callback is called if the processes is asked to stop.
	// This needs to be called before the main loop is started so that
	// it can register the signals that stop or query (on Windows) the loop.
	service.HandleSignals(b.Exit)

	logp.Info("%s sucessfully setup. Start running.", b.Name)

	// Run beater specific stuff
	err = b.BT.Run(b)
	if err != nil {
		logp.Critical("Running the beat returned an error: %v", err)
	}

	service.Cleanup()

	logp.Info("Cleaning up %s before shutting down.", b.Name)

	// Call beater cleanup function
	err = b.BT.Cleanup(b)
	if err != nil {
		logp.Err("Cleanup returned an error: %v", err)
	}
	return err
}

// Stop calls the beater Stop action.
// It can happen that this function is called more then once.
func (beat *Beat) Stop() {
	logp.Info("Stopping Beat")
	beat.BT.Stop()
}

// Exiting beat -> shutdown
func (b *Beat) Exit() {
	logp.Info("Start exiting beat")
	close(b.exit)
}
