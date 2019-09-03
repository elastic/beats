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
	ts    time.Time
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
			// Try later when we have more data
			p.buf.Restore(bufBeforeStart)
			return nil
		}
		debugf("Processing: %s", raw)

		nbytes := len(raw)
		jsize := len(s.journal)
		jend := jsize - 1

		isCode := nbytes >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])
		isRequ := nbytes < constMinRespSize || !isCode

		// See if can establish sync. Things to look for:
		//  - Multiline requests: it's a DATA payload transmission
		//  - Request followed by a response: parsers are in command state
		//
		// First line of the capture can be incomplete, so we don't use it
		// to establish sync
		if jsize > 0 && isRequ {
			s.requSeen, s.requDir = true, dir
		}
		if s.firstMessageLines > 1 || jsize > 1 {
			if isRequ {
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
			s.journal = append(s.journal, jentry{ts: ts, dir: dir})
		}

		s.nbytesSeen[dir] += nbytes
		if jsize == 0 || jsize == 1 && s.journal[jend].dir == dir {
			s.firstMessageLines++
		}
	}

	p.buf.Restore(bufBeforeStart)

	if s.done {
		debugf("Sync established, journal size %d", len(s.journal))

		// The leading entries, if any, can't be stateData or it would
		// have caused synchronization to happen earlier
		for i := 0; i < len(s.journal)-2; i++ {
			s.journal[i].state = stateCommand
		}

		// First line of the capture can be mangled. Since SMTP
		// sessions start with a (prompt) response, a non-response
		// line means an incomplete session anyway, so we skip it.
		dir0 := s.journal[0].dir
		p := s.parsers[dir0]
		bufBeforeStart := p.buf.Snapshot()
		raw, _ := p.buf.CollectUntil(constCRLF)
		isCode := len(raw) >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])
		if isCode && s.journal[0].dir != s.requDir {
			// All good
			p.buf.Restore(bufBeforeStart)
		} else {
			// Not a response; we got rid of the potentially mangled
			// line; is there anything left of the message?
			if s.firstMessageLines == 1 {
				s.journal = s.journal[1:]
			}
		}

		// Set main parsers' initial states from journal
		dir0 = s.journal[0].dir
		s.parsers[dir0].state = s.journal[0].state
		dir1 := dir0 ^ 1
		for _, je := range s.journal {
			if je.dir == dir1 {
				s.parsers[dir1].state = je.state
				break
			}
		}

		// Call main parsers in the order the accumulated messages
		// came in
		for _, je := range s.journal {
			err := s.parsers[je.dir].process(je.ts, je.dir, true)
			if err != nil {
				return err
			}
		}
	} else {
		debugf("No sync, journal size %d", len(s.journal))
	}
	return nil
}
