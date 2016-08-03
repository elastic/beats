package cgroup

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// BlockIOSubsystem contains limits and metrics from the "blkio" subsystem. The
// blkio subsystem controls and monitors access to I/O on block devices by tasks
// in a cgroup.
//
// https://www.kernel.org/doc/Documentation/cgroup-v1/blkio-controller.txt
type BlockIOSubsystem struct {
	Metadata
	Throttle ThrottlePolicy `json:"throttle,omitempty"` // Throttle limits for upper IO rates and metrics.
	//CFQ      CFQScheduler   `json:"cfq,omitempty"`      // Completely fair queue scheduler limits and metrics.
}

// CFQScheduler contains limits and metrics for the proportional weight time
// based division of disk policy. It is implemented in CFQ. Hence this policy
// takes effect only on leaf nodes when CFQ is being used.
//
// https://www.kernel.org/doc/Documentation/block/cfq-iosched.txt
type CFQScheduler struct {
	Weight  uint64      `json:"weight"` // Default weight for all devices unless overridden. Allowed range of weights is from 10 to 1000.
	Devices []CFQDevice `json:"devices,omitempty"`
}

// CFQDevice contains CFQ limits and metrics associated with a single device.
type CFQDevice struct {
	DeviceID DeviceID `json:"device_id"` // ID of the device.

	// Proportional weight for the device. 0 means a per device weight is not set and
	// that the blkio.weight value is used.
	Weight uint64 `json:"weight"`

	TimeMs           uint64          `json:"time_ms"`          // Disk time allocated to cgroup per device in milliseconds.
	Sectors          uint64          `json:"sectors"`          // Number of sectors transferred to/from disk by the cgroup.
	Bytes            OperationValues `json:"io_service_bytes"` // Number of bytes transferred to/from the disk by the cgroup.
	IOs              OperationValues `json:"io_serviced"`      // Number of IO operations issued to the disk by the cgroup.
	ServiceTimeNanos OperationValues `json:"io_service_time"`  // Amount of time between request dispatch and request completion for the IOs done by this cgroup.
	WaitTimeNanos    OperationValues `json:"io_wait_time"`     // Amount of time the IOs for this cgroup spent waiting in the scheduler queues for service.
	Merges           OperationValues `json:"io_merged"`        // Total number of bios/requests merged into requests belonging to this cgroup.
}

// ThrottlePolicy contains the upper IO limits and metrics for devices used
// by the cgroup.
type ThrottlePolicy struct {
	Devices    []ThrottleDevice `json:"devices,omitempty"`      // Device centric view of limits and metrics.
	TotalBytes uint64           `json:"total_io_service_bytes"` // Total number of bytes serviced by all devices.
	TotalIOs   uint64           `json:"total_io_serviced"`      // Total number of IO operations serviced by all devices.
}

// ThrottleDevice contains throttle limits and metrics associated with a single device.
type ThrottleDevice struct {
	DeviceID DeviceID `json:"device_id"` // ID of the device.

	ReadLimitBPS   uint64 `json:"read_bps_device"`   // Read limit in bytes per second (BPS). Zero means no limit.
	WriteLimitBPS  uint64 `json:"write_bps_device"`  // Write limit in bytes per second (BPS). Zero mean no limit.
	ReadLimitIOPS  uint64 `json:"read_iops_device"`  // Read limit in IOPS. Zero means no limit.
	WriteLimitIOPS uint64 `json:"write_iops_device"` // Write limit in IOPS. Zero means no limit.

	Bytes OperationValues `json:"io_service_bytes"` // Number of bytes transferred to/from the disk by the cgroup.
	IOs   OperationValues `json:"io_serviced"`      // Number of IO operations issued to the disk by the cgroup.
}

// OperationValues contains the I/O limits or metrics associated with read,
// write, sync, and async operations.
type OperationValues struct {
	Read  uint64 `json:"read"`
	Write uint64 `json:"write"`
	Async uint64 `json:"async"`
	Sync  uint64 `json:"sync"`
}

// DeviceID identifies a Linux block device.
type DeviceID struct {
	Major uint64
	Minor uint64
}

// blkioValue holds a single blkio value associated with a device.
type blkioValue struct {
	DeviceID
	Operation string
	Value     uint64
}

// get reads metrics from the "blkio" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (blkio *BlockIOSubsystem) get(path string) error {
	if err := blkioThrottle(path, blkio); err != nil {
		return err
	}

	// TODO(akroh): Implement reading for the CFQ values.

	return nil
}

