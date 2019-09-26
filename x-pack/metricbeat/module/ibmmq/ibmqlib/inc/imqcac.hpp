/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqcac.pre_hpp */
#ifndef _IMQCAC_HPP_
#define _IMQCAC_HPP_

//  Library:       IBM MQ
//  Component:     IMQB (IBM MQ C++ MQI)
//  Part:          IMQCAC.HPP
//
//  Description:   "ImqCache" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="2822103983" >
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

extern "C" {
#include <string.h>
}

#include <imqerr.hpp> // ImqError


#define ImqCache ImqCac

class IMQ_EXPORTCLASS ImqCache : public virtual ImqError {
  char * opszBuffer ;
  size_t ouiDataOffset ;
  size_t ouiBufferLength ;
  size_t ouiMessageLength ;
  ImqBoolean obAutomaticBuffer ;
  char ocPadding1[ 3 ];
  // New methods:
  void setAutomaticBuffer ( const ImqBoolean bAutomatic )
    { obAutomaticBuffer = bAutomatic ; }
public :
  // New methods:
  ImqCache ( );
  ImqCache ( const ImqCache & );
  virtual ~ ImqCache ( );
  void operator = ( const ImqCache & );
  ImqBoolean automaticBuffer ( ) const { return obAutomaticBuffer ; }
  size_t bufferLength ( ) const { return ouiBufferLength ; }
  char * bufferPointer ( ) const { return opszBuffer ; }
  void clearMessage ( );
  size_t dataLength ( ) const ;
  size_t dataOffset ( ) const { return ouiDataOffset ; }
  char * dataPointer ( ) const { return bufferPointer( ) + dataOffset( ); }
  size_t messageLength ( ) const { return ouiMessageLength ; }
  ImqBoolean moreBytes ( const size_t );
  ImqBoolean read ( const size_t, char * & );
  ImqBoolean resizeBuffer ( const size_t );
  ImqBoolean setDataOffset ( const size_t );
  ImqBoolean setMessageLength ( const size_t );
  ImqBoolean useEmptyBuffer ( const char *, const size_t );
  ImqBoolean useFullBuffer ( const char *, const size_t );
  ImqBoolean write ( const size_t, const char * );
} ;



#endif
