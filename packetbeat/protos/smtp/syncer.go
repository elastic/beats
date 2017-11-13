package smtp

import "time"

type syncer struct {
	done              bool
	requSeen          bool
	requDir           uint8
	parsers           [2]*parser
	nbytesSeen        [2]int
	firstMessageLines int
	journal           []jentry
}

type jentry struct {
	dir   uint8
	state parseState
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

	for p.buf.Len() > 0 && !s.done {
		raw, err := p.buf.CollectUntil(constCRLF)
		if err != nil {
			// Get more data
			return nil
		}

		nbytes := len(raw)
		jsize := len(s.journal)
		jend := jsize - 1

		isCode := nbytes >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])

		// See if can establish sync. Things to look for:
		//  - Multiline requests: it's a DATA payload transmission
		//  - Request followed by a response: parsers are in command state
		//
		// First line of the capture can be incomplete, so we don't use it
		// to establish sync
		if jsize > 0 && (s.firstMessageLines > 1 || jsize > 1) {
			if nbytes < constMinRespSize || !isCode {
				s.requSeen, s.requDir = true, dir
				if s.journal[jend].dir == dir {
					// Multiline requests can only be a DATA payload
					// transmission
					s.journal[jend].state = stateData
					// This one is part of the data request too; sync
					// established
					s.done = true
				}
			} else {
				// Possible response, but can be a DATA payload line that
				// has bytes looking like a response code
				if s.requSeen && dir != s.requDir {
					// Definitely a response; sync established
					s.done = true
				}
			}
		}

		// Multiline messages only get one journal entry
		if jsize == 0 || s.journal[jend].dir != dir {
			s.journal = append(s.journal, jentry{dir: dir})
		}

		s.nbytesSeen[dir] += nbytes
		if jsize == 0 {
			s.firstMessageLines++
		}
	}

	p.buf.Restore(bufBeforeStart)

	if s.done {
		// The leading entries, if any, can't be stateData or it would
		// have caused synchronization to happen earlier
		for i := 0; i < len(s.journal)-1; i++ {
			s.journal[i].state = stateCommand
		}

		// Set main parsers' initial states from journal
		dir0 := s.journal[0].dir
		s.parsers[dir0].state = s.journal[0].state
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

		// First line of the capture can be mangled. Since SMTP
		// sessions start with a (prompt) response, a non-response
		// line means an incomplete session anyway, so we skip it.
		p := s.parsers[dir0]
		bufBeforeStart := p.buf.Snapshot()
		raw, _ := p.buf.CollectUntil(constCRLF)
		isCode := len(raw) >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])
		if isCode && len(s.journal) > 1 && s.journal[1].dir == s.requDir {
			// All good
			p.buf.Restore(bufBeforeStart)
		} else {
			// Not a response; we got rid of the potentially mangled
			// line; is there anything left of the message?
			if s.firstMessageLines == 1 {
				s.journal = s.journal[1:]
			}
		}

		// Call main parsers in the order the accumulated messages
		// came in
		for _, je := range s.journal {
			err := s.parsers[je.dir].process(time.Now(), true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
