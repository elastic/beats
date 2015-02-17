package outputs

import (
	"bytes"
	"io/ioutil"
	"os"
	"packetbeat/log"
	"path/filepath"
	"testing"
)

func Test_Rotator(t *testing.T) {

	if testing.Verbose() {
		log.LogInit(log.LOG_DEBUG, "", false, []string{"rotator"})
	}

	dir, err := ioutil.TempDir("", "test_rotator_")
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	log.DEBUG("rotator", "Direcotry: %s", dir)

	rotator := FileRotator{
		Path:             dir,
		Name:             "packetbeat",
		RotateEveryBytes: 1000,
		KeepFiles:        3,
	}

	err = rotator.Rotate()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	if _, err = os.Stat(filepath.Join(dir, "packetbeat")); os.IsNotExist(err) {
		t.Errorf("File %s doesn't exist", filepath.Join(dir, "packetbeat"))
	}

	if err = rotator.WriteLine([]byte("1")); err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	err = rotator.Rotate()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	if err = rotator.WriteLine([]byte("2")); err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	err = rotator.Rotate()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	if err = rotator.WriteLine([]byte("3")); err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	err = rotator.Rotate()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	if err = rotator.WriteLine([]byte("4")); err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	file_0, err := ioutil.ReadFile(rotator.FilePath(0))
	if err != nil || bytes.Equal(file_0, []byte("4")) {
		t.Errorf("Wrong contents of file 0: %s, expected: %s", string(file_0), "4")
	}

	file_1, err := ioutil.ReadFile(rotator.FilePath(1))
	if err != nil || bytes.Equal(file_1, []byte("3")) {
		t.Errorf("Wrong contents of file 1: %s", string(file_1))
	}

	file_2, err := ioutil.ReadFile(rotator.FilePath(2))
	if err != nil || bytes.Equal(file_2, []byte("2")) {
		t.Errorf("Wrong contents of file 2: %s", string(file_2))
	}

	if rotator.FileExists(3) {
		t.Errorf("File path %s shouldn't exist", rotator.FilePath(3))
	}

	os.RemoveAll(dir)
}

func Test_Rotator_By_Bytes(t *testing.T) {

	if testing.Verbose() {
		log.LogInit(log.LOG_DEBUG, "", false, []string{"rotator"})
	}

	dir, err := ioutil.TempDir("", "test_rotator_")
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	log.DEBUG("rotator", "Direcotry: %s", dir)

	rotator := FileRotator{
		Path:             dir,
		Name:             "packetbeat",
		RotateEveryBytes: 100,
		KeepFiles:        7,
	}

	for i := 0; i < 300; i++ {
		rotator.WriteLine([]byte("01234567890"))
	}
}
