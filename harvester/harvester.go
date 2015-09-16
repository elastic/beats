package harvester

import (
	"github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
	"os"
)

type Harvester struct {
	Path        string /* the file path to harvest */
	FileConfig  config.FileConfig
	Offset      int64
	FinishChan  chan int64
	SpoolerChan chan *input.FileEvent

	file *os.File /* the file being watched */
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
