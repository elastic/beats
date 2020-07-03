package multiline

import (
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
)

// MultiLine reader combining multiple line events into one multi-line event.
//
// Consecutive lines that satisfy the regular expression will be combined.
//
// The maximum number of bytes and lines to be returned is fully configurable.
// Even if limits are reached subsequent lines are matched, until event is
// fully finished.
//
// Errors will force the multiline reader to return the currently active
// multiline event first and finally return the actual error on next call to Next.
type whilePatternReader struct {
	reader    reader.Reader
	matcher   lineMatcherFunc
	logger    *logp.Logger
	msgBuffer *messageBuffer
	state     func(*whilePatternReader) (reader.Message, error)
}

func newMultilineWhilePatternReader(
	r reader.Reader,
	separator string,
	maxBytes int,
	config *Config,
) (reader.Reader, error) {
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

	matcherFunc := lineMatcher(*config.Pattern)
	if config.Negate {
		matcherFunc = negatedLineMatcher(matcherFunc)
	}

	pr := &whilePatternReader{
		reader:    r,
		matcher:   matcherFunc,
		msgBuffer: newMessageBuffer(maxBytes, maxLines, []byte(separator), config.SkipNewLine),
		logger:    logp.NewLogger("reader_multiline"),
		state:     (*whilePatternReader).readFirst,
	}
	return pr, nil
}

// Next returns next multi-line event.
func (pr *whilePatternReader) Next() (reader.Message, error) {
	return pr.state(pr)
}

func (pr *whilePatternReader) readFirst() (reader.Message, error) {
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

		// no match, return message
		if !pr.matcher(message.Content) {
			return message, nil
		}

		// Start new multiline event
		pr.msgBuffer.startNewMessage(message)
		pr.setState((*whilePatternReader).readNext)
		return pr.readNext()
	}
}

func (pr *whilePatternReader) readNext() (reader.Message, error) {
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
			if pr.msgBuffer.isEmptyMessage() || pr.matcher(message.Content) {
				pr.msgBuffer.addLine(message)

				// return multiline and error on next read
				return pr.collectMessageAfterError(err)
			}

			// no match, return current multiline and return current line on next
			// call to readNext
			msg := pr.msgBuffer.finalize()
			pr.msgBuffer.load(message)
			pr.setState((*whilePatternReader).notMatchedMessageLoad)
			return msg, nil
		}

		// no match, return message if buffer is empty, otherwise return current
		// multiline and save message to buffer
		if !pr.matcher(message.Content) {
			if pr.msgBuffer.isEmptyMessage() {
				return message, nil
			} else {
				msg := pr.msgBuffer.finalize()
				pr.msgBuffer.load(message)
				pr.setState((*whilePatternReader).notMatchedMessageLoad)
				return msg, nil
			}
		}

		// add line to current multiline event
		pr.msgBuffer.addLine(message)
	}
}

func (pr *whilePatternReader) collectMessageAfterError(err error) (reader.Message, error) {
	msg := pr.msgBuffer.finalize()
	pr.msgBuffer.setErr(err)
	pr.setState((*whilePatternReader).readFailed)
	return msg, nil
}

// readFailed returns empty message and error and resets line reader
func (pr *whilePatternReader) readFailed() (reader.Message, error) {
	err := pr.msgBuffer.err
	pr.msgBuffer.setErr(nil)
	pr.resetState()
	return reader.Message{}, err
}

// notMatchedMessageLoad returns not matched message from buffer
func (pr *whilePatternReader) notMatchedMessageLoad() (reader.Message, error) {
	msg := pr.msgBuffer.finalize()
	pr.resetState()
	return msg, nil
}

// resetState sets state of the reader to readFirst
func (pr *whilePatternReader) resetState() {
	pr.setState((*whilePatternReader).readFirst)
}

// setState sets state to the given function
func (pr *whilePatternReader) setState(next func(pr *whilePatternReader) (reader.Message, error)) {
	pr.state = next
}

type lineMatcherFunc func(content []byte) bool

func lineMatcher(pat match.Matcher) lineMatcherFunc {
	return func(content []byte) bool {
		return pat.Match(content)
	}
}

func negatedLineMatcher(m lineMatcherFunc) lineMatcherFunc {
	return func(content []byte) bool {
		return !m(content)
	}
}
