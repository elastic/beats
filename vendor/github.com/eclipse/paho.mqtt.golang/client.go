/*
 * Copyright (c) 2013 IBM Corp.
 *
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *    Seth Hoenig
 *    Allan Stockdill-Mander
 *    Mike Robertson
 */

// Portions copyright © 2018 TIBCO Software Inc.

// Package mqtt provides an MQTT v3.1.1 client library.
package mqtt

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

const (
	disconnected uint32 = iota
	connecting
	reconnecting
	connected
)

// Client is the interface definition for a Client as used by this
// library, the interface is primarily to allow mocking tests.
//
// It is an MQTT v3.1.1 client for communicating
// with an MQTT server using non-blocking methods that allow work
// to be done in the background.
// An application may connect to an MQTT server using:
//   A plain TCP socket
//   A secure SSL/TLS socket
//   A websocket
// To enable ensured message delivery at Quality of Service (QoS) levels
// described in the MQTT spec, a message persistence mechanism must be
// used. This is done by providing a type which implements the Store
// interface. For convenience, FileStore and MemoryStore are provided
// implementations that should be sufficient for most use cases. More
// information can be found in their respective documentation.
// Numerous connection options may be specified by configuring a
// and then supplying a ClientOptions type.
// Implementations of Client must be safe for concurrent use by multiple
// goroutines
type Client interface {
	// IsConnected returns a bool signifying whether
	// the client is connected or not.
	IsConnected() bool
	// IsConnectionOpen return a bool signifying whether the client has an active
	// connection to mqtt broker, i.e not in disconnected or reconnect mode
	IsConnectionOpen() bool
	// Connect will create a connection to the message broker, by default
	// it will attempt to connect at v3.1.1 and auto retry at v3.1 if that
	// fails
	Connect() Token
	// Disconnect will end the connection with the server, but not before waiting
	// the specified number of milliseconds to wait for existing work to be
	// completed.
	Disconnect(quiesce uint)
	// Publish will publish a message with the specified QoS and content
	// to the specified topic.
	// Returns a token to track delivery of the message to the broker
	Publish(topic string, qos byte, retained bool, payload interface{}) Token
	// Subscribe starts a new subscription. Provide a MessageHandler to be executed when
	// a message is published on the topic provided, or nil for the default handler.
	//
	// If options.OrderMatters is true (the default) then callback must not block or
	// call functions within this package that may block (e.g. Publish) other than in
	// a new go routine.
	// callback must be safe for concurrent use by multiple goroutines.
	Subscribe(topic string, qos byte, callback MessageHandler) Token
	// SubscribeMultiple starts a new subscription for multiple topics. Provide a MessageHandler to
	// be executed when a message is published on one of the topics provided, or nil for the
	// default handler.
	//
	// If options.OrderMatters is true (the default) then callback must not block or
	// call functions within this package that may block (e.g. Publish) other than in
	// a new go routine.
	// callback must be safe for concurrent use by multiple goroutines.
	SubscribeMultiple(filters map[string]byte, callback MessageHandler) Token
	// Unsubscribe will end the subscription from each of the topics provided.
	// Messages published to those topics from other clients will no longer be
	// received.
	Unsubscribe(topics ...string) Token
	// AddRoute allows you to add a handler for messages on a specific topic
	// without making a subscription. For example having a different handler
	// for parts of a wildcard subscription or for receiving retained messages
	// upon connection (before Sub scribe can be processed).
	//
	// If options.OrderMatters is true (the default) then callback must not block or
	// call functions within this package that may block (e.g. Publish) other than in
	// a new go routine.
	// callback must be safe for concurrent use by multiple goroutines.
	AddRoute(topic string, callback MessageHandler)
	// OptionsReader returns a ClientOptionsReader which is a copy of the clientoptions
	// in use by the client.
	OptionsReader() ClientOptionsReader
}

// client implements the Client interface
// clients are safe for concurrent use by multiple
// goroutines
type client struct {
	lastSent        atomic.Value // time.Time - the last time a packet was successfully sent to network
	lastReceived    atomic.Value // time.Time - the last time a packet was successfully received from network
	pingOutstanding int32        // set to 1 if a ping has been sent but response not ret received

	status       uint32 // see const definitions at top of file for possible values
	sync.RWMutex        // Protects the above two variables (note: atomic writes are also used somewhat inconsistently)

	messageIds // effectively a map from message id to token completor

	obound    chan *PacketAndToken // outgoing publish packet
	oboundP   chan *PacketAndToken // outgoing 'priority' packet (anything other than publish)
	msgRouter *router              // routes topics to handlers
	persist   Store
	options   ClientOptions
	optionsMu sync.Mutex // Protects the options in a few limited cases where needed for testing

	conn   net.Conn   // the network connection, must only be set with connMu locked (only used when starting/stopping workers)
	connMu sync.Mutex // mutex for the connection (again only used in two functions)

	stop         chan struct{}  // Closed to request that workers stop
	workers      sync.WaitGroup // used to wait for workers to complete (ping, keepalive, errwatch, resume)
	commsStopped chan struct{}  // closed when the comms routines have stopped (kept running until after workers have closed to avoid deadlocks)
}

