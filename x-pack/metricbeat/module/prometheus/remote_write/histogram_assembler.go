// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"math"
	"sort"
	"strconv"
	"time"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/prometheus/collector"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type histogramAssemblyConfig struct {
	QuietPeriod          time.Duration
	HardTimeout          time.Duration
	MaxPendingHistograms int
	MaxPendingBuckets    int
}

func defaultHistogramAssemblyConfig() histogramAssemblyConfig {
	return histogramAssemblyConfig{
		QuietPeriod:          5 * time.Second,
		HardTimeout:          30 * time.Second,
		MaxPendingHistograms: 10_000,
		MaxPendingBuckets:    100_000,
	}
}

type histogramIdentity struct {
	bucketMetricName string
	labelsKey        string
	timestamp        time.Time
}

func (id histogramIdentity) key() string {
	return id.bucketMetricName + "\x00" + id.labelsKey + "\x00" + id.timestamp.String()
}

type pendingHistogram struct {
	identity   histogramIdentity
	labels     mapstr.M
	buckets    map[float64]*p.Bucket
	hasInf     bool
	firstSeen  time.Time
	lastUpdate time.Time
}

type tombstone struct {
	expiresAt time.Time
	seq       uint64
}

type histogramAssembler struct {
	cfg histogramAssemblyConfig

	pending    map[string]*pendingHistogram
	tombstones map[string]tombstone
	tombSeq    uint64

	stats histogramAssemblerStats
	mon   *histogramAssemblerMonitoring
}

type histogramAssemblerStats struct {
	PendingHistograms     int
	PendingBuckets        int
	RetainedHistograms    int
	RetainedBuckets       int
	Tombstones            int
	LateDrops             uint64
	FlushesComplete       uint64
	FlushesPartial        uint64
	RetentionDrops        uint64
	RetentionBucketDrops  uint64
	ShutdownDroppedHists  uint64
	ShutdownDroppedEvents uint64
}

func newHistogramAssembler(cfg histogramAssemblyConfig, mon *histogramAssemblerMonitoring) *histogramAssembler {
	a := &histogramAssembler{
		cfg:        cfg,
		pending:    make(map[string]*pendingHistogram),
		tombstones: make(map[string]tombstone),
		mon:        mon,
	}
	a.syncPendingGauges()
	return a
}

func (a *histogramAssembler) syncPendingGauges() {
	if a.mon == nil {
		return
	}
	s := a.statsSnapshot()
	a.mon.setPending(s.PendingHistograms, s.PendingBuckets)
}

func (a *histogramAssembler) statsSnapshot() histogramAssemblerStats {
	s := a.stats
	s.PendingHistograms = len(a.pending)
	s.Tombstones = len(a.tombstones)
	var buckets int
	for _, p := range a.pending {
		buckets += len(p.buckets)
	}
	s.PendingBuckets = buckets
	return s
}

func histogramIdentityFromParts(bucketMetricName string, labels mapstr.M, ts time.Time) histogramIdentity {
	canonical := labels.Clone()
	_ = canonical.Delete("le")
	return histogramIdentity{
		bucketMetricName: bucketMetricName,
		labelsKey:        canonical.String(),
		timestamp:        ts,
	}
}

