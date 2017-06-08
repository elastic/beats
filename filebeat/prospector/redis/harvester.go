package redis

import (
	"fmt"
	"time"

	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"strings"

	"github.com/elastic/beats/filebeat/harvester"
	rd "github.com/garyburd/redigo/redis"
	"github.com/satori/go.uuid"
)

// Harvester contains all redis harvester data
type Harvester struct {
	id        uuid.UUID
	done      chan struct{}
	conn      rd.Conn
	forwarder *harvester.Forwarder
}

// log contains all data related to one slowlog entry
//
// 	The data is in the following format:
// 	1) (integer) 13
// 	2) (integer) 1309448128
// 	3) (integer) 30
// 	4) 1) "slowlog"
// 	   2) "get"
// 	   3) "100"
//
type log struct {
	id        int64
	timestamp int64
	duration  int
	cmd       string
	key       string
	args      []string
}

// NewHarvester creates a new harvester with the given connection
func NewHarvester(conn rd.Conn) *Harvester {
	return &Harvester{
		id:   uuid.NewV4(),
		done: make(chan struct{}),
		conn: conn,
	}
}

// Run starts a new redis harvester
func (h *Harvester) Run() error {
	defer h.conn.Close()

	select {
	case <-h.done:
		return nil
	default:
	}
	// Writes Slowlog get and slowlog reset both to the buffer so they are executed together
	h.conn.Send("SLOWLOG", "GET")
	h.conn.Send("SLOWLOG", "RESET")

	// Flush the buffer to execute both commands and receive the reply from SLOWLOG GET
	h.conn.Flush()

	// Receives first reply from redis which is the one from GET
	logs, err := rd.Values(h.conn.Receive())
	if err != nil {
		return fmt.Errorf("error receiving slowlog data: %s", err)
	}

	// Read reply from RESET
	_, err = h.conn.Receive()
	if err != nil {
		return fmt.Errorf("error receiving reset data: %s", err)
	}

	for _, item := range logs {
		// Stopping here means some of the slowlog events are lost!
		select {
		case <-h.done:
			return nil
		default:
		}
		entry, err := rd.Values(item, nil)
		if err != nil {
			logp.Err("Error loading slowlog values: %s", err)
			continue
		}

		var log log
		var args []string
		rd.Scan(entry, &log.id, &log.timestamp, &log.duration, &args)

		// This splits up the args into cmd, key, args.
		argsLen := len(args)
		if argsLen > 0 {
			log.cmd = args[0]
		}
		if argsLen > 1 {
			log.key = args[1]
		}

		// This could contain confidential data, processors should be used to drop it if needed
		if argsLen > 2 {
			log.args = args[2:]
		}

		data := util.NewData()
		subEvent := common.MapStr{
			"id":  log.id,
			"cmd": log.cmd,
			"key": log.key,
			"duration": common.MapStr{
				"us": log.duration,
			},
		}

		if log.args != nil {
			subEvent["args"] = log.args

		}

		data.Event = common.MapStr{
			"@timestamp": common.Time(time.Unix(log.timestamp, 0).UTC()),
			"message":    strings.Join(args, " "),
			"redis": common.MapStr{
				"slowlog": subEvent,
			},
			"read_timestamp": common.Time(time.Now()),
			"prospector": common.MapStr{
				"type": "redis",
			},
		}

		h.forwarder.Send(data)
	}
	return nil
}

// Stop stopps the harvester
func (h *Harvester) Stop() {
	close(h.done)
}

// ID returns the unique harvester ID
func (h *Harvester) ID() uuid.UUID {
	return h.id
}
