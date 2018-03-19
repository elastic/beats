package reader

import (
	"bytes"
	"encoding/json"
	"strings"
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

type crioLog struct {
	Timestamp time.Time
	Stream    string
	Log       []byte
}

// NewDockerJSON creates a new reader renaming a field
func NewDockerJSON(r Reader, stream string) *DockerJSON {
	return &DockerJSON{
		stream: stream,
		reader: r,
	}
}

// parseCRILog parses logs in CRI log format.
// CRI log format example :
// 2017-09-12T22:32:21.212861448Z stdout 2017-09-12 22:32:21.212 [INFO][88] table.go 710: Invalidating dataplane cache
func parseCRILog(message Message, msg *crioLog) (Message, error) {
	log := strings.SplitN(string(message.Content), " ", 3)
	if len(log) < 3 {
		return message, errors.New("invalid CRI log")
	}
	ts, err := time.Parse(time.RFC3339, log[0])
	if err != nil {
		return message, errors.Wrap(err, "parsing CRI timestamp")
	}

	msg.Timestamp = ts
	msg.Stream = log[1]
	msg.Log = []byte(log[2])
	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	message.Content = msg.Log
	message.Ts = ts

	return message, nil
}

// parseDockerJSONLog parses logs in Docker JSON log format.
// Docker JSON log format example:
// {"log":"1:M 09 Nov 13:27:36.276 # User requested shutdown...\n","stream":"stdout"}
func parseDockerJSONLog(message Message, msg *dockerLog) (Message, error) {
	dec := json.NewDecoder(bytes.NewReader(message.Content))
	if err := dec.Decode(&msg); err != nil {
		return message, errors.Wrap(err, "decoding docker JSON")
	}

	// Parse timestamp
	ts, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err != nil {
		return message, errors.Wrap(err, "parsing docker timestamp")
	}

	message.AddFields(common.MapStr{
		"stream": msg.Stream,
	})
	message.Content = []byte(msg.Log)
	message.Ts = ts

	return message, nil
}

// Next returns the next line.
func (p *DockerJSON) Next() (Message, error) {
	for {
		message, err := p.reader.Next()
		if err != nil {
			return message, err
		}

		var dockerLine dockerLog
		var crioLine crioLog

		if strings.HasPrefix(string(message.Content), "{") {
			message, err = parseDockerJSONLog(message, &dockerLine)
		} else {
			message, err = parseCRILog(message, &crioLine)
		}

		if p.stream != "all" && p.stream != dockerLine.Stream && p.stream != crioLine.Stream {
			continue
		}

		return message, err
	}
}
