/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqstr.pre_hpp */
#ifndef _IMQSTR_HPP_
#define _IMQSTR_HPP_

//  Library:       IBM MQ
//  Component:     IMQB (IBM MQ C++ Support)
//  Part:          IMQSTR.HPP
//
//  Description:   "ImqString" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="899501206" >
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


#define ImqString ImqStr

class IMQ_EXPORTCLASS ImqString : public ImqItem {
  // New methods:
  const ImqString concatenate ( const ImqString &, const ImqString & ) const ;
  size_t copyOut1 ( ImqString &, const char ) const ;
  static ImqBoolean lock ( );
  static ImqBoolean unlock ( );
protected :
  char * opszString ;
  size_t ouiSize ;
  // New methods:
  ImqBoolean assign ( const ImqString & );
public :
  // Overloaded "ImqItem" methods:
  virtual ImqBoolean copyOut ( ImqMsg & );
  virtual ImqBoolean pasteIn ( ImqMsg & );
  // New methods:
  ImqString ( );
  ImqString ( const ImqString & );
  ImqString ( const char );
  ImqString ( const char * );
  ImqString ( const void *, const size_t );
  virtual ~ ImqString ( );
  void operator = ( const ImqString & );
  ImqString operator + ( const ImqString & ) const ;
  ImqString operator + ( const char ) const ;
  ImqString operator + ( const char * ) const ;
  ImqString operator + ( const double ) const ;
  ImqString operator + ( const long ) const ;
  friend ImqString operator + ( const char * pszThis,
                                const ImqString & strThat )
    { return (ImqString)pszThis + strThat ; }
  void operator += ( const ImqString & );
  void operator += ( const char );
  void operator += ( const char * );
  void operator += ( const double );
  void operator += ( const long );
  ImqString operator ( ) ( const size_t, const size_t ) const ;
  ImqString operator ( ) ( const size_t ui ) const
    { return operator ( ) ( ui, (size_t)1 ); }
  operator char * ( ) const { return opszString ; }
  char & operator [ ] ( const size_t uiIndex ) const
    { return * ( opszString + uiIndex ); }
  ImqBoolean operator < ( const ImqString & str ) const
    { return this -> compare( str ) < 0 ; }
  ImqBoolean operator > ( const ImqString & str ) const
    { return this -> compare( str ) > 0 ; }
  ImqBoolean operator <= ( const ImqString & str ) const
    { return this -> compare( str ) <= 0 ; }
  ImqBoolean operator >= ( const ImqString & str ) const
    { return this -> compare( str ) >= 0 ; }
  ImqBoolean operator == ( const ImqString & str ) const
    { return this -> compare( str ) == 0 ; }
  ImqBoolean operator != ( const ImqString & str ) const
    { return this -> compare( str ) != 0 ; }
  short compare ( const ImqString & ) const ;
  static ImqBoolean copy ( char *, const size_t, const char *,
                           ImqError &, const char = 0 );
  static ImqBoolean copy ( char *, const size_t, const char *,
                           const char = 0 );
  ImqBoolean copyOut ( char *, const size_t, const char = 0 );
  size_t copyOut ( long & ) const ;
  size_t copyOut ( ImqString &, const char = ' ' ) const ;
  size_t cutOut ( long & );
  size_t cutOut ( ImqString &, const char = ' ' );
  ImqBoolean find ( const ImqString & );
  ImqBoolean find ( const ImqString &, size_t & );
  size_t length ( ) const ;
  ImqBoolean pasteIn ( const double, const char * = "%f" );
  ImqBoolean pasteIn ( const long );
  ImqBoolean pasteIn ( const void *, const size_t );
  ImqBoolean set ( const char *, const size_t );
  size_t storage ( ) const { return ouiSize ; }
  ImqBoolean setStorage ( const size_t );
  size_t stripLeading ( const char = ' ' );
  size_t stripTrailing ( const char = ' ' );
  ImqString upperCase ( ) const ;
} ;


#endif