// NewClient will create an MQTT v3.1.1 client with all of the options specified
// in the provided ClientOptions. The client must have the Connect method called
// on it before it may be used. This is to make sure resources (such as a net
// connection) are created before the application is actually ready.
func NewClient(o *ClientOptions) Client {
	c := &client{}
	c.options = *o

	if c.options.Store == nil {
		c.options.Store = NewMemoryStore()
	}
	switch c.options.ProtocolVersion {
	case 3, 4:
		c.options.protocolVersionExplicit = true
	case 0x83, 0x84:
		c.options.protocolVersionExplicit = true
	default:
		c.options.ProtocolVersion = 4
		c.options.protocolVersionExplicit = false
	}
	c.persist = c.options.Store
	c.status = disconnected
	c.messageIds = messageIds{index: make(map[uint16]tokenCompletor)}
	c.msgRouter = newRouter()
	c.msgRouter.setDefaultHandler(c.options.DefaultPublishHandler)
	c.obound = make(chan *PacketAndToken)
	c.oboundP = make(chan *PacketAndToken)
	return c
}

// AddRoute allows you to add a handler for messages on a specific topic
// without making a subscription. For example having a different handler
// for parts of a wildcard subscription
//
// If options.OrderMatters is true (the default) then callback must not block or
// call functions within this package that may block (e.g. Publish) other than in
// a new go routine.
// callback must be safe for concurrent use by multiple goroutines.
func (c *client) AddRoute(topic string, callback MessageHandler) {
	if callback != nil {
		c.msgRouter.addRoute(topic, callback)
	}
}

// IsConnected returns a bool signifying whether
// the client is connected or not.
// connected means that the connection is up now OR it will
// be established/reestablished automatically when possible
func (c *client) IsConnected() bool {
	c.RLock()
	defer c.RUnlock()
	status := atomic.LoadUint32(&c.status)
	switch {
	case status == connected:
		return true
	case c.options.AutoReconnect && status > connecting:
		return true
	case c.options.ConnectRetry && status == connecting:
		return true
	default:
		return false
	}
}

// IsConnectionOpen return a bool signifying whether the client has an active
// connection to mqtt broker, i.e not in disconnected or reconnect mode
func (c *client) IsConnectionOpen() bool {
	c.RLock()
	defer c.RUnlock()
	status := atomic.LoadUint32(&c.status)
	switch {
	case status == connected:
		return true
	default:
		return false
	}
}

func (c *client) connectionStatus() uint32 {
	c.RLock()
	defer c.RUnlock()
	status := atomic.LoadUint32(&c.status)
	return status
}

func (c *client) setConnected(status uint32) {
	c.Lock()
	defer c.Unlock()
	atomic.StoreUint32(&c.status, status)
}

// ErrNotConnected is the error returned from function calls that are
// made when the client is not connected to a broker
var ErrNotConnected = errors.New("not Connected")

// Connect will create a connection to the message broker, by default
// it will attempt to connect at v3.1.1 and auto retry at v3.1 if that
// fails
// Note: If using QOS1+ and CleanSession=false it is advisable to add
// routes (or a DefaultPublishHandler) prior to calling Connect()
// because queued messages may be delivered immediately post connection
func (c *client) Connect() Token {
	t := newToken(packets.Connect).(*ConnectToken)
	DEBUG.Println(CLI, "Connect()")

	if c.options.ConnectRetry && atomic.LoadUint32(&c.status) != disconnected {
		// if in any state other than disconnected and ConnectRetry is
		// enabled then the connection will come up automatically
		// client can assume connection is up
		WARN.Println(CLI, "Connect() called but not disconnected")
		t.returnCode = packets.Accepted
		t.flowComplete()
		return t
	}

	c.persist.Open()
	if c.options.ConnectRetry {
		c.reserveStoredPublishIDs() // Reserve IDs to allow publish before connect complete
	}
	c.setConnected(connecting)

	go func() {
		if len(c.options.Servers) == 0 {
			t.setError(fmt.Errorf("no servers defined to connect to"))
			return
		}

	RETRYCONN:
		var conn net.Conn
		var rc byte
		var err error
		conn, rc, t.sessionPresent, err = c.attemptConnection()
		if err != nil {
			if c.options.ConnectRetry {
				DEBUG.Println(CLI, "Connect failed, sleeping for", int(c.options.ConnectRetryInterval.Seconds()), "seconds and will then retry")
				time.Sleep(c.options.ConnectRetryInterval)

				if atomic.LoadUint32(&c.status) == connecting {
					goto RETRYCONN
				}
			}
			ERROR.Println(CLI, "Failed to connect to a broker")
			c.setConnected(disconnected)
			c.persist.Close()
			t.returnCode = rc
			t.setError(err)
			return
		}
		inboundFromStore := make(chan packets.ControlPacket) // there may be some inbound comms packets in the store that are awaiting processing
		if c.startCommsWorkers(conn, inboundFromStore) {
			// Take care of any messages in the store
			if !c.options.CleanSession {
				c.resume(c.options.ResumeSubs, inboundFromStore)
			} else {
				c.persist.Reset()
			}
		} else {
			WARN.Println(CLI, "Connect() called but connection established in another goroutine")
		}

		close(inboundFromStore)
		t.flowComplete()
		DEBUG.Println(CLI, "exit startClient")
	}()
	return t
}

