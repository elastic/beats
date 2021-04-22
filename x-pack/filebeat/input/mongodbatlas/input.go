package mongodbatlas

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/mongodb-forks/digest"
	atlas "go.mongodb.org/atlas/mongodbatlas"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

const (
	pluginName   = "mongodbatlas"
	fieldsPrefix = pluginName
)

type mongodbatlasinput struct {
	config Config
}

func (inp *mongodbatlasinput) Name() string {
	return pluginName
}

type stream struct {
	groupId string
	logs    []string
}

func (s *stream) Name() string {
	return s.groupId + "::" + strings.Join(s.logs, "_")
}

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "MongoDB Atlas logs",
		Doc:        "Collect logs from mongodb atlas service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

func configure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		fmt.Printf("\n%v\n%v\n%v\n", cfg, config, err)
		return nil, nil, errors.Wrap(err, "reading config")
	}

	var sources []cursor.Source
	for _, groupId := range config.GroupId {
		sources = append(sources, &stream{
			groupId: groupId,
			logs:    config.LogName,
		})
	}
	return sources, &mongodbatlasinput{config: config}, nil
}

func (inp *mongodbatlasinput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (inp *mongodbatlasinput) Run(
	ctx v2.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	for ctx.Cancelation.Err() == nil {
		err := inp.runOnce(ctx, src, cursor, publisher)
		if err == nil {
			break
		}
		if ctx.Cancelation.Err() != err && err != context.Canceled {
			msg := common.MapStr{}
			msg.Put("error.message", err.Error())
			msg.Put("event.kind", "pipeline_error")
			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    msg,
			}
			publisher.Publish(event, nil)
			ctx.Logger.Errorf("Input failed: %v", err)
			ctx.Logger.Infof("Restarting in %v", inp.config.API.ErrorRetryInterval)
			timed.Wait(ctx.Cancelation, inp.config.API.ErrorRetryInterval)
		}
	}
	return nil
}

func (inp *mongodbatlasinput) runOnce(
	ctx v2.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	fmt.Printf("Foobar 2\n")
	stream := src.(*stream)
	groupId, logs := stream.groupId, stream.logs
	log := ctx.Logger.With("groupId", groupId, "logs", logs)

	config := &inp.config

	t := digest.NewTransport(config.PublicKey, config.PrivateKey)
	tc, err := t.Client()
	if err != nil {
		log.Fatalf(err.Error())
	}

	log.Debugf("Getting processes from groupId %v", groupId)
	cancellationCtx := ctxtool.FromCanceller(ctx.Cancelation)
	client := atlas.NewClient(tc)
	processes, _, err := client.Processes.List(cancellationCtx, groupId, nil)

	if err != nil {
		return errors.Wrap(err, "failed to create API poller")
	}

	var cp checkpoint
	maxRetention := config.API.MaxRetention
	retentionLimit := time.Now().UTC().Add(-maxRetention)

	if cursor.IsNew() {
		log.Infof("No saved state found. Will fetch events for the last %v.", maxRetention.String())
		cp.Timestamp = retentionLimit
	} else {
		err := cursor.Unpack(&cp)
		if err != nil {
			log.Errorw("Error loading saved state. Will fetch all retained events. "+
				"Depending on max_retention, this can cause event loss or duplication.",
				"error", err,
				"max_retention", maxRetention.String())
			cp.Timestamp = retentionLimit
		}
	}

	for _, process := range processes {
		opts := &atlas.DateRangetOptions{
			StartDate: strconv.FormatInt(cp.Timestamp.Unix(), 10),
		}
		for _, logName := range logs {
			pipe := NewGzipPipe()
			log.Debugf("Getting %v from %v with %v\n", logName, process.Hostname, opts)
			resp, err := client.Logs.Get(cancellationCtx, groupId, process.Hostname, logName, pipe, opts)
			//w.Close()
			log.Debugf("Resp from mongoApi %v", resp)
			if err != nil {
				return fmt.Errorf("failed to get logs err: %w", err)
			}

			reader := bufio.NewReader(pipe)

			var prevPrefix []byte
			for {
				line, prefix, err := reader.ReadLine()
				if err == io.EOF {
					break;
				}
				if err != nil {
					return fmt.Errorf("ReadLine failed, %w", err)
				}
				if prefix {
					prevPrefix = append(prevPrefix, line...)
					continue
				} else {
					line = append(prevPrefix, line...)
					prevPrefix = []byte{}
				}
				err = publisher.Publish(inp.createEvent(line))
				if err != nil {
					return err
				}
			}

			err = pipe.Close()
			if err != nil{
				return err
			}
		}
	}

	return nil
}

type GzipPipe struct{
	writeBuf *bytes.Buffer
	reader *gzip.Reader

	lock *sync.Mutex
}

func NewGzipPipe() *GzipPipe {
	writeBuf:= new(bytes.Buffer)
	lock := new(sync.Mutex)

	return &GzipPipe{
		writeBuf: writeBuf,
		lock: lock,
	}
}
func (p *GzipPipe) Write(b []byte) (int, error) {
	return p.writeBuf.Write(b)
}

func (p *GzipPipe) Read(b []byte) (n int, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.reader == nil {
		p.reader, err = gzip.NewReader(p.writeBuf)
		if err != nil {
			return 0, err
		}
		p.reader.Multistream(false)
	}
	return p.reader.Read(b)
}

func (p *GzipPipe) Close() error {
	var err error
	if p.reader != nil {
		err = p.reader.Close()
	}
	return err
}

func (inp *mongodbatlasinput) createEvent(line []byte) (beat.Event, interface{}) {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: common.MapStr{
			"message": string(line),
		},
	}

	return event, nil
}

// A checkpoint represents a point in time within an event stream
// that can be persisted and used to resume processing from that point.
type checkpoint struct {
	// createdTime for the last seen blob.
	Timestamp time.Time `struct:"timestamp"`

	// index of object count (1...n) within a blob.
	Line int `struct:"line"`

	// startTime used in the last list content query.
	// This is necessary to ensure that the same blobs are observed.
	StartTime time.Time `struct:"start_time"`
}
