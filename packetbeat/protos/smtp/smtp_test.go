// +build !integration

package smtp

import (
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/stretchr/testify/assert"
)

type input struct {
	requ string
	resp string
}

const (
	none     = ""
	prompt   = "220 localhost ESMTP Exim 4.89 Sun, 12 Nov 2017 22:22:42 -0800\r\n"
	ehlo     = "EHLO localhost\r\n"
	ehloResp = "250-localhost Hello localhost [::1]\r\n" +
		"250-SIZE 52428800\r\n" +
		"250-8BITMIME\r\n" +
		"250-PIPELINING\r\n" +
		"250-PRDR\r\n" +
		"250 HELP\r\n"
	mailf    = "MAIL FROM:<bar@example.org>\r\n"
	mailResp = "250 OK\r\n"
	rcpt     = "RCPT TO:<foo@example.org>\r\n"
	rcptResp = "250 Accepted\r\n"
	data     = "DATA\r\n"
	dataResp = "354 Enter message, ending with \".\" on a line by itself\r\n"
	mime     = "From: bar@example.org\r\n" +
		"To: foo@example.com\r\n" +
		"Subject: Test\r\n" +
		"Date: Thu, 20 Dec 2012 12:00:00 +0000\r\n" +
		"\r\n" +
		"Testing\r\n" +
		".\r\n"
	mimeResp = "250 OK id=1eE893-0006ry-Gi\r\n"
	quit     = "QUIT\r\n"
	quitResp = "221 localhost closing connection\r\n"

	promptSplit1 = "220 localhost ESMTP Exim 4.89 S"
	promptSplit2 = "un, 12 Nov 2017 22:22:42 -0800\r\n"

	ehloSplit1 = "EHLO local"
	ehloSplit2 = "host\r\n"

	ehloRespSplit1 = "250-localhost"
	ehloRespSplit2 = " Hello localhost [::1]\r\n" +
		"250-SIZE 52428800\r\n" +
		"250-8BITMIME\r\n" +
		"250-PIPELINING\r\n" +
		"250-PRDR\r\n" +
		"250 HELP\r\n"
	mailfSplit1    = "MAIL FROM:<ba"
	mailfSplit2    = "r@example.org>\r\n"
	mailRespSplit1 = "250 OK\r"
	mailRespSplit2 = "\n"
	rcptSplit1     = "RC"
	rcptSplit2     = "PT TO:<"
	rcptSplit3     = "foo@example.org>\r\n"
	rcptRespSplit1 = "2"
	rcptRespSplit2 = "50 Acc"
	rcptRespSplit3 = "epted\r\n"
	mimeSplit1     = "From: bar@example.org\r\n" +
		"To: foo@example.com\r\n" +
		"Subject: Test\r\n" +
		"Date: Thu, 20 Dec 2012 12:00:00 +0000\r\n" +
		"Testing\r\n" +
		".\r"
	mimeSplit2     = "\n"
	mimeRespSplit1 = "250 OK id=1eE893-0006ry-Gi\r"
	mimeRespSplit2 = "\n"
)

type script []input

type messageStore struct {
	messages []message
}

func (m *messageStore) publish(msg *message) error {
	m.messages = append(m.messages, *msg)
	return nil
}

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	e.events = append(e.events, event)
}

func TestParser(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"smtp"})
	}

	t.Run("basic", func(t *testing.T) {
		testParser(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, rcptResp},
			{data, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		})
	})

	t.Run("basic split messages", func(t *testing.T) {
		testParser(t, script{
			{none, promptSplit1},
			{none, promptSplit2},
			{ehloSplit1, none},
			{ehloSplit2, ehloRespSplit1},
			{none, ehloRespSplit2},
			{mailfSplit1, none},
			{mailfSplit2, mailRespSplit1},
			{none, mailRespSplit2},
			{rcptSplit1, none},
			{rcptSplit2, none},
			{rcptSplit3, rcptRespSplit1},
			{none, rcptRespSplit2},
			{none, rcptRespSplit3},
			{data, dataResp},
			{mimeSplit1, none},
			{mimeSplit2, mimeRespSplit1},
			{none, mimeRespSplit2},
			{quit, quitResp},
		})
	})
}

