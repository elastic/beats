package log

import (
	"os"
)

// Stdin reads all incoming traffic from stdin and sends it directly to the output

func (h *Harvester) openStdin() error {
	h.source = Pipe{File: os.Stdin}

	var err error
	h.encoding, err = h.encodingFactory(h.source)

	return err
}

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type Pipe struct {
	File *os.File
}

func (p Pipe) Read(b []byte) (int, error) { return p.File.Read(b) }
func (p Pipe) Close() error               { return p.File.Close() }
func (p Pipe) Name() string               { return p.File.Name() }
func (p Pipe) Stat() (os.FileInfo, error) { return p.File.Stat() }
func (p Pipe) Continuable() bool          { return false }
func (p Pipe) HasState() bool             { return false }
