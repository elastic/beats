/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqmsg.pre_hpp */
#ifndef _IMQMSG_HPP_
#define _IMQMSG_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQMSG.HPP
//
//  Description:   "ImqMessage" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="2170584926" >
//  Licensed Materials - Property of IBM
//
//
//
//  (C) Copyright IBM Corp. 1994, 2016 All Rights Reserved.
//
//  US Government Users Restricted Rights - Use, duplication or
//  disclosure restricted by GSA ADP Schedule Contract with
//  IBM Corp.
//  </copyright>

#include <imqbin.hpp> // ImqBinary
#include <imqcac.hpp> // ImqCache
#include <imqmtr.hpp> // ImqMessageTracker
#include <imqstr.hpp> // ImqString

#include <cmqc.h>     // for MQMD2 typedef


#define ImqMessage                        ImqMsg

class ImqItm ;
class ImqQue ;
class IMQ_EXPORTCLASS ImqMessage : public ImqCache, public ImqMessageTracker {
protected :
  MQMD2 omqmd ;
  size_t ouiTotalMessageLength ;
  friend class ImqQue ;
  static void setVersionSupported ( const MQLONG );
public :
  // New methods:
  ImqMessage ( );
  ImqMessage ( const ImqMessage & );
  virtual ~ ImqMessage ( );
  void operator = ( const ImqMessage & );
  ImqString applicationIdData ( ) const ;
  ImqString applicationOriginData ( ) const ;
  MQLONG backoutCount ( ) const { return omqmd.BackoutCount ; }
  MQLONG characterSet ( ) const { return omqmd.CodedCharSetId ; }
  MQLONG encoding ( ) const { return omqmd.Encoding ; }
  MQLONG expiry ( ) const { return omqmd.Expiry ; }
  ImqString format ( ) const ;
  ImqBoolean formatIs ( const char * ) const ;
  MQLONG messageFlags ( ) const { return omqmd.MsgFlags ; }
  MQLONG messageType ( ) const { return omqmd.MsgType ; }
  MQLONG offset ( ) const { return omqmd.Offset ; }
  MQLONG originalLength ( ) const { return omqmd.OriginalLength ; }
  MQLONG persistence ( ) const { return omqmd.Persistence ; }
  MQLONG priority ( ) const { return omqmd.Priority ; }
  ImqString putApplicationName ( ) const ;
  MQLONG putApplicationType ( ) const { return omqmd.PutApplType ; }
  ImqString putDate ( ) const ;
  ImqString putTime ( ) const ;
  ImqBoolean readItem ( ImqItm & );
  ImqString replyToQueueManagerName ( ) const ;
  ImqString replyToQueueName ( ) const ;
  MQLONG report ( ) const { return omqmd.Report ; }
  MQLONG sequenceNumber ( ) const { return omqmd.MsgSeqNumber ; }
  void setApplicationIdData ( const char * = 0 );
  void setApplicationOriginData ( const char * = 0 );
  void setCharacterSet ( const MQLONG l = MQCCSI_Q_MGR )
    { omqmd.CodedCharSetId = l ; }
  void setEncoding ( const MQLONG l = MQENC_NATIVE ) { omqmd.Encoding = l ; }
  void setExpiry ( const MQLONG l ) { omqmd.Expiry = l ; }
  void setFormat ( const char * = 0 );
  void setMessageFlags ( const MQLONG l ) { omqmd.MsgFlags = l ; }
  void setMessageType ( const MQLONG l ) { omqmd.MsgType = l ; }
  void setOffset ( const MQLONG l ) { omqmd.Offset = l ; }
  void setOriginalLength ( const MQLONG l ) { omqmd.OriginalLength = l ; }
  void setPersistence ( const MQLONG l ) { omqmd.Persistence = l ; }
  void setPriority ( const MQLONG l ) { omqmd.Priority = l ; }
  void setPutApplicationName ( const char * = 0 );
  void setPutApplicationType ( const MQLONG l = MQAT_NO_CONTEXT )
    { omqmd.PutApplType = l ; }
  void setPutDate ( const char * = 0 );
  void setPutTime ( const char * = 0 );
  void setReplyToQueueManagerName ( const char * = 0 );
  void setReplyToQueueName ( const char * = 0 );
  void setReport ( const MQLONG l ) { omqmd.Report = l ; }
  void setSequenceNumber ( const MQLONG l ) { omqmd.MsgSeqNumber = l ; }
  void setUserId ( const char * = 0 );
  size_t totalMessageLength ( ) const { return ouiTotalMessageLength ; }
  ImqString userId ( ) const ;
  ImqBoolean writeItem ( ImqItm & );
} ;


#endif
