package multiline

import (
	"errors"
	"flag"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
)

type prospectorConfig struct {
	Encoding  string                 `config:"encoding"`
	MaxBytes  int                    `config:"max_bytes" validate:"min=0,nonzero"`
	Multiline reader.MultilineConfig `config:"multiline"`
}

// flags
var (
	negate       bool
	pattern      string
	matchType    string
	maxLines     int
	maxBytes     int
	flushPattern string
	codec        string

	prospectorID int
)

func addFlags(cmd *cobra.Command) {
	fs := cmd.PersistentFlags()
	fs.BoolVarP(&negate, "negate", "n", false, "Negate the pattern matching")
	fs.StringVarP(&pattern, "pattern", "p", "", "Multiline regex pattern")
	fs.StringVarP(&matchType, "match", "m", "after", "Multiline match type (before/after)")
	fs.IntVarP(&maxLines, "lines", "l", 0, "Maximum number of lines per event")
	fs.IntVar(&maxBytes, "max_bytes", 0, "Maximum bytes per event")
	fs.StringVar(&flushPattern, "flush", "", "Multiline flush pattern")
	fs.StringVar(&codec, "encoding", "", "File encoding")
	fs.IntVarP(&prospectorID, "idx", "i", 0, "Select prospector from filebeat config")
}

func loadConfig() (*prospectorConfig, error) {
	if checkConfigFileSet() {
		loadConfigFromFile()
	}

	config := &prospectorConfig{}
	config.Encoding = codec
	config.MaxBytes = maxBytes
	config.Multiline.Negate = negate
	config.Multiline.Match = matchType

	if maxLines > 0 {
		config.Multiline.MaxLines = &maxLines
	}

	if pattern == "" {
		return nil, errors.New("pattern is required")
	}
	tmp, err := match.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling pattern failed with %v", err)
	}
	config.Multiline.Pattern = &tmp

	if flushPattern != "" {
		tmp, err := match.Compile(flushPattern)
		if err != nil {
			return nil, fmt.Errorf("compiling flush pattern failed with %v", err)
		}
		config.Multiline.FlushPattern = &tmp
	}

	return config, nil
}

func loadConfigFromFile() (*prospectorConfig, error) {
	// load config file

	cfg, err := cfgfile.Load("")
	if err != nil {
		return nil, err
	}

	var prospectors = struct {
		List []*common.Config `config:"filebeat.prospectors"`
	}{}
	if err := cfg.Unpack(&prospectors); err != nil {
		return nil, err
	}

	if prospectorID < 0 {
		return nil, fmt.Errorf("prospector id %v is invalid", prospectorID)
	}
	if L := prospectors.List; prospectorID >= len(L) {
		return nil, fmt.Errorf("prospector list of length %v is less then index %v", L, prospectorID)
	}

	cfg = prospectors.List[prospectorID]
	prospector := prospectorConfig{}
	if err := cfg.Unpack(&prospector); err != nil {
		return nil, err
	}

	return &prospector, nil
}

func checkConfigFileSet() bool {
	f := flag.Lookup("c")
	if f == nil {
		return false
	}

	type tester interface {
		IsSet() bool
	}

	t, ok := f.Value.(tester)
	return ok && t.IsSet()
}
