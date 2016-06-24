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
	"fmt"
	"regexp"
	"sync"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
)

type Harvester struct {
	Path               string /* the file path to harvest */
	Config             harvesterConfig
	offset             int64
	State              file.State
	stateMutex         sync.Mutex
	SpoolerChan        chan *input.FileEvent
	encoding           encoding.EncodingFactory
	file               source.FileSource /* the file being watched */
	ExcludeLinesRegexp []*regexp.Regexp
	IncludeLinesRegexp []*regexp.Regexp
	done               chan struct{}
}

func NewHarvester(
	cfg *common.Config,
	path string,
	state file.State,
	spooler chan *input.FileEvent,
	offset int64,
	done chan struct{},
) (*Harvester, error) {

	h := &Harvester{
		Path:        path,
		Config:      defaultConfig,
		State:       state,
		SpoolerChan: spooler,
		offset:      offset,
		done:        done,
	}

	if err := cfg.Unpack(&h.Config); err != nil {
		return nil, err
	}
	if err := h.Config.Validate(); err != nil {
		return nil, err
	}

	encoding, ok := encoding.FindEncoding(h.Config.Encoding)
	if !ok || encoding == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", h.Config.Encoding)
	}
	h.encoding = encoding

	h.ExcludeLinesRegexp = h.Config.ExcludeLines
	h.IncludeLinesRegexp = h.Config.IncludeLines
	return h, nil
}

// open does open the file given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() (encoding.Encoding, error) {

	switch h.Config.InputType {
	case config.StdinInputType:
		return h.openStdin()
	case config.LogInputType:
		return h.openFile()
	default:
		return nil, fmt.Errorf("Invalid input type")
	}
}
