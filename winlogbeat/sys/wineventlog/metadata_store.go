// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build windows

package wineventlog

import (
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/sys"
)

var (
	// eventDataNameTransform removes spaces from parameter names.
	eventDataNameTransform = strings.NewReplacer(" ", "_")

	// eventMessageTemplateFuncs contains functions for use in message templates.
	eventMessageTemplateFuncs = template.FuncMap{
		"eventParam": eventParam,
	}
)

// publisherMetadataStore stores metadata from a publisher.
type publisherMetadataStore struct {
	Metadata *PublisherMetadata // Handle to the publisher metadata. May be nil.
	Keywords map[int64]string   // Keyword bit mask to keyword name.
	Opcodes  map[uint8]string   // Opcode value to name.
	Levels   map[uint8]string   // Level value to name.
	Tasks    map[uint16]string  // Task value to name.

	// Event ID to event metadata (message and event data param names).
	Events map[uint16]*eventMetadata
	// Event ID to map of fingerprints to event metadata. The fingerprint value
	// is hash of the event data parameters count and types.
	EventFingerprints map[uint16]map[uint64]*eventMetadata

	mutex sync.RWMutex
	log   *logp.Logger
}

func newPublisherMetadataStore(session EvtHandle, provider string, log *logp.Logger) (*publisherMetadataStore, error) {
	md, err := NewPublisherMetadata(session, provider)
	if err != nil {
		return nil, err
	}
	store := &publisherMetadataStore{
		Metadata:          md,
		EventFingerprints: map[uint16]map[uint64]*eventMetadata{},
		log:               log.With("publisher", provider),
	}

	// Query the provider metadata to build an in-memory cache of the
	// information to optimize event reading.
	err = multierr.Combine(
		store.initKeywords(),
		store.initOpcodes(),
		store.initLevels(),
		store.initTasks(),
		store.initEvents(),
	)
	if err != nil {
		return nil, err
	}

	return store, nil
}

// newEmptyPublisherMetadataStore creates an empty metadata store for cases
// where no local publisher metadata exists.
func newEmptyPublisherMetadataStore(provider string, log *logp.Logger) *publisherMetadataStore {
	return &publisherMetadataStore{
		Keywords:          map[int64]string{},
		Opcodes:           map[uint8]string{},
		Levels:            map[uint8]string{},
		Tasks:             map[uint16]string{},
		Events:            map[uint16]*eventMetadata{},
		EventFingerprints: map[uint16]map[uint64]*eventMetadata{},
		log:               log.With("publisher", provider, "empty", true),
	}
}

func (s *publisherMetadataStore) initKeywords() error {
	keywords, err := s.Metadata.Keywords()
	if err != nil {
		return err
	}

	s.Keywords = make(map[int64]string, len(keywords))
	for _, keywordMeta := range keywords {
		val := keywordMeta.Name
		if val == "" {
			val = keywordMeta.Message
		}
		s.Keywords[int64(keywordMeta.Mask)] = val
	}
	return nil
}

func (s *publisherMetadataStore) initOpcodes() error {
	opcodes, err := s.Metadata.Opcodes()
	if err != nil {
		return err
	}
	s.Opcodes = make(map[uint8]string, len(opcodes))
	for _, opcodeMeta := range opcodes {
		val := opcodeMeta.Message
		if val == "" {
			val = opcodeMeta.Name
		}
		s.Opcodes[uint8(opcodeMeta.Mask)] = val
	}
	return nil
}

func (s *publisherMetadataStore) initLevels() error {
	levels, err := s.Metadata.Levels()
	if err != nil {
		return err
	}

	s.Levels = make(map[uint8]string, len(levels))
	for _, levelMeta := range levels {
		val := levelMeta.Name
		if val == "" {
			val = levelMeta.Message
		}
		s.Levels[uint8(levelMeta.Mask)] = val
	}
	return nil
}

func (s *publisherMetadataStore) initTasks() error {
	tasks, err := s.Metadata.Tasks()
	if err != nil {
		return err
	}
	s.Tasks = make(map[uint16]string, len(tasks))
	for _, taskMeta := range tasks {
		val := taskMeta.Message
		if val == "" {
			val = taskMeta.Name
		}
		s.Tasks[uint16(taskMeta.Mask)] = val
	}
	return nil
}

