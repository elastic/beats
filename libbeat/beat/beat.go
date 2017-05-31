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
	"encoding/json"
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

	"github.com/satori/go.uuid"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/api"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/dashboards"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring/report"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/plugin"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/publisher"
	svc "github.com/elastic/beats/libbeat/service"
	"github.com/elastic/beats/libbeat/template"
	"github.com/elastic/beats/libbeat/version"

	// Register default processors.
	_ "github.com/elastic/beats/libbeat/processors/actions"
	_ "github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_locale"
	_ "github.com/elastic/beats/libbeat/processors/kubernetes"

	// Register default monitoring reporting
	_ "github.com/elastic/beats/libbeat/monitoring/report/elasticsearch"
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
	Info      common.BeatInfo     // beat metadata.
	RawConfig *common.Config      // Raw config that can be unpacked to get Beat specific config data.
	Config    BeatConfig          // Common Beat configuration data.
	Publisher publisher.Publisher // Publisher
}

// BeatConfig struct contains the basic configuration of every beat
type BeatConfig struct {
	Shipper    publisher.ShipperConfig   `config:",inline"`
	Output     map[string]*common.Config `config:"output"`
	Monitoring *common.Config            `config:"xpack.monitoring"`
	Logging    logp.Logging              `config:"logging"`
	Processors processors.PluginConfig   `config:"processors"`
	Path       paths.Path                `config:"path"`
	Dashboards *common.Config            `config:"setup.dashboards"`
	Template   *common.Config            `config:"setup.template"`
	Http       *common.Config            `config:"http"`
}

var (
	printVersion = flag.Bool("version", false, "Print the version and exit")
	setup        = flag.Bool("setup", false, "Load the sample Kibana dashboards")
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
	return handleError(func() error {
		b, err := newBeat(name, version)
		if err != nil {
			return err
		}
		return b.launch(bt)
	}())
}

// newBeat creates a new beat instance
func newBeat(name, v string) (*Beat, error) {
	if v == "" {
		v = version.GetDefaultVersion()
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &Beat{
		Info: common.BeatInfo{
			Beat:     name,
			Version:  v,
			Name:     hostname,
			Hostname: hostname,
			UUID:     uuid.NewV4(),
		},
	}, nil
}

func (b *Beat) launch(bt Creator) error {
	err := b.handleFlags()
	if err != nil {
		return err
	}

	if err := plugin.Initialize(); err != nil {
		return err
	}

	if err := b.configure(); err != nil {
		return err
	}

	svc.BeforeRun()
	defer svc.Cleanup()

	// load the beats config section
	var sub *common.Config
	configName := strings.ToLower(b.Info.Beat)
	if b.RawConfig.HasField(configName) {
		sub, err = b.RawConfig.Child(configName, -1)
		if err != nil {
			return err
		}
	} else {
		sub = common.NewConfig()
	}

	logp.Info("Setup Beat: %s; Version: %s", b.Info.Beat, b.Info.Version)
	processors, err := processors.New(b.Config.Processors)
	if err != nil {
		return fmt.Errorf("error initializing processors: %v", err)
	}

	err = b.registerTemplateLoading()
	if err != nil {
		return err
	}

	debugf("Initializing output plugins")
	publisher, err := publisher.New(b.Info, b.Config.Output, b.Config.Shipper, processors)
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

	if b.Config.Monitoring.Enabled() {
		reporter, err := report.New(b.Info, b.Config.Monitoring, b.Config.Output)
		if err != nil {
			return err
		}
		defer reporter.Stop()
	}

	// If -configtest was specified, exit now prior to run.
	if cfgfile.IsTestConfig() {
		fmt.Println("Config OK")
		return GracefulExit
	}

	svc.HandleSignals(beater.Stop)

	err = b.loadDashboards()
	if err != nil {
		return err
	}

	logp.Info("%s start running.", b.Info.Beat)
	defer logp.Info("%s stopped.", b.Info.Beat)
	defer logp.LogTotalExpvars(&b.Config.Logging)

	if b.Config.Http.Enabled() {
		api.Start(b.Config.Http, b.Info)
	}

	return beater.Run(b)
}

// handleFlags parses the command line flags. It handles the '-version' flag
// and invokes the HandleFlags callback if implemented by the Beat.
func (b *Beat) handleFlags() error {
	// Due to a dependence upon the beat name, the default config file path
	// must be updated prior to CLI flag handling.
	err := cfgfile.ChangeDefaultCfgfileFlag(b.Info.Beat)
	if err != nil {
		return fmt.Errorf("failed to set default config file path: %v", err)
	}
	flag.Parse()

	if *printVersion {
		fmt.Printf("%s version %s (%s), libbeat %s\n",
			b.Info.Beat, b.Info.Version, runtime.GOARCH, version.GetDefaultVersion())
		return GracefulExit
	}

	if err := logp.HandleFlags(b.Info.Beat); err != nil {
		return err
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

	if name := b.Config.Shipper.Name; name != "" {
		b.Info.Name = name
	}

	err = paths.InitPaths(&b.Config.Path)
	if err != nil {
		return fmt.Errorf("error setting default paths: %v", err)
	}

	err = logp.Init(b.Info.Beat, &b.Config.Logging)
	if err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}
	// Disable stderr logging if requested by cmdline flag
	logp.SetStderr()

	// log paths values to help with troubleshooting
	logp.Info(paths.Paths.String())

	err = b.loadMeta()
	if err != nil {
		return err
	}

	logp.Info("Beat UUID: %v", b.Info.UUID)

	if b.Config.Shipper.MaxProcs != nil {
		maxProcs := *b.Config.Shipper.MaxProcs
		if maxProcs > 0 {
			runtime.GOMAXPROCS(maxProcs)
		}
	}

	return nil
}

func (b *Beat) loadMeta() error {
	type meta struct {
		UUID uuid.UUID `json:"uuid"`
	}

	metaPath := paths.Resolve(paths.Data, "meta.json")
	logp.Info("Beat metadata path: %v", metaPath)

	f, err := openRegular(metaPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Beat meta file failed to open: %s", err)
	}

	if err == nil {
		m := meta{}
		if err := json.NewDecoder(f).Decode(&m); err != nil {
			f.Close()
			return fmt.Errorf("Beat meta file reading error: %v", err)
		}

		f.Close()
		valid := !uuid.Equal(m.UUID, uuid.Nil)
		if valid {
			b.Info.UUID = m.UUID
			return nil
		}
	}

	// file does not exist or UUID is invalid, let's create a new one

	// write temporary file first
	tempFile := metaPath + ".new"
	f, err = os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Failed to create Beat meta file: %s", err)
	}

	err = json.NewEncoder(f).Encode(meta{UUID: b.Info.UUID})
	f.Close()
	if err != nil {
		return fmt.Errorf("Beat meta file failed to write: %s", err)
	}

	// move temporary file into final location
	err = file.SafeFileRotate(metaPath, tempFile)
	return err
}

