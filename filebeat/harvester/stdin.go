package harvester

import (
	"os"

	"github.com/elastic/beats/filebeat/harvester/source"
	"golang.org/x/text/encoding"
)

// Stdin reads all incoming traffic from stdin and sends it directly to the output

func (h *Harvester) openStdin() (encoding.Encoding, error) {
	h.file = source.Pipe{os.Stdin}
	return h.encoding(h.file)
}