func (s *publisherMetadataStore) initEvents() error {
	itr, err := s.Metadata.EventMetadataIterator()
	if err != nil {
		return err
	}
	defer itr.Close()

	s.Events = map[uint16]*eventMetadata{}
	for itr.Next() {
		evt, err := newEventMetadataFromPublisherMetadata(itr, s.Metadata)
		if err != nil {
			s.log.Warnw("Failed to read event metadata from publisher. Continuing to next event.",
				"error", err)
			continue
		}
		s.Events[evt.EventID] = evt
	}
	return itr.Err()
}

func (s *publisherMetadataStore) getEventMetadata(eventID uint16, eventDataFingerprint uint64, eventHandle EvtHandle) *eventMetadata {
	// Use a read lock to get a cached value.
	s.mutex.RLock()
	fingerprints, found := s.EventFingerprints[eventID]
	if found {
		em, found := fingerprints[eventDataFingerprint]
		if found {
			s.mutex.RUnlock()
			return em
		}
	}

	// Elevate to write lock.
	s.mutex.RUnlock()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fingerprints, found = s.EventFingerprints[eventID]
	if !found {
		fingerprints = map[uint64]*eventMetadata{}
		s.EventFingerprints[eventID] = fingerprints
	}

	em, found := fingerprints[eventDataFingerprint]
	if found {
		return em
	}

	// To ensure we always match the correct event data parameter names to
	// values we will rely a fingerprint made of the number of event data
	// properties and each of their EvtVariant type values.
	//
	// The first time we observe a new fingerprint value we get the XML
	// representation of the event in order to know the parameter names.
	// If they turn out to match the values that we got from the provider's
	// metadata then we just associate the fingerprint with a pointer to the
	// providers metadata for the event ID.

	defaultEM := s.Events[eventID]

	// Use XML to get the parameters names.
	em, err := newEventMetadataFromEventHandle(s.Metadata, eventHandle)
	if err != nil {
		s.log.Debugw("Failed to make event metadata from event handle. Will "+
			"use default event metadata from the publisher.",
			"event_id", eventID,
			"fingerprint", eventDataFingerprint,
			"error", err)

		if defaultEM != nil {
			fingerprints[eventDataFingerprint] = defaultEM
		}
		return defaultEM
	}

	// Are the parameters the same as what the provider metadata listed?
	// (This ignores the message values.)
	if em.equal(defaultEM) {
		fingerprints[eventDataFingerprint] = defaultEM
		return defaultEM
	}

	// If we couldn't get a message from the event handle use the one
	// from the installed provider metadata.
	if defaultEM != nil && em.MsgStatic == "" && em.MsgTemplate == nil {
		em.MsgStatic = defaultEM.MsgStatic
		em.MsgTemplate = defaultEM.MsgTemplate
	}

	s.log.Debugw("Obtained unique event metadata from event handle. "+
		"It differed from what was listed in the publisher's metadata.",
		"event_id", eventID,
		"fingerprint", eventDataFingerprint,
		"default_event_metadata", defaultEM,
		"event_metadata", em)

	fingerprints[eventDataFingerprint] = em
	return em
}

func (s *publisherMetadataStore) Close() error {
	if s.Metadata != nil {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		return s.Metadata.Close()
	}
	return nil
}

type eventMetadata struct {
	EventID     uint16             // Event ID.
	Version     uint8              // Event format version.
	MsgStatic   string             // Used when the message has no parameters.
	MsgTemplate *template.Template `json:"-"` // Template that expects an array of values as its data.
	EventData   []eventData        // Names of parameters from XML template.
}

