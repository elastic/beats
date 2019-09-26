/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqrfh.pre_hpp */
#ifndef _IMQRFH_HPP_
#define _IMQRFH_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQRFH.HPP
//
//  Description:   "ImqReferenceHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="1358229214" >
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
typedef struct tagMQRMH1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG StrucLength ;
  MQLONG Encoding ;
  MQLONG CodedCharSetId ;
  MQCHAR8 Format ;
  MQLONG Flags ;
  MQCHAR8 ObjectType ;
  MQBYTE24 ObjectInstanceId ;
  MQLONG SrcEnvLength ;
  MQLONG SrcEnvOffset ;
  MQLONG SrcNameLength ;
  MQLONG SrcNameOffset ;
  MQLONG DestEnvLength ;
  MQLONG DestEnvOffset ;
  MQLONG DestNameLength ;
  MQLONG DestNameOffset ;
  MQLONG DataLogicalLength ;
  MQLONG DataLogicalOffset ;
  MQLONG DataLogicalOffset2 ;
} MQRMH1 ;
typedef MQRMH1 MQPOINTER PMQRMH1 ;
}

#define ImqReferenceHeader ImqRfh

class IMQ_EXPORTCLASS ImqReferenceHeader : public ImqHeader {
  ImqString ostrDestinationEnvironment ;
  ImqString ostrDestinationName ;
  ImqString ostrSourceEnvironment ;
  ImqString ostrSourceName ;
protected :
  MQRMH1 omqrmh ;
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
  ImqReferenceHeader ( );
  ImqReferenceHeader ( const ImqReferenceHeader & );
  virtual ~ ImqReferenceHeader ( );
  void operator = ( const ImqReferenceHeader & );
  ImqString destinationEnvironment ( ) const ;
  ImqString destinationName ( ) const ;
  ImqBinary instanceId ( ) const ;
  MQLONG logicalLength ( ) const { return omqrmh.DataLogicalLength ; }
  MQLONG logicalOffset ( ) const { return omqrmh.DataLogicalOffset ; }
  MQLONG logicalOffset2 ( ) const { return omqrmh.DataLogicalOffset2 ; }
  ImqString referenceType ( ) const ;
  void setDestinationEnvironment ( const char * = 0 );
  void setDestinationName ( const char * = 0 );
  ImqBoolean setInstanceId ( const ImqBinary & );
  void setInstanceId ( const unsigned char * = 0 );
  void setLogicalLength ( const MQLONG l ) { omqrmh.DataLogicalLength = l ; }
  void setLogicalOffset ( const MQLONG l ) { omqrmh.DataLogicalOffset = l ; }
  void setLogicalOffset2 ( const MQLONG l ) { omqrmh.DataLogicalOffset2 = l ; }
  void setReferenceType ( const char * = 0 );
  void setSourceEnvironment ( const char * = 0 );
  void setSourceName ( const char * = 0 );
  ImqString sourceEnvironment ( ) const ;
  ImqString sourceName ( ) const ;
} ;


#endif
