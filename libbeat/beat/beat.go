/*
Package beat provides the functions required to manage the life-cycle of a Beat.
It provides the standard mechanism for launching a Beat. It manages
configuration, logging, and publisher initialization and registers a signal
handler to gracefully stop the process.

Each Beat implementation must implement the `Beater` interface and a `Creator`
to create and initialize the Beater instance. See the `Beater` interface and `Creator`
documentation for more details.

To use this package, create a simple main that invokes the Run() function.

  func main() {
  	if err := beat.Run("mybeat", myVersion, beater.New); err != nil {
  		os.Exit(1)
  	}
  }

In the example above, the beater package contains the implementation of the
Beater interface and the New method returns a new instance of Beater. The
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
	"errors"
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/processors"
	_ "github.com/elastic/beats/libbeat/processors/actions"
	"github.com/elastic/beats/libbeat/publisher"
	svc "github.com/elastic/beats/libbeat/service"
	"github.com/satori/go.uuid"
)

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

// Creator initializes and configures a new Beater instance used to execute
// the beat its run-loop.
type Creator func(*Beat, *common.Config) (Beater, error)

// Beat contains the basic beat data and the publisher client used to publish
// events.
type Beat struct {
	Name      string              // Beat name.
	Version   string              // Beat version number. Defaults to the libbeat version when an implementation does not set a version.
	UUID      uuid.UUID           // ID assigned to a Beat instance.
	RawConfig *common.Config      // Raw config that can be unpacked to get Beat specific config data.
	Config    BeatConfig          // Common Beat configuration data.
	Publisher publisher.Publisher // Publisher
}

// BeatConfig struct contains the basic configuration of every beat
type BeatConfig struct {
	Shipper    publisher.ShipperConfig   `config:",inline"`
	Output     map[string]*common.Config `config:"output"`
	Logging    logp.Logging              `config:"logging"`
	Processors processors.PluginConfig   `config:"processors"`
	Path       paths.Path                `config:"path"`
}

var (
	printVersion = flag.Bool("version", false, "Print the version and exit")
)

var debugf = logp.MakeDebug("beat")

// GracefulExit is an error that signals to exit with a code of 0.
var GracefulExit = errors.New("graceful exit")

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

// Run initializes and runs a Beater implementation. name is the name of the
// Beat (e.g. packetbeat or metricbeat). version is version number of the Beater
// implementation. bt is the `Creator` callback for creating a new beater
// instance.
func Run(name, version string, bt Creator) error {
	return handleError(newBeat(name, version).launch(bt))
}

// newBeat creates a new beat instance
func newBeat(name, version string) *Beat {
	if version == "" {
		version = defaultBeatVersion
	}

	return &Beat{
		Name:    name,
		Version: version,
		UUID:    uuid.NewV4(),
	}
}

func (b *Beat) launch(bt Creator) error {
	err := b.handleFlags()
	if err != nil {
		return err
	}

	svc.BeforeRun()
	defer svc.Cleanup()

	if err := b.configure(); err != nil {
		return err
	}

	// load the beats config section
	var sub *common.Config
	configName := strings.ToLower(b.Name)
	if b.RawConfig.HasField(configName) {
		sub, err = b.RawConfig.Child(configName, -1)
		if err != nil {
			return err
		}
	} else {
		sub = common.NewConfig()
	}

	logp.Info("Setup Beat: %s; Version: %s", b.Name, b.Version)
	processors, err := processors.New(b.Config.Processors)
	if err != nil {
		return fmt.Errorf("error initializing processors: %v", err)
	}

	debugf("Initializing output plugins")
	publisher, err := publisher.New(b.Name, b.Config.Output, b.Config.Shipper, processors)
	if err != nil {
		return fmt.Errorf("error initializing publisher: %v", err)
	}

	// TODO: some beats race on shutdown with publisher.Stop -> do not call Stop yet,
	//       but refine publisher to disconnect clients on stop automatically
	// defer publisher.Stop()

	b.Publisher = publisher
	beater, err := bt(b, sub)
	if err != nil {
		return err
	}

	// If -configtest was specified, exit now prior to run.
	if cfgfile.IsTestConfig() {
		fmt.Println("Config OK")
		return GracefulExit
	}

	svc.HandleSignals(beater.Stop)

	logp.Info("%s start running.", b.Name)
	defer logp.Info("%s stopped.", b.Name)
	defer logp.LogTotalExpvars(&b.Config.Logging)

	return beater.Run(b)
}

// handleFlags parses the command line flags. It handles the '-version' flag
// and invokes the HandleFlags callback if implemented by the Beat.
func (b *Beat) handleFlags() error {
	// Due to a dependence upon the beat name, the default config file path
	// must be updated prior to CLI flag handling.
	err := cfgfile.ChangeDefaultCfgfileFlag(b.Name)
	if err != nil {
		return fmt.Errorf("failed to set default config file path: %v", err)
	}
	flag.Parse()

	if *printVersion {
		fmt.Printf("%s version %s (%s), libbeat %s\n",
			b.Name, b.Version, runtime.GOARCH, defaultBeatVersion)
		return GracefulExit
	}

	if err := cfgfile.HandleFlags(); err != nil {
		return err
	}
	return handleFlags(b)
}

// config reads the configuration file from disk, parses the common options
// defined in BeatConfig, initializes logging, and set GOMAXPROCS if defined
// in the config. Lastly it invokes the Config method implemented by the beat.
func (b *Beat) configure() error {
	var err error

	cfg, err := cfgfile.Load("")
	if err != nil {
		return fmt.Errorf("error loading config file: %v", err)
	}

	b.RawConfig = cfg
	err = cfg.Unpack(&b.Config)
	if err != nil {
		return fmt.Errorf("error unpacking config data: %v", err)
	}

	err = paths.InitPaths(&b.Config.Path)
	if err != nil {
		return fmt.Errorf("error setting default paths: %v", err)
	}

	err = logp.Init(b.Name, &b.Config.Logging)
	if err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}
	// Disable stderr logging if requested by cmdline flag
	logp.SetStderr()

	// log paths values to help with troubleshooting
	logp.Info(paths.Paths.String())

	if b.Config.Shipper.MaxProcs != nil {
		maxProcs := *b.Config.Shipper.MaxProcs
		if maxProcs > 0 {
			runtime.GOMAXPROCS(maxProcs)
		}
	}

	return nil
}

// handleError handles the given error by logging it and then returning the
// error. If the err is nil or is a GracefulExit error then the method will
// return nil without logging anything.
func handleError(err error) error {
	if err == nil || err == GracefulExit {
		return nil
	}

	// logp may not be initialized so log the err to stderr too.
	logp.Critical("Exiting: %v", err)
	fmt.Fprintf(os.Stderr, "Exiting: %v\n", err)
	return err
}

// GetDefaultVersion returns the current libbeat version.
func GetDefaultVersion() string {
	return defaultBeatVersion
}