// newEventMetadataFromEventHandle collects metadata about an event type using
// the handle of an event.
func newEventMetadataFromEventHandle(publisher *PublisherMetadata, eventHandle EvtHandle) (*eventMetadata, error) {
	xml, err := getEventXML(publisher, eventHandle)
	if err != nil {
		return nil, err
	}

	// By parsing the XML we can get the names of the parameters even if the
	// publisher metadata is unavailable or is out of sync with the events.
	event, err := sys.UnmarshalEventXML([]byte(xml))
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal XML")
	}

	em := &eventMetadata{
		EventID: uint16(event.EventIdentifier.ID),
		Version: uint8(event.Version),
	}
	if len(event.EventData.Pairs) > 0 {
		for _, pair := range event.EventData.Pairs {
			em.EventData = append(em.EventData, eventData{Name: pair.Key})
		}
	} else {
		for _, pair := range event.UserData.Pairs {
			em.EventData = append(em.EventData, eventData{Name: pair.Key})
		}
	}

	// The message template is only available from the publisher metadata. This
	// message template may not match up with the event data we got from the
	// event's XML, but it's the only option available. Even forwarded events
	// with "RenderedText" won't help because their messages are already
	// rendered.
	if publisher != nil {
		msg, err := getMessageStringFromHandle(publisher, eventHandle, templateInserts.Slice())
		if err != nil {
			return nil, err
		}
		if err = em.setMessage(msg); err != nil {
			return nil, err
		}
	}

	return em, nil
}

// newEventMetadataFromPublisherMetadata collects metadata about an event type
// using the publisher metadata.
func newEventMetadataFromPublisherMetadata(itr *EventMetadataIterator, publisher *PublisherMetadata) (*eventMetadata, error) {
	em := &eventMetadata{}
	err := multierr.Combine(
		em.initEventID(itr),
		em.initVersion(itr),
		em.initEventDataTemplate(itr),
		em.initEventMessage(itr, publisher),
	)
	if err != nil {
		return nil, err
	}
	return em, nil
}

func (em *eventMetadata) initEventID(itr *EventMetadataIterator) error {
	id, err := itr.EventID()
	if err != nil {
		return err
	}
	// The upper 16 bits are the qualifier and lower 16 are the ID.
	em.EventID = uint16(0xFFFF & id)
	return nil
}

func (em *eventMetadata) initVersion(itr *EventMetadataIterator) error {
	version, err := itr.Version()
	if err != nil {
		return err
	}
	em.Version = uint8(version)
	return nil
}

func (em *eventMetadata) initEventDataTemplate(itr *EventMetadataIterator) error {
	xml, err := itr.Template()
	if err != nil {
		return err
	}
	// Some events do not have templates.
	if xml == "" {
		return nil
	}

	tmpl := &eventTemplate{}
	if err = tmpl.Unmarshal([]byte(xml)); err != nil {
		return err
	}

	for _, kv := range tmpl.Data {
		kv.Name = eventDataNameTransform.Replace(kv.Name)
	}

	em.EventData = tmpl.Data
	return nil
}

func (em *eventMetadata) initEventMessage(itr *EventMetadataIterator, publisher *PublisherMetadata) error {
	messageID, err := itr.MessageID()
	if err != nil {
		return err
	}

	msg, err := getMessageString(publisher, NilHandle, messageID, templateInserts.Slice())
	if err != nil {
		return err
	}

	return em.setMessage(msg)
}

func (em *eventMetadata) setMessage(msg string) error {
	msg = sys.RemoveWindowsLineEndings(msg)
	tmplID := strconv.Itoa(int(em.EventID))

	tmpl, err := template.New(tmplID).Funcs(eventMessageTemplateFuncs).Parse(msg)
	if err != nil {
		return err
	}

	// One node means there were no parameters so this will optimize that case
	// by using a static string rather than a text/template.
	if len(tmpl.Root.Nodes) == 1 {
		em.MsgStatic = msg
	} else {
		em.MsgTemplate = tmpl
	}
	return nil
}

func (em *eventMetadata) equal(other *eventMetadata) bool {
	if em == other {
		return true
	}
	if em == nil || other == nil {
		return false
	}

	eventDataNamesEqual := func(a, b []eventData) bool {
		if len(a) != len(b) {
			return false
		}
		for n, v := range a {
			if v.Name != b[n].Name {
				return false
			}
		}
		return true
	}

	return em.EventID == other.EventID &&
		em.Version == other.Version &&
		eventDataNamesEqual(em.EventData, other.EventData)
}

// --- Template Funcs

// eventParam return an event data value inside a text/template.
func eventParam(items []interface{}, paramNumber int) (interface{}, error) {
	// Windows parameter values start at %1 so adjust index value by -1.
	index := paramNumber - 1
	if index < len(items) {
		return items[index], nil
	}
	// Windows Event Viewer leaves the original placeholder (e.g. %22) in the
	// rendered message when no value provided.
	return "%" + strconv.Itoa(paramNumber), nil
}
