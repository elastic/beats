/*
  The harvester package harvest different inputs for new information. Currently
  two harvester types exist:

   * log
   * stdin

  The log harvester reads a file line by line. In case the end of a file is found
  with an incomplete line, the line pointer stays at the beginning of the incomplete
  line. As soon as the line is completed, it is read and returned.

  The stdin harvesters reads data from stdin.
*/
package harvester

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input"
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
	config          harvesterConfig
	state           file.State
	prospectorChan  chan *input.Event
	file            source.FileSource /* the file being watched */
	fileReader      *LogFile
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	done            chan struct{}
}

func NewHarvester(
	cfg *common.Config,
	state file.State,
	prospectorChan chan *input.Event,
	done chan struct{},
) (*Harvester, error) {

	h := &Harvester{
		config:         defaultConfig,
		state:          state,
		prospectorChan: prospectorChan,
		done:           done,
	}

	if err := cfg.Unpack(&h.config); err != nil {
		return nil, err
	}

	encodingFactory, ok := encoding.FindEncoding(h.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.config.Encoding)
	}
	h.encodingFactory = encodingFactory

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
