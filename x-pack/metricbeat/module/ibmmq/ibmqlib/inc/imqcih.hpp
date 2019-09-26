/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqcih.pre_hpp */
#ifndef _IMQCIH_HPP_
#define _IMQCIH_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQCIH.HPP
//
//  Description:   "ImqCICSBridgeHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1998,2016"
//  crc="2643494309" >
//  Licensed Materials - Property of IBM
//
//
//
//  (C) Copyright IBM Corp. 1998, 2016 All Rights Reserved.
//
//  US Government Users Restricted Rights - Use, duplication or
//  disclosure restricted by GSA ADP Schedule Contract with
//  IBM Corp.
//  </copyright>

#include "imqbin.hpp" // ImqBinary
#include "imqhdr.hpp" // ImqHeader


extern "C" {
typedef struct tagMQCIH1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG StrucLength ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG Flags ;
  MQLONG ReturnCode ;
  MQLONG CompCode ;
  MQLONG Reason ;
  MQLONG UOWControl ;
  MQLONG GetWaitInterval ;
  MQLONG LinkType ;
  MQLONG OutputDataLength ;
  MQLONG FacilityKeepTime ;
  MQLONG ADSDescriptor ;
  MQLONG ConversationalTask ;
  MQLONG TaskEndStatus ;
  MQBYTE8 Facility ;
  MQCHAR4 Function ;
  MQCHAR4 AbendCode ;
  MQCHAR8 Authenticator ;
  MQCHAR8 Reserved1 ;
  MQCHAR8 ReplyToFormat ;
  MQCHAR4 RemoteSysId ;
  MQCHAR4 RemoteTransId ;
  MQCHAR4 TransactionId ;
  MQCHAR4 FacilityLike ;
  MQCHAR4 AttentionId ;
  MQCHAR4 StartCode ;
  MQCHAR4 CancelCode ;
  MQCHAR4 NextTransactionId ;
  MQCHAR8 Reserved2 ;
  MQCHAR8 Reserved3 ;
} MQCIH1 ;
typedef MQCIH1 MQPOINTER PMQCIH1 ;

typedef struct tagMQCIH2 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG StrucLength ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG Flags ;
  MQLONG ReturnCode ;
  MQLONG CompCode ;
  MQLONG Reason ;
  MQLONG UOWControl ;
  MQLONG GetWaitInterval ;
  MQLONG LinkType ;
  MQLONG OutputDataLength ;
  MQLONG FacilityKeepTime ;
  MQLONG ADSDescriptor ;
  MQLONG ConversationalTask ;
  MQLONG TaskEndStatus ;
  MQBYTE8 Facility ;
  MQCHAR4 Function ;
  MQCHAR4 AbendCode ;
  MQCHAR8 Authenticator ;
  MQCHAR8 Reserved1 ;
  MQCHAR8 ReplyToFormat ;
  MQCHAR4 RemoteSysId ;
  MQCHAR4 RemoteTransId ;
  MQCHAR4 TransactionId ;
  MQCHAR4 FacilityLike ;
  MQCHAR4 AttentionId ;
  MQCHAR4 StartCode ;
  MQCHAR4 CancelCode ;
  MQCHAR4 NextTransactionId ;
  MQCHAR8 Reserved2 ;
  MQCHAR8 Reserved3 ;
  MQLONG CursorPosition ;
  MQLONG ErrorOffset ;
  MQLONG InputItem ;
  MQLONG Reserved4 ;
} MQCIH2 ;
typedef MQCIH2 MQPOINTER PMQCIH2 ;
}

#ifndef MQCIH_VERSION_1
#define MQCADSD_NONE    0
#define MQCCT_NO        0
#define MQCGWI_DEFAULT  (-2)
#define MQCLT_PROGRAM   1
#define MQCODL_AS_INPUT (-1)
#define MQCUOWC_ONLY    0x00000111L
#define MQCIH_VERSION_1 1
#define MQCIH_STRUC_ID  "CIH "
#define MQCIH_NONE      0
#define MQCRC_OK        0
#define MQCTES_NOSYNC   0
#define MQFMT_CICS      "MQCICS  "
#endif

#ifndef MQCIH_VERSION_2
#define MQCIH_VERSION_2 2
#endif

#define ImqCICSBridgeHeader ImqCih

class IMQ_EXPORTCLASS ImqCICSBridgeHeader : public ImqHeader {
protected :
  MQLONG olVersion ;
  PMQCIH2 opcih ;
  void * oReserved1 ;
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
  ImqCICSBridgeHeader ( );
  ImqCICSBridgeHeader ( const ImqCICSBridgeHeader & );
  virtual ~ ImqCICSBridgeHeader ( );
  void operator = ( const ImqCICSBridgeHeader & );
  MQLONG ADSDescriptor ( ) const ;
  ImqString attentionIdentifier ( ) const ;
  ImqString authenticator ( ) const ;
  ImqString bridgeAbendCode ( ) const ;
  ImqString bridgeCancelCode ( ) const ;
  MQLONG bridgeCompletionCode ( ) const ;
  MQLONG bridgeErrorOffset ( ) const ;
  MQLONG bridgeReasonCode ( ) const ;
  MQLONG bridgeReturnCode ( ) const ;
  MQLONG conversationalTask ( ) const ;
  MQLONG cursorPosition ( ) const ;
  MQLONG facilityKeepTime ( ) const ;
  ImqString facilityLike ( ) const ;
  ImqBinary facilityToken ( ) const ;
  ImqString function ( ) const ;
  MQLONG getWaitInterval ( ) const ;
  MQLONG linkType ( ) const ;
  ImqString nextTransactionIdentifier ( ) const ;
  MQLONG outputDataLength ( ) const ;
  ImqString replyToFormat ( ) const ;
  void setADSDescriptor ( const MQLONG = MQCADSD_NONE );
  void setAttentionIdentifier ( const char * = 0 );
  void setAuthenticator ( const char * = 0 );
  void setBridgeCancelCode ( const char * = 0 );
  void setConversationalTask ( const MQLONG = MQCCT_NO );
  void setCursorPosition ( const MQLONG = 0 );
  void setFacilityKeepTime ( const MQLONG = 0 );
  void setFacilityLike ( const char * = 0 );
  ImqBoolean setFacilityToken ( const ImqBinary & ) ;
  void setFacilityToken ( const unsigned char * = 0 );
  void setGetWaitInterval ( const MQLONG = MQCGWI_DEFAULT );
  void setLinkType ( const MQLONG = MQCLT_PROGRAM );
  void setOutputDataLength ( const MQLONG = MQCODL_AS_INPUT );
  void setReplyToFormat ( const char * = 0 );
  void setStartCode ( const char * = 0 );
  void setTransactionIdentifier ( const char * = 0 );
  void setUOWControl ( const MQLONG = MQCUOWC_ONLY );
  ImqBoolean setVersion ( const MQLONG = MQCIH_VERSION_2 );
  ImqString startCode ( ) const ;
  MQLONG taskEndStatus ( ) const ;
  ImqString transactionIdentifier ( ) const ;
  MQLONG UOWControl ( ) const ;
  MQLONG version ( ) const ;
} ;


#endif