func TestSyncer(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"smtp"})
	}

	t.Run("single-line prompt", func(t *testing.T) {
		testSyncerNotDone(t, script{
			{none, prompt},
		})
	})

	t.Run("multiline prompt", func(t *testing.T) {
		testSyncerNotDone(t, script{
			{none, ehloResp},
		})
	})

	t.Run("single-line request", func(t *testing.T) {
		testSyncerNotDone(t, script{
			{quit, none},
		})
	})

	t.Run("multiline request", func(t *testing.T) {
		testSyncerDone(t, script{
			{ehlo, none},
			{ehlo, none},
			{ehlo, none},
		})
	})

	t.Run("multiline split request", func(t *testing.T) {
		testSyncerDone(t, script{
			{ehloSplit1, none},
			{ehloSplit2, none},
			{ehlo, none},
			{ehlo, none},
		})
	})

	t.Run("request-response", func(t *testing.T) {
		testSyncerDone(t, script{
			{none, prompt},
			{ehlo, ehloResp},
		})
	})
}

func TestIntegration(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"smtp"})
	}

	t.Run("basic", func(t *testing.T) {
		testIntegration(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, rcptResp},
			{data, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		})
	})

	t.Run("multiple recipients", func(t *testing.T) {
		testIntegrationMultiple(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, rcptResp},
			{rcpt, rcptResp},
			{data, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		})
	})

	t.Run("late start data command", func(t *testing.T) {
		testIntegrationGap(t, script{
			{data, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		}, []string{"MAIL", "COMMAND"})
	})

	t.Run("late start data payload", func(t *testing.T) {
		testIntegrationGap(t, script{
			{mime, mimeResp},
			{quit, quitResp},
		}, []string{"MAIL", "COMMAND"})
	})

	t.Run("gap before data payload", func(t *testing.T) {
		testIntegrationGap(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{ehlo, none},
			{none, none},
			{none, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		}, []string{"PROMPT", "COMMAND", "MAIL", "COMMAND"})
	})

	t.Run("gap between data payloads", func(t *testing.T) {
		testIntegrationGap(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, rcptResp},
			{data, dataResp},
			{mime, mimeResp},
			{none, none},
			{mime, mimeResp},
			{quit, quitResp},
		}, []string{"PROMPT", "COMMAND", "MAIL", "MAIL", "COMMAND"})
	})

	t.Run("gap during mail transaction", func(t *testing.T) {
		testIntegrationGap(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, none},
			{none, none},
			{data, dataResp},
			{mime, mimeResp},
			{quit, quitResp},
		}, []string{"PROMPT", "COMMAND", "MAIL", "MAIL", "COMMAND"})
	})

	t.Run("fin during mail transaction", func(t *testing.T) {
		testIntegrationGap(t, script{
			{none, prompt},
			{ehlo, ehloResp},
			{mailf, mailResp},
			{rcpt, none},
			{none, none},
		}, []string{"PROMPT", "COMMAND", "MAIL"})
	})

}

