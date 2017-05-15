// Package harvester harvests different inputs for new information. Currently
// two harvester types exist:
//
//   * log
//   * stdin
//
//  The log harvester reads a file line by line. In case the end of a file is found
//  with an incomplete line, the line pointer stays at the beginning of the incomplete
//  line. As soon as the line is completed, it is read and returned.
//
//  The stdin harvesters reads data from stdin.
package log

import (
	"errors"
	"fmt"
	"sync"

	"github.com/satori/go.uuid"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
)

var (
	ErrFileTruncate = errors.New("detected file being truncated")
	ErrRenamed      = errors.New("file was renamed")
	ErrRemoved      = errors.New("file was removed")
	ErrInactive     = errors.New("file inactive")
	ErrClosed       = errors.New("reader closed")
)

type Harvester struct {
	forwarder       *harvester.Forwarder
	config          config
	state           file.State
	states          *file.States
	file            source.FileSource /* the file being watched */
	fileReader      *LogFile
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	done            chan struct{}
	stopOnce        sync.Once
	stopWg          *sync.WaitGroup
	id              uuid.UUID
	reader          reader.Reader
}

func NewHarvester(
	config *common.Config,
	state file.State,
	states *file.States,
	outlet harvester.Outlet,
) (*Harvester, error) {

	h := &Harvester{
		config: defaultConfig,
		state:  state,
		states: states,
		done:   make(chan struct{}),
		stopWg: &sync.WaitGroup{},
		id:     uuid.NewV4(),
	}

	if err := config.Unpack(&h.config); err != nil {
		return nil, err
	}

	encodingFactory, ok := encoding.FindEncoding(h.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.config.Encoding)
	}
	h.encodingFactory = encodingFactory

	// Add ttl if clean_inactive is set
	if h.config.CleanInactive > 0 {
		h.state.TTL = h.config.CleanInactive
	}

	// Add outlet signal so harvester can also stop itself
	outlet.SetSignal(h.done)

	var err error
	h.forwarder, err = harvester.NewForwarder(config, outlet)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// open does open the file given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() error {

	switch h.config.Type {
	case cfg.StdinType:
		return h.openStdin()
	case cfg.LogType:
		return h.openFile()
	default:
		return fmt.Errorf("Invalid harvester type: %+v", h.config)
	}
}

// ID returns the unique harvester identifier
func (h *Harvester) ID() uuid.UUID {
	return h.id
}