func openRegular(filename string) (*os.File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return f, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if !info.Mode().IsRegular() {
		f.Close()
		if info.IsDir() {
			return nil, fmt.Errorf("%s is a directory", filename)
		}
		return nil, fmt.Errorf("%s is not a regular file", filename)
	}

	return f, nil
}

func (b *Beat) loadDashboards() error {
	if *setup {
		// -setup implies dashboards.enabled=true
		if b.Config.Dashboards == nil {
			b.Config.Dashboards = common.NewConfig()
		}
		err := b.Config.Dashboards.SetBool("enabled", -1, true)
		if err != nil {
			return fmt.Errorf("Error setting dashboard.enabled=true: %v", err)
		}
	}

	if b.Config.Dashboards != nil && b.Config.Dashboards.Enabled() {
		esConfig := b.Config.Output["elasticsearch"]
		if esConfig == nil || !esConfig.Enabled() {
			return fmt.Errorf("Dashboard loading requested but the Elasticsearch output is not configured/enabled")
		}
		esClient, err := elasticsearch.NewConnectedClient(esConfig)
		if err != nil {
			return fmt.Errorf("Error creating ES client: %v", err)
		}
		defer esClient.Close()

		err = dashboards.ImportDashboards(b.Info.Beat, b.Info.Version, esClient, b.Config.Dashboards)
		if err != nil {
			return fmt.Errorf("Error importing Kibana dashboards: %v", err)
		}
		logp.Info("Kibana dashboards successfully loaded.")
	}

	return nil
}

// registerTemplateLoading registers the loading of the template as a callback with
// the elasticsearch output. It is important the the registration happens before
// the publisher is created.
func (b *Beat) registerTemplateLoading() error {
	if *setup {
		// -setup implies template.enabled=true
		if b.Config.Template == nil {
			b.Config.Template = common.NewConfig()
		}
		err := b.Config.Template.SetBool("enabled", -1, true)
		if err != nil {
			return fmt.Errorf("Error setting template.enabled=true: %v", err)
		}
	}

	// Check if outputting to file is enabled, and output to file if it is
	if b.Config.Template != nil && b.Config.Template.Enabled() {
		var cfg template.TemplateConfig
		err := b.Config.Template.Unpack(&cfg)
		if err != nil {
			return fmt.Errorf("unpacking template config fails: %v", err)
		}
		if len(cfg.OutputToFile.Path) > 0 {
			// output to file is enabled
			loader, err := template.NewLoader(b.Config.Template, nil, b.Info)
			if err != nil {
				return fmt.Errorf("Error creating Elasticsearch template loader: %v", err)
			}
			err = loader.Generate()
			if err != nil {
				return fmt.Errorf("Error generating template: %v", err)
			}

			// XXX: Should we kill the Beat here or just continue?
			return fmt.Errorf("Stopping after successfully writing the template to the file.")
		}
	}

	esConfig := b.Config.Output["elasticsearch"]
	// Loads template by default if esOutput is enabled
	if (b.Config.Template == nil && esConfig.Enabled()) || (b.Config.Template != nil && b.Config.Template.Enabled()) {
		if esConfig == nil || !esConfig.Enabled() {
			return fmt.Errorf("Template loading requested but the Elasticsearch output is not configured/enabled")
		}

		// load template through callback to make sure it is also loaded
		// on reconnecting
		callback := func(esClient *elasticsearch.Client) error {

			if b.Config.Template == nil {
				b.Config.Template = common.NewConfig()
			}

			loader, err := template.NewLoader(b.Config.Template, esClient, b.Info)
			if err != nil {
				return fmt.Errorf("Error creating Elasticsearch template loader: %v", err)
			}

			err = loader.Load()
			if err != nil {
				return fmt.Errorf("Error loading Elasticsearch template: %v", err)
			}

			return nil
		}

		elasticsearch.RegisterConnectCallback(callback)
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
