package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/server"
	"github.com/elastic/beats/metricbeat/mb"
)

type metricProcessor struct {
	paths       map[string]PathConfig
	defaultPath PathConfig
	sync.RWMutex
}

func NewMetricProcessor(paths []PathConfig, defaultPath PathConfig) *metricProcessor {
	pathMap := map[string]PathConfig{}
	for _, path := range paths {
		pathMap[path.Path] = path
	}

	return &metricProcessor{
		paths:       pathMap,
		defaultPath: defaultPath,
	}
}

func (m *metricProcessor) AddPath(path PathConfig) {
	m.Lock()
	m.paths[path.Path] = path
	m.Unlock()
}

func (m *metricProcessor) RemovePath(path PathConfig) {
	m.Lock()
	delete(m.paths, path.Path)
	m.Unlock()
}

func (p *metricProcessor) Process(event server.Event) (common.MapStr, error) {
	urlRaw, ok := event.GetMeta()["path"]
	if !ok {
		return nil, errors.New("Malformed HTTP event. Path missing.")
	}
	url, _ := urlRaw.(string)

	typeRaw, ok := event.GetMeta()["Content-Type"]
	if !ok {
		return nil, errors.New("Unable to get Content-Type of request")
	}
	contentType := typeRaw.(string)
	pathConf := p.findPath(url)

	bytesRaw, ok := event.GetEvent()[server.EventDataKey]
	if !ok {
		return nil, errors.New("Unable to retrieve response bytes")
	}

	bytes, _ := bytesRaw.([]byte)
	if len(bytes) == 0 {
		return nil, errors.New("Request has no data")
	}

	out := common.MapStr{}
	switch contentType {
	case "application/json":
		err := json.Unmarshal(bytes, &out)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New(fmt.Sprintf("Unsupported Content-Type: %s", contentType))
	}

	out[mb.NamespaceKey] = pathConf.Namespace
	if len(pathConf.Fields) != 0 {
		// Overwrite any keys that are present in the incoming payload
		common.MergeFields(out, pathConf.Fields, true)
	}
	return out, nil
}

func (p *metricProcessor) findPath(url string) *PathConfig {
	for path, conf := range p.paths {
		if strings.Index(url, path) == 0 {
			return &conf
		}
	}

	return &p.defaultPath
}
