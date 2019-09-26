/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqpmo.pre_hpp */
#ifndef _IMQPMO_HPP_
#define _IMQPMO_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQPMO.HPP
//
//  Description:   "ImqPutMessageOptions" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="2118702644" >
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

#include <imqmtr.hpp> // ImqMessageTracker
#include <imqstr.hpp> // ImqString


extern "C" {
typedef struct tagMQPMO2 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG Options ;
  MQLONG Timeout ;
  MQHOBJ Context ;
  MQLONG KnownDestCount ;
  MQLONG UnknownDestCount ;
  MQLONG InvalidDestCount ;
  MQCHAR48 ResolvedQName ;
  MQCHAR48 ResolvedQMgrName ;
  MQLONG RecsPresent ;
  MQLONG PutMsgRecFields ;
  MQLONG PutMsgRecOffset ;
  MQLONG ResponseRecOffset ;
  MQPTR PutMsgRecPtr ;
  MQPTR ResponseRecPtr ;
} MQPMO2 ;
typedef MQPMO2 MQPOINTER PMQPMO2 ;
}

#define ImqPutMessageOptions ImqPmo

class ImqDst ;
class ImqQue ;
class IMQ_EXPORTCLASS ImqPutMessageOptions : public ImqError {
  ImqQue * opqueueContext ;
protected :
  MQPMO2 omqpmo ;
  friend class ImqDst ;
  friend class ImqQue ;
  // New methods:
  ImqBoolean allocateRecords ( const int, const ImqBoolean = 0 );
  void freeRecords ( );
  void readRecord ( const int, ImqMessageTracker & );
  void readResponse ( const int, ImqError & );
  void writeRecord ( const int, const ImqMessageTracker & );
public :
  // New methods:
  ImqPutMessageOptions ( );
  ImqPutMessageOptions ( const ImqPutMessageOptions & );
  void operator = ( const ImqPutMessageOptions & );
  MQLONG options ( ) const { return omqpmo.Options ; }
  MQLONG recordFields ( ) const { return omqpmo.PutMsgRecFields ; }
  ImqQue * contextReference( ) const { return opqueueContext ; }
  ImqString resolvedQueueManagerName ( ) const ;
  ImqString resolvedQueueName ( ) const ;
  void setContextReference ( const ImqQue & queue )
    { opqueueContext = (ImqQue *) & queue ; }
  void setContextReference ( const ImqQue * pqueue = 0 )
    { opqueueContext = (ImqQue *)pqueue ; }
  void setRecordFields ( const MQLONG l ) { omqpmo.PutMsgRecFields = l ; }
  void setOptions ( const MQLONG l ) { omqpmo.Options = l ; }
  void setSyncPointParticipation ( const ImqBoolean );
  ImqBoolean syncPointParticipation ( ) const ;
} ;


#endif
