/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqobj.pre_hpp */
#ifndef _IMQOBJ_HPP_
#define _IMQOBJ_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQOBJ.HPP
//
//  Description:   "ImqObject" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="3448760058" >
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
#include <imqstr.hpp> // ImqString


extern "C" {
typedef struct tagMQOD2 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG ObjectType ;
  MQCHAR48 ObjectName ;
  MQCHAR48 ObjectQMgrName ;
  MQCHAR48 DynamicQName ;
  MQCHAR12 AlternateUserId ;
  MQLONG RecsPresent ;
  MQLONG KnownDestCount ;
  MQLONG UnknownDestCount ;
  MQLONG InvalidDestCount ;
  MQLONG ObjectRecOffset ;
  MQLONG ResponseRecOffset ;
  MQPTR ObjectRecPtr ;
  MQPTR ResponseRecPtr ;
} MQOD2 ;
typedef MQOD2 MQPOINTER PMQOD2 ;

typedef struct tagMQOD3 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG ObjectType ;
  MQCHAR48 ObjectName ;
  MQCHAR48 ObjectQMgrName ;
  MQCHAR48 DynamicQName ;
  MQCHAR12 AlternateUserId ;
  MQLONG RecsPresent ;
  MQLONG KnownDestCount ;
  MQLONG UnknownDestCount ;
  MQLONG InvalidDestCount ;
  MQLONG ObjectRecOffset ;
  MQLONG ResponseRecOffset ;
  MQPTR ObjectRecPtr ;
  MQPTR ResponseRecPtr ;
  MQBYTE40 AlternateSecurityId ;
  MQCHAR48 ResolvedQName ;
  MQCHAR48 ResolvedQMgrName ;
} MQOD3 ;
typedef MQOD3 MQPOINTER PMQOD3 ;

typedef struct tagMQOD23 {
  MQCHAR4 Unused1 ;
  MQLONG Version ;
  MQLONG Unused2 ;
  MQCHAR48 Unused3 ;
  MQCHAR48 Unused4 ;
  MQCHAR48 Unused5 ;
  MQCHAR12 Unused6 ;
  MQLONG Unused7 ;
  MQLONG Unused8 ;
  MQLONG Unused9  ;
  MQLONG Unused10 ;
  MQLONG Unused11 ;
  MQLONG Unused12 ;
  MQPTR Unused13 ;
  PMQOD3 pmqod ;
} MQOD23 ;
}

#define ImqObject       ImqObj
#define ImqQueueManager ImqMgr

#define IMQ_IMPL_OPEN 8

#ifndef MQOO_RESOLVE_NAMES
#define MQOO_RESOLVE_NAMES 0x00010000L
#endif

class ImqQueueManager ;
class ImqPmo ;
class IMQ_EXPORTCLASS ImqObject : public virtual ImqError {
  MQLONG olOpenOptions ;
  MQLONG olCloseOptions ;
  ImqQueueManager * opmanager ;
  ImqObject * opobjectNext ;
  ImqObject * opobjectPrevious ;
protected :
  friend class ImqPmo ;
  MQHOBJ ohobj ;
  MQOD23 omqod ;
  unsigned int obOpen : 1 ;
  unsigned int obContextSaved : 1 ;
  unsigned int obBrowsing : 1 ;
  unsigned int obCursorLost : 1 ;
  unsigned int obPadding1 : 12 ;
  unsigned int obPadding2 : 16 ;
  // New methods:
  ImqBoolean allocateRecords ( const int, const ImqBoolean = 0 );
  virtual ImqBoolean closeTemporarily ( );
  MQHCONN connectionHandle( ) const ;
  void freeRecords ( );
  ImqBoolean inquire ( const MQLONG, MQLONG & );
  ImqBoolean inquire ( const MQLONG, char * &, const size_t );
  virtual void openInformationDisperse ( );
  virtual ImqBoolean openInformationPrepare ( );
  void readResponse ( const int, ImqError & );
  ImqBoolean set ( const MQLONG, const MQLONG );
  ImqBoolean set ( const MQLONG, const char *, const size_t );
  void setNextManagedObject ( const ImqObject * pobject = 0 )
    { opobjectNext = (ImqObject *)pobject ; }
  void setPreviousManagedObject ( const ImqObject * pobject = 0 )
    { opobjectPrevious = (ImqObject *)pobject ; }
  void writeRecord ( const int, const ImqObject & );
public :
  // New methods:
  ImqObject ( );
  ImqObject ( const ImqObject & );
  virtual ~ ImqObject ( );
  void operator = ( const ImqObject & );
#ifdef MQCA_ALTERATION_DATE
  ImqBoolean alterationDate ( ImqString & );
  ImqString alterationDate ( );
#endif
#ifdef MQCA_ALTERATION_TIME
  ImqBoolean alterationTime ( ImqString & );
  ImqString alterationTime ( );
#endif
#ifdef MQOD_VERSION_3
  ImqBinary alternateSecurityId ( ) const ;
#endif
  ImqString alternateUserId ( ) const ;
  static MQLONG behavior ( );
  static MQLONG behaviour ( ) { return ImqObject::behavior( ); }
  ImqBoolean close ( );
  MQLONG closeOptions ( ) const { return olCloseOptions ; }
  ImqQueueManager * connectionReference ( ) const { return opmanager ; }
  virtual ImqBoolean description ( ImqString & ) = 0 ;
  ImqString description ( ) ;
  virtual ImqBoolean name ( ImqString & );
  ImqString name ( );
  ImqObject * nextManagedObject ( ) const { return opobjectNext ; }
  ImqBoolean open ( );
  ImqBoolean openFor ( const MQLONG = 0 );
  MQLONG openOptions ( ) const { return olOpenOptions ; }
  ImqBoolean openStatus ( ) const { return (ImqBoolean)obOpen ; }
  ImqObject * previousManagedObject ( ) const { return opobjectPrevious ; }
#ifdef MQCA_Q_MGR_IDENTIFIER
  ImqBoolean queueManagerIdentifier ( ImqString & );
  ImqString queueManagerIdentifier ( );
#endif
#ifdef MQOD_VERSION_3
  ImqBoolean setAlternateSecurityId ( const ImqBinary & );
  ImqBoolean setAlternateSecurityId ( const unsigned char * = 0 );
#endif
  ImqBoolean setAlternateUserId ( const char * );
  static void setBehavior ( const MQLONG = 0 );
  static void setBehaviour ( const MQLONG l = 0 )
    { ImqObject::setBehavior( l ); }
  void setCloseOptions ( const MQLONG lCloseOptions )
    { olCloseOptions = lCloseOptions ; }
  void setConnectionReference ( ImqQueueManager * = 0 );
  void setConnectionReference ( ImqQueueManager & mgr )
    { setConnectionReference( & mgr ); }
  ImqBoolean setName ( const char * = 0 );
  ImqBoolean setOpenOptions ( const MQLONG );
} ;


#endif
