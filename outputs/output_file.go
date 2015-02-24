package outputs

import (
	"encoding/json"
	"fmt"
	"os"
	"packetbeat/common"
	"packetbeat/config"
	"packetbeat/logp"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const RotatorMaxFiles = 1000

type FileOutputType struct {
	OutputInterface

	rotator FileRotator
}

type FileRotator struct {
	Path             string
	Name             string
	RotateEveryBytes uint64
	KeepFiles        int

	current      *os.File
	current_size uint64
}

func (out *FileOutputType) Init(config config.MothershipConfig) error {
	out.rotator.Path = config.Path
	out.rotator.Name = config.Filename
	out.rotator.RotateEveryBytes = uint64(config.Rotate_every_kb) * 1024
	if out.rotator.RotateEveryBytes == 0 {
		out.rotator.RotateEveryBytes = 10 * 1024 * 1024
	}
	out.rotator.KeepFiles = config.Number_of_files
	if out.rotator.KeepFiles == 0 {
		out.rotator.KeepFiles = 7
	}

	err := out.rotator.CreateDirectory()
	if err != nil {
		return err
	}

	err = out.rotator.CheckIfConfigSane()
	if err != nil {
		return err
	}

	return nil
}

func (out *FileOutputType) PublishIPs(name string, localAddrs []string) error {
	// not supported by this output type
	return nil
}

func (out *FileOutputType) UpdateLocalTopologyMap() {
	// not supported by this output type
}

func (out *FileOutputType) PublishEvent(ts time.Time, event common.MapStr) error {

	json_event, err := json.Marshal(event)
	if err != nil {
		logp.Err("Fail to convert the event to JSON: %s", err)
		return err
	}

	err = out.rotator.WriteLine(json_event)
	if err != nil {
		return err
	}

	return nil
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
	if rotator.KeepFiles < 2 || rotator.KeepFiles >= RotatorMaxFiles {
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
	_, err := rotator.current.Write(line)
	if err != nil {
		return err
	}
	_, err = rotator.current.Write([]byte("\n"))
	if err != nil {
		return err
	}
	rotator.current_size += uint64(len(line) + 1)

	return nil
}

func (rotator *FileRotator) shouldRotate() bool {
	if rotator.current == nil {
		return true
	}

	if rotator.current_size >= rotator.RotateEveryBytes {
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
	for file_no := rotator.KeepFiles; file_no < RotatorMaxFiles; file_no++ {
		if rotator.FileExists(file_no) {
			perr := os.Remove(rotator.FilePath(file_no))
			if perr != nil {
				return perr
			}
		}
	}

	// shift all files from last to first
	for file_no := rotator.KeepFiles - 1; file_no >= 0; file_no-- {
		if !rotator.FileExists(file_no) {
			// file doesn't exist, don't rotate
			continue
		}
		file_path := rotator.FilePath(file_no)

		if rotator.FileExists(file_no + 1) {
			// next file exists, something is strange
			return fmt.Errorf("File %s exists, when rotating would overwrite it", rotator.FilePath(file_no+1))
		}

		err := os.Rename(file_path, rotator.FilePath(file_no+1))
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
	file_path = rotator.FilePath(rotator.KeepFiles)
	os.Remove(file_path)

	return nil
}
