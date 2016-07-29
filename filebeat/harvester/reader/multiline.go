package reader

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/elastic/beats/libbeat/common"
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
type Multiline struct {
	reader    Reader
	pred      matcher
	maxBytes  int // bytes stored in content
	maxLines  int
	separator []byte
	ts        time.Time
	content   []byte
	last      []byte
	readBytes int // bytes as read from input source
	numLines  int
	fields    common.MapStr
	err       error // last seen error
	state     func(*Multiline) (Message, error)
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
	sigMultilineTimeout = errors.New("multline timeout")
)

// NewMultiline creates a new multi-line reader combining stream of
// line events into stream of multi-line events.
func NewMultiline(
	r Reader,
	separator string,
	maxBytes int,
	config *MultilineConfig,
) (*Multiline, error) {
	types := map[string]func(*regexp.Regexp) (matcher, error){
		"before": beforeMatcher,
		"after":  afterMatcher,
	}

	matcherType, ok := types[config.Match]
	if !ok {
		return nil, fmt.Errorf("unknown matcher type: %s", config.Match)
	}

	matcher, err := matcherType(config.Pattern)
	if err != nil {
		return nil, err
	}

	if config.Negate {
		matcher = negatedMatcher(matcher)
	}

	maxLines := defaultMaxLines
	if config.MaxLines != nil {
		maxLines = *config.MaxLines
	}

	timeout := defaultMultilineTimeout
	if config.Timeout != nil {
		timeout = *config.Timeout
		if timeout < 0 {
			return nil, fmt.Errorf("timeout %v must not be negative", config.Timeout)
		}
	}

	if timeout > 0 {
		r = NewTimeout(r, sigMultilineTimeout, timeout)
	}

	mlr := &Multiline{
		reader:    r,
		pred:      matcher,
		state:     (*Multiline).readFirst,
		maxBytes:  maxBytes,
		maxLines:  maxLines,
		separator: []byte(separator),
	}
	return mlr, nil
}

// Next returns next multi-line event.
func (mlr *Multiline) Next() (Message, error) {
	return mlr.state(mlr)
}

func (mlr *Multiline) readFirst() (Message, error) {
	for {
		message, err := mlr.reader.Next()
		if err != nil {
			// no lines buffered -> ignore timeout
			if err == sigMultilineTimeout {
				continue
			}

			// pass error to caller (next layer) for handling
			return message, err
		}

		if message.Bytes == 0 {
			continue
		}

		// Start new multiline event
		mlr.clearBuffer()
		mlr.resetBuffer(message)
		mlr.setState((*Multiline).readNext)
		return mlr.readNext()
	}
}

func (mlr *Multiline) readNext() (Message, error) {
	for {
		message, err := mlr.reader.Next()
		if err != nil {
			// handle multiline timeout signal
			if err == sigMultilineTimeout {
				// no lines buffered -> ignore timeout
				if mlr.numLines == 0 {
					continue
				}

				// return collected multiline event and
				// empty buffer for new multiline event
				message := mlr.finalize()
				mlr.resetState()
				return message, nil
			}

			// handle error without any bytes returned from reader
			if message.Bytes == 0 {
				// no lines buffered -> return error
				if mlr.numLines == 0 {
					return Message{}, err
				}

				// lines buffered, return multiline and error on next read
				m := mlr.finalize()
				mlr.err = err
				mlr.setState((*Multiline).readFailed)
				return m, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if mlr.readBytes == 0 || mlr.pred(mlr.last, message.Content) {
				mlr.addLine(message)

				// return multiline and error on next read
				m := mlr.finalize()
				mlr.err = err
				mlr.setState((*Multiline).readFailed)
				return m, nil
			}

			// no match, return current multline and retry with current line on next
			// call to readNext awaiting the error being reproduced (or resolved)
			// in next call to Next
			m := mlr.finalize()
			mlr.resetBuffer(message)
			return m, nil
		}

		// if predicate does not match current multiline -> return multiline event
		if mlr.readBytes > 0 && !mlr.pred(mlr.last, message.Content) {
			m := mlr.finalize()
			mlr.resetBuffer(message)
			return m, nil
		}

		// add line to current multiline event
		mlr.addLine(message)
	}
}

// readFailed returns empty message and error and resets line reader
func (mlr *Multiline) readFailed() (Message, error) {
	err := mlr.err
	mlr.err = nil
	mlr.resetState()
	return Message{}, err
}

// resetBuffer loads the reader with the given message. It is recommend to either
// run reset or finalize before.
func (mlr *Multiline) resetBuffer(m Message) {
	mlr.addLine(m)
	// Timestamp of first message is taken as overall timestamp
	mlr.ts = m.Ts
	mlr.fields = m.Fields
}

// finalize writes the existing content into the returned message and resets all reader variables.
func (mlr *Multiline) finalize() Message {

	// Create message from existing content
	m := Message{
		Ts:      mlr.ts,
		Content: mlr.content,
		Fields:  mlr.fields,
		Bytes:   mlr.readBytes,
	}

	mlr.clearBuffer()

	return m
}

// clearBuffer resets the reader buffer variables
func (mlr *Multiline) clearBuffer() {
	mlr.content = nil
	mlr.last = nil
	mlr.readBytes = 0
	mlr.numLines = 0
	mlr.err = nil
	mlr.fields = nil
}

// addLine adds the read content to the message
// The content is only added if maxBytes and maxLines is not exceed. In case one of the
// two is exceeded, addLine keeps processing but does not add it to the content.
func (mlr *Multiline) addLine(m Message) {
	if m.Bytes <= 0 {
		return
	}

	sz := len(mlr.content)
	addSeparator := len(mlr.content) > 0 && len(mlr.separator) > 0
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

		tmp := mlr.content
		if addSeparator {
			tmp = append(tmp, mlr.separator...)
		}
		mlr.content = append(tmp, m.Content[:space]...)
		mlr.numLines++
	}

	mlr.last = m.Content
	mlr.readBytes += m.Bytes
}

// resetState sets state of the reader to readFirst
func (mlr *Multiline) resetState() {
	mlr.setState((*Multiline).readFirst)
}

// setState sets state to the given function
func (mlr *Multiline) setState(next func(mlr *Multiline) (Message, error)) {
	mlr.state = next
}

// matchers

func afterMatcher(regex *regexp.Regexp) (matcher, error) {
	return genPatternMatcher(regex, func(last, current []byte) []byte {
		return current
	})
}

func beforeMatcher(regex *regexp.Regexp) (matcher, error) {
	return genPatternMatcher(regex, func(last, current []byte) []byte {
		return last
	})
}

func negatedMatcher(m matcher) matcher {
	return func(last, current []byte) bool {
		return !m(last, current)
	}
}

func genPatternMatcher(
	regex *regexp.Regexp,
	sel func(last, current []byte) []byte,
) (matcher, error) {
	matcher := func(last, current []byte) bool {
		line := sel(last, current)
		return regex.Match(line)
	}
	return matcher, nil
}
