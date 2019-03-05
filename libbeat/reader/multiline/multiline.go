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

package multiline

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/reader"
	"github.com/elastic/beats/libbeat/reader/readfile"
)

// MultiLine reader combining multiple line events into one multi-line event.
//
// Lines to be combined are matched by some configurable predicate using
// regular expression.
//
// The maximum number of bytes and lines to be returned is fully configurable.
// Even if limits are reached subsequent lines are matched, until event is
// fully finished.
//
// Errors will force the multiline reader to return the currently active
// multiline event first and finally return the actual error on next call to Next.
type Reader struct {
	reader       reader.Reader
	pred         matcher
	flushMatcher *match.Matcher
	maxBytes     int // bytes stored in content
	maxLines     int
	separator    []byte
	last         []byte
	numLines     int
	truncated    int
	err          error // last seen error
	state        func(*Reader) (reader.Message, error)
	message      reader.Message
}

const (
	// Default maximum number of lines to return in one multi-line event
	defaultMaxLines = 500

	// Default timeout to finish a multi-line event.
	defaultMultilineTimeout = 5 * time.Second
)

// Matcher represents the predicate comparing any two lines
// to find start and end of multiline events in stream of line events.
type matcher func(last, current []byte) bool

var (
	sigMultilineTimeout = errors.New("multiline timeout")
)

