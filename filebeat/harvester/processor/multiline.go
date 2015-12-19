package processor

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/elastic/beats/filebeat/config"
)

// MultiLine processor combining multiple line events into one multi-line event.
//
// Lines to be combined are matched by some configurable predicate using
// regular expression.
//
// The maximum number of bytes and lines to be returned is fully configurable.
// Even if limits are reached subsequent lines are matched, until event is
// fully finished.
//
// Errors will force the multiline processor to return the currently active
// multiline event first and finally return the actual error on next call to Next.
type MultiLine struct {
	reader   LineProcessor
	pred     matcher
	maxBytes int // bytes stored in content
	maxLines int

	ts        time.Time
	content   []byte
	last      []byte
	readBytes int // bytes as read from input source
	numLines  int

	err   error // last seen error
	state func(*MultiLine) (Line, error)
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
	errMultilineTimeout = errors.New("multline timeout")
)

// NewMultiline creates a new multi-line processor combining stream of
// line events into stream of multi-line events.
func NewMultiline(
	r LineProcessor,
	maxBytes int,
	config *config.MultilineConfig,
) (*MultiLine, error) {
	type matcherFactory func(pattern string) (matcher, error)
	types := map[string]matcherFactory{
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
	if config.Timeout != "" {
		timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration '%s': %v", config.Timeout, err)
		}
		if timeout < 0 {
			return nil, fmt.Errorf("timeout %v must not be negative", config.Timeout)
		}
	}

	if timeout > 0 {
		r = newTimeoutProcessor(r, errMultilineTimeout, timeout)
	}

	mlr := &MultiLine{
		reader:   r,
		pred:     matcher,
		state:    (*MultiLine).readNext,
		maxBytes: maxBytes,
		maxLines: maxLines,
	}
	return mlr, nil
}

// Next returns next multi-line event.
func (mlr *MultiLine) Next() (Line, error) {
	return mlr.state(mlr)
}

func (mlr *MultiLine) readNext() (Line, error) {
	for {
		l, err := mlr.reader.Next()
		if err != nil {
			// handle multiline timeout signal
			if err == errMultilineTimeout {
				// no lines buffered -> ignore timeout
				if mlr.numLines == 0 {
					continue
				}

				// return collected multiline event and
				// empty buffer for new multiline event
				l := mlr.pushLine()
				return l, nil
			}

			// handle error without any bytes returned from reader
			if l.Bytes == 0 {
				// no lines buffered -> return error
				if mlr.numLines == 0 {
					return Line{}, err
				}

				// lines buffered, return multiline and error on next read
				l := mlr.pushLine()
				mlr.err = err
				mlr.state = (*MultiLine).readFailed
				return l, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if mlr.readBytes == 0 || mlr.pred(mlr.last, l.Content) {
				mlr.addLine(l)

				// return multiline and error on next read
				l := mlr.pushLine()
				mlr.err = err
				mlr.state = (*MultiLine).readFailed
				return l, nil
			}

			// no match, return current multline and retry with current line on next
			// call to readNext awaiting the error being reproduced (or resolved)
			// in next call to Next
			l := mlr.startNewLine(l)
			return l, nil
		}

		// if predicate does not match current multiline -> return multiline event
		if mlr.readBytes > 0 && !mlr.pred(mlr.last, l.Content) {
			l := mlr.startNewLine(l)
			return l, nil
		}

		// add line to current multiline event
		mlr.addLine(l)
	}
}

func (mlr *MultiLine) readFailed() (Line, error) {
	// return error and reset line reader
	err := mlr.err
	mlr.err = nil
	mlr.state = (*MultiLine).readNext
	return Line{}, err
}

func (mlr *MultiLine) startNewLine(l Line) Line {
	retLine := mlr.pushLine()
	mlr.addLine(l)
	mlr.ts = l.Ts
	return retLine
}

func (mlr *MultiLine) pushLine() Line {
	content := mlr.content
	sz := mlr.readBytes

	mlr.content = nil
	mlr.last = nil
	mlr.readBytes = 0
	mlr.numLines = 0
	mlr.err = nil

	return Line{Ts: mlr.ts, Content: content, Bytes: sz}
}

func (mlr *MultiLine) addLine(l Line) {
	if l.Bytes <= 0 {
		return
	}

	space := mlr.maxBytes - len(mlr.content)
	spaceLeft := (mlr.maxBytes <= 0 || space > 0) &&
		(mlr.maxLines <= 0 || mlr.numLines < mlr.maxLines)
	if spaceLeft {
		if space < 0 || space > len(l.Content) {
			space = len(l.Content)
		}
		mlr.content = append(mlr.content, l.Content[:space]...)
		mlr.numLines++
	}

	mlr.last = l.Content
	mlr.readBytes += l.Bytes
}

// matchers

func afterMatcher(pattern string) (matcher, error) {
	return genPatternMatcher(pattern, func(last, current []byte) []byte {
		return current
	})
}

func beforeMatcher(pattern string) (matcher, error) {
	return genPatternMatcher(pattern, func(last, current []byte) []byte {
		return last
	})
}

func negatedMatcher(m matcher) matcher {
	return func(last, current []byte) bool {
		return !m(last, current)
	}
}

func genPatternMatcher(
	pattern string,
	sel func(last, current []byte) []byte,
) (matcher, error) {
	reg, err := regexp.CompilePOSIX(pattern)
	if err != nil {
		return nil, err
	}

	matcher := func(last, current []byte) bool {
		line := sel(last, current)
		return reg.Match(line)
	}
	return matcher, nil
}
