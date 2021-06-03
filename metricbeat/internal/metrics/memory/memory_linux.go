package memory

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ParseMeminfo parses the contents of /proc/meminfo into a hashmap
func ParseMeminfo(rootfs string) (map[string]uint64, error) {
	table := map[string]uint64{}

	meminfoPath := filepath.Join(rootfs, "/proc/meminfo")
	err := readFile(meminfoPath, func(line string) bool {
		fields := strings.Split(line, ":")

		if len(fields) != 2 {
			return true // skip on errors
		}

		valueUnit := strings.Fields(fields[1])
		value, err := strconv.ParseUint(valueUnit[0], 10, 64)
		if err != nil {
			return true // skip on errors
		}

		if len(valueUnit) > 1 && valueUnit[1] == "kB" {
			value *= 1024
		}
		table[fields[0]] = value

		return true
	})
	return table, err
}

func readFile(file string, handler func(string) bool) error {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrapf(err, "error reading file %s", file)
	}

	reader := bufio.NewReader(bytes.NewBuffer(contents))

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if !handler(string(line)) {
			break
		}
	}

	return nil
}
