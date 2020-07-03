package multiline

import (
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
)

type wildPatternReader struct {
	reader    reader.Reader
	matcher   wildMatcherFunc
	logger    *logp.Logger
	msgBuffer *messageBuffer
}

func newMultilineWildPatternReader(
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

	matcherFunc := wildMatcher(*config.Pattern)
	if config.Negate {
		matcherFunc = negatedWildMatcher(matcherFunc)
	}

	pr := &wildPatternReader{
		reader:    r,
		matcher:   matcherFunc,
		msgBuffer: newMessageBuffer(maxBytes, maxLines, []byte(separator), config.SkipNewLine),
		logger:    logp.NewLogger("reader_multiline"),
	}
	return pr, nil
}

// Next returns next multi-line event.
func (pr *wildPatternReader) Next() (reader.Message, error) {
	for {
		if pr.msgBuffer.err != nil {
			err := pr.msgBuffer.err
			pr.msgBuffer.setErr(nil)
			return reader.Message{}, err
		}

		if pr.msgBuffer.notMatched {
			msg := pr.msgBuffer.finalize()
			return msg, nil
		}

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
				return msg, nil
			}

			// handle error without any bytes returned from reader
			if message.Bytes == 0 {
				// no lines buffered -> return error
				if pr.msgBuffer.isEmpty() {
					return reader.Message{}, err
				}

				// lines buffered, return multiline and error on next read
				msg := pr.msgBuffer.finalize()
				pr.msgBuffer.setErr(err)
				return msg, nil
			}

			// handle error with some content being returned by reader and
			// line matching multiline criteria or no multiline started yet
			if pr.msgBuffer.isEmptyMessage() || pr.matcher(message.Content) {
				pr.msgBuffer.addLine(message)

				// return multiline and error on next read
				msg := pr.msgBuffer.finalize()
				pr.msgBuffer.setErr(err)
				return msg, nil
			}

			// no match, return current multiline and retry with current line on next
			// call to readNext awaiting the error being reproduced (or resolved)
			// in next call to Next
			msg := pr.msgBuffer.finalize()
			pr.msgBuffer.loadNotMatched(message)
			return msg, nil
		}

		if message.Bytes == 0 {
			continue
		}

		isMatch := pr.matcher(message.Content)

		if !isMatch {
			if pr.msgBuffer.isEmpty() {
				return message, nil
			} else {
				msg := pr.msgBuffer.finalize()
				pr.msgBuffer.loadNotMatched(message)
				return msg, nil
			}
		}

		// add line to current multiline event
		pr.msgBuffer.addLine(message)
	}
}

type wildMatcherFunc func(content []byte) bool

func wildMatcher(pat match.Matcher) wildMatcherFunc {
	return func(content []byte) bool {
		return pat.Match(content)
	}
}

func negatedWildMatcher(m wildMatcherFunc) wildMatcherFunc {
	return func(content []byte) bool {
		return !m(content)
	}
}
