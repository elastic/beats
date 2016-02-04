package logstash

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type testSyncDriver struct {
	client  mode.ProtocolClient
	ch      chan testDriverCommand
	returns []testClientReturn
	wg      sync.WaitGroup
}

type clientServer struct {
	*mockServer
}

func TestClientSendZero(t *testing.T) {
	testSendZero(t, makeTestClient)
}

func TestClientSimpleEvent(t *testing.T) {
	testSimpleEvent(t, makeTestClient)
}

func TestClientStructuredEvent(t *testing.T) {
	testStructuredEvent(t, makeTestClient)
}

func TestClientCloseAfterWindowSize(t *testing.T) {
	testCloseAfterWindowSize(t, makeTestClient)
}

func newClientServerTCP(t *testing.T, to time.Duration) *clientServer {
	return &clientServer{newMockServerTCP(t, to, "")}
}

func (s *clientServer) connectPair(compressLevel int) (*mockConn, *client, error) {
	client, transp, err := s.mockServer.connectPair(100 * time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	lc, err := newLumberjackClient(transp, compressLevel,
		defaultMaxWindowSize, 100*time.Millisecond)
	if err != nil {
		return nil, nil, err
	}

	conn := &mockConn{client, streambuf.New(nil)}
	return conn, lc, nil
}

func makeTestClient(conn TransportClient) testClientDriver {
	return newClientTestDriver(newLumberjackTestClient(conn))
}

func newClientTestDriver(client mode.ProtocolClient) *testSyncDriver {
	driver := &testSyncDriver{
		client:  client,
		ch:      make(chan testDriverCommand),
		returns: nil,
	}

	driver.wg.Add(1)
	go func() {
		defer driver.wg.Done()

		for {
			cmd, ok := <-driver.ch
			if !ok {
				return
			}

			switch cmd.code {
			case driverCmdQuit:
				return
			case driverCmdPublish:
				events, err := driver.client.PublishEvents(cmd.events)
				n := len(cmd.events) - len(events)
				driver.returns = append(driver.returns, testClientReturn{n, err})
			}
		}
	}()

	return driver
}

func (t *testSyncDriver) Stop() {
	t.ch <- testDriverCommand{code: driverCmdQuit}
	t.wg.Wait()
	close(t.ch)
	t.client.Close()
}

func (t *testSyncDriver) Publish(events []common.MapStr) {
	t.ch <- testDriverCommand{code: driverCmdPublish, events: events}
}

func (t *testSyncDriver) Returns() []testClientReturn {
	return t.returns
}
