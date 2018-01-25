package reader

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"
	"strings"
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
func parseCRILog(b []byte, msg *crioLog, message Message) (Message, error) {
	log := string(b)
	ts, err := time.Parse(time.RFC3339, strings.Fields(log)[0])
	if err != nil {
		return message, errors.Wrap(err, "parsing CRI timestamp")
	}

	stream := strings.Fields(log)[1]
	// Anything after stream.
	logMessage := strings.Fields(log)[2:]
	msg.Timestamp = ts
	msg.Stream = stream
	msg.Log = []byte(strings.Join(logMessage, " "))
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
func parseDockerJSONLog(b []byte, msg *dockerLog, message Message) (Message, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
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
			message, err = parseDockerJSONLog(message.Content, &dockerLine, message)
		} else {
			message, err = parseCRILog(message.Content, &crioLine, message)
		}

		if p.stream != "all" && p.stream != dockerLine.Stream && p.stream != crioLine.Stream {
			continue
		}

		return message, err
	}
}
