/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqerr.pre_hpp */
#ifndef _IMQERR_HPP_
#define _IMQERR_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQERR.HPP
//
//  Description:   "ImqError" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3674605453" >
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
#include <cmqc.h>
#include <cmqxc.h>
#include <imqtype.h>
#include <string.h>
}


#define ImqError ImqErr

class ImqObj ;
class ImqPmo ;
class IMQ_EXPORTCLASS ImqError {
  MQLONG olCompletionCode ;
  MQLONG olReasonCode ;
protected :
  friend class ImqObj ;
  friend class ImqPmo ;
  friend class ImqStr ;
  // New methods:
  ImqBoolean checkReadPointer ( const void *, const size_t );
  ImqBoolean checkWritePointer ( const void *, const size_t );
  void setCompletionCode ( const MQLONG lCode = 0 )
    { olCompletionCode = lCode ; }
  void setReasonCode ( const MQLONG lCode = 0 ) { olReasonCode = lCode ; }
public :
  // New methods:
  ImqError ( );
  ImqError ( const ImqError & );
  virtual ~ ImqError ( );
  void operator = ( const ImqError & );
  void clearErrorCodes ( );
  MQLONG completionCode ( ) const { return olCompletionCode ; }
  MQLONG reasonCode ( ) const { return olReasonCode ; }

#if defined( __WIN__ ) && defined( _MSC_VER )

#ifndef _IMQ_EXPORTCLASS
/* Only export new and delete individually if whole class has not been exported */
#define IMQ_COND_EXPORT __declspec(dllexport)
#else
#define IMQ_COND_EXPORT
#endif

#if ( _MSC_VER >= 1100 )
  /* Overloading of vector new and delete only supported on later compilers */
  IMQ_COND_EXPORT void * operator new [ ]( size_t );
  IMQ_COND_EXPORT void operator delete [ ]( void * );
#endif

  IMQ_COND_EXPORT void * operator new ( size_t );
  IMQ_COND_EXPORT void operator delete ( void * );
#endif
} ;


#endif // _IMQERR_HPP_

