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

package file

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type SuffixType uint32

const (
	// MaxBackupsLimit is the upper bound on the number of backup files. Any values
	// greater will result in an error.
	MaxBackupsLimit = 1024

	SuffixCount SuffixType = iota + 1
	SuffixDate
)

var suffixes = map[string]SuffixType{
	"count": SuffixCount,
	"date":  SuffixDate,
}

// rotater is the interface responsible for rotating and finding files.
type rotater interface {
	// ActiveFile returns the path to the file that is actively written.
	ActiveFile() string
	// RotatedFiles returns the list of rotated files. The oldest comes first.
	RotatedFiles() []string
	// Rotate rotates the file.
	Rotate(reason rotateReason, rotateTime time.Time) error
}

// Rotator is a io.WriteCloser that automatically rotates the file it is
// writing to when it reaches a maximum size and optionally on a time interval
// basis. It also purges the oldest rotated files when the maximum number of
// backups is reached.
type Rotator struct {
	rot      rotater
	triggers []trigger

	filename        string
	maxSizeBytes    uint
	maxBackups      uint
	interval        time.Duration
	permissions     os.FileMode
	log             Logger // Optional Logger (may be nil).
	suffix          SuffixType
	rotateOnStartup bool
	redirectStderr  bool

	file  *os.File
	mutex sync.Mutex
}

// Logger allows the rotator to write debug information.
type Logger interface {
	Debugw(msg string, keysAndValues ...interface{}) // Debug
}

// RotatorOption is a configuration option for Rotator.
type RotatorOption func(r *Rotator)

// Interval sets the time interval for log rotation in addition to log
// rotation by size. The default is 0 for disabled.
func Suffix(s SuffixType) RotatorOption {
	return func(r *Rotator) {
		r.suffix = s
	}
}

// MaxSizeBytes configures the maximum number of bytes that a file should
// contain before being rotated. The default is 10 MiB.
func MaxSizeBytes(n uint) RotatorOption {
	return func(r *Rotator) {
		r.maxSizeBytes = n
	}
}

// MaxBackups configures the maximum number of backup files to save (not
// counting the active file). The upper limit is 1024 on this value is.
// The default is 7.
func MaxBackups(n uint) RotatorOption {
	return func(r *Rotator) {
		r.maxBackups = n
	}
}

// Permissions configures the file permissions to use for the file that
// the Rotator creates. The default is 0600.
func Permissions(m os.FileMode) RotatorOption {
	return func(r *Rotator) {
		r.permissions = m
	}
}

// WithLogger injects a logger implementation for logging debug information.
// If no logger is injected then the no logging will occur.
func WithLogger(l Logger) RotatorOption {
	return func(r *Rotator) {
		r.log = l
	}
}

// Interval sets the time interval for log rotation in addition to log
// rotation by size. The default is 0 for disabled.
func Interval(d time.Duration) RotatorOption {
	return func(r *Rotator) {
		r.interval = d
	}
}

// RotateOnStartup immediately rotates files on startup rather than appending to
// the existing file. The default is true.
func RotateOnStartup(b bool) RotatorOption {
	return func(r *Rotator) {
		r.rotateOnStartup = b
	}
}

// RedirectStderr causes all writes to standard error to be redirected
// to this rotator.
func RedirectStderr(redirect bool) RotatorOption {
	return func(r *Rotator) {
		r.redirectStderr = redirect
	}
}

// NewFileRotator returns a new Rotator.
func NewFileRotator(filename string, options ...RotatorOption) (*Rotator, error) {
	r := &Rotator{
		maxSizeBytes:    10 * 1024 * 1024, // 10 MiB
		maxBackups:      7,
		permissions:     0600,
		interval:        0,
		rotateOnStartup: true,
		suffix:          SuffixCount,
	}

	for _, opt := range options {
		opt(r)
	}

	if r.maxSizeBytes == 0 {
		return nil, errors.New("file rotator max file size must be greater than 0")
	}
	if r.maxBackups > MaxBackupsLimit {
		return nil, errors.Errorf("file rotator max backups %d is greater than the limit of %v", r.maxBackups, MaxBackupsLimit)
	}
	if r.permissions > os.ModePerm {
		return nil, errors.Errorf("file rotator permissions mask of %o is invalid", r.permissions)
	}

	if r.interval != 0 && r.interval < time.Second {
		return nil, errors.New("the minimum time interval for log rotation is 1 second")
	}

	r.rot = newRotater(r.log, r.suffix, filename, r.maxBackups, r.interval)

	shouldRotateOnStart := r.rotateOnStartup
	if _, err := os.Stat(r.rot.ActiveFile()); os.IsNotExist(err) {
		shouldRotateOnStart = false
	}

	r.triggers = newTriggers(shouldRotateOnStart, r.interval, r.maxSizeBytes)

	if r.log != nil {
		r.log.Debugw("Initialized file rotator",
			"filename", r.filename,
			"max_size_bytes", r.maxSizeBytes,
			"max_backups", r.maxBackups,
			"permissions", r.permissions,
			"suffix", r.suffix,
		)
	}

	return r, nil
}

