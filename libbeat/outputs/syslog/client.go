package syslog

import (
	"expvar"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher"
)

var (
	shippedLines = expvar.NewInt("libbeatSyslogShippedLines")
	debugf       = logp.MakeDebug("syslog")
)

type syslogClient struct {
	*transport.Client
	observer       outputs.Observer
	syslogProgram  string
	syslogPriority uint64
	syslogSeverity uint64
	hostname       string
	timeout        time.Duration
}

func newClient(
	tc *transport.Client,
	observer outputs.Observer,
	prog string,
	pri uint64,
	sev uint64,
	timeout time.Duration,
) *syslogClient {
	// hostname only needs to be set once.
	// It's already set in the event by the publisher, but it doesn't make
	// sense to waste CPU extracting it from there for each event, when it's
	// always going to be the same. So let's set it once here an reuse.
	hostname, err := os.Hostname()
	if err != nil {
		debugf("Count not get hostname: %v. Setting to 'unknow'.", err)
	}
	return &syslogClient{
		Client:         tc,
		observer:       observer,
		syslogProgram:  prog,
		syslogPriority: pri,
		syslogSeverity: sev,
		hostname:       hostname,
		timeout:        timeout,
	}
}

func (c *syslogClient) Connect() error {
	debugf("connect")
	return c.Client.Connect()
}

func (c *syslogClient) Close() error {
	debugf("close connection")
	return c.Client.Close()
}

func (c *syslogClient) Publish(batch publisher.Batch) error {
	defer batch.ACK()

	if c == nil {
		panic("no client")
	}
	if batch == nil {
		panic("no batch")
	}

	events := batch.Events()
	c.observer.NewBatch(len(events))

	for _, d := range events {
		msg, err := c.CreateSyslogString(d)
		if err != nil {
			logp.Err("Dropping event: %v, Event Message: %s\n", err, msg)
			c.observer.Dropped(1)
			continue
		}
		c.Client.Write([]byte(msg))
		shippedLines.Add(1)
	}
	return nil
}

func (c *syslogClient) CreateSyslogString(event publisher.Event) (string, error) {
	// Pull some values from event, which we'll use to construct
	// our syslog string

	// @timestamp is guaranteed to be present
	// We need it in RFC3339 format for syslog
	ts := time.Time(event.Content.Timestamp).UTC().Format(time.RFC3339)

	var local_prog string = c.syslogProgram
	var local_pri uint64 = c.syslogPriority
	var local_sev uint64 = c.syslogSeverity

	// check for overrides from the event, if event["fields"] exists
	if len(event.Content.Fields) > 0 {
		// A value for program may have bean supplied in the config.
		if program_name, ok := event.Content.GetValue("program"); ok == nil {
			local_prog = program_name.(string)
		}
		// A value for priority may have bean supplied in the config.
		if priority_num, ok := event.Content.GetValue("priority"); ok == nil {
			local_pri = priority_num.(uint64)
		}
		// A value for severity may have bean supplied in the config.
		if severity_num, ok := event.Content.GetValue("severity"); ok == nil {
			local_sev = severity_num.(uint64)
		}
	}
	// Calculate the RPI number for the protocol according to RFC5424
	// If the priority and severity are both zero, use "0"
	// If the priority is zero but the severity is not, print a
	// leading zero followed by the severity.
	// otherwise, multiple the priority by 8, and add the severity
	var pri_num string
	if local_pri == 0 && local_sev == 0 {
		pri_num = "0"
	} else if local_pri == 0 && local_sev != 0 {
		pri_num = fmt.Sprintf("0%d", local_sev)
	} else {
		pri_num = fmt.Sprintf("%d", ((local_pri * 8) + local_sev))
	}

	var filesetName string = "-"
	if fna, ok := event.Content.Fields.GetValue("fileset.name"); ok == nil {
		filesetName = fna.(string)
	}

	// this is the log line which was read in.
	msg, err := event.Content.GetValue("message")
	line := fmt.Sprintf("<%s>%s %s %s[%s]: %s\n", pri_num, ts, c.hostname, local_prog, filesetName, msg)
	return line, err
}
