package smtp

import (
	"bytes"
	"time"
)

type syncer struct {
	done       bool
	parsers    [2]*parser
	nbytesSeen [2]int
	journal    []jentry
}

type jentry struct {
	dir    uint8
	isRequ bool
	state  parseState
}

// syncer.process() tries to sync by correlating the new payload
// against the accumulated sequence of payload data from both streams.
// If successful, it replays for the main parser the accumulated
// payloads and hands control over to the main parser. In case of
// failure to establish sync, it restores the buffers to the original
// state and returns, to be called again when a new payload becomes
// available.
func (s *syncer) process(ts time.Time, dir uint8) error {
	p := s.parsers[dir]
	bufBeforeStart := p.buf.Snapshot()
	// Skip to new payload
	if err := p.buf.Advance(s.nbytesSeen[dir]); err != nil {
		return err
	}

	var err error

	raw, err := p.buf.CollectUntil(constCRLF)
	if err != nil {
		// Get more data
		return nil
	}

	nbytes := len(raw)
	jsize := len(s.journal)
	jend := jsize - 1
	isRequ := false

	isCode := nbytes >= constMinRespSize &&
		isResponseCode(raw[:constMinRespSize])

	// See if can establish sync. Things to look for:
	//  - ".\r\n": it's the end of DATA payload transmission
	//  - Multiline requests: it's a DATA payload transmission
	//  - Request followed by a response: parsers are in command state
	if jsize == 0 && nbytes == constEODSize && bytes.Compare(raw, constEOD) == 0 {
		s.parsers[dir].state = stateData
		s.done = true
	}
	if jsize > 0 {
		if nbytes < constMinRespSize || !isCode {
			isRequ = true
			if s.journal[jend].dir == dir {
				// Multiline requests can only be a DATA payload
				// transmission
				s.journal[jend].state = stateData
				// This one is a data request too; sync established
				s.done = true
			}
		} else {
			// Possible response, but can be a DATA payload line that
			// has bytes looking like a response code
			if s.journal[jend].isRequ && s.journal[jend].dir != dir {
				// Definitely a response; sync established
				s.done = true
			}
		}
	}

	// Multiline messages only get one journal entry
	if jsize == 0 || s.journal[jend].dir != dir {
		s.journal = append(s.journal, jentry{dir: dir, isRequ: isRequ})
	}

	s.nbytesSeen[dir] += nbytes
	p.buf.Restore(bufBeforeStart)

	if s.done {
		// The leading entries, if any, can't be stateData or it would
		// have caused synchronization to happen earlier
		for i := 0; i < jend; i++ {
			s.journal[i].state = stateCommand
		}

		// Set main parsers' initial states from journal, unless
		// already initialized above
		dir0 := s.journal[0].dir
		if s.parsers[dir0].state == stateUnsynced {
			s.parsers[dir0].state = s.journal[0].state
		}
		dir1 := dir0 ^ 1
		for _, je := range s.journal {
			if je.dir == dir1 {
				s.parsers[dir1].state = je.state
				break
			}
		}
		if s.parsers[dir1].state == stateUnsynced {
			// First message of this stream is a response, syncer got
			// done before getting to it, so it wasn't logged
			s.parsers[dir1].state = stateCommand
		}

		// Call main parsers in the order the accumulated messages
		// came in
		for _, je := range s.journal {
			err = s.parsers[je.dir].process(time.Now())
			if err != nil {
				return err
			}
		}
	}

	return nil
}