// Write writes the given bytes to the file. This implements io.Writer. If
// the write would trigger a rotation the rotation is done before writing to
// avoid going over the max size. Write is safe for concurrent use.
func (r *Rotator) Write(data []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	dataLen := uint(len(data))
	if dataLen > r.maxSizeBytes {
		return 0, errors.Errorf("data size (%d bytes) is greater than "+
			"the max file size (%d bytes)", dataLen, r.maxSizeBytes)
	}

	if r.file == nil {
		if err := r.openNew(); err != nil {
			return 0, errors.Wrap(err, "failed to open new log file for writing")
		}
	} else {
		if reason, t := r.isRotationTriggered(dataLen); reason != rotateReasonNoRotate {
			if err := r.rotateWithTime(reason, t); err != nil {
				return 0, errors.Wrapf(err, "error file rotating files reason: %s", reason)
			}

			if err := r.openFile(); err != nil {
				return 0, errors.Wrap(err, "failed to open existing log file for writing")
			}
		}
	}

	n, err := r.file.Write(data)
	return n, errors.Wrap(err, "failed to write to file")
}

// openNew opens r's log file for the first time, creating it if it doesn't
// exist.
func (r *Rotator) openNew() error {
	err := os.MkdirAll(r.dir(), r.dirMode())
	if err != nil {
		return errors.Wrap(err, "failed to make directories for new file")
	}

	_, err = os.Stat(r.rot.ActiveFile())
	if err == nil {
		// check if the file has to be rotated before writing to it
		reason, t := r.isRotationTriggered(0)
		if reason == rotateReasonNoRotate {
			return r.appendToFile()
		}
		if err = r.rot.Rotate(reason, t); err != nil {
			return errors.Wrap(err, "failed to rotate backups")
		}
		if err = r.purge(); err != nil {
			return errors.Wrap(err, "failed to purge unnecessary rotated files")
		}
	}

	return r.openFile()
}

// appendToFile opens an existing log file for appending. Unlike openFile it
// does not call MkdirAll because it is an error for the file to not already
// exist.
func (r *Rotator) appendToFile() error {
	var err error
	r.file, err = os.OpenFile(r.rot.ActiveFile(), os.O_WRONLY|os.O_APPEND, r.permissions)
	if err != nil {
		return errors.Wrap(err, "failed to append to existing file")
	}
	if r.redirectStderr {
		RedirectStandardError(r.file)
	}
	return nil
}

func (r *Rotator) openFile() error {
	err := os.MkdirAll(r.dir(), r.dirMode())
	if err != nil {
		return errors.Wrap(err, "failed to make directories for new file")
	}

	r.file, err = os.OpenFile(r.rot.ActiveFile(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, r.permissions)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open new file '%s'", r.rot.ActiveFile()))
	}
	if r.redirectStderr {
		RedirectStandardError(r.file)
	}
	return nil
}

func (r *Rotator) rotate(reason rotateReason) error {
	return r.rotateWithTime(reason, time.Now())
}

// rotateWithTime closes the actively written file, and rotates it along with exising
// rotated files if needed. When it is done, unnecessary files are removed.
func (r *Rotator) rotateWithTime(reason rotateReason, rotationTime time.Time) error {
	if err := r.closeFile(); err != nil {
		return errors.Wrap(err, "error file closing current file")
	}

	if err := r.rot.Rotate(reason, rotationTime); err != nil {
		return errors.Wrap(err, "failed to rotate backups")
	}

	return r.purge()
}

func (r *Rotator) purge() error {
	rotatedFiles := r.rot.RotatedFiles()
	count := uint(len(rotatedFiles))
	if count <= r.maxBackups {
		return nil
	}

	purgeUntil := count - r.maxBackups
	filesToPurge := rotatedFiles[:purgeUntil]
	for _, name := range filesToPurge {
		_, err := os.Stat(name)
		switch {
		case err == nil:
			if err = os.Remove(name); err != nil {
				return errors.Wrapf(err, "failed to delete %v during rotation", name)
			}
		case os.IsNotExist(err):
			return nil
		default:
			return errors.Wrapf(err, "failed on %v during rotation", name)
		}
	}

	return nil
}

func (r *Rotator) isRotationTriggered(dataLen uint) (rotateReason, time.Time) {
	for _, t := range r.triggers {
		reason := t.TriggerRotation(dataLen)
		if reason != rotateReasonNoRotate {
			return reason, time.Now()
		}
	}
	return rotateReasonNoRotate, time.Time{}
}

// Sync commits the current contents of the file to stable storage. Typically,
// this means flushing the file system's in-memory copy of recently written data
// to disk.
func (r *Rotator) Sync() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.file == nil {
		return nil
	}
	return r.file.Sync()
}

// Rotate triggers a file rotation.
func (r *Rotator) Rotate() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.rotate(rotateReasonManualTrigger)
}

