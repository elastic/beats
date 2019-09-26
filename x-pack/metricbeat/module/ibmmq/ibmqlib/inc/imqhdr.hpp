/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqhdr.pre_hpp */
#ifndef _IMQHDR_HPP_
#define _IMQHDR_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQHDR.HPP
//
//  Description:   "ImqHeader" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3735575883" >
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


#define ImqHeader ImqHdr

class IMQ_EXPORTCLASS ImqHeader : public ImqItem {
public :
  // New methods:
  ImqHeader ( );
  ImqHeader ( const ImqHeader & );
  virtual ~ ImqHeader ( );
  void operator = ( const ImqHeader & );
  virtual MQLONG characterSet ( ) const = 0 ;
  virtual MQLONG encoding ( ) const = 0 ;
  virtual ImqString format ( ) const = 0 ;
  virtual MQLONG headerFlags ( ) const = 0 ;
  virtual void setCharacterSet ( const MQLONG = MQCCSI_Q_MGR ) = 0 ;
  virtual void setEncoding ( const MQLONG = MQENC_NATIVE ) = 0 ;
  virtual void setFormat ( const char * = 0 ) = 0 ;
  virtual void setHeaderFlags ( const MQLONG = 0 ) = 0 ;
} ;


#endif
