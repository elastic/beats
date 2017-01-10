package perf

import (
	"fmt"
	"github.com/elastic/beats/metricbeat/module/ceph"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type socket struct {
	sockId   string
	sockType string
	socket   string
}

func findSockets(c *ceph.CephConfig) ([]*socket, error) {
	listing, err := ioutil.ReadDir(c.SocketDir)
	if err != nil {
		return []*socket{}, fmt.Errorf("Failed to read socket directory '%s': %v", c.SocketDir, err)
	}
	sockets := make([]*socket, 0, len(listing))
	for _, info := range listing {
		f := info.Name()
		var sockType string
		var sockPrefix string
		if strings.HasPrefix(f, c.MonPrefix) {
			sockType = typeMon
			sockPrefix = monPrefix
		}
		if strings.HasPrefix(f, c.OsdPrefix) {
			sockType = typeOsd
			sockPrefix = osdPrefix

		}
		if sockType == typeOsd || sockType == typeMon {
			path := filepath.Join(c.SocketDir, f)
			sockets = append(sockets, &socket{parseSockId(f, sockPrefix, c.SocketSuffix), sockType, path})
		}
	}
	return sockets, nil
}

func parseSockId(fname, prefix, suffix string) string {
	s := fname
	s = strings.TrimPrefix(s, prefix)
	s = strings.TrimSuffix(s, suffix)
	s = strings.Trim(s, ".-_")
	return s
}
