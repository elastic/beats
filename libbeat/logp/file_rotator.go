package logp

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const RotatorMaxFiles = 1024
const DefaultKeepFiles = 7
const DefaultRotateEveryBytes = 10 * 1024 * 1024

type FileRotator struct {
	Path             string
	Name             string
	RotateEveryBytes *uint64
	KeepFiles        *int

	current      *os.File
	current_size uint64
}

func (rotator *FileRotator) CreateDirectory() error {
	fileinfo, err := os.Stat(rotator.Path)
	if err == nil {
		if !fileinfo.IsDir() {
			return fmt.Errorf("%s exists but it's not a directory", rotator.Path)
		}
	}

	if os.IsNotExist(err) {
		err = os.MkdirAll(rotator.Path, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rotator *FileRotator) CheckIfConfigSane() error {
	if len(rotator.Name) == 0 {
		return fmt.Errorf("File logging requires a name for the file names")
	}
	if rotator.KeepFiles == nil {
		rotator.KeepFiles = new(int)
		*rotator.KeepFiles = DefaultKeepFiles
	}
	if rotator.RotateEveryBytes == nil {
		rotator.RotateEveryBytes = new(uint64)
		*rotator.RotateEveryBytes = DefaultRotateEveryBytes
	}

	if *rotator.KeepFiles < 2 || *rotator.KeepFiles >= RotatorMaxFiles {
		return fmt.Errorf("The number of files to keep should be between 2 and %d", RotatorMaxFiles-1)
	}
	return nil
}

func (rotator *FileRotator) WriteLine(line []byte) error {
	if rotator.shouldRotate() {
		err := rotator.Rotate()
		if err != nil {
			return err
		}
	}

	line = append(line, '\n')
	_, err := rotator.current.Write(line)
	if err != nil {
		return err
	}
	rotator.current_size += uint64(len(line))

	return nil
}

func (rotator *FileRotator) shouldRotate() bool {
	if rotator.current == nil {
		return true
	}

	if rotator.current_size >= *rotator.RotateEveryBytes {
		return true
	}

	return false
}

func (rotator *FileRotator) FilePath(file_no int) string {
	if file_no == 0 {
		return filepath.Join(rotator.Path, rotator.Name)
	}
	filename := strings.Join([]string{rotator.Name, strconv.Itoa(file_no)}, ".")
	return filepath.Join(rotator.Path, filename)
}

func (rotator *FileRotator) FileExists(file_no int) bool {
	file_path := rotator.FilePath(file_no)
	_, err := os.Stat(file_path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func (rotator *FileRotator) Rotate() error {

	if rotator.current != nil {
		if err := rotator.current.Close(); err != nil {
			return err
		}
	}

	// delete any extra files, normally we shouldn't have any
	for file_no := *rotator.KeepFiles; file_no < RotatorMaxFiles; file_no++ {
		if rotator.FileExists(file_no) {
			perr := os.Remove(rotator.FilePath(file_no))
			if perr != nil {
				return perr
			}
		}
	}

	// shift all files from last to first
	for fileNo := *rotator.KeepFiles - 1; fileNo >= 0; fileNo-- {
		if !rotator.FileExists(fileNo) {
			// file doesn't exist, don't rotate
			continue
		}
		file_path := rotator.FilePath(fileNo)

		if rotator.FileExists(fileNo + 1) {
			// next file exists, something is strange
			return fmt.Errorf("File %s exists, when rotating would overwrite it", rotator.FilePath(fileNo+1))
		}

		err := os.Rename(file_path, rotator.FilePath(fileNo+1))
		if err != nil {
			return err
		}
	}

	// create the new file
	file_path := rotator.FilePath(0)
	current, err := os.Create(file_path)
	if err != nil {
		return err
	}
	rotator.current = current
	rotator.current_size = 0

	// delete the extra file, ignore errors here
	file_path = rotator.FilePath(*rotator.KeepFiles)
	os.Remove(file_path)

	return nil
}
