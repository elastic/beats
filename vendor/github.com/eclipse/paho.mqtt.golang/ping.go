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
	"sync/atomic"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

// keepalive - Send ping when connection unused for set period
// connection passed in to avoid race condition on shutdown
func keepalive(c *client, conn io.Writer) {
	defer c.workers.Done()
	DEBUG.Println(PNG, "keepalive starting")
	var checkInterval int64
	var pingSent time.Time

	if c.options.KeepAlive > 10 {
		checkInterval = 5
	} else {
		checkInterval = c.options.KeepAlive / 2
	}

	intervalTicker := time.NewTicker(time.Duration(checkInterval * int64(time.Second)))
	defer intervalTicker.Stop()

	for {
		select {
		case <-c.stop:
			DEBUG.Println(PNG, "keepalive stopped")
			return
		case <-intervalTicker.C:
			lastSent := c.lastSent.Load().(time.Time)
			lastReceived := c.lastReceived.Load().(time.Time)

			DEBUG.Println(PNG, "ping check", time.Since(lastSent).Seconds())
			if time.Since(lastSent) >= time.Duration(c.options.KeepAlive*int64(time.Second)) || time.Since(lastReceived) >= time.Duration(c.options.KeepAlive*int64(time.Second)) {
				if atomic.LoadInt32(&c.pingOutstanding) == 0 {
					DEBUG.Println(PNG, "keepalive sending ping")
					ping := packets.NewControlPacket(packets.Pingreq).(*packets.PingreqPacket)
					// We don't want to wait behind large messages being sent, the Write call
					// will block until it it able to send the packet.
					atomic.StoreInt32(&c.pingOutstanding, 1)
					if err := ping.Write(conn); err != nil {
						ERROR.Println(PNG, err)
					}
					c.lastSent.Store(time.Now())
					pingSent = time.Now()
				}
			}
			if atomic.LoadInt32(&c.pingOutstanding) > 0 && time.Since(pingSent) >= c.options.PingTimeout {
				CRITICAL.Println(PNG, "pingresp not received, disconnecting")
				c.internalConnLost(errors.New("pingresp not received, disconnecting")) // no harm in calling this if the connection is already down (or shutdown is in progress)
				return
			}
		}
	}
}
