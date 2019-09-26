/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqtrg.pre_hpp */
#ifndef _IMQTRG_HPP_
#define _IMQTRG_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQTRG.HPP
//
//  Description:   "ImqTrigger" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="582746503" >
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


extern "C" {
typedef struct tagMQTM1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQCHAR48 QName ;
  MQCHAR48 ProcessName ;
  MQCHAR64 TriggerData ;
  MQLONG ApplType ;
  MQCHAR256 ApplId ;
  MQCHAR128 EnvData ;
  MQCHAR128 UserData ;
} MQTM1 ;
typedef MQTM1 MQPOINTER PMQTM1 ;
}

#define ImqTrigger ImqTrg

class ImqMsg ;
class IMQ_EXPORTCLASS ImqTrigger : public ImqItem {
protected :
  MQTM1 omqtm ;
  friend class ImqMsg ;
public :
  // Overloaded "ImqItem" methods:
  virtual ImqBoolean copyOut ( ImqMsg & );
  virtual ImqBoolean pasteIn ( ImqMsg & );
  // New methods:
  ImqTrigger ( );
  ImqTrigger ( const ImqTrigger & );
  virtual ~ ImqTrigger ( );
  void operator = ( const ImqTrigger & );
  ImqString applicationId ( ) const ;
  MQLONG applicationType ( ) const { return omqtm.ApplType ; }
  ImqBoolean copyOut ( MQTMC2 * );
  ImqString environmentData ( ) const ;
  ImqString processName ( ) const ;
  ImqString queueName ( ) const ;
  void setApplicationId ( const char * );
  void setApplicationType ( const MQLONG lType ) { omqtm.ApplType = lType ; }
  void setEnvironmentData ( const char * );
  void setProcessName ( const char * );
  void setQueueName ( const char * );
  void setTriggerData ( const char * );
  void setUserData ( const char * );
  ImqString triggerData ( ) const ;
  ImqString userData ( ) const ;
} ;


#endif