// blkioThrottle reads all of the limits and metrics associated with blkio
// throttling policy.
func blkioThrottle(path string, blkio *BlockIOSubsystem) error {
	devices := map[DeviceID]*ThrottleDevice{}

	getDevice := func(id DeviceID) *ThrottleDevice {
		td := devices[id]
		if td == nil {
			td = &ThrottleDevice{DeviceID: id}
			devices[id] = td
		}
		return td
	}

	values, err := readBlkioValues(path, "blkio.throttle.io_service_bytes")
	if err != nil {
		return err
	}
	if values != nil {
		for id, opValues := range collectOpValues(values) {
			getDevice(id).Bytes = *opValues
		}
	}

	values, err = readBlkioValues(path, "blkio.throttle.io_serviced")
	if err != nil {
		return err
	}
	if values != nil {
		for id, opValues := range collectOpValues(values) {
			getDevice(id).IOs = *opValues
		}
	}

	values, err = readBlkioValues(path, "blkio.throttle.read_bps_device")
	if err != nil {
		return err
	}
	if values != nil {
		for _, bv := range values {
			getDevice(bv.DeviceID).ReadLimitBPS = bv.Value
		}
	}

	values, err = readBlkioValues(path, "blkio.throttle.write_bps_device")
	if err != nil {
		return err
	}
	if values != nil {
		for _, bv := range values {
			getDevice(bv.DeviceID).WriteLimitBPS = bv.Value
		}
	}

	values, err = readBlkioValues(path, "blkio.throttle.read_iops_device")
	if err != nil {
		return err
	}
	if values != nil {
		for _, bv := range values {
			getDevice(bv.DeviceID).ReadLimitIOPS = bv.Value
		}
	}

	values, err = readBlkioValues(path, "blkio.throttle.write_iops_device")
	if err != nil {
		return err
	}
	if values != nil {
		for _, bv := range values {
			getDevice(bv.DeviceID).WriteLimitIOPS = bv.Value
		}
	}

	blkio.Throttle.Devices = make([]ThrottleDevice, 0, len(devices))
	for _, dev := range devices {
		blkio.Throttle.Devices = append(blkio.Throttle.Devices, *dev)
		blkio.Throttle.TotalBytes += dev.Bytes.Read + dev.Bytes.Write
		blkio.Throttle.TotalIOs += dev.IOs.Read + dev.IOs.Write
	}

	return nil
}

// collectOpValues collects the discreet I/O values (e.g. read, write, sync,
// async) for a given device into a single OperationValues object. It returns a
// mapping of device ID to OperationValues.
func collectOpValues(values []blkioValue) map[DeviceID]*OperationValues {
	opValues := map[DeviceID]*OperationValues{}
	for _, bv := range values {
		opValue := opValues[bv.DeviceID]
		if opValue == nil {
			opValue = &OperationValues{}
			opValues[bv.DeviceID] = opValue
		}

		switch bv.Operation {
		case "read":
			opValue.Read = bv.Value
		case "write":
			opValue.Write = bv.Value
		case "async":
			opValue.Async = bv.Value
		case "sync":
			opValue.Sync = bv.Value
		}
	}

	return opValues
}

// readDeviceValues reads values from a single blkio file.
// It expects to read values like "245:1 read 18880" or "254:1 1909". It returns
// an array containing an entry for each valid line read.
func readBlkioValues(path ...string) ([]blkioValue, error) {
	f, err := os.Open(filepath.Join(path...))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var values []blkioValue
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if len(line) == 0 {
			continue
		}
		// Valid lines start with a device ID.
		if !unicode.IsNumber(rune(line[0])) {
			continue
		}

		v, err := parseBlkioValue(sc.Text())
		if err != nil {
			return nil, err
		}

		values = append(values, v)
	}

	return values, nil
}

func isColonOrSpace(r rune) bool {
	return unicode.IsSpace(r) || r == ':'
}

func parseBlkioValue(line string) (blkioValue, error) {
	fields := strings.FieldsFunc(line, isColonOrSpace)
	if len(fields) != 3 && len(fields) != 4 {
		return blkioValue{}, ErrInvalidFormat
	}

	major, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return blkioValue{}, err
	}

	minor, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return blkioValue{}, err
	}

	var value uint64
	var operation string
	if len(fields) == 3 {
		value, err = parseUint([]byte(fields[2]))
		if err != nil {
			return blkioValue{}, err
		}
	} else {
		operation = strings.ToLower(fields[2])

		value, err = parseUint([]byte(fields[3]))
		if err != nil {
			return blkioValue{}, err
		}
	}

	return blkioValue{
		DeviceID:  DeviceID{major, minor},
		Operation: operation,
		Value:     value,
	}, nil
}