func testIntegration(t *testing.T, s script) {
	var store eventStore
	integrationPlayScript(s, &store)

	ee := store.events

	for _, e := range ee {
		assert.Equal(t, "OK", e.Fields["status"])
		assert.NotEmpty(t, e.Fields["smtp"].(common.MapStr)["session_id"])
	}

	var ff, requ, resp common.MapStr

	ff = ee[0].Fields["smtp"].(common.MapStr)
	assert.Equal(t, "PROMPT", ff["type"])
	resp = ff["response"].(common.MapStr)
	assert.Equal(t, 220, int(resp["code"].(uint)))
	assert.Equal(t, 1, len(resp["phrases"].([]common.NetString)))

	ff = ee[1].Fields["smtp"].(common.MapStr)
	assert.Equal(t, "COMMAND", ff["type"])
	requ = ff["request"].(common.MapStr)
	assert.Equal(t, "EHLO", string(requ["command"].(common.NetString)))
	assert.NotEmpty(t, requ["param"])
	resp = ff["response"].(common.MapStr)
	assert.Equal(t, 250, int(resp["code"].(uint)))
	assert.Equal(t, 6, len(resp["phrases"].([]common.NetString)))

	ff = ee[2].Fields["smtp"].(common.MapStr)
	assert.Equal(t, "MAIL", ff["type"])
	requ = ff["request"].(common.MapStr)
	assert.NotEmpty(t, requ["body"])
	assert.Equal(t, 4, len(requ["headers"].(map[string]common.NetString)))
	assert.NotEmpty(t, requ["envelope_sender"])
	assert.Equal(t, 1, len(requ["envelope_recipients"].([]common.NetString)))

	ff = ee[3].Fields["smtp"].(common.MapStr)
	assert.Equal(t, "COMMAND", ff["type"])
	requ = ff["request"].(common.MapStr)
	assert.Equal(t, "QUIT", string(requ["command"].(common.NetString)))
	assert.Empty(t, requ["param"])
	resp = ff["response"].(common.MapStr)
	assert.Equal(t, 221, int(resp["code"].(uint)))
	assert.Equal(t, 1, len(resp["phrases"].([]common.NetString)))
}

func testIntegrationMultiple(t *testing.T, s script) {
	var store eventStore
	integrationPlayScript(s, &store)

	ee := store.events
	for _, e := range ee {
		assert.Equal(t, "OK", e.Fields["status"])
	}

	ff := ee[2].Fields["smtp"].(common.MapStr)
	assert.Equal(t, "MAIL", ff["type"])
	requ := ff["request"].(common.MapStr)
	assert.Equal(t, 2, len(requ["envelope_recipients"].([]common.NetString)))
}

func testIntegrationGap(t *testing.T, s script, transSeq []string) {
	var store eventStore
	integrationPlayScript(s, &store)

	ee := store.events

	for i, typ := range transSeq {
		ff := ee[i].Fields["smtp"].(common.MapStr)
		assert.Equal(t, typ, ff["type"])
	}
}

func integrationPlayScript(s script, store *eventStore) {
	smtp := modForTests(store)
	smtp.pub.sendDataHeaders = true
	smtp.pub.sendDataBody = true

	tcpTuple := testCreateTCPTuple()
	var private protos.ProtocolData

	for i, t := range s {
		if t.requ == none && t.resp == none {
			if i == len(s)-1 {
				smtp.ReceivedFin(tcpTuple, 0, private)
			} else {
				smtp.GapInStream(tcpTuple, 0, 0, private)
			}
			private = nil
			continue
		}
		dir := uint8(0)
		for _, s := range []string{t.requ, t.resp} {
			if s != "" {
				private = smtp.Parse(
					&protos.Packet{Payload: []byte(s)},
					tcpTuple,
					dir,
					private)
			}
			dir ^= 1
		}
	}
}

func testSyncerNotDone(t *testing.T, s script) {
	syncer := syncerForTests()
	syncerPlayScript(s, syncer)
	assert.False(t, syncer.done)
}

func testSyncerDone(t *testing.T, s script) {
	syncer := syncerForTests()
	syncerPlayScript(s, syncer)
	assert.True(t, syncer.done)
}

func syncerPlayScript(s script, syncer *syncer) {
	for _, t := range s {
		dir := uint8(0)
		for _, s := range []string{t.requ, t.resp} {
			if s != "" {
				err := syncer.parsers[dir].append([]byte(s))
				if err != nil {
					panic(err)
				}
				err = syncer.process(time.Now(), dir)
				if err != nil {
					panic(err)
				}
			}
			dir ^= 1
		}
	}
}