// internal function used to reconnect the client when it loses its connection
func (c *client) reconnect() {
	DEBUG.Println(CLI, "enter reconnect")
	var (
		sleep = 1 * time.Second
		conn  net.Conn
	)

	for {
		if nil != c.options.OnReconnecting {
			c.options.OnReconnecting(c, &c.options)
		}
		var err error
		conn, _, _, err = c.attemptConnection()
		if err == nil {
			break
		}
		DEBUG.Println(CLI, "Reconnect failed, sleeping for", int(sleep.Seconds()), "seconds:", err)
		time.Sleep(sleep)
		if sleep < c.options.MaxReconnectInterval {
			sleep *= 2
		}

		if sleep > c.options.MaxReconnectInterval {
			sleep = c.options.MaxReconnectInterval
		}
		// Disconnect may have been called
		if atomic.LoadUint32(&c.status) == disconnected {
			break
		}
	}

	// Disconnect() must have been called while we were trying to reconnect.
	if c.connectionStatus() == disconnected {
		if conn != nil {
			conn.Close()
		}
		DEBUG.Println(CLI, "Client moved to disconnected state while reconnecting, abandoning reconnect")
		return
	}

	inboundFromStore := make(chan packets.ControlPacket) // there may be some inbound comms packets in the store that are awaiting processing
	if c.startCommsWorkers(conn, inboundFromStore) {
		c.resume(c.options.ResumeSubs, inboundFromStore)
	}
	close(inboundFromStore)
}

// attemptConnection makes a single attempt to connect to each of the brokers
// the protocol version to use is passed in (as c.options.ProtocolVersion)
// Note: Does not set c.conn in order to minimise race conditions
// Returns:
// net.Conn - Connected network connection
// byte - Return code (packets.Accepted indicates a successful connection).
// bool - SessionPresent flag from the connect ack (only valid if packets.Accepted)
// err - Error (err != nil guarantees that conn has been set to active connection).
func (c *client) attemptConnection() (net.Conn, byte, bool, error) {
	protocolVersion := c.options.ProtocolVersion
	var (
		sessionPresent bool
		conn           net.Conn
		err            error
		rc             byte
	)

	c.optionsMu.Lock() // Protect c.options.Servers so that servers can be added in test cases
	brokers := c.options.Servers
	c.optionsMu.Unlock()
	for _, broker := range brokers {
		cm := newConnectMsgFromOptions(&c.options, broker)
		DEBUG.Println(CLI, "about to write new connect msg")
	CONN:
		tlsCfg := c.options.TLSConfig
		if c.options.OnConnectAttempt != nil {
			DEBUG.Println(CLI, "using custom onConnectAttempt handler...")
			tlsCfg = c.options.OnConnectAttempt(broker, c.options.TLSConfig)
		}
		// Start by opening the network connection (tcp, tls, ws) etc
		conn, err = openConnection(broker, tlsCfg, c.options.ConnectTimeout, c.options.HTTPHeaders, c.options.WebsocketOptions)
		if err != nil {
			ERROR.Println(CLI, err.Error())
			WARN.Println(CLI, "failed to connect to broker, trying next")
			rc = packets.ErrNetworkError
			continue
		}
		DEBUG.Println(CLI, "socket connected to broker")

		// Now we send the perform the MQTT connection handshake
		rc, sessionPresent, err = connectMQTT(conn, cm, protocolVersion)
		if rc == packets.Accepted {
			break // successfully connected
		}

		// We may be have to attempt the connection with MQTT 3.1
		if conn != nil {
			_ = conn.Close()
		}
		if !c.options.protocolVersionExplicit && protocolVersion == 4 { // try falling back to 3.1?
			DEBUG.Println(CLI, "Trying reconnect using MQTT 3.1 protocol")
			protocolVersion = 3
			goto CONN
		}
		if c.options.protocolVersionExplicit { // to maintain logging from previous version
			ERROR.Println(CLI, "Connecting to", broker, "CONNACK was not CONN_ACCEPTED, but rather", packets.ConnackReturnCodes[rc])
		}
	}
	// If the connection was successful we set member variable and lock in the protocol version for future connection attempts (and users)
	if rc == packets.Accepted {
		c.options.ProtocolVersion = protocolVersion
		c.options.protocolVersionExplicit = true
	} else {
		// Maintain same error format as used previously
		if rc != packets.ErrNetworkError { // mqtt error
			err = packets.ConnErrors[rc]
		} else { // network error (if this occurred in ConnectMQTT then err will be nil)
			err = fmt.Errorf("%s : %s", packets.ConnErrors[rc], err)
		}
	}
	return conn, rc, sessionPresent, err
}

