/*
Package beat provides the functions required to manage the life-cycle of a Beat.
It provides the standard mechanism for launching a Beat. It manages
configuration, logging, and publisher initialization and registers a signal
handler to gracefully stop the process.

Each Beat implementation must implement the Beater interface and may optionally
implement the FlagsHandler interface. See the Beater interface documentation for
more details.

To use this package, create a simple main that invokes the Run() function.

  func main() {
  	if err := beat.Run("mybeat", myVersion, beater.New()); err != nil {
  		os.Exit(1)
  	}
  }

In the example above, the beater package contains the implementation of the
Beater interface and the New() method returns a new instance of Beater. The
Beater implementation is placed into its own package so that it can be reused
or combined with other Beats.

Recommendations

  * Use the logp package for logging rather than writing to stdout or stderr.
  * Do not call os.Exit in any of your code. Return an error instead. Or if your
    code needs to exit without an error, return beat.GracefulExit.
*/
package beat

import (
	cryptRand "crypto/rand"
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/filter"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher"
	svc "github.com/elastic/beats/libbeat/service"
	"github.com/satori/go.uuid"
)

var (
	printVersion = flag.Bool("version", false, "Print the version and exit")
)

var debugf = logp.MakeDebug("beat")

// Beater is the interface that must be implemented every Beat. The full
// lifecycle of a Beat instance is managed through this interface.
//
// Life-cycle of Beater
//
// The four operational methods are always invoked serially in the following
// order:
//
//   Config -> Setup -> Run -> Cleanup
//
// The Stop() method is invoked the first time (and only the first time) a
// shutdown signal is received. The Stop() method is eligible to be invoked
// at any point after Setup() completes (this ensures that the Beater
// implementation is fully initialized before Stop() can be invoked).
//
// The Cleanup() method is guaranteed to be invoked upon shutdown iff the Beater
// reaches the Setup stage. For example, if there is a failure in the
// Config stage then Cleanup will not be invoked.
type Beater interface {
	Config(*Beat) error  // Read and validate configuration.
	Setup(*Beat) error   // Initialize the Beat.
	Run(*Beat) error     // The main event loop. This method should block until signalled to stop by an invocation of the Stop() method.
	Cleanup(*Beat) error // Cleanup is invoked to perform any final clean-up prior to exiting.
	Stop()               // Stop is invoked to signal that the Run method should finish its execution. It will be invoked at most once.
}

// FlagsHandler is an interface that can optionally be implemented by a Beat
// if it needs to process command line flags on startup. If implemented, the
// HandleFlags method will be invoked after parsing the command line flags
// and before any of the Beater interface methods are invoked. There will be
// no callback when '-help' or '-version' are specified.
type FlagsHandler interface {
	HandleFlags(*Beat) error // Handle any custom command line arguments.
}

// Beat contains the basic beat data and the publisher client used to publish
// events.
type Beat struct {
	Name      string               // Beat name.
	Version   string               // Beat version number. Defaults to the libbeat version when an implementation does not set a version.
	UUID      uuid.UUID            // ID assigned to a Beat instance.
	BT        Beater               // Beater implementation.
	RawConfig *common.Config       // Raw config that can be unpacked to get Beat specific config data.
	Config    BeatConfig           // Common Beat configuration data.
	Publisher *publisher.Publisher // Publisher

	filters *filter.FilterList // Filters
}

// BeatConfig struct contains the basic configuration of every beat
type BeatConfig struct {
	Output  map[string]*common.Config
	Logging logp.Logging
	Shipper publisher.ShipperConfig
	Filters []filter.FilterConfig
	Path    paths.Path
}

// Run initializes and runs a Beater implementation. name is the name of the
// Beat (e.g. packetbeat or topbeat). version is version number of the Beater
// implementation. bt is Beater implementation to run.
func Run(name, version string, bt Beater) error {
	return newInstance(name, version, bt).launch(true)
}

// instance contains everything related to a single instance of a beat.
type instance struct {
	data   *Beat
	beater Beater
}

func init() {
	// Initialize runtime random number generator seed using global, shared
	// cryptographically strong pseudo random number generator.
	//
	// On linux Reader might use getrandom(2) or /udev/random. On windows systems
	// CryptGenRandom is used.
	n, err := cryptRand.Int(cryptRand.Reader, big.NewInt(math.MaxInt64))
	var seed int64
	if err != nil {
		// fallback to current timestamp on error
		seed = time.Now().UnixNano()
	} else {
		seed = n.Int64()
	}

	rand.Seed(seed)
}

// newInstance creates and initializes a new Beat instance.
func newInstance(name string, version string, bt Beater) *instance {
	if version == "" {
		version = defaultBeatVersion
	}

	return &instance{
		data: &Beat{
			Name:    name,
			Version: version,
			UUID:    uuid.NewV4(),
			BT:      bt,
		},
		beater: bt,
	}
}

