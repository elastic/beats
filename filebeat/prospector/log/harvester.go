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
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

var (
	ErrFileTruncate = errors.New("detected file being truncated")
	ErrRenamed      = errors.New("file was renamed")
	ErrRemoved      = errors.New("file was removed")
	ErrInactive     = errors.New("file inactive")
	ErrClosed       = errors.New("reader closed")
)

type Outlet interface {
	SetSignal(signal <-chan struct{})
	OnEventSignal(data *util.Data) bool
	OnEvent(data *util.Data) bool
}

type Harvester struct {
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
	outlet          Outlet
	id              uuid.UUID
	processors      *processors.Processors
	reader          reader.Reader
}

func NewHarvester(
	config *common.Config,
	state file.State,
	states *file.States,
	outlet Outlet,
) (*Harvester, error) {

	h := &Harvester{
		config: defaultConfig,
		state:  state,
		states: states,
		done:   make(chan struct{}),
		stopWg: &sync.WaitGroup{},
		outlet: outlet,
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

	f, err := processors.New(h.config.Processors)
	if err != nil {
		return nil, err
	}

	h.processors = f

	// Add ttl if clean_inactive is set
	if h.config.CleanInactive > 0 {
		h.state.TTL = h.config.CleanInactive
	}

	// Add outlet signal so harvester can also stop itself
	h.outlet.SetSignal(h.done)

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

// updateState updates the prospector state and forwards the event to the spooler
// All state updates done by the prospector itself are synchronous to make sure not states are overwritten
func (h *Harvester) forwardEvent(data *util.Data) error {

	// Add additional prospector meta data to the event
	data.Meta.Pipeline = h.config.Pipeline
	data.Meta.Module = h.config.Module
	data.Meta.Fileset = h.config.Fileset

	if data.HasEvent() {
		data.Event[common.EventMetadataKey] = h.config.EventMetadata
		data.Event.Put("prospector.type", h.config.Type)

		// run the filters before sending to spooler
		data.Event = h.processors.Run(data.Event)
	}

	ok := h.outlet.OnEventSignal(data)

	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	return nil
}

// ID returns the unique harvester identifier
func (h *Harvester) ID() uuid.UUID {
	return h.id
}
