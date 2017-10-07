package smtp

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
	buf     streambuf.Buffer
	config  *parserConfig
	message *message
	state   parseState
	payload []byte

	onMessage func(m *message) error
}

type parserConfig struct {
	maxBytes int
}

type message struct {
	applayer.Message

	// Request
	command common.NetString
	param   common.NetString
	// Request data payload
	headers map[string]common.NetString
	body    common.NetString

	// Response
	statusCode    int
	statusPhrases []common.NetString

	// list element use by 'transactions' for correlation
	next *message
}

// Error code if stream exceeds max allowed size on append.
var (
	constCRLF = []byte("\r\n")
	constEOD  = []byte(".\r\n")
	// Responses are at least len("XXX\r\n") in size
	constMinRespSize  = 5
	constPhraseOffset = 4
	constStatusSize   = 3
	constCRLFSize     = 2

	ErrStreamTooLarge = errors.New("Stream data too large")
)

func (p *parser) init(
	cfg *parserConfig,
	onMessage func(*message) error,
) {
	*p = parser{
		buf:       streambuf.Buffer{},
		config:    cfg,
		onMessage: onMessage,
	}
}

func (p *parser) append(data []byte) error {
	_, err := p.buf.Write(data)
	if err != nil {
		return err
	}

	if p.config.maxBytes > 0 && p.buf.Total() > p.config.maxBytes {
		return ErrStreamTooLarge
	}
	return nil
}

func (p *parser) feed(ts time.Time, data []byte) error {
	if err := p.append(data); err != nil {
		return err
	}

	for p.buf.Total() > 0 {
		if p.message == nil {
			// allocate new message object to be used by parser with current timestamp
			p.message = p.newMessage(ts)
		}

		msg, err := p.parse()
		if err != nil {
			return err
		}
		if msg == nil {
			break // wait for more data
		}

		// reset buffer and message -> handle next message in buffer
		p.buf.Reset()
		p.message = nil

		// call message handler callback
		if err := p.onMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func (p *parser) newMessage(ts time.Time) *message {
	return &message{
		Message: applayer.Message{
			Ts: ts,
		},
	}
}

// SMTP server's reply message is separated from the 3-digit status
// code with either a space or a hyphen. The latter indicates that
// more lines are to follow.
func (*parser) isResponseComplete(byts []byte) bool {
	switch byts[constStatusSize] {
	case ' ':
		return true
	case '-':
		return false
	default:
		debugf("Failed to understand SMTP status code: %s",
			string(byts[:len(byts)-constCRLFSize]))
	}

	return true
}

func (p *parser) parsePayload() error {
	m := p.message

	payload, err := mail.ReadMessage(bytes.NewReader(p.payload))
	if err != nil {
		return err
	}

	if m.headers == nil {
		m.headers = make(map[string]common.NetString)
	}

	for k := range payload.Header {
		m.headers[k] = common.NetString(payload.Header.Get(k))
	}

	if body, err := ioutil.ReadAll(payload.Body); err != nil {
		return err
	} else {
		m.body = common.NetString(body)
	}

	return nil
}

func (p *parser) parse() (*message, error) {
	m := p.message

	for {
		var err error
		var code int

		byts, err := p.buf.CollectUntil(constCRLF)
		if err != nil {
			// Get more data
			return nil, nil
		}

		nbytes := len(byts)
		str := string(byts[:nbytes-constCRLFSize])
		words := []string{}

		if nbytes >= constMinRespSize {
			code, err = strconv.Atoi(str[:constStatusSize])
		}

		if nbytes < constMinRespSize || err != nil || p.state == stateData {
			// Request
			m.IsRequest = true
			if p.state != stateData {
				words = strings.SplitN(str, " ", 2)
				m.command = common.NetString(strings.ToUpper(words[0]))
				if len(words) == 2 {
					m.param = common.NetString(words[1])
				}
			}
		} else {
			// Response
			m.statusCode = code
			if nbytes > constMinRespSize+1 {
				m.statusPhrases = append(m.statusPhrases,
					byts[constPhraseOffset:nbytes-constCRLFSize])
			}
		}

		m.Size += uint64(nbytes)

		switch p.state {

		case stateCommand:
			if m.IsRequest {
				if words[0] == "DATA" {
					p.state = stateData
				}
			} else {
				if !p.isResponseComplete(byts) {
					continue
				}
			}

		case stateData:
			// Request only state
			if bytes.Compare(byts, constEOD) != 0 {
				p.payload = append(p.payload, byts...)
				continue
			} else {
				// We want to provide a request section since the
				// server reply is going to be a response (as opposed
				// to just a prompt). So we use a pseudo command.
				m.command = common.NetString("EOD")
				if err := p.parsePayload(); err != nil {
					return nil, err
				}
			}

			p.state = stateCommand
		}

		break
	}

	return m, nil
}
