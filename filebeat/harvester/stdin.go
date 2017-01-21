package harvester

import (
	"os"

	"github.com/elastic/beats/filebeat/harvester/source"
)

// Stdin reads all incoming traffic from stdin and sends it directly to the output

func (h *Harvester) openStdin() error {
	h.file = source.Pipe{os.Stdin}

	var err error
	h.encoding, err = h.encodingFactory(h.file)

	return err
}
