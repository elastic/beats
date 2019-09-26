/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqwih.pre_hpp */
#ifndef _IMQWIH_HPP_
#define _IMQWIH_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQWIH.HPP
//
//  Description:   "ImqWorkHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1999,2016"
//  crc="2847716590" >
//  Licensed Materials - Property of IBM
//
//
//
//  (C) Copyright IBM Corp. 1999, 2016 All Rights Reserved.
//
//  US Government Users Restricted Rights - Use, duplication or
//  disclosure restricted by GSA ADP Schedule Contract with
//  IBM Corp.
//  </copyright>

#include "imqbin.hpp" // ImqBinary
#include "imqhdr.hpp" // ImqHeader


#ifdef MQWIH_VERSION_1

extern "C" {
typedef struct tagMQWIH1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG StrucLength ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG Flags ;
  MQCHAR32 ServiceName ;
  MQCHAR8 ServiceStep ;
  MQBYTE16 MsgToken ;
  MQCHAR32 Reserved ;
} MQWIH1 ;
}

#define ImqWorkHeader ImqWih

class IMQ_EXPORTCLASS ImqWorkHeader : public ImqHeader {
protected :
  MQWIH1 omqwih ;
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
  ImqWorkHeader ( );
  ImqWorkHeader ( const ImqWorkHeader & );
  virtual ~ ImqWorkHeader ( );
  void operator = ( const ImqWorkHeader & );
  ImqBinary messageToken( ) const ;
  ImqString serviceName ( ) const ;
  ImqString serviceStep ( ) const ;
  ImqBoolean setMessageToken ( const ImqBinary & );
  void setMessageToken ( const unsigned char * = 0 );
  void setServiceName ( const char * );
  void setServiceStep ( const char * );
} ;

#endif


#endif