func parseLEUpperBound(le string) (float64, bool) {
	if le == "+Inf" {
		return math.Inf(1), true
	}
	v, err := strconv.ParseFloat(le, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

type capacityBatchImpact struct {
	newHistograms int
	newBuckets    int
}

func (a *histogramAssembler) capacityImpact(now time.Time, batch []bucketSample) capacityBatchImpact {
	groups := make([]preparedHistogramGroup, 0, len(batch))
	for _, s := range batch {
		id := histogramIdentityFromParts(s.bucketMetricName, s.labels, s.timestamp)
		groups = append(groups, preparedHistogramGroup{
			identity: id,
			key:      id.key(),
			labels:   s.labels,
			buckets:  s.buckets,
		})
	}
	return a.capacityImpactGrouped(now, groups)
}

// capacityImpactGrouped computes admission impact using precomputed identity keys.
func (a *histogramAssembler) capacityImpactGrouped(now time.Time, groups []preparedHistogramGroup) capacityBatchImpact {
	var impact capacityBatchImpact
	newHistInBatch := make(map[string]struct{})
	newBucketsInBatch := make(map[string]map[float64]struct{})

	for _, group := range groups {
		key := group.key
		if a.isActiveTombstone(key, now) {
			continue
		}
		if _, ok := a.pending[key]; ok {
			for _, b := range group.buckets {
				bound := b.upperBound
				if a.pending[key].hasBucket(bound) {
					continue
				}
				if batchHasBound(newBucketsInBatch, key, bound) {
					continue
				}
				recordBatchBound(newBucketsInBatch, key, bound)
				impact.newBuckets++
			}
			continue
		}
		if _, ok := newHistInBatch[key]; !ok {
			newHistInBatch[key] = struct{}{}
			impact.newHistograms++
		}
		for _, b := range group.buckets {
			bound := b.upperBound
			if batchHasBound(newBucketsInBatch, key, bound) {
				continue
			}
			recordBatchBound(newBucketsInBatch, key, bound)
			impact.newBuckets++
		}
	}
	return impact
}

func batchHasBound(m map[string]map[float64]struct{}, key string, bound float64) bool {
	bounds, ok := m[key]
	if !ok {
		return false
	}
	_, ok = bounds[boundKey(bound)]
	return ok
}

func recordBatchBound(m map[string]map[float64]struct{}, key string, bound float64) {
	if m[key] == nil {
		m[key] = make(map[float64]struct{})
	}
	m[key][boundKey(bound)] = struct{}{}
}

func boundKey(bound float64) float64 {
	if math.IsInf(bound, 1) {
		return math.Inf(1)
	}
	return bound
}

func (a *histogramAssembler) wouldExceedCapacity(impact capacityBatchImpact, retained flushEventCapacity) bool {
	if len(a.pending)+impact.newHistograms+retained.histograms > a.cfg.MaxPendingHistograms {
		return true
	}
	pendingBuckets := a.statsSnapshot().PendingBuckets
	if pendingBuckets+impact.newBuckets+retained.buckets > a.cfg.MaxPendingBuckets {
		return true
	}
	return false
}

type bucketSample struct {
	bucketMetricName string
	labels           mapstr.M
	timestamp        time.Time
	buckets          []bucketUpdate
}

type bucketUpdate struct {
	upperBound      float64
	cumulativeCount float64
}

// preparedHistogramGroup is one histogram identity with precomputed key and
// bucket updates for a single remote_write request.
type preparedHistogramGroup struct {
	identity histogramIdentity
	key      string
	labels   mapstr.M
	buckets  []bucketUpdate
}

func (p *pendingHistogram) hasBucket(upperBound float64) bool {
	_, ok := p.buckets[boundKey(upperBound)]
	return ok
}

func (a *histogramAssembler) isActiveTombstone(key string, now time.Time) bool {
	ts, ok := a.tombstones[key]
	if !ok {
		return false
	}
	return now.Before(ts.expiresAt)
}

func (a *histogramAssembler) ingest(now time.Time, sample bucketSample) bool {
	id := histogramIdentityFromParts(sample.bucketMetricName, sample.labels, sample.timestamp)
	return a.ingestGrouped(now, preparedHistogramGroup{
		identity: id,
		key:      id.key(),
		labels:   sample.labels,
		buckets:  sample.buckets,
	})
}

// ingestGrouped applies one prepared histogram identity using a precomputed key.
func (a *histogramAssembler) ingestGrouped(now time.Time, group preparedHistogramGroup) bool {
	key := group.key
	if a.isActiveTombstone(key, now) {
		a.stats.LateDrops++
		if a.mon != nil {
			a.mon.observeLateBucket(true)
		}
		return false
	}
	delete(a.tombstones, key)

	entry := a.pending[key]
	if entry == nil {
		entry = &pendingHistogram{
			identity:   group.identity,
			labels:     group.labels.Clone(),
			buckets:    make(map[float64]*p.Bucket),
			firstSeen:  now,
			lastUpdate: now,
		}
		a.pending[key] = entry
	}

	changed := false
	for _, upd := range group.buckets {
		bk := boundKey(upd.upperBound)
		val := upd.cumulativeCount
		if existing, ok := entry.buckets[bk]; ok {
			if val <= existing.GetCumulativeCount() {
				continue
			}
		}
		count := val
		upper := upd.upperBound
		entry.buckets[bk] = &p.Bucket{
			CumulativeCount: &count,
			UpperBound:      &upper,
		}
		if math.IsInf(upd.upperBound, 1) {
			entry.hasInf = true
		}
		changed = true
	}
	if changed {
		entry.lastUpdate = now
	}
	a.syncPendingGauges()
	return changed
}

func (a *histogramAssembler) expireTombstones(now time.Time) {
	for key, ts := range a.tombstones {
		if !now.Before(ts.expiresAt) {
			delete(a.tombstones, key)
		}
	}
}

func (a *histogramAssembler) addTombstone(key string, now time.Time) {
	a.tombSeq++
	// Keep the flushed identity closed for one hard timeout so late buckets
	// cannot reopen it and produce a duplicate histogram event.
	ts := tombstone{
		expiresAt: now.Add(a.cfg.HardTimeout),
		seq:       a.tombSeq,
	}
	a.tombstones[key] = ts
	a.enforceTombstoneLimit()
}

func (a *histogramAssembler) enforceTombstoneLimit() {
	max := a.cfg.MaxPendingHistograms
	if len(a.tombstones) <= max {
		return
	}
	type kv struct {
		key string
		ts  tombstone
	}
	items := make([]kv, 0, len(a.tombstones))
	for k, v := range a.tombstones {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ts.expiresAt.Equal(items[j].ts.expiresAt) {
			return items[i].ts.seq < items[j].ts.seq
		}
		return items[i].ts.expiresAt.Before(items[j].ts.expiresAt)
	})
	for len(a.tombstones) > max {
		delete(a.tombstones, items[0].key)
		items = items[1:]
	}
}

type flushReason int

const (
	flushReasonQuiet flushReason = iota
	flushReasonHard
)

func (a *histogramAssembler) flushExpired(now time.Time, counterCache collector.CounterCache, metricsCount bool) map[string]mb.Event {
	a.expireTombstones(now)

	events := make(map[string]mb.Event)
	var toFlush []struct {
		key    string
		reason flushReason
	}

	for key, entry := range a.pending {
		hardDue := !now.Before(entry.firstSeen.Add(a.cfg.HardTimeout))
		quietDue := entry.hasInf && !now.Before(entry.lastUpdate.Add(a.cfg.QuietPeriod))
		switch {
		case hardDue:
			toFlush = append(toFlush, struct {
				key    string
				reason flushReason
			}{key, flushReasonHard})
		case quietDue:
			toFlush = append(toFlush, struct {
				key    string
				reason flushReason
			}{key, flushReasonQuiet})
		}
	}

	sort.Slice(toFlush, func(i, j int) bool {
		if toFlush[i].reason == toFlush[j].reason {
			return toFlush[i].key < toFlush[j].key
		}
		return toFlush[i].reason == flushReasonHard
	})

	for _, f := range toFlush {
		entry := a.pending[f.key]
		if entry == nil {
			continue
		}
		eventKey, ev := a.entryToEvent(entry, counterCache, metricsCount)
		if existing, ok := events[eventKey]; ok {
			existing.ModuleFields.Update(ev.ModuleFields)
			if len(ev.RootFields) > 0 {
				if existing.RootFields == nil {
					existing.RootFields = mapstr.M{}
				}
				existing.RootFields.Update(ev.RootFields)
			}
			ev = existing
		}
		events[eventKey] = ev
		delete(a.pending, f.key)
		a.addTombstone(f.key, now)
		switch f.reason {
		case flushReasonHard:
			a.stats.FlushesPartial++
		case flushReasonQuiet:
			a.stats.FlushesComplete++
		}
		if a.mon != nil {
			a.mon.observeFlush(f.reason)
		}
	}
	a.syncPendingGauges()
	return events
}

func (a *histogramAssembler) entryToEvent(entry *pendingHistogram, counterCache collector.CounterCache, metricsCount bool) (string, mb.Event) {
	bucketList := histogramBucketsInOrder(entry.buckets)
	hist := p.Histogram{Bucket: bucketList}
	name := entry.identity.bucketMetricName
	baseName := trimBucketSuffix(name)

	labelsHash := entry.labels.String() + entry.identity.timestamp.String()
	event := mb.Event{
		RootFields:   mapstr.M{},
		ModuleFields: mapstr.M{},
		Timestamp:    entry.identity.timestamp,
	}
	if len(entry.labels) > 0 {
		event.ModuleFields["labels"] = entry.labels
	}

	data := mapstr.M{
		baseName: mapstr.M{
			"histogram": collector.PromHistogramToES(counterCache, name, entry.labels, &hist),
		},
	}
	event.ModuleFields.Update(data)

	if metricsCount {
		if _, hasLabels := event.ModuleFields["labels"]; hasLabels {
			event.RootFields["metrics_count"] = len(event.ModuleFields) - 1
		} else {
			event.RootFields["metrics_count"] = len(event.ModuleFields)
		}
	}
	return labelsHash, event
}

func histogramBucketsInOrder(buckets map[float64]*p.Bucket) []*p.Bucket {
	ordered := make([]*p.Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		ordered = append(ordered, bucket)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].GetUpperBound() < ordered[j].GetUpperBound()
	})
	return ordered
}

func trimBucketSuffix(metricName string) string {
	if len(metricName) > len("_bucket") && metricName[len(metricName)-len("_bucket"):] == "_bucket" {
		return metricName[:len(metricName)-len("_bucket")]
	}
	return metricName
}

func (a *histogramAssembler) shutdown(now time.Time) histogramAssemblerStats {
	dropped := uint64FromCount(len(a.pending))
	a.stats.ShutdownDroppedHists += dropped
	a.pending = make(map[string]*pendingHistogram)
	a.tombstones = make(map[string]tombstone)
	a.syncPendingGauges()
	return a.statsSnapshot()
}

func (a *histogramAssembler) dropRetainedEvents(count int) {
	a.stats.ShutdownDroppedEvents += uint64FromCount(count)
}
