/* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqmgr.pre_hpp */
#ifndef _IMQMGR_HPP_
#define _IMQMGR_HPP_

//  Library:       IBM MQ
//  Component:     IMQI (IBM MQ C++ MQI)
//  Part:          IMQMGR.HPP
//
//  Description:   "ImqQueueManager" class declaration
//  <copyright
//  notice="lm-source-program"
//  pids=""
//  years="1994,2016"
//  crc="2052935467" >
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

#include <imqobj.hpp>  // ImqObject


extern "C" {
typedef struct tagMQBO1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG Options ;
} MQBO1 ;
typedef MQBO1 MQPOINTER PMQBO1 ;
}

extern "C" {
typedef struct tagMQCNO1 {
  MQCHAR4 StrucId ;
  MQLONG Version ;
  MQLONG Options ;
} MQCNO1 ;
typedef MQCNO1 MQPOINTER PMQCNO1 ;
}

#define IMQ_EXPL_DISC_BACKOUT 0
#define IMQ_EXPL_DISC_COMMIT  1
#define IMQ_IMPL_CONN         2
#define IMQ_IMPL_DISC_BACKOUT 0
#define IMQ_IMPL_DISC_COMMIT  4

#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
class ImqAir ;
#endif
class ImqBin ;
class ImqChl ;
class IMQ_EXPORTCLASS ImqQueueManager : public ImqObject {
  void * oplink ;
  void * opdata ;
  ImqObject * opobjectFirst ;
  unsigned int obConnected : 1 ;
  unsigned int obOriginalConnection : 1 ;
  unsigned int obImplicitDisconnect : 1 ;
  unsigned int obDisconnectInProgress : 1 ;
  unsigned int obPadding1 : 12 ;
  unsigned int obPadding2 : 16 ;
  // New methods:
  void init ( );
  static ImqBoolean lock ( );
  static ImqBoolean unlock ( );
protected :
#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
  friend class ImqAir ;
#endif
  friend class ImqObject ;
  MQHCONN ohconn ;
  MQBO1 omqbo ;
  MQCNO1 omqcno ;
  // New methods:
  void setFirstManagedObject ( const ImqObject * pobject = 0 )
    { opobjectFirst = (ImqObject *)pobject ; }
#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
  void setFirstAuthenticationRecord ( const ImqAir * = 0 );
#endif
public :
  // Overloaded "ImqObject" methods:
  virtual ImqBoolean description ( ImqString & );
  virtual ImqBoolean name ( ImqString & );
  // Directed "ImqObject" methods:
  ImqString description ( ) { return ImqObject::description( ); }
  ImqString name ( ) { return ImqObject::name( ); }
  // New methods:
  ImqQueueManager ( );
  ImqQueueManager ( const char * );
  ImqQueueManager ( const ImqQueueManager & );
  virtual ~ ImqQueueManager ( );
  void operator = ( const ImqQueueManager & );
  ImqBoolean accountingConnOverride ( MQLONG & );
  MQLONG accountingConnOverride ( );
  ImqBoolean accountingInterval ( MQLONG & );
  MQLONG accountingInterval ( );
  ImqBoolean activityRecording ( MQLONG & );
  MQLONG activityRecording ( );
  ImqBoolean adoptNewMCACheck ( MQLONG & );
  MQLONG adoptNewMCACheck ( );
  ImqBoolean adoptNewMCAType( MQLONG & );
  MQLONG adoptNewMCAType ( );
  MQLONG authenticationType ( ) const;
  ImqBoolean authorityEvent ( MQLONG & );
  MQLONG authorityEvent ( );
  ImqBoolean backout ( );
  ImqBoolean begin ( );
  MQLONG beginOptions ( ) const { return omqbo.Options ; }
  static MQLONG behavior ( );
  static MQLONG behaviour ( ) { return ImqQueueManager::behavior( ); }
  ImqBoolean bridgeEvent ( MQLONG & );
  MQLONG bridgeEvent ( );
  ImqBoolean channelAutoDefinition ( MQLONG & );
  MQLONG channelAutoDefinition ( );
  ImqBoolean channelAutoDefinitionEvent ( MQLONG & );
  MQLONG channelAutoDefinitionEvent ( );
  ImqBoolean channelAutoDefinitionExit ( ImqString & );
  ImqString channelAutoDefinitionExit ( );
  ImqBoolean channelEvent ( MQLONG & );
  MQLONG channelEvent( );
  MQLONG channelInitiatorControl ( );
  ImqBoolean channelInitiatorControl ( MQLONG & );
  MQLONG channelInitiatorAdapters ( );
  ImqBoolean channelInitiatorAdapters ( MQLONG & );
  MQLONG channelInitiatorDispatchers ( );
  ImqBoolean channelInitiatorDispatchers ( MQLONG & );
  MQLONG channelInitiatorTraceAutoStart ( );
  ImqBoolean channelInitiatorTraceAutoStart ( MQLONG & );
  MQLONG channelInitiatorTraceTableSize ( );
  ImqBoolean channelInitiatorTraceTableSize ( MQLONG & );
  ImqBoolean channelMonitoring ( MQLONG & );
  MQLONG channelMonitoring ( );
  ImqBoolean channelReference ( ImqChl * & );
  ImqChl * channelReference ( );
  ImqBoolean channelStatistics ( MQLONG & );
  MQLONG channelStatistics ( );
  ImqBoolean characterSet ( MQLONG & );
  MQLONG characterSet ( );
  MQLONG clientSslKeyResetCount ( ) const;
  ImqBoolean clusterSenderMonitoring ( MQLONG & );
  MQLONG clusterSenderMonitoring ( );
  ImqBoolean clusterSenderStatistics ( MQLONG & );
  MQLONG clusterSenderStatistics ( );
  ImqBoolean clusterWorkLoadData ( ImqString & );
  ImqString clusterWorkLoadData ( );
  ImqBoolean clusterWorkLoadExit ( ImqString & );
  ImqString clusterWorkLoadExit ( );
  ImqBoolean clusterWorkLoadLength ( MQLONG & );
  MQLONG clusterWorkLoadLength ( );
  ImqBoolean clusterWorkLoadMRU ( MQLONG & );
  MQLONG clusterWorkLoadMRU ( );
  ImqBoolean clusterWorkLoadUseQ ( MQLONG & );
  MQLONG clusterWorkLoadUseQ ( );
  ImqBoolean commandEvent ( MQLONG & );
  MQLONG commandEvent ( );
  ImqBoolean commandInputQueueName ( ImqString & );
  ImqString commandInputQueueName ( );
  ImqBoolean commandLevel ( MQLONG & );
  MQLONG commandLevel ( );
  ImqBoolean commandServerControl ( MQLONG & );
  MQLONG commandServerControl ( );
  ImqBoolean commit ( );
  ImqBoolean connect ( );
  ImqBin connectionId ( ) const ;
  MQLONG connectOptions ( ) const { return omqcno.Options ; }
  ImqBoolean connectionStatus ( ) const { return (ImqBoolean)obConnected ; }
#if defined( MQCNO_VERSION_3 ) || defined( __OS400__ )
  ImqBin connectionTag ( ) const ;
#endif
#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
  ImqString cryptographicHardware ( );
#endif
  ImqString deadLetterQueueName ( );
  ImqBoolean deadLetterQueueName ( ImqString & );
  ImqBoolean defaultTransmissionQueueName ( ImqString & );
  ImqString defaultTransmissionQueueName ( );
  ImqBoolean disconnect ( );
  ImqBoolean distributionLists ( MQLONG & );
  MQLONG distributionLists ( );
#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
  ImqAir * firstAuthenticationRecord ( ) const ;
#endif
  ImqBoolean dnsGroup ( ImqString & );
  ImqString dnsGroup ( );
  ImqBoolean dnsWlm ( MQLONG & );
  MQLONG dnsWlm ( );
  ImqObject * firstManagedObject ( ) const { return opobjectFirst ; }
  ImqBoolean inhibitEvent ( MQLONG & );
  MQLONG inhibitEvent ( );
  ImqBoolean ipAddressVersion ( MQLONG & );
  MQLONG ipAddressVersion ( );
  ImqBoolean keepAlive ( MQLONG & );
  MQLONG keepAlive ( );
#if defined( MQCNO_VERSION_4 ) || defined( __OS400__ )
  ImqString keyRepository ( );
#endif
  ImqBoolean listenerTimer ( MQLONG & );
  MQLONG listenerTimer ( );
  ImqBoolean localEvent ( MQLONG & );
  MQLONG localEvent ( );
  ImqBoolean loggerEvent ( MQLONG & count );
  MQLONG loggerEvent ( );
  ImqBoolean luGroupName ( ImqString & );
  ImqString luGroupName ( );
  ImqBoolean luName ( ImqString & );
  ImqString luName ( );
  ImqBoolean lu62ARMSuffix ( ImqString & );
  ImqString lu62ARMSuffix ( );
  ImqBoolean maximumHandles ( MQLONG & );
  MQLONG maximumHandles ( );
  ImqBoolean maximumActiveChannels ( MQLONG & );
  MQLONG maximumActiveChannels ( );
  ImqBoolean maximumCurrentChannels ( MQLONG & );
  MQLONG maximumCurrentChannels ( );
  ImqBoolean maximumLu62Channels ( MQLONG & );
  MQLONG maximumLu62Channels ( );
  ImqBoolean maximumMessageLength ( MQLONG & );
  MQLONG maximumMessageLength ( );
  ImqBoolean maximumPriority ( MQLONG & );
  MQLONG maximumPriority ( );
  ImqBoolean maximumTcpChannels ( MQLONG & );
  MQLONG maximumTcpChannels ( );
#if defined( MQIA_MAX_UNCOMMITTED_MSGS )
  ImqBoolean maximumUncommittedMessages ( MQLONG & );
  MQLONG maximumUncommittedMessages ( );
#endif
  ImqBoolean mqiAccounting ( MQLONG & );
  MQLONG mqiAccounting ( );
  ImqBoolean mqiStatistics ( MQLONG & );
  MQLONG mqiStatistics ( );
  ImqBoolean outboundPortMax ( MQLONG & );
  MQLONG outboundPortMax ( );
  ImqBoolean outboundPortMin ( MQLONG & );
  MQLONG outboundPortMin ( );
  ImqBinary password ( ) const;
  ImqBoolean performanceEvent ( MQLONG & );
  MQLONG performanceEvent ( );
  ImqBoolean platform ( MQLONG & );
  MQLONG platform ( );
  ImqBoolean queueAccounting ( MQLONG & );
  MQLONG queueAccounting ( );
  ImqBoolean queueMonitoring ( MQLONG & );
  MQLONG queueMonitoring ( );
  ImqBoolean queueStatistics ( MQLONG & );
  MQLONG queueStatistics ( );
  ImqBoolean receiveTimeout ( MQLONG & );
  MQLONG receiveTimeout ( );
  ImqBoolean receiveTimeoutMin ( MQLONG & );
  MQLONG receiveTimeoutMin ( );
  ImqBoolean receiveTimeoutType ( MQLONG & );
  MQLONG receiveTimeoutType ( );
  ImqBoolean remoteEvent ( MQLONG & );
  MQLONG remoteEvent ( );
#ifdef MQCA_REPOSITORY_NAME
  ImqBoolean repositoryName ( ImqString & );
  ImqString repositoryName ( );
#endif
#ifdef MQCA_REPOSITORY_NAMELIST
  ImqBoolean repositoryNamelistName ( ImqString & );
  ImqString repositoryNamelistName ( );
#endif
  void setAuthenticationType ( const MQLONG = MQCSP_AUTH_NONE );
#if defined( MQBO_NONE )
  void setBeginOptions ( const MQLONG l = MQBO_NONE ) { omqbo.Options = l ; }
#endif
  static void setBehavior ( const MQLONG = 0 );
  static void setBehaviour ( const MQLONG l = 0 )
    { ImqQueueManager::setBehavior( l ); }
  ImqBoolean setChannelReference ( ImqChl & );
  ImqBoolean setChannelReference ( ImqChl * = 0 );
  void setClientSslKeyResetCount( const MQLONG );
#if defined( MQCNO_NONE ) || defined( __OS400__ )
  void setConnectOptions ( const MQLONG l = MQCNO_NONE )
    { omqcno.Options = l ; }
#endif
#if defined( MQCNO_VERSION_3 ) || defined ( __OS400__ )
  ImqBoolean setConnectionTag ( const unsigned char * = 0 );
  ImqBoolean setConnectionTag ( const ImqBin & );
#endif
  ImqBoolean setPassword ( const ImqString & );
  ImqBoolean setPassword ( const char * = 0 );
  ImqBoolean setPassword ( const ImqBinary & );
  ImqBoolean setUserId ( const ImqString & );
  ImqBoolean setUserId ( const char * = 0 );
  ImqBoolean setUserId ( const ImqBinary & );
  ImqBoolean sharedQueueQueueManagerName ( MQLONG & );
  MQLONG sharedQueueQueueManagerName ( );
  ImqBoolean sslEvent ( MQLONG & );
  MQLONG sslEvent ( );
  ImqBoolean sslFips ( MQLONG & );
  MQLONG sslFips ( );
  ImqBoolean sslKeyResetCount ( MQLONG & );
  MQLONG sslKeyResetCount ( );
#if defined( MQCNO_VERSION_4 ) || defined ( __OS400__ )
  ImqBoolean setCryptographicHardware ( const char * = 0 );
  ImqBoolean setKeyRepository ( const char * = 0 );
#endif
  ImqBoolean startStopEvent ( MQLONG & );
  MQLONG startStopEvent ( );
  ImqBoolean statisticsInterval ( MQLONG & );
  MQLONG statisticsInterval ( );
  ImqBoolean syncPointAvailability ( MQLONG & );
  MQLONG syncPointAvailability ( );
  ImqBoolean tcpName ( ImqString & );
  ImqString tcpName ( );
  ImqBoolean tcpStackType ( MQLONG & );
  MQLONG tcpStackType ( );
  ImqBoolean traceRouteRecording ( MQLONG & );
  MQLONG traceRouteRecording ( );
  ImqBoolean triggerInterval ( MQLONG & );
  MQLONG triggerInterval ( );
  ImqBinary userId ( ) const;
} ;


#endif
