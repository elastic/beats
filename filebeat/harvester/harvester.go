package harvester

import (
	"os"

	"golang.org/x/text/encoding"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/input"
	"time"
)

type Harvester struct {
	Path             string /* the file path to harvest */
	ProspectorConfig config.ProspectorConfig
	Config           *config.HarvesterConfig
	Offset           int64
	FinishChan       chan int64
	SpoolerChan      chan *input.FileEvent
	encoding         encoding.Encoding
	file             *os.File /* the file being watched */
	backoff          time.Duration
}

// Interface for the different harvester types
type Typer interface {
	open()
	read()
}

func (h *Harvester) Start() {
	// Starts harvester and picks the right type. In case type is not set, set it to defeault (log)

	go h.Harvest()
}
