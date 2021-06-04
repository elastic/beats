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

// get is the linux implementation for fetching Memory data
func get(rootfs string) (Memory, error) {
	table, err := ParseMeminfo(rootfs)
	if err != nil {
		return Memory{}, errors.Wrap(err, "error fetching meminfo")
	}

	memData := newMemory()

	var free, cached uint64
	if total, ok := table["MemTotal"]; ok {
		memData.Total.Some(total)
	}
	if free, ok := table["MemFree"]; ok {
		memData.Free.Some(free)
	}
	if cached, ok := table["Cached"]; ok {
		memData.Cached.Some(cached)
	}

	// overlook parsing issues here
	// On the very small chance some of these don't exist,
	// It's not the end of the world
	buffers, _ := table["Buffers"]

	if memAvail, ok := table["MemAvailable"]; ok {
		// MemAvailable is in /proc/meminfo (kernel 3.14+)
		memData.ActualFree.Some(memAvail)
	} else {
		// in the future we may want to find another way to do this.
		// "MemAvailable" and other more derivied metrics
		// Are very relative, and can be unhelpful in cerntain workloads
		// We may want to find a way to more clearly express to users
		// where a certain value is coming from and what it represents

		// The use of `cached` here is particularly concerning,
		// as under certain intense DB server workloads, the cached memory can be quite large
		// and give the impression that we've passed memory usage watermark
		memData.ActualFree.Some(free + buffers + cached)
	}

	// Populate swap data
	swapTotal, okST := table["SwapTotal"]
	if okST {
		memData.SwapTotal.Some(swapTotal)
	}
	swapFree, okSF := table["SwapFree"]
	if okSF {
		memData.SwapTotal.Some(swapFree)
	}

	if okSF && okST {
		memData.SwapUsed.Some(swapTotal - swapFree)
	}

	return memData, nil

}

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