// handleFlags parses the command line flags. It handles the '-version' flag
// and invokes the HandleFlags callback if implemented by the Beat.
func (bc *instance) handleFlags() error {
	// Due to a dependence upon the beat name, the default config file path
	// must be updated prior to CLI flag handling.
	err := cfgfile.ChangeDefaultCfgfileFlag(bc.data.Name)
	if err != nil {
		return fmt.Errorf("failed to set default config file path: %v", err)
	}

	flag.Parse()

	if *printVersion {
		fmt.Printf("%s version %s (%s), libbeat %s\n", bc.data.Name,
			bc.data.Version, runtime.GOARCH, defaultBeatVersion)
		return GracefulExit
	}

	// Invoke HandleFlags if FlagsHandler is implemented.
	if flagsHandler, ok := bc.beater.(FlagsHandler); ok {
		err = flagsHandler.HandleFlags(bc.data)
	}

	return err
}

// config reads the configuration file from disk, parses the common options
// defined in BeatConfig, initializes logging, and set GOMAXPROCS if defined
// in the config. Lastly it invokes the Config method implemented by the beat.
func (bc *instance) config() error {
	var err error
	bc.data.RawConfig, err = cfgfile.Load("")
	if err != nil {
		return fmt.Errorf("error loading config file: %v", err)
	}

	err = bc.data.RawConfig.Unpack(&bc.data.Config)
	if err != nil {
		return fmt.Errorf("error unpacking config data: %v", err)
	}

	err = paths.InitPaths(&bc.data.Config.Path)
	if err != nil {
		return fmt.Errorf("error setting default paths: %v", err)
	}

	err = logp.Init(bc.data.Name, &bc.data.Config.Logging)
	if err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}
	// Disable stderr logging if requested by cmdline flag
	logp.SetStderr()

	// log paths values to help with troubleshooting
	logp.Info(paths.Paths.String())

	bc.data.filters, err = filter.New(bc.data.Config.Filters)
	if err != nil {
		return fmt.Errorf("error initializing filters: %v", err)
	}
	debugf("Filters: %+v", bc.data.filters)

	if bc.data.Config.Shipper.MaxProcs != nil {
		maxProcs := *bc.data.Config.Shipper.MaxProcs
		if maxProcs > 0 {
			runtime.GOMAXPROCS(maxProcs)
		}
	}

	return bc.beater.Config(bc.data)

	// TODO: If -configtest is set it should exit at this point. But changing
	// this now would mean a change in behavior. Some Beats may depend on the
	// Setup() method being invoked in order to do configuration validation.
	// If we do not change this, it means -configtest requires the outputs to
	// be available because the publisher is being started (this is not
	// desirable - elastic/beats#1213). It (may?) also cause the index template
	// to be loaded.
}

// setup initializes the Publisher and then invokes the Setup method of the
// Beat.
func (bc *instance) setup() error {
	logp.Info("Setup Beat: %s; Version: %s", bc.data.Name, bc.data.Version)

	debugf("Initializing output plugins")
	var err error
	bc.data.Publisher, err = publisher.New(bc.data.Name, bc.data.Config.Output,
		bc.data.Config.Shipper)
	if err != nil {
		return fmt.Errorf("error initializing publisher: %v", err)
	}

	bc.data.Publisher.RegisterFilter(bc.data.filters)
	err = bc.beater.Setup(bc.data)
	if err != nil {
		return err
	}

	// If -configtest was specified, exit now prior to run.
	if cfgfile.IsTestConfig() {
		fmt.Println("Config OK")
		return GracefulExit
	}

	return nil
}

// run calls the beater Setup and Run methods. In case of errors
// during the setup phase, it exits the process.
func (bc *instance) run() error {
	logp.Info("%s start running.", bc.data.Name)
	return bc.beater.Run(bc.data)
}

// cleanup is invoked prior to exit for the purposes of performing any final
// clean-up. This method is guaranteed to be invoked on shutdown if the beat
// reaches the setup stage.
func (bc *instance) cleanup() error {
	logp.Info("%s cleanup", bc.data.Name)
	defer svc.Cleanup()
	return bc.beater.Cleanup(bc.data)
}

// launch manages the lifecycle of the beat and guarantees the order in which
// the Beater methods are invokes and ensures a a proper exit code is set when
// an error occurs. The exit flag controls if this method calls os.Exit when
// it completes.
func (bc *instance) launch(exit bool) error {
	var err error
	if exit {
		defer func() { exitProcess(err) }()
	}

	err = bc.handleFlags()
	if err != nil {
		return err
	}

	err = bc.config()
	if err != nil {
		return err
	}

	defer bc.cleanup()
	err = bc.setup()
	if err != nil {
		return err
	}

	svc.BeforeRun()
	svc.HandleSignals(bc.beater.Stop)
	err = bc.run()
	return err
}

// exitProcess causes the process to exit. If no error is provided then it will
// exit with code 0. If an error is provided it will set a non-zero exit code
// and log the error logp and to stderr.
//
// The exit code can controlled if the error is an ExitError.
func exitProcess(err error) {
	code := 0
	if ee, ok := err.(ExitError); ok {
		code = ee.ExitCode
	} else if err != nil {
		code = 1
	}

	if err != nil && code != 0 {
		// logp may not be initialized so log the err to stderr too.
		logp.Critical("Exiting: %v", err)
		fmt.Fprintf(os.Stderr, "Exiting: %v\n", err)
	}

	os.Exit(code)
}
