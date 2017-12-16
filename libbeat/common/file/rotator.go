package file

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

// MaxBackupsLimit is the upper bound on the number of backup files. Any values
// greater will result in an error.
const MaxBackupsLimit = 1024

// Rotator is a io.WriteCloser that automatically rotates the file it is
// writing to when it reaches a maximum size. It also purges the oldest rotated
// files when the maximum number of backups is reached.
type Rotator struct {
	filename     string
	maxSizeBytes uint
	maxBackups   uint
	permissions  os.FileMode

	file  *os.File
	size  uint
	mutex sync.Mutex
}

// RotatorOption is a configuration option for Rotator.
type RotatorOption func(r *Rotator)

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

// NewFileRotator returns a new Rotator.
func NewFileRotator(filename string, options ...RotatorOption) (*Rotator, error) {
	r := &Rotator{
		filename:     filename,
		maxSizeBytes: 10 * 1024 * 1024, // 10 MiB
		maxBackups:   7,
		permissions:  0600,
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
			return 0, err
		}
	} else if r.size+dataLen > r.maxSizeBytes {
		if err := r.rotate(); err != nil {
			return 0, err
		}
		if err := r.openFile(); err != nil {
			return 0, err
		}
	}

	n, err := r.file.Write(data)
	r.size += uint(n)
	return n, errors.Wrap(err, "failed to write to file")
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
	return r.rotate()
}

// Close closes the currently open file.
func (r *Rotator) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.closeFile()
}

func (r *Rotator) backupName(n uint) string {
	if n == 0 {
		return r.filename
	}
	return r.filename + "." + strconv.Itoa(int(n))
}

func (r *Rotator) dir() string {
	return filepath.Dir(r.filename)
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

func (r *Rotator) openNew() error {
	err := os.MkdirAll(r.dir(), r.dirMode())
	if err != nil {
		return errors.Wrap(err, "failed to make directories for new file")
	}

	_, err = os.Stat(r.filename)
	if err == nil {
		if err = r.rotate(); err != nil {
			return err
		}
	}

	return r.openFile()
}

func (r *Rotator) openFile() error {
	err := os.MkdirAll(r.dir(), r.dirMode())
	if err != nil {
		return errors.Wrap(err, "failed to make directories for new file")
	}

	r.file, err = os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, r.permissions)
	if err != nil {
		return errors.Wrap(err, "failed to open new file")
	}

	return nil
}

func (r *Rotator) closeFile() error {
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	r.size = 0
	return err
}

func (r *Rotator) purgeOldBackups() error {
	for i := r.maxBackups; i < MaxBackupsLimit; i++ {
		name := r.backupName(i + 1)

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

func (r *Rotator) rotate() error {
	if err := r.closeFile(); err != nil {
		return errors.Wrap(err, "error file closing current file")
	}

	for i := r.maxBackups + 1; i > 0; i-- {
		old := r.backupName(i - 1)
		older := r.backupName(i)

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
		}
	}

	return r.purgeOldBackups()
}
