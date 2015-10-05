package harvester

import (
	"os"

	"github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
)

type Harvester struct {
	Path             string /* the file path to harvest */
	ProspectorConfig config.ProspectorConfig
	Offset           int64
	FinishChan       chan int64
	SpoolerChan      chan *input.FileEvent
	BufferSize       int
	TailOnRotate     bool
	file             *os.File /* the file being watched */
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
