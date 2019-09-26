/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqpro.pre_hpp */
#ifndef _IMQPRO_HPP_
#define _IMQPRO_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQPRO.HPP
//
//  Description:   "ImqProcess" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="913912487" >
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

#include <imqobj.hpp> // ImqObject


#define ImqProcess ImqPro

class IMQ_EXPORTCLASS ImqProcess : public ImqObject {
public :
  // Overloaded "ImqObject" methods:
  virtual ImqBoolean description ( ImqString & );
  virtual ImqBoolean name ( ImqString & );
  // Directed "ImqObject" methods:
  ImqString description ( ) { return ImqObject::description( ); }
  ImqString name ( ) { return ImqObject::name( ); }
  // New methods:
  ImqProcess ( );
  ImqProcess ( const char * );
  ImqProcess ( const ImqProcess & );
  virtual ~ ImqProcess ( );
  void operator = ( const ImqProcess & );
  ImqBoolean applicationId ( ImqString & );
  ImqString applicationId ( );
  ImqBoolean applicationType ( MQLONG & );
  MQLONG applicationType ( );
  ImqBoolean environmentData ( ImqString & );
  ImqString environmentData ( );
  ImqBoolean userData ( ImqString & );
  ImqString userData ( );
} ;


#endif
