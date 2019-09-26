/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqiih.pre_hpp */
#ifndef _IMQIIH_HPP_
#define _IMQIIH_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQIIH.HPP
//
//  Description:   "ImqIMSBridgeHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="2118536152" >
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
#include <imqhdr.hpp> // ImqHeader


extern "C" {
typedef struct tagMQIIH1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG StrucLength ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG Flags ;
  MQCHAR8 LTermOverride ;
  MQCHAR8 MFSMapName ;
  MQCHAR8 ReplyToFormat ;
  MQCHAR8 Authenticator ;
  MQBYTE16 TranInstanceId ;
  MQCHAR TranState ;
  MQCHAR CommitMode ;
  MQCHAR SecurityScope ;
  MQCHAR Reserved ;
} MQIIH1 ;
typedef MQIIH1 MQPOINTER PMQIIH1 ;
}

#define ImqImsBridgeHeader ImqIih
#define ImqIMSBridgeHeader ImqIih

class IMQ_EXPORTCLASS ImqIMSBridgeHeader : public ImqHeader {
protected :
  MQIIH1 omqiih ;
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
  ImqIMSBridgeHeader ( );
  ImqIMSBridgeHeader ( const ImqIMSBridgeHeader & );
  virtual ~ ImqIMSBridgeHeader ( );
  void operator = ( const ImqIMSBridgeHeader & );
  ImqString authenticator ( ) const ;
  MQCHAR commitMode ( ) const { return omqiih.CommitMode ; }
  ImqString logicalTerminalOverride ( ) const ;
  ImqString messageFormatServicesMapName ( ) const ;
  ImqString replyToFormat( ) const ;
  MQCHAR securityScope ( ) const { return omqiih.SecurityScope ; }
  ImqBinary transactionInstanceId ( ) const ;
  MQCHAR transactionState ( ) const { return omqiih.TranState ; }
  void setAuthenticator ( const char * );
  void setCommitMode ( const MQCHAR c ) { omqiih.CommitMode = c ; }
  void setLogicalTerminalOverride ( const char * );
  void setMessageFormatServicesMapName ( const char * );
  void setReplyToFormat ( const char * );
  void setSecurityScope ( const MQCHAR c ) { omqiih.SecurityScope = c ; }
  ImqBoolean setTransactionInstanceId ( const ImqBinary & );
  void setTransactionInstanceId ( const unsigned char * = 0 );
  void setTransactionState ( const MQCHAR c ) { omqiih.TranState = c ; }
} ;


#endif
