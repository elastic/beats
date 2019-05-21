package ibmmqi

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>

*/
import "C"

import (
	"unsafe"
)

/*
MQCD is a structure containing the MQ Channel Definition (MQCD)
Only fields relevant to a client connection are included in the
Go version of this structure.
*/
type MQCD struct {
	ChannelName          string
	ConnectionName       string
	DiscInterval         int32
	SecurityExit         string
	SecurityUserData     string
	MaxMsgLength         int32
	HeartbeatInterval    int32
	SSLCipherSpec        string
	SSLPeerName          string
	SSLClientAuth        int32
	KeepAliveInterval    int32
	SharingConversations int32
	PropertyControl      int32
	ClientChannelWeight  int32
	ConnectionAffinity   int32
	DefReconnect         int32
	CertificateLabel     string
}

/*
NewMQCD fills in default values for the MQCD structure, based on the
MQCD_CLIENT_CONN_DEFAULT
*/
func NewMQCD() *MQCD {

	cd := new(MQCD)

	cd.ChannelName = ""
	cd.DiscInterval = 6000
	cd.SecurityExit = ""
	cd.SecurityUserData = ""
	cd.MaxMsgLength = 4194304
	cd.ConnectionName = ""
	cd.HeartbeatInterval = 1
	cd.SSLCipherSpec = ""
	cd.SSLPeerName = ""
	cd.SSLClientAuth = int32(C.MQSCA_REQUIRED)
	cd.KeepAliveInterval = -1
	cd.SharingConversations = 10
	cd.PropertyControl = int32(C.MQPROP_COMPATIBILITY)
	cd.ClientChannelWeight = 0
	cd.ConnectionAffinity = int32(C.MQCAFTY_PREFERRED)
	cd.DefReconnect = int32(C.MQRCN_NO)
	cd.CertificateLabel = ""

	return cd
}