// Disconnect will end the connection with the server, but not before waiting
// the specified number of milliseconds to wait for existing work to be
// completed.
func (c *client) Disconnect(quiesce uint) {
	status := atomic.LoadUint32(&c.status)
	if status == connected {
		DEBUG.Println(CLI, "disconnecting")
		c.setConnected(disconnected)

		dm := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
		dt := newToken(packets.Disconnect)
		disconnectSent := false
		select {
		case c.oboundP <- &PacketAndToken{p: dm, t: dt}:
			disconnectSent = true
		case <-c.commsStopped:
			WARN.Println("Disconnect packet could not be sent because comms stopped")
		case <-time.After(time.Duration(quiesce) * time.Millisecond):
			WARN.Println("Disconnect packet not sent due to timeout")
		}

		// wait for work to finish, or quiesce time consumed
		if disconnectSent {
			DEBUG.Println(CLI, "calling WaitTimeout")
			dt.WaitTimeout(time.Duration(quiesce) * time.Millisecond)
			DEBUG.Println(CLI, "WaitTimeout done")
		}
	} else {
		WARN.Println(CLI, "Disconnect() called but not connected (disconnected/reconnecting)")
		c.setConnected(disconnected)
	}

	c.disconnect()
}

// forceDisconnect will end the connection with the mqtt broker immediately (used for tests only)
func (c *client) forceDisconnect() {
	if !c.IsConnected() {
		WARN.Println(CLI, "already disconnected")
		return
	}
	c.setConnected(disconnected)
	DEBUG.Println(CLI, "forcefully disconnecting")
	c.disconnect()
}

// disconnect cleans up after a final disconnection (user requested so no auto reconnection)
func (c *client) disconnect() {
	done := c.stopCommsWorkers()
	if done != nil {
		<-done // Wait until the disconnect is complete (to limit chance that another connection will be started)
		DEBUG.Println(CLI, "forcefully disconnecting")
		c.messageIds.cleanUp()
		DEBUG.Println(CLI, "disconnected")
		c.persist.Close()
	}
}

// internalConnLost cleanup when connection is lost or an error occurs
// Note: This function will not block
func (c *client) internalConnLost(err error) {
	// It is possible that internalConnLost will be called multiple times simultaneously
	// (including after sending a DisconnectPacket) as such we only do cleanup etc if the
	// routines were actually running and are not being disconnected at users request
	DEBUG.Println(CLI, "internalConnLost called")
	stopDone := c.stopCommsWorkers()
	if stopDone != nil { // stopDone will be nil if workers already in the process of stopping or stopped
		go func() {
			DEBUG.Println(CLI, "internalConnLost waiting on workers")
			<-stopDone
			DEBUG.Println(CLI, "internalConnLost workers stopped")
			// It is possible that Disconnect was called which led to this error so reconnection depends upon status
			reconnect := c.options.AutoReconnect && c.connectionStatus() > connecting

			if c.options.CleanSession && !reconnect {
				c.messageIds.cleanUp()
			}
			if reconnect {
				c.setConnected(reconnecting)
				go c.reconnect()
			} else {
				c.setConnected(disconnected)
			}
			if c.options.OnConnectionLost != nil {
				go c.options.OnConnectionLost(c, err)
			}
			DEBUG.Println(CLI, "internalConnLost complete")
		}()
	}
}

