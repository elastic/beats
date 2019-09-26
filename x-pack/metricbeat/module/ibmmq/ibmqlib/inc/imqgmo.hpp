/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqgmo.pre_hpp */
#ifndef _IMQGMO_HPP_
#define _IMQGMO_HPP_

//  Library:       IBM MQ
//  Component:     IMQI IBM MQ C++ MQI)
//  Part:          IMQGMO.HPP
//
//  Description:   "ImqGetMessageOptions" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="797855000" >
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
typedef struct tagMQGMO3 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG Options ;
  MQLONG WaitInterval ;
  MQLONG Signal1 ;
  MQLONG Signal2 ;
  MQCHAR48 ResolvedQName ;
  MQLONG MatchOptions ;
  MQCHAR GroupStatus ;
  MQCHAR SegmentStatus ;
  MQCHAR Segmentation ;
  MQCHAR Reserved1 ;
  MQBYTE16 MsgToken ;
  MQLONG ReturnedLength ;
} MQGMO3 ;
typedef MQGMO3 MQPOINTER PMQGMO3 ;
}

#define ImqGetMessageOptions ImqGmo3

class ImqQue ;
class IMQ_EXPORTCLASS ImqGetMessageOptions : public ImqError {
protected :
  MQLONG olVersion ;
  PMQGMO3 opgmo ;
  friend class ImqQue ;
  static void setVersionSupported ( const MQLONG );
public :
  // New methods:
  ImqGetMessageOptions ( );
  ImqGetMessageOptions ( const ImqGetMessageOptions & );
  virtual ~ ImqGetMessageOptions ( );
  void operator = ( const ImqGetMessageOptions & );
  MQCHAR groupStatus ( ) const { return opgmo -> GroupStatus ; }
  MQLONG matchOptions ( ) const { return opgmo -> MatchOptions ; }
  ImqBinary messageToken ( ) const ;
  MQLONG options ( ) const { return opgmo -> Options ; }
  ImqString resolvedQueueName ( ) const ;
  MQLONG returnedLength ( ) const { return opgmo -> ReturnedLength ; }
  MQCHAR segmentation ( ) const { return opgmo -> Segmentation ; }
  MQCHAR segmentStatus ( ) const { return opgmo -> SegmentStatus ; }
  void setGroupStatus ( const MQCHAR c ) { opgmo -> GroupStatus = c ; }
  void setMatchOptions ( const MQLONG l ) { opgmo -> MatchOptions = l ; }
  ImqBoolean setMessageToken ( const ImqBinary & );
  void setMessageToken ( const unsigned char * = 0 );
  void setOptions ( const MQLONG lOptions ) { opgmo -> Options = lOptions ; }
  void setSegmentation ( const MQCHAR c ) { opgmo -> Segmentation = c ; }
  void setSegmentStatus ( const MQCHAR c ) { opgmo -> SegmentStatus = c ; }
  void setSyncPointParticipation ( const ImqBoolean );
  void setWaitInterval ( const MQLONG l ) { opgmo -> WaitInterval = l ; }
  ImqBoolean syncPointParticipation ( ) const ;
  MQLONG waitInterval ( ) const { return opgmo -> WaitInterval ; }
} ;


#endif
