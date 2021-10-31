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
	"fmt"
	"sync"
	"time"
)

// MId is 16 bit message id as specified by the MQTT spec.
// In general, these values should not be depended upon by
// the client application.
type MId uint16

type messageIds struct {
	sync.RWMutex
	index map[uint16]tokenCompletor

	lastIssuedID uint16 // The most recently issued ID. Used so we cycle through ids rather than immediately reusing them (can make debugging easier)
}

const (
	midMin uint16 = 1
	midMax uint16 = 65535
)

func (mids *messageIds) cleanUp() {
	mids.Lock()
	for _, token := range mids.index {
		switch token.(type) {
		case *PublishToken:
			token.setError(fmt.Errorf("connection lost before Publish completed"))
		case *SubscribeToken:
			token.setError(fmt.Errorf("connection lost before Subscribe completed"))
		case *UnsubscribeToken:
			token.setError(fmt.Errorf("connection lost before Unsubscribe completed"))
		case nil:
			continue
		}
		token.flowComplete()
	}
	mids.index = make(map[uint16]tokenCompletor)
	mids.Unlock()
	DEBUG.Println(MID, "cleaned up")
}

func (mids *messageIds) freeID(id uint16) {
	mids.Lock()
	delete(mids.index, id)
	mids.Unlock()
}

func (mids *messageIds) claimID(token tokenCompletor, id uint16) {
	mids.Lock()
	defer mids.Unlock()
	if _, ok := mids.index[id]; !ok {
		mids.index[id] = token
	} else {
		old := mids.index[id]
		old.flowComplete()
		mids.index[id] = token
	}
	if id > mids.lastIssuedID {
		mids.lastIssuedID = id
	}
}

// getID will return an available id or 0 if none available
// The id will generally be the previous id + 1 (because this makes tracing messages a bit simpler)
func (mids *messageIds) getID(t tokenCompletor) uint16 {
	mids.Lock()
	defer mids.Unlock()
	i := mids.lastIssuedID // note: the only situation where lastIssuedID is 0 the map will be empty
	looped := false        // uint16 will loop from 65535->0
	for {
		i++
		if i == 0 { // skip 0 because its not a valid id (Control Packets MUST contain a non-zero 16-bit Packet Identifier [MQTT-2.3.1-1])
			i++
			looped = true
		}
		if _, ok := mids.index[i]; !ok {
			mids.index[i] = t
			mids.lastIssuedID = i
			return i
		}
		if (looped && i == mids.lastIssuedID) || (mids.lastIssuedID == 0 && i == midMax) { // lastIssuedID will be 0 at startup
			return 0 // no free ids
		}
	}
}

func (mids *messageIds) getToken(id uint16) tokenCompletor {
	mids.RLock()
	defer mids.RUnlock()
	if token, ok := mids.index[id]; ok {
		return token
	}
	return &DummyToken{id: id}
}

type DummyToken struct {
	id uint16
}

// Wait implements the Token Wait method.
func (d *DummyToken) Wait() bool {
	return true
}

// WaitTimeout implements the Token WaitTimeout method.
func (d *DummyToken) WaitTimeout(t time.Duration) bool {
	return true
}

// Done implements the Token Done method.
func (d *DummyToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (d *DummyToken) flowComplete() {
	ERROR.Printf("A lookup for token %d returned nil\n", d.id)
}

func (d *DummyToken) Error() error {
	return nil
}

func (d *DummyToken) setError(e error) {}

// PlaceHolderToken does nothing and was implemented to allow a messageid to be reserved
// it differs from DummyToken in that calling flowComplete does not generate an error (it
// is expected that flowComplete will be called when the token is overwritten with a real token)
type PlaceHolderToken struct {
	id uint16
}

// Wait implements the Token Wait method.
func (p *PlaceHolderToken) Wait() bool {
	return true
}

// WaitTimeout implements the Token WaitTimeout method.
func (p *PlaceHolderToken) WaitTimeout(t time.Duration) bool {
	return true
}

// Done implements the Token Done method.
func (p *PlaceHolderToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (p *PlaceHolderToken) flowComplete() {
}

func (p *PlaceHolderToken) Error() error {
	return nil
}

func (p *PlaceHolderToken) setError(e error) {}
