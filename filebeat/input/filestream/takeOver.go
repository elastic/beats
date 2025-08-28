package filestream

import (
	"fmt"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
)

type PocStore interface {
	BulkInsert([]loginp.LogInputState, string) error
}

func TakeOverFromEA(
	logger *logp.Logger,
	oldRegistry string,
	dstStore PocStore,
	newID func(loginp.Source) string,
	files map[string]loginp.FileDescriptor,
	identifier fileIdentifier,
) error {

	logger = logger.Named("take-over")
	store, err := memlog.OpenStore(
		logger,
		oldRegistry,
		384, // magic number, copied from original use of OpenStore
		4096,
		false,
		func(uint64) bool { return false }, //never create a checkpoint
	)
	if err != nil {
		return fmt.Errorf("cannot open store at '%s': %w", oldRegistry, err)
	}

	states := []loginp.LogInputState{}
	// func must return true to keep iterating, on any error the iteration stops
	fn := func(k string, v backend.ValueDecoder) (bool, error) {
		m := map[string]any{}
		if err := v.Decode(&m); err != nil {
			logger.Errorf("could not decode '%s': %s", k, err)
			return true, nil
		}

		st, err := loginp.LogInputStateFromMapM(m)
		if err != nil {
			logger.Errorf("could not convert state from '%s': %s", k, err)
		}

		fd, exists := files[st.Source]
		if !exists {
			// if this file is not in the file system, skip its state migration
			return true, nil
		}

		st.NewKey = newID(identifier.GetSource(loginp.FSEvent{NewPath: st.Source, Descriptor: fd}))
		states = append(states, st)
		logger.Infof("Key: '%s', Value: '%#v'", k, st)

		return true, nil
	}

	if err := store.Each(fn); err != nil {
		logger.Errorf("could not run store.Each: %s", err)
	}

	dstStore.BulkInsert(states, identifier.Name())
	return nil
}
