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

package mqtt

import (
	"errors"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

const closedNetConnErrorText = "use of closed network connection" // error string for closed conn (https://golang.org/src/net/error_test.go)

// ConnectMQTT takes a connected net.Conn and performs the initial MQTT handshake. Parameters are:
// conn - Connected net.Conn
// cm - Connect Packet with everything other than the protocol name/version populated (historical reasons)
// protocolVersion - The protocol version to attempt to connect with
//
// Note that, for backward compatibility, ConnectMQTT() suppresses the actual connection error (compare to connectMQTT()).
func ConnectMQTT(conn net.Conn, cm *packets.ConnectPacket, protocolVersion uint) (byte, bool) {
	rc, sessionPresent, _ := connectMQTT(conn, cm, protocolVersion)
	return rc, sessionPresent
}

func connectMQTT(conn io.ReadWriter, cm *packets.ConnectPacket, protocolVersion uint) (byte, bool, error) {
	switch protocolVersion {
	case 3:
		DEBUG.Println(CLI, "Using MQTT 3.1 protocol")
		cm.ProtocolName = "MQIsdp"
		cm.ProtocolVersion = 3
	case 0x83:
		DEBUG.Println(CLI, "Using MQTT 3.1b protocol")
		cm.ProtocolName = "MQIsdp"
		cm.ProtocolVersion = 0x83
	case 0x84:
		DEBUG.Println(CLI, "Using MQTT 3.1.1b protocol")
		cm.ProtocolName = "MQTT"
		cm.ProtocolVersion = 0x84
	default:
		DEBUG.Println(CLI, "Using MQTT 3.1.1 protocol")
		cm.ProtocolName = "MQTT"
		cm.ProtocolVersion = 4
	}

	if err := cm.Write(conn); err != nil {
		ERROR.Println(CLI, err)
		return packets.ErrNetworkError, false, err
	}

	rc, sessionPresent, err := verifyCONNACK(conn)
	return rc, sessionPresent, err
}

// This function is only used for receiving a connack
// when the connection is first started.
// This prevents receiving incoming data while resume
// is in progress if clean session is false.
func verifyCONNACK(conn io.Reader) (byte, bool, error) {
	DEBUG.Println(NET, "connect started")

	ca, err := packets.ReadPacket(conn)
	if err != nil {
		ERROR.Println(NET, "connect got error", err)
		return packets.ErrNetworkError, false, err
	}

	if ca == nil {
		ERROR.Println(NET, "received nil packet")
		return packets.ErrNetworkError, false, errors.New("nil CONNACK packet")
	}

	msg, ok := ca.(*packets.ConnackPacket)
	if !ok {
		ERROR.Println(NET, "received msg that was not CONNACK")
		return packets.ErrNetworkError, false, errors.New("non-CONNACK first packet received")
	}

	DEBUG.Println(NET, "received connack")
	return msg.ReturnCode, msg.SessionPresent, nil
}

// inbound encapsulates the output from startIncoming.
// err  - If != nil then an error has occurred
// cp - A control packet received over the network link
type inbound struct {
	err error
	cp  packets.ControlPacket
}

// startIncoming initiates a goroutine that reads incoming messages off the wire and sends them to the channel (returned).
// If there are any issues with the network connection then the returned channel will be closed and the goroutine will exit
// (so closing the connection will terminate the goroutine)
func startIncoming(conn io.Reader) <-chan inbound {
	var err error
	var cp packets.ControlPacket
	ibound := make(chan inbound)

	DEBUG.Println(NET, "incoming started")

	go func() {
		for {
			if cp, err = packets.ReadPacket(conn); err != nil {
				// We do not want to log the error if it is due to the network connection having been closed
				// elsewhere (i.e. after sending DisconnectPacket). Detecting this situation is the subject of
				// https://github.com/golang/go/issues/4373
				if !strings.Contains(err.Error(), closedNetConnErrorText) {
					ibound <- inbound{err: err}
				}
				close(ibound)
				DEBUG.Println(NET, "incoming complete")
				return
			}
			DEBUG.Println(NET, "startIncoming Received Message")
			ibound <- inbound{cp: cp}
		}
	}()

	return ibound
}

// incomingComms encapsulates the possible output of the incomingComms routine. If err != nil then an error has occurred and
// the routine will have terminated; otherwise one of the other members should be non-nil
type incomingComms struct {
	err         error                  // If non-nil then there has been an error (ignore everything else)
	outbound    *PacketAndToken        // Packet (with token) than needs to be sent out (e.g. an acknowledgement)
	incomingPub *packets.PublishPacket // A new publish has been received; this will need to be passed on to our user
}

// startIncomingComms initiates incoming communications; this includes starting a goroutine to process incoming
// messages.
// Accepts a channel of inbound messages from the store (persisted messages); note this must be closed as soon as the
// everything in the store has been sent.
// Returns a channel that will be passed any received packets; this will be closed on a network error (and inboundFromStore closed)
func startIncomingComms(conn io.Reader,
	c commsFns,
	inboundFromStore <-chan packets.ControlPacket,
) <-chan incomingComms {
	ibound := startIncoming(conn) // Start goroutine that reads from network connection
	output := make(chan incomingComms)

	DEBUG.Println(NET, "startIncomingComms started")
	go func() {
		for {
			if inboundFromStore == nil && ibound == nil {
				close(output)
				DEBUG.Println(NET, "startIncomingComms goroutine complete")
				return // As soon as ibound is closed we can exit (should have already processed an error)
			}
			DEBUG.Println(NET, "logic waiting for msg on ibound")

			var msg packets.ControlPacket
			var ok bool
			select {
			case msg, ok = <-inboundFromStore:
				if !ok {
					DEBUG.Println(NET, "startIncomingComms: inboundFromStore complete")
					inboundFromStore = nil // should happen quickly as this is only for persisted messages
					continue
				}
				DEBUG.Println(NET, "startIncomingComms: got msg from store")
			case ibMsg, ok := <-ibound:
				if !ok {
					DEBUG.Println(NET, "startIncomingComms: ibound complete")
					ibound = nil
					continue
				}
				DEBUG.Println(NET, "startIncomingComms: got msg on ibound")
				// If the inbound comms routine encounters any issues it will send us an error.
				if ibMsg.err != nil {
					output <- incomingComms{err: ibMsg.err}
					continue // Usually the channel will be closed immediately after sending an error but safer that we do not assume this
				}
				msg = ibMsg.cp

				c.persistInbound(msg)
				c.UpdateLastReceived() // Notify keepalive logic that we recently received a packet
			}

			switch m := msg.(type) {
			case *packets.PingrespPacket:
				DEBUG.Println(NET, "startIncomingComms: received pingresp")
				c.pingRespReceived()
			case *packets.SubackPacket:
				DEBUG.Println(NET, "startIncomingComms: received suback, id:", m.MessageID)
				token := c.getToken(m.MessageID)

				if t, ok := token.(*SubscribeToken); ok {
					DEBUG.Println(NET, "startIncomingComms: granted qoss", m.ReturnCodes)
					for i, qos := range m.ReturnCodes {
						t.subResult[t.subs[i]] = qos
					}
				}

				token.flowComplete()
				c.freeID(m.MessageID)
			case *packets.UnsubackPacket:
				DEBUG.Println(NET, "startIncomingComms: received unsuback, id:", m.MessageID)
				c.getToken(m.MessageID).flowComplete()
				c.freeID(m.MessageID)
			case *packets.PublishPacket:
				DEBUG.Println(NET, "startIncomingComms: received publish, msgId:", m.MessageID)
				output <- incomingComms{incomingPub: m}
			case *packets.PubackPacket:
				DEBUG.Println(NET, "startIncomingComms: received puback, id:", m.MessageID)
				c.getToken(m.MessageID).flowComplete()
				c.freeID(m.MessageID)
			case *packets.PubrecPacket:
				DEBUG.Println(NET, "startIncomingComms: received pubrec, id:", m.MessageID)
				prel := packets.NewControlPacket(packets.Pubrel).(*packets.PubrelPacket)
				prel.MessageID = m.MessageID
				output <- incomingComms{outbound: &PacketAndToken{p: prel, t: nil}}
			case *packets.PubrelPacket:
				DEBUG.Println(NET, "startIncomingComms: received pubrel, id:", m.MessageID)
				pc := packets.NewControlPacket(packets.Pubcomp).(*packets.PubcompPacket)
				pc.MessageID = m.MessageID
				c.persistOutbound(pc)
				output <- incomingComms{outbound: &PacketAndToken{p: pc, t: nil}}
			case *packets.PubcompPacket:
				DEBUG.Println(NET, "startIncomingComms: received pubcomp, id:", m.MessageID)
				c.getToken(m.MessageID).flowComplete()
				c.freeID(m.MessageID)
			}
		}
	}()
	return output
}

// startOutgoingComms initiates a go routine to transmit outgoing packets.
// Pass in an open network connection and channels for outbound messages (including those triggered
// directly from incoming comms).
// Returns a channel that will receive details of any errors (closed when the goroutine exits)
// This function wil only terminate when all input channels are closed
func startOutgoingComms(conn net.Conn,
	c commsFns,
	oboundp <-chan *PacketAndToken,
	obound <-chan *PacketAndToken,
	oboundFromIncoming <-chan *PacketAndToken,
) <-chan error {
	errChan := make(chan error)
	DEBUG.Println(NET, "outgoing started")

	go func() {
		for {
			DEBUG.Println(NET, "outgoing waiting for an outbound message")

			// This goroutine will only exits when all of the input channels we receive on have been closed. This approach is taken to avoid any
			// deadlocks (if the connection goes down there are limited options as to what we can do with anything waiting on us and
			// throwing away the packets seems the best option)
			if oboundp == nil && obound == nil && oboundFromIncoming == nil {
				DEBUG.Println(NET, "outgoing comms stopping")
				close(errChan)
				return
			}

			select {
			case pub, ok := <-obound:
				if !ok {
					obound = nil
					continue
				}
				msg := pub.p.(*packets.PublishPacket)
				DEBUG.Println(NET, "obound msg to write", msg.MessageID)

				writeTimeout := c.getWriteTimeOut()
				if writeTimeout > 0 {
					if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
						ERROR.Println(NET, "SetWriteDeadline ", err)
					}
				}

				if err := msg.Write(conn); err != nil {
					ERROR.Println(NET, "outgoing obound reporting error ", err)
					pub.t.setError(err)
					// report error if it's not due to the connection being closed elsewhere
					if !strings.Contains(err.Error(), closedNetConnErrorText) {
						errChan <- err
					}
					continue
				}

				if writeTimeout > 0 {
					// If we successfully wrote, we don't want the timeout to happen during an idle period
					// so we reset it to infinite.
					if err := conn.SetWriteDeadline(time.Time{}); err != nil {
						ERROR.Println(NET, "SetWriteDeadline to 0 ", err)
					}
				}

				if msg.Qos == 0 {
					pub.t.flowComplete()
				}
				DEBUG.Println(NET, "obound wrote msg, id:", msg.MessageID)
			case msg, ok := <-oboundp:
				if !ok {
					oboundp = nil
					continue
				}
				DEBUG.Println(NET, "obound priority msg to write, type", reflect.TypeOf(msg.p))
				if err := msg.p.Write(conn); err != nil {
					ERROR.Println(NET, "outgoing oboundp reporting error ", err)
					if msg.t != nil {
						msg.t.setError(err)
					}
					errChan <- err
					continue
				}

				if _, ok := msg.p.(*packets.DisconnectPacket); ok {
					msg.t.(*DisconnectToken).flowComplete()
					DEBUG.Println(NET, "outbound wrote disconnect, closing connection")
					// As per the MQTT spec "After sending a DISCONNECT Packet the Client MUST close the Network Connection"
					// Closing the connection will cause the goroutines to end in sequence (starting with incoming comms)
					conn.Close()
				}
			case msg, ok := <-oboundFromIncoming: // message triggered by an inbound message (PubrecPacket or PubrelPacket)
				if !ok {
					oboundFromIncoming = nil
					continue
				}
				DEBUG.Println(NET, "obound from incoming msg to write, type", reflect.TypeOf(msg.p), " ID ", msg.p.Details().MessageID)
				if err := msg.p.Write(conn); err != nil {
					ERROR.Println(NET, "outgoing oboundFromIncoming reporting error", err)
					if msg.t != nil {
						msg.t.setError(err)
					}
					errChan <- err
					continue
				}
			}
			c.UpdateLastSent() // Record that a packet has been received (for keepalive routine)
		}
	}()
	return errChan
}

