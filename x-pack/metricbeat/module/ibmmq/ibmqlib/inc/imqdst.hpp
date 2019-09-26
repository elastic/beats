/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqdst.pre_hpp */
#ifndef _IMQDST_HPP_
#define _IMQDST_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQDST.HPP
//
//  Description:   "ImqDistributionList" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="257479060" >
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

#include <imqque.hpp> // ImqQueue


#define ImqDistributionList ImqDst

class IMQ_EXPORTCLASS ImqDistributionList : public ImqQueue {
  ImqQueue * opfirstDistributedQueue;
protected :
  friend class ImqQueue ;
  // Overloaded "ImqObject" methods:
  virtual void openInformationDisperse ( );
  virtual ImqBoolean openInformationPrepare ( );
  // Overloaded "ImqQueue" methods:
  virtual void putInformationDisperse ( ImqPmo & );
  virtual ImqBoolean putInformationPrepare ( const ImqMsg &, ImqPmo & );
  // New methods:
  void setFirstDistributedQueue ( ImqQueue * pqueue = 0 )
    { opfirstDistributedQueue = pqueue ; }
public :
  // New methods:
  ImqDistributionList ( );
  ImqDistributionList ( const ImqDistributionList & );
  virtual ~ ImqDistributionList ( );
  void operator = ( const ImqDistributionList & );
  ImqQueue * firstDistributedQueue ( ) const
    { return opfirstDistributedQueue ; }
} ;


#endif
