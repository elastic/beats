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

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/input"
)

type Harvester struct {
	Path               string /* the file path to harvest */
	Config             *config.HarvesterConfig
	Offset             int64
	Stat               *FileStat
	SpoolerChan        chan *input.FileEvent
	encoding           encoding.EncodingFactory
	file               FileSource /* the file being watched */
	ExcludeLinesRegexp []*regexp.Regexp
	IncludeLinesRegexp []*regexp.Regexp
}

func NewHarvester(
	cfg *config.HarvesterConfig,
	path string,
	stat *FileStat,
	spooler chan *input.FileEvent,
) (*Harvester, error) {

	var err error
	encoding, ok := encoding.FindEncoding(cfg.Encoding)
	if !ok || encoding == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", cfg.Encoding)
	}

	h := &Harvester{
		Path:        path,
		Config:      cfg,
		Stat:        stat,
		SpoolerChan: spooler,
		encoding:    encoding,
	}
	h.ExcludeLinesRegexp, err = InitRegexps(cfg.ExcludeLines)
	if err != nil {
		return h, err
	}
	h.IncludeLinesRegexp, err = InitRegexps(cfg.IncludeLines)
	if err != nil {
		return h, err
	}
	return h, nil
}

func (h *Harvester) Start() {
	// Starts harvester and picks the right type. In case type is not set, set it to defeault (log)

	go h.Harvest()
}
