/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqmtr.pre_hpp */
#ifndef _IMQMTR_HPP_
#define _IMQMTR_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQMTR.HPP
//
//  Description:   "ImqMessageTracker" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3841220564" >
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


#define ImqMessageTracker ImqMtr

class IMQ_EXPORTCLASS ImqMessageTracker : public virtual ImqError {
protected :
  void * opvoidAccountingToken ;
  void * opvoidCorrelationId ;
  MQLONG * oplFeedback ;
  void * opvoidGroupId ;
  void * opvoidMessageId ;
public :
  // New methods:
  ImqMessageTracker ( );
  ImqMessageTracker ( const ImqMessageTracker & );
  virtual ~ ImqMessageTracker ( );
  void operator = ( const ImqMessageTracker & );
  ImqBinary accountingToken ( ) const ;
  ImqBinary correlationId ( ) const ;
  MQLONG feedback ( ) const ;
  ImqBinary groupId ( ) const ;
  ImqBinary messageId ( ) const ;
  ImqBoolean setAccountingToken ( const ImqBinary & );
  void setAccountingToken ( const unsigned char * = 0 );
  ImqBoolean setCorrelationId ( const ImqBinary & );
  void setCorrelationId ( const unsigned char * = 0 );
  void setFeedback ( const MQLONG );
  ImqBoolean setGroupId ( const ImqBinary & );
  void setGroupId ( const unsigned char * = 0 );
  ImqBoolean setMessageId ( const ImqBinary & );
  void setMessageId ( const unsigned char * = 0 );
} ;


#endif