// startCommsWorkers is called when the connection is up.
// It starts off all of the routines needed to process incoming and outgoing messages.
// Returns true if the comms workers were started (i.e. they were not already running)
func (c *client) startCommsWorkers(conn net.Conn, inboundFromStore <-chan packets.ControlPacket) bool {
	DEBUG.Println(CLI, "startCommsWorkers called")
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		WARN.Println(CLI, "startCommsWorkers called when commsworkers already running")
		conn.Close() // No use for the new network connection
		return false
	}
	c.conn = conn // Store the connection

	c.stop = make(chan struct{})
	if c.options.KeepAlive != 0 {
		atomic.StoreInt32(&c.pingOutstanding, 0)
		c.lastReceived.Store(time.Now())
		c.lastSent.Store(time.Now())
		c.workers.Add(1)
		go keepalive(c, conn)
	}

	// matchAndDispatch will process messages received from the network. It may generate acknowledgements
	// It will complete when incomingPubChan is closed and will close ackOut prior to exiting
	incomingPubChan := make(chan *packets.PublishPacket)
	c.workers.Add(1) // Done will be called when ackOut is closed
	ackOut := c.msgRouter.matchAndDispatch(incomingPubChan, c.options.Order, c)

	c.setConnected(connected)
	DEBUG.Println(CLI, "client is connected/reconnected")
	if c.options.OnConnect != nil {
		go c.options.OnConnect(c)
	}

	// c.oboundP and c.obound need to stay active for the life of the client because, depending upon the options,
	// messages may be published while the client is disconnected (they will block unless in a goroutine). However
	// to keep the comms routines clean we want to shutdown the input messages it uses so create out own channels
	// and copy data across.
	commsobound := make(chan *PacketAndToken)  // outgoing publish packets
	commsoboundP := make(chan *PacketAndToken) // outgoing 'priority' packet
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		for {
			select {
			case msg := <-c.oboundP:
				commsoboundP <- msg
			case msg := <-c.obound:
				commsobound <- msg
			case msg, ok := <-ackOut:
				if !ok {
					ackOut = nil     // ignore channel going forward
					c.workers.Done() // matchAndDispatch has completed
					continue         // await next message
				}
				commsoboundP <- msg
			case <-c.stop:
				// Attempt to transmit any outstanding acknowledgements (this may well fail but should work if this is a clean disconnect)
				if ackOut != nil {
					for msg := range ackOut {
						commsoboundP <- msg
					}
					c.workers.Done() // matchAndDispatch has completed
				}
				close(commsoboundP) // Nothing sending to these channels anymore so close them and allow comms routines to exit
				close(commsobound)
				DEBUG.Println(CLI, "startCommsWorkers output redirector finished")
				return
			}
		}
	}()

	commsIncomingPub, commsErrors := startComms(c.conn, c, inboundFromStore, commsoboundP, commsobound)
	c.commsStopped = make(chan struct{})
	go func() {
		for {
			if commsIncomingPub == nil && commsErrors == nil {
				break
			}
			select {
			case pub, ok := <-commsIncomingPub:
				if !ok {
					// Incoming comms has shutdown
					close(incomingPubChan) // stop the router
					commsIncomingPub = nil
					continue
				}
				// Care is needed here because an error elsewhere could trigger a deadlock
			sendPubLoop:
				for {
					select {
					case incomingPubChan <- pub:
						break sendPubLoop
					case err, ok := <-commsErrors:
						if !ok { // commsErrors has been closed so we can ignore it
							commsErrors = nil
							continue
						}
						ERROR.Println(CLI, "Connect comms goroutine - error triggered during send Pub", err)
						c.internalConnLost(err) // no harm in calling this if the connection is already down (or shutdown is in progress)
						continue
					}
				}
			case err, ok := <-commsErrors:
				if !ok {
					commsErrors = nil
					continue
				}
				ERROR.Println(CLI, "Connect comms goroutine - error triggered", err)
				c.internalConnLost(err) // no harm in calling this if the connection is already down (or shutdown is in progress)
				continue
			}
		}
		DEBUG.Println(CLI, "incoming comms goroutine done")
		close(c.commsStopped)
	}()
	DEBUG.Println(CLI, "startCommsWorkers done")
	return true
}

// stopWorkersAndComms - Cleanly shuts down worker go routines (including the comms routines) and waits until everything has stopped
// Returns nil it workers did not need to be stopped; otherwise returns a channel which will be closed when the stop is complete
// Note: This may block so run as a go routine if calling from any of the comms routines
func (c *client) stopCommsWorkers() chan struct{} {
	DEBUG.Println(CLI, "stopCommsWorkers called")
	// It is possible that this function will be called multiple times simultaneously due to the way things get shutdown
	c.connMu.Lock()
	if c.conn == nil {
		DEBUG.Println(CLI, "stopCommsWorkers done (not running)")
		c.connMu.Unlock()
		return nil
	}

	// It is important that everything is stopped in the correct order to avoid deadlocks. The main issue here is
	// the router because it both receives incoming publish messages and also sends outgoing acknowledgements. To
	// avoid issues we signal the workers to stop and close the connection (it is probably already closed but
	// there is no harm in being sure). We can then wait for the workers to finnish before closing outbound comms
	// channels which will allow the comms routines to exit.

	// We stop all non-comms related workers first (ping, keepalive, errwatch, resume etc) so they don't get blocked waiting on comms
	close(c.stop)     // Signal for workers to stop
	c.conn.Close()    // Possible that this is already closed but no harm in closing again
	c.conn = nil      // Important that this is the only place that this is set to nil
	c.connMu.Unlock() // As the connection is now nil we can unlock the mu (allowing subsequent calls to exit immediately)

	doneChan := make(chan struct{})

	go func() {
		DEBUG.Println(CLI, "stopCommsWorkers waiting for workers")
		c.workers.Wait()

		// Stopping the workers will allow the comms routines to exit; we wait for these to complete
		DEBUG.Println(CLI, "stopCommsWorkers waiting for comms")
		<-c.commsStopped // wait for comms routine to stop

		DEBUG.Println(CLI, "stopCommsWorkers done")
		close(doneChan)
	}()
	return doneChan
}

