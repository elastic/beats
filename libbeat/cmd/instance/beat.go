package instance

import (
	"context"
	cryptRand "crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/elastic/beats/libbeat/api"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cloudid"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/dashboards"
	"github.com/elastic/beats/libbeat/keystore"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/logp/configure"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/report"
	"github.com/elastic/beats/libbeat/monitoring/report/log"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/plugin"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	svc "github.com/elastic/beats/libbeat/service"
	"github.com/elastic/beats/libbeat/template"
	"github.com/elastic/beats/libbeat/version"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"

	// Register publisher pipeline modules
	_ "github.com/elastic/beats/libbeat/publisher/includes"

	// Register default processors.
	_ "github.com/elastic/beats/libbeat/processors/actions"
	_ "github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_host_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_locale"

	// Register autodiscover providers
	_ "github.com/elastic/beats/libbeat/autodiscover/providers/docker"
	_ "github.com/elastic/beats/libbeat/autodiscover/providers/kubernetes"

	// Register default monitoring reporting
	_ "github.com/elastic/beats/libbeat/monitoring/report/elasticsearch"
)

// Beat provides the runnable and configurable instance of a beat.
type Beat struct {
	beat.Beat

	Config    beatConfig
	RawConfig *common.Config // Raw config that can be unpacked to get Beat specific config data.
	keystore  keystore.Keystore
}

type beatConfig struct {
	beat.BeatConfig `config:",inline"`

	// instance internal configs

	// beat top-level settings
	Name     string `config:"name"`
	MaxProcs int    `config:"max_procs"`

	// beat internal components configurations
	HTTP          *common.Config `config:"http"`
	Path          paths.Path     `config:"path"`
	Logging       *common.Config `config:"logging"`
	MetricLogging *common.Config `config:"logging.metrics"`
	Keystore      *common.Config `config:"keystore"`

	// output/publishing related configurations
	Pipeline   pipeline.Config `config:",inline"`
	Monitoring *common.Config  `config:"xpack.monitoring"`

	// elastic stack 'setup' configurations
	Dashboards *common.Config `config:"setup.dashboards"`
	Template   *common.Config `config:"setup.template"`
	Kibana     *common.Config `config:"setup.kibana"`
}

var (
	printVersion bool
	setup        bool
)

var debugf = logp.MakeDebug("beat")

func init() {
	initRand()

	flag.BoolVar(&printVersion, "version", false, "Print the version and exit")
	flag.BoolVar(&setup, "setup", false, "Load sample Kibana dashboards and setup Machine Learning")
}

// initRand initializes the runtime random number generator seed using
// global, shared cryptographically strong pseudo random number generator.
//
// On linux Reader might use getrandom(2) or /udev/random. On windows systems
// CryptGenRandom is used.
func initRand() {
	n, err := cryptRand.Int(cryptRand.Reader, big.NewInt(math.MaxInt64))
	seed := n.Int64()
	if err != nil {
		// fallback to current timestamp
		seed = time.Now().UnixNano()
	}

	rand.Seed(seed)
}

// Run initializes and runs a Beater implementation. name is the name of the
// Beat (e.g. packetbeat or metricbeat). version is version number of the Beater
// implementation. bt is the `Creator` callback for creating a new beater
// instance.
// XXX Move this as a *Beat method?
func Run(name, idxPrefix, version string, bt beat.Creator) error {
	return handleError(func() error {
		defer func() {
			if r := recover(); r != nil {
				logp.NewLogger(name).Fatalw("Failed due to panic.",
					"panic", r, zap.Stack("stack"))
			}
		}()
		b, err := NewBeat(name, idxPrefix, version)
		if err != nil {
			return err
		}
		return b.launch(bt)
	}())
}

