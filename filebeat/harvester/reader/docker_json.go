package reader

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
)

// DockerJSON processor renames a given field
type DockerJSON struct {
	reader Reader
	// stream filter, `all`, `stderr` or `stdout`
	stream string
}

type dockerLog struct {
	Timestamp string `json:"time"`
	Log       string `json:"log"`
	Stream    string `json:"stream"`
}

// NewDockerJSON creates a new reader renaming a field
func NewDockerJSON(r Reader, stream string) *DockerJSON {
	return &DockerJSON{
		stream: stream,
		reader: r,
	}
}

// Next returns the next line.
func (p *DockerJSON) Next() (Message, error) {
	for {
		message, err := p.reader.Next()
		if err != nil {
			return message, err
		}

		var line dockerLog
		dec := json.NewDecoder(bytes.NewReader(message.Content))
		if err = dec.Decode(&line); err != nil {
			return message, errors.Wrap(err, "decoding docker JSON")
		}

		if p.stream != "all" && p.stream != line.Stream {
			continue
		}

		// Parse timestamp
		ts, err := time.Parse(time.RFC3339, line.Timestamp)
		if err != nil {
			return message, errors.Wrap(err, "parsing docker timestamp")
		}

		message.AddFields(common.MapStr{
			"stream": line.Stream,
		})
		message.Content = []byte(line.Log)
		message.Ts = ts
		return message, nil
	}
}
