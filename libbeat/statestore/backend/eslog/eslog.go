package eslog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-elasticsearch/v8"
)

var (
	errRegClosed    = errors.New("registry has been closed")
	errIndexInvalid = errors.New("cannot perform operation on index, a reindex is required")
	errKeyUnknown   = errors.New("key unknown")
)

// Registry configures access to Elasticsearch based stores.
type Registry struct {
	log *logp.Logger

	mu     sync.Mutex
	active bool

	settings Settings

	wg sync.WaitGroup
}

// Settings configures a new Registry.
type Settings struct {
	// Elasticsearch client configuration
	ESClient *elasticsearch.Client

	// Index name prefix for the stores
	IndexPrefix string

	// Reindex predicate that can trigger a reindex operation
	Reindex ReindexPredicate

	// Other Elasticsearch-specific settings can be added here
}

// ReindexPredicate is the type for configurable reindex checks.
type ReindexPredicate func(docCount int64) bool

// store implements an actual Elasticsearch based store.
type store struct {
	log      *logp.Logger
	lock     sync.RWMutex
	esClient *elasticsearch.Client
	index    string

	// nextSeqNo is the sequential counter that tracks
	// all updates to the store.
	nextSeqNo int64

	// internal state and metrics
	docCount        int64
	indexInvalid    bool
	needsReindex    bool
	reindexPred     ReindexPredicate
	lastReindexTime time.Time
}

func New(log *logp.Logger, settings Settings) (*Registry, error) {
	if settings.ESClient == nil {
		return nil, fmt.Errorf("Elasticsearch client is required")
	}

	if settings.Reindex == nil {
		settings.Reindex = defaultReindexPredicate
	}

	return &Registry{
		log:      log,
		active:   true,
		settings: settings,
	}, nil
}

func (r *Registry) Access(name string) (backend.Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.active {
		return nil, errRegClosed
	}

	logger := r.log.With("store", name)

	indexName := fmt.Sprintf("%s-%s", r.settings.IndexPrefix, name)
	store, err := openStore(logger, r.settings.ESClient, indexName, r.settings.Reindex)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	r.active = false
	r.mu.Unlock()

	r.wg.Wait()
	return nil
}

func openStore(log *logp.Logger, esClient *elasticsearch.Client, indexName string, reindexPred ReindexPredicate) (*store, error) {
	exists, err := indexExists(esClient, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to check index existence: %w", err)
	}

	if !exists {
		if err := createIndex(esClient, indexName); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	docCount, err := getDocCount(esClient, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	return &store{
		log:             log,
		esClient:        esClient,
		index:           indexName,
		nextSeqNo:       0,
		docCount:        docCount,
		indexInvalid:    false,
		needsReindex:    false,
		reindexPred:     reindexPred,
		lastReindexTime: time.Now(),
	}, nil
}

func (s *store) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	// No need to close Elasticsearch client here, as it's managed by the Registry
	return nil
}

func (s *store) Has(key string) (bool, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.indexInvalid {
		return false, errIndexInvalid
	}

	res, err := s.esClient.Exists(s.index, key)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

func (s *store) Set(key string, value interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.indexInvalid {
		return errIndexInvalid
	}

	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	res, err := s.esClient.Index(
		s.index,
		strings.NewReader(string(body)),
		s.esClient.Index.WithDocumentID(key),
		s.esClient.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	s.docCount++
	s.nextSeqNo++

	if s.reindexPred(s.docCount) {
		s.needsReindex = true
	}

	return nil
}

func (s *store) Remove(key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.indexInvalid {
		return errIndexInvalid
	}

	res, err := s.esClient.Delete(s.index, key, s.esClient.Delete.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil // Key not found, consider it as removed
	}

	if res.IsError() {
		return fmt.Errorf("error removing document: %s", res.String())
	}

	s.docCount--
	s.nextSeqNo++

	return nil
}

func indexExists(esClient *elasticsearch.Client, indexName string) (bool, error) {
	res, err := esClient.Indices.Exists([]string{indexName})
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

func createIndex(esClient *elasticsearch.Client, indexName string) error {
	res, err := esClient.Indices.Create(indexName)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error creating index: %s", res.String())
	}

	return nil
}

func getDocCount(esClient *elasticsearch.Client, indexName string) (int64, error) {
	res, err := esClient.Count(esClient.Count.WithIndex(indexName))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return 0, err
	}

	return int64(r["count"].(float64)), nil
}

func defaultReindexPredicate(docCount int64) bool {
	const limit = 1000000 // set reindex limit to 1 million documents by default
	return docCount >= limit
}

// Reindex method to be implemented
func (s *store) Reindex() error {
	// Implementation for reindexing
	// This should create a new index with updated mappings if necessary
	// and reindex all documents from the old index to the new one
	// Finally, it should update the alias to point to the new index
	return nil
}

func (s *store) Get(key string, to interface{}) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.indexInvalid {
		return errIndexInvalid
	}

	res, err := s.esClient.Get(s.index, key)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return errKeyUnknown
	}

	var source map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&source); err != nil {
		return err
	}

	sourceData, ok := source["_source"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected document structure")
	}

	// Convert the source data to JSON
	jsonData, err := json.Marshal(sourceData)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into the provided interface
	return json.Unmarshal(jsonData, to)
}

type esValueDecoder struct {
	value map[string]interface{}
}

func (d esValueDecoder) Decode(to interface{}) error {
	// Convert the value to JSON
	jsonData, err := json.Marshal(d.value)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into the provided interface
	return json.Unmarshal(jsonData, to)
}

// ... (keep all the other methods and functions)

func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.indexInvalid {
		return errIndexInvalid
	}

	var buf strings.Builder
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return err
	}

	res, err := s.esClient.Search(
		s.esClient.Search.WithIndex(s.index),
		s.esClient.Search.WithBody(strings.NewReader(buf.String())),
		s.esClient.Search.WithScroll(time.Minute),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	for {
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}

		hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
		for _, hit := range hits {
			h := hit.(map[string]interface{})
			key := h["_id"].(string)
			value := h["_source"].(map[string]interface{})

			cont, err := fn(key, esValueDecoder{value: value})
			if err != nil {
				return err
			}
			if !cont {
				return nil
			}
		}

		scrollID := r["_scroll_id"]
		if scrollID == nil {
			break
		}

		res, err = s.esClient.Scroll(s.esClient.Scroll.WithScrollID(scrollID.(string)), s.esClient.Scroll.WithScroll(time.Minute))
		if err != nil {
			return err
		}
	}

	return nil
}