// Close closes the currently open file.
func (r *Rotator) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.closeFile()
}

func (r *Rotator) dir() string {
	return filepath.Dir(r.rot.ActiveFile())
}

func (r *Rotator) dirMode() os.FileMode {
	mode := 0700
	if r.permissions&0070 > 0 {
		mode |= 0050
	}
	if r.permissions&0007 > 0 {
		mode |= 0005
	}
	return os.FileMode(mode)
}

func (r *Rotator) closeFile() error {
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return errors.Wrap(err, "failed to close active file")
}

type countRotator struct {
	log             Logger
	filename        string
	intervalRotator *intervalRotator
	maxBackups      uint
}

type dateRotator struct {
	log             Logger
	format          string
	filenamePrefix  string
	currentFilename string
	intervalRotator *intervalRotator
}

func newRotater(log Logger, s SuffixType, filename string, maxBackups uint, interval time.Duration) rotater {
	switch s {
	case SuffixCount:
		if interval > 0 {
			return newIntervalRotator(log, interval, filename)
		}
		return &countRotator{
			log:        log,
			filename:   filename,
			maxBackups: maxBackups,
		}
	case SuffixDate:
		return newDateRotater(log, filename)
	default:
		return &countRotator{
			log:        log,
			filename:   filename,
			maxBackups: maxBackups,
		}
	}
}

func newDateRotater(log Logger, filename string) rotater {
	d := &dateRotator{
		log:            log,
		filenamePrefix: filename + "-",
		format:         "20060102150405",
	}

	d.currentFilename = d.filenamePrefix + time.Now().Format(d.format)
	files, err := filepath.Glob(d.filenamePrefix + "*")
	if err != nil {
		return d
	}

	// continue from last file
	if len(files) != 0 {
		if len(files) == 1 {
			d.currentFilename = files[0]
		} else {
			d.SortModTimeLogs(files)
			d.currentFilename = files[len(files)-1]
		}
	}

	return d
}

func (d *dateRotator) ActiveFile() string {
	return d.currentFilename
}

func (d *dateRotator) Rotate(reason rotateReason, rotateTime time.Time) error {
	if d.log != nil {
		d.log.Debugw("Rotating file", "filename", d.currentFilename, "reason", reason)
	}

	d.currentFilename = d.filenamePrefix + rotateTime.Format(d.format)
	return nil
}

func (d *dateRotator) RotatedFiles() []string {
	files, err := filepath.Glob(d.filenamePrefix + "*")
	if err != nil {
		if d.log != nil {
			d.log.Debugw("failed to list existing logs: %+v", err)
		}
	}

	d.SortModTimeLogs(files)
	return files
}

func (d *dateRotator) SortModTimeLogs(strings []string) {
	sort.Slice(
		strings,
		func(i, j int) bool {
			return d.OrderLog(strings[i]).After(d.OrderLog(strings[j]))
		},
	)
}

func (d *dateRotator) OrderLog(filename string) time.Time {
	ts, err := time.Parse(d.filenamePrefix+d.format, filepath.Base(filename))
	if err != nil {
		return time.Time{}
	}
	return ts
}

func (c *countRotator) ActiveFile() string {
	return c.filename
}

func (c *countRotator) RotatedFiles() []string {
	files := make([]string, 0)
	for i := c.maxBackups + 1; i >= 1; i-- {
		name := c.backupName(i)
		if _, err := os.Stat(name); os.IsNotExist(err) {
			continue
		} else if err != nil {
			c.log.Debugw("failed to stat rotated file")
			return files
		}
		files = append(files, name)
	}

	return files
}

func (c *countRotator) backupName(n uint) string {
	if n == 0 {
		return c.ActiveFile()
	}
	return c.ActiveFile() + "." + strconv.Itoa(int(n))
}

func (c *countRotator) Rotate(reason rotateReason, _ time.Time) error {
	for i := c.maxBackups + 1; i > 0; i-- {
		old := c.backupName(i - 1)
		older := c.backupName(i)

		if _, err := os.Stat(old); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errors.Wrap(err, "failed to rotate backups")
		}

		if err := os.Remove(older); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to rotate backups")
		}
		if err := os.Rename(old, older); err != nil {
			return errors.Wrap(err, "failed to rotate backups")
		} else if i == 1 {
			// Log when rotation of the main file occurs.
			if c.log != nil {
				c.log.Debugw("Rotating file", "filename", old, "reason", reason)
			}
		}
	}
	return nil
}

func (s *SuffixType) Unpack(v string) error {
	i, err := strconv.Atoi(v)
	if err == nil {
		t := SuffixType(i)
		v = t.String()
	}

	val, ok := suffixes[v]
	if !ok {
		return fmt.Errorf("invalid suffix type: %+v", v)
	}

	*s = val
	return nil
}

func (s *SuffixType) String() string {
	for k, v := range suffixes {
		if v == *s {
			return k
		}
	}
	return ""
}
