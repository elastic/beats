package smtp

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/mail"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
	buf     streambuf.Buffer
	config  *parserConfig
	pub     *transPub
	message *message
	conn    *connection
	state   parseState

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
	statusCode    uint
	statusPhrases []common.NetString

	raw []byte
	// list element use by 'transactions' for correlation
	next *message
}

// Error code if stream exceeds max allowed size on append.
var (
	constCRLF    = []byte("\r\n")
	constEOD     = []byte(".\r\n")
	constEODSize = 3
	// Responses are at least len("XXX\r\n") in size
	constMinRespSize  = 5
	constRespCodeSize = 3
	constPhraseOffset = 4
	constStatusSize   = 3
	constCRLFSize     = 2

	ErrStreamTooLarge = errors.New("Stream data too large")
)

func (p *parser) init(
	cfg *parserConfig,
	pub *transPub,
	onMessage func(*message) error,
) {
	*p = parser{
		buf:       streambuf.Buffer{},
		config:    cfg,
		pub:       pub,
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

func (p *parser) process(ts time.Time) error {
	if p.message == nil {
		// allocate new message object to be used by parser with current timestamp
		p.message = p.newMessage(ts)
	}

	msg, err := p.parse()
	if err != nil {
		return err
	}
	if msg == nil {
		return nil // wait for more data
	}

	// remove processed bytes from buffer
	p.buf.Reset()
	p.message = nil

	// call message handler callback
	if err := p.onMessage(msg); err != nil {
		return err
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

// SMTP server's reply message (if any) is separated from the
// 3-digit status code with either a space or a hyphen. The latter
// indicates that more lines are to follow.
func (*parser) isResponseComplete(raw []byte) bool {
	switch raw[constStatusSize] {
	case ' ', '\r':
		return true
	case '-':
		return false
	default:
		debugf("Failed to parse SMTP status code: %s",
			string(raw[:len(raw)-constCRLFSize]))
	}

	return true
}

func isResponseCode(raw []byte) bool {
	tail := constRespCodeSize

	for i := 0; i < tail; i++ {
		if raw[i] < '0' || raw[0] > '9' {
			return false
		}
	}
	if raw[tail] != ' ' && raw[tail] != '-' && raw[tail] != '\r' {
		return false
	}

	return true
}

func btoi(raw []byte) uint {
	var res uint
	for _, b := range raw {
		res = res*10 + uint(b-'0')
	}
	return res
}

func (p *parser) parsePayload() error {
	if !p.pub.sendDataHeaders && !p.pub.sendDataBody {
		return nil
	}

	m := p.message

	// ".\r\n" is not part of the payload
	parseTo := len(m.raw) - len(constEOD)

	payload, err := mail.ReadMessage(bytes.NewReader(m.raw[:parseTo]))
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

		raw, err := p.buf.CollectUntil(constCRLF)
		if err != nil {
			// Get more data
			return nil, nil
		}

		nbytes := len(raw)
		var words [][]byte
		isCode := nbytes >= constMinRespSize &&
			isResponseCode(raw[:constMinRespSize])

		if nbytes < constMinRespSize || !isCode || p.state == stateData {
			// Request
			m.IsRequest = true
			if p.state != stateData {
				words = bytes.SplitN(raw[:nbytes-constCRLFSize], []byte(" "), 2)
				m.command = common.NetString(strings.ToUpper(string(words[0])))
				if len(words) == 2 {
					m.param = common.NetString(words[1])
				}
			}
			if p.pub.sendRequest {
				m.raw = append(m.raw, raw...)
			}
		} else {
			// Response
			m.statusCode = btoi(raw[:3])
			if nbytes > constMinRespSize+1 {
				m.statusPhrases = append(m.statusPhrases,
					raw[constPhraseOffset:nbytes-constCRLFSize])
			}
			if p.pub.sendResponse {
				m.raw = append(m.raw, raw...)
			}
		}

		m.Size += uint64(nbytes)

		switch p.state {

		case stateCommand:
			if m.IsRequest {
				if bytes.Compare(words[0], []byte("DATA")) == 0 {
					p.state = stateData
				}
			} else {
				if !p.isResponseComplete(raw) {
					continue
				}
			}

		case stateData: // request only state
			if (p.pub.sendDataHeaders || p.pub.sendDataBody) && !p.pub.sendRequest {
				m.raw = append(m.raw, raw...)
			}
			if bytes.Compare(raw, constEOD) != 0 {
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
