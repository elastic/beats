package lutool

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

type Runner interface {
	Exec(common.MapStr) (common.MapStr, error)
}

type cachedLookup struct {
	FieldsUnderRoot bool
	name            string
	cache           *cache
	runner          *runner
	keyBuilder      KeyBuilder
}

type CacheConfig struct {
	Backoff         BackoffConfig `config:"backoff"`
	GCInterval      time.Duration `config:"gc_interval"       validate:"min=0s"`
	ExpireUnused    time.Duration `config:"expire_unused"     validate:"min=1s"`
	FieldsUnderRoot bool          `config:"fields_under_root"`
}

var debugf = logp.MakeDebug("lookup")

var DefaultCacheConfig = CacheConfig{
	Backoff: BackoffConfig{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Max:      60 * time.Second,
	},
	GCInterval:      10 * time.Second,
	ExpireUnused:    60 * time.Second,
	FieldsUnderRoot: false,
}

func NewCachedLookupTool(
	name string,
	config CacheConfig,
	keyBuilder KeyBuilder,
	backend Runner,
) (processors.Processor, error) {
	runner, err := newRunner(config.Backoff, backend)
	if err != nil {
		return nil, err
	}

	module := &cachedLookup{
		name:            name,
		FieldsUnderRoot: config.FieldsUnderRoot,
		cache:           newCache(config.GCInterval, config.ExpireUnused),
		runner:          runner,
		keyBuilder:      keyBuilder,
	}
	return module, nil
}

func (l *cachedLookup) Run(event common.MapStr) (common.MapStr, error) {
	var ts time.Time
	timestamp, found := event["@timestamp"]
	if found {
		if commonTS, ok := timestamp.(common.Time); ok {
			ts = time.Time(commonTS)
		} else {
			ts = time.Now()
		}
	} else {
		ts = time.Now()
	}

	key, ok := l.keyBuilder.ExtractKey(event)
	if !ok { // if key is not extractable from event, do not touch the event
		return event, nil
	}

	entry, err := l.cache.getEntry(ts, key)
	if err != nil {
		return event, err
	}

	err = entry.exec.Do(func() error { return l.runner.do(entry, event) })
	if err != nil {
		l.annotateError(event, err)
		return event, nil
	}

	fields := entry.value.value()
	return l.mergeFields(event, fields), nil
}

func (l *cachedLookup) String() string {
	return l.name
}

func (l *cachedLookup) annotateError(event common.MapStr, err error) common.MapStr {
	// TODO: where/how do we want to store errors in events?
	return event
}

func (l *cachedLookup) mergeFields(event, fields common.MapStr) common.MapStr {
	debugf("merging fields")
	err := common.MergeFields(event, fields, l.FieldsUnderRoot)
	if err != nil {
		logp.Warn("Merging lookup fields failed with: ", err)
		return l.annotateError(event, err)
	}
	return event
}

func (fn KeyBuilderFunc) ExtractKey(event common.MapStr) (Key, bool) {
	return fn(event)
}
