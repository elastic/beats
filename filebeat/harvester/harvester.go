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
package harvester

import (
	"errors"
	"fmt"
	"sync"

	"github.com/satori/go.uuid"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
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
	OnEventSignal(event *input.Data) bool
	OnEvent(event *input.Data) bool
}

type Harvester struct {
	config          harvesterConfig
	state           file.State
	states          *file.States
	prospectorChan  chan *input.Event
	file            source.FileSource /* the file being watched */
	fileReader      *LogFile
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	done            chan struct{}
	stopOnce        sync.Once
	stopWg          *sync.WaitGroup
	outlet          Outlet
	ID              uuid.UUID
	processors      *processors.Processors
}

func NewHarvester(
	cfg *common.Config,
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
		ID:     uuid.NewV4(),
	}

	if err := cfg.Unpack(&h.config); err != nil {
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

	switch h.config.InputType {
	case config.StdinInputType:
		return h.openStdin()
	case config.LogInputType:
		return h.openFile()
	default:
		return fmt.Errorf("Invalid input type")
	}
}

// updateState updates the prospector state and forwards the event to the spooler
// All state updates done by the prospector itself are synchronous to make sure not states are overwritten
func (h *Harvester) forwardEvent(event *input.Event) error {

	// Add additional prospector meta data to the event
	event.EventMetadata = h.config.EventMetadata
	event.InputType = h.config.InputType
	event.DocumentType = h.config.DocumentType
	event.JSONConfig = h.config.JSON
	event.Pipeline = h.config.Pipeline
	event.Module = h.config.Module
	event.Fileset = h.config.Fileset

	eventHolder := event.GetData()
	//run the filters before sending to spooler
	if event.Bytes > 0 {
		eventHolder.Event = h.processors.Run(eventHolder.Event)
	}

	if eventHolder.Event == nil {
		eventHolder.Metadata.Bytes = 0
	}

	ok := h.outlet.OnEventSignal(&eventHolder)

	if !ok {
		logp.Info("Prospector outlet closed")
		return errors.New("prospector outlet closed")
	}

	return nil
}
