/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqdlh.pre_hpp */
#ifndef _IMQDLH_HPP_
#define _IMQDLH_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQDLH.HPP
//
//  Description:   "ImqDeadLetterHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3703764355" >
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

#include <imqhdr.hpp> // ImqHeader


extern "C" {
typedef struct tagMQDLH1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG Reason ;
  MQCHAR48 DestQName ;
  MQCHAR48 DestQMgrName ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG PutApplType ;
  MQCHAR28 PutApplName ;
  MQCHAR8 PutDate ;
  MQCHAR8 PutTime ;
} MQDLH1 ;
typedef MQDLH1 MQPOINTER PMQDLH1 ;
}

#define ImqDeadLetterHeader ImqDlh

class IMQ_EXPORTCLASS ImqDeadLetterHeader : public ImqHeader {
  MQLONG olHeaderFlags ;
protected :
  MQDLH1 omqdlh ;
public :
  // Overloaded "ImqItem" methods:
  virtual ImqBoolean copyOut ( ImqMsg & );
  virtual ImqBoolean pasteIn ( ImqMsg & );
  // Overloaded "ImqHeader" methods:
  virtual MQLONG characterSet ( ) const ;
  virtual MQLONG encoding ( ) const ;
  virtual ImqString format ( ) const ;
  virtual MQLONG headerFlags ( ) const ;
  virtual void setCharacterSet ( const MQLONG = MQCCSI_Q_MGR );
  virtual void setEncoding ( const MQLONG = MQENC_NATIVE );
  virtual void setFormat ( const char * = 0 );
  virtual void setHeaderFlags ( const MQLONG = 0 );
  // New methods:
  ImqDeadLetterHeader ( );
  ImqDeadLetterHeader ( const ImqDeadLetterHeader & );
  virtual ~ ImqDeadLetterHeader ( );
  void operator = ( const ImqDeadLetterHeader & );
  MQLONG deadLetterReasonCode ( ) const { return omqdlh.Reason ; }
  ImqString destinationQueueManagerName ( ) const ;
  ImqString destinationQueueName ( ) const ;
  ImqString putApplicationName ( ) const ;
  MQLONG putApplicationType ( ) const { return omqdlh.PutApplType ; }
  ImqString putDate ( ) const ;
  ImqString putTime ( ) const ;
  void setDestinationQueueManagerName ( const char * );
  void setDestinationQueueName ( const char * );
  void setPutApplicationName ( const char * = 0 );
  void setPutApplicationType ( const MQLONG lType = MQAT_NO_CONTEXT )
    { omqdlh.PutApplType = lType ; }
  void setPutDate ( const char * = 0 );
  void setPutTime ( const char * = 0 );
  void setDeadLetterReasonCode ( const MQLONG lReason )
    { omqdlh.Reason = lReason ; }
} ;


#endif
