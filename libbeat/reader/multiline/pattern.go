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
	"io"
	"time"

	"github.com/elastic/beats/v8/libbeat/common/match"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/libbeat/reader"
	"github.com/elastic/beats/v8/libbeat/reader/readfile"
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
type patternReader struct {
	reader       reader.Reader
	pred         matcher
	flushMatcher *match.Matcher
	state        func(*patternReader) (reader.Message, error)
	logger       *logp.Logger
	msgBuffer    *messageBuffer
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

func newMultilinePatternReader(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (reader.Reader, error) {

	matcher, err := setupPatternMatcher(config)
	if err != nil {
		return nil, err
	}

	maxLines := defaultMaxLines
	if config.MaxLines != nil {
		maxLines = *config.MaxLines
	}

	tout := defaultMultilineTimeout
	if config.Timeout != nil {
		tout = *config.Timeout
	}

	if tout > 0 {
		r = readfile.NewTimeoutReader(r, sigMultilineTimeout, tout)
	}

	pr := &patternReader{
		reader:       r,
		pred:         matcher,
		flushMatcher: config.FlushPattern,
		state:        (*patternReader).readFirst,
		msgBuffer:    newMessageBuffer(maxBytes, maxLines, []byte(separator), config.SkipNewLine),
		logger:       logp.NewLogger("reader_multiline"),
	}
	return pr, nil
}

func setupPatternMatcher(config *Config) (matcher, error) {
	types := map[string]func(match.Matcher) (matcher, error){
		"before": beforeMatcher,
		"after":  afterMatcher,
	}

	matcherType, ok := types[config.Match]
	if !ok {
		return nil, fmt.Errorf("unknown matcher type: %s", config.Match)
	}

	m, err := matcherType(*config.Pattern)
	if err != nil {
		return nil, err
	}

	if config.Negate {
		m = negatedMatcher(m)
	}

	return m, nil
}

// Next returns next multi-line event.
func (pr *patternReader) Next() (reader.Message, error) {
	return pr.state(pr)
}

func (pr *patternReader) readFirst() (reader.Message, error) {
	for {
		message, err := pr.reader.Next()
		if err != nil {
			// no lines buffered -> ignore timeout
			if err == sigMultilineTimeout {
				continue
			}

			pr.logger.Debug("Multiline event flushed because timeout reached.")

			// pass error to caller (next layer) for handling
			return message, err
		}

		if message.Bytes == 0 {
			continue
		}

		// Start new multiline event
		pr.msgBuffer.startNewMessage(message)
		pr.setState((*patternReader).readNext)
		return pr.readNext()
	}
}

func (pr *patternReader) readNext() (reader.Message, error) {
	for {
		message, err := pr.reader.Next()
		if err != nil {
			// handle multiline timeout signal
			if err == sigMultilineTimeout {
				// no lines buffered -> ignore timeout
				if pr.msgBuffer.isEmpty() {
					continue
				}

				pr.logger.Debug("Multiline event flushed because timeout reached.")

				// return collected multiline event and
				// empty buffer for new multiline event
				msg := pr.msgBuffer.finalize()
				pr.resetState()
				return msg, nil
			}

			// handle error without any bytes returned from reader
			if message.Bytes == 0 {
				// no lines buffered -> return error
				if pr.msgBuffer.isEmpty() {
					return reader.Message{}, err
				}

				// lines buffered, return multiline and error on next read
				return pr.collectMessageAfterError(err)
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if pr.msgBuffer.isEmptyMessage() || pr.pred(pr.msgBuffer.last, message.Content) {
				pr.msgBuffer.addLine(message)

				// return multiline and error on next read
				return pr.collectMessageAfterError(err)
			}

			// no match, return current multiline and retry with current line on next
			// call to readNext awaiting the error being reproduced (or resolved)
			// in next call to Next
			msg := pr.msgBuffer.finalize()
			pr.msgBuffer.load(message)
			return msg, nil
		}

		// handle case when endPattern is reached
		if pr.flushMatcher != nil {
			endPatternReached := (pr.flushMatcher.Match(message.Content))

			if endPatternReached == true {
				// return collected multiline event and
				// empty buffer for new multiline event
				pr.msgBuffer.addLine(message)
				msg := pr.msgBuffer.finalize()
				pr.resetState()
				return msg, nil
			}
		}

		// if predicate does not match current multiline -> return multiline event
		if !pr.msgBuffer.isEmptyMessage() && !pr.pred(pr.msgBuffer.last, message.Content) {
			msg := pr.msgBuffer.finalize()
			pr.msgBuffer.load(message)
			return msg, nil
		}

		// add line to current multiline event
		pr.msgBuffer.addLine(message)
	}
}

func (pr *patternReader) collectMessageAfterError(err error) (reader.Message, error) {
	msg := pr.msgBuffer.finalize()
	pr.msgBuffer.setErr(err)
	pr.setState((*patternReader).readFailed)
	return msg, nil
}

// readFailed returns empty message and error and resets line reader
func (pr *patternReader) readFailed() (reader.Message, error) {
	err := pr.msgBuffer.err
	pr.msgBuffer.setErr(nil)
	pr.resetState()
	return reader.Message{}, err
}

// resetState sets state of the reader to readFirst
func (pr *patternReader) resetState() {
	pr.setState((*patternReader).readFirst)
}

// setState sets state to the given function
func (pr *patternReader) setState(next func(pr *patternReader) (reader.Message, error)) {
	pr.state = next
}

func (pr *patternReader) Close() error {
	pr.setState((*patternReader).readClosed)
	return pr.reader.Close()
}

func (pr *patternReader) readClosed() (reader.Message, error) {
	return reader.Message{}, io.EOF
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
