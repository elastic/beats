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
	"io"
	"os"
	"regexp"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/input"
)

type Harvester struct {
	Path               string /* the file path to harvest */
	ProspectorConfig   config.ProspectorConfig
	Config             *config.HarvesterConfig
	Offset             int64
	Stat               *FileStat
	SpoolerChan        chan *input.FileEvent
	encoding           encoding.EncodingFactory
	file               FileSource /* the file being watched */
	ExcludeLinesRegexp []*regexp.Regexp
	IncludeLinesRegexp []*regexp.Regexp
}

// Contains statistic about file when it was last seend by the prospector
type FileStat struct {
	Fileinfo      os.FileInfo /* the file info */
	Return        chan int64  /* the harvester will send an event with its offset when it closes */
	LastIteration uint32      /* int number of the last iterations in which we saw this file */
}

type LogSource interface {
	io.ReadCloser
	Name() string
}

type FileSource interface {
	LogSource
	Stat() (os.FileInfo, error)
	Continuable() bool // can we continue processing after EOF?
}

// Interface for the different harvester types
type Typer interface {
	open()
	read()
}

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type pipeSource struct{ file *os.File }

func (p pipeSource) Read(b []byte) (int, error) { return p.file.Read(b) }
func (p pipeSource) Close() error               { return p.file.Close() }
func (p pipeSource) Name() string               { return p.file.Name() }
func (p pipeSource) Stat() (os.FileInfo, error) { return p.file.Stat() }
func (p pipeSource) Continuable() bool          { return false }

type fileSource struct{ *os.File }

func (fileSource) Continuable() bool { return true }

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
