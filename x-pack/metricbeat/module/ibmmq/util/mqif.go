// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build ibm

package util

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
This file holds most of the calls to the MQI, so we
don't need to repeat common setups eg of MQMD or MQSD structures.
*/

import (
	"errors"
	"fmt"
	"os"

	"github.com/ibm-messaging/mq-golang/ibmmq"

	"github.com/elastic/beats/libbeat/logp"
)

var (
	qMgr             ibmmq.MQQueueManager
	cmdQObj          ibmmq.MQObject
	replyQObj        ibmmq.MQObject
	statsQObj        ibmmq.MQObject
	getBuffer        = make([]byte, 32768)
	platform         int32
	commandLevel     int32
	resolvedQMgrName string
	bindingQMgrName  string
	remoteQMgrName   string

	qmgrConnected     = false
	queuesOpened      = false
	statsQueuesOpened = false
	commandQueueOpen  = false
	subsOpened        = false
)

/*
InitConnection connects to the queue manager, and then
opens both the command queue and a dynamic reply queue
to be used for all responses including the publications
*/
func InitConnection(qMgrName string, replyQ string, cc *ConnectionConfig) error {
	return InitConnectionStats(qMgrName, replyQ, "", cc)
}

/*
InitConnectionStats is the same as InitConnection with the addition
of a call to open the queue manager statistics queue.
*/
func InitConnectionStats(qMgrName string, replyQ string, statsQ string, cc *ConnectionConfig) error {
	var err error
	gocno := ibmmq.NewMQCNO()
	gocsp := ibmmq.NewMQCSP()

	if cc.ClientMode {
		os.Setenv("MQSERVER", cc.MqServer)
		gocno.Options = ibmmq.MQCNO_CLIENT_BINDING
	} else {
		gocno.Options = ibmmq.MQCNO_LOCAL_BINDING
	}
	gocno.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	if cc.Password != "" {
		gocsp.Password = cc.Password
	}
	if cc.UserID != "" {
		gocsp.UserId = cc.UserID
		gocno.SecurityParms = gocsp
	}

	qMgr, err = ibmmq.Connx(qMgrName, gocno)
	if err == nil {
		qmgrConnected = true
	}

	// MQOPEN of the statistics queue
	if err == nil && statsQ != "" {
		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF | ibmmq.MQOO_FAIL_IF_QUIESCING
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = statsQ
		statsQObj, err = qMgr.Open(mqod, openOptions)
		if err == nil {
			statsQueuesOpened = true
		}
	}

	// MQOPEN of a reply queue
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF | ibmmq.MQOO_FAIL_IF_QUIESCING
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = replyQ
		replyQObj, err = qMgr.Open(mqod, openOptions)
		if err == nil {
			queuesOpened = true
		}
	}

	//Initally open command queue with current q manager
	if err == nil {
		err = OpenCommandQueue(qMgrName)
	}

	if err != nil {
		return fmt.Errorf("Cannot access queue manager. Error: %v", err)
	}
	bindingQMgrName = qMgrName
	return err
}

/*
OpenCommandQueue is opening a new command queue for the current queue manager.
This enables the beat to collect from the different queue manager then the direct connection was made to.
To change the target q manager call this function first to route the command queue to the new target
*/
func OpenCommandQueue(remoteQMgr string) error {
	// MQOPEN of the COMMAND QUEUE
	if commandQueueOpen {
		cmdQObj.Close(0)
	}
	logp.Info("Connect to command queue of q mgr name: %v", remoteQMgr)
	mqod := ibmmq.NewMQOD()

	openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_FAIL_IF_QUIESCING

	mqod.ObjectType = ibmmq.MQOT_Q
	if remoteQMgr != "" {
		mqod.ObjectQMgrName = remoteQMgr
	}
	mqod.ObjectName = "SYSTEM.ADMIN.COMMAND.QUEUE"

	cmdQObj, err = qMgr.Open(mqod, openOptions)
	if err == nil {
		commandQueueOpen = true
		remoteQMgrName = remoteQMgr
		err = DiscoverQmgrMetadata(remoteQMgrName)
		if err != nil {
			logp.Error(err)
		}
		return nil
	}

	return err
}

/*
DiscoverQmgrMetadata discovers important information about the qmgr - its real name
and the platform type. Also check if it is at least V9 (on Distributed platforms)
so that pub sub monitoring will work.
*/
func DiscoverQmgrMetadata(remoteQMgr string) error {

	data, err := getQManagerMetadata(remoteQMgr)

	if err == nil {
		for _, obj := range data {
			if obj.Values["mqia_platform"] != nil {
				platform = int32(obj.Values["mqia_platform"].(int64))
			} else {
				platform = 0
			}
			if obj.Values["mqia_command_level"] != nil {
				commandLevel = int32(obj.Values["mqia_command_level"].(int64))
			} else {
				commandLevel = 0
			}
			if platform != 0 && commandLevel != 0 {
				logp.Info("Successfully collected q mgr metadata. Name: %v, Platform: %v", obj.TargetObject, ibmmq.MQItoString("PL", int(platform)))
				return nil
			}
			return errors.New("Not able to get platfrom information")
		}
	}

	return err
}

/*
EndConnection tidies up by closing the queues and disconnecting.
*/
func EndConnection() {

	// MQCLOSE all subscriptions
	if subsOpened {
		for _, cl := range Metrics.Classes {
			for _, ty := range cl.Types {
				for _, hObj := range ty.subHobj {
					hObj.Close(0)
				}
			}
		}
	}

	// MQCLOSE the queues
	if queuesOpened {
		cmdQObj.Close(0)
		replyQObj.Close(0)
	}

	if statsQueuesOpened {
		statsQObj.Close(0)
	}

	// MQDISC regardless of other errors
	if qmgrConnected {
		qMgr.Disc()
	}

}

/*
getMessage returns a message from the replyQ. The only
parameter to the function says whether this should block
for 30 seconds or return immediately if there is no message
available. When working with the command queue, blocking is
required; when getting publications, non-blocking is better.

A 32K buffer was created at the top of this file, and should always
be big enough for what we are expecting.
*/
func getMessage(wait bool) ([]byte, error) {
	return getMessageWithHObj(wait, replyQObj)
}

func getMessageWithHObj(wait bool, hObj ibmmq.MQObject) ([]byte, error) {
	var err error
	var datalen int

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
	gmo.Options |= ibmmq.MQGMO_CONVERT

	gmo.MatchOptions = ibmmq.MQMO_NONE

	if wait {
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 2 * 1000
	}
	datalen, err = hObj.Get(getmqmd, gmo, getBuffer)

	return getBuffer[0:datalen], err
}

/*
subscribe to the nominated topic. The previously-opened
replyQ is used for publications; we do not use a managed queue here,
so that everything can be read from one queue. The object handle for the
subscription is returned so we can close it when it's no longer needed.
*/
func subscribe(topic string) (ibmmq.MQObject, error) {
	var err error

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE
	mqsd.Options |= ibmmq.MQSO_NON_DURABLE
	mqsd.Options |= ibmmq.MQSO_FAIL_IF_QUIESCING

	mqsd.ObjectString = topic

	subObj, err := qMgr.Sub(mqsd, &replyQObj)
	if err != nil {
		return subObj, fmt.Errorf("Error subscribing to topic '%s': %v", topic, err)
	}

	return subObj, err
}
