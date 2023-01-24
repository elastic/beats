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

package backup

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewFileBackuper creates a new backuper that creates backups for the given files.
func NewFileBackuper(log *logp.Logger, files []string) Backuper {
	return &fileBackuper{
		log:   log,
		files: files,
	}
}

type fileBackuper struct {
	log     *logp.Logger
	files   []string
	backups []string
}

// Backup creates temporary backups for given files and returns a callback that
// removes every created backup file
func (fb *fileBackuper) Backup() error {
	var (
		buf = make([]byte, 64*1024) // 64KB
	)

	for _, file := range fb.files {
		err := func() error {
			src, err := os.Open(file)
			if err != nil {
				return err
			}
			defer src.Close()

			// we must put the timestamp as a prefix, so after the restart the new backups don't override the previous ones
			backupFilename := fmt.Sprintf("%s-%d%s", file, time.Now().UnixNano(), backupSuffix)
			dst, err := os.OpenFile(backupFilename, os.O_CREATE|os.O_EXCL|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.CopyBuffer(dst, src, buf)
			if err != nil {
				return err
			}

			fb.backups = append(fb.backups, backupFilename)
			return nil
		}()
		if err != nil {
			return fmt.Errorf("failed to backup a file %s: %w", file, err)
		}
	}

	return nil
}

// Remove removes all backups created by this backuper
func (fb fileBackuper) Remove() error {
	fb.log.Infof("Removing backup files: %v...", fb.backups)

	var errs []error
	for _, backup := range fb.backups {
		err := os.Remove(backup)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("failed to remove some backups: %v", errs)
	}

	return nil
}