// New creates a new multi-line reader combining stream of
// line events into stream of multi-line events.
func New(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (*Reader, error) {
	types := map[string]func(match.Matcher) (matcher, error){
		"before": beforeMatcher,
		"after":  afterMatcher,
	}

	matcherType, ok := types[config.Match]
	if !ok {
		return nil, fmt.Errorf("unknown matcher type: %s", config.Match)
	}

	matcher, err := matcherType(*config.Pattern)
	if err != nil {
		return nil, err
	}

	flushMatcher := config.FlushPattern

	if config.Negate {
		matcher = negatedMatcher(matcher)
	}

	maxLines := defaultMaxLines
	if config.MaxLines != nil {
		maxLines = *config.MaxLines
	}

	tout := defaultMultilineTimeout
	if config.Timeout != nil {
		tout = *config.Timeout
		if tout < 0 {
			return nil, fmt.Errorf("timeout %v must not be negative", config.Timeout)
		}
	}

	if tout > 0 {
		r = readfile.NewTimeoutReader(r, sigMultilineTimeout, tout)
	}

	mlr := &Reader{
		reader:       r,
		pred:         matcher,
		flushMatcher: flushMatcher,
		state:        (*Reader).readFirst,
		maxBytes:     maxBytes,
		maxLines:     maxLines,
		separator:    []byte(separator),
		message:      reader.Message{},
	}
	return mlr, nil
}

// Next returns next multi-line event.
func (mlr *Reader) Next() (reader.Message, error) {
	return mlr.state(mlr)
}

func (mlr *Reader) readFirst() (reader.Message, error) {
	for {
		message, err := mlr.reader.Next()
		if err != nil {
			// no lines buffered -> ignore timeout
			if err == sigMultilineTimeout {
				continue
			}

			logp.Debug("multiline", "Multiline event flushed because timeout reached.")

			// pass error to caller (next layer) for handling
			return message, err
		}

		if message.Bytes == 0 {
			continue
		}

		// Start new multiline event
		mlr.clear()
		mlr.load(message)
		mlr.setState((*Reader).readNext)
		return mlr.readNext()
	}
}

func (mlr *Reader) readNext() (reader.Message, error) {
	for {
		message, err := mlr.reader.Next()
		if err != nil {
			// handle multiline timeout signal
			if err == sigMultilineTimeout {
				// no lines buffered -> ignore timeout
				if mlr.numLines == 0 {
					continue
				}

				logp.Debug("multiline", "Multiline event flushed because timeout reached.")

				// return collected multiline event and
				// empty buffer for new multiline event
				msg := mlr.finalize()
				mlr.resetState()
				return msg, nil
			}

			// handle error without any bytes returned from reader
			if message.Bytes == 0 {
				// no lines buffered -> return error
				if mlr.numLines == 0 {
					return reader.Message{}, err
				}

				// lines buffered, return multiline and error on next read
				msg := mlr.finalize()
				mlr.err = err
				mlr.setState((*Reader).readFailed)
				return msg, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if mlr.message.Bytes == 0 || mlr.pred(mlr.last, message.Content) {
				mlr.addLine(message)

				// return multiline and error on next read
				msg := mlr.finalize()
				mlr.err = err
				mlr.setState((*Reader).readFailed)
				return msg, nil
			}

			// no match, return current multiline and retry with current line on next
			// call to readNext awaiting the error being reproduced (or resolved)
			// in next call to Next
			msg := mlr.finalize()
			mlr.load(message)
			return msg, nil
		}

		// handle case when endPattern is reached
		if mlr.flushMatcher != nil {
			endPatternReached := (mlr.flushMatcher.Match(message.Content))

			if endPatternReached == true {
				// return collected multiline event and
				// empty buffer for new multiline event
				mlr.addLine(message)
				msg := mlr.finalize()
				mlr.resetState()
				return msg, nil
			}
		}

		// if predicate does not match current multiline -> return multiline event
		if mlr.message.Bytes > 0 && !mlr.pred(mlr.last, message.Content) {
			msg := mlr.finalize()
			mlr.load(message)
			return msg, nil
		}

		// add line to current multiline event
		mlr.addLine(message)
	}
}

// readFailed returns empty message and error and resets line reader
func (mlr *Reader) readFailed() (reader.Message, error) {
	err := mlr.err
	mlr.err = nil
	mlr.resetState()
	return reader.Message{}, err
}

// load loads the reader with the given message. It is recommend to either
// run clear or finalize before.
func (mlr *Reader) load(m reader.Message) {
	mlr.addLine(m)
	// Timestamp of first message is taken as overall timestamp
	mlr.message.Ts = m.Ts
	mlr.message.AddFields(m.Fields)
}

// clearBuffer resets the reader buffer variables
func (mlr *Reader) clear() {
	mlr.message = reader.Message{}
	mlr.last = nil
	mlr.numLines = 0
	mlr.truncated = 0
	mlr.err = nil
}

// finalize writes the existing content into the returned message and resets all reader variables.
func (mlr *Reader) finalize() reader.Message {
	if mlr.truncated > 0 {
		mlr.message.AddFlagsWithKey("log.flags", "truncated")
	}

	if mlr.numLines > 1 {
		mlr.message.AddFlagsWithKey("log.flags", "multiline")
	}

	// Copy message from existing content
	msg := mlr.message

	mlr.clear()
	return msg
}

// addLine adds the read content to the message
// The content is only added if maxBytes and maxLines is not exceed. In case one of the
// two is exceeded, addLine keeps processing but does not add it to the content.
func (mlr *Reader) addLine(m reader.Message) {
	if m.Bytes <= 0 {
		return
	}

	sz := len(mlr.message.Content)
	addSeparator := len(mlr.message.Content) > 0 && len(mlr.separator) > 0
	if addSeparator {
		sz += len(mlr.separator)
	}

	space := mlr.maxBytes - sz

	maxBytesReached := (mlr.maxBytes <= 0 || space > 0)
	maxLinesReached := (mlr.maxLines <= 0 || mlr.numLines < mlr.maxLines)

	if maxBytesReached && maxLinesReached {
		if space < 0 || space > len(m.Content) {
			space = len(m.Content)
		}

		tmp := mlr.message.Content
		if addSeparator {
			tmp = append(tmp, mlr.separator...)
		}
		mlr.message.Content = append(tmp, m.Content[:space]...)
		mlr.numLines++

		// add number of truncated bytes to fields
		diff := len(m.Content) - space
		if diff > 0 {
			mlr.truncated += diff
		}
	} else {
		// increase the number of skipped bytes, if cannot add
		mlr.truncated += len(m.Content)

	}

	mlr.last = m.Content
	mlr.message.Bytes += m.Bytes
	mlr.message.AddFields(m.Fields)
}

// resetState sets state of the reader to readFirst
func (mlr *Reader) resetState() {
	mlr.setState((*Reader).readFirst)
}

// setState sets state to the given function
func (mlr *Reader) setState(next func(mlr *Reader) (reader.Message, error)) {
	mlr.state = next
}

// matchers

func afterMatcher(pat match.Matcher) (matcher, error) {
	return genPatternMatcher(pat, func(last, current []byte) []byte {
		return current
	})
}

func beforeMatcher(pat match.Matcher) (matcher, error) {
	return genPatternMatcher(pat, func(last, current []byte) []byte {
		return last
	})
}

func negatedMatcher(m matcher) matcher {
	return func(last, current []byte) bool {
		return !m(last, current)
	}
}

func genPatternMatcher(
	pat match.Matcher,
	sel func(last, current []byte) []byte,
) (matcher, error) {
	matcher := func(last, current []byte) bool {
		line := sel(last, current)
		return pat.Match(line)
	}
	return matcher, nil
}
