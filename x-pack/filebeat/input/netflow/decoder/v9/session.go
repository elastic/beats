// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/atomic"
	"github.com/elastic/beats/x-pack/filebeat/input/netflow/decoder/template"
)

type SessionKey string

func MakeSessionKey(addr net.Addr) SessionKey {
	return SessionKey(addr.String())
}

type TemplateKey struct {
	SourceID   uint32
	TemplateID uint16
}

type TemplateWrapper struct {
	Template *template.Template
	Delete   atomic.Bool
}

type SessionState struct {
	mutex        sync.RWMutex
	Templates    map[TemplateKey]*TemplateWrapper
	lastSequence uint32
	logger       *log.Logger
	Delete       atomic.Bool
}

func NewSession(logger *log.Logger) *SessionState {
	return &SessionState{
		logger:    logger,
		Templates: make(map[TemplateKey]*TemplateWrapper),
	}
}

func (s *SessionState) AddTemplate(sourceId uint32, t *template.Template) {
	key := TemplateKey{sourceId, t.ID}
	s.logger.Printf("state %p addTemplate %v %p", s, key, t)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Templates[key] = &TemplateWrapper{Template: t}
}

func (s *SessionState) GetTemplate(sourceId uint32, id uint16) (template *template.Template) {
	key := TemplateKey{sourceId, id}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	wrapper, found := s.Templates[key]
	if found {
		template = wrapper.Template
		wrapper.Delete.Store(false)
	}
	return template
}

func (s *SessionState) ExpireTemplates() (alive int, removed int) {
	var toDelete []TemplateKey
	s.mutex.RLock()
	for id, template := range s.Templates {
		if !template.Delete.CAS(false, true) {
			toDelete = append(toDelete, id)
		}
	}
	total := len(s.Templates)
	s.mutex.RUnlock()
	if len(toDelete) > 0 {
		s.mutex.Lock()
		total = len(s.Templates)
		for _, id := range toDelete {
			if template, found := s.Templates[id]; found && template.Delete.Load() {
				s.logger.Printf("expired template %v", id)
				delete(s.Templates, id)
				removed++
			}
		}
		s.mutex.Unlock()
	}
	return total - removed, removed
}

func (s *SessionState) CheckReset(seqNum uint32) (reset bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if reset = seqNum < s.lastSequence && seqNum-s.lastSequence > MaxSequenceDifference; reset {
		s.Templates = make(map[TemplateKey]*TemplateWrapper)
	}
	s.lastSequence = seqNum
	return
}

type SessionMap struct {
	mutex    sync.RWMutex
	Sessions map[SessionKey]*SessionState
	logger   *log.Logger
}

func NewSessionMap(logger *log.Logger) SessionMap {
	return SessionMap{
		logger:   logger,
		Sessions: make(map[SessionKey]*SessionState),
	}
}

func (m *SessionMap) GetOrCreate(key SessionKey) *SessionState {
	m.mutex.RLock()
	session, found := m.Sessions[key]
	if found {
		session.Delete.Store(false)
	}
	m.mutex.RUnlock()
	if !found {
		m.mutex.Lock()
		if session, found = m.Sessions[key]; !found {
			session = NewSession(m.logger)
			m.Sessions[key] = session
		}
		m.mutex.Unlock()
	}
	return session
}

func (m *SessionMap) cleanup() (aliveSession int, removedSession int, aliveTemplates int, removedTemplates int) {
	var toDelete []SessionKey
	m.mutex.RLock()
	total := len(m.Sessions)
	for key, session := range m.Sessions {
		a, r := session.ExpireTemplates()
		aliveTemplates += a
		removedTemplates += r
		if !session.Delete.CAS(false, true) {
			toDelete = append(toDelete, key)
		}
	}
	m.mutex.RUnlock()
	if len(toDelete) > 0 {
		m.mutex.Lock()
		total = len(m.Sessions)
		for _, key := range toDelete {
			if session, found := m.Sessions[key]; found && session.Delete.Load() {
				delete(m.Sessions, key)
				removedSession++
			}
		}
		m.mutex.Unlock()
	}
	return total - removedSession, removedSession, aliveTemplates, removedTemplates
}

func (m *SessionMap) CleanupLoop(interval time.Duration, done <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-done:
			return

		case <-t.C:
			aliveS, removedS, aliveT, removedT := m.cleanup()
			if removedS > 0 || removedT > 0 {
				m.logger.Printf("Expired %d sessions (%d remain) / %d templates (%d remain)", removedS, aliveS, removedT, aliveT)
			}
		}
	}
}
