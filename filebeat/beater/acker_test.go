package beater

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/input/file"
)

type mockStatefulLogger struct {
	states []file.State
}

func (sf *mockStatefulLogger) Published(states []file.State) {
	sf.states = states
}

type mockStatelessLogger struct {
	count int
}

func (sl *mockStatelessLogger) Published(count int) bool {
	sl.count = count
	return true
}

func TestACKer(t *testing.T) {
	tests := []struct {
		name      string
		data      []interface{}
		stateless int
		stateful  []file.State
	}{
		{
			name:      "only stateless",
			data:      []interface{}{nil, nil},
			stateless: 2,
		},
		{
			name:      "only stateful",
			data:      []interface{}{file.State{Source: "-"}, file.State{Source: "-"}},
			stateful:  []file.State{file.State{Source: "-"}, file.State{Source: "-"}},
			stateless: 0,
		},
		{
			name:      "both",
			data:      []interface{}{file.State{Source: "-"}, nil, file.State{Source: "-"}},
			stateful:  []file.State{file.State{Source: "-"}, file.State{Source: "-"}},
			stateless: 1,
		},
		{
			name:      "any other Private type",
			data:      []interface{}{struct{}{}, nil},
			stateless: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sl := &mockStatelessLogger{}
			sf := &mockStatefulLogger{}

			h := newEventACKer(sl, sf)

			h.ackEvents(test.data)
			assert.Equal(t, test.stateless, sl.count)
			assert.Equal(t, test.stateful, sf.states)
		})
	}
}
