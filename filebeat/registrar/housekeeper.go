package registrar

import (
	"strings"
	"sync"
	"time"

	input "github.com/elastic/beats/filebeat/input"

	"github.com/elastic/beats/filebeat/input/file"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/logp"
)

type HouseKeeper struct {
	in              chan []*input.Event
	done            chan struct{}
	states          *file.States
	wg              sync.WaitGroup
	cleanupInterval int64
	remove          func(name string) error
}

func NewHouseKeeper(states *file.States, cleanupInterval int64, rmFunc func(name string) error) *HouseKeeper {
	return &HouseKeeper{
		in:              make(chan []*input.Event, 10),
		done:            make(chan struct{}),
		states:          states,
		cleanupInterval: cleanupInterval,
		remove:          rmFunc,
	}
}

func (h *HouseKeeper) Published(events []*input.Event) bool {
	h.in <- events
	return true
}

func (h *HouseKeeper) Start() {
	h.wg.Add(1)
	go h.Run()
}

func (h *HouseKeeper) Run() {
	logp.Info("47 Starting HouseKeeper")

	defer func() {
		h.Cleanup()
		h.wg.Done()
	}()

	last := time.Now().Unix()
	for {
		var events []*input.Event

		select {
		case <-h.done:
			logp.Info("Ending HouseKeeper")
			return
		case events = <-h.in:
		}

		for _, event := range events {
			if event.InputType == cfg.StdinInputType {
				continue
			}
			h.states.Update(event.State)
		}

		current := time.Now().Unix()
		if current-last >= h.cleanupInterval {
			h.Cleanup()
			last = current
		}
	}
}

func (h *HouseKeeper) Cleanup() {
	logp.Debug("TRACE", "housekeeper cleanup inactive files")
	states := h.states.GetStates()

	// key: dirname, value: file states
	dirs := make(map[string]*DirFilesState)

	for _, state := range states {
		key := dirname(state.Source)
		if dirs[key] == nil {
			dirs[key] = &DirFilesState{states: []file.State{}}
		}
		dirs[key].Add(state)
	}

	timeoutStates := []file.State{}
	h.states.CleanupWithFunc(func(state file.State) {
		logp.Info("file[%+v] inactive", state.Source)
		timeoutStates = append(timeoutStates, state)
	})

	count := 0
	// TODO: for stderr
	// TODO: for stdout
	// TODO: remove file size > xxx
	for _, state := range timeoutStates {
		if strings.HasSuffix(state.Source, "stderr") || strings.HasSuffix(state.Source, "stdout") {
			logp.Info("ignore last inactive file %s", state.Source)
			continue
		}
		dir := dirname(state.Source)
		if dirs[dir].Len() > 1 {
			logp.Info("remove inactive file %s", state.Source)
			err := h.remove(state.Source)
			if err != nil {
				logp.Err("remove file failed, err[%s]", err)
			}
			dirs[dir].Remove(state)
			count++
		} else {
			logp.Info("last inactive log file[%s], will not delete", state.Source)
		}
	}
	logp.Debug("TRACE", "housekeeper cleanup %d inactive files", count)
}

func (h *HouseKeeper) Stop() {
	logp.Info("Stopping HouseKeeper")
	close(h.done)
	h.wg.Wait()
	close(h.in)
}
