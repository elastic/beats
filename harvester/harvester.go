package harvester

import (
	"os"

	"golang.org/x/text/encoding"

	"time"

	"github.com/elastic/filebeat/config"
	"github.com/elastic/filebeat/input"
)

type Harvester struct {
	Path             string /* the file path to harvest */
	ProspectorConfig config.ProspectorConfig
	Config           *config.HarvesterConfig
	Offset           int64
	Stat             *FileStat
	SpoolerChan      chan *input.FileEvent
	encoding         encoding.Encoding
	file             *os.File /* the file being watched */
	backoff          time.Duration
}

// Contains statistic about file when it was last seend by the prospector
type FileStat struct {
	Fileinfo      os.FileInfo /* the file info */
	Return        chan int64  /* the harvester will send an event with its offset when it closes */
	LastIteration uint32      /* int number of the last iterations in which we saw this file */
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

func NewFileStat(fi os.FileInfo, lastIteration uint32) *FileStat {
	fs := &FileStat{
		Fileinfo:      fi,
		Return:        make(chan int64, 1),
		LastIteration: lastIteration,
	}
	return fs
}

func (fs *FileStat) Finished() bool {
	return len(fs.Return) != 0
}

// Ignore forgets about the previous harvester results and let it continue on the old
// file - start a new channel to use with the new harvester.
func (fs *FileStat) Ignore() {
	fs.Return = make(chan int64, 1)
}

func (fs *FileStat) Continue(old *FileStat) {
	if old != nil {
		fs.Return = old.Return
	}
}

func (fs *FileStat) Skip(returnOffset int64) {
	fs.Return <- returnOffset
}
