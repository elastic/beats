package elasticsearch

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/monitoring"
)

type snapshotVisitor struct {
	key   keyStack
	event eventStack
	depth int
}

type keyStack struct {
	current string
	stack   []string
	stack0  [32]string
}

type eventStack struct {
	current common.MapStr
	stack   []common.MapStr
	stack0  [32]common.MapStr
}

func makeSnapshot(R *monitoring.Registry) common.MapStr {
	vs := snapshotVisitor{}
	vs.key.stack = vs.key.stack0[:0]
	vs.event.stack = vs.event.stack0[:0]

	if R == nil {
		R = monitoring.Default
	}

	R.Visit(monitoring.Reported, &vs)
	return vs.event.current
}

func (s *snapshotVisitor) OnRegistryStart() {
	if s.depth > 0 {
		s.event.push()
	}
	s.depth++
}

func (s *snapshotVisitor) OnRegistryFinished() {
	s.depth--
	if s.depth == 0 {
		return
	}

	event := s.event.pop()
	if event == nil {
		s.key.pop()
		return
	}

	s.setValue(event)
}

func (s *snapshotVisitor) OnKey(key string) {
	s.key.push(key)
}

func (s *snapshotVisitor) OnString(str string) { s.setValue(str) }
func (s *snapshotVisitor) OnBool(b bool)       { s.setValue(b) }
func (s *snapshotVisitor) OnInt(i int64)       { s.setValue(i) }
func (s *snapshotVisitor) OnFloat(f float64)   { s.setValue(f) }

func (s *snapshotVisitor) setValue(v interface{}) {
	if s.event.current == nil {
		s.event.current = common.MapStr{}
	}

	s.event.current[s.key.current] = v
	s.key.pop()
}

func (s *keyStack) push(key string) {
	s.stack = append(s.stack, s.current)
	s.current = key
}

func (s *keyStack) pop() {
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
}

func (s *eventStack) push() {
	s.stack = append(s.stack, s.current)
	s.current = nil
}

func (s *eventStack) pop() common.MapStr {
	event := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return event
}