// Publish will publish a message with the specified QoS and content
// to the specified topic.
// Returns a token to track delivery of the message to the broker
func (c *client) Publish(topic string, qos byte, retained bool, payload interface{}) Token {
	token := newToken(packets.Publish).(*PublishToken)
	DEBUG.Println(CLI, "enter Publish")
	switch {
	case !c.IsConnected():
		token.setError(ErrNotConnected)
		return token
	case c.connectionStatus() == reconnecting && qos == 0:
		token.flowComplete()
		return token
	}
	pub := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
	pub.Qos = qos
	pub.TopicName = topic
	pub.Retain = retained
	switch p := payload.(type) {
	case string:
		pub.Payload = []byte(p)
	case []byte:
		pub.Payload = p
	case bytes.Buffer:
		pub.Payload = p.Bytes()
	default:
		token.setError(fmt.Errorf("unknown payload type"))
		return token
	}

	if pub.Qos != 0 && pub.MessageID == 0 {
		mID := c.getID(token)
		if mID == 0 {
			token.setError(fmt.Errorf("no message IDs available"))
			return token
		}
		pub.MessageID = mID
		token.messageID = mID
	}
	persistOutbound(c.persist, pub)
	switch c.connectionStatus() {
	case connecting:
		DEBUG.Println(CLI, "storing publish message (connecting), topic:", topic)
	case reconnecting:
		DEBUG.Println(CLI, "storing publish message (reconnecting), topic:", topic)
	default:
		DEBUG.Println(CLI, "sending publish message, topic:", topic)
		publishWaitTimeout := c.options.WriteTimeout
		if publishWaitTimeout == 0 {
			publishWaitTimeout = time.Second * 30
		}
		select {
		case c.obound <- &PacketAndToken{p: pub, t: token}:
		case <-time.After(publishWaitTimeout):
			token.setError(errors.New("publish was broken by timeout"))
		}
	}
	return token
}

// Subscribe starts a new subscription. Provide a MessageHandler to be executed when
// a message is published on the topic provided.
//
// If options.OrderMatters is true (the default) then callback must not block or
// call functions within this package that may block (e.g. Publish) other than in
// a new go routine.
// callback must be safe for concurrent use by multiple goroutines.
func (c *client) Subscribe(topic string, qos byte, callback MessageHandler) Token {
	token := newToken(packets.Subscribe).(*SubscribeToken)
	DEBUG.Println(CLI, "enter Subscribe")
	if !c.IsConnected() {
		token.setError(ErrNotConnected)
		return token
	}
	if !c.IsConnectionOpen() {
		switch {
		case !c.options.ResumeSubs:
			// if not connected and resumesubs not set this sub will be thrown away
			token.setError(fmt.Errorf("not currently connected and ResumeSubs not set"))
			return token
		case c.options.CleanSession && c.connectionStatus() == reconnecting:
			// if reconnecting and cleansession is true this sub will be thrown away
			token.setError(fmt.Errorf("reconnecting state and cleansession is true"))
			return token
		}
	}
	sub := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
	if err := validateTopicAndQos(topic, qos); err != nil {
		token.setError(err)
		return token
	}
	sub.Topics = append(sub.Topics, topic)
	sub.Qoss = append(sub.Qoss, qos)

	if strings.HasPrefix(topic, "$share/") {
		topic = strings.Join(strings.Split(topic, "/")[2:], "/")
	}

	if strings.HasPrefix(topic, "$queue/") {
		topic = strings.TrimPrefix(topic, "$queue/")
	}

	if callback != nil {
		c.msgRouter.addRoute(topic, callback)
	}

	token.subs = append(token.subs, topic)

	if sub.MessageID == 0 {
		mID := c.getID(token)
		if mID == 0 {
			token.setError(fmt.Errorf("no message IDs available"))
			return token
		}
		sub.MessageID = mID
		token.messageID = mID
	}
	DEBUG.Println(CLI, sub.String())

	persistOutbound(c.persist, sub)
	switch c.connectionStatus() {
	case connecting:
		DEBUG.Println(CLI, "storing subscribe message (connecting), topic:", topic)
	case reconnecting:
		DEBUG.Println(CLI, "storing subscribe message (reconnecting), topic:", topic)
	default:
		DEBUG.Println(CLI, "sending subscribe message, topic:", topic)
		subscribeWaitTimeout := c.options.WriteTimeout
		if subscribeWaitTimeout == 0 {
			subscribeWaitTimeout = time.Second * 30
		}
		select {
		case c.oboundP <- &PacketAndToken{p: sub, t: token}:
		case <-time.After(subscribeWaitTimeout):
			token.setError(errors.New("subscribe was broken by timeout"))
		}
	}
	DEBUG.Println(CLI, "exit Subscribe")
	return token
}