/*
It is expected that copyXXtoC and copyXXfromC will be called as
matching pairs.
Most of the fields in the MQCD structure are not relevant for client
channels, but the default settings of such fields may still not be 0
or NULL (they are just ignored). The values here are taken from
MQ_CLIENT_CONN_DEFAULT structure for consistency.
*/
func copyCDtoC(mqcd *C.MQCD, gocd *MQCD) {

	setMQIString((*C.char)(&mqcd.ChannelName[0]), gocd.ChannelName, C.MQ_CHANNEL_NAME_LENGTH)
	mqcd.Version = C.MQCD_VERSION_11 // The version this is written to match
	mqcd.ChannelType = C.MQCHT_CLNTCONN
	mqcd.TransportType = C.MQXPT_TCP
	setMQIString((*C.char)(&mqcd.Desc[0]), "", C.MQ_CHANNEL_DESC_LENGTH)
	setMQIString((*C.char)(&mqcd.QMgrName[0]), "", C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.XmitQName[0]), "", C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.ShortConnectionName[0]), "", C.MQ_SHORT_CONN_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.MCAName[0]), "", C.MQ_MCA_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.ModeName[0]), "", C.MQ_MODE_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.TpName[0]), "", C.MQ_TP_NAME_LENGTH)
	mqcd.BatchSize = 50
	mqcd.DiscInterval = 6000
	mqcd.ShortRetryCount = 10
	mqcd.ShortRetryInterval = 60
	mqcd.LongRetryCount = 999999999
	mqcd.LongRetryInterval = 1200
	setMQIString((*C.char)(&mqcd.SecurityExit[0]), gocd.SecurityExit, C.MQ_EXIT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.MsgExit[0]), "", C.MQ_EXIT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.SendExit[0]), "", C.MQ_EXIT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.ReceiveExit[0]), "", C.MQ_EXIT_NAME_LENGTH)
	mqcd.SeqNumberWrap = 999999999
	mqcd.MaxMsgLength = C.MQLONG(gocd.MaxMsgLength)
	mqcd.PutAuthority = C.MQPA_DEFAULT
	mqcd.DataConversion = C.MQCDC_NO_SENDER_CONVERSION
	setMQIString((*C.char)(&mqcd.SecurityUserData[0]), gocd.SecurityUserData, C.MQ_EXIT_DATA_LENGTH)
	setMQIString((*C.char)(&mqcd.MsgUserData[0]), "", C.MQ_EXIT_DATA_LENGTH)
	setMQIString((*C.char)(&mqcd.SendUserData[0]), "", C.MQ_EXIT_DATA_LENGTH)
	setMQIString((*C.char)(&mqcd.ReceiveUserData[0]), "", C.MQ_EXIT_DATA_LENGTH)
	setMQIString((*C.char)(&mqcd.UserIdentifier[0]), "", C.MQ_USER_ID_LENGTH)
	setMQIString((*C.char)(&mqcd.Password[0]), "", C.MQ_PASSWORD_LENGTH)
	setMQIString((*C.char)(&mqcd.MCAUserIdentifier[0]), "", C.MQ_USER_ID_LENGTH)
	mqcd.MCAType = C.MQMCAT_PROCESS
	setMQIString((*C.char)(&mqcd.ConnectionName[0]), gocd.ConnectionName, C.MQ_CONN_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.RemoteUserIdentifier[0]), "", C.MQ_USER_ID_LENGTH)
	setMQIString((*C.char)(&mqcd.RemotePassword[0]), "", C.MQ_PASSWORD_LENGTH)
	setMQIString((*C.char)(&mqcd.MsgRetryExit[0]), "", C.MQ_EXIT_NAME_LENGTH)
	setMQIString((*C.char)(&mqcd.MsgRetryUserData[0]), "", C.MQ_EXIT_DATA_LENGTH)
	mqcd.MsgRetryCount = 10
	mqcd.MsgRetryInterval = 1000
	mqcd.HeartbeatInterval = 1
	mqcd.BatchInterval = 0
	mqcd.NonPersistentMsgSpeed = C.MQNPMS_FAST
	mqcd.StrucLength = C.MQCD_LENGTH_11
	mqcd.ExitNameLength = C.MQ_EXIT_NAME_LENGTH
	mqcd.ExitDataLength = C.MQ_EXIT_DATA_LENGTH
	mqcd.MsgExitsDefined = 0
	mqcd.SendExitsDefined = 0
	mqcd.ReceiveExitsDefined = 0
	mqcd.MsgExitPtr = C.MQPTR(nil)
	mqcd.MsgUserDataPtr = C.MQPTR(nil)
	mqcd.SendExitPtr = C.MQPTR(nil)
	mqcd.SendUserDataPtr = C.MQPTR(nil)
	mqcd.ReceiveExitPtr = C.MQPTR(nil)
	mqcd.ReceiveUserDataPtr = C.MQPTR(nil)
	mqcd.ClusterPtr = C.MQPTR(nil)
	mqcd.ClustersDefined = 0
	mqcd.NetworkPriority = 0
	mqcd.LongMCAUserIdLength = 0
	mqcd.LongRemoteUserIdLength = 0
	mqcd.LongMCAUserIdPtr = C.MQPTR(nil)
	mqcd.LongRemoteUserIdPtr = C.MQPTR(nil)
	C.memset((unsafe.Pointer)(&mqcd.MCASecurityId[0]), 0, C.MQ_SECURITY_ID_LENGTH)
	C.memset((unsafe.Pointer)(&mqcd.RemoteSecurityId[0]), 0, C.MQ_SECURITY_ID_LENGTH)
	setMQIString((*C.char)(&mqcd.SSLCipherSpec[0]), gocd.SSLCipherSpec, C.MQ_SSL_CIPHER_SPEC_LENGTH)
	mqcd.SSLPeerNamePtr = C.MQPTR(nil)
	mqcd.SSLPeerNameLength = 0
	if gocd.SSLPeerName != "" {
		mqcd.SSLPeerNamePtr = C.MQPTR(unsafe.Pointer(C.CString(gocd.SSLPeerName)))
		mqcd.SSLPeerNameLength = C.MQLONG(len(gocd.SSLPeerName))
	}
	mqcd.SSLClientAuth = C.MQLONG(gocd.SSLClientAuth)
	mqcd.KeepAliveInterval = C.MQLONG(gocd.KeepAliveInterval)
	setMQIString((*C.char)(&mqcd.LocalAddress[0]), "", C.MQ_LOCAL_ADDRESS_LENGTH)
	mqcd.BatchHeartbeat = 0
	for i := 0; i < 2; i++ {
		mqcd.HdrCompList[i] = C.MQCOMPRESS_NOT_AVAILABLE
	}
	for i := 0; i < 16; i++ {
		mqcd.MsgCompList[i] = C.MQCOMPRESS_NOT_AVAILABLE
	}
	mqcd.CLWLChannelRank = 0
	mqcd.CLWLChannelPriority = 0
	mqcd.CLWLChannelWeight = 50
	mqcd.ChannelMonitoring = C.MQMON_OFF
	mqcd.ChannelStatistics = C.MQMON_OFF
	mqcd.SharingConversations = C.MQLONG(gocd.SharingConversations)
	mqcd.PropertyControl = C.MQLONG(gocd.PropertyControl)
	mqcd.MaxInstances = 999999999
	mqcd.MaxInstancesPerClient = 999999999
	mqcd.ClientChannelWeight = C.MQLONG(gocd.ClientChannelWeight)
	mqcd.ConnectionAffinity = C.MQLONG(gocd.ConnectionAffinity)
	mqcd.BatchDataLimit = 5000
	mqcd.UseDLQ = C.MQUSEDLQ_YES
	mqcd.DefReconnect = C.MQLONG(gocd.DefReconnect)
	setMQIString((*C.char)(&mqcd.CertificateLabel[0]), gocd.CertificateLabel, C.MQ_CERT_LABEL_LENGTH)

	return
}

/*
Most of the parameters in the MQCD are input only.
Just need to clear up anything that was allocated in the copyCDtoC function
*/
func copyCDfromC(mqcd *C.MQCD, gocd *MQCD) {

	if mqcd.SSLPeerNamePtr != nil {
		C.free(unsafe.Pointer(mqcd.SSLPeerNamePtr))
	}

	return
}