func testParser(t *testing.T, s script) {
	var store messageStore
	parserPlayScript(s, &store)

	mm := store.messages

	assert.False(t, mm[0].IsRequest)
	assert.Equal(t, uint(220), mm[0].statusCode)
	assert.Equal(t,
		"localhost ESMTP Exim 4.89 Sun, 12 Nov 2017 22:22:42 -0800",
		string(mm[0].statusPhrases[0]))

	assert.True(t, mm[1].IsRequest)
	assert.Equal(t, "EHLO", string(mm[1].command))
	assert.Equal(t, "localhost", string(mm[1].param))
	assert.Equal(t, len("EHLO localhost\r\n"), int(mm[1].Size))

	assert.False(t, mm[2].IsRequest)
	assert.Equal(t, uint(250), mm[2].statusCode)
	assert.Equal(t,
		"HELP",
		string(mm[2].statusPhrases[5]))

	assert.True(t, mm[3].IsRequest)
	assert.Equal(t, "MAIL", string(mm[3].command))
	assert.Equal(t, "FROM:<bar@example.org>", string(mm[3].param))

	assert.False(t, mm[4].IsRequest)
	assert.Equal(t, uint(250), mm[4].statusCode)
	assert.Equal(t,
		"OK",
		string(mm[4].statusPhrases[0]))
	assert.Equal(t, len("250 OK\r\n"), int(mm[4].Size))

	assert.True(t, mm[5].IsRequest)
	assert.Equal(t, "RCPT", string(mm[5].command))
	assert.Equal(t, "TO:<foo@example.org>", string(mm[5].param))

	assert.False(t, mm[6].IsRequest)
	assert.Equal(t, uint(250), mm[6].statusCode)
	assert.Equal(t,
		"Accepted",
		string(mm[6].statusPhrases[0]))

	assert.True(t, mm[7].IsRequest)
	assert.Equal(t, "DATA", string(mm[7].command))
	assert.Equal(t, "", string(mm[7].param))

	assert.False(t, mm[8].IsRequest)
	assert.Equal(t, uint(354), mm[8].statusCode)
	assert.Equal(t,
		"Enter message, ending with \".\" on a line by itself",
		string(mm[8].statusPhrases[0]))

	assert.True(t, mm[9].IsRequest)
	assert.Equal(t, constEOD, []byte(mm[9].command))
	assert.Equal(t, "", string(mm[9].param))

	assert.False(t, mm[10].IsRequest)
	assert.Equal(t, uint(250), mm[10].statusCode)
	assert.Equal(t,
		"OK id=1eE893-0006ry-Gi",
		string(mm[10].statusPhrases[0]))

	assert.True(t, mm[11].IsRequest)
	assert.Equal(t, "QUIT", string(mm[11].command))
	assert.Equal(t, "", string(mm[11].param))

	assert.False(t, mm[12].IsRequest)
	assert.Equal(t, uint(221), mm[12].statusCode)
	assert.Equal(t,
		"localhost closing connection",
		string(mm[12].statusPhrases[0]))
}

func parserPlayScript(s script, store *messageStore) {
	pp := []*parser{parserForParserTests(store),
		parserForParserTests(store)}

	for _, t := range s {
		dir := 0
		for _, s := range []string{t.requ, t.resp} {
			if s != "" {
				p := pp[dir]
				err := p.append([]byte(s))
				if err != nil {
					panic(err)
				}
				err = p.process(time.Now(), false)
				if err != nil {
					panic(err)
				}
			}
			dir ^= 1
		}
	}
}

func parserForTests() *parser {
	return &parser{
		config:    &parserConfig{tcp.TCPMaxDataInStream},
		onMessage: func(*message) error { return nil },
		pub:       &transPub{},
	}
}

func parserForParserTests(store *messageStore) *parser {
	p := parserForTests()
	p.state = stateCommand
	p.onMessage = store.publish

	return p
}

func syncerForTests() *syncer {
	s := syncer{}
	s.parsers = [2]*parser{parserForTests(), parserForTests()}

	return &s
}

func modForTests(store *eventStore) *smtpPlugin {
	callback := func(beat.Event) {}
	if store != nil {
		callback = store.publish
	}

	smtp, err := New(false, callback, common.NewConfig())
	if err != nil {
		panic(err)
	}

	return smtp.(*smtpPlugin)
}

func testCreateTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		SrcIP:    net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
		SrcPort: 6512, DstPort: 25,
	}
	t.ComputeHashebles()

	return t
}
