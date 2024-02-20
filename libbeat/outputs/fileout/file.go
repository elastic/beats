// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fileout

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/publisher"
	c "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/file"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	outputs.RegisterType("file", makeFileout)
}

const timeNowVar = "TIME_NOW"

type fileOutput struct {
	log      *logp.Logger
	filePath string
	beat     beat.Info
	observer outputs.Observer
	rotator  *file.Rotator
	codec    codec.Codec
}

// makeFileout instantiates a new file output instance.
func makeFileout(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *c.C,
) (outputs.Group, error) {
	foConfig, err := readConfig(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	fo := &fileOutput{
		log:      logp.NewLogger("file"),
		beat:     beat,
		observer: observer,
	}
	if err = fo.init(beat, *foConfig); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(foConfig.Queue, -1, 0, fo)
}

func (out *fileOutput) init(beat beat.Info, c fileOutConfig) error {
	var path string
	if c.Filename != "" {
		path = filepath.Join(c.Path, c.Filename)
	} else {
		path = filepath.Join(c.Path, out.beat.Beat)
	}

	out.filePath = path

	var err error
	out.rotator, err = file.NewFileRotator(
		path,
		file.MaxSizeBytes(c.RotateEveryKb*1024),
		file.MaxBackups(c.NumberOfFiles),
		file.Permissions(os.FileMode(c.Permissions)),
		file.RotateOnStartup(c.RotateOnStartup),
		file.WithLogger(logp.NewLogger("rotator").With(logp.Namespace("rotator"))),
	)
	if err != nil {
		return err
	}

	out.codec, err = codec.CreateEncoder(beat, c.Codec)
	if err != nil {
		return err
	}

	out.log.Infof("Initialized file output. "+
		"path=%v max_size_bytes=%v max_backups=%v permissions=%v",
		path, c.RotateEveryKb*1024, c.NumberOfFiles, os.FileMode(c.Permissions))

	return nil
}

// Implement Outputer
func (out *fileOutput) Close() error {
	return out.rotator.Close()
}

func (out *fileOutput) Publish(_ context.Context, batch publisher.Batch) error {
	defer batch.ACK()

	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))

	dropped := 0

	for i := range events {
		event := &events[i]

		serializedEvent, err := out.codec.Encode(out.beat.Beat, &event.Content)
		if err != nil {
			if event.Guaranteed() {
				out.log.Errorf("Failed to serialize the event: %+v", err)
			} else {
				out.log.Warnf("Failed to serialize the event: %+v", err)
			}
			out.log.Debugf("Failed event: %v", event)

			dropped++
			continue
		}

		begin := time.Now()
		if _, err = out.rotator.Write(append(serializedEvent, '\n')); err != nil {
			st.WriteError(err)

			if event.Guaranteed() {
				out.log.Errorf("Writing event to file failed with: %+v", err)
			} else {
				out.log.Warnf("Writing event to file failed with: %+v", err)
			}

			dropped++
			continue
		}

		st.WriteBytes(len(serializedEvent) + 1)
		took := time.Since(begin)
		st.ReportLatency(took)
	}

	st.Dropped(dropped)

	st.Acked(len(events) - dropped)

	return nil
}

func (out *fileOutput) String() string {
	return "file(" + out.filePath + ")"
}