// NewBeat creates a new beat instance
func NewBeat(name, indexPrefix, v string) (*Beat, error) {
	if v == "" {
		v = version.GetDefaultVersion()
	}
	if indexPrefix == "" {
		indexPrefix = name
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	b := beat.Beat{
		Info: beat.Info{
			Beat:        name,
			IndexPrefix: indexPrefix,
			Version:     v,
			Name:        hostname,
			Hostname:    hostname,
			UUID:        uuid.NewV4(),
		},
	}

	return &Beat{Beat: b}, nil
}

// init does initialization of things common to all actions (read confs, flags)
func (b *Beat) Init() error {
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

	return nil
}

// BeatConfig returns config section for this beat
func (b *Beat) BeatConfig() (*common.Config, error) {
	configName := strings.ToLower(b.Info.Beat)
	if b.RawConfig.HasField(configName) {
		sub, err := b.RawConfig.Child(configName, -1)
		if err != nil {
			return nil, err
		}

		return sub, nil
	}

	return common.NewConfig(), nil
}

// Keystore return the configured keystore for this beat
func (b *Beat) Keystore() keystore.Keystore {
	return b.keystore
}

// create and return the beater, this method also initializes all needed items,
// including template registering, publisher, xpack monitoring
func (b *Beat) createBeater(bt beat.Creator) (beat.Beater, error) {
	sub, err := b.BeatConfig()
	if err != nil {
		return nil, err
	}

	logSystemInfo(b.Info)
	logp.Info("Setup Beat: %s; Version: %s", b.Info.Beat, b.Info.Version)

	err = b.registerTemplateLoading()
	if err != nil {
		return nil, err
	}

	reg := monitoring.Default.GetRegistry("libbeat")
	if reg == nil {
		reg = monitoring.Default.NewRegistry("libbeat")
	}

	err = setupMetrics(b.Info.Beat)
	if err != nil {
		return nil, err
	}

	debugf("Initializing output plugins")
	pipeline, err := pipeline.Load(b.Info, reg, b.Config.Pipeline, b.Config.Output)
	if err != nil {
		return nil, fmt.Errorf("error initializing publisher: %v", err)
	}

	// TODO: some beats race on shutdown with publisher.Stop -> do not call Stop yet,
	//       but refine publisher to disconnect clients on stop automatically
	// defer pipeline.Close()

	b.Publisher = pipeline
	beater, err := bt(&b.Beat, sub)
	if err != nil {
		return nil, err
	}

	return beater, nil
}

func (b *Beat) launch(bt beat.Creator) error {
	defer logp.Sync()
	defer logp.Info("%s stopped.", b.Info.Beat)

	err := b.Init()
	if err != nil {
		return err
	}

	svc.BeforeRun()
	defer svc.Cleanup()

	beater, err := b.createBeater(bt)
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

	if b.Config.MetricLogging == nil || b.Config.MetricLogging.Enabled() {
		reporter, err := log.MakeReporter(b.Info, b.Config.MetricLogging)
		if err != nil {
			return err
		}
		defer reporter.Stop()
	}

	// If -configtest was specified, exit now prior to run.
	if cfgfile.IsTestConfig() {
		cfgwarn.Deprecate("6.0", "-configtest flag has been deprecated, use configtest subcommand")
		fmt.Println("Config OK")
		return beat.GracefulExit
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc.HandleSignals(beater.Stop, cancel)

	err = b.loadDashboards(ctx, false)
	if err != nil {
		return err
	}
	if setup && b.SetupMLCallback != nil {
		err = b.SetupMLCallback(&b.Beat, b.Config.Kibana)
		if err != nil {
			return err
		}
	}

	logp.Info("%s start running.", b.Info.Beat)

	if b.Config.HTTP.Enabled() {
		api.Start(b.Config.HTTP, b.Info)
	}

	return beater.Run(&b.Beat)
}

// TestConfig check all settings are ok and the beat can be run
func (b *Beat) TestConfig(bt beat.Creator) error {
	return handleError(func() error {
		err := b.Init()
		if err != nil {
			return err
		}

		// Create beater to ensure all settings are OK
		_, err = b.createBeater(bt)
		if err != nil {
			return err
		}

		fmt.Println("Config OK")
		return beat.GracefulExit
	}())
}

// Setup registers ES index template and kibana dashboards
func (b *Beat) Setup(bt beat.Creator, template, dashboards, machineLearning, pipelines bool) error {
	return handleError(func() error {
		err := b.Init()
		if err != nil {
			return err
		}

		// Tell the beat that we're in the setup command
		b.InSetupCmd = true

		// Create beater to give it the opportunity to set loading callbacks
		_, err = b.createBeater(bt)
		if err != nil {
			return err
		}

		if template {
			outCfg := b.Config.Output

			if outCfg.Name() != "elasticsearch" {
				return fmt.Errorf("Template loading requested but the Elasticsearch output is not configured/enabled")
			}

			esConfig := outCfg.Config()
			if tmplCfg := b.Config.Template; tmplCfg == nil || tmplCfg.Enabled() {
				loadCallback, err := b.templateLoadingCallback()
				if err != nil {
					return err
				}

				esClient, err := elasticsearch.NewConnectedClient(esConfig)
				if err != nil {
					return err
				}

				// Load template
				err = loadCallback(esClient)
				if err != nil {
					return err
				}
			}

			fmt.Println("Loaded index template")
		}

		if dashboards {
			fmt.Println("Loading dashboards (Kibana must be running and reachable)")
			err = b.loadDashboards(context.Background(), true)
			if err != nil {
				return err
			}

			fmt.Println("Loaded dashboards")
		}

		if machineLearning && b.SetupMLCallback != nil {
			err = b.SetupMLCallback(&b.Beat, b.Config.Kibana)
			if err != nil {
				return err
			}
			fmt.Println("Loaded machine learning job configurations")
		}

		if pipelines && b.OverwritePipelinesCallback != nil {
			esConfig := b.Config.Output.Config()
			err = b.OverwritePipelinesCallback(esConfig)
			if err != nil {
				return err
			}

			fmt.Println("Loaded Ingest pipelines")
		}

		return nil
	}())
}

// handleFlags parses the command line flags. It handles the '-version' flag
// and invokes the HandleFlags callback if implemented by the Beat.
func (b *Beat) handleFlags() error {
	flag.Parse()

	if printVersion {
		cfgwarn.Deprecate("6.0", "-version flag has been deprecated, use version subcommand")
		fmt.Printf("%s version %s (%s), libbeat %s\n",
			b.Info.Beat, b.Info.Version, runtime.GOARCH, version.GetDefaultVersion())
		return beat.GracefulExit
	}

	return cfgfile.HandleFlags()
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

	// We have to initialize the keystore before any unpack or merging the cloud
	// options.
	keystoreCfg, _ := cfg.Child("keystore", -1)
	defaultPathConfig, _ := cfg.String("path.config", -1)
	defaultPathConfig = filepath.Join(defaultPathConfig, fmt.Sprintf("%s.keystore", b.Info.Beat))
	store, err := keystore.Factory(keystoreCfg, defaultPathConfig)
	if err != nil {
		return fmt.Errorf("could not initialize the keystore: %v", err)
	}

	// TODO: Allow the options to be more flexible for dynamic changes
	common.OverwriteConfigOpts(keystore.ConfigOpts(store))
	b.keystore = store
	err = cloudid.OverwriteSettings(cfg)
	if err != nil {
		return err
	}

	b.RawConfig = cfg
	err = cfg.Unpack(&b.Config)
	if err != nil {
		return fmt.Errorf("error unpacking config data: %v", err)
	}

	b.Beat.Config = &b.Config.BeatConfig

	err = cfgwarn.CheckRemoved5xSettings(cfg, "queue_size", "bulk_queue_size")
	if err != nil {
		return err
	}

	if name := b.Config.Name; name != "" {
		b.Info.Name = name
	}

	err = paths.InitPaths(&b.Config.Path)
	if err != nil {
		return fmt.Errorf("error setting default paths: %v", err)
	}

	if err := configure.Logging(b.Info.Beat, b.Config.Logging); err != nil {
		return fmt.Errorf("error initializing logging: %v", err)
	}

	// log paths values to help with troubleshooting
	logp.Info(paths.Paths.String())

	err = b.loadMeta()
	if err != nil {
		return err
	}

	logp.Info("Beat UUID: %v", b.Info.UUID)

	if maxProcs := b.Config.MaxProcs; maxProcs > 0 {
		runtime.GOMAXPROCS(maxProcs)
	}

	b.Beat.BeatConfig, err = b.BeatConfig()
	if err != nil {
		return err
	}

	return nil
}

func (b *Beat) loadMeta() error {
	type meta struct {
		UUID uuid.UUID `json:"uuid"`
	}

	metaPath := paths.Resolve(paths.Data, "meta.json")
	logp.Debug("beat", "Beat metadata path: %v", metaPath)

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

func (b *Beat) loadDashboards(ctx context.Context, force bool) error {
	if setup || force {
		// -setup implies dashboards.enabled=true
		if b.Config.Dashboards == nil {
			b.Config.Dashboards = common.NewConfig()
		}
		err := b.Config.Dashboards.SetBool("enabled", -1, true)
		if err != nil {
			return fmt.Errorf("Error setting dashboard.enabled=true: %v", err)
		}
	}

	if b.Config.Dashboards.Enabled() {
		var esConfig *common.Config

		if b.Config.Output.Name() == "elasticsearch" {
			esConfig = b.Config.Output.Config()
		}
		err := dashboards.ImportDashboards(ctx, b.Info.Beat, b.Info.Hostname, paths.Resolve(paths.Home, ""),
			b.Config.Kibana, esConfig, b.Config.Dashboards, nil)
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

	var cfg template.TemplateConfig

	// Check if outputting to file is enabled, and output to file if it is
	if b.Config.Template.Enabled() {
		err := b.Config.Template.Unpack(&cfg)
		if err != nil {
			return fmt.Errorf("unpacking template config fails: %v", err)
		}
	}

	// Loads template by default if esOutput is enabled
	if b.Config.Output.Name() == "elasticsearch" {

		// Get ES Index name for comparison
		esCfg := struct {
			Index string `config:"index"`
		}{}
		err := b.Config.Output.Config().Unpack(&esCfg)
		if err != nil {
			return err
		}

		if esCfg.Index != "" && (cfg.Name == "" || cfg.Pattern == "") && (b.Config.Template == nil || b.Config.Template.Enabled()) {
			return fmt.Errorf("setup.template.name and setup.template.pattern have to be set if index name is modified.")
		}

		if b.Config.Template == nil || (b.Config.Template != nil && b.Config.Template.Enabled()) {

			// load template through callback to make sure it is also loaded
			// on reconnecting
			callback, err := b.templateLoadingCallback()
			if err != nil {
				return err
			}
			elasticsearch.RegisterConnectCallback(callback)
		}
	}

	return nil
}

// Build and return a callback to load index template into ES
func (b *Beat) templateLoadingCallback() (func(esClient *elasticsearch.Client) error, error) {
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

	return callback, nil
}

// handleError handles the given error by logging it and then returning the
// error. If the err is nil or is a GracefulExit error then the method will
// return nil without logging anything.
func handleError(err error) error {
	if err == nil || err == beat.GracefulExit {
		return nil
	}

	// logp may not be initialized so log the err to stderr too.
	logp.Critical("Exiting: %v", err)
	fmt.Fprintf(os.Stderr, "Exiting: %v\n", err)
	return err
}

// logSystemInfo logs information about this system for situational awareness
// in debugging. This information includes data about the beat, build, go
// runtime, host, and process. If any of the data is not available it will be
// omitted.
func logSystemInfo(info beat.Info) {
	defer logp.Recover("An unexpected error occurred while collecting " +
		"information about the system.")
	log := logp.NewLogger("beat").With(logp.Namespace("system_info"))

	// Beat
	beat := common.MapStr{
		"type": info.Beat,
		"uuid": info.UUID,
		"path": common.MapStr{
			"config": paths.Resolve(paths.Config, ""),
			"data":   paths.Resolve(paths.Data, ""),
			"home":   paths.Resolve(paths.Home, ""),
			"logs":   paths.Resolve(paths.Logs, ""),
		},
	}
	log.Infow("Beat info", "beat", beat)

	// Build
	build := common.MapStr{
		"commit":  version.Commit(),
		"time":    version.BuildTime(),
		"version": info.Version,
		"libbeat": version.GetDefaultVersion(),
	}
	log.Infow("Build info", "build", build)

	// Go Runtime
	log.Infow("Go runtime info", "go", sysinfo.Go())

	// Host
	if host, err := sysinfo.Host(); err == nil {
		log.Infow("Host info", "host", host.Info())
	}

	// Process
	if self, err := sysinfo.Self(); err == nil {
		process := common.MapStr{}

		if info, err := self.Info(); err == nil {
			process["name"] = info.Name
			process["pid"] = info.PID
			process["ppid"] = info.PPID
			process["cwd"] = info.CWD
			process["exe"] = info.Exe
			process["start_time"] = info.StartTime
		}

		if proc, ok := self.(types.Seccomp); ok {
			if seccomp, err := proc.Seccomp(); err == nil {
				process["seccomp"] = seccomp
			}
		}

		if proc, ok := self.(types.Capabilities); ok {
			if caps, err := proc.Capabilities(); err == nil {
				process["capabilities"] = caps
			}
		}

		if len(process) > 0 {
			log.Infow("Process info", "process", process)
		}
	}
}
