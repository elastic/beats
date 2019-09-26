/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqchl.pre_hpp */
#ifndef _IMQCHL_HPP_
#define _IMQCHL_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQCHL.HPP
//
//  Description:   "ImqChannel" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3711594574" >
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

#include <imqstr.hpp> // ImqString


#define ImqChannel ImqChl

class IMQ_EXPORTCLASS ImqChannel : public ImqError {
  void * opdata ;
  ImqBoolean checkNames ( const size_t, const char * [ ] );
  ImqBoolean checkNames ( const size_t, const ImqString * [ ] );
  ImqBoolean getItems ( const size_t, ImqString * [ ],
                        const char *, const size_t, const size_t );
  ImqBoolean setData ( const MQLONG, const size_t, const char * [ ], MQPTR & );
  ImqBoolean setData ( const MQLONG, const size_t, const ImqString * [ ],
                       MQPTR & );
  ImqBoolean setNames ( const size_t, const char * [ ], MQPTR &, MQPTR & );
  ImqBoolean setNames ( const size_t, const ImqString * [ ], MQPTR &,
                        MQPTR & );
  void varyStorage ( MQLONG &, const size_t, MQPTR &, MQPTR & );
protected :
  friend class ImqMgr ;
  PMQCD MQCD ( ) const ;
public :
  // New methods:
  ImqChannel ( );
  ImqChannel ( const ImqChannel & );
  virtual ~ ImqChannel ( );
  void operator = ( const ImqChannel & );
  MQLONG batchHeartBeat ( ) const;
  ImqString channelName ( ) const ;
  ImqString connectionName ( ) const ;
  ImqBoolean headerCompression ( const size_t, MQLONG [ ] ) const ;
  size_t headerCompressionCount ( ) const ;
  MQLONG heartBeatInterval ( ) const ;
  MQLONG keepAliveInterval ( ) const ;
  ImqString localAddress() const;
  MQLONG maximumMessageLength ( ) const ;
  ImqBoolean messageCompression ( const size_t, MQLONG [ ] ) const ;
  size_t messageCompressionCount ( ) const ;
  ImqString modeName ( ) const ;
  ImqString password ( ) const ;
  size_t receiveExitCount ( ) const ;
  ImqString receiveExitName ( );
  ImqBoolean receiveExitNames ( const size_t, ImqString * [ ] );
  ImqString receiveUserData ( );
  ImqBoolean receiveUserData ( const size_t, ImqString * [ ] );
  ImqString securityExitName ( ) const ;
  ImqString securityUserData ( ) const ;
  size_t sendExitCount ( ) const ;
  ImqString sendExitName ( );
  ImqBoolean sendExitNames ( const size_t, ImqString * [ ] );
  ImqString sendUserData ( );
  ImqBoolean sendUserData ( const size_t, ImqString * [ ] );
  ImqBoolean setBatchHeartBeat ( const MQLONG = 0L );
  ImqBoolean setChannelName ( const char * = 0 );
  ImqBoolean setConnectionName ( const char * = 0 );
  ImqBoolean setHeaderCompression ( const size_t, const MQLONG [ ] );
  ImqBoolean setHeartBeatInterval ( const MQLONG = 300 );
  ImqBoolean setKeepAliveInterval ( const MQLONG = MQKAI_AUTO );
  ImqBoolean setLocalAddress ( const char * = 0 );
  ImqBoolean setMaximumMessageLength ( const MQLONG = 4194304 );
  ImqBoolean setMessageCompression( const size_t count,
                                    const MQLONG compress [ ] );
  ImqBoolean setModeName ( const char * = 0 );
  ImqBoolean setPassword ( const char * = 0 );
  ImqBoolean setReceiveExitName ( const char * = 0 );
  ImqBoolean setReceiveExitNames ( const size_t, const char * [ ] );
  ImqBoolean setReceiveExitNames ( const size_t, const ImqString * [ ] );
  ImqBoolean setReceiveUserData ( const char * = 0 );
  ImqBoolean setReceiveUserData ( const size_t, const char * [ ] );
  ImqBoolean setReceiveUserData ( const size_t, const ImqString * [ ] );
  ImqBoolean setSecurityExitName ( const char * = 0 );
  ImqBoolean setSecurityUserData ( const char * = 0 );
  ImqBoolean setSendExitName ( const char * = 0 );
  ImqBoolean setSendExitNames ( const size_t, const char * [ ] );
  ImqBoolean setSendExitNames ( const size_t, const ImqString * [ ] );
  ImqBoolean setSendUserData ( const char * = 0 );
  ImqBoolean setSendUserData ( const size_t, const char * [ ] );
  ImqBoolean setSendUserData ( const size_t, const ImqString * [ ] );
  ImqBoolean setSslCipherSpecification ( const char * = 0 );
  ImqBoolean setSslClientAuthentication ( const MQLONG = MQSCA_REQUIRED );
  ImqBoolean setSslPeerName ( const char * = 0 );
  ImqBoolean setTransactionProgramName ( const char * = 0 );
  ImqBoolean setTransportType ( const MQLONG = MQXPT_LU62 );
  ImqBoolean setUserId ( const char * = 0 );
  ImqString sslCipherSpecification ( ) const ;
  MQLONG    sslClientAuthentication ( ) const ;
  ImqString sslPeerName ( ) const ;
  ImqString transactionProgramName ( ) const ;
  MQLONG transportType ( ) const ;
  ImqString userId ( ) const ;
} ;


#endif