// SubscribeMultiple starts a new subscription for multiple topics. Provide a MessageHandler to
// be executed when a message is published on one of the topics provided.
//
// If options.OrderMatters is true (the default) then callback must not block or
// call functions within this package that may block (e.g. Publish) other than in
// a new go routine.
// callback must be safe for concurrent use by multiple goroutines.
func (c *client) SubscribeMultiple(filters map[string]byte, callback MessageHandler) Token {
	var err error
	token := newToken(packets.Subscribe).(*SubscribeToken)
	DEBUG.Println(CLI, "enter SubscribeMultiple")
	if !c.IsConnected() {
		token.setError(ErrNotConnected)
		return token
	}
	if !c.IsConnectionOpen() {
		switch {
		case !c.options.ResumeSubs:
			// if not connected and resumesubs not set this sub will be thrown away
			token.setError(fmt.Errorf("not currently connected and ResumeSubs not set"))
			return token
		case c.options.CleanSession && c.connectionStatus() == reconnecting:
			// if reconnecting and cleansession is true this sub will be thrown away
			token.setError(fmt.Errorf("reconnecting state and cleansession is true"))
			return token
		}
	}
	sub := packets.NewControlPacket(packets.Subscribe).(*packets.SubscribePacket)
	if sub.Topics, sub.Qoss, err = validateSubscribeMap(filters); err != nil {
		token.setError(err)
		return token
	}

	if callback != nil {
		for topic := range filters {
			c.msgRouter.addRoute(topic, callback)
		}
	}
	token.subs = make([]string, len(sub.Topics))
	copy(token.subs, sub.Topics)

	if sub.MessageID == 0 {
		mID := c.getID(token)
		if mID == 0 {
			token.setError(fmt.Errorf("no message IDs available"))
			return token
		}
		sub.MessageID = mID
		token.messageID = mID
	}
	persistOutbound(c.persist, sub)
	switch c.connectionStatus() {
	case connecting:
		DEBUG.Println(CLI, "storing subscribe message (connecting), topics:", sub.Topics)
	case reconnecting:
		DEBUG.Println(CLI, "storing subscribe message (reconnecting), topics:", sub.Topics)
	default:
		DEBUG.Println(CLI, "sending subscribe message, topics:", sub.Topics)
		subscribeWaitTimeout := c.options.WriteTimeout
		if subscribeWaitTimeout == 0 {
			subscribeWaitTimeout = time.Second * 30
		}
		select {
		case c.oboundP <- &PacketAndToken{p: sub, t: token}:
		case <-time.After(subscribeWaitTimeout):
			token.setError(errors.New("subscribe was broken by timeout"))
		}
	}
	DEBUG.Println(CLI, "exit SubscribeMultiple")
	return token
}

// reserveStoredPublishIDs reserves the ids for publish packets in the persistent store to ensure these are not duplicated
func (c *client) reserveStoredPublishIDs() {
	// The resume function sets the stored id for publish packets only (some other packets
	// will get new ids in net code). This means that the only keys we need to ensure are
	// unique are the publish ones (and these will completed/replaced in resume() )
	if !c.options.CleanSession {
		storedKeys := c.persist.All()
		for _, key := range storedKeys {
			packet := c.persist.Get(key)
			if packet == nil {
				continue
			}
			switch packet.(type) {
			case *packets.PublishPacket:
				details := packet.Details()
				token := &PlaceHolderToken{id: details.MessageID}
				c.claimID(token, details.MessageID)
			}
		}
	}
}

// Load all stored messages and resend them
// Call this to ensure QOS > 1,2 even after an application crash
// Note: This function will exit if c.stop is closed (this allows the shutdown to proceed avoiding a potential deadlock)
//
func (c *client) resume(subscription bool, ibound chan packets.ControlPacket) {
	DEBUG.Println(STR, "enter Resume")

	storedKeys := c.persist.All()
	for _, key := range storedKeys {
		packet := c.persist.Get(key)
		if packet == nil {
			DEBUG.Println(STR, fmt.Sprintf("resume found NIL packet (%s)", key))
			continue
		}
		details := packet.Details()
		if isKeyOutbound(key) {
			switch p := packet.(type) {
			case *packets.SubscribePacket:
				if subscription {
					DEBUG.Println(STR, fmt.Sprintf("loaded pending subscribe (%d)", details.MessageID))
					subPacket := packet.(*packets.SubscribePacket)
					token := newToken(packets.Subscribe).(*SubscribeToken)
					token.messageID = details.MessageID
					token.subs = append(token.subs, subPacket.Topics...)
					c.claimID(token, details.MessageID)
					select {
					case c.oboundP <- &PacketAndToken{p: packet, t: token}:
					case <-c.stop:
						DEBUG.Println(STR, "resume exiting due to stop")
						return
					}
				} else {
					c.persist.Del(key) // Unsubscribe packets should not be retained following a reconnect
				}
			case *packets.UnsubscribePacket:
				if subscription {
					DEBUG.Println(STR, fmt.Sprintf("loaded pending unsubscribe (%d)", details.MessageID))
					token := newToken(packets.Unsubscribe).(*UnsubscribeToken)
					select {
					case c.oboundP <- &PacketAndToken{p: packet, t: token}:
					case <-c.stop:
						DEBUG.Println(STR, "resume exiting due to stop")
						return
					}
				} else {
					c.persist.Del(key) // Unsubscribe packets should not be retained following a reconnect
				}
			case *packets.PubrelPacket:
				DEBUG.Println(STR, fmt.Sprintf("loaded pending pubrel (%d)", details.MessageID))
				select {
				case c.oboundP <- &PacketAndToken{p: packet, t: nil}:
				case <-c.stop:
					DEBUG.Println(STR, "resume exiting due to stop")
					return
				}
			case *packets.PublishPacket:
				// spec: If the DUP flag is set to 0, it indicates that this is the first occasion that the Client or
				// Server has attempted to send this MQTT PUBLISH Packet. If the DUP flag is set to 1, it indicates that
				// this might be re-delivery of an earlier attempt to send the Packet.
				//
				// If the message is in the store than an attempt at delivery has been made (note that the message may
				// never have made it onto the wire but tracking that would be complicated!).
				if p.Qos != 0 { // spec: The DUP flag MUST be set to 0 for all QoS 0 messages
					p.Dup = true
				}
				token := newToken(packets.Publish).(*PublishToken)
				token.messageID = details.MessageID
				c.claimID(token, details.MessageID)
				DEBUG.Println(STR, fmt.Sprintf("loaded pending publish (%d)", details.MessageID))
				DEBUG.Println(STR, details)
				select {
				case c.obound <- &PacketAndToken{p: p, t: token}:
				case <-c.stop:
					DEBUG.Println(STR, "resume exiting due to stop")
					return
				}
			default:
				ERROR.Println(STR, "invalid message type in store (discarded)")
				c.persist.Del(key)
			}
		} else {
			switch packet.(type) {
			case *packets.PubrelPacket:
				DEBUG.Println(STR, fmt.Sprintf("loaded pending incomming (%d)", details.MessageID))
				select {
				case ibound <- packet:
				case <-c.stop:
					DEBUG.Println(STR, "resume exiting due to stop (ibound <- packet)")
					return
				}
			default:
				ERROR.Println(STR, "invalid message type in store (discarded)")
				c.persist.Del(key)
			}
		}
	}
	DEBUG.Println(STR, "exit resume")
}

