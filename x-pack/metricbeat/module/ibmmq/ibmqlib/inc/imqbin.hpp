/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqbin.pre_hpp */
#ifndef _IMQBIN_HPP_
#define _IMQBIN_HPP_

//  Library:       IBM MQ
//  Component:     IMQB (IBM MQ C++ Support)
//  Part:          IMQBIN.HPP
//
//  Description:   "ImqBinary" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="1519572636" >
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

#include <imqitm.hpp> // ImqItem


#define ImqBinary ImqBin

class IMQ_EXPORTCLASS ImqBinary : public ImqItem {
  void * opvoid ;
  size_t ouiLength ;
protected :
  // New methods:
  void clear ( );
public :
  // Overloaded "ImqItem" methods:
  virtual ImqBoolean copyOut ( ImqMsg & );
  virtual ImqBoolean pasteIn ( ImqMsg & );
  // New methods:
  ImqBinary ( );
  ImqBinary ( const ImqBinary & );
  ImqBinary ( const void *, const size_t );
  virtual ~ ImqBinary ( );
  void operator = ( const ImqBinary & );
  ImqBoolean operator == ( const ImqBinary & ) const ;
  ImqBoolean copyOut ( void *, const size_t, const char = 0 );
  size_t dataLength ( ) const { return ouiLength ; }
  ImqBoolean isNull ( ) const ;
  ImqBoolean setDataLength ( const size_t );
  void * dataPointer ( ) const { return opvoid ; }
  ImqBoolean set ( const void *, const size_t );
} ;


#endif
