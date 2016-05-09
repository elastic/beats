package processor

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
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
	reader    LineProcessor
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
	separator string,
	maxBytes int,
	config *config.MultilineConfig,
) (*MultiLine, error) {
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
		r = newTimeoutProcessor(r, errMultilineTimeout, timeout)
	}

	mlr := &MultiLine{
		reader:    r,
		pred:      matcher,
		state:     (*MultiLine).readFirst,
		maxBytes:  maxBytes,
		maxLines:  maxLines,
		separator: []byte(separator),
	}
	return mlr, nil
}

// Next returns next multi-line event.
func (mlr *MultiLine) Next() (Line, error) {
	return mlr.state(mlr)
}

func (mlr *MultiLine) readFirst() (Line, error) {
	for {
		l, err := mlr.reader.Next()
		if err == nil {
			if l.Bytes == 0 {
				continue
			}

			mlr.startNewLine(l)
			mlr.state = (*MultiLine).readNext
			return mlr.readNext()
		}

		// no lines buffered -> ignore timeout
		if err == errMultilineTimeout {
			continue
		}

		// something is wrong here
		return l, err
	}

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
				mlr.reset()
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
	mlr.reset()
	return Line{}, err
}

func (mlr *MultiLine) startNewLine(l Line) Line {
	retLine := mlr.pushLine()
	mlr.addLine(l)
	mlr.ts = l.Ts
	mlr.fields = l.Fields
	return retLine
}

func (mlr *MultiLine) pushLine() Line {
	content := mlr.content
	sz := mlr.readBytes
	fields := mlr.fields

	mlr.content = nil
	mlr.last = nil
	mlr.readBytes = 0
	mlr.numLines = 0
	mlr.err = nil
	mlr.fields = nil

	return Line{Ts: mlr.ts, Content: content, Fields: fields, Bytes: sz}
}

func (mlr *MultiLine) addLine(l Line) {
	if l.Bytes <= 0 {
		return
	}

	sz := len(mlr.content)
	addSeparator := len(mlr.content) > 0 && len(mlr.separator) > 0
	if addSeparator {
		sz += len(mlr.separator)
	}

	space := mlr.maxBytes - sz
	spaceLeft := (mlr.maxBytes <= 0 || space > 0) &&
		(mlr.maxLines <= 0 || mlr.numLines < mlr.maxLines)
	if spaceLeft {
		if space < 0 || space > len(l.Content) {
			space = len(l.Content)
		}

		tmp := mlr.content
		if addSeparator {
			tmp = append(tmp, mlr.separator...)
		}
		mlr.content = append(tmp, l.Content[:space]...)
		mlr.numLines++
	}

	mlr.last = l.Content
	mlr.readBytes += l.Bytes
}

func (mlr *MultiLine) reset() {
	mlr.state = (*MultiLine).readFirst
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