// Unsubscribe will end the subscription from each of the topics provided.
// Messages published to those topics from other clients will no longer be
// received.
func (c *client) Unsubscribe(topics ...string) Token {
	token := newToken(packets.Unsubscribe).(*UnsubscribeToken)
	DEBUG.Println(CLI, "enter Unsubscribe")
	if !c.IsConnected() {
		token.setError(ErrNotConnected)
		return token
	}
	if !c.IsConnectionOpen() {
		switch {
		case !c.options.ResumeSubs:
			// if not connected and resumesubs not set this unsub will be thrown away
			token.setError(fmt.Errorf("not currently connected and ResumeSubs not set"))
			return token
		case c.options.CleanSession && c.connectionStatus() == reconnecting:
			// if reconnecting and cleansession is true this unsub will be thrown away
			token.setError(fmt.Errorf("reconnecting state and cleansession is true"))
			return token
		}
	}
	unsub := packets.NewControlPacket(packets.Unsubscribe).(*packets.UnsubscribePacket)
	unsub.Topics = make([]string, len(topics))
	copy(unsub.Topics, topics)

	if unsub.MessageID == 0 {
		mID := c.getID(token)
		if mID == 0 {
			token.setError(fmt.Errorf("no message IDs available"))
			return token
		}
		unsub.MessageID = mID
		token.messageID = mID
	}

	persistOutbound(c.persist, unsub)

	switch c.connectionStatus() {
	case connecting:
		DEBUG.Println(CLI, "storing unsubscribe message (connecting), topics:", topics)
	case reconnecting:
		DEBUG.Println(CLI, "storing unsubscribe message (reconnecting), topics:", topics)
	default:
		DEBUG.Println(CLI, "sending unsubscribe message, topics:", topics)
		subscribeWaitTimeout := c.options.WriteTimeout
		if subscribeWaitTimeout == 0 {
			subscribeWaitTimeout = time.Second * 30
		}
		select {
		case c.oboundP <- &PacketAndToken{p: unsub, t: token}:
			for _, topic := range topics {
				c.msgRouter.deleteRoute(topic)
			}
		case <-time.After(subscribeWaitTimeout):
			token.setError(errors.New("unsubscribe was broken by timeout"))
		}
	}

	DEBUG.Println(CLI, "exit Unsubscribe")
	return token
}

// OptionsReader returns a ClientOptionsReader which is a copy of the clientoptions
// in use by the client.
func (c *client) OptionsReader() ClientOptionsReader {
	r := ClientOptionsReader{options: &c.options}
	return r
}

// DefaultConnectionLostHandler is a definition of a function that simply
// reports to the DEBUG log the reason for the client losing a connection.
func DefaultConnectionLostHandler(client Client, reason error) {
	DEBUG.Println("Connection lost:", reason.Error())
}

// UpdateLastReceived - Will be called whenever a packet is received off the network
// This is used by the keepalive routine to
func (c *client) UpdateLastReceived() {
	if c.options.KeepAlive != 0 {
		c.lastReceived.Store(time.Now())
	}
}

// UpdateLastReceived - Will be called whenever a packet is successfully transmitted to the network
func (c *client) UpdateLastSent() {
	if c.options.KeepAlive != 0 {
		c.lastSent.Store(time.Now())
	}
}

// getWriteTimeOut returns the writetimeout (duration to wait when writing to the connection) or 0 if none
func (c *client) getWriteTimeOut() time.Duration {
	return c.options.WriteTimeout
}

// persistOutbound adds the packet to the outbound store
func (c *client) persistOutbound(m packets.ControlPacket) {
	persistOutbound(c.persist, m)
}

// persistInbound adds the packet to the inbound store
func (c *client) persistInbound(m packets.ControlPacket) {
	persistInbound(c.persist, m)
}

// pingRespReceived will be called by the network routines when a ping response is received
func (c *client) pingRespReceived() {
	atomic.StoreInt32(&c.pingOutstanding, 0)
}
