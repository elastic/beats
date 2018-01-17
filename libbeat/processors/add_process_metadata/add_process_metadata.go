package add_process_metadata

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const name = "add_process_metadata"

type Processor struct {
	Config
	log     *logp.Logger
	cgroups *common.Cache
}

func New(c *common.Config) (processors.Processor, error) {
	config := defaultConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unpack %v config", name)
	}

	log := logp.NewLogger(name)
	cgroups := common.NewCacheWithRemovalListener(5*time.Minute, 100, func(k common.Key, v common.Value) {
		log.Debugf("evicted cached cgroups for PID=%v", k)
	})
	cgroups.StartJanitor(5 * time.Second)

	return &Processor{
		Config:  config,
		log:     logp.NewLogger(name),
		cgroups: cgroups,
	}, nil
}

func (p *Processor) Run(event *beat.Event) (*beat.Event, error) {
	var cgroups map[string]string
	for _, field := range p.PIDFields {
		v, err := event.GetValue(field)
		if err != nil {
			continue
		}

		pid, ok := tryToInt(v)
		if !ok {
			p.log.Debugf("field %v is not a PID (type=%T, value=%v)", field, v, v)
			continue
		}

		cgroups, err = p.getProcessCgroups(pid)
		if err != nil && os.IsNotExist(errors.Cause(err)) {
			continue
		}
		if err != nil {
			p.log.Debugf("failed to get cgroups for pid=%v: %v", pid, err)
		}

		break
	}

	if len(cgroups) == 0 {
		return event, nil
	}

	// Write data to targets.
	for _, metadata := range p.Metadata {
		switch metadata {
		case ContainerID:
			if cid := getContainerID(cgroups); cid != "" {
				event.PutValue(p.ContainerIDTarget, cid)
			}
		case Cgroups:
			event.PutValue(p.CgroupsTarget, cgroups)
		}
	}
	return event, nil
}

func (p *Processor) String() string {
	return fmt.Sprintf("add_process_metadata=[pid_fields=[%v], "+
		"metadata_types=%v, target.container_id=%v target.cgroups=%v]",
		strings.Join(p.PIDFields, ","),
		p.Metadata,
		p.ContainerIDTarget,
		p.CgroupsTarget)
}

// getProcessCgroups returns a mapping of cgroup subsystem name to path. It
// returns an error if it failed to retrieve the cgroup info.
func (p *Processor) getProcessCgroups(pid int) (map[string]string, error) {
	// TODO (andrewkroh): Add -system.host flag like Metricbeat has to specify
	// where to find the host systems /proc mountpoint.

	cgroups, ok := p.cgroups.Get(pid).(map[string]string)
	if ok {
		p.log.Debugf("using cached cgroups for pid=%v", pid)
		return cgroups, nil
	}

	cgroups, err := processCgroupPaths("", pid)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read cgroups for pid=%v", pid)
	}

	p.cgroups.Put(pid, cgroups)
	return cgroups, nil
}

func tryToInt(number interface{}) (int, bool) {
	var rtn int
	switch v := number.(type) {
	case int:
		rtn = int(v)
	case int8:
		rtn = int(v)
	case int16:
		rtn = int(v)
	case int32:
		rtn = int(v)
	case int64:
		rtn = int(v)
	case uint:
		rtn = int(v)
	case uint8:
		rtn = int(v)
	case uint16:
		rtn = int(v)
	case uint32:
		rtn = int(v)
	case uint64:
		rtn = int(v)
	case string:
		var err error
		rtn, err = strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return rtn, true
}

// processCgroupPaths returns the cgroups to which a process belongs and the
// pathname of the cgroup relative to the mountpoint of the subsystem.
func processCgroupPaths(rootfsMountpoint string, pid int) (map[string]string, error) {
	if rootfsMountpoint == "" {
		rootfsMountpoint = "/"
	}

	cgroup, err := os.Open(filepath.Join(rootfsMountpoint, "proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return nil, err
	}
	defer cgroup.Close()

	paths := map[string]string{}
	sc := bufio.NewScanner(cgroup)
	for sc.Scan() {
		// http://man7.org/linux/man-pages/man7/cgroups.7.html
		// Format: hierarchy-ID:subsystem-list:cgroup-path
		// Example:
		// 2:cpu:/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242
		line := sc.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 3 {
			continue
		}

		path := fields[2]
		subsystems := strings.Split(fields[1], ",")
		for _, subsystem := range subsystems {
			paths[subsystem] = path
		}
	}

	return paths, sc.Err()
}

func getContainerID(cgroups map[string]string) string {
	for _, path := range cgroups {
		if strings.HasPrefix(path, "/docker/") {
			return filepath.Base(path)
		}
	}

	return ""
}
