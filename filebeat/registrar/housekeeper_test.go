package registrar

import (
	"log"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/stretchr/testify/assert"
)

type eventSource struct {
	event       input.Event
	houseKeeper *HouseKeeper
	wg          sync.WaitGroup
	done        chan struct{}
}

func (s *eventSource) Start() {
	s.wg.Add(1)
	go s.Run()
}

func (s *eventSource) Run() {
	defer func() {
		s.wg.Done()
	}()
	for {
		select {
		case <-s.done:
			return
		default:
			s.event.State.TTL = 1 * time.Hour
			s.houseKeeper.Published([]*input.Event{&s.event})
		}
	}
}

func (s *eventSource) Stop() {
	close(s.done)
	s.wg.Wait()
}

func Test_HouseKeeper(t *testing.T) {
	expected := "/var/log/serviceA/b.log"

	states := []file.State{
		// active file
		{
			Source:   "/var/log/serviceA/a.log",
			TTL:      1 * time.Hour,
			Finished: false,
		},
		// timeout inactive file, not last file in dir
		// expected to be deleted
		{
			Source:   expected,
			TTL:      0,
			Finished: true,
		},
		// timeout inactive file, last file in dir
		{
			Source:   "/var/log/serviceB/stdout",
			TTL:      0,
			Finished: true,
		},
	}
	oldStates := file.States{}
	oldStates.SetStates(states)

	actual := []string{}
	rmFunc := func(name string) error {
		actual = append(actual, name)
		return nil
	}
	hk := NewHouseKeeper(&oldStates, 1, rmFunc)
	hk.Start()

	// start eventSource
	eSource := eventSource{
		event: input.Event{
			InputType: "log",
			State: file.State{
				Source: "/var/log/serviceA/a.log",
				TTL:    1 * time.Hour,
			},
		},
		houseKeeper: hk,
		done:        make(chan struct{}),
	}
	eSource.Start()
	log.Println("------94---", len(actual))

	time.Sleep(5 * time.Second)
	log.Println("------97---", len(actual))

	eSource.Stop()
	log.Println("------100---", len(actual))

	time.Sleep(5 * time.Second)

	hk.Stop()

	log.Println("------106---", len(actual))
	assert.Equal(t, 1, len(actual))
	assert.Equal(t, expected, actual[0])
}