// commsFns provide access to the client state (messageids, requesting disconnection and updating timing)
type commsFns interface {
	getToken(id uint16) tokenCompletor       // Retrieve the token for the specified messageid (if none then a dummy token must be returned)
	freeID(id uint16)                        // Release the specified messageid (clearing out of any persistent store)
	UpdateLastReceived()                     // Must be called whenever a packet is received
	UpdateLastSent()                         // Must be called whenever a packet is successfully sent
	getWriteTimeOut() time.Duration          // Return the writetimeout (or 0 if none)
	persistOutbound(m packets.ControlPacket) // add the packet to the outbound store
	persistInbound(m packets.ControlPacket)  // add the packet to the inbound store
	pingRespReceived()                       // Called when a ping response is received
}

// startComms initiates goroutines that handles communications over the network connection
// Messages will be stored (via commsFns) and deleted from the store as necessary
// It returns two channels:
//  packets.PublishPacket - Will receive publish packets received over the network.
//  Closed when incoming comms routines exit (on shutdown or if network link closed)
//  error - Any errors will be sent on this channel. The channel is closed when all comms routines have shut down
//
// Note: The comms routines monitoring oboundp and obound will not shutdown until those channels are both closed. Any messages received between the
// connection being closed and those channels being closed will generate errors (and nothing will be sent). That way the chance of a deadlock is
// minimised.
func startComms(conn net.Conn, // Network connection (must be active)
	c commsFns, // getters and setters to enable us to cleanly interact with client
	inboundFromStore <-chan packets.ControlPacket, // Inbound packets from the persistence store (should be closed relatively soon after startup)
	oboundp <-chan *PacketAndToken,
	obound <-chan *PacketAndToken) (
	<-chan *packets.PublishPacket, // Publishpackages received over the network
	<-chan error, // Any errors (should generally trigger a disconnect)
) {
	// Start inbound comms handler; this needs to be able to transmit messages so we start a go routine to add these to the priority outbound channel
	ibound := startIncomingComms(conn, c, inboundFromStore)
	outboundFromIncoming := make(chan *PacketAndToken) // Will accept outgoing messages triggered by startIncomingComms (e.g. acknowledgements)

	// Start the outgoing handler. It is important to note that output from startIncomingComms is fed into startOutgoingComms (for ACK's)
	oboundErr := startOutgoingComms(conn, c, oboundp, obound, outboundFromIncoming)
	DEBUG.Println(NET, "startComms started")

	// Run up go routines to handle the output from the above comms functions - these are handled in separate
	// go routines because they can interact (e.g. ibound triggers an ACK to obound which triggers an error)
	var wg sync.WaitGroup
	wg.Add(2)

	outPublish := make(chan *packets.PublishPacket)
	outError := make(chan error)

	// Any messages received get passed to the appropriate channel
	go func() {
		for ic := range ibound {
			if ic.err != nil {
				outError <- ic.err
				continue
			}
			if ic.outbound != nil {
				outboundFromIncoming <- ic.outbound
				continue
			}
			if ic.incomingPub != nil {
				outPublish <- ic.incomingPub
				continue
			}
			ERROR.Println(STR, "startComms received empty incomingComms msg")
		}
		// Close channels that will not be written to again (allowing other routines to exit)
		close(outboundFromIncoming)
		close(outPublish)
		wg.Done()
	}()

	// Any errors will be passed out to our caller
	go func() {
		for err := range oboundErr {
			outError <- err
		}
		wg.Done()
	}()

	// outError is used by both routines so can only be closed when they are both complete
	go func() {
		wg.Wait()
		close(outError)
		DEBUG.Println(NET, "startComms closing outError")
	}()

	return outPublish, outError
}

// ackFunc acknowledges a packet
// WARNING the function returned must not be called if the comms routine is shutting down or not running
// (it needs outgoing comms in order to send the acknowledgement). Currently this is only called from
// matchAndDispatch which will be shutdown before the comms are
func ackFunc(oboundP chan *PacketAndToken, persist Store, packet *packets.PublishPacket) func() {
	return func() {
		switch packet.Qos {
		case 2:
			pr := packets.NewControlPacket(packets.Pubrec).(*packets.PubrecPacket)
			pr.MessageID = packet.MessageID
			DEBUG.Println(NET, "putting pubrec msg on obound")
			oboundP <- &PacketAndToken{p: pr, t: nil}
			DEBUG.Println(NET, "done putting pubrec msg on obound")
		case 1:
			pa := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
			pa.MessageID = packet.MessageID
			DEBUG.Println(NET, "putting puback msg on obound")
			persistOutbound(persist, pa)
			oboundP <- &PacketAndToken{p: pa, t: nil}
			DEBUG.Println(NET, "done putting puback msg on obound")
		case 0:
			// do nothing, since there is no need to send an ack packet back
		}
	}
}
