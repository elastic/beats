/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqque.pre_hpp */
#ifndef _IMQQUE_HPP_
#define _IMQQUE_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQQUE.HPP
//
//  Description:   "ImqQueue" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="443571981" >
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
#include <imqgmo.hpp> // ImqGetMessageOptions
#include <imqmtr.hpp> // ImqMessageTracker
#include <imqobj.hpp> // ImqObject


#define ImqQueue ImqQue

class ImqDst ;
class ImqGmo ;
class ImqMsg ;
class ImqPmo ;
class IMQ_EXPORTCLASS ImqQueue : public ImqObject, public ImqMessageTracker {
  ImqQueue * opqueueGlobalNext ;
  ImqQueue * opqueueGlobalPrevious ;
  ImqQueue * opqueueDistributedNext ;
  ImqQueue * opqueueDistributedPrevious ;
  ImqDst * opdlist ;
  MQLONG olFeedback ;
  MQBYTE32 otokenAccountingToken ;
  MQBYTE24 otokenCorrelId ;
  MQBYTE24 otokenGroupId ;
  MQBYTE24 otokenMsgId ;
  // New methods:
  void add ( );
  ImqBoolean genericGet ( const MQHCONN, ImqMsg &, const size_t, void *,
                          size_t &, MQGMO3 *, bool bCanConvBuffer );
  ImqBoolean genericPut ( const MQHCONN, ImqMsg &, const MQLONG, void *,
                          ImqPmo & );
  void init ( );
  ImqBoolean openForResolvedNames ( );
  static ImqBoolean lock ( );
  static ImqBoolean unlock ( );
protected :
  friend class ImqDst ;
  unsigned int obGetWithSize : 1 ;
  unsigned int obPadding1 : 15 ;
  unsigned int obPadding2 : 16 ;
  // Overloaded "ImqObject" methods:
  virtual ImqBoolean openInformationPrepare ( );
  // New methods:
  virtual void putInformationDisperse ( ImqPmo & );
  virtual ImqBoolean putInformationPrepare ( const ImqMsg &, ImqPmo & );
  void setNextDistributedQueue ( ImqQueue * pqueue = 0 )
    { opqueueDistributedNext = pqueue ; }
  void setPreviousDistributedQueue ( ImqQueue * pqueue = 0 )
    { opqueueDistributedPrevious = pqueue ; }
public :
  // Overloaded "ImqObject" methods:
  virtual ImqBoolean closeTemporarily ( );
  virtual ImqBoolean description ( ImqString & );
  virtual ImqBoolean name ( ImqString & );
  // Directed "ImqObject" methods:
  ImqString description ( ) { return ImqObject::description( ); }
  ImqString name ( ) { return ImqObject::name( ); }
  // New methods:
  ImqQueue ( );
  ImqQueue ( const char * );
  ImqQueue ( const ImqQueue & );
  virtual ~ ImqQueue ( );
  void operator = ( const ImqQueue & );
  ImqBoolean backoutRequeueName ( ImqString & );
  ImqString backoutRequeueName ( );
  ImqBoolean backoutThreshold ( MQLONG & );
  MQLONG backoutThreshold ( );
  ImqBoolean baseQueueName ( ImqString & );
  ImqString baseQueueName ( );
#ifdef MQCA_CLUSTER_NAME
  ImqBoolean clusterName ( ImqString & );
  ImqString clusterName ( );
#endif
#ifdef MQCA_CLUSTER_NAMELIST
  ImqBoolean clusterNamelistName ( ImqString & );
  ImqString clusterNamelistName ( );
#endif
  ImqBoolean clusterWorkLoadPriority ( MQLONG & );
  MQLONG clusterWorkLoadPriority ( );
  ImqBoolean clusterWorkLoadRank ( MQLONG & );
  MQLONG clusterWorkLoadRank ( );
  ImqBoolean clusterWorkLoadUseQ ( MQLONG & );
  MQLONG clusterWorkLoadUseQ ( );
  ImqBoolean creationDate ( ImqString & );
  ImqString creationDate ( );
  ImqBoolean creationTime ( ImqString & );
  ImqString creationTime ( );
  ImqBoolean currentDepth ( MQLONG & );
  MQLONG currentDepth ( );
#ifdef MQIA_DEF_BIND
  ImqBoolean defaultBind ( MQLONG & );
  MQLONG defaultBind ( );
#endif
  ImqBoolean defaultInputOpenOption ( MQLONG & );
  MQLONG defaultInputOpenOption ( );
  ImqBoolean defaultPersistence ( MQLONG & );
  MQLONG defaultPersistence ( );
  ImqBoolean defaultPriority ( MQLONG & );
  MQLONG defaultPriority ( );
  ImqBoolean definitionType ( MQLONG & );
  MQLONG definitionType ( );
  ImqBoolean depthHighEvent ( MQLONG & );
  MQLONG depthHighEvent ( );
  ImqBoolean depthHighLimit ( MQLONG & );
  MQLONG depthHighLimit ( );
  ImqBoolean depthLowEvent ( MQLONG & );
  MQLONG depthLowEvent ( );
  ImqBoolean depthLowLimit ( MQLONG & );
  MQLONG depthLowLimit ( );
  ImqBoolean depthMaximumEvent ( MQLONG & );
  MQLONG depthMaximumEvent ( );
  ImqBoolean distributionLists ( MQLONG & );
  MQLONG distributionLists ( );
  ImqDst * distributionListReference ( ) const
    { return opdlist ; }
  ImqString dynamicQueueName ( ) const ;
  ImqBoolean get ( ImqMsg & );
  ImqBoolean get ( ImqMsg &, const size_t );
  ImqBoolean get ( ImqMsg &, ImqGmo & );
  ImqBoolean get ( ImqMsg &, ImqGmo &, const size_t );
  ImqBoolean get ( ImqMsg &, ImqGetMessageOptions & );
  ImqBoolean get ( ImqMsg &, ImqGetMessageOptions &, const size_t );
  ImqBoolean hardenGetBackout ( MQLONG & );
  MQLONG hardenGetBackout ( );
#ifdef MQIA_INDEX_TYPE
  ImqBoolean indexType ( MQLONG & );
  MQLONG indexType ( );
#endif
  ImqBoolean inhibitGet ( MQLONG & );
  MQLONG inhibitGet ( );
  ImqBoolean inhibitPut ( MQLONG & );
  MQLONG inhibitPut ( );
  ImqBoolean initiationQueueName ( ImqString & );
  ImqString initiationQueueName ( );
  ImqBoolean maximumDepth ( MQLONG & );
  MQLONG maximumDepth ( );
  ImqBoolean maximumMessageLength ( MQLONG & );
  MQLONG maximumMessageLength ( );
  ImqBoolean messageDeliverySequence ( MQLONG & );
  MQLONG messageDeliverySequence ( );
  ImqQueue * nextDistributedQueue ( ) const
    { return opqueueDistributedNext ; }
  ImqBoolean nonPersistentMessageClass ( MQLONG & );
  MQLONG nonPersistentMessageClass ( );
  ImqBoolean openInputCount ( MQLONG & );
  MQLONG openInputCount ( );
  ImqBoolean openOutputCount ( MQLONG & );
  MQLONG openOutputCount ( );
  ImqQueue * previousDistributedQueue ( ) const
    { return opqueueDistributedPrevious ; }
  ImqBoolean processName ( ImqString & );
  ImqString processName ( );
  ImqBoolean put ( ImqMsg & );
  ImqBoolean put ( ImqMsg &, ImqPmo & );
  ImqBoolean queueAccounting ( MQLONG & );
  MQLONG queueAccounting ( );
  ImqString queueManagerName ( ) const ;
  ImqBoolean queueMonitoring ( MQLONG & );
  MQLONG queueMonitoring ( );
  ImqBoolean queueStatistics  ( MQLONG & );
  MQLONG queueStatistics ( );
  ImqBoolean queueType ( MQLONG & );
  MQLONG queueType ( );
  ImqBoolean remoteQueueManagerName ( ImqString & );
  ImqString remoteQueueManagerName ( );
  ImqBoolean remoteQueueName ( ImqString & );
  ImqString remoteQueueName ( );
#ifdef MQOD_VERSION_3
  ImqBoolean resolvedQueueManagerName ( ImqString & );
  ImqString resolvedQueueManagerName ( );
  ImqBoolean resolvedQueueName ( ImqString & );
  ImqString resolvedQueueName ( );
#endif
  ImqBoolean retentionInterval ( MQLONG & );
  MQLONG retentionInterval ( );
  ImqBoolean scope ( MQLONG & );
  MQLONG scope ( );
  ImqBoolean serviceInterval ( MQLONG & );
  MQLONG serviceInterval ( );
  ImqBoolean serviceIntervalEvent ( MQLONG & );
  MQLONG serviceIntervalEvent ( );
  ImqBoolean setDistributionLists ( const MQLONG );
  void setDistributionListReference ( ImqDst * = 0 );
  void setDistributionListReference ( ImqDst & dlist )
    { setDistributionListReference( & dlist ) ; }
  ImqBoolean setDynamicQueueName ( const char * );
  ImqBoolean setInhibitGet ( const MQLONG );
  ImqBoolean setInhibitPut ( const MQLONG );
  ImqBoolean setQueueManagerName ( const char * );
  ImqBoolean setTriggerControl ( const MQLONG );
  ImqBoolean setTriggerData ( const char * );
  ImqBoolean setTriggerDepth ( const MQLONG );
  ImqBoolean setTriggerMessagePriority ( const MQLONG );
  ImqBoolean setTriggerType ( const MQLONG );
  ImqBoolean shareability ( MQLONG & );
  MQLONG shareability ( );
#ifdef MQCA_STORAGE_CLASS
  ImqBoolean storageClass ( ImqString & );
  ImqString storageClass ( );
#endif
  ImqBoolean transmissionQueueName ( ImqString & );
  ImqString transmissionQueueName ( );
  ImqBoolean triggerControl ( MQLONG & );
  MQLONG triggerControl ( );
  ImqBoolean triggerData ( ImqString & );
  ImqString triggerData ( );
  ImqBoolean triggerDepth ( MQLONG & );
  MQLONG triggerDepth ( );
  ImqBoolean triggerMessagePriority ( MQLONG & );
  MQLONG triggerMessagePriority ( );
  ImqBoolean triggerType ( MQLONG & );
  MQLONG triggerType ( );
  ImqBoolean usage ( MQLONG & );
  MQLONG usage ( );
} ;


#endif
