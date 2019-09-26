/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqnml.pre_hpp */
#ifndef _IMQNML_HPP_
#define _IMQNML_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQNML.HPP
//
//  Description:   "ImqNamelist" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1999,2016"
//  crc="900577296" >
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

#include "imqobj.hpp" // ImqObject


#define ImqNamelist ImqNml

class IMQ_EXPORTCLASS ImqNamelist : public ImqObject {
  char * opszNames ;
  MQLONG olNameCount ;
  unsigned int obCountRetrieved : 1 ;
  unsigned int obNamesRetrieved : 1 ;
  unsigned int obPadding1 : 14 ;
  unsigned int obPadding2 : 16 ;
public:
  // Overloaded "ImqObject" methods:
  virtual ImqBoolean description ( ImqString & );
  virtual ImqBoolean name ( ImqString & );
  // Directed "ImqObject" methods:
  ImqString description ( ) { return ImqObject::description( ); }
  ImqString name ( ) { return ImqObject::name( ); }
  // New methods:
  ImqNamelist ( );
  ImqNamelist ( const char * );
  ImqNamelist ( const ImqNamelist & );
  virtual ~ ImqNamelist ( );
  void operator = ( const ImqNamelist & );
  ImqBoolean nameCount ( MQLONG & );
  MQLONG nameCount ( );
  ImqBoolean namelistName ( const MQLONG, ImqString & );
  ImqString namelistName ( const MQLONG );
} ;


#endif
