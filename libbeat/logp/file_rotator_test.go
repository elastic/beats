// +build !integration

package logp

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Rotator(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, true, []string{"rotator"})
	}

	dir, err := ioutil.TempDir("", "test_rotator_")
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	Debug("rotator", "Direcotry: %s", dir)

	rotateeverybytes := uint64(1000)
	keepfiles := 3

	rotator := FileRotator{
		Path:             dir,
		Name:             "packetbeat",
		RotateEveryBytes: &rotateeverybytes,
		KeepFiles:        &keepfiles,
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

	file0, err := ioutil.ReadFile(rotator.FilePath(0))
	if err != nil || bytes.Equal(file0, []byte("4")) {
		t.Errorf("Wrong contents of file 0: %s, expected: %s", string(file0), "4")
	}

	file1, err := ioutil.ReadFile(rotator.FilePath(1))
	if err != nil || bytes.Equal(file1, []byte("3")) {
		t.Errorf("Wrong contents of file 1: %s", string(file1))
	}

	file2, err := ioutil.ReadFile(rotator.FilePath(2))
	if err != nil || bytes.Equal(file2, []byte("2")) {
		t.Errorf("Wrong contents of file 2: %s", string(file2))
	}

	if rotator.FileExists(3) {
		t.Errorf("File path %s shouldn't exist", rotator.FilePath(3))
	}

	os.RemoveAll(dir)
}

func Test_Rotator_By_Bytes(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, true, []string{"rotator"})
	}

	dir, err := ioutil.TempDir("", "test_rotator_")
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}

	Debug("rotator", "Direcotry: %s", dir)

	rotateeverybytes := uint64(100)
	keepfiles := 3

	rotator := FileRotator{
		Path:             dir,
		Name:             "packetbeat",
		RotateEveryBytes: &rotateeverybytes,
		KeepFiles:        &keepfiles,
	}

	for i := 0; i < 300; i++ {
		rotator.WriteLine([]byte("01234567890"))
	}
}

func TestConfigSane(t *testing.T) {
	rotator := FileRotator{
		Name: "test",
	}
	assert.Nil(t, rotator.CheckIfConfigSane())

	keepfiles := 1023
	rotator = FileRotator{
		Name:      "test",
		KeepFiles: &keepfiles,
	}
	assert.Nil(t, rotator.CheckIfConfigSane())

	keepfiles = 10000
	rotator = FileRotator{
		Name:      "test",
		KeepFiles: &keepfiles,
	}
	assert.NotNil(t, rotator.CheckIfConfigSane())

	rotator = FileRotator{
		Name: "",
	}
	assert.NotNil(t, rotator.CheckIfConfigSane())

}
