package crawler

import (
	"flag"
	"github.com/elastic/libbeat/cfgfile"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Filebeat FilebeatConfig
}

type FilebeatConfig struct {
	Files               []FileConfig
	SpoolSize           uint64
	HarvesterBufferSize int
	CpuProfileFile      string
	IdleTimeout         time.Duration
	TailOnRotate        bool
	Quiet               bool
}

type FileConfig struct {
	Paths        []string
	Fields       map[string]string
	DeadTime     string
	Input        string
	DeadtimeSpan time.Duration
}

// TODO: Log is only used here now. Do we need it?
const logflags = log.Ldate | log.Ltime | log.Lmicroseconds

func init() {
	log.SetFlags(logflags)
}

// TODO: Default options should be set as part of default config otherwise it always overwrites
var CmdlineOptions = &FilebeatConfig{
	SpoolSize:           1024,
	HarvesterBufferSize: 16 << 10, // 16384
	IdleTimeout:         time.Second * 5,
}

func init() {

	flag.Uint64Var(&CmdlineOptions.SpoolSize, "spool-size", CmdlineOptions.SpoolSize, "event count spool threshold - forces network flush")
	flag.Uint64Var(&CmdlineOptions.SpoolSize, "sv", CmdlineOptions.SpoolSize, "event count spool threshold - forces network flush")

	flag.IntVar(&CmdlineOptions.HarvesterBufferSize, "harvest-buffer-size", CmdlineOptions.HarvesterBufferSize, "harvester reader buffer size")
	flag.IntVar(&CmdlineOptions.HarvesterBufferSize, "hb", CmdlineOptions.HarvesterBufferSize, "harvester reader buffer size")

	flag.BoolVar(&CmdlineOptions.TailOnRotate, "tail", CmdlineOptions.TailOnRotate, "always tail on log rotation -note: may skip entries ")
	flag.BoolVar(&CmdlineOptions.TailOnRotate, "t", CmdlineOptions.TailOnRotate, "always tail on log rotation -note: may skip entries ")

	flag.BoolVar(&CmdlineOptions.Quiet, "quiet", CmdlineOptions.Quiet, "operate in quiet mode - only emit errors to log")
}

// getConfigFiles returns list of config files.
// In case path is a file, it will be directly returned.
// In case it is a directory, it will fetch all .yml files inside this directory
func getConfigFiles(path string) (configFiles []string, err error) {

	// Check if path is valid file or dir
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Create empty slice for config file list
	configFiles = make([]string, 0)

	if stat.IsDir() {
		files, err := filepath.Glob(path + "/*.yml")

		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, files...)

	} else {
		// Only 1 config file
		configFiles = append(configFiles, path)
	}

	return configFiles, nil
}

// mergeConfigFiles reads in all config files given by list configFiles and merges them into config
func mergeConfigFiles(configFiles []string, config *Config) error {

	for _, file := range configFiles {
		tmpConfig := &Config{}
		cfgfile.Read(tmpConfig, file)

		config.Filebeat.Files = append(config.Filebeat.Files, tmpConfig.Filebeat.Files...)
	}

	return nil
}

// Fetches and merges all config files given by Options.configArgs. All are put into one config object
func FetchConfigs(path string) *Config {

	config := &Config{}

	configFiles, err := getConfigFiles(path)

	if err != nil {
		log.Fatal("Could not use -config of ", path, err)
	}

	err = mergeConfigFiles(configFiles, config)

	if err != nil {
		log.Fatal("Error merging config files: ", err)
	}

	if len(config.Filebeat.Files) == 0 {
		log.Fatalf("No paths given. What files do you want me to watch?")
	}

	return config
}
