 #if !defined(CMQSTRC_INCLUDED)
   #define CMQSTRC_INCLUDED
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQSTRC                                     */
 /*                                                              */
 /****************************************************************/
 /*  <N_OCO_COPYRIGHT>                                           */
 /*  Licensed Materials - Property of IBM                        */
 /*                                                              */
 /*  5724-H72                                                    */
 /*                                                              */
 /*  (c) Copyright IBM Corp. 1993, 2018 All Rights Reserved.     */
 /*                                                              */
 /*  US Government Users Restricted Rights - Use, duplication or */
 /*  disclosure restricted by GSA ADP Schedule Contract with     */
 /*  IBM Corp.                                                   */
 /*  <NOC_COPYRIGHT>                                             */
 /****************************************************************/
 /*                                                              */
 /*  FUNCTION:       This file provides mappings between MQI     */
 /*                  constant values and string versions of      */
 /*                  their definitions.                          */
 /*                                                              */
 /*  PROCESSOR:      C                                           */
 /*                                                              */
 /****************************************************************/
 /****************************************************************/
 /* <BEGIN_BUILDINFO>                                            */
 /* Generated on:  05/02/19 11:08                                */
 /* Build Level:   p911-L190205                                  */
 /* Build Type:    Production                                    */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/
 char *MQACTP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQACTP_NEW"; break;
   case          1: c = "MQACTP_FORWARD"; break;
   case          2: c = "MQACTP_REPLY"; break;
   case          3: c = "MQACTP_REPORT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQACTV_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQACTV_DETAIL_LOW"; break;
   case          2: c = "MQACTV_DETAIL_MEDIUM"; break;
   case          3: c = "MQACTV_DETAIL_HIGH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQACT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQACT_FORCE_REMOVE"; break;
   case          2: c = "MQACT_ADVANCE_LOG"; break;
   case          3: c = "MQACT_COLLECT_STATISTICS"; break;
   case          4: c = "MQACT_PUBSUB"; break;
   case          5: c = "MQACT_ADD"; break;
   case          6: c = "MQACT_REPLACE"; break;
   case          7: c = "MQACT_REMOVE"; break;
   case          8: c = "MQACT_REMOVEALL"; break;
   case          9: c = "MQACT_FAIL"; break;
   case         10: c = "MQACT_REDUCE_LOG"; break;
   case         11: c = "MQACT_ARCHIVE_LOG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQADOPT_CHECK_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQADOPT_CHECK_NONE"; break;
   case          1: c = "MQADOPT_CHECK_ALL"; break;
   case          2: c = "MQADOPT_CHECK_Q_MGR_NAME"; break;
   case          4: c = "MQADOPT_CHECK_NET_ADDR"; break;
   case          8: c = "MQADOPT_CHECK_CHANNEL_NAME"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQADOPT_TYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQADOPT_TYPE_NO"; break;
   case          1: c = "MQADOPT_TYPE_ALL"; break;
   case          2: c = "MQADOPT_TYPE_SVR"; break;
   case          4: c = "MQADOPT_TYPE_SDR"; break;
   case          8: c = "MQADOPT_TYPE_RCVR"; break;
   case         16: c = "MQADOPT_TYPE_CLUSRCVR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQADPCTX_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQADPCTX_NO"; break;
   case          1: c = "MQADPCTX_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAIT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQAIT_ALL"; break;
   case          1: c = "MQAIT_CRL_LDAP"; break;
   case          2: c = "MQAIT_OCSP"; break;
   case          3: c = "MQAIT_IDPW_OS"; break;
   case          4: c = "MQAIT_IDPW_LDAP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQAS_NONE"; break;
   case          1: c = "MQAS_STARTED"; break;
   case          2: c = "MQAS_START_WAIT"; break;
   case          3: c = "MQAS_STOPPED"; break;
   case          4: c = "MQAS_SUSPENDED"; break;
   case          5: c = "MQAS_SUSPENDED_TEMPORARY"; break;
   case          6: c = "MQAS_ACTIVE"; break;
   case          7: c = "MQAS_INACTIVE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQAT_UNKNOWN"; break;
   case          0: c = "MQAT_NO_CONTEXT"; break;
   case          1: c = "MQAT_CICS"; break;
   case          2: c = "MQAT_ZOS"; break;
   case          3: c = "MQAT_IMS"; break;
   case          4: c = "MQAT_OS2"; break;
   case          5: c = "MQAT_DOS"; break;
   case          6: c = "MQAT_UNIX"; break;
   case          7: c = "MQAT_QMGR"; break;
   case          8: c = "MQAT_OS400"; break;
   case          9: c = "MQAT_WINDOWS"; break;
   case         10: c = "MQAT_CICS_VSE"; break;
   case         11: c = "MQAT_WINDOWS_NT"; break;
   case         12: c = "MQAT_VMS"; break;
   case         13: c = "MQAT_NSK"; break;
   case         14: c = "MQAT_VOS"; break;
   case         15: c = "MQAT_OPEN_TP1"; break;
   case         18: c = "MQAT_VM"; break;
   case         19: c = "MQAT_IMS_BRIDGE"; break;
   case         20: c = "MQAT_XCF"; break;
   case         21: c = "MQAT_CICS_BRIDGE"; break;
   case         22: c = "MQAT_NOTES_AGENT"; break;
   case         23: c = "MQAT_TPF"; break;
   case         25: c = "MQAT_USER"; break;
   case         26: c = "MQAT_QMGR_PUBLISH"; break;
   case         28: c = "MQAT_JAVA"; break;
   case         29: c = "MQAT_DQM"; break;
   case         30: c = "MQAT_CHANNEL_INITIATOR"; break;
   case         31: c = "MQAT_WLM"; break;
   case         32: c = "MQAT_BATCH"; break;
   case         33: c = "MQAT_RRS_BATCH"; break;
   case         34: c = "MQAT_SIB"; break;
   case         35: c = "MQAT_SYSTEM_EXTENSION"; break;
   case         36: c = "MQAT_MCAST_PUBLISH"; break;
   case         37: c = "MQAT_AMQP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAUTHENTICATE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQAUTHENTICATE_OS"; break;
   case          1: c = "MQAUTHENTICATE_PAM"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAUTHOPT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQAUTHOPT_ENTITY_EXPLICIT"; break;
   case          2: c = "MQAUTHOPT_ENTITY_SET"; break;
   case         16: c = "MQAUTHOPT_NAME_EXPLICIT"; break;
   case         32: c = "MQAUTHOPT_NAME_ALL_MATCHING"; break;
   case         64: c = "MQAUTHOPT_NAME_AS_WILDCARD"; break;
   case        256: c = "MQAUTHOPT_CUMULATIVE"; break;
   case        512: c = "MQAUTHOPT_EXCLUDE_TEMP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAUTH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -3: c = "MQAUTH_ALL_MQI"; break;
   case         -2: c = "MQAUTH_ALL_ADMIN"; break;
   case         -1: c = "MQAUTH_ALL"; break;
   case          0: c = "MQAUTH_NONE"; break;
   case          1: c = "MQAUTH_ALT_USER_AUTHORITY"; break;
   case          2: c = "MQAUTH_BROWSE"; break;
   case          3: c = "MQAUTH_CHANGE"; break;
   case          4: c = "MQAUTH_CLEAR"; break;
   case          5: c = "MQAUTH_CONNECT"; break;
   case          6: c = "MQAUTH_CREATE"; break;
   case          7: c = "MQAUTH_DELETE"; break;
   case          8: c = "MQAUTH_DISPLAY"; break;
   case          9: c = "MQAUTH_INPUT"; break;
   case         10: c = "MQAUTH_INQUIRE"; break;
   case         11: c = "MQAUTH_OUTPUT"; break;
   case         12: c = "MQAUTH_PASS_ALL_CONTEXT"; break;
   case         13: c = "MQAUTH_PASS_IDENTITY_CONTEXT"; break;
   case         14: c = "MQAUTH_SET"; break;
   case         15: c = "MQAUTH_SET_ALL_CONTEXT"; break;
   case         16: c = "MQAUTH_SET_IDENTITY_CONTEXT"; break;
   case         17: c = "MQAUTH_CONTROL"; break;
   case         18: c = "MQAUTH_CONTROL_EXTENDED"; break;
   case         19: c = "MQAUTH_PUBLISH"; break;
   case         20: c = "MQAUTH_SUBSCRIBE"; break;
   case         21: c = "MQAUTH_RESUME"; break;
   case         22: c = "MQAUTH_SYSTEM"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQAUTO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQAUTO_START_NO"; break;
   case          1: c = "MQAUTO_START_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBACF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       7001: c = "MQBACF_EVENT_ACCOUNTING_TOKEN"; break;
   case       7002: c = "MQBACF_EVENT_SECURITY_ID"; break;
   case       7003: c = "MQBACF_RESPONSE_SET"; break;
   case       7004: c = "MQBACF_RESPONSE_ID"; break;
   case       7005: c = "MQBACF_EXTERNAL_UOW_ID"; break;
   case       7006: c = "MQBACF_CONNECTION_ID"; break;
   case       7007: c = "MQBACF_GENERIC_CONNECTION_ID"; break;
   case       7008: c = "MQBACF_ORIGIN_UOW_ID"; break;
   case       7009: c = "MQBACF_Q_MGR_UOW_ID"; break;
   case       7010: c = "MQBACF_ACCOUNTING_TOKEN"; break;
   case       7011: c = "MQBACF_CORREL_ID"; break;
   case       7012: c = "MQBACF_GROUP_ID"; break;
   case       7013: c = "MQBACF_MSG_ID"; break;
   case       7014: c = "MQBACF_CF_LEID"; break;
   case       7015: c = "MQBACF_DESTINATION_CORREL_ID"; break;
   case       7016: c = "MQBACF_SUB_ID"; break;
   case       7019: c = "MQBACF_ALTERNATE_SECURITYID"; break;
   case       7020: c = "MQBACF_MESSAGE_DATA"; break;
   case       7021: c = "MQBACF_MQBO_STRUCT"; break;
   case       7022: c = "MQBACF_MQCB_FUNCTION"; break;
   case       7023: c = "MQBACF_MQCBC_STRUCT"; break;
   case       7024: c = "MQBACF_MQCBD_STRUCT"; break;
   case       7025: c = "MQBACF_MQCD_STRUCT"; break;
   case       7026: c = "MQBACF_MQCNO_STRUCT"; break;
   case       7027: c = "MQBACF_MQGMO_STRUCT"; break;
   case       7028: c = "MQBACF_MQMD_STRUCT"; break;
   case       7029: c = "MQBACF_MQPMO_STRUCT"; break;
   case       7030: c = "MQBACF_MQSD_STRUCT"; break;
   case       7031: c = "MQBACF_MQSTS_STRUCT"; break;
   case       7032: c = "MQBACF_SUB_CORREL_ID"; break;
   case       7033: c = "MQBACF_XA_XID"; break;
   case       7034: c = "MQBACF_XQH_CORREL_ID"; break;
   case       7035: c = "MQBACF_XQH_MSG_ID"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQBL_NULL_TERMINATED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBMHO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQBMHO_NONE"; break;
   case          1: c = "MQBMHO_DELETE_PROPERTIES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBND_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQBND_BIND_ON_OPEN"; break;
   case          1: c = "MQBND_BIND_NOT_FIXED"; break;
   case          2: c = "MQBND_BIND_ON_GROUP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQBO_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBPLOCATION_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQBPLOCATION_BELOW"; break;
   case          1: c = "MQBPLOCATION_ABOVE"; break;
   case          2: c = "MQBPLOCATION_SWITCHING_ABOVE"; break;
   case          3: c = "MQBPLOCATION_SWITCHING_BELOW"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQBT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQBT_OTMA"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCACF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       3001: c = "MQCACF_FROM_Q_NAME"; break;
   case       3002: c = "MQCACF_TO_Q_NAME"; break;
   case       3003: c = "MQCACF_FROM_PROCESS_NAME"; break;
   case       3004: c = "MQCACF_TO_PROCESS_NAME"; break;
   case       3005: c = "MQCACF_FROM_NAMELIST_NAME"; break;
   case       3006: c = "MQCACF_TO_NAMELIST_NAME"; break;
   case       3007: c = "MQCACF_FROM_CHANNEL_NAME"; break;
   case       3008: c = "MQCACF_TO_CHANNEL_NAME"; break;
   case       3009: c = "MQCACF_FROM_AUTH_INFO_NAME"; break;
   case       3010: c = "MQCACF_TO_AUTH_INFO_NAME"; break;
   case       3011: c = "MQCACF_Q_NAMES"; break;
   case       3012: c = "MQCACF_PROCESS_NAMES"; break;
   case       3013: c = "MQCACF_NAMELIST_NAMES"; break;
   case       3014: c = "MQCACF_ESCAPE_TEXT"; break;
   case       3015: c = "MQCACF_LOCAL_Q_NAMES"; break;
   case       3016: c = "MQCACF_MODEL_Q_NAMES"; break;
   case       3017: c = "MQCACF_ALIAS_Q_NAMES"; break;
   case       3018: c = "MQCACF_REMOTE_Q_NAMES"; break;
   case       3019: c = "MQCACF_SENDER_CHANNEL_NAMES"; break;
   case       3020: c = "MQCACF_SERVER_CHANNEL_NAMES"; break;
   case       3021: c = "MQCACF_REQUESTER_CHANNEL_NAMES"; break;
   case       3022: c = "MQCACF_RECEIVER_CHANNEL_NAMES"; break;
   case       3023: c = "MQCACF_OBJECT_Q_MGR_NAME"; break;
   case       3024: c = "MQCACF_APPL_NAME"; break;
   case       3025: c = "MQCACF_USER_IDENTIFIER"; break;
   case       3026: c = "MQCACF_AUX_ERROR_DATA_STR_1"; break;
   case       3027: c = "MQCACF_AUX_ERROR_DATA_STR_2"; break;
   case       3028: c = "MQCACF_AUX_ERROR_DATA_STR_3"; break;
   case       3029: c = "MQCACF_BRIDGE_NAME"; break;
   case       3030: c = "MQCACF_STREAM_NAME"; break;
   case       3031: c = "MQCACF_TOPIC"; break;
   case       3032: c = "MQCACF_PARENT_Q_MGR_NAME"; break;
   case       3033: c = "MQCACF_CORREL_ID"; break;
   case       3034: c = "MQCACF_PUBLISH_TIMESTAMP"; break;
   case       3035: c = "MQCACF_STRING_DATA"; break;
   case       3036: c = "MQCACF_SUPPORTED_STREAM_NAME"; break;
   case       3037: c = "MQCACF_REG_TOPIC"; break;
   case       3038: c = "MQCACF_REG_TIME"; break;
   case       3039: c = "MQCACF_REG_USER_ID"; break;
   case       3040: c = "MQCACF_CHILD_Q_MGR_NAME"; break;
   case       3041: c = "MQCACF_REG_STREAM_NAME"; break;
   case       3042: c = "MQCACF_REG_Q_MGR_NAME"; break;
   case       3043: c = "MQCACF_REG_Q_NAME"; break;
   case       3044: c = "MQCACF_REG_CORREL_ID"; break;
   case       3045: c = "MQCACF_EVENT_USER_ID"; break;
   case       3046: c = "MQCACF_OBJECT_NAME"; break;
   case       3047: c = "MQCACF_EVENT_Q_MGR"; break;
   case       3048: c = "MQCACF_AUTH_INFO_NAMES"; break;
   case       3049: c = "MQCACF_EVENT_APPL_IDENTITY"; break;
   case       3050: c = "MQCACF_EVENT_APPL_NAME"; break;
   case       3051: c = "MQCACF_EVENT_APPL_ORIGIN"; break;
   case       3052: c = "MQCACF_SUBSCRIPTION_NAME"; break;
   case       3053: c = "MQCACF_REG_SUB_NAME"; break;
   case       3054: c = "MQCACF_SUBSCRIPTION_IDENTITY"; break;
   case       3055: c = "MQCACF_REG_SUB_IDENTITY"; break;
   case       3056: c = "MQCACF_SUBSCRIPTION_USER_DATA"; break;
   case       3057: c = "MQCACF_REG_SUB_USER_DATA"; break;
   case       3058: c = "MQCACF_APPL_TAG"; break;
   case       3059: c = "MQCACF_DATA_SET_NAME"; break;
   case       3060: c = "MQCACF_UOW_START_DATE"; break;
   case       3061: c = "MQCACF_UOW_START_TIME"; break;
   case       3062: c = "MQCACF_UOW_LOG_START_DATE"; break;
   case       3063: c = "MQCACF_UOW_LOG_START_TIME"; break;
   case       3064: c = "MQCACF_UOW_LOG_EXTENT_NAME"; break;
   case       3065: c = "MQCACF_PRINCIPAL_ENTITY_NAMES"; break;
   case       3066: c = "MQCACF_GROUP_ENTITY_NAMES"; break;
   case       3067: c = "MQCACF_AUTH_PROFILE_NAME"; break;
   case       3068: c = "MQCACF_ENTITY_NAME"; break;
   case       3069: c = "MQCACF_SERVICE_COMPONENT"; break;
   case       3070: c = "MQCACF_RESPONSE_Q_MGR_NAME"; break;
   case       3071: c = "MQCACF_CURRENT_LOG_EXTENT_NAME"; break;
   case       3072: c = "MQCACF_RESTART_LOG_EXTENT_NAME"; break;
   case       3073: c = "MQCACF_MEDIA_LOG_EXTENT_NAME"; break;
   case       3074: c = "MQCACF_LOG_PATH"; break;
   case       3075: c = "MQCACF_COMMAND_MQSC"; break;
   case       3076: c = "MQCACF_Q_MGR_CPF"; break;
   case       3078: c = "MQCACF_USAGE_LOG_RBA"; break;
   case       3079: c = "MQCACF_USAGE_LOG_LRSN"; break;
   case       3080: c = "MQCACF_COMMAND_SCOPE"; break;
   case       3081: c = "MQCACF_ASID"; break;
   case       3082: c = "MQCACF_PSB_NAME"; break;
   case       3083: c = "MQCACF_PST_ID"; break;
   case       3084: c = "MQCACF_TASK_NUMBER"; break;
   case       3085: c = "MQCACF_TRANSACTION_ID"; break;
   case       3086: c = "MQCACF_Q_MGR_UOW_ID"; break;
   case       3088: c = "MQCACF_ORIGIN_NAME"; break;
   case       3089: c = "MQCACF_ENV_INFO"; break;
   case       3090: c = "MQCACF_SECURITY_PROFILE"; break;
   case       3091: c = "MQCACF_CONFIGURATION_DATE"; break;
   case       3092: c = "MQCACF_CONFIGURATION_TIME"; break;
   case       3093: c = "MQCACF_FROM_CF_STRUC_NAME"; break;
   case       3094: c = "MQCACF_TO_CF_STRUC_NAME"; break;
   case       3095: c = "MQCACF_CF_STRUC_NAMES"; break;
   case       3096: c = "MQCACF_FAIL_DATE"; break;
   case       3097: c = "MQCACF_FAIL_TIME"; break;
   case       3098: c = "MQCACF_BACKUP_DATE"; break;
   case       3099: c = "MQCACF_BACKUP_TIME"; break;
   case       3100: c = "MQCACF_SYSTEM_NAME"; break;
   case       3101: c = "MQCACF_CF_STRUC_BACKUP_START"; break;
   case       3102: c = "MQCACF_CF_STRUC_BACKUP_END"; break;
   case       3103: c = "MQCACF_CF_STRUC_LOG_Q_MGRS"; break;
   case       3104: c = "MQCACF_FROM_STORAGE_CLASS"; break;
   case       3105: c = "MQCACF_TO_STORAGE_CLASS"; break;
   case       3106: c = "MQCACF_STORAGE_CLASS_NAMES"; break;
   case       3108: c = "MQCACF_DSG_NAME"; break;
   case       3109: c = "MQCACF_DB2_NAME"; break;
   case       3110: c = "MQCACF_SYSP_CMD_USER_ID"; break;
   case       3111: c = "MQCACF_SYSP_OTMA_GROUP"; break;
   case       3112: c = "MQCACF_SYSP_OTMA_MEMBER"; break;
   case       3113: c = "MQCACF_SYSP_OTMA_DRU_EXIT"; break;
   case       3114: c = "MQCACF_SYSP_OTMA_TPIPE_PFX"; break;
   case       3115: c = "MQCACF_SYSP_ARCHIVE_PFX1"; break;
   case       3116: c = "MQCACF_SYSP_ARCHIVE_UNIT1"; break;
   case       3117: c = "MQCACF_SYSP_LOG_CORREL_ID"; break;
   case       3118: c = "MQCACF_SYSP_UNIT_VOLSER"; break;
   case       3119: c = "MQCACF_SYSP_Q_MGR_TIME"; break;
   case       3120: c = "MQCACF_SYSP_Q_MGR_DATE"; break;
   case       3121: c = "MQCACF_SYSP_Q_MGR_RBA"; break;
   case       3122: c = "MQCACF_SYSP_LOG_RBA"; break;
   case       3123: c = "MQCACF_SYSP_SERVICE"; break;
   case       3124: c = "MQCACF_FROM_LISTENER_NAME"; break;
   case       3125: c = "MQCACF_TO_LISTENER_NAME"; break;
   case       3126: c = "MQCACF_FROM_SERVICE_NAME"; break;
   case       3127: c = "MQCACF_TO_SERVICE_NAME"; break;
   case       3128: c = "MQCACF_LAST_PUT_DATE"; break;
   case       3129: c = "MQCACF_LAST_PUT_TIME"; break;
   case       3130: c = "MQCACF_LAST_GET_DATE"; break;
   case       3131: c = "MQCACF_LAST_GET_TIME"; break;
   case       3132: c = "MQCACF_OPERATION_DATE"; break;
   case       3133: c = "MQCACF_OPERATION_TIME"; break;
   case       3134: c = "MQCACF_ACTIVITY_DESC"; break;
   case       3135: c = "MQCACF_APPL_IDENTITY_DATA"; break;
   case       3136: c = "MQCACF_APPL_ORIGIN_DATA"; break;
   case       3137: c = "MQCACF_PUT_DATE"; break;
   case       3138: c = "MQCACF_PUT_TIME"; break;
   case       3139: c = "MQCACF_REPLY_TO_Q"; break;
   case       3140: c = "MQCACF_REPLY_TO_Q_MGR"; break;
   case       3141: c = "MQCACF_RESOLVED_Q_NAME"; break;
   case       3142: c = "MQCACF_STRUC_ID"; break;
   case       3143: c = "MQCACF_VALUE_NAME"; break;
   case       3144: c = "MQCACF_SERVICE_START_DATE"; break;
   case       3145: c = "MQCACF_SERVICE_START_TIME"; break;
   case       3146: c = "MQCACF_SYSP_OFFLINE_RBA"; break;
   case       3147: c = "MQCACF_SYSP_ARCHIVE_PFX2"; break;
   case       3148: c = "MQCACF_SYSP_ARCHIVE_UNIT2"; break;
   case       3149: c = "MQCACF_TO_TOPIC_NAME"; break;
   case       3150: c = "MQCACF_FROM_TOPIC_NAME"; break;
   case       3151: c = "MQCACF_TOPIC_NAMES"; break;
   case       3152: c = "MQCACF_SUB_NAME"; break;
   case       3153: c = "MQCACF_DESTINATION_Q_MGR"; break;
   case       3154: c = "MQCACF_DESTINATION"; break;
   case       3156: c = "MQCACF_SUB_USER_ID"; break;
   case       3159: c = "MQCACF_SUB_USER_DATA"; break;
   case       3160: c = "MQCACF_SUB_SELECTOR"; break;
   case       3161: c = "MQCACF_LAST_PUB_DATE"; break;
   case       3162: c = "MQCACF_LAST_PUB_TIME"; break;
   case       3163: c = "MQCACF_FROM_SUB_NAME"; break;
   case       3164: c = "MQCACF_TO_SUB_NAME"; break;
   case       3167: c = "MQCACF_LAST_MSG_TIME"; break;
   case       3168: c = "MQCACF_LAST_MSG_DATE"; break;
   case       3169: c = "MQCACF_SUBSCRIPTION_POINT"; break;
   case       3170: c = "MQCACF_FILTER"; break;
   case       3171: c = "MQCACF_NONE"; break;
   case       3172: c = "MQCACF_ADMIN_TOPIC_NAMES"; break;
   case       3173: c = "MQCACF_ROUTING_FINGER_PRINT"; break;
   case       3174: c = "MQCACF_APPL_DESC"; break;
   case       3175: c = "MQCACF_Q_MGR_START_DATE"; break;
   case       3176: c = "MQCACF_Q_MGR_START_TIME"; break;
   case       3177: c = "MQCACF_FROM_COMM_INFO_NAME"; break;
   case       3178: c = "MQCACF_TO_COMM_INFO_NAME"; break;
   case       3179: c = "MQCACF_CF_OFFLOAD_SIZE1"; break;
   case       3180: c = "MQCACF_CF_OFFLOAD_SIZE2"; break;
   case       3181: c = "MQCACF_CF_OFFLOAD_SIZE3"; break;
   case       3182: c = "MQCACF_CF_SMDS_GENERIC_NAME"; break;
   case       3183: c = "MQCACF_CF_SMDS"; break;
   case       3184: c = "MQCACF_RECOVERY_DATE"; break;
   case       3185: c = "MQCACF_RECOVERY_TIME"; break;
   case       3186: c = "MQCACF_CF_SMDSCONN"; break;
   case       3187: c = "MQCACF_CF_STRUC_NAME"; break;
   case       3188: c = "MQCACF_ALTERNATE_USERID"; break;
   case       3189: c = "MQCACF_CHAR_ATTRS"; break;
   case       3190: c = "MQCACF_DYNAMIC_Q_NAME"; break;
   case       3191: c = "MQCACF_HOST_NAME"; break;
   case       3192: c = "MQCACF_MQCB_NAME"; break;
   case       3193: c = "MQCACF_OBJECT_STRING"; break;
   case       3194: c = "MQCACF_RESOLVED_LOCAL_Q_MGR"; break;
   case       3195: c = "MQCACF_RESOLVED_LOCAL_Q_NAME"; break;
   case       3196: c = "MQCACF_RESOLVED_OBJECT_STRING"; break;
   case       3197: c = "MQCACF_RESOLVED_Q_MGR"; break;
   case       3198: c = "MQCACF_SELECTION_STRING"; break;
   case       3199: c = "MQCACF_XA_INFO"; break;
   case       3200: c = "MQCACF_APPL_FUNCTION"; break;
   case       3201: c = "MQCACF_XQH_REMOTE_Q_NAME"; break;
   case       3202: c = "MQCACF_XQH_REMOTE_Q_MGR"; break;
   case       3203: c = "MQCACF_XQH_PUT_TIME"; break;
   case       3204: c = "MQCACF_XQH_PUT_DATE"; break;
   case       3205: c = "MQCACF_EXCL_OPERATOR_MESSAGES"; break;
   case       3206: c = "MQCACF_CSP_USER_IDENTIFIER"; break;
   case       3207: c = "MQCACF_AMQP_CLIENT_ID"; break;
   case       3208: c = "MQCACF_ARCHIVE_LOG_EXTENT_NAME"; break;
   case       5507: c = "MQCACF_CLUS_CHAN_Q_MGR_NAME"; break;
   case       5508: c = "MQCACF_CLUS_SHORT_CONN_NAME"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCACH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       3501: c = "MQCACH_CHANNEL_NAME"; break;
   case       3502: c = "MQCACH_DESC"; break;
   case       3503: c = "MQCACH_MODE_NAME"; break;
   case       3504: c = "MQCACH_TP_NAME"; break;
   case       3505: c = "MQCACH_XMIT_Q_NAME"; break;
   case       3506: c = "MQCACH_CONNECTION_NAME"; break;
   case       3507: c = "MQCACH_MCA_NAME"; break;
   case       3508: c = "MQCACH_SEC_EXIT_NAME"; break;
   case       3509: c = "MQCACH_MSG_EXIT_NAME"; break;
   case       3510: c = "MQCACH_SEND_EXIT_NAME"; break;
   case       3511: c = "MQCACH_RCV_EXIT_NAME"; break;
   case       3512: c = "MQCACH_CHANNEL_NAMES"; break;
   case       3513: c = "MQCACH_SEC_EXIT_USER_DATA"; break;
   case       3514: c = "MQCACH_MSG_EXIT_USER_DATA"; break;
   case       3515: c = "MQCACH_SEND_EXIT_USER_DATA"; break;
   case       3516: c = "MQCACH_RCV_EXIT_USER_DATA"; break;
   case       3517: c = "MQCACH_USER_ID"; break;
   case       3518: c = "MQCACH_PASSWORD"; break;
   case       3520: c = "MQCACH_LOCAL_ADDRESS"; break;
   case       3521: c = "MQCACH_LOCAL_NAME"; break;
   case       3524: c = "MQCACH_LAST_MSG_TIME"; break;
   case       3525: c = "MQCACH_LAST_MSG_DATE"; break;
   case       3527: c = "MQCACH_MCA_USER_ID"; break;
   case       3528: c = "MQCACH_CHANNEL_START_TIME"; break;
   case       3529: c = "MQCACH_CHANNEL_START_DATE"; break;
   case       3530: c = "MQCACH_MCA_JOB_NAME"; break;
   case       3531: c = "MQCACH_LAST_LUWID"; break;
   case       3532: c = "MQCACH_CURRENT_LUWID"; break;
   case       3533: c = "MQCACH_FORMAT_NAME"; break;
   case       3534: c = "MQCACH_MR_EXIT_NAME"; break;
   case       3535: c = "MQCACH_MR_EXIT_USER_DATA"; break;
   case       3544: c = "MQCACH_SSL_CIPHER_SPEC"; break;
   case       3545: c = "MQCACH_SSL_PEER_NAME"; break;
   case       3546: c = "MQCACH_SSL_HANDSHAKE_STAGE"; break;
   case       3547: c = "MQCACH_SSL_SHORT_PEER_NAME"; break;
   case       3548: c = "MQCACH_REMOTE_APPL_TAG"; break;
   case       3549: c = "MQCACH_SSL_CERT_USER_ID"; break;
   case       3550: c = "MQCACH_SSL_CERT_ISSUER_NAME"; break;
   case       3551: c = "MQCACH_LU_NAME"; break;
   case       3552: c = "MQCACH_IP_ADDRESS"; break;
   case       3553: c = "MQCACH_TCP_NAME"; break;
   case       3554: c = "MQCACH_LISTENER_NAME"; break;
   case       3555: c = "MQCACH_LISTENER_DESC"; break;
   case       3556: c = "MQCACH_LISTENER_START_DATE"; break;
   case       3557: c = "MQCACH_LISTENER_START_TIME"; break;
   case       3558: c = "MQCACH_SSL_KEY_RESET_DATE"; break;
   case       3559: c = "MQCACH_SSL_KEY_RESET_TIME"; break;
   case       3560: c = "MQCACH_REMOTE_VERSION"; break;
   case       3561: c = "MQCACH_REMOTE_PRODUCT"; break;
   case       3562: c = "MQCACH_GROUP_ADDRESS"; break;
   case       3563: c = "MQCACH_JAAS_CONFIG"; break;
   case       3564: c = "MQCACH_CLIENT_ID"; break;
   case       3565: c = "MQCACH_SSL_KEY_PASSPHRASE"; break;
   case       3566: c = "MQCACH_CONNECTION_NAME_LIST"; break;
   case       3567: c = "MQCACH_CLIENT_USER_ID"; break;
   case       3568: c = "MQCACH_MCA_USER_ID_LIST"; break;
   case       3569: c = "MQCACH_SSL_CIPHER_SUITE"; break;
   case       3570: c = "MQCACH_WEBCONTENT_PATH"; break;
   case       3571: c = "MQCACH_TOPIC_ROOT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCADSD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCADSD_NONE"; break;
   case          1: c = "MQCADSD_SEND"; break;
   case         16: c = "MQCADSD_RECV"; break;
   case        256: c = "MQCADSD_MSGFORMAT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCAFTY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCAFTY_NONE"; break;
   case          1: c = "MQCAFTY_PREFERRED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCAMO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       2701: c = "MQCAMO_CLOSE_DATE"; break;
   case       2702: c = "MQCAMO_CLOSE_TIME"; break;
   case       2703: c = "MQCAMO_CONN_DATE"; break;
   case       2704: c = "MQCAMO_CONN_TIME"; break;
   case       2705: c = "MQCAMO_DISC_DATE"; break;
   case       2706: c = "MQCAMO_DISC_TIME"; break;
   case       2707: c = "MQCAMO_END_DATE"; break;
   case       2708: c = "MQCAMO_END_TIME"; break;
   case       2709: c = "MQCAMO_OPEN_DATE"; break;
   case       2710: c = "MQCAMO_OPEN_TIME"; break;
   case       2711: c = "MQCAMO_START_DATE"; break;
   case       2712: c = "MQCAMO_START_TIME"; break;
   case       2713: c = "MQCAMO_MONITOR_CLASS"; break;
   case       2714: c = "MQCAMO_MONITOR_TYPE"; break;
   case       2715: c = "MQCAMO_MONITOR_DESC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCAP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCAP_NOT_SUPPORTED"; break;
   case          1: c = "MQCAP_SUPPORTED"; break;
   case          2: c = "MQCAP_EXPIRED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCAUT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCAUT_ALL"; break;
   case          1: c = "MQCAUT_BLOCKUSER"; break;
   case          2: c = "MQCAUT_BLOCKADDR"; break;
   case          3: c = "MQCAUT_SSLPEERMAP"; break;
   case          4: c = "MQCAUT_ADDRESSMAP"; break;
   case          5: c = "MQCAUT_USERMAP"; break;
   case          6: c = "MQCAUT_QMGRMAP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       2001: c = "MQCA_APPL_ID"; break;
   case       2002: c = "MQCA_BASE_OBJECT_NAME"; break;
   case       2003: c = "MQCA_COMMAND_INPUT_Q_NAME"; break;
   case       2004: c = "MQCA_CREATION_DATE"; break;
   case       2005: c = "MQCA_CREATION_TIME"; break;
   case       2006: c = "MQCA_DEAD_LETTER_Q_NAME"; break;
   case       2007: c = "MQCA_ENV_DATA"; break;
   case       2008: c = "MQCA_INITIATION_Q_NAME"; break;
   case       2009: c = "MQCA_NAMELIST_DESC"; break;
   case       2010: c = "MQCA_NAMELIST_NAME"; break;
   case       2011: c = "MQCA_PROCESS_DESC"; break;
   case       2012: c = "MQCA_PROCESS_NAME"; break;
   case       2013: c = "MQCA_Q_DESC"; break;
   case       2014: c = "MQCA_Q_MGR_DESC"; break;
   case       2015: c = "MQCA_Q_MGR_NAME"; break;
   case       2016: c = "MQCA_Q_NAME"; break;
   case       2017: c = "MQCA_REMOTE_Q_MGR_NAME"; break;
   case       2018: c = "MQCA_REMOTE_Q_NAME"; break;
   case       2019: c = "MQCA_BACKOUT_REQ_Q_NAME"; break;
   case       2020: c = "MQCA_NAMES"; break;
   case       2021: c = "MQCA_USER_DATA"; break;
   case       2022: c = "MQCA_STORAGE_CLASS"; break;
   case       2023: c = "MQCA_TRIGGER_DATA"; break;
   case       2024: c = "MQCA_XMIT_Q_NAME"; break;
   case       2025: c = "MQCA_DEF_XMIT_Q_NAME"; break;
   case       2026: c = "MQCA_CHANNEL_AUTO_DEF_EXIT"; break;
   case       2027: c = "MQCA_ALTERATION_DATE"; break;
   case       2028: c = "MQCA_ALTERATION_TIME"; break;
   case       2029: c = "MQCA_CLUSTER_NAME"; break;
   case       2030: c = "MQCA_CLUSTER_NAMELIST"; break;
   case       2031: c = "MQCA_CLUSTER_Q_MGR_NAME"; break;
   case       2032: c = "MQCA_Q_MGR_IDENTIFIER"; break;
   case       2033: c = "MQCA_CLUSTER_WORKLOAD_EXIT"; break;
   case       2034: c = "MQCA_CLUSTER_WORKLOAD_DATA"; break;
   case       2035: c = "MQCA_REPOSITORY_NAME"; break;
   case       2036: c = "MQCA_REPOSITORY_NAMELIST"; break;
   case       2037: c = "MQCA_CLUSTER_DATE"; break;
   case       2038: c = "MQCA_CLUSTER_TIME"; break;
   case       2039: c = "MQCA_CF_STRUC_NAME"; break;
   case       2040: c = "MQCA_QSG_NAME"; break;
   case       2041: c = "MQCA_IGQ_USER_ID"; break;
   case       2042: c = "MQCA_STORAGE_CLASS_DESC"; break;
   case       2043: c = "MQCA_XCF_GROUP_NAME"; break;
   case       2044: c = "MQCA_XCF_MEMBER_NAME"; break;
   case       2045: c = "MQCA_AUTH_INFO_NAME"; break;
   case       2046: c = "MQCA_AUTH_INFO_DESC"; break;
   case       2047: c = "MQCA_LDAP_USER_NAME"; break;
   case       2048: c = "MQCA_LDAP_PASSWORD"; break;
   case       2049: c = "MQCA_SSL_KEY_REPOSITORY"; break;
   case       2050: c = "MQCA_SSL_CRL_NAMELIST"; break;
   case       2051: c = "MQCA_SSL_CRYPTO_HARDWARE"; break;
   case       2052: c = "MQCA_CF_STRUC_DESC"; break;
   case       2053: c = "MQCA_AUTH_INFO_CONN_NAME"; break;
   case       2060: c = "MQCA_CICS_FILE_NAME"; break;
   case       2061: c = "MQCA_TRIGGER_TRANS_ID"; break;
   case       2062: c = "MQCA_TRIGGER_PROGRAM_NAME"; break;
   case       2063: c = "MQCA_TRIGGER_TERM_ID"; break;
   case       2064: c = "MQCA_TRIGGER_CHANNEL_NAME"; break;
   case       2065: c = "MQCA_SYSTEM_LOG_Q_NAME"; break;
   case       2066: c = "MQCA_MONITOR_Q_NAME"; break;
   case       2067: c = "MQCA_COMMAND_REPLY_Q_NAME"; break;
   case       2068: c = "MQCA_BATCH_INTERFACE_ID"; break;
   case       2069: c = "MQCA_SSL_KEY_LIBRARY"; break;
   case       2070: c = "MQCA_SSL_KEY_MEMBER"; break;
   case       2071: c = "MQCA_DNS_GROUP"; break;
   case       2072: c = "MQCA_LU_GROUP_NAME"; break;
   case       2073: c = "MQCA_LU_NAME"; break;
   case       2074: c = "MQCA_LU62_ARM_SUFFIX"; break;
   case       2075: c = "MQCA_TCP_NAME"; break;
   case       2076: c = "MQCA_CHINIT_SERVICE_PARM"; break;
   case       2077: c = "MQCA_SERVICE_NAME"; break;
   case       2078: c = "MQCA_SERVICE_DESC"; break;
   case       2079: c = "MQCA_SERVICE_START_COMMAND"; break;
   case       2080: c = "MQCA_SERVICE_START_ARGS"; break;
   case       2081: c = "MQCA_SERVICE_STOP_COMMAND"; break;
   case       2082: c = "MQCA_SERVICE_STOP_ARGS"; break;
   case       2083: c = "MQCA_STDOUT_DESTINATION"; break;
   case       2084: c = "MQCA_STDERR_DESTINATION"; break;
   case       2085: c = "MQCA_TPIPE_NAME"; break;
   case       2086: c = "MQCA_PASS_TICKET_APPL"; break;
   case       2090: c = "MQCA_AUTO_REORG_START_TIME"; break;
   case       2091: c = "MQCA_AUTO_REORG_CATALOG"; break;
   case       2092: c = "MQCA_TOPIC_NAME"; break;
   case       2093: c = "MQCA_TOPIC_DESC"; break;
   case       2094: c = "MQCA_TOPIC_STRING"; break;
   case       2096: c = "MQCA_MODEL_DURABLE_Q"; break;
   case       2097: c = "MQCA_MODEL_NON_DURABLE_Q"; break;
   case       2098: c = "MQCA_RESUME_DATE"; break;
   case       2099: c = "MQCA_RESUME_TIME"; break;
   case       2101: c = "MQCA_CHILD"; break;
   case       2102: c = "MQCA_PARENT"; break;
   case       2105: c = "MQCA_ADMIN_TOPIC_NAME"; break;
   case       2108: c = "MQCA_TOPIC_STRING_FILTER"; break;
   case       2109: c = "MQCA_AUTH_INFO_OCSP_URL"; break;
   case       2110: c = "MQCA_COMM_INFO_NAME"; break;
   case       2111: c = "MQCA_COMM_INFO_DESC"; break;
   case       2112: c = "MQCA_POLICY_NAME"; break;
   case       2113: c = "MQCA_SIGNER_DN"; break;
   case       2114: c = "MQCA_RECIPIENT_DN"; break;
   case       2115: c = "MQCA_INSTALLATION_DESC"; break;
   case       2116: c = "MQCA_INSTALLATION_NAME"; break;
   case       2117: c = "MQCA_INSTALLATION_PATH"; break;
   case       2118: c = "MQCA_CHLAUTH_DESC"; break;
   case       2119: c = "MQCA_CUSTOM"; break;
   case       2120: c = "MQCA_VERSION"; break;
   case       2121: c = "MQCA_CERT_LABEL"; break;
   case       2122: c = "MQCA_XR_VERSION"; break;
   case       2123: c = "MQCA_XR_SSL_CIPHER_SUITES"; break;
   case       2124: c = "MQCA_CLUS_CHL_NAME"; break;
   case       2125: c = "MQCA_CONN_AUTH"; break;
   case       2126: c = "MQCA_LDAP_BASE_DN_USERS"; break;
   case       2127: c = "MQCA_LDAP_SHORT_USER_FIELD"; break;
   case       2128: c = "MQCA_LDAP_USER_OBJECT_CLASS"; break;
   case       2129: c = "MQCA_LDAP_USER_ATTR_FIELD"; break;
   case       2130: c = "MQCA_SSL_CERT_ISSUER_NAME"; break;
   case       2131: c = "MQCA_QSG_CERT_LABEL"; break;
   case       2132: c = "MQCA_LDAP_BASE_DN_GROUPS"; break;
   case       2133: c = "MQCA_LDAP_GROUP_OBJECT_CLASS"; break;
   case       2134: c = "MQCA_LDAP_GROUP_ATTR_FIELD"; break;
   case       2135: c = "MQCA_LDAP_FIND_GROUP_FIELD"; break;
   case       2136: c = "MQCA_AMQP_VERSION"; break;
   case       2137: c = "MQCA_AMQP_SSL_CIPHER_SUITES"; break;
   case       4000: c = "MQCA_USER_LIST"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBCF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCBCF_NONE"; break;
   case          1: c = "MQCBCF_READA_BUFFER_EMPTY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBCT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCBCT_START_CALL"; break;
   case          2: c = "MQCBCT_STOP_CALL"; break;
   case          3: c = "MQCBCT_REGISTER_CALL"; break;
   case          4: c = "MQCBCT_DEREGISTER_CALL"; break;
   case          5: c = "MQCBCT_EVENT_CALL"; break;
   case          6: c = "MQCBCT_MSG_REMOVED"; break;
   case          7: c = "MQCBCT_MSG_NOT_REMOVED"; break;
   case          8: c = "MQCBCT_MC_EVENT_CALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBDO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCBDO_NONE"; break;
   case          1: c = "MQCBDO_START_CALL"; break;
   case          4: c = "MQCBDO_STOP_CALL"; break;
   case        256: c = "MQCBDO_REGISTER_CALL"; break;
   case        512: c = "MQCBDO_DEREGISTER_CALL"; break;
   case       8192: c = "MQCBDO_FAIL_IF_QUIESCING"; break;
   case      16384: c = "MQCBDO_EVENT_CALL"; break;
   case      32768: c = "MQCBDO_MC_EVENT_CALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQCBD_FULL_MSG_LENGTH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCBO_NONE"; break;
   case          1: c = "MQCBO_ADMIN_BAG"; break;
   case          2: c = "MQCBO_LIST_FORM_ALLOWED"; break;
   case          4: c = "MQCBO_REORDER_AS_REQUIRED"; break;
   case          8: c = "MQCBO_CHECK_SELECTORS"; break;
   case         16: c = "MQCBO_COMMAND_BAG"; break;
   case         32: c = "MQCBO_SYSTEM_BAG"; break;
   case         64: c = "MQCBO_GROUP_BAG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCBT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCBT_MESSAGE_CONSUMER"; break;
   case          2: c = "MQCBT_EVENT_HANDLER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCCSI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -4: c = "MQCCSI_AS_PUBLISHED"; break;
   case         -3: c = "MQCCSI_APPL"; break;
   case         -2: c = "MQCCSI_INHERIT"; break;
   case         -1: c = "MQCCSI_EMBEDDED"; break;
   case          0: c = "MQCCSI_DEFAULT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCCT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCCT_NO"; break;
   case          1: c = "MQCCT_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQCC_UNKNOWN"; break;
   case          0: c = "MQCC_OK"; break;
   case          1: c = "MQCC_WARNING"; break;
   case          2: c = "MQCC_FAILED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCDC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCDC_NO_SENDER_CONVERSION"; break;
   case          1: c = "MQCDC_SENDER_CONVERSION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFACCESS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFACCESS_ENABLED"; break;
   case          1: c = "MQCFACCESS_SUSPENDED"; break;
   case          2: c = "MQCFACCESS_DISABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFCONLOS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFCONLOS_TERMINATE"; break;
   case          1: c = "MQCFCONLOS_TOLERATE"; break;
   case          2: c = "MQCFCONLOS_ASQMGR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFOFFLD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFOFFLD_NONE"; break;
   case          1: c = "MQCFOFFLD_SMDS"; break;
   case          2: c = "MQCFOFFLD_DB2"; break;
   case          3: c = "MQCFOFFLD_BOTH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFOP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCFOP_LESS"; break;
   case          2: c = "MQCFOP_EQUAL"; break;
   case          3: c = "MQCFOP_NOT_GREATER"; break;
   case          4: c = "MQCFOP_GREATER"; break;
   case          5: c = "MQCFOP_NOT_EQUAL"; break;
   case          6: c = "MQCFOP_NOT_LESS"; break;
   case         10: c = "MQCFOP_CONTAINS"; break;
   case         13: c = "MQCFOP_EXCLUDES"; break;
   case         18: c = "MQCFOP_LIKE"; break;
   case         21: c = "MQCFOP_NOT_LIKE"; break;
   case         26: c = "MQCFOP_CONTAINS_GEN"; break;
   case         29: c = "MQCFOP_EXCLUDES_GEN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFO_REFRESH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFO_REFRESH_REPOSITORY_NO"; break;
   case          1: c = "MQCFO_REFRESH_REPOSITORY_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFO_REMOVE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFO_REMOVE_QUEUES_NO"; break;
   case          1: c = "MQCFO_REMOVE_QUEUES_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFR_NO"; break;
   case          1: c = "MQCFR_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFSTATUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFSTATUS_NOT_FOUND"; break;
   case          1: c = "MQCFSTATUS_ACTIVE"; break;
   case          2: c = "MQCFSTATUS_IN_RECOVER"; break;
   case          3: c = "MQCFSTATUS_IN_BACKUP"; break;
   case          4: c = "MQCFSTATUS_FAILED"; break;
   case          5: c = "MQCFSTATUS_NONE"; break;
   case          6: c = "MQCFSTATUS_UNKNOWN"; break;
   case          7: c = "MQCFSTATUS_RECOVERED"; break;
   case          8: c = "MQCFSTATUS_EMPTY"; break;
   case          9: c = "MQCFSTATUS_NEW"; break;
   case         20: c = "MQCFSTATUS_ADMIN_INCOMPLETE"; break;
   case         21: c = "MQCFSTATUS_NEVER_USED"; break;
   case         22: c = "MQCFSTATUS_NO_BACKUP"; break;
   case         23: c = "MQCFSTATUS_NOT_FAILED"; break;
   case         24: c = "MQCFSTATUS_NOT_RECOVERABLE"; break;
   case         25: c = "MQCFSTATUS_XES_ERROR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFTYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFTYPE_APPL"; break;
   case          1: c = "MQCFTYPE_ADMIN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCFT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCFT_NONE"; break;
   case          1: c = "MQCFT_COMMAND"; break;
   case          2: c = "MQCFT_RESPONSE"; break;
   case          3: c = "MQCFT_INTEGER"; break;
   case          4: c = "MQCFT_STRING"; break;
   case          5: c = "MQCFT_INTEGER_LIST"; break;
   case          6: c = "MQCFT_STRING_LIST"; break;
   case          7: c = "MQCFT_EVENT"; break;
   case          8: c = "MQCFT_USER"; break;
   case          9: c = "MQCFT_BYTE_STRING"; break;
   case         10: c = "MQCFT_TRACE_ROUTE"; break;
   case         12: c = "MQCFT_REPORT"; break;
   case         13: c = "MQCFT_INTEGER_FILTER"; break;
   case         14: c = "MQCFT_STRING_FILTER"; break;
   case         15: c = "MQCFT_BYTE_STRING_FILTER"; break;
   case         16: c = "MQCFT_COMMAND_XR"; break;
   case         17: c = "MQCFT_XR_MSG"; break;
   case         18: c = "MQCFT_XR_ITEM"; break;
   case         19: c = "MQCFT_XR_SUMMARY"; break;
   case         20: c = "MQCFT_GROUP"; break;
   case         21: c = "MQCFT_STATISTICS"; break;
   case         22: c = "MQCFT_ACCOUNTING"; break;
   case         23: c = "MQCFT_INTEGER64"; break;
   case         25: c = "MQCFT_INTEGER64_LIST"; break;
   case         26: c = "MQCFT_APP_ACTIVITY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCF_NONE"; break;
   case          1: c = "MQCF_DIST_LISTS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCGWI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQCGWI_DEFAULT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHAD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHAD_DISABLED"; break;
   case          1: c = "MQCHAD_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHIDS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHIDS_NOT_INDOUBT"; break;
   case          1: c = "MQCHIDS_INDOUBT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHK_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHK_OPTIONAL"; break;
   case          1: c = "MQCHK_NONE"; break;
   case          2: c = "MQCHK_REQUIRED_ADMIN"; break;
   case          3: c = "MQCHK_REQUIRED"; break;
   case          4: c = "MQCHK_AS_Q_MGR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHLA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHLA_DISABLED"; break;
   case          1: c = "MQCHLA_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHLD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQCHLD_ALL"; break;
   case          1: c = "MQCHLD_DEFAULT"; break;
   case          2: c = "MQCHLD_SHARED"; break;
   case          4: c = "MQCHLD_PRIVATE"; break;
   case          5: c = "MQCHLD_FIXSHARED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHRR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHRR_RESET_NOT_REQUESTED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHSH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHSH_RESTART_NO"; break;
   case          1: c = "MQCHSH_RESTART_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHSR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHSR_STOP_NOT_REQUESTED"; break;
   case          1: c = "MQCHSR_STOP_REQUESTED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHSSTATE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHSSTATE_OTHER"; break;
   case        100: c = "MQCHSSTATE_END_OF_BATCH"; break;
   case        200: c = "MQCHSSTATE_SENDING"; break;
   case        300: c = "MQCHSSTATE_RECEIVING"; break;
   case        400: c = "MQCHSSTATE_SERIALIZING"; break;
   case        500: c = "MQCHSSTATE_RESYNCHING"; break;
   case        600: c = "MQCHSSTATE_HEARTBEATING"; break;
   case        700: c = "MQCHSSTATE_IN_SCYEXIT"; break;
   case        800: c = "MQCHSSTATE_IN_RCVEXIT"; break;
   case        900: c = "MQCHSSTATE_IN_SENDEXIT"; break;
   case       1000: c = "MQCHSSTATE_IN_MSGEXIT"; break;
   case       1100: c = "MQCHSSTATE_IN_MREXIT"; break;
   case       1200: c = "MQCHSSTATE_IN_CHADEXIT"; break;
   case       1250: c = "MQCHSSTATE_NET_CONNECTING"; break;
   case       1300: c = "MQCHSSTATE_SSL_HANDSHAKING"; break;
   case       1400: c = "MQCHSSTATE_NAME_SERVER"; break;
   case       1500: c = "MQCHSSTATE_IN_MQPUT"; break;
   case       1600: c = "MQCHSSTATE_IN_MQGET"; break;
   case       1700: c = "MQCHSSTATE_IN_MQI_CALL"; break;
   case       1800: c = "MQCHSSTATE_COMPRESSING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCHS_INACTIVE"; break;
   case          1: c = "MQCHS_BINDING"; break;
   case          2: c = "MQCHS_STARTING"; break;
   case          3: c = "MQCHS_RUNNING"; break;
   case          4: c = "MQCHS_STOPPING"; break;
   case          5: c = "MQCHS_RETRYING"; break;
   case          6: c = "MQCHS_STOPPED"; break;
   case          7: c = "MQCHS_REQUESTING"; break;
   case          8: c = "MQCHS_PAUSED"; break;
   case          9: c = "MQCHS_DISCONNECTED"; break;
   case         13: c = "MQCHS_INITIALIZING"; break;
   case         14: c = "MQCHS_SWITCHING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHTAB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCHTAB_Q_MGR"; break;
   case          2: c = "MQCHTAB_CLNTCONN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCHT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCHT_SENDER"; break;
   case          2: c = "MQCHT_SERVER"; break;
   case          3: c = "MQCHT_RECEIVER"; break;
   case          4: c = "MQCHT_REQUESTER"; break;
   case          5: c = "MQCHT_ALL"; break;
   case          6: c = "MQCHT_CLNTCONN"; break;
   case          7: c = "MQCHT_SVRCONN"; break;
   case          8: c = "MQCHT_CLUSRCVR"; break;
   case          9: c = "MQCHT_CLUSSDR"; break;
   case         10: c = "MQCHT_MQTT"; break;
   case         11: c = "MQCHT_AMQP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCIH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCIH_NONE"; break;
   case          1: c = "MQCIH_PASS_EXPIRATION"; break;
   case          2: c = "MQCIH_REPLY_WITHOUT_NULLS"; break;
   case          4: c = "MQCIH_SYNC_ON_RETURN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCIT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCIT_MULTICAST"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLCT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCLCT_STATIC"; break;
   case          1: c = "MQCLCT_DYNAMIC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLROUTE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCLROUTE_DIRECT"; break;
   case          1: c = "MQCLROUTE_TOPIC_HOST"; break;
   case          2: c = "MQCLROUTE_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLRS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCLRS_LOCAL"; break;
   case          2: c = "MQCLRS_GLOBAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLRT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCLRT_RETAINED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLST_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCLST_ACTIVE"; break;
   case          1: c = "MQCLST_PENDING"; break;
   case          2: c = "MQCLST_INVALID"; break;
   case          3: c = "MQCLST_ERROR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCLT_PROGRAM"; break;
   case          2: c = "MQCLT_TRANSACTION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLWL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -3: c = "MQCLWL_USEQ_AS_Q_MGR"; break;
   case          0: c = "MQCLWL_USEQ_LOCAL"; break;
   case          1: c = "MQCLWL_USEQ_ANY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCLXQ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCLXQ_SCTQ"; break;
   case          1: c = "MQCLXQ_CHANNEL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCMDI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCMDI_CMDSCOPE_ACCEPTED"; break;
   case          2: c = "MQCMDI_CMDSCOPE_GENERATED"; break;
   case          3: c = "MQCMDI_CMDSCOPE_COMPLETED"; break;
   case          4: c = "MQCMDI_QSG_DISP_COMPLETED"; break;
   case          5: c = "MQCMDI_COMMAND_ACCEPTED"; break;
   case          6: c = "MQCMDI_CLUSTER_REQUEST_QUEUED"; break;
   case          7: c = "MQCMDI_CHANNEL_INIT_STARTED"; break;
   case         11: c = "MQCMDI_RECOVER_STARTED"; break;
   case         12: c = "MQCMDI_BACKUP_STARTED"; break;
   case         13: c = "MQCMDI_RECOVER_COMPLETED"; break;
   case         14: c = "MQCMDI_SEC_TIMER_ZERO"; break;
   case         16: c = "MQCMDI_REFRESH_CONFIGURATION"; break;
   case         17: c = "MQCMDI_SEC_SIGNOFF_ERROR"; break;
   case         18: c = "MQCMDI_IMS_BRIDGE_SUSPENDED"; break;
   case         19: c = "MQCMDI_DB2_SUSPENDED"; break;
   case         20: c = "MQCMDI_DB2_OBSOLETE_MSGS"; break;
   case         21: c = "MQCMDI_SEC_UPPERCASE"; break;
   case         22: c = "MQCMDI_SEC_MIXEDCASE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCMDL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case        100: c = "MQCMDL_LEVEL_1"; break;
   case        101: c = "MQCMDL_LEVEL_101"; break;
   case        110: c = "MQCMDL_LEVEL_110"; break;
   case        114: c = "MQCMDL_LEVEL_114"; break;
   case        120: c = "MQCMDL_LEVEL_120"; break;
   case        200: c = "MQCMDL_LEVEL_200"; break;
   case        201: c = "MQCMDL_LEVEL_201"; break;
   case        210: c = "MQCMDL_LEVEL_210"; break;
   case        211: c = "MQCMDL_LEVEL_211"; break;
   case        220: c = "MQCMDL_LEVEL_220"; break;
   case        221: c = "MQCMDL_LEVEL_221"; break;
   case        230: c = "MQCMDL_LEVEL_230"; break;
   case        320: c = "MQCMDL_LEVEL_320"; break;
   case        420: c = "MQCMDL_LEVEL_420"; break;
   case        500: c = "MQCMDL_LEVEL_500"; break;
   case        510: c = "MQCMDL_LEVEL_510"; break;
   case        520: c = "MQCMDL_LEVEL_520"; break;
   case        530: c = "MQCMDL_LEVEL_530"; break;
   case        531: c = "MQCMDL_LEVEL_531"; break;
   case        600: c = "MQCMDL_LEVEL_600"; break;
   case        700: c = "MQCMDL_LEVEL_700"; break;
   case        701: c = "MQCMDL_LEVEL_701"; break;
   case        710: c = "MQCMDL_LEVEL_710"; break;
   case        711: c = "MQCMDL_LEVEL_711"; break;
   case        750: c = "MQCMDL_LEVEL_750"; break;
   case        800: c = "MQCMDL_LEVEL_800"; break;
   case        801: c = "MQCMDL_LEVEL_801"; break;
   case        802: c = "MQCMDL_LEVEL_802"; break;
   case        900: c = "MQCMDL_LEVEL_900"; break;
   case        901: c = "MQCMDL_LEVEL_901"; break;
   case        902: c = "MQCMDL_LEVEL_902"; break;
   case        903: c = "MQCMDL_LEVEL_903"; break;
   case        904: c = "MQCMDL_LEVEL_904"; break;
   case        905: c = "MQCMDL_LEVEL_905"; break;
   case        910: c = "MQCMDL_LEVEL_910"; break;
   case        911: c = "MQCMDL_LEVEL_911"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCMD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCMD_NONE"; break;
   case          1: c = "MQCMD_CHANGE_Q_MGR"; break;
   case          2: c = "MQCMD_INQUIRE_Q_MGR"; break;
   case          3: c = "MQCMD_CHANGE_PROCESS"; break;
   case          4: c = "MQCMD_COPY_PROCESS"; break;
   case          5: c = "MQCMD_CREATE_PROCESS"; break;
   case          6: c = "MQCMD_DELETE_PROCESS"; break;
   case          7: c = "MQCMD_INQUIRE_PROCESS"; break;
   case          8: c = "MQCMD_CHANGE_Q"; break;
   case          9: c = "MQCMD_CLEAR_Q"; break;
   case         10: c = "MQCMD_COPY_Q"; break;
   case         11: c = "MQCMD_CREATE_Q"; break;
   case         12: c = "MQCMD_DELETE_Q"; break;
   case         13: c = "MQCMD_INQUIRE_Q"; break;
   case         16: c = "MQCMD_REFRESH_Q_MGR"; break;
   case         17: c = "MQCMD_RESET_Q_STATS"; break;
   case         18: c = "MQCMD_INQUIRE_Q_NAMES"; break;
   case         19: c = "MQCMD_INQUIRE_PROCESS_NAMES"; break;
   case         20: c = "MQCMD_INQUIRE_CHANNEL_NAMES"; break;
   case         21: c = "MQCMD_CHANGE_CHANNEL"; break;
   case         22: c = "MQCMD_COPY_CHANNEL"; break;
   case         23: c = "MQCMD_CREATE_CHANNEL"; break;
   case         24: c = "MQCMD_DELETE_CHANNEL"; break;
   case         25: c = "MQCMD_INQUIRE_CHANNEL"; break;
   case         26: c = "MQCMD_PING_CHANNEL"; break;
   case         27: c = "MQCMD_RESET_CHANNEL"; break;
   case         28: c = "MQCMD_START_CHANNEL"; break;
   case         29: c = "MQCMD_STOP_CHANNEL"; break;
   case         30: c = "MQCMD_START_CHANNEL_INIT"; break;
   case         31: c = "MQCMD_START_CHANNEL_LISTENER"; break;
   case         32: c = "MQCMD_CHANGE_NAMELIST"; break;
   case         33: c = "MQCMD_COPY_NAMELIST"; break;
   case         34: c = "MQCMD_CREATE_NAMELIST"; break;
   case         35: c = "MQCMD_DELETE_NAMELIST"; break;
   case         36: c = "MQCMD_INQUIRE_NAMELIST"; break;
   case         37: c = "MQCMD_INQUIRE_NAMELIST_NAMES"; break;
   case         38: c = "MQCMD_ESCAPE"; break;
   case         39: c = "MQCMD_RESOLVE_CHANNEL"; break;
   case         40: c = "MQCMD_PING_Q_MGR"; break;
   case         41: c = "MQCMD_INQUIRE_Q_STATUS"; break;
   case         42: c = "MQCMD_INQUIRE_CHANNEL_STATUS"; break;
   case         43: c = "MQCMD_CONFIG_EVENT"; break;
   case         44: c = "MQCMD_Q_MGR_EVENT"; break;
   case         45: c = "MQCMD_PERFM_EVENT"; break;
   case         46: c = "MQCMD_CHANNEL_EVENT"; break;
   case         60: c = "MQCMD_DELETE_PUBLICATION"; break;
   case         61: c = "MQCMD_DEREGISTER_PUBLISHER"; break;
   case         62: c = "MQCMD_DEREGISTER_SUBSCRIBER"; break;
   case         63: c = "MQCMD_PUBLISH"; break;
   case         64: c = "MQCMD_REGISTER_PUBLISHER"; break;
   case         65: c = "MQCMD_REGISTER_SUBSCRIBER"; break;
   case         66: c = "MQCMD_REQUEST_UPDATE"; break;
   case         67: c = "MQCMD_BROKER_INTERNAL"; break;
   case         69: c = "MQCMD_ACTIVITY_MSG"; break;
   case         70: c = "MQCMD_INQUIRE_CLUSTER_Q_MGR"; break;
   case         71: c = "MQCMD_RESUME_Q_MGR_CLUSTER"; break;
   case         72: c = "MQCMD_SUSPEND_Q_MGR_CLUSTER"; break;
   case         73: c = "MQCMD_REFRESH_CLUSTER"; break;
   case         74: c = "MQCMD_RESET_CLUSTER"; break;
   case         75: c = "MQCMD_TRACE_ROUTE"; break;
   case         78: c = "MQCMD_REFRESH_SECURITY"; break;
   case         79: c = "MQCMD_CHANGE_AUTH_INFO"; break;
   case         80: c = "MQCMD_COPY_AUTH_INFO"; break;
   case         81: c = "MQCMD_CREATE_AUTH_INFO"; break;
   case         82: c = "MQCMD_DELETE_AUTH_INFO"; break;
   case         83: c = "MQCMD_INQUIRE_AUTH_INFO"; break;
   case         84: c = "MQCMD_INQUIRE_AUTH_INFO_NAMES"; break;
   case         85: c = "MQCMD_INQUIRE_CONNECTION"; break;
   case         86: c = "MQCMD_STOP_CONNECTION"; break;
   case         87: c = "MQCMD_INQUIRE_AUTH_RECS"; break;
   case         88: c = "MQCMD_INQUIRE_ENTITY_AUTH"; break;
   case         89: c = "MQCMD_DELETE_AUTH_REC"; break;
   case         90: c = "MQCMD_SET_AUTH_REC"; break;
   case         91: c = "MQCMD_LOGGER_EVENT"; break;
   case         92: c = "MQCMD_RESET_Q_MGR"; break;
   case         93: c = "MQCMD_CHANGE_LISTENER"; break;
   case         94: c = "MQCMD_COPY_LISTENER"; break;
   case         95: c = "MQCMD_CREATE_LISTENER"; break;
   case         96: c = "MQCMD_DELETE_LISTENER"; break;
   case         97: c = "MQCMD_INQUIRE_LISTENER"; break;
   case         98: c = "MQCMD_INQUIRE_LISTENER_STATUS"; break;
   case         99: c = "MQCMD_COMMAND_EVENT"; break;
   case        100: c = "MQCMD_CHANGE_SECURITY"; break;
   case        101: c = "MQCMD_CHANGE_CF_STRUC"; break;
   case        102: c = "MQCMD_CHANGE_STG_CLASS"; break;
   case        103: c = "MQCMD_CHANGE_TRACE"; break;
   case        104: c = "MQCMD_ARCHIVE_LOG"; break;
   case        105: c = "MQCMD_BACKUP_CF_STRUC"; break;
   case        106: c = "MQCMD_CREATE_BUFFER_POOL"; break;
   case        107: c = "MQCMD_CREATE_PAGE_SET"; break;
   case        108: c = "MQCMD_CREATE_CF_STRUC"; break;
   case        109: c = "MQCMD_CREATE_STG_CLASS"; break;
   case        110: c = "MQCMD_COPY_CF_STRUC"; break;
   case        111: c = "MQCMD_COPY_STG_CLASS"; break;
   case        112: c = "MQCMD_DELETE_CF_STRUC"; break;
   case        113: c = "MQCMD_DELETE_STG_CLASS"; break;
   case        114: c = "MQCMD_INQUIRE_ARCHIVE"; break;
   case        115: c = "MQCMD_INQUIRE_CF_STRUC"; break;
   case        116: c = "MQCMD_INQUIRE_CF_STRUC_STATUS"; break;
   case        117: c = "MQCMD_INQUIRE_CMD_SERVER"; break;
   case        118: c = "MQCMD_INQUIRE_CHANNEL_INIT"; break;
   case        119: c = "MQCMD_INQUIRE_QSG"; break;
   case        120: c = "MQCMD_INQUIRE_LOG"; break;
   case        121: c = "MQCMD_INQUIRE_SECURITY"; break;
   case        122: c = "MQCMD_INQUIRE_STG_CLASS"; break;
   case        123: c = "MQCMD_INQUIRE_SYSTEM"; break;
   case        124: c = "MQCMD_INQUIRE_THREAD"; break;
   case        125: c = "MQCMD_INQUIRE_TRACE"; break;
   case        126: c = "MQCMD_INQUIRE_USAGE"; break;
   case        127: c = "MQCMD_MOVE_Q"; break;
   case        128: c = "MQCMD_RECOVER_BSDS"; break;
   case        129: c = "MQCMD_RECOVER_CF_STRUC"; break;
   case        130: c = "MQCMD_RESET_TPIPE"; break;
   case        131: c = "MQCMD_RESOLVE_INDOUBT"; break;
   case        132: c = "MQCMD_RESUME_Q_MGR"; break;
   case        133: c = "MQCMD_REVERIFY_SECURITY"; break;
   case        134: c = "MQCMD_SET_ARCHIVE"; break;
   case        136: c = "MQCMD_SET_LOG"; break;
   case        137: c = "MQCMD_SET_SYSTEM"; break;
   case        138: c = "MQCMD_START_CMD_SERVER"; break;
   case        139: c = "MQCMD_START_Q_MGR"; break;
   case        140: c = "MQCMD_START_TRACE"; break;
   case        141: c = "MQCMD_STOP_CHANNEL_INIT"; break;
   case        142: c = "MQCMD_STOP_CHANNEL_LISTENER"; break;
   case        143: c = "MQCMD_STOP_CMD_SERVER"; break;
   case        144: c = "MQCMD_STOP_Q_MGR"; break;
   case        145: c = "MQCMD_STOP_TRACE"; break;
   case        146: c = "MQCMD_SUSPEND_Q_MGR"; break;
   case        147: c = "MQCMD_INQUIRE_CF_STRUC_NAMES"; break;
   case        148: c = "MQCMD_INQUIRE_STG_CLASS_NAMES"; break;
   case        149: c = "MQCMD_CHANGE_SERVICE"; break;
   case        150: c = "MQCMD_COPY_SERVICE"; break;
   case        151: c = "MQCMD_CREATE_SERVICE"; break;
   case        152: c = "MQCMD_DELETE_SERVICE"; break;
   case        153: c = "MQCMD_INQUIRE_SERVICE"; break;
   case        154: c = "MQCMD_INQUIRE_SERVICE_STATUS"; break;
   case        155: c = "MQCMD_START_SERVICE"; break;
   case        156: c = "MQCMD_STOP_SERVICE"; break;
   case        157: c = "MQCMD_DELETE_BUFFER_POOL"; break;
   case        158: c = "MQCMD_DELETE_PAGE_SET"; break;
   case        159: c = "MQCMD_CHANGE_BUFFER_POOL"; break;
   case        160: c = "MQCMD_CHANGE_PAGE_SET"; break;
   case        161: c = "MQCMD_INQUIRE_Q_MGR_STATUS"; break;
   case        162: c = "MQCMD_CREATE_LOG"; break;
   case        164: c = "MQCMD_STATISTICS_MQI"; break;
   case        165: c = "MQCMD_STATISTICS_Q"; break;
   case        166: c = "MQCMD_STATISTICS_CHANNEL"; break;
   case        167: c = "MQCMD_ACCOUNTING_MQI"; break;
   case        168: c = "MQCMD_ACCOUNTING_Q"; break;
   case        169: c = "MQCMD_INQUIRE_AUTH_SERVICE"; break;
   case        170: c = "MQCMD_CHANGE_TOPIC"; break;
   case        171: c = "MQCMD_COPY_TOPIC"; break;
   case        172: c = "MQCMD_CREATE_TOPIC"; break;
   case        173: c = "MQCMD_DELETE_TOPIC"; break;
   case        174: c = "MQCMD_INQUIRE_TOPIC"; break;
   case        175: c = "MQCMD_INQUIRE_TOPIC_NAMES"; break;
   case        176: c = "MQCMD_INQUIRE_SUBSCRIPTION"; break;
   case        177: c = "MQCMD_CREATE_SUBSCRIPTION"; break;
   case        178: c = "MQCMD_CHANGE_SUBSCRIPTION"; break;
   case        179: c = "MQCMD_DELETE_SUBSCRIPTION"; break;
   case        181: c = "MQCMD_COPY_SUBSCRIPTION"; break;
   case        182: c = "MQCMD_INQUIRE_SUB_STATUS"; break;
   case        183: c = "MQCMD_INQUIRE_TOPIC_STATUS"; break;
   case        184: c = "MQCMD_CLEAR_TOPIC_STRING"; break;
   case        185: c = "MQCMD_INQUIRE_PUBSUB_STATUS"; break;
   case        186: c = "MQCMD_INQUIRE_SMDS"; break;
   case        187: c = "MQCMD_CHANGE_SMDS"; break;
   case        188: c = "MQCMD_RESET_SMDS"; break;
   case        190: c = "MQCMD_CREATE_COMM_INFO"; break;
   case        191: c = "MQCMD_INQUIRE_COMM_INFO"; break;
   case        192: c = "MQCMD_CHANGE_COMM_INFO"; break;
   case        193: c = "MQCMD_COPY_COMM_INFO"; break;
   case        194: c = "MQCMD_DELETE_COMM_INFO"; break;
   case        195: c = "MQCMD_PURGE_CHANNEL"; break;
   case        196: c = "MQCMD_MQXR_DIAGNOSTICS"; break;
   case        197: c = "MQCMD_START_SMDSCONN"; break;
   case        198: c = "MQCMD_STOP_SMDSCONN"; break;
   case        199: c = "MQCMD_INQUIRE_SMDSCONN"; break;
   case        200: c = "MQCMD_INQUIRE_MQXR_STATUS"; break;
   case        201: c = "MQCMD_START_CLIENT_TRACE"; break;
   case        202: c = "MQCMD_STOP_CLIENT_TRACE"; break;
   case        203: c = "MQCMD_SET_CHLAUTH_REC"; break;
   case        204: c = "MQCMD_INQUIRE_CHLAUTH_RECS"; break;
   case        205: c = "MQCMD_INQUIRE_PROT_POLICY"; break;
   case        206: c = "MQCMD_CREATE_PROT_POLICY"; break;
   case        207: c = "MQCMD_DELETE_PROT_POLICY"; break;
   case        208: c = "MQCMD_CHANGE_PROT_POLICY"; break;
   case        209: c = "MQCMD_ACTIVITY_TRACE"; break;
   case        213: c = "MQCMD_RESET_CF_STRUC"; break;
   case        214: c = "MQCMD_INQUIRE_XR_CAPABILITY"; break;
   case        216: c = "MQCMD_INQUIRE_AMQP_CAPABILITY"; break;
   case        217: c = "MQCMD_AMQP_DIAGNOSTICS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCMHO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCMHO_NONE"; break;
   case          1: c = "MQCMHO_NO_VALIDATION"; break;
   case          2: c = "MQCMHO_VALIDATE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCNO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCNO_NONE"; break;
   case          1: c = "MQCNO_FASTPATH_BINDING"; break;
   case          2: c = "MQCNO_SERIALIZE_CONN_TAG_Q_MGR"; break;
   case          4: c = "MQCNO_SERIALIZE_CONN_TAG_QSG"; break;
   case          8: c = "MQCNO_RESTRICT_CONN_TAG_Q_MGR"; break;
   case         16: c = "MQCNO_RESTRICT_CONN_TAG_QSG"; break;
   case         32: c = "MQCNO_HANDLE_SHARE_NONE"; break;
   case         64: c = "MQCNO_HANDLE_SHARE_BLOCK"; break;
   case        128: c = "MQCNO_HANDLE_SHARE_NO_BLOCK"; break;
   case        256: c = "MQCNO_SHARED_BINDING"; break;
   case        512: c = "MQCNO_ISOLATED_BINDING"; break;
   case       1024: c = "MQCNO_LOCAL_BINDING"; break;
   case       2048: c = "MQCNO_CLIENT_BINDING"; break;
   case       4096: c = "MQCNO_ACCOUNTING_MQI_ENABLED"; break;
   case       8192: c = "MQCNO_ACCOUNTING_MQI_DISABLED"; break;
   case      16384: c = "MQCNO_ACCOUNTING_Q_ENABLED"; break;
   case      32768: c = "MQCNO_ACCOUNTING_Q_DISABLED"; break;
   case      65536: c = "MQCNO_NO_CONV_SHARING"; break;
   case     262144: c = "MQCNO_ALL_CONVS_SHARE"; break;
   case     524288: c = "MQCNO_CD_FOR_OUTPUT_ONLY"; break;
   case    1048576: c = "MQCNO_USE_CD_SELECTION"; break;
   case   16777216: c = "MQCNO_RECONNECT"; break;
   case   33554432: c = "MQCNO_RECONNECT_DISABLED"; break;
   case   67108864: c = "MQCNO_RECONNECT_Q_MGR"; break;
   case  134217728: c = "MQCNO_ACTIVITY_TRACE_ENABLED"; break;
   case  268435456: c = "MQCNO_ACTIVITY_TRACE_DISABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCODL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQCODL_AS_INPUT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCOMPRESS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQCOMPRESS_NOT_AVAILABLE"; break;
   case          0: c = "MQCOMPRESS_NONE"; break;
   case          1: c = "MQCOMPRESS_RLE"; break;
   case          2: c = "MQCOMPRESS_ZLIBFAST"; break;
   case          4: c = "MQCOMPRESS_ZLIBHIGH"; break;
   case          8: c = "MQCOMPRESS_SYSTEM"; break;
   case  268435455: c = "MQCOMPRESS_ANY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCOPY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCOPY_NONE"; break;
   case          1: c = "MQCOPY_ALL"; break;
   case          2: c = "MQCOPY_FORWARD"; break;
   case          4: c = "MQCOPY_PUBLISH"; break;
   case          8: c = "MQCOPY_REPLY"; break;
   case         16: c = "MQCOPY_REPORT"; break;
   case         22: c = "MQCOPY_DEFAULT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCO_NONE"; break;
   case          1: c = "MQCO_DELETE"; break;
   case          2: c = "MQCO_DELETE_PURGE"; break;
   case          4: c = "MQCO_KEEP_SUB"; break;
   case          8: c = "MQCO_REMOVE_SUB"; break;
   case         32: c = "MQCO_QUIESCE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCQT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQCQT_LOCAL_Q"; break;
   case          2: c = "MQCQT_ALIAS_Q"; break;
   case          3: c = "MQCQT_REMOTE_Q"; break;
   case          4: c = "MQCQT_Q_MGR_ALIAS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCRC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCRC_OK"; break;
   case          1: c = "MQCRC_CICS_EXEC_ERROR"; break;
   case          2: c = "MQCRC_MQ_API_ERROR"; break;
   case          3: c = "MQCRC_BRIDGE_ERROR"; break;
   case          4: c = "MQCRC_BRIDGE_ABEND"; break;
   case          5: c = "MQCRC_APPLICATION_ABEND"; break;
   case          6: c = "MQCRC_SECURITY_ERROR"; break;
   case          7: c = "MQCRC_PROGRAM_NOT_AVAILABLE"; break;
   case          8: c = "MQCRC_BRIDGE_TIMEOUT"; break;
   case          9: c = "MQCRC_TRANSID_NOT_AVAILABLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCSP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCSP_AUTH_NONE"; break;
   case          1: c = "MQCSP_AUTH_USER_ID_AND_PWD"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCSRV_CONVERT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCSRV_CONVERT_NO"; break;
   case          1: c = "MQCSRV_CONVERT_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCSRV_DLQ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCSRV_DLQ_NO"; break;
   case          1: c = "MQCSRV_DLQ_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCS_NONE"; break;
   case          1: c = "MQCS_SUSPENDED_TEMPORARY"; break;
   case          2: c = "MQCS_SUSPENDED_USER_ACTION"; break;
   case          3: c = "MQCS_SUSPENDED"; break;
   case          4: c = "MQCS_STOPPED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCTES_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCTES_NOSYNC"; break;
   case        256: c = "MQCTES_COMMIT"; break;
   case       4352: c = "MQCTES_BACKOUT"; break;
   case      65536: c = "MQCTES_ENDTASK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCTLO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQCTLO_NONE"; break;
   case          1: c = "MQCTLO_THREAD_AFFINITY"; break;
   case       8192: c = "MQCTLO_FAIL_IF_QUIESCING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQCUOWC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         16: c = "MQCUOWC_MIDDLE"; break;
   case        256: c = "MQCUOWC_COMMIT"; break;
   case        273: c = "MQCUOWC_ONLY"; break;
   case       4352: c = "MQCUOWC_BACKOUT"; break;
   case      65536: c = "MQCUOWC_CONTINUE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDCC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDCC_NONE"; break;
   case          1: c = "MQDCC_DEFAULT_CONVERSION"; break;
   case          2: c = "MQDCC_FILL_TARGET_BUFFER"; break;
   case          4: c = "MQDCC_INT_DEFAULT_CONVERSION"; break;
   case         16: c = "MQDCC_SOURCE_ENC_NORMAL"; break;
   case         32: c = "MQDCC_SOURCE_ENC_REVERSED"; break;
   case        240: c = "MQDCC_SOURCE_ENC_MASK"; break;
   case        256: c = "MQDCC_TARGET_ENC_NORMAL"; break;
   case        512: c = "MQDCC_TARGET_ENC_REVERSED"; break;
   case       3840: c = "MQDCC_TARGET_ENC_MASK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQDC_MANAGED"; break;
   case          2: c = "MQDC_PROVIDED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDELO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDELO_NONE"; break;
   case          4: c = "MQDELO_LOCAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDHF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDHF_NONE"; break;
   case          1: c = "MQDHF_NEW_MSG_IDS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDISCONNECT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDISCONNECT_NORMAL"; break;
   case          1: c = "MQDISCONNECT_IMPLICIT"; break;
   case          2: c = "MQDISCONNECT_Q_MGR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDLV_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDLV_AS_PARENT"; break;
   case          1: c = "MQDLV_ALL"; break;
   case          2: c = "MQDLV_ALL_DUR"; break;
   case          3: c = "MQDLV_ALL_AVAIL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDL_NOT_SUPPORTED"; break;
   case          1: c = "MQDL_SUPPORTED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDMHO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDMHO_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDMPO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDMPO_NONE"; break;
   case          1: c = "MQDMPO_DEL_PROP_UNDER_CURSOR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDNSWLM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDNSWLM_NO"; break;
   case          1: c = "MQDNSWLM_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDOPT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDOPT_RESOLVED"; break;
   case          1: c = "MQDOPT_DEFINED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDSB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDSB_DEFAULT"; break;
   case          1: c = "MQDSB_8K"; break;
   case          2: c = "MQDSB_16K"; break;
   case          3: c = "MQDSB_32K"; break;
   case          4: c = "MQDSB_64K"; break;
   case          5: c = "MQDSB_128K"; break;
   case          6: c = "MQDSB_256K"; break;
   case          7: c = "MQDSB_512K"; break;
   case          8: c = "MQDSB_1M"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQDSE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQDSE_DEFAULT"; break;
   case          1: c = "MQDSE_YES"; break;
   case          2: c = "MQDSE_NO"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          2: c = "MQEC_MSG_ARRIVED"; break;
   case          3: c = "MQEC_WAIT_INTERVAL_EXPIRED"; break;
   case          4: c = "MQEC_WAIT_CANCELED"; break;
   case          5: c = "MQEC_Q_MGR_QUIESCING"; break;
   case          6: c = "MQEC_CONNECTION_QUIESCING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQEI_UNLIMITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQENC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case      -4096: c = "MQENC_RESERVED_MASK"; break;
   case         -1: c = "MQENC_AS_PUBLISHED"; break;
   case          1: c = "MQENC_INTEGER_NORMAL"; break;
   case          2: c = "MQENC_INTEGER_REVERSED"; break;
   case         15: c = "MQENC_INTEGER_MASK"; break;
   case         16: c = "MQENC_DECIMAL_NORMAL"; break;
   case         32: c = "MQENC_DECIMAL_REVERSED"; break;
   case        240: c = "MQENC_DECIMAL_MASK"; break;
   case        256: c = "MQENC_FLOAT_IEEE_NORMAL"; break;
   case        273: c = "MQENC_NORMAL"; break;
   case        512: c = "MQENC_FLOAT_IEEE_REVERSED"; break;
   case        546: c = "MQENC_REVERSED"; break;
   case        768: c = "MQENC_FLOAT_S390"; break;
   case        785: c = "MQENC_S390"; break;
   case       1024: c = "MQENC_FLOAT_TNS"; break;
   case       1041: c = "MQENC_TNS"; break;
   case       3840: c = "MQENC_FLOAT_MASK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEPH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEPH_NONE"; break;
   case          1: c = "MQEPH_CCSID_EMBEDDED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQET_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQET_MQSC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEVO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEVO_OTHER"; break;
   case          1: c = "MQEVO_CONSOLE"; break;
   case          2: c = "MQEVO_INIT"; break;
   case          3: c = "MQEVO_MSG"; break;
   case          4: c = "MQEVO_MQSET"; break;
   case          5: c = "MQEVO_INTERNAL"; break;
   case          6: c = "MQEVO_MQSUB"; break;
   case          7: c = "MQEVO_CTLMSG"; break;
   case          8: c = "MQEVO_REST"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEVR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEVR_DISABLED"; break;
   case          1: c = "MQEVR_ENABLED"; break;
   case          2: c = "MQEVR_EXCEPTION"; break;
   case          3: c = "MQEVR_NO_DISPLAY"; break;
   case          4: c = "MQEVR_API_ONLY"; break;
   case          5: c = "MQEVR_ADMIN_ONLY"; break;
   case          6: c = "MQEVR_USER_ONLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEXPI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEXPI_OFF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEXTATTRS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEXTATTRS_ALL"; break;
   case          1: c = "MQEXTATTRS_NONDEF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQEXT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQEXT_ALL"; break;
   case          1: c = "MQEXT_OBJECT"; break;
   case          2: c = "MQEXT_AUTHORITY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQFB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQFB_NONE"; break;
   case        256: c = "MQFB_QUIT"; break;
   case        258: c = "MQFB_EXPIRATION"; break;
   case        259: c = "MQFB_COA"; break;
   case        260: c = "MQFB_COD"; break;
   case        262: c = "MQFB_CHANNEL_COMPLETED"; break;
   case        263: c = "MQFB_CHANNEL_FAIL_RETRY"; break;
   case        264: c = "MQFB_CHANNEL_FAIL"; break;
   case        265: c = "MQFB_APPL_CANNOT_BE_STARTED"; break;
   case        266: c = "MQFB_TM_ERROR"; break;
   case        267: c = "MQFB_APPL_TYPE_ERROR"; break;
   case        268: c = "MQFB_STOPPED_BY_MSG_EXIT"; break;
   case        269: c = "MQFB_ACTIVITY"; break;
   case        271: c = "MQFB_XMIT_Q_MSG_ERROR"; break;
   case        275: c = "MQFB_PAN"; break;
   case        276: c = "MQFB_NAN"; break;
   case        277: c = "MQFB_STOPPED_BY_CHAD_EXIT"; break;
   case        279: c = "MQFB_STOPPED_BY_PUBSUB_EXIT"; break;
   case        280: c = "MQFB_NOT_A_REPOSITORY_MSG"; break;
   case        281: c = "MQFB_BIND_OPEN_CLUSRCVR_DEL"; break;
   case        282: c = "MQFB_MAX_ACTIVITIES"; break;
   case        283: c = "MQFB_NOT_FORWARDED"; break;
   case        284: c = "MQFB_NOT_DELIVERED"; break;
   case        285: c = "MQFB_UNSUPPORTED_FORWARDING"; break;
   case        286: c = "MQFB_UNSUPPORTED_DELIVERY"; break;
   case        291: c = "MQFB_DATA_LENGTH_ZERO"; break;
   case        292: c = "MQFB_DATA_LENGTH_NEGATIVE"; break;
   case        293: c = "MQFB_DATA_LENGTH_TOO_BIG"; break;
   case        294: c = "MQFB_BUFFER_OVERFLOW"; break;
   case        295: c = "MQFB_LENGTH_OFF_BY_ONE"; break;
   case        296: c = "MQFB_IIH_ERROR"; break;
   case        298: c = "MQFB_NOT_AUTHORIZED_FOR_IMS"; break;
   case        300: c = "MQFB_IMS_ERROR"; break;
   case        401: c = "MQFB_CICS_INTERNAL_ERROR"; break;
   case        402: c = "MQFB_CICS_NOT_AUTHORIZED"; break;
   case        403: c = "MQFB_CICS_BRIDGE_FAILURE"; break;
   case        404: c = "MQFB_CICS_CORREL_ID_ERROR"; break;
   case        405: c = "MQFB_CICS_CCSID_ERROR"; break;
   case        406: c = "MQFB_CICS_ENCODING_ERROR"; break;
   case        407: c = "MQFB_CICS_CIH_ERROR"; break;
   case        408: c = "MQFB_CICS_UOW_ERROR"; break;
   case        409: c = "MQFB_CICS_COMMAREA_ERROR"; break;
   case        410: c = "MQFB_CICS_APPL_NOT_STARTED"; break;
   case        411: c = "MQFB_CICS_APPL_ABENDED"; break;
   case        412: c = "MQFB_CICS_DLQ_ERROR"; break;
   case        413: c = "MQFB_CICS_UOW_BACKED_OUT"; break;
   case        501: c = "MQFB_PUBLICATIONS_ON_REQUEST"; break;
   case        502: c = "MQFB_SUBSCRIBER_IS_PUBLISHER"; break;
   case        503: c = "MQFB_MSG_SCOPE_MISMATCH"; break;
   case        504: c = "MQFB_SELECTOR_MISMATCH"; break;
   case        505: c = "MQFB_NOT_A_GROUPUR_MSG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQFC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQFC_NO"; break;
   case          1: c = "MQFC_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQFIELD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       8000: c = "MQFIELD_WQR_StrucId"; break;
   case       8001: c = "MQFIELD_WQR_Version"; break;
   case       8002: c = "MQFIELD_WQR_StrucLength"; break;
   case       8003: c = "MQFIELD_WQR_QFlags"; break;
   case       8004: c = "MQFIELD_WQR_QName"; break;
   case       8005: c = "MQFIELD_WQR_QMgrIdentifier"; break;
   case       8006: c = "MQFIELD_WQR_ClusterRecOffset"; break;
   case       8007: c = "MQFIELD_WQR_QType"; break;
   case       8008: c = "MQFIELD_WQR_QDesc"; break;
   case       8009: c = "MQFIELD_WQR_DefBind"; break;
   case       8010: c = "MQFIELD_WQR_DefPersistence"; break;
   case       8011: c = "MQFIELD_WQR_DefPriority"; break;
   case       8012: c = "MQFIELD_WQR_InhibitPut"; break;
   case       8013: c = "MQFIELD_WQR_CLWLQueuePriority"; break;
   case       8014: c = "MQFIELD_WQR_CLWLQueueRank"; break;
   case       8015: c = "MQFIELD_WQR_DefPutResponse"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQFUN_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQFUN_TYPE_UNKNOWN"; break;
   case          1: c = "MQFUN_TYPE_JVM"; break;
   case          2: c = "MQFUN_TYPE_PROGRAM"; break;
   case          3: c = "MQFUN_TYPE_PROCEDURE"; break;
   case          4: c = "MQFUN_TYPE_USERDEF"; break;
   case          5: c = "MQFUN_TYPE_COMMAND"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQGACF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       8001: c = "MQGACF_COMMAND_CONTEXT"; break;
   case       8002: c = "MQGACF_COMMAND_DATA"; break;
   case       8003: c = "MQGACF_TRACE_ROUTE"; break;
   case       8004: c = "MQGACF_OPERATION"; break;
   case       8005: c = "MQGACF_ACTIVITY"; break;
   case       8006: c = "MQGACF_EMBEDDED_MQMD"; break;
   case       8007: c = "MQGACF_MESSAGE"; break;
   case       8008: c = "MQGACF_MQMD"; break;
   case       8009: c = "MQGACF_VALUE_NAMING"; break;
   case       8010: c = "MQGACF_Q_ACCOUNTING_DATA"; break;
   case       8011: c = "MQGACF_Q_STATISTICS_DATA"; break;
   case       8012: c = "MQGACF_CHL_STATISTICS_DATA"; break;
   case       8013: c = "MQGACF_ACTIVITY_TRACE"; break;
   case       8014: c = "MQGACF_APP_DIST_LIST"; break;
   case       8015: c = "MQGACF_MONITOR_CLASS"; break;
   case       8016: c = "MQGACF_MONITOR_TYPE"; break;
   case       8017: c = "MQGACF_MONITOR_ELEMENT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQGMO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQGMO_NONE"; break;
   case          1: c = "MQGMO_WAIT"; break;
   case          2: c = "MQGMO_SYNCPOINT"; break;
   case          4: c = "MQGMO_NO_SYNCPOINT"; break;
   case          8: c = "MQGMO_SET_SIGNAL"; break;
   case         32: c = "MQGMO_BROWSE_NEXT"; break;
   case         64: c = "MQGMO_ACCEPT_TRUNCATED_MSG"; break;
   case        128: c = "MQGMO_MARK_SKIP_BACKOUT"; break;
   case        256: c = "MQGMO_MSG_UNDER_CURSOR"; break;
   case        512: c = "MQGMO_LOCK"; break;
   case       1024: c = "MQGMO_UNLOCK"; break;
   case       2048: c = "MQGMO_BROWSE_MSG_UNDER_CURSOR"; break;
   case       4096: c = "MQGMO_SYNCPOINT_IF_PERSISTENT"; break;
   case       8192: c = "MQGMO_FAIL_IF_QUIESCING"; break;
   case      16384: c = "MQGMO_CONVERT"; break;
   case      32768: c = "MQGMO_LOGICAL_ORDER"; break;
   case      65536: c = "MQGMO_COMPLETE_MSG"; break;
   case     131072: c = "MQGMO_ALL_MSGS_AVAILABLE"; break;
   case     262144: c = "MQGMO_ALL_SEGMENTS_AVAILABLE"; break;
   case    1048576: c = "MQGMO_MARK_BROWSE_HANDLE"; break;
   case    2097152: c = "MQGMO_MARK_BROWSE_CO_OP"; break;
   case    4194304: c = "MQGMO_UNMARK_BROWSE_CO_OP"; break;
   case    8388608: c = "MQGMO_UNMARK_BROWSE_HANDLE"; break;
   case   16777216: c = "MQGMO_UNMARKED_BROWSE_MSG"; break;
   case   17825808: c = "MQGMO_BROWSE_HANDLE"; break;
   case   18874384: c = "MQGMO_BROWSE_CO_OP"; break;
   case   33554432: c = "MQGMO_PROPERTIES_FORCE_MQRFH2"; break;
   case   67108864: c = "MQGMO_NO_PROPERTIES"; break;
   case  134217728: c = "MQGMO_PROPERTIES_IN_HANDLE"; break;
   case  268435456: c = "MQGMO_PROPERTIES_COMPATIBILITY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQGUR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQGUR_DISABLED"; break;
   case          1: c = "MQGUR_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       4001: c = "MQHA_BAG_HANDLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQHB_NONE"; break;
   case         -1: c = "MQHB_UNUSABLE_HBAG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -3: c = "MQHC_UNASSOCIATED_HCONN"; break;
   case         -1: c = "MQHC_UNUSABLE_HCONN"; break;
   case          0: c = "MQHC_DEF_HCONN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQHM_UNUSABLE_HMSG"; break;
   case          0: c = "MQHM_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQHO_UNUSABLE_HOBJ"; break;
   case          0: c = "MQHO_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQHSTATE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQHSTATE_INACTIVE"; break;
   case          1: c = "MQHSTATE_ACTIVE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIACF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       1001: c = "MQIACF_Q_MGR_ATTRS"; break;
   case       1002: c = "MQIACF_Q_ATTRS"; break;
   case       1003: c = "MQIACF_PROCESS_ATTRS"; break;
   case       1004: c = "MQIACF_NAMELIST_ATTRS"; break;
   case       1005: c = "MQIACF_FORCE"; break;
   case       1006: c = "MQIACF_REPLACE"; break;
   case       1007: c = "MQIACF_PURGE"; break;
   case       1008: c = "MQIACF_QUIESCE"; break;
   case       1009: c = "MQIACF_ALL"; break;
   case       1010: c = "MQIACF_EVENT_APPL_TYPE"; break;
   case       1011: c = "MQIACF_EVENT_ORIGIN"; break;
   case       1012: c = "MQIACF_PARAMETER_ID"; break;
   case       1013: c = "MQIACF_ERROR_ID"; break;
   case       1014: c = "MQIACF_SELECTOR"; break;
   case       1015: c = "MQIACF_CHANNEL_ATTRS"; break;
   case       1016: c = "MQIACF_OBJECT_TYPE"; break;
   case       1017: c = "MQIACF_ESCAPE_TYPE"; break;
   case       1018: c = "MQIACF_ERROR_OFFSET"; break;
   case       1019: c = "MQIACF_AUTH_INFO_ATTRS"; break;
   case       1020: c = "MQIACF_REASON_QUALIFIER"; break;
   case       1021: c = "MQIACF_COMMAND"; break;
   case       1022: c = "MQIACF_OPEN_OPTIONS"; break;
   case       1023: c = "MQIACF_OPEN_TYPE"; break;
   case       1024: c = "MQIACF_PROCESS_ID"; break;
   case       1025: c = "MQIACF_THREAD_ID"; break;
   case       1026: c = "MQIACF_Q_STATUS_ATTRS"; break;
   case       1027: c = "MQIACF_UNCOMMITTED_MSGS"; break;
   case       1028: c = "MQIACF_HANDLE_STATE"; break;
   case       1070: c = "MQIACF_AUX_ERROR_DATA_INT_1"; break;
   case       1071: c = "MQIACF_AUX_ERROR_DATA_INT_2"; break;
   case       1072: c = "MQIACF_CONV_REASON_CODE"; break;
   case       1073: c = "MQIACF_BRIDGE_TYPE"; break;
   case       1074: c = "MQIACF_INQUIRY"; break;
   case       1075: c = "MQIACF_WAIT_INTERVAL"; break;
   case       1076: c = "MQIACF_OPTIONS"; break;
   case       1077: c = "MQIACF_BROKER_OPTIONS"; break;
   case       1078: c = "MQIACF_REFRESH_TYPE"; break;
   case       1079: c = "MQIACF_SEQUENCE_NUMBER"; break;
   case       1080: c = "MQIACF_INTEGER_DATA"; break;
   case       1081: c = "MQIACF_REGISTRATION_OPTIONS"; break;
   case       1082: c = "MQIACF_PUBLICATION_OPTIONS"; break;
   case       1083: c = "MQIACF_CLUSTER_INFO"; break;
   case       1084: c = "MQIACF_Q_MGR_DEFINITION_TYPE"; break;
   case       1085: c = "MQIACF_Q_MGR_TYPE"; break;
   case       1086: c = "MQIACF_ACTION"; break;
   case       1087: c = "MQIACF_SUSPEND"; break;
   case       1088: c = "MQIACF_BROKER_COUNT"; break;
   case       1089: c = "MQIACF_APPL_COUNT"; break;
   case       1090: c = "MQIACF_ANONYMOUS_COUNT"; break;
   case       1091: c = "MQIACF_REG_REG_OPTIONS"; break;
   case       1092: c = "MQIACF_DELETE_OPTIONS"; break;
   case       1093: c = "MQIACF_CLUSTER_Q_MGR_ATTRS"; break;
   case       1094: c = "MQIACF_REFRESH_INTERVAL"; break;
   case       1095: c = "MQIACF_REFRESH_REPOSITORY"; break;
   case       1096: c = "MQIACF_REMOVE_QUEUES"; break;
   case       1098: c = "MQIACF_OPEN_INPUT_TYPE"; break;
   case       1099: c = "MQIACF_OPEN_OUTPUT"; break;
   case       1100: c = "MQIACF_OPEN_SET"; break;
   case       1101: c = "MQIACF_OPEN_INQUIRE"; break;
   case       1102: c = "MQIACF_OPEN_BROWSE"; break;
   case       1103: c = "MQIACF_Q_STATUS_TYPE"; break;
   case       1104: c = "MQIACF_Q_HANDLE"; break;
   case       1105: c = "MQIACF_Q_STATUS"; break;
   case       1106: c = "MQIACF_SECURITY_TYPE"; break;
   case       1107: c = "MQIACF_CONNECTION_ATTRS"; break;
   case       1108: c = "MQIACF_CONNECT_OPTIONS"; break;
   case       1110: c = "MQIACF_CONN_INFO_TYPE"; break;
   case       1111: c = "MQIACF_CONN_INFO_CONN"; break;
   case       1112: c = "MQIACF_CONN_INFO_HANDLE"; break;
   case       1113: c = "MQIACF_CONN_INFO_ALL"; break;
   case       1114: c = "MQIACF_AUTH_PROFILE_ATTRS"; break;
   case       1115: c = "MQIACF_AUTHORIZATION_LIST"; break;
   case       1116: c = "MQIACF_AUTH_ADD_AUTHS"; break;
   case       1117: c = "MQIACF_AUTH_REMOVE_AUTHS"; break;
   case       1118: c = "MQIACF_ENTITY_TYPE"; break;
   case       1120: c = "MQIACF_COMMAND_INFO"; break;
   case       1121: c = "MQIACF_CMDSCOPE_Q_MGR_COUNT"; break;
   case       1122: c = "MQIACF_Q_MGR_SYSTEM"; break;
   case       1123: c = "MQIACF_Q_MGR_EVENT"; break;
   case       1124: c = "MQIACF_Q_MGR_DQM"; break;
   case       1125: c = "MQIACF_Q_MGR_CLUSTER"; break;
   case       1126: c = "MQIACF_QSG_DISPS"; break;
   case       1128: c = "MQIACF_UOW_STATE"; break;
   case       1129: c = "MQIACF_SECURITY_ITEM"; break;
   case       1130: c = "MQIACF_CF_STRUC_STATUS"; break;
   case       1132: c = "MQIACF_UOW_TYPE"; break;
   case       1133: c = "MQIACF_CF_STRUC_ATTRS"; break;
   case       1134: c = "MQIACF_EXCLUDE_INTERVAL"; break;
   case       1135: c = "MQIACF_CF_STATUS_TYPE"; break;
   case       1136: c = "MQIACF_CF_STATUS_SUMMARY"; break;
   case       1137: c = "MQIACF_CF_STATUS_CONNECT"; break;
   case       1138: c = "MQIACF_CF_STATUS_BACKUP"; break;
   case       1139: c = "MQIACF_CF_STRUC_TYPE"; break;
   case       1140: c = "MQIACF_CF_STRUC_SIZE_MAX"; break;
   case       1141: c = "MQIACF_CF_STRUC_SIZE_USED"; break;
   case       1142: c = "MQIACF_CF_STRUC_ENTRIES_MAX"; break;
   case       1143: c = "MQIACF_CF_STRUC_ENTRIES_USED"; break;
   case       1144: c = "MQIACF_CF_STRUC_BACKUP_SIZE"; break;
   case       1145: c = "MQIACF_MOVE_TYPE"; break;
   case       1146: c = "MQIACF_MOVE_TYPE_MOVE"; break;
   case       1147: c = "MQIACF_MOVE_TYPE_ADD"; break;
   case       1148: c = "MQIACF_Q_MGR_NUMBER"; break;
   case       1149: c = "MQIACF_Q_MGR_STATUS"; break;
   case       1150: c = "MQIACF_DB2_CONN_STATUS"; break;
   case       1151: c = "MQIACF_SECURITY_ATTRS"; break;
   case       1152: c = "MQIACF_SECURITY_TIMEOUT"; break;
   case       1153: c = "MQIACF_SECURITY_INTERVAL"; break;
   case       1154: c = "MQIACF_SECURITY_SWITCH"; break;
   case       1155: c = "MQIACF_SECURITY_SETTING"; break;
   case       1156: c = "MQIACF_STORAGE_CLASS_ATTRS"; break;
   case       1157: c = "MQIACF_USAGE_TYPE"; break;
   case       1158: c = "MQIACF_BUFFER_POOL_ID"; break;
   case       1159: c = "MQIACF_USAGE_TOTAL_PAGES"; break;
   case       1160: c = "MQIACF_USAGE_UNUSED_PAGES"; break;
   case       1161: c = "MQIACF_USAGE_PERSIST_PAGES"; break;
   case       1162: c = "MQIACF_USAGE_NONPERSIST_PAGES"; break;
   case       1163: c = "MQIACF_USAGE_RESTART_EXTENTS"; break;
   case       1164: c = "MQIACF_USAGE_EXPAND_COUNT"; break;
   case       1165: c = "MQIACF_PAGESET_STATUS"; break;
   case       1166: c = "MQIACF_USAGE_TOTAL_BUFFERS"; break;
   case       1167: c = "MQIACF_USAGE_DATA_SET_TYPE"; break;
   case       1168: c = "MQIACF_USAGE_PAGESET"; break;
   case       1169: c = "MQIACF_USAGE_DATA_SET"; break;
   case       1170: c = "MQIACF_USAGE_BUFFER_POOL"; break;
   case       1171: c = "MQIACF_MOVE_COUNT"; break;
   case       1172: c = "MQIACF_EXPIRY_Q_COUNT"; break;
   case       1173: c = "MQIACF_CONFIGURATION_OBJECTS"; break;
   case       1174: c = "MQIACF_CONFIGURATION_EVENTS"; break;
   case       1175: c = "MQIACF_SYSP_TYPE"; break;
   case       1176: c = "MQIACF_SYSP_DEALLOC_INTERVAL"; break;
   case       1177: c = "MQIACF_SYSP_MAX_ARCHIVE"; break;
   case       1178: c = "MQIACF_SYSP_MAX_READ_TAPES"; break;
   case       1179: c = "MQIACF_SYSP_IN_BUFFER_SIZE"; break;
   case       1180: c = "MQIACF_SYSP_OUT_BUFFER_SIZE"; break;
   case       1181: c = "MQIACF_SYSP_OUT_BUFFER_COUNT"; break;
   case       1182: c = "MQIACF_SYSP_ARCHIVE"; break;
   case       1183: c = "MQIACF_SYSP_DUAL_ACTIVE"; break;
   case       1184: c = "MQIACF_SYSP_DUAL_ARCHIVE"; break;
   case       1185: c = "MQIACF_SYSP_DUAL_BSDS"; break;
   case       1186: c = "MQIACF_SYSP_MAX_CONNS"; break;
   case       1187: c = "MQIACF_SYSP_MAX_CONNS_FORE"; break;
   case       1188: c = "MQIACF_SYSP_MAX_CONNS_BACK"; break;
   case       1189: c = "MQIACF_SYSP_EXIT_INTERVAL"; break;
   case       1190: c = "MQIACF_SYSP_EXIT_TASKS"; break;
   case       1191: c = "MQIACF_SYSP_CHKPOINT_COUNT"; break;
   case       1192: c = "MQIACF_SYSP_OTMA_INTERVAL"; break;
   case       1193: c = "MQIACF_SYSP_Q_INDEX_DEFER"; break;
   case       1194: c = "MQIACF_SYSP_DB2_TASKS"; break;
   case       1195: c = "MQIACF_SYSP_RESLEVEL_AUDIT"; break;
   case       1196: c = "MQIACF_SYSP_ROUTING_CODE"; break;
   case       1197: c = "MQIACF_SYSP_SMF_ACCOUNTING"; break;
   case       1198: c = "MQIACF_SYSP_SMF_STATS"; break;
   case       1199: c = "MQIACF_SYSP_SMF_INTERVAL"; break;
   case       1200: c = "MQIACF_SYSP_TRACE_CLASS"; break;
   case       1201: c = "MQIACF_SYSP_TRACE_SIZE"; break;
   case       1202: c = "MQIACF_SYSP_WLM_INTERVAL"; break;
   case       1203: c = "MQIACF_SYSP_ALLOC_UNIT"; break;
   case       1204: c = "MQIACF_SYSP_ARCHIVE_RETAIN"; break;
   case       1205: c = "MQIACF_SYSP_ARCHIVE_WTOR"; break;
   case       1206: c = "MQIACF_SYSP_BLOCK_SIZE"; break;
   case       1207: c = "MQIACF_SYSP_CATALOG"; break;
   case       1208: c = "MQIACF_SYSP_COMPACT"; break;
   case       1209: c = "MQIACF_SYSP_ALLOC_PRIMARY"; break;
   case       1210: c = "MQIACF_SYSP_ALLOC_SECONDARY"; break;
   case       1211: c = "MQIACF_SYSP_PROTECT"; break;
   case       1212: c = "MQIACF_SYSP_QUIESCE_INTERVAL"; break;
   case       1213: c = "MQIACF_SYSP_TIMESTAMP"; break;
   case       1214: c = "MQIACF_SYSP_UNIT_ADDRESS"; break;
   case       1215: c = "MQIACF_SYSP_UNIT_STATUS"; break;
   case       1216: c = "MQIACF_SYSP_LOG_COPY"; break;
   case       1217: c = "MQIACF_SYSP_LOG_USED"; break;
   case       1218: c = "MQIACF_SYSP_LOG_SUSPEND"; break;
   case       1219: c = "MQIACF_SYSP_OFFLOAD_STATUS"; break;
   case       1220: c = "MQIACF_SYSP_TOTAL_LOGS"; break;
   case       1221: c = "MQIACF_SYSP_FULL_LOGS"; break;
   case       1222: c = "MQIACF_LISTENER_ATTRS"; break;
   case       1223: c = "MQIACF_LISTENER_STATUS_ATTRS"; break;
   case       1224: c = "MQIACF_SERVICE_ATTRS"; break;
   case       1225: c = "MQIACF_SERVICE_STATUS_ATTRS"; break;
   case       1226: c = "MQIACF_Q_TIME_INDICATOR"; break;
   case       1227: c = "MQIACF_OLDEST_MSG_AGE"; break;
   case       1228: c = "MQIACF_AUTH_OPTIONS"; break;
   case       1229: c = "MQIACF_Q_MGR_STATUS_ATTRS"; break;
   case       1230: c = "MQIACF_CONNECTION_COUNT"; break;
   case       1231: c = "MQIACF_Q_MGR_FACILITY"; break;
   case       1232: c = "MQIACF_CHINIT_STATUS"; break;
   case       1233: c = "MQIACF_CMD_SERVER_STATUS"; break;
   case       1234: c = "MQIACF_ROUTE_DETAIL"; break;
   case       1235: c = "MQIACF_RECORDED_ACTIVITIES"; break;
   case       1236: c = "MQIACF_MAX_ACTIVITIES"; break;
   case       1237: c = "MQIACF_DISCONTINUITY_COUNT"; break;
   case       1238: c = "MQIACF_ROUTE_ACCUMULATION"; break;
   case       1239: c = "MQIACF_ROUTE_DELIVERY"; break;
   case       1240: c = "MQIACF_OPERATION_TYPE"; break;
   case       1241: c = "MQIACF_BACKOUT_COUNT"; break;
   case       1242: c = "MQIACF_COMP_CODE"; break;
   case       1243: c = "MQIACF_ENCODING"; break;
   case       1244: c = "MQIACF_EXPIRY"; break;
   case       1245: c = "MQIACF_FEEDBACK"; break;
   case       1247: c = "MQIACF_MSG_FLAGS"; break;
   case       1248: c = "MQIACF_MSG_LENGTH"; break;
   case       1249: c = "MQIACF_MSG_TYPE"; break;
   case       1250: c = "MQIACF_OFFSET"; break;
   case       1251: c = "MQIACF_ORIGINAL_LENGTH"; break;
   case       1252: c = "MQIACF_PERSISTENCE"; break;
   case       1253: c = "MQIACF_PRIORITY"; break;
   case       1254: c = "MQIACF_REASON_CODE"; break;
   case       1255: c = "MQIACF_REPORT"; break;
   case       1256: c = "MQIACF_VERSION"; break;
   case       1257: c = "MQIACF_UNRECORDED_ACTIVITIES"; break;
   case       1258: c = "MQIACF_MONITORING"; break;
   case       1259: c = "MQIACF_ROUTE_FORWARDING"; break;
   case       1260: c = "MQIACF_SERVICE_STATUS"; break;
   case       1261: c = "MQIACF_Q_TYPES"; break;
   case       1262: c = "MQIACF_USER_ID_SUPPORT"; break;
   case       1263: c = "MQIACF_INTERFACE_VERSION"; break;
   case       1264: c = "MQIACF_AUTH_SERVICE_ATTRS"; break;
   case       1265: c = "MQIACF_USAGE_EXPAND_TYPE"; break;
   case       1266: c = "MQIACF_SYSP_CLUSTER_CACHE"; break;
   case       1267: c = "MQIACF_SYSP_DB2_BLOB_TASKS"; break;
   case       1268: c = "MQIACF_SYSP_WLM_INT_UNITS"; break;
   case       1269: c = "MQIACF_TOPIC_ATTRS"; break;
   case       1271: c = "MQIACF_PUBSUB_PROPERTIES"; break;
   case       1273: c = "MQIACF_DESTINATION_CLASS"; break;
   case       1274: c = "MQIACF_DURABLE_SUBSCRIPTION"; break;
   case       1275: c = "MQIACF_SUBSCRIPTION_SCOPE"; break;
   case       1277: c = "MQIACF_VARIABLE_USER_ID"; break;
   case       1280: c = "MQIACF_REQUEST_ONLY"; break;
   case       1283: c = "MQIACF_PUB_PRIORITY"; break;
   case       1287: c = "MQIACF_SUB_ATTRS"; break;
   case       1288: c = "MQIACF_WILDCARD_SCHEMA"; break;
   case       1289: c = "MQIACF_SUB_TYPE"; break;
   case       1290: c = "MQIACF_MESSAGE_COUNT"; break;
   case       1291: c = "MQIACF_Q_MGR_PUBSUB"; break;
   case       1292: c = "MQIACF_Q_MGR_VERSION"; break;
   case       1294: c = "MQIACF_SUB_STATUS_ATTRS"; break;
   case       1295: c = "MQIACF_TOPIC_STATUS"; break;
   case       1296: c = "MQIACF_TOPIC_SUB"; break;
   case       1297: c = "MQIACF_TOPIC_PUB"; break;
   case       1300: c = "MQIACF_RETAINED_PUBLICATION"; break;
   case       1301: c = "MQIACF_TOPIC_STATUS_ATTRS"; break;
   case       1302: c = "MQIACF_TOPIC_STATUS_TYPE"; break;
   case       1303: c = "MQIACF_SUB_OPTIONS"; break;
   case       1304: c = "MQIACF_PUBLISH_COUNT"; break;
   case       1305: c = "MQIACF_CLEAR_TYPE"; break;
   case       1306: c = "MQIACF_CLEAR_SCOPE"; break;
   case       1307: c = "MQIACF_SUB_LEVEL"; break;
   case       1308: c = "MQIACF_ASYNC_STATE"; break;
   case       1309: c = "MQIACF_SUB_SUMMARY"; break;
   case       1310: c = "MQIACF_OBSOLETE_MSGS"; break;
   case       1311: c = "MQIACF_PUBSUB_STATUS"; break;
   case       1314: c = "MQIACF_PS_STATUS_TYPE"; break;
   case       1318: c = "MQIACF_PUBSUB_STATUS_ATTRS"; break;
   case       1321: c = "MQIACF_SELECTOR_TYPE"; break;
   case       1322: c = "MQIACF_LOG_COMPRESSION"; break;
   case       1323: c = "MQIACF_GROUPUR_CHECK_ID"; break;
   case       1324: c = "MQIACF_MULC_CAPTURE"; break;
   case       1325: c = "MQIACF_PERMIT_STANDBY"; break;
   case       1326: c = "MQIACF_OPERATION_MODE"; break;
   case       1327: c = "MQIACF_COMM_INFO_ATTRS"; break;
   case       1328: c = "MQIACF_CF_SMDS_BLOCK_SIZE"; break;
   case       1329: c = "MQIACF_CF_SMDS_EXPAND"; break;
   case       1330: c = "MQIACF_USAGE_FREE_BUFF"; break;
   case       1331: c = "MQIACF_USAGE_FREE_BUFF_PERC"; break;
   case       1332: c = "MQIACF_CF_STRUC_ACCESS"; break;
   case       1333: c = "MQIACF_CF_STATUS_SMDS"; break;
   case       1334: c = "MQIACF_SMDS_ATTRS"; break;
   case       1335: c = "MQIACF_USAGE_SMDS"; break;
   case       1336: c = "MQIACF_USAGE_BLOCK_SIZE"; break;
   case       1337: c = "MQIACF_USAGE_DATA_BLOCKS"; break;
   case       1338: c = "MQIACF_USAGE_EMPTY_BUFFERS"; break;
   case       1339: c = "MQIACF_USAGE_INUSE_BUFFERS"; break;
   case       1340: c = "MQIACF_USAGE_LOWEST_FREE"; break;
   case       1341: c = "MQIACF_USAGE_OFFLOAD_MSGS"; break;
   case       1342: c = "MQIACF_USAGE_READS_SAVED"; break;
   case       1343: c = "MQIACF_USAGE_SAVED_BUFFERS"; break;
   case       1344: c = "MQIACF_USAGE_TOTAL_BLOCKS"; break;
   case       1345: c = "MQIACF_USAGE_USED_BLOCKS"; break;
   case       1346: c = "MQIACF_USAGE_USED_RATE"; break;
   case       1347: c = "MQIACF_USAGE_WAIT_RATE"; break;
   case       1348: c = "MQIACF_SMDS_OPENMODE"; break;
   case       1349: c = "MQIACF_SMDS_STATUS"; break;
   case       1350: c = "MQIACF_SMDS_AVAIL"; break;
   case       1351: c = "MQIACF_MCAST_REL_INDICATOR"; break;
   case       1352: c = "MQIACF_CHLAUTH_TYPE"; break;
   case       1354: c = "MQIACF_MQXR_DIAGNOSTICS_TYPE"; break;
   case       1355: c = "MQIACF_CHLAUTH_ATTRS"; break;
   case       1356: c = "MQIACF_OPERATION_ID"; break;
   case       1357: c = "MQIACF_API_CALLER_TYPE"; break;
   case       1358: c = "MQIACF_API_ENVIRONMENT"; break;
   case       1359: c = "MQIACF_TRACE_DETAIL"; break;
   case       1360: c = "MQIACF_HOBJ"; break;
   case       1361: c = "MQIACF_CALL_TYPE"; break;
   case       1362: c = "MQIACF_MQCB_OPERATION"; break;
   case       1363: c = "MQIACF_MQCB_TYPE"; break;
   case       1364: c = "MQIACF_MQCB_OPTIONS"; break;
   case       1365: c = "MQIACF_CLOSE_OPTIONS"; break;
   case       1366: c = "MQIACF_CTL_OPERATION"; break;
   case       1367: c = "MQIACF_GET_OPTIONS"; break;
   case       1368: c = "MQIACF_RECS_PRESENT"; break;
   case       1369: c = "MQIACF_KNOWN_DEST_COUNT"; break;
   case       1370: c = "MQIACF_UNKNOWN_DEST_COUNT"; break;
   case       1371: c = "MQIACF_INVALID_DEST_COUNT"; break;
   case       1372: c = "MQIACF_RESOLVED_TYPE"; break;
   case       1373: c = "MQIACF_PUT_OPTIONS"; break;
   case       1374: c = "MQIACF_BUFFER_LENGTH"; break;
   case       1375: c = "MQIACF_TRACE_DATA_LENGTH"; break;
   case       1376: c = "MQIACF_SMDS_EXPANDST"; break;
   case       1378: c = "MQIACF_ITEM_COUNT"; break;
   case       1379: c = "MQIACF_EXPIRY_TIME"; break;
   case       1380: c = "MQIACF_CONNECT_TIME"; break;
   case       1381: c = "MQIACF_DISCONNECT_TIME"; break;
   case       1382: c = "MQIACF_HSUB"; break;
   case       1383: c = "MQIACF_SUBRQ_OPTIONS"; break;
   case       1384: c = "MQIACF_XA_RMID"; break;
   case       1385: c = "MQIACF_XA_FLAGS"; break;
   case       1386: c = "MQIACF_XA_RETCODE"; break;
   case       1387: c = "MQIACF_XA_HANDLE"; break;
   case       1388: c = "MQIACF_XA_RETVAL"; break;
   case       1389: c = "MQIACF_STATUS_TYPE"; break;
   case       1390: c = "MQIACF_XA_COUNT"; break;
   case       1391: c = "MQIACF_SELECTOR_COUNT"; break;
   case       1392: c = "MQIACF_SELECTORS"; break;
   case       1393: c = "MQIACF_INTATTR_COUNT"; break;
   case       1394: c = "MQIACF_INT_ATTRS"; break;
   case       1395: c = "MQIACF_SUBRQ_ACTION"; break;
   case       1396: c = "MQIACF_NUM_PUBS"; break;
   case       1397: c = "MQIACF_POINTER_SIZE"; break;
   case       1398: c = "MQIACF_REMOVE_AUTHREC"; break;
   case       1399: c = "MQIACF_XR_ATTRS"; break;
   case       1400: c = "MQIACF_APPL_FUNCTION_TYPE"; break;
   case       1401: c = "MQIACF_AMQP_ATTRS"; break;
   case       1402: c = "MQIACF_EXPORT_TYPE"; break;
   case       1403: c = "MQIACF_EXPORT_ATTRS"; break;
   case       1404: c = "MQIACF_SYSTEM_OBJECTS"; break;
   case       1405: c = "MQIACF_CONNECTION_SWAP"; break;
   case       1406: c = "MQIACF_AMQP_DIAGNOSTICS_TYPE"; break;
   case       1408: c = "MQIACF_BUFFER_POOL_LOCATION"; break;
   case       1409: c = "MQIACF_LDAP_CONNECTION_STATUS"; break;
   case       1410: c = "MQIACF_SYSP_MAX_ACE_POOL"; break;
   case       1411: c = "MQIACF_PAGECLAS"; break;
   case       1412: c = "MQIACF_AUTH_REC_TYPE"; break;
   case       1413: c = "MQIACF_SYSP_MAX_CONC_OFFLOADS"; break;
   case       1414: c = "MQIACF_SYSP_ZHYPERWRITE"; break;
   case       1415: c = "MQIACF_Q_MGR_STATUS_LOG"; break;
   case       1416: c = "MQIACF_ARCHIVE_LOG_SIZE"; break;
   case       1417: c = "MQIACF_MEDIA_LOG_SIZE"; break;
   case       1418: c = "MQIACF_RESTART_LOG_SIZE"; break;
   case       1419: c = "MQIACF_REUSABLE_LOG_SIZE"; break;
   case       1420: c = "MQIACF_LOG_IN_USE"; break;
   case       1421: c = "MQIACF_LOG_UTILIZATION"; break;
   case       1422: c = "MQIACF_LOG_REDUCTION"; break;
   case       1423: c = "MQIACF_IGNORE_STATE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIACH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       1501: c = "MQIACH_XMIT_PROTOCOL_TYPE"; break;
   case       1502: c = "MQIACH_BATCH_SIZE"; break;
   case       1503: c = "MQIACH_DISC_INTERVAL"; break;
   case       1504: c = "MQIACH_SHORT_TIMER"; break;
   case       1505: c = "MQIACH_SHORT_RETRY"; break;
   case       1506: c = "MQIACH_LONG_TIMER"; break;
   case       1507: c = "MQIACH_LONG_RETRY"; break;
   case       1508: c = "MQIACH_PUT_AUTHORITY"; break;
   case       1509: c = "MQIACH_SEQUENCE_NUMBER_WRAP"; break;
   case       1510: c = "MQIACH_MAX_MSG_LENGTH"; break;
   case       1511: c = "MQIACH_CHANNEL_TYPE"; break;
   case       1512: c = "MQIACH_DATA_COUNT"; break;
   case       1513: c = "MQIACH_NAME_COUNT"; break;
   case       1514: c = "MQIACH_MSG_SEQUENCE_NUMBER"; break;
   case       1515: c = "MQIACH_DATA_CONVERSION"; break;
   case       1516: c = "MQIACH_IN_DOUBT"; break;
   case       1517: c = "MQIACH_MCA_TYPE"; break;
   case       1518: c = "MQIACH_SESSION_COUNT"; break;
   case       1519: c = "MQIACH_ADAPTER"; break;
   case       1520: c = "MQIACH_COMMAND_COUNT"; break;
   case       1521: c = "MQIACH_SOCKET"; break;
   case       1522: c = "MQIACH_PORT"; break;
   case       1523: c = "MQIACH_CHANNEL_INSTANCE_TYPE"; break;
   case       1524: c = "MQIACH_CHANNEL_INSTANCE_ATTRS"; break;
   case       1525: c = "MQIACH_CHANNEL_ERROR_DATA"; break;
   case       1526: c = "MQIACH_CHANNEL_TABLE"; break;
   case       1527: c = "MQIACH_CHANNEL_STATUS"; break;
   case       1528: c = "MQIACH_INDOUBT_STATUS"; break;
   case       1529: c = "MQIACH_LAST_SEQ_NUMBER"; break;
   case       1531: c = "MQIACH_CURRENT_MSGS"; break;
   case       1532: c = "MQIACH_CURRENT_SEQ_NUMBER"; break;
   case       1533: c = "MQIACH_SSL_RETURN_CODE"; break;
   case       1534: c = "MQIACH_MSGS"; break;
   case       1535: c = "MQIACH_BYTES_SENT"; break;
   case       1536: c = "MQIACH_BYTES_RCVD"; break;
   case       1537: c = "MQIACH_BATCHES"; break;
   case       1538: c = "MQIACH_BUFFERS_SENT"; break;
   case       1539: c = "MQIACH_BUFFERS_RCVD"; break;
   case       1540: c = "MQIACH_LONG_RETRIES_LEFT"; break;
   case       1541: c = "MQIACH_SHORT_RETRIES_LEFT"; break;
   case       1542: c = "MQIACH_MCA_STATUS"; break;
   case       1543: c = "MQIACH_STOP_REQUESTED"; break;
   case       1544: c = "MQIACH_MR_COUNT"; break;
   case       1545: c = "MQIACH_MR_INTERVAL"; break;
   case       1562: c = "MQIACH_NPM_SPEED"; break;
   case       1563: c = "MQIACH_HB_INTERVAL"; break;
   case       1564: c = "MQIACH_BATCH_INTERVAL"; break;
   case       1565: c = "MQIACH_NETWORK_PRIORITY"; break;
   case       1566: c = "MQIACH_KEEP_ALIVE_INTERVAL"; break;
   case       1567: c = "MQIACH_BATCH_HB"; break;
   case       1568: c = "MQIACH_SSL_CLIENT_AUTH"; break;
   case       1570: c = "MQIACH_ALLOC_RETRY"; break;
   case       1571: c = "MQIACH_ALLOC_FAST_TIMER"; break;
   case       1572: c = "MQIACH_ALLOC_SLOW_TIMER"; break;
   case       1573: c = "MQIACH_DISC_RETRY"; break;
   case       1574: c = "MQIACH_PORT_NUMBER"; break;
   case       1575: c = "MQIACH_HDR_COMPRESSION"; break;
   case       1576: c = "MQIACH_MSG_COMPRESSION"; break;
   case       1577: c = "MQIACH_CLWL_CHANNEL_RANK"; break;
   case       1578: c = "MQIACH_CLWL_CHANNEL_PRIORITY"; break;
   case       1579: c = "MQIACH_CLWL_CHANNEL_WEIGHT"; break;
   case       1580: c = "MQIACH_CHANNEL_DISP"; break;
   case       1581: c = "MQIACH_INBOUND_DISP"; break;
   case       1582: c = "MQIACH_CHANNEL_TYPES"; break;
   case       1583: c = "MQIACH_ADAPS_STARTED"; break;
   case       1584: c = "MQIACH_ADAPS_MAX"; break;
   case       1585: c = "MQIACH_DISPS_STARTED"; break;
   case       1586: c = "MQIACH_DISPS_MAX"; break;
   case       1587: c = "MQIACH_SSLTASKS_STARTED"; break;
   case       1588: c = "MQIACH_SSLTASKS_MAX"; break;
   case       1589: c = "MQIACH_CURRENT_CHL"; break;
   case       1590: c = "MQIACH_CURRENT_CHL_MAX"; break;
   case       1591: c = "MQIACH_CURRENT_CHL_TCP"; break;
   case       1592: c = "MQIACH_CURRENT_CHL_LU62"; break;
   case       1593: c = "MQIACH_ACTIVE_CHL"; break;
   case       1594: c = "MQIACH_ACTIVE_CHL_MAX"; break;
   case       1595: c = "MQIACH_ACTIVE_CHL_PAUSED"; break;
   case       1596: c = "MQIACH_ACTIVE_CHL_STARTED"; break;
   case       1597: c = "MQIACH_ACTIVE_CHL_STOPPED"; break;
   case       1598: c = "MQIACH_ACTIVE_CHL_RETRY"; break;
   case       1599: c = "MQIACH_LISTENER_STATUS"; break;
   case       1600: c = "MQIACH_SHARED_CHL_RESTART"; break;
   case       1601: c = "MQIACH_LISTENER_CONTROL"; break;
   case       1602: c = "MQIACH_BACKLOG"; break;
   case       1604: c = "MQIACH_XMITQ_TIME_INDICATOR"; break;
   case       1605: c = "MQIACH_NETWORK_TIME_INDICATOR"; break;
   case       1606: c = "MQIACH_EXIT_TIME_INDICATOR"; break;
   case       1607: c = "MQIACH_BATCH_SIZE_INDICATOR"; break;
   case       1608: c = "MQIACH_XMITQ_MSGS_AVAILABLE"; break;
   case       1609: c = "MQIACH_CHANNEL_SUBSTATE"; break;
   case       1610: c = "MQIACH_SSL_KEY_RESETS"; break;
   case       1611: c = "MQIACH_COMPRESSION_RATE"; break;
   case       1612: c = "MQIACH_COMPRESSION_TIME"; break;
   case       1613: c = "MQIACH_MAX_XMIT_SIZE"; break;
   case       1614: c = "MQIACH_DEF_CHANNEL_DISP"; break;
   case       1615: c = "MQIACH_SHARING_CONVERSATIONS"; break;
   case       1616: c = "MQIACH_MAX_SHARING_CONVS"; break;
   case       1617: c = "MQIACH_CURRENT_SHARING_CONVS"; break;
   case       1618: c = "MQIACH_MAX_INSTANCES"; break;
   case       1619: c = "MQIACH_MAX_INSTS_PER_CLIENT"; break;
   case       1620: c = "MQIACH_CLIENT_CHANNEL_WEIGHT"; break;
   case       1621: c = "MQIACH_CONNECTION_AFFINITY"; break;
   case       1623: c = "MQIACH_RESET_REQUESTED"; break;
   case       1624: c = "MQIACH_BATCH_DATA_LIMIT"; break;
   case       1625: c = "MQIACH_MSG_HISTORY"; break;
   case       1626: c = "MQIACH_MULTICAST_PROPERTIES"; break;
   case       1627: c = "MQIACH_NEW_SUBSCRIBER_HISTORY"; break;
   case       1628: c = "MQIACH_MC_HB_INTERVAL"; break;
   case       1629: c = "MQIACH_USE_CLIENT_ID"; break;
   case       1630: c = "MQIACH_MQTT_KEEP_ALIVE"; break;
   case       1631: c = "MQIACH_IN_DOUBT_IN"; break;
   case       1632: c = "MQIACH_IN_DOUBT_OUT"; break;
   case       1633: c = "MQIACH_MSGS_SENT"; break;
   case       1634: c = "MQIACH_MSGS_RCVD"; break;
   case       1635: c = "MQIACH_PENDING_OUT"; break;
   case       1636: c = "MQIACH_AVAILABLE_CIPHERSPECS"; break;
   case       1637: c = "MQIACH_MATCH"; break;
   case       1638: c = "MQIACH_USER_SOURCE"; break;
   case       1639: c = "MQIACH_WARNING"; break;
   case       1640: c = "MQIACH_DEF_RECONNECT"; break;
   case       1642: c = "MQIACH_CHANNEL_SUMMARY_ATTRS"; break;
   case       1643: c = "MQIACH_PROTOCOL"; break;
   case       1644: c = "MQIACH_AMQP_KEEP_ALIVE"; break;
   case       1645: c = "MQIACH_SECURITY_PROTOCOL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIAMO64_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case        703: c = "MQIAMO64_AVG_Q_TIME"; break;
   case        741: c = "MQIAMO64_Q_TIME_AVG"; break;
   case        742: c = "MQIAMO64_Q_TIME_MAX"; break;
   case        743: c = "MQIAMO64_Q_TIME_MIN"; break;
   case        745: c = "MQIAMO64_BROWSE_BYTES"; break;
   case        746: c = "MQIAMO64_BYTES"; break;
   case        747: c = "MQIAMO64_GET_BYTES"; break;
   case        748: c = "MQIAMO64_PUT_BYTES"; break;
   case        783: c = "MQIAMO64_TOPIC_PUT_BYTES"; break;
   case        785: c = "MQIAMO64_PUBLISH_MSG_BYTES"; break;
   case        838: c = "MQIAMO64_HIGHRES_TIME"; break;
   case        844: c = "MQIAMO64_QMGR_OP_DURATION"; break;
   case        845: c = "MQIAMO64_MONITOR_INTERVAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIAMO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIAMO_MONITOR_FLAGS_NONE"; break;
   case          1: c = "MQIAMO_MONITOR_FLAGS_OBJNAME"; break;
   case          2: c = "MQIAMO_MONITOR_DELTA"; break;
   case        100: c = "MQIAMO_MONITOR_HUNDREDTHS"; break;
   case        702: c = "MQIAMO_AVG_BATCH_SIZE"; break;
   case        703: c = "MQIAMO_AVG_Q_TIME"; break;
   case        704: c = "MQIAMO_BACKOUTS"; break;
   case        705: c = "MQIAMO_BROWSES"; break;
   case        706: c = "MQIAMO_BROWSE_MAX_BYTES"; break;
   case        707: c = "MQIAMO_BROWSE_MIN_BYTES"; break;
   case        708: c = "MQIAMO_BROWSES_FAILED"; break;
   case        709: c = "MQIAMO_CLOSES"; break;
   case        710: c = "MQIAMO_COMMITS"; break;
   case        711: c = "MQIAMO_COMMITS_FAILED"; break;
   case        712: c = "MQIAMO_CONNS"; break;
   case        713: c = "MQIAMO_CONNS_MAX"; break;
   case        714: c = "MQIAMO_DISCS"; break;
   case        715: c = "MQIAMO_DISCS_IMPLICIT"; break;
   case        716: c = "MQIAMO_DISC_TYPE"; break;
   case        717: c = "MQIAMO_EXIT_TIME_AVG"; break;
   case        718: c = "MQIAMO_EXIT_TIME_MAX"; break;
   case        719: c = "MQIAMO_EXIT_TIME_MIN"; break;
   case        720: c = "MQIAMO_FULL_BATCHES"; break;
   case        721: c = "MQIAMO_GENERATED_MSGS"; break;
   case        722: c = "MQIAMO_GETS"; break;
   case        723: c = "MQIAMO_GET_MAX_BYTES"; break;
   case        724: c = "MQIAMO_GET_MIN_BYTES"; break;
   case        725: c = "MQIAMO_GETS_FAILED"; break;
   case        726: c = "MQIAMO_INCOMPLETE_BATCHES"; break;
   case        727: c = "MQIAMO_INQS"; break;
   case        728: c = "MQIAMO_MSGS"; break;
   case        729: c = "MQIAMO_NET_TIME_AVG"; break;
   case        730: c = "MQIAMO_NET_TIME_MAX"; break;
   case        731: c = "MQIAMO_NET_TIME_MIN"; break;
   case        732: c = "MQIAMO_OBJECT_COUNT"; break;
   case        733: c = "MQIAMO_OPENS"; break;
   case        734: c = "MQIAMO_PUT1S"; break;
   case        735: c = "MQIAMO_PUTS"; break;
   case        736: c = "MQIAMO_PUT_MAX_BYTES"; break;
   case        737: c = "MQIAMO_PUT_MIN_BYTES"; break;
   case        738: c = "MQIAMO_PUT_RETRIES"; break;
   case        739: c = "MQIAMO_Q_MAX_DEPTH"; break;
   case        740: c = "MQIAMO_Q_MIN_DEPTH"; break;
   case        741: c = "MQIAMO_Q_TIME_AVG"; break;
   case        742: c = "MQIAMO_Q_TIME_MAX"; break;
   case        743: c = "MQIAMO_Q_TIME_MIN"; break;
   case        744: c = "MQIAMO_SETS"; break;
   case        749: c = "MQIAMO_CONNS_FAILED"; break;
   case        751: c = "MQIAMO_OPENS_FAILED"; break;
   case        752: c = "MQIAMO_INQS_FAILED"; break;
   case        753: c = "MQIAMO_SETS_FAILED"; break;
   case        754: c = "MQIAMO_PUTS_FAILED"; break;
   case        755: c = "MQIAMO_PUT1S_FAILED"; break;
   case        757: c = "MQIAMO_CLOSES_FAILED"; break;
   case        758: c = "MQIAMO_MSGS_EXPIRED"; break;
   case        759: c = "MQIAMO_MSGS_NOT_QUEUED"; break;
   case        760: c = "MQIAMO_MSGS_PURGED"; break;
   case        764: c = "MQIAMO_SUBS_DUR"; break;
   case        765: c = "MQIAMO_SUBS_NDUR"; break;
   case        766: c = "MQIAMO_SUBS_FAILED"; break;
   case        767: c = "MQIAMO_SUBRQS"; break;
   case        768: c = "MQIAMO_SUBRQS_FAILED"; break;
   case        769: c = "MQIAMO_CBS"; break;
   case        770: c = "MQIAMO_CBS_FAILED"; break;
   case        771: c = "MQIAMO_CTLS"; break;
   case        772: c = "MQIAMO_CTLS_FAILED"; break;
   case        773: c = "MQIAMO_STATS"; break;
   case        774: c = "MQIAMO_STATS_FAILED"; break;
   case        775: c = "MQIAMO_SUB_DUR_HIGHWATER"; break;
   case        776: c = "MQIAMO_SUB_DUR_LOWWATER"; break;
   case        777: c = "MQIAMO_SUB_NDUR_HIGHWATER"; break;
   case        778: c = "MQIAMO_SUB_NDUR_LOWWATER"; break;
   case        779: c = "MQIAMO_TOPIC_PUTS"; break;
   case        780: c = "MQIAMO_TOPIC_PUTS_FAILED"; break;
   case        781: c = "MQIAMO_TOPIC_PUT1S"; break;
   case        782: c = "MQIAMO_TOPIC_PUT1S_FAILED"; break;
   case        784: c = "MQIAMO_PUBLISH_MSG_COUNT"; break;
   case        786: c = "MQIAMO_UNSUBS_DUR"; break;
   case        787: c = "MQIAMO_UNSUBS_NDUR"; break;
   case        788: c = "MQIAMO_UNSUBS_FAILED"; break;
   case        789: c = "MQIAMO_INTERVAL"; break;
   case        790: c = "MQIAMO_MSGS_SENT"; break;
   case        791: c = "MQIAMO_BYTES_SENT"; break;
   case        792: c = "MQIAMO_REPAIR_BYTES"; break;
   case        793: c = "MQIAMO_FEEDBACK_MODE"; break;
   case        794: c = "MQIAMO_RELIABILITY_TYPE"; break;
   case        795: c = "MQIAMO_LATE_JOIN_MARK"; break;
   case        796: c = "MQIAMO_NACKS_RCVD"; break;
   case        797: c = "MQIAMO_REPAIR_PKTS"; break;
   case        798: c = "MQIAMO_HISTORY_PKTS"; break;
   case        799: c = "MQIAMO_PENDING_PKTS"; break;
   case        800: c = "MQIAMO_PKT_RATE"; break;
   case        801: c = "MQIAMO_MCAST_XMIT_RATE"; break;
   case        802: c = "MQIAMO_MCAST_BATCH_TIME"; break;
   case        803: c = "MQIAMO_MCAST_HEARTBEAT"; break;
   case        804: c = "MQIAMO_DEST_DATA_PORT"; break;
   case        805: c = "MQIAMO_DEST_REPAIR_PORT"; break;
   case        806: c = "MQIAMO_ACKS_RCVD"; break;
   case        807: c = "MQIAMO_ACTIVE_ACKERS"; break;
   case        808: c = "MQIAMO_PKTS_SENT"; break;
   case        809: c = "MQIAMO_TOTAL_REPAIR_PKTS"; break;
   case        810: c = "MQIAMO_TOTAL_PKTS_SENT"; break;
   case        811: c = "MQIAMO_TOTAL_MSGS_SENT"; break;
   case        812: c = "MQIAMO_TOTAL_BYTES_SENT"; break;
   case        813: c = "MQIAMO_NUM_STREAMS"; break;
   case        814: c = "MQIAMO_ACK_FEEDBACK"; break;
   case        815: c = "MQIAMO_NACK_FEEDBACK"; break;
   case        816: c = "MQIAMO_PKTS_LOST"; break;
   case        817: c = "MQIAMO_MSGS_RCVD"; break;
   case        818: c = "MQIAMO_MSG_BYTES_RCVD"; break;
   case        819: c = "MQIAMO_MSGS_DELIVERED"; break;
   case        820: c = "MQIAMO_PKTS_PROCESSED"; break;
   case        821: c = "MQIAMO_PKTS_DELIVERED"; break;
   case        822: c = "MQIAMO_PKTS_DROPPED"; break;
   case        823: c = "MQIAMO_PKTS_DUPLICATED"; break;
   case        824: c = "MQIAMO_NACKS_CREATED"; break;
   case        825: c = "MQIAMO_NACK_PKTS_SENT"; break;
   case        826: c = "MQIAMO_REPAIR_PKTS_RQSTD"; break;
   case        827: c = "MQIAMO_REPAIR_PKTS_RCVD"; break;
   case        828: c = "MQIAMO_PKTS_REPAIRED"; break;
   case        829: c = "MQIAMO_TOTAL_MSGS_RCVD"; break;
   case        830: c = "MQIAMO_TOTAL_MSG_BYTES_RCVD"; break;
   case        831: c = "MQIAMO_TOTAL_REPAIR_PKTS_RCVD"; break;
   case        832: c = "MQIAMO_TOTAL_REPAIR_PKTS_RQSTD"; break;
   case        833: c = "MQIAMO_TOTAL_MSGS_PROCESSED"; break;
   case        834: c = "MQIAMO_TOTAL_MSGS_SELECTED"; break;
   case        835: c = "MQIAMO_TOTAL_MSGS_EXPIRED"; break;
   case        836: c = "MQIAMO_TOTAL_MSGS_DELIVERED"; break;
   case        837: c = "MQIAMO_TOTAL_MSGS_RETURNED"; break;
   case        839: c = "MQIAMO_MONITOR_CLASS"; break;
   case        840: c = "MQIAMO_MONITOR_TYPE"; break;
   case        841: c = "MQIAMO_MONITOR_ELEMENT"; break;
   case        842: c = "MQIAMO_MONITOR_DATATYPE"; break;
   case        843: c = "MQIAMO_MONITOR_FLAGS"; break;
   case       1024: c = "MQIAMO_MONITOR_KB"; break;
   case      10000: c = "MQIAMO_MONITOR_PERCENT"; break;
   case    1000000: c = "MQIAMO_MONITOR_MICROSEC"; break;
   case    1048576: c = "MQIAMO_MONITOR_MB"; break;
   case  100000000: c = "MQIAMO_MONITOR_GB"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIASY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -9: c = "MQIASY_VERSION"; break;
   case         -8: c = "MQIASY_BAG_OPTIONS"; break;
   case         -7: c = "MQIASY_REASON"; break;
   case         -6: c = "MQIASY_COMP_CODE"; break;
   case         -5: c = "MQIASY_CONTROL"; break;
   case         -4: c = "MQIASY_MSG_SEQ_NUMBER"; break;
   case         -3: c = "MQIASY_COMMAND"; break;
   case         -2: c = "MQIASY_TYPE"; break;
   case         -1: c = "MQIASY_CODED_CHAR_SET_ID"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIAV_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQIAV_UNDEFINED"; break;
   case         -1: c = "MQIAV_NOT_APPLICABLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQIA_APPL_TYPE"; break;
   case          2: c = "MQIA_CODED_CHAR_SET_ID"; break;
   case          3: c = "MQIA_CURRENT_Q_DEPTH"; break;
   case          4: c = "MQIA_DEF_INPUT_OPEN_OPTION"; break;
   case          5: c = "MQIA_DEF_PERSISTENCE"; break;
   case          6: c = "MQIA_DEF_PRIORITY"; break;
   case          7: c = "MQIA_DEFINITION_TYPE"; break;
   case          8: c = "MQIA_HARDEN_GET_BACKOUT"; break;
   case          9: c = "MQIA_INHIBIT_GET"; break;
   case         10: c = "MQIA_INHIBIT_PUT"; break;
   case         11: c = "MQIA_MAX_HANDLES"; break;
   case         12: c = "MQIA_USAGE"; break;
   case         13: c = "MQIA_MAX_MSG_LENGTH"; break;
   case         14: c = "MQIA_MAX_PRIORITY"; break;
   case         15: c = "MQIA_MAX_Q_DEPTH"; break;
   case         16: c = "MQIA_MSG_DELIVERY_SEQUENCE"; break;
   case         17: c = "MQIA_OPEN_INPUT_COUNT"; break;
   case         18: c = "MQIA_OPEN_OUTPUT_COUNT"; break;
   case         19: c = "MQIA_NAME_COUNT"; break;
   case         20: c = "MQIA_Q_TYPE"; break;
   case         21: c = "MQIA_RETENTION_INTERVAL"; break;
   case         22: c = "MQIA_BACKOUT_THRESHOLD"; break;
   case         23: c = "MQIA_SHAREABILITY"; break;
   case         24: c = "MQIA_TRIGGER_CONTROL"; break;
   case         25: c = "MQIA_TRIGGER_INTERVAL"; break;
   case         26: c = "MQIA_TRIGGER_MSG_PRIORITY"; break;
   case         27: c = "MQIA_CPI_LEVEL"; break;
   case         28: c = "MQIA_TRIGGER_TYPE"; break;
   case         29: c = "MQIA_TRIGGER_DEPTH"; break;
   case         30: c = "MQIA_SYNCPOINT"; break;
   case         31: c = "MQIA_COMMAND_LEVEL"; break;
   case         32: c = "MQIA_PLATFORM"; break;
   case         33: c = "MQIA_MAX_UNCOMMITTED_MSGS"; break;
   case         34: c = "MQIA_DIST_LISTS"; break;
   case         35: c = "MQIA_TIME_SINCE_RESET"; break;
   case         36: c = "MQIA_HIGH_Q_DEPTH"; break;
   case         37: c = "MQIA_MSG_ENQ_COUNT"; break;
   case         38: c = "MQIA_MSG_DEQ_COUNT"; break;
   case         39: c = "MQIA_EXPIRY_INTERVAL"; break;
   case         40: c = "MQIA_Q_DEPTH_HIGH_LIMIT"; break;
   case         41: c = "MQIA_Q_DEPTH_LOW_LIMIT"; break;
   case         42: c = "MQIA_Q_DEPTH_MAX_EVENT"; break;
   case         43: c = "MQIA_Q_DEPTH_HIGH_EVENT"; break;
   case         44: c = "MQIA_Q_DEPTH_LOW_EVENT"; break;
   case         45: c = "MQIA_SCOPE"; break;
   case         46: c = "MQIA_Q_SERVICE_INTERVAL_EVENT"; break;
   case         47: c = "MQIA_AUTHORITY_EVENT"; break;
   case         48: c = "MQIA_INHIBIT_EVENT"; break;
   case         49: c = "MQIA_LOCAL_EVENT"; break;
   case         50: c = "MQIA_REMOTE_EVENT"; break;
   case         51: c = "MQIA_CONFIGURATION_EVENT"; break;
   case         52: c = "MQIA_START_STOP_EVENT"; break;
   case         53: c = "MQIA_PERFORMANCE_EVENT"; break;
   case         54: c = "MQIA_Q_SERVICE_INTERVAL"; break;
   case         55: c = "MQIA_CHANNEL_AUTO_DEF"; break;
   case         56: c = "MQIA_CHANNEL_AUTO_DEF_EVENT"; break;
   case         57: c = "MQIA_INDEX_TYPE"; break;
   case         58: c = "MQIA_CLUSTER_WORKLOAD_LENGTH"; break;
   case         59: c = "MQIA_CLUSTER_Q_TYPE"; break;
   case         60: c = "MQIA_ARCHIVE"; break;
   case         61: c = "MQIA_DEF_BIND"; break;
   case         62: c = "MQIA_PAGESET_ID"; break;
   case         63: c = "MQIA_QSG_DISP"; break;
   case         64: c = "MQIA_INTRA_GROUP_QUEUING"; break;
   case         65: c = "MQIA_IGQ_PUT_AUTHORITY"; break;
   case         66: c = "MQIA_AUTH_INFO_TYPE"; break;
   case         68: c = "MQIA_MSG_MARK_BROWSE_INTERVAL"; break;
   case         69: c = "MQIA_SSL_TASKS"; break;
   case         70: c = "MQIA_CF_LEVEL"; break;
   case         71: c = "MQIA_CF_RECOVER"; break;
   case         72: c = "MQIA_NAMELIST_TYPE"; break;
   case         73: c = "MQIA_CHANNEL_EVENT"; break;
   case         74: c = "MQIA_BRIDGE_EVENT"; break;
   case         75: c = "MQIA_SSL_EVENT"; break;
   case         76: c = "MQIA_SSL_RESET_COUNT"; break;
   case         77: c = "MQIA_SHARED_Q_Q_MGR_NAME"; break;
   case         78: c = "MQIA_NPM_CLASS"; break;
   case         80: c = "MQIA_MAX_OPEN_Q"; break;
   case         81: c = "MQIA_MONITOR_INTERVAL"; break;
   case         82: c = "MQIA_Q_USERS"; break;
   case         83: c = "MQIA_MAX_GLOBAL_LOCKS"; break;
   case         84: c = "MQIA_MAX_LOCAL_LOCKS"; break;
   case         85: c = "MQIA_LISTENER_PORT_NUMBER"; break;
   case         86: c = "MQIA_BATCH_INTERFACE_AUTO"; break;
   case         87: c = "MQIA_CMD_SERVER_AUTO"; break;
   case         88: c = "MQIA_CMD_SERVER_CONVERT_MSG"; break;
   case         89: c = "MQIA_CMD_SERVER_DLQ_MSG"; break;
   case         90: c = "MQIA_MAX_Q_TRIGGERS"; break;
   case         91: c = "MQIA_TRIGGER_RESTART"; break;
   case         92: c = "MQIA_SSL_FIPS_REQUIRED"; break;
   case         93: c = "MQIA_IP_ADDRESS_VERSION"; break;
   case         94: c = "MQIA_LOGGER_EVENT"; break;
   case         95: c = "MQIA_CLWL_Q_RANK"; break;
   case         96: c = "MQIA_CLWL_Q_PRIORITY"; break;
   case         97: c = "MQIA_CLWL_MRU_CHANNELS"; break;
   case         98: c = "MQIA_CLWL_USEQ"; break;
   case         99: c = "MQIA_COMMAND_EVENT"; break;
   case        100: c = "MQIA_ACTIVE_CHANNELS"; break;
   case        101: c = "MQIA_CHINIT_ADAPTERS"; break;
   case        102: c = "MQIA_ADOPTNEWMCA_CHECK"; break;
   case        103: c = "MQIA_ADOPTNEWMCA_TYPE"; break;
   case        104: c = "MQIA_ADOPTNEWMCA_INTERVAL"; break;
   case        105: c = "MQIA_CHINIT_DISPATCHERS"; break;
   case        106: c = "MQIA_DNS_WLM"; break;
   case        107: c = "MQIA_LISTENER_TIMER"; break;
   case        108: c = "MQIA_LU62_CHANNELS"; break;
   case        109: c = "MQIA_MAX_CHANNELS"; break;
   case        110: c = "MQIA_OUTBOUND_PORT_MIN"; break;
   case        111: c = "MQIA_RECEIVE_TIMEOUT"; break;
   case        112: c = "MQIA_RECEIVE_TIMEOUT_TYPE"; break;
   case        113: c = "MQIA_RECEIVE_TIMEOUT_MIN"; break;
   case        114: c = "MQIA_TCP_CHANNELS"; break;
   case        115: c = "MQIA_TCP_KEEP_ALIVE"; break;
   case        116: c = "MQIA_TCP_STACK_TYPE"; break;
   case        117: c = "MQIA_CHINIT_TRACE_AUTO_START"; break;
   case        118: c = "MQIA_CHINIT_TRACE_TABLE_SIZE"; break;
   case        119: c = "MQIA_CHINIT_CONTROL"; break;
   case        120: c = "MQIA_CMD_SERVER_CONTROL"; break;
   case        121: c = "MQIA_SERVICE_TYPE"; break;
   case        122: c = "MQIA_MONITORING_CHANNEL"; break;
   case        123: c = "MQIA_MONITORING_Q"; break;
   case        124: c = "MQIA_MONITORING_AUTO_CLUSSDR"; break;
   case        127: c = "MQIA_STATISTICS_MQI"; break;
   case        128: c = "MQIA_STATISTICS_Q"; break;
   case        129: c = "MQIA_STATISTICS_CHANNEL"; break;
   case        130: c = "MQIA_STATISTICS_AUTO_CLUSSDR"; break;
   case        131: c = "MQIA_STATISTICS_INTERVAL"; break;
   case        133: c = "MQIA_ACCOUNTING_MQI"; break;
   case        134: c = "MQIA_ACCOUNTING_Q"; break;
   case        135: c = "MQIA_ACCOUNTING_INTERVAL"; break;
   case        136: c = "MQIA_ACCOUNTING_CONN_OVERRIDE"; break;
   case        137: c = "MQIA_TRACE_ROUTE_RECORDING"; break;
   case        138: c = "MQIA_ACTIVITY_RECORDING"; break;
   case        139: c = "MQIA_SERVICE_CONTROL"; break;
   case        140: c = "MQIA_OUTBOUND_PORT_MAX"; break;
   case        141: c = "MQIA_SECURITY_CASE"; break;
   case        150: c = "MQIA_QMOPT_CSMT_ON_ERROR"; break;
   case        151: c = "MQIA_QMOPT_CONS_INFO_MSGS"; break;
   case        152: c = "MQIA_QMOPT_CONS_WARNING_MSGS"; break;
   case        153: c = "MQIA_QMOPT_CONS_ERROR_MSGS"; break;
   case        154: c = "MQIA_QMOPT_CONS_CRITICAL_MSGS"; break;
   case        155: c = "MQIA_QMOPT_CONS_COMMS_MSGS"; break;
   case        156: c = "MQIA_QMOPT_CONS_REORG_MSGS"; break;
   case        157: c = "MQIA_QMOPT_CONS_SYSTEM_MSGS"; break;
   case        158: c = "MQIA_QMOPT_LOG_INFO_MSGS"; break;
   case        159: c = "MQIA_QMOPT_LOG_WARNING_MSGS"; break;
   case        160: c = "MQIA_QMOPT_LOG_ERROR_MSGS"; break;
   case        161: c = "MQIA_QMOPT_LOG_CRITICAL_MSGS"; break;
   case        162: c = "MQIA_QMOPT_LOG_COMMS_MSGS"; break;
   case        163: c = "MQIA_QMOPT_LOG_REORG_MSGS"; break;
   case        164: c = "MQIA_QMOPT_LOG_SYSTEM_MSGS"; break;
   case        165: c = "MQIA_QMOPT_TRACE_MQI_CALLS"; break;
   case        166: c = "MQIA_QMOPT_TRACE_COMMS"; break;
   case        167: c = "MQIA_QMOPT_TRACE_REORG"; break;
   case        168: c = "MQIA_QMOPT_TRACE_CONVERSION"; break;
   case        169: c = "MQIA_QMOPT_TRACE_SYSTEM"; break;
   case        170: c = "MQIA_QMOPT_INTERNAL_DUMP"; break;
   case        171: c = "MQIA_MAX_RECOVERY_TASKS"; break;
   case        172: c = "MQIA_MAX_CLIENTS"; break;
   case        173: c = "MQIA_AUTO_REORGANIZATION"; break;
   case        174: c = "MQIA_AUTO_REORG_INTERVAL"; break;
   case        175: c = "MQIA_DURABLE_SUB"; break;
   case        176: c = "MQIA_MULTICAST"; break;
   case        181: c = "MQIA_INHIBIT_PUB"; break;
   case        182: c = "MQIA_INHIBIT_SUB"; break;
   case        183: c = "MQIA_TREE_LIFE_TIME"; break;
   case        184: c = "MQIA_DEF_PUT_RESPONSE_TYPE"; break;
   case        185: c = "MQIA_TOPIC_DEF_PERSISTENCE"; break;
   case        186: c = "MQIA_MASTER_ADMIN"; break;
   case        187: c = "MQIA_PUBSUB_MODE"; break;
   case        188: c = "MQIA_DEF_READ_AHEAD"; break;
   case        189: c = "MQIA_READ_AHEAD"; break;
   case        190: c = "MQIA_PROPERTY_CONTROL"; break;
   case        192: c = "MQIA_MAX_PROPERTIES_LENGTH"; break;
   case        193: c = "MQIA_BASE_TYPE"; break;
   case        195: c = "MQIA_PM_DELIVERY"; break;
   case        196: c = "MQIA_NPM_DELIVERY"; break;
   case        199: c = "MQIA_PROXY_SUB"; break;
   case        203: c = "MQIA_PUBSUB_NP_MSG"; break;
   case        204: c = "MQIA_SUB_COUNT"; break;
   case        205: c = "MQIA_PUBSUB_NP_RESP"; break;
   case        206: c = "MQIA_PUBSUB_MAXMSG_RETRY_COUNT"; break;
   case        207: c = "MQIA_PUBSUB_SYNC_PT"; break;
   case        208: c = "MQIA_TOPIC_TYPE"; break;
   case        215: c = "MQIA_PUB_COUNT"; break;
   case        216: c = "MQIA_WILDCARD_OPERATION"; break;
   case        218: c = "MQIA_SUB_SCOPE"; break;
   case        219: c = "MQIA_PUB_SCOPE"; break;
   case        221: c = "MQIA_GROUP_UR"; break;
   case        222: c = "MQIA_UR_DISP"; break;
   case        223: c = "MQIA_COMM_INFO_TYPE"; break;
   case        224: c = "MQIA_CF_OFFLOAD"; break;
   case        225: c = "MQIA_CF_OFFLOAD_THRESHOLD1"; break;
   case        226: c = "MQIA_CF_OFFLOAD_THRESHOLD2"; break;
   case        227: c = "MQIA_CF_OFFLOAD_THRESHOLD3"; break;
   case        228: c = "MQIA_CF_SMDS_BUFFERS"; break;
   case        229: c = "MQIA_CF_OFFLDUSE"; break;
   case        230: c = "MQIA_MAX_RESPONSES"; break;
   case        231: c = "MQIA_RESPONSE_RESTART_POINT"; break;
   case        232: c = "MQIA_COMM_EVENT"; break;
   case        233: c = "MQIA_MCAST_BRIDGE"; break;
   case        234: c = "MQIA_USE_DEAD_LETTER_Q"; break;
   case        235: c = "MQIA_TOLERATE_UNPROTECTED"; break;
   case        236: c = "MQIA_SIGNATURE_ALGORITHM"; break;
   case        237: c = "MQIA_ENCRYPTION_ALGORITHM"; break;
   case        238: c = "MQIA_POLICY_VERSION"; break;
   case        239: c = "MQIA_ACTIVITY_CONN_OVERRIDE"; break;
   case        240: c = "MQIA_ACTIVITY_TRACE"; break;
   case        242: c = "MQIA_SUB_CONFIGURATION_EVENT"; break;
   case        243: c = "MQIA_XR_CAPABILITY"; break;
   case        244: c = "MQIA_CF_RECAUTO"; break;
   case        245: c = "MQIA_QMGR_CFCONLOS"; break;
   case        246: c = "MQIA_CF_CFCONLOS"; break;
   case        247: c = "MQIA_SUITE_B_STRENGTH"; break;
   case        248: c = "MQIA_CHLAUTH_RECORDS"; break;
   case        249: c = "MQIA_PUBSUB_CLUSTER"; break;
   case        250: c = "MQIA_DEF_CLUSTER_XMIT_Q_TYPE"; break;
   case        251: c = "MQIA_PROT_POLICY_CAPABILITY"; break;
   case        252: c = "MQIA_CERT_VAL_POLICY"; break;
   case        253: c = "MQIA_TOPIC_NODE_COUNT"; break;
   case        254: c = "MQIA_REVERSE_DNS_LOOKUP"; break;
   case        255: c = "MQIA_CLUSTER_PUB_ROUTE"; break;
   case        256: c = "MQIA_CLUSTER_OBJECT_STATE"; break;
   case        257: c = "MQIA_CHECK_LOCAL_BINDING"; break;
   case        258: c = "MQIA_CHECK_CLIENT_BINDING"; break;
   case        259: c = "MQIA_AUTHENTICATION_FAIL_DELAY"; break;
   case        260: c = "MQIA_ADOPT_CONTEXT"; break;
   case        261: c = "MQIA_LDAP_SECURE_COMM"; break;
   case        262: c = "MQIA_DISPLAY_TYPE"; break;
   case        263: c = "MQIA_LDAP_AUTHORMD"; break;
   case        264: c = "MQIA_LDAP_NESTGRP"; break;
   case        265: c = "MQIA_AMQP_CAPABILITY"; break;
   case        266: c = "MQIA_AUTHENTICATION_METHOD"; break;
   case        267: c = "MQIA_KEY_REUSE_COUNT"; break;
   case        268: c = "MQIA_MEDIA_IMAGE_SCHEDULING"; break;
   case        269: c = "MQIA_MEDIA_IMAGE_INTERVAL"; break;
   case        270: c = "MQIA_MEDIA_IMAGE_LOG_LENGTH"; break;
   case        271: c = "MQIA_MEDIA_IMAGE_RECOVER_OBJ"; break;
   case        272: c = "MQIA_MEDIA_IMAGE_RECOVER_Q"; break;
   case        273: c = "MQIA_ADVANCED_CAPABILITY"; break;
   case       2000: c = "MQIA_USER_LIST"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIDO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQIDO_COMMIT"; break;
   case          2: c = "MQIDO_BACKOUT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIEPF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIEPF_NONE"; break;
   case          1: c = "MQIEPF_THREADED_LIBRARY"; break;
   case          2: c = "MQIEPF_LOCAL_LIBRARY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIGQPA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQIGQPA_DEFAULT"; break;
   case          2: c = "MQIGQPA_CONTEXT"; break;
   case          3: c = "MQIGQPA_ONLY_IGQ"; break;
   case          4: c = "MQIGQPA_ALTERNATE_OR_IGQ"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIGQ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIGQ_DISABLED"; break;
   case          1: c = "MQIGQ_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIIH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIIH_NONE"; break;
   case          1: c = "MQIIH_PASS_EXPIRATION"; break;
   case          8: c = "MQIIH_REPLY_FORMAT_NONE"; break;
   case         16: c = "MQIIH_IGNORE_PURG"; break;
   case         32: c = "MQIIH_CM0_REQUEST_RESPONSE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIMGRCOV_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIMGRCOV_NO"; break;
   case          1: c = "MQIMGRCOV_YES"; break;
   case          2: c = "MQIMGRCOV_AS_Q_MGR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIMPO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIMPO_NONE"; break;
   case          2: c = "MQIMPO_CONVERT_TYPE"; break;
   case          4: c = "MQIMPO_QUERY_LENGTH"; break;
   case          8: c = "MQIMPO_INQ_NEXT"; break;
   case         16: c = "MQIMPO_INQ_PROP_UNDER_CURSOR"; break;
   case         32: c = "MQIMPO_CONVERT_VALUE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQINBD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQINBD_Q_MGR"; break;
   case          3: c = "MQINBD_GROUP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIND_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQIND_ALL"; break;
   case         -1: c = "MQIND_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIPADDR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIPADDR_IPV4"; break;
   case          1: c = "MQIPADDR_IPV6"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIS_NO"; break;
   case          1: c = "MQIS_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQIT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQIT_NONE"; break;
   case          1: c = "MQIT_MSG_ID"; break;
   case          2: c = "MQIT_CORREL_ID"; break;
   case          4: c = "MQIT_MSG_TOKEN"; break;
   case          5: c = "MQIT_GROUP_ID"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQKAI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQKAI_AUTO"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQKEY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQKEY_REUSE_UNLIMITED"; break;
   case          0: c = "MQKEY_REUSE_DISABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQLDAPC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQLDAPC_INACTIVE"; break;
   case          1: c = "MQLDAPC_CONNECTED"; break;
   case          2: c = "MQLDAPC_ERROR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQLDAP_AUTHORMD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQLDAP_AUTHORMD_OS"; break;
   case          1: c = "MQLDAP_AUTHORMD_SEARCHGRP"; break;
   case          2: c = "MQLDAP_AUTHORMD_SEARCHUSR"; break;
   case          3: c = "MQLDAP_AUTHORMD_SRCHGRPSN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQLDAP_NESTGRP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQLDAP_NESTGRP_NO"; break;
   case          1: c = "MQLDAP_NESTGRP_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQLR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQLR_MAX"; break;
   case         -1: c = "MQLR_AUTO"; break;
   case          1: c = "MQLR_ONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMASTER_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMASTER_NO"; break;
   case          1: c = "MQMASTER_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMATCH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMATCH_GENERIC"; break;
   case          1: c = "MQMATCH_RUNCHECK"; break;
   case          2: c = "MQMATCH_EXACT"; break;
   case          3: c = "MQMATCH_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMCAS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMCAS_STOPPED"; break;
   case          3: c = "MQMCAS_RUNNING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMCAT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQMCAT_PROCESS"; break;
   case          2: c = "MQMCAT_THREAD"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMCB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMCB_DISABLED"; break;
   case          1: c = "MQMCB_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMCEV_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQMCEV_PACKET_LOSS"; break;
   case          2: c = "MQMCEV_HEARTBEAT_TIMEOUT"; break;
   case          3: c = "MQMCEV_VERSION_CONFLICT"; break;
   case          4: c = "MQMCEV_RELIABILITY"; break;
   case          5: c = "MQMCEV_CLOSED_TRANS"; break;
   case          6: c = "MQMCEV_STREAM_ERROR"; break;
   case         10: c = "MQMCEV_NEW_SOURCE"; break;
   case         11: c = "MQMCEV_RECEIVE_QUEUE_TRIMMED"; break;
   case         12: c = "MQMCEV_PACKET_LOSS_NACK_EXPIRE"; break;
   case         13: c = "MQMCEV_ACK_RETRIES_EXCEEDED"; break;
   case         14: c = "MQMCEV_STREAM_SUSPEND_NACK"; break;
   case         15: c = "MQMCEV_STREAM_RESUME_NACK"; break;
   case         16: c = "MQMCEV_STREAM_EXPELLED"; break;
   case         20: c = "MQMCEV_FIRST_MESSAGE"; break;
   case         21: c = "MQMCEV_LATE_JOIN_FAILURE"; break;
   case         22: c = "MQMCEV_MESSAGE_LOSS"; break;
   case         23: c = "MQMCEV_SEND_PACKET_FAILURE"; break;
   case         24: c = "MQMCEV_REPAIR_DELAY"; break;
   case         25: c = "MQMCEV_MEMORY_ALERT_ON"; break;
   case         26: c = "MQMCEV_MEMORY_ALERT_OFF"; break;
   case         27: c = "MQMCEV_NACK_ALERT_ON"; break;
   case         28: c = "MQMCEV_NACK_ALERT_OFF"; break;
   case         29: c = "MQMCEV_REPAIR_ALERT_ON"; break;
   case         30: c = "MQMCEV_REPAIR_ALERT_OFF"; break;
   case         31: c = "MQMCEV_RELIABILITY_CHANGED"; break;
   case         80: c = "MQMCEV_SHM_DEST_UNUSABLE"; break;
   case         81: c = "MQMCEV_SHM_PORT_UNUSABLE"; break;
   case        110: c = "MQMCEV_CCT_GETTIME_FAILED"; break;
   case        120: c = "MQMCEV_DEST_INTERFACE_FAILURE"; break;
   case        121: c = "MQMCEV_DEST_INTERFACE_FAILOVER"; break;
   case        122: c = "MQMCEV_PORT_INTERFACE_FAILURE"; break;
   case        123: c = "MQMCEV_PORT_INTERFACE_FAILOVER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMCP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQMCP_COMPAT"; break;
   case         -1: c = "MQMCP_ALL"; break;
   case          0: c = "MQMCP_NONE"; break;
   case          1: c = "MQMCP_USER"; break;
   case          2: c = "MQMCP_REPLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMC_AS_PARENT"; break;
   case          1: c = "MQMC_ENABLED"; break;
   case          2: c = "MQMC_DISABLED"; break;
   case          3: c = "MQMC_ONLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMDEF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMDEF_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMDS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMDS_PRIORITY"; break;
   case          1: c = "MQMDS_FIFO"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMEDIMGINTVL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMEDIMGINTVL_OFF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMEDIMGLOGLN_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMEDIMGLOGLN_OFF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMEDIMGSCHED_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMEDIMGSCHED_MANUAL"; break;
   case          1: c = "MQMEDIMGSCHED_AUTO"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case   -1048576: c = "MQMF_ACCEPT_UNSUP_MASK"; break;
   case          0: c = "MQMF_NONE"; break;
   case          1: c = "MQMF_SEGMENTATION_ALLOWED"; break;
   case          2: c = "MQMF_SEGMENT"; break;
   case          4: c = "MQMF_LAST_SEGMENT"; break;
   case          8: c = "MQMF_MSG_IN_GROUP"; break;
   case         16: c = "MQMF_LAST_MSG_IN_GROUP"; break;
   case       4095: c = "MQMF_REJECT_UNSUP_MASK"; break;
   case    1044480: c = "MQMF_ACCEPT_UNSUP_IF_XMIT_MASK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMHBO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMHBO_NONE"; break;
   case          1: c = "MQMHBO_PROPERTIES_IN_MQRFH2"; break;
   case          2: c = "MQMHBO_DELETE_PROPERTIES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMLP_ENCRYPTION_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMLP_ENCRYPTION_ALG_NONE"; break;
   case          1: c = "MQMLP_ENCRYPTION_ALG_RC2"; break;
   case          2: c = "MQMLP_ENCRYPTION_ALG_DES"; break;
   case          3: c = "MQMLP_ENCRYPTION_ALG_3DES"; break;
   case          4: c = "MQMLP_ENCRYPTION_ALG_AES128"; break;
   case          5: c = "MQMLP_ENCRYPTION_ALG_AES256"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMLP_SIGN_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMLP_SIGN_ALG_NONE"; break;
   case          1: c = "MQMLP_SIGN_ALG_MD5"; break;
   case          2: c = "MQMLP_SIGN_ALG_SHA1"; break;
   case          3: c = "MQMLP_SIGN_ALG_SHA224"; break;
   case          4: c = "MQMLP_SIGN_ALG_SHA256"; break;
   case          5: c = "MQMLP_SIGN_ALG_SHA384"; break;
   case          6: c = "MQMLP_SIGN_ALG_SHA512"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMLP_TOLERATE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMLP_TOLERATE_UNPROTECTED_NO"; break;
   case          1: c = "MQMLP_TOLERATE_UNPROTECTED_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMMBI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQMMBI_UNLIMITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMODE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMODE_FORCE"; break;
   case          1: c = "MQMODE_QUIESCE"; break;
   case          2: c = "MQMODE_TERMINATE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMON_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -3: c = "MQMON_Q_MGR"; break;
   case         -1: c = "MQMON_NONE"; break;
   case          0: c = "MQMON_OFF"; break;
   case          1: c = "MQMON_ON"; break;
   case         17: c = "MQMON_LOW"; break;
   case         33: c = "MQMON_MEDIUM"; break;
   case         65: c = "MQMON_HIGH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMO_NONE"; break;
   case          1: c = "MQMO_MATCH_MSG_ID"; break;
   case          2: c = "MQMO_MATCH_CORREL_ID"; break;
   case          4: c = "MQMO_MATCH_GROUP_ID"; break;
   case          8: c = "MQMO_MATCH_MSG_SEQ_NUMBER"; break;
   case         16: c = "MQMO_MATCH_OFFSET"; break;
   case         32: c = "MQMO_MATCH_MSG_TOKEN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQMT_REQUEST"; break;
   case          2: c = "MQMT_REPLY"; break;
   case          4: c = "MQMT_REPORT"; break;
   case          8: c = "MQMT_DATAGRAM"; break;
   case        112: c = "MQMT_MQE_FIELDS_FROM_MQE"; break;
   case        113: c = "MQMT_MQE_FIELDS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQMULC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQMULC_STANDARD"; break;
   case          1: c = "MQMULC_REFINED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQNC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case        256: c = "MQNC_MAX_NAMELIST_NAME_COUNT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQNPMS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQNPMS_NORMAL"; break;
   case          2: c = "MQNPMS_FAST"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQNPM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQNPM_CLASS_NORMAL"; break;
   case         10: c = "MQNPM_CLASS_HIGH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQNSH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQNSH_ALL"; break;
   case          0: c = "MQNSH_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQNT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQNT_NONE"; break;
   case          1: c = "MQNT_Q"; break;
   case          2: c = "MQNT_CLUSTER"; break;
   case          4: c = "MQNT_AUTH_INFO"; break;
   case       1001: c = "MQNT_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQOL_UNDEFINED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQOM_NO"; break;
   case          1: c = "MQOM_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQOO_READ_AHEAD_AS_Q_DEF"; break;
   case          1: c = "MQOO_INPUT_AS_Q_DEF"; break;
   case          2: c = "MQOO_INPUT_SHARED"; break;
   case          4: c = "MQOO_INPUT_EXCLUSIVE"; break;
   case          8: c = "MQOO_BROWSE"; break;
   case         16: c = "MQOO_OUTPUT"; break;
   case         32: c = "MQOO_INQUIRE"; break;
   case         64: c = "MQOO_SET"; break;
   case        128: c = "MQOO_SAVE_ALL_CONTEXT"; break;
   case        256: c = "MQOO_PASS_IDENTITY_CONTEXT"; break;
   case        512: c = "MQOO_PASS_ALL_CONTEXT"; break;
   case       1024: c = "MQOO_SET_IDENTITY_CONTEXT"; break;
   case       2048: c = "MQOO_SET_ALL_CONTEXT"; break;
   case       4096: c = "MQOO_ALTERNATE_USER_AUTHORITY"; break;
   case       8192: c = "MQOO_FAIL_IF_QUIESCING"; break;
   case      16384: c = "MQOO_BIND_ON_OPEN"; break;
   case      32768: c = "MQOO_BIND_NOT_FIXED"; break;
   case      65536: c = "MQOO_RESOLVE_NAMES"; break;
   case     131072: c = "MQOO_CO_OP"; break;
   case     262144: c = "MQOO_RESOLVE_LOCAL_Q"; break;
   case     524288: c = "MQOO_NO_READ_AHEAD"; break;
   case    1048576: c = "MQOO_READ_AHEAD"; break;
   case    2097152: c = "MQOO_NO_MULTICAST"; break;
   case    4194304: c = "MQOO_BIND_ON_GROUP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOPER_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQOPER_UNKNOWN"; break;
   case          1: c = "MQOPER_BROWSE"; break;
   case          2: c = "MQOPER_DISCARD"; break;
   case          3: c = "MQOPER_GET"; break;
   case          4: c = "MQOPER_PUT"; break;
   case          5: c = "MQOPER_PUT_REPLY"; break;
   case          6: c = "MQOPER_PUT_REPORT"; break;
   case          7: c = "MQOPER_RECEIVE"; break;
   case          8: c = "MQOPER_SEND"; break;
   case          9: c = "MQOPER_TRANSFORM"; break;
   case         10: c = "MQOPER_PUBLISH"; break;
   case         11: c = "MQOPER_EXCLUDED_PUBLISH"; break;
   case         12: c = "MQOPER_DISCARDED_PUBLISH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOPMODE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQOPMODE_COMPAT"; break;
   case          1: c = "MQOPMODE_NEW_FUNCTION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQOP_START"; break;
   case          2: c = "MQOP_START_WAIT"; break;
   case          4: c = "MQOP_STOP"; break;
   case        256: c = "MQOP_REGISTER"; break;
   case        512: c = "MQOP_DEREGISTER"; break;
   case      65536: c = "MQOP_SUSPEND"; break;
   case     131072: c = "MQOP_RESUME"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQOT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQOT_NONE"; break;
   case          1: c = "MQOT_Q"; break;
   case          2: c = "MQOT_NAMELIST"; break;
   case          3: c = "MQOT_PROCESS"; break;
   case          4: c = "MQOT_STORAGE_CLASS"; break;
   case          5: c = "MQOT_Q_MGR"; break;
   case          6: c = "MQOT_CHANNEL"; break;
   case          7: c = "MQOT_AUTH_INFO"; break;
   case          8: c = "MQOT_TOPIC"; break;
   case          9: c = "MQOT_COMM_INFO"; break;
   case         10: c = "MQOT_CF_STRUC"; break;
   case         11: c = "MQOT_LISTENER"; break;
   case         12: c = "MQOT_SERVICE"; break;
   case        999: c = "MQOT_RESERVED_1"; break;
   case       1001: c = "MQOT_ALL"; break;
   case       1002: c = "MQOT_ALIAS_Q"; break;
   case       1003: c = "MQOT_MODEL_Q"; break;
   case       1004: c = "MQOT_LOCAL_Q"; break;
   case       1005: c = "MQOT_REMOTE_Q"; break;
   case       1007: c = "MQOT_SENDER_CHANNEL"; break;
   case       1008: c = "MQOT_SERVER_CHANNEL"; break;
   case       1009: c = "MQOT_REQUESTER_CHANNEL"; break;
   case       1010: c = "MQOT_RECEIVER_CHANNEL"; break;
   case       1011: c = "MQOT_CURRENT_CHANNEL"; break;
   case       1012: c = "MQOT_SAVED_CHANNEL"; break;
   case       1013: c = "MQOT_SVRCONN_CHANNEL"; break;
   case       1014: c = "MQOT_CLNTCONN_CHANNEL"; break;
   case       1015: c = "MQOT_SHORT_CHANNEL"; break;
   case       1016: c = "MQOT_CHLAUTH"; break;
   case       1017: c = "MQOT_REMOTE_Q_MGR_NAME"; break;
   case       1019: c = "MQOT_PROT_POLICY"; break;
   case       1020: c = "MQOT_TT_CHANNEL"; break;
   case       1021: c = "MQOT_AMQP_CHANNEL"; break;
   case       1022: c = "MQOT_AUTH_REC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPAGECLAS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPAGECLAS_4KB"; break;
   case          1: c = "MQPAGECLAS_FIXED4KB"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQPA_DEFAULT"; break;
   case          2: c = "MQPA_CONTEXT"; break;
   case          3: c = "MQPA_ONLY_MCA"; break;
   case          4: c = "MQPA_ALTERNATE_OR_MCA"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case   -1048576: c = "MQPD_REJECT_UNSUP_MASK"; break;
   case          0: c = "MQPD_NONE"; break;
   case          1: c = "MQPD_SUPPORT_OPTIONAL"; break;
   case       1023: c = "MQPD_ACCEPT_UNSUP_MASK"; break;
   case       1024: c = "MQPD_SUPPORT_REQUIRED_IF_LOCAL"; break;
   case    1047552: c = "MQPD_ACCEPT_UNSUP_IF_XMIT_MASK"; break;
   case    1048576: c = "MQPD_SUPPORT_REQUIRED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPER_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQPER_PERSISTENCE_AS_PARENT"; break;
   case          0: c = "MQPER_NOT_PERSISTENT"; break;
   case          1: c = "MQPER_PERSISTENT"; break;
   case          2: c = "MQPER_PERSISTENCE_AS_Q_DEF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQPL_ZOS"; break;
   case          2: c = "MQPL_OS2"; break;
   case          3: c = "MQPL_UNIX"; break;
   case          4: c = "MQPL_OS400"; break;
   case          5: c = "MQPL_WINDOWS"; break;
   case         11: c = "MQPL_WINDOWS_NT"; break;
   case         12: c = "MQPL_VMS"; break;
   case         13: c = "MQPL_NSK"; break;
   case         15: c = "MQPL_OPEN_TP1"; break;
   case         18: c = "MQPL_VM"; break;
   case         23: c = "MQPL_TPF"; break;
   case         27: c = "MQPL_VSE"; break;
   case         28: c = "MQPL_APPLIANCE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPMO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPMO_NONE"; break;
   case          2: c = "MQPMO_SYNCPOINT"; break;
   case          4: c = "MQPMO_NO_SYNCPOINT"; break;
   case         32: c = "MQPMO_DEFAULT_CONTEXT"; break;
   case         64: c = "MQPMO_NEW_MSG_ID"; break;
   case        128: c = "MQPMO_NEW_CORREL_ID"; break;
   case        256: c = "MQPMO_PASS_IDENTITY_CONTEXT"; break;
   case        512: c = "MQPMO_PASS_ALL_CONTEXT"; break;
   case       1024: c = "MQPMO_SET_IDENTITY_CONTEXT"; break;
   case       2048: c = "MQPMO_SET_ALL_CONTEXT"; break;
   case       4096: c = "MQPMO_ALTERNATE_USER_AUTHORITY"; break;
   case       8192: c = "MQPMO_FAIL_IF_QUIESCING"; break;
   case      16384: c = "MQPMO_NO_CONTEXT"; break;
   case      32768: c = "MQPMO_LOGICAL_ORDER"; break;
   case      65536: c = "MQPMO_ASYNC_RESPONSE"; break;
   case     131072: c = "MQPMO_SYNC_RESPONSE"; break;
   case     262144: c = "MQPMO_RESOLVE_LOCAL_Q"; break;
   case     524288: c = "MQPMO_WARN_IF_NO_SUBS_MATCHED"; break;
   case    2097152: c = "MQPMO_RETAIN"; break;
   case    8388608: c = "MQPMO_MD_FOR_OUTPUT_ONLY"; break;
   case   67108864: c = "MQPMO_SCOPE_QMGR"; break;
   case  134217728: c = "MQPMO_SUPPRESS_REPLYTO"; break;
   case  268435456: c = "MQPMO_NOT_OWN_SUBS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPMRF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPMRF_NONE"; break;
   case          1: c = "MQPMRF_MSG_ID"; break;
   case          2: c = "MQPMRF_CORREL_ID"; break;
   case          4: c = "MQPMRF_GROUP_ID"; break;
   case          8: c = "MQPMRF_FEEDBACK"; break;
   case         16: c = "MQPMRF_ACCOUNTING_TOKEN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPO_NO"; break;
   case          1: c = "MQPO_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPRI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -3: c = "MQPRI_PRIORITY_AS_PUBLISHED"; break;
   case         -2: c = "MQPRI_PRIORITY_AS_PARENT"; break;
   case         -1: c = "MQPRI_PRIORITY_AS_Q_DEF"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPROP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQPROP_UNRESTRICTED_LENGTH"; break;
   case          0: c = "MQPROP_COMPATIBILITY"; break;
   case          1: c = "MQPROP_NONE"; break;
   case          2: c = "MQPROP_ALL"; break;
   case          3: c = "MQPROP_FORCE_MQRFH2"; break;
   case          4: c = "MQPROP_V6COMPAT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPROTO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQPROTO_MQTTV3"; break;
   case          2: c = "MQPROTO_HTTP"; break;
   case          3: c = "MQPROTO_AMQP"; break;
   case          4: c = "MQPROTO_MQTTV311"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPRT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPRT_RESPONSE_AS_PARENT"; break;
   case          1: c = "MQPRT_SYNC_RESPONSE"; break;
   case          2: c = "MQPRT_ASYNC_RESPONSE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPSCLUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPSCLUS_DISABLED"; break;
   case          1: c = "MQPSCLUS_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPSCT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQPSCT_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPSM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPSM_DISABLED"; break;
   case          1: c = "MQPSM_COMPAT"; break;
   case          2: c = "MQPSM_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPSPROP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPSPROP_NONE"; break;
   case          1: c = "MQPSPROP_COMPAT"; break;
   case          2: c = "MQPSPROP_RFH2"; break;
   case          3: c = "MQPSPROP_MSGPROP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPSST_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPSST_ALL"; break;
   case          1: c = "MQPSST_LOCAL"; break;
   case          2: c = "MQPSST_PARENT"; break;
   case          3: c = "MQPSST_CHILD"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPS_STATUS_INACTIVE"; break;
   case          1: c = "MQPS_STATUS_STARTING"; break;
   case          2: c = "MQPS_STATUS_STOPPING"; break;
   case          3: c = "MQPS_STATUS_ACTIVE"; break;
   case          4: c = "MQPS_STATUS_COMPAT"; break;
   case          5: c = "MQPS_STATUS_ERROR"; break;
   case          6: c = "MQPS_STATUS_REFUSED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQPUBO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQPUBO_NONE"; break;
   case          1: c = "MQPUBO_CORREL_ID_AS_IDENTITY"; break;
   case          2: c = "MQPUBO_RETAIN_PUBLICATION"; break;
   case          4: c = "MQPUBO_OTHER_SUBSCRIBERS_ONLY"; break;
   case          8: c = "MQPUBO_NO_REGISTRATION"; break;
   case         16: c = "MQPUBO_IS_RETAINED_PUBLICATION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQA_BACKOUT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQA_BACKOUT_NOT_HARDENED"; break;
   case          1: c = "MQQA_BACKOUT_HARDENED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQA_GET_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQA_GET_ALLOWED"; break;
   case          1: c = "MQQA_GET_INHIBITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQA_PUT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQA_PUT_ALLOWED"; break;
   case          1: c = "MQQA_PUT_INHIBITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQDT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQDT_PREDEFINED"; break;
   case          2: c = "MQQDT_PERMANENT_DYNAMIC"; break;
   case          3: c = "MQQDT_TEMPORARY_DYNAMIC"; break;
   case          4: c = "MQQDT_SHARED_DYNAMIC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQF_LOCAL_Q"; break;
   case         64: c = "MQQF_CLWL_USEQ_ANY"; break;
   case        128: c = "MQQF_CLWL_USEQ_LOCAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMDT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQMDT_EXPLICIT_CLUSTER_SENDER"; break;
   case          2: c = "MQQMDT_AUTO_CLUSTER_SENDER"; break;
   case          3: c = "MQQMDT_CLUSTER_RECEIVER"; break;
   case          4: c = "MQQMDT_AUTO_EXP_CLUSTER_SENDER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMFAC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQMFAC_IMS_BRIDGE"; break;
   case          2: c = "MQQMFAC_DB2"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          2: c = "MQQMF_REPOSITORY_Q_MGR"; break;
   case          8: c = "MQQMF_CLUSSDR_USER_DEFINED"; break;
   case         16: c = "MQQMF_CLUSSDR_AUTO_DEFINED"; break;
   case         32: c = "MQQMF_AVAILABLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMOPT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQMOPT_DISABLED"; break;
   case          1: c = "MQQMOPT_ENABLED"; break;
   case          2: c = "MQQMOPT_REPLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMSTA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQMSTA_STARTING"; break;
   case          2: c = "MQQMSTA_RUNNING"; break;
   case          3: c = "MQQMSTA_QUIESCING"; break;
   case          4: c = "MQQMSTA_STANDBY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQMT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQMT_NORMAL"; break;
   case          1: c = "MQQMT_REPOSITORY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQO_NO"; break;
   case          1: c = "MQQO_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSGD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQQSGD_ALL"; break;
   case          0: c = "MQQSGD_Q_MGR"; break;
   case          1: c = "MQQSGD_COPY"; break;
   case          2: c = "MQQSGD_SHARED"; break;
   case          3: c = "MQQSGD_GROUP"; break;
   case          4: c = "MQQSGD_PRIVATE"; break;
   case          6: c = "MQQSGD_LIVE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSGS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQSGS_UNKNOWN"; break;
   case          1: c = "MQQSGS_CREATED"; break;
   case          2: c = "MQQSGS_ACTIVE"; break;
   case          3: c = "MQQSGS_INACTIVE"; break;
   case          4: c = "MQQSGS_FAILED"; break;
   case          5: c = "MQQSGS_PENDING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSIE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQSIE_NONE"; break;
   case          1: c = "MQQSIE_HIGH"; break;
   case          2: c = "MQQSIE_OK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSOT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQSOT_ALL"; break;
   case          2: c = "MQQSOT_INPUT"; break;
   case          3: c = "MQQSOT_OUTPUT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQSO_NO"; break;
   case          1: c = "MQQSO_YES"; break;
   case          2: c = "MQQSO_EXCLUSIVE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQSUM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQQSUM_NO"; break;
   case          1: c = "MQQSUM_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQQT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQQT_LOCAL"; break;
   case          2: c = "MQQT_MODEL"; break;
   case          3: c = "MQQT_ALIAS"; break;
   case          6: c = "MQQT_REMOTE"; break;
   case          7: c = "MQQT_CLUSTER"; break;
   case       1001: c = "MQQT_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRAR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRAR_NO"; break;
   case          1: c = "MQRAR_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRCCF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case       3001: c = "MQRCCF_CFH_TYPE_ERROR"; break;
   case       3002: c = "MQRCCF_CFH_LENGTH_ERROR"; break;
   case       3003: c = "MQRCCF_CFH_VERSION_ERROR"; break;
   case       3004: c = "MQRCCF_CFH_MSG_SEQ_NUMBER_ERR"; break;
   case       3005: c = "MQRCCF_CFH_CONTROL_ERROR"; break;
   case       3006: c = "MQRCCF_CFH_PARM_COUNT_ERROR"; break;
   case       3007: c = "MQRCCF_CFH_COMMAND_ERROR"; break;
   case       3008: c = "MQRCCF_COMMAND_FAILED"; break;
   case       3009: c = "MQRCCF_CFIN_LENGTH_ERROR"; break;
   case       3010: c = "MQRCCF_CFST_LENGTH_ERROR"; break;
   case       3011: c = "MQRCCF_CFST_STRING_LENGTH_ERR"; break;
   case       3012: c = "MQRCCF_FORCE_VALUE_ERROR"; break;
   case       3013: c = "MQRCCF_STRUCTURE_TYPE_ERROR"; break;
   case       3014: c = "MQRCCF_CFIN_PARM_ID_ERROR"; break;
   case       3015: c = "MQRCCF_CFST_PARM_ID_ERROR"; break;
   case       3016: c = "MQRCCF_MSG_LENGTH_ERROR"; break;
   case       3017: c = "MQRCCF_CFIN_DUPLICATE_PARM"; break;
   case       3018: c = "MQRCCF_CFST_DUPLICATE_PARM"; break;
   case       3019: c = "MQRCCF_PARM_COUNT_TOO_SMALL"; break;
   case       3020: c = "MQRCCF_PARM_COUNT_TOO_BIG"; break;
   case       3021: c = "MQRCCF_Q_ALREADY_IN_CELL"; break;
   case       3022: c = "MQRCCF_Q_TYPE_ERROR"; break;
   case       3023: c = "MQRCCF_MD_FORMAT_ERROR"; break;
   case       3024: c = "MQRCCF_CFSL_LENGTH_ERROR"; break;
   case       3025: c = "MQRCCF_REPLACE_VALUE_ERROR"; break;
   case       3026: c = "MQRCCF_CFIL_DUPLICATE_VALUE"; break;
   case       3027: c = "MQRCCF_CFIL_COUNT_ERROR"; break;
   case       3028: c = "MQRCCF_CFIL_LENGTH_ERROR"; break;
   case       3029: c = "MQRCCF_QUIESCE_VALUE_ERROR"; break;
   case       3030: c = "MQRCCF_MSG_SEQ_NUMBER_ERROR"; break;
   case       3031: c = "MQRCCF_PING_DATA_COUNT_ERROR"; break;
   case       3032: c = "MQRCCF_PING_DATA_COMPARE_ERROR"; break;
   case       3033: c = "MQRCCF_CFSL_PARM_ID_ERROR"; break;
   case       3034: c = "MQRCCF_CHANNEL_TYPE_ERROR"; break;
   case       3035: c = "MQRCCF_PARM_SEQUENCE_ERROR"; break;
   case       3036: c = "MQRCCF_XMIT_PROTOCOL_TYPE_ERR"; break;
   case       3037: c = "MQRCCF_BATCH_SIZE_ERROR"; break;
   case       3038: c = "MQRCCF_DISC_INT_ERROR"; break;
   case       3039: c = "MQRCCF_SHORT_RETRY_ERROR"; break;
   case       3040: c = "MQRCCF_SHORT_TIMER_ERROR"; break;
   case       3041: c = "MQRCCF_LONG_RETRY_ERROR"; break;
   case       3042: c = "MQRCCF_LONG_TIMER_ERROR"; break;
   case       3043: c = "MQRCCF_SEQ_NUMBER_WRAP_ERROR"; break;
   case       3044: c = "MQRCCF_MAX_MSG_LENGTH_ERROR"; break;
   case       3045: c = "MQRCCF_PUT_AUTH_ERROR"; break;
   case       3046: c = "MQRCCF_PURGE_VALUE_ERROR"; break;
   case       3047: c = "MQRCCF_CFIL_PARM_ID_ERROR"; break;
   case       3048: c = "MQRCCF_MSG_TRUNCATED"; break;
   case       3049: c = "MQRCCF_CCSID_ERROR"; break;
   case       3050: c = "MQRCCF_ENCODING_ERROR"; break;
   case       3051: c = "MQRCCF_QUEUES_VALUE_ERROR"; break;
   case       3052: c = "MQRCCF_DATA_CONV_VALUE_ERROR"; break;
   case       3053: c = "MQRCCF_INDOUBT_VALUE_ERROR"; break;
   case       3054: c = "MQRCCF_ESCAPE_TYPE_ERROR"; break;
   case       3055: c = "MQRCCF_REPOS_VALUE_ERROR"; break;
   case       3062: c = "MQRCCF_CHANNEL_TABLE_ERROR"; break;
   case       3063: c = "MQRCCF_MCA_TYPE_ERROR"; break;
   case       3064: c = "MQRCCF_CHL_INST_TYPE_ERROR"; break;
   case       3065: c = "MQRCCF_CHL_STATUS_NOT_FOUND"; break;
   case       3066: c = "MQRCCF_CFSL_DUPLICATE_PARM"; break;
   case       3067: c = "MQRCCF_CFSL_TOTAL_LENGTH_ERROR"; break;
   case       3068: c = "MQRCCF_CFSL_COUNT_ERROR"; break;
   case       3069: c = "MQRCCF_CFSL_STRING_LENGTH_ERR"; break;
   case       3070: c = "MQRCCF_BROKER_DELETED"; break;
   case       3071: c = "MQRCCF_STREAM_ERROR"; break;
   case       3072: c = "MQRCCF_TOPIC_ERROR"; break;
   case       3073: c = "MQRCCF_NOT_REGISTERED"; break;
   case       3074: c = "MQRCCF_Q_MGR_NAME_ERROR"; break;
   case       3075: c = "MQRCCF_INCORRECT_STREAM"; break;
   case       3076: c = "MQRCCF_Q_NAME_ERROR"; break;
   case       3077: c = "MQRCCF_NO_RETAINED_MSG"; break;
   case       3078: c = "MQRCCF_DUPLICATE_IDENTITY"; break;
   case       3079: c = "MQRCCF_INCORRECT_Q"; break;
   case       3080: c = "MQRCCF_CORREL_ID_ERROR"; break;
   case       3081: c = "MQRCCF_NOT_AUTHORIZED"; break;
   case       3082: c = "MQRCCF_UNKNOWN_STREAM"; break;
   case       3083: c = "MQRCCF_REG_OPTIONS_ERROR"; break;
   case       3084: c = "MQRCCF_PUB_OPTIONS_ERROR"; break;
   case       3085: c = "MQRCCF_UNKNOWN_BROKER"; break;
   case       3086: c = "MQRCCF_Q_MGR_CCSID_ERROR"; break;
   case       3087: c = "MQRCCF_DEL_OPTIONS_ERROR"; break;
   case       3088: c = "MQRCCF_CLUSTER_NAME_CONFLICT"; break;
   case       3089: c = "MQRCCF_REPOS_NAME_CONFLICT"; break;
   case       3090: c = "MQRCCF_CLUSTER_Q_USAGE_ERROR"; break;
   case       3091: c = "MQRCCF_ACTION_VALUE_ERROR"; break;
   case       3092: c = "MQRCCF_COMMS_LIBRARY_ERROR"; break;
   case       3093: c = "MQRCCF_NETBIOS_NAME_ERROR"; break;
   case       3094: c = "MQRCCF_BROKER_COMMAND_FAILED"; break;
   case       3095: c = "MQRCCF_CFST_CONFLICTING_PARM"; break;
   case       3096: c = "MQRCCF_PATH_NOT_VALID"; break;
   case       3097: c = "MQRCCF_PARM_SYNTAX_ERROR"; break;
   case       3098: c = "MQRCCF_PWD_LENGTH_ERROR"; break;
   case       3150: c = "MQRCCF_FILTER_ERROR"; break;
   case       3151: c = "MQRCCF_WRONG_USER"; break;
   case       3152: c = "MQRCCF_DUPLICATE_SUBSCRIPTION"; break;
   case       3153: c = "MQRCCF_SUB_NAME_ERROR"; break;
   case       3154: c = "MQRCCF_SUB_IDENTITY_ERROR"; break;
   case       3155: c = "MQRCCF_SUBSCRIPTION_IN_USE"; break;
   case       3156: c = "MQRCCF_SUBSCRIPTION_LOCKED"; break;
   case       3157: c = "MQRCCF_ALREADY_JOINED"; break;
   case       3160: c = "MQRCCF_OBJECT_IN_USE"; break;
   case       3161: c = "MQRCCF_UNKNOWN_FILE_NAME"; break;
   case       3162: c = "MQRCCF_FILE_NOT_AVAILABLE"; break;
   case       3163: c = "MQRCCF_DISC_RETRY_ERROR"; break;
   case       3164: c = "MQRCCF_ALLOC_RETRY_ERROR"; break;
   case       3165: c = "MQRCCF_ALLOC_SLOW_TIMER_ERROR"; break;
   case       3166: c = "MQRCCF_ALLOC_FAST_TIMER_ERROR"; break;
   case       3167: c = "MQRCCF_PORT_NUMBER_ERROR"; break;
   case       3168: c = "MQRCCF_CHL_SYSTEM_NOT_ACTIVE"; break;
   case       3169: c = "MQRCCF_ENTITY_NAME_MISSING"; break;
   case       3170: c = "MQRCCF_PROFILE_NAME_ERROR"; break;
   case       3171: c = "MQRCCF_AUTH_VALUE_ERROR"; break;
   case       3172: c = "MQRCCF_AUTH_VALUE_MISSING"; break;
   case       3173: c = "MQRCCF_OBJECT_TYPE_MISSING"; break;
   case       3174: c = "MQRCCF_CONNECTION_ID_ERROR"; break;
   case       3175: c = "MQRCCF_LOG_TYPE_ERROR"; break;
   case       3176: c = "MQRCCF_PROGRAM_NOT_AVAILABLE"; break;
   case       3177: c = "MQRCCF_PROGRAM_AUTH_FAILED"; break;
   case       3200: c = "MQRCCF_NONE_FOUND"; break;
   case       3201: c = "MQRCCF_SECURITY_SWITCH_OFF"; break;
   case       3202: c = "MQRCCF_SECURITY_REFRESH_FAILED"; break;
   case       3203: c = "MQRCCF_PARM_CONFLICT"; break;
   case       3204: c = "MQRCCF_COMMAND_INHIBITED"; break;
   case       3205: c = "MQRCCF_OBJECT_BEING_DELETED"; break;
   case       3207: c = "MQRCCF_STORAGE_CLASS_IN_USE"; break;
   case       3208: c = "MQRCCF_OBJECT_NAME_RESTRICTED"; break;
   case       3209: c = "MQRCCF_OBJECT_LIMIT_EXCEEDED"; break;
   case       3210: c = "MQRCCF_OBJECT_OPEN_FORCE"; break;
   case       3211: c = "MQRCCF_DISPOSITION_CONFLICT"; break;
   case       3212: c = "MQRCCF_Q_MGR_NOT_IN_QSG"; break;
   case       3213: c = "MQRCCF_ATTR_VALUE_FIXED"; break;
   case       3215: c = "MQRCCF_NAMELIST_ERROR"; break;
   case       3217: c = "MQRCCF_NO_CHANNEL_INITIATOR"; break;
   case       3218: c = "MQRCCF_CHANNEL_INITIATOR_ERROR"; break;
   case       3222: c = "MQRCCF_COMMAND_LEVEL_CONFLICT"; break;
   case       3223: c = "MQRCCF_Q_ATTR_CONFLICT"; break;
   case       3224: c = "MQRCCF_EVENTS_DISABLED"; break;
   case       3225: c = "MQRCCF_COMMAND_SCOPE_ERROR"; break;
   case       3226: c = "MQRCCF_COMMAND_REPLY_ERROR"; break;
   case       3227: c = "MQRCCF_FUNCTION_RESTRICTED"; break;
   case       3228: c = "MQRCCF_PARM_MISSING"; break;
   case       3229: c = "MQRCCF_PARM_VALUE_ERROR"; break;
   case       3230: c = "MQRCCF_COMMAND_LENGTH_ERROR"; break;
   case       3231: c = "MQRCCF_COMMAND_ORIGIN_ERROR"; break;
   case       3232: c = "MQRCCF_LISTENER_CONFLICT"; break;
   case       3233: c = "MQRCCF_LISTENER_STARTED"; break;
   case       3234: c = "MQRCCF_LISTENER_STOPPED"; break;
   case       3235: c = "MQRCCF_CHANNEL_ERROR"; break;
   case       3236: c = "MQRCCF_CF_STRUC_ERROR"; break;
   case       3237: c = "MQRCCF_UNKNOWN_USER_ID"; break;
   case       3238: c = "MQRCCF_UNEXPECTED_ERROR"; break;
   case       3239: c = "MQRCCF_NO_XCF_PARTNER"; break;
   case       3240: c = "MQRCCF_CFGR_PARM_ID_ERROR"; break;
   case       3241: c = "MQRCCF_CFIF_LENGTH_ERROR"; break;
   case       3242: c = "MQRCCF_CFIF_OPERATOR_ERROR"; break;
   case       3243: c = "MQRCCF_CFIF_PARM_ID_ERROR"; break;
   case       3244: c = "MQRCCF_CFSF_FILTER_VAL_LEN_ERR"; break;
   case       3245: c = "MQRCCF_CFSF_LENGTH_ERROR"; break;
   case       3246: c = "MQRCCF_CFSF_OPERATOR_ERROR"; break;
   case       3247: c = "MQRCCF_CFSF_PARM_ID_ERROR"; break;
   case       3248: c = "MQRCCF_TOO_MANY_FILTERS"; break;
   case       3249: c = "MQRCCF_LISTENER_RUNNING"; break;
   case       3250: c = "MQRCCF_LSTR_STATUS_NOT_FOUND"; break;
   case       3251: c = "MQRCCF_SERVICE_RUNNING"; break;
   case       3252: c = "MQRCCF_SERV_STATUS_NOT_FOUND"; break;
   case       3253: c = "MQRCCF_SERVICE_STOPPED"; break;
   case       3254: c = "MQRCCF_CFBS_DUPLICATE_PARM"; break;
   case       3255: c = "MQRCCF_CFBS_LENGTH_ERROR"; break;
   case       3256: c = "MQRCCF_CFBS_PARM_ID_ERROR"; break;
   case       3257: c = "MQRCCF_CFBS_STRING_LENGTH_ERR"; break;
   case       3258: c = "MQRCCF_CFGR_LENGTH_ERROR"; break;
   case       3259: c = "MQRCCF_CFGR_PARM_COUNT_ERROR"; break;
   case       3260: c = "MQRCCF_CONN_NOT_STOPPED"; break;
   case       3261: c = "MQRCCF_SERVICE_REQUEST_PENDING"; break;
   case       3262: c = "MQRCCF_NO_START_CMD"; break;
   case       3263: c = "MQRCCF_NO_STOP_CMD"; break;
   case       3264: c = "MQRCCF_CFBF_LENGTH_ERROR"; break;
   case       3265: c = "MQRCCF_CFBF_PARM_ID_ERROR"; break;
   case       3266: c = "MQRCCF_CFBF_OPERATOR_ERROR"; break;
   case       3267: c = "MQRCCF_CFBF_FILTER_VAL_LEN_ERR"; break;
   case       3268: c = "MQRCCF_LISTENER_STILL_ACTIVE"; break;
   case       3269: c = "MQRCCF_DEF_XMIT_Q_CLUS_ERROR"; break;
   case       3300: c = "MQRCCF_TOPICSTR_ALREADY_EXISTS"; break;
   case       3301: c = "MQRCCF_SHARING_CONVS_ERROR"; break;
   case       3302: c = "MQRCCF_SHARING_CONVS_TYPE"; break;
   case       3303: c = "MQRCCF_SECURITY_CASE_CONFLICT"; break;
   case       3305: c = "MQRCCF_TOPIC_TYPE_ERROR"; break;
   case       3306: c = "MQRCCF_MAX_INSTANCES_ERROR"; break;
   case       3307: c = "MQRCCF_MAX_INSTS_PER_CLNT_ERR"; break;
   case       3308: c = "MQRCCF_TOPIC_STRING_NOT_FOUND"; break;
   case       3309: c = "MQRCCF_SUBSCRIPTION_POINT_ERR"; break;
   case       3311: c = "MQRCCF_SUB_ALREADY_EXISTS"; break;
   case       3312: c = "MQRCCF_UNKNOWN_OBJECT_NAME"; break;
   case       3313: c = "MQRCCF_REMOTE_Q_NAME_ERROR"; break;
   case       3314: c = "MQRCCF_DURABILITY_NOT_ALLOWED"; break;
   case       3315: c = "MQRCCF_HOBJ_ERROR"; break;
   case       3316: c = "MQRCCF_DEST_NAME_ERROR"; break;
   case       3317: c = "MQRCCF_INVALID_DESTINATION"; break;
   case       3318: c = "MQRCCF_PUBSUB_INHIBITED"; break;
   case       3319: c = "MQRCCF_GROUPUR_CHECKS_FAILED"; break;
   case       3320: c = "MQRCCF_COMM_INFO_TYPE_ERROR"; break;
   case       3321: c = "MQRCCF_USE_CLIENT_ID_ERROR"; break;
   case       3322: c = "MQRCCF_CLIENT_ID_NOT_FOUND"; break;
   case       3323: c = "MQRCCF_CLIENT_ID_ERROR"; break;
   case       3324: c = "MQRCCF_PORT_IN_USE"; break;
   case       3325: c = "MQRCCF_SSL_ALT_PROVIDER_REQD"; break;
   case       3326: c = "MQRCCF_CHLAUTH_TYPE_ERROR"; break;
   case       3327: c = "MQRCCF_CHLAUTH_ACTION_ERROR"; break;
   case       3328: c = "MQRCCF_POLICY_NOT_FOUND"; break;
   case       3329: c = "MQRCCF_ENCRYPTION_ALG_ERROR"; break;
   case       3330: c = "MQRCCF_SIGNATURE_ALG_ERROR"; break;
   case       3331: c = "MQRCCF_TOLERATION_POL_ERROR"; break;
   case       3332: c = "MQRCCF_POLICY_VERSION_ERROR"; break;
   case       3333: c = "MQRCCF_RECIPIENT_DN_MISSING"; break;
   case       3334: c = "MQRCCF_POLICY_NAME_MISSING"; break;
   case       3335: c = "MQRCCF_CHLAUTH_USERSRC_ERROR"; break;
   case       3336: c = "MQRCCF_WRONG_CHLAUTH_TYPE"; break;
   case       3337: c = "MQRCCF_CHLAUTH_ALREADY_EXISTS"; break;
   case       3338: c = "MQRCCF_CHLAUTH_NOT_FOUND"; break;
   case       3339: c = "MQRCCF_WRONG_CHLAUTH_ACTION"; break;
   case       3340: c = "MQRCCF_WRONG_CHLAUTH_USERSRC"; break;
   case       3341: c = "MQRCCF_CHLAUTH_WARN_ERROR"; break;
   case       3342: c = "MQRCCF_WRONG_CHLAUTH_MATCH"; break;
   case       3343: c = "MQRCCF_IPADDR_RANGE_CONFLICT"; break;
   case       3344: c = "MQRCCF_CHLAUTH_MAX_EXCEEDED"; break;
   case       3345: c = "MQRCCF_ADDRESS_ERROR"; break;
   case       3346: c = "MQRCCF_IPADDR_RANGE_ERROR"; break;
   case       3347: c = "MQRCCF_PROFILE_NAME_MISSING"; break;
   case       3348: c = "MQRCCF_CHLAUTH_CLNTUSER_ERROR"; break;
   case       3349: c = "MQRCCF_CHLAUTH_NAME_ERROR"; break;
   case       3350: c = "MQRCCF_CHLAUTH_RUNCHECK_ERROR"; break;
   case       3351: c = "MQRCCF_CF_STRUC_ALREADY_FAILED"; break;
   case       3352: c = "MQRCCF_CFCONLOS_CHECKS_FAILED"; break;
   case       3353: c = "MQRCCF_SUITE_B_ERROR"; break;
   case       3354: c = "MQRCCF_CHANNEL_NOT_STARTED"; break;
   case       3355: c = "MQRCCF_CUSTOM_ERROR"; break;
   case       3356: c = "MQRCCF_BACKLOG_OUT_OF_RANGE"; break;
   case       3357: c = "MQRCCF_CHLAUTH_DISABLED"; break;
   case       3358: c = "MQRCCF_SMDS_REQUIRES_DSGROUP"; break;
   case       3359: c = "MQRCCF_PSCLUS_DISABLED_TOPDEF"; break;
   case       3360: c = "MQRCCF_PSCLUS_TOPIC_EXISTS"; break;
   case       3361: c = "MQRCCF_SSL_CIPHER_SUITE_ERROR"; break;
   case       3362: c = "MQRCCF_SOCKET_ERROR"; break;
   case       3363: c = "MQRCCF_CLUS_XMIT_Q_USAGE_ERROR"; break;
   case       3364: c = "MQRCCF_CERT_VAL_POLICY_ERROR"; break;
   case       3365: c = "MQRCCF_INVALID_PROTOCOL"; break;
   case       3366: c = "MQRCCF_REVDNS_DISABLED"; break;
   case       3367: c = "MQRCCF_CLROUTE_NOT_ALTERABLE"; break;
   case       3368: c = "MQRCCF_CLUSTER_TOPIC_CONFLICT"; break;
   case       3369: c = "MQRCCF_DEFCLXQ_MODEL_Q_ERROR"; break;
   case       3370: c = "MQRCCF_CHLAUTH_CHKCLI_ERROR"; break;
   case       3371: c = "MQRCCF_CERT_LABEL_NOT_ALLOWED"; break;
   case       3372: c = "MQRCCF_Q_MGR_ATTR_CONFLICT"; break;
   case       3373: c = "MQRCCF_ENTITY_TYPE_MISSING"; break;
   case       3374: c = "MQRCCF_CLWL_EXIT_NAME_ERROR"; break;
   case       3375: c = "MQRCCF_SERVICE_NAME_ERROR"; break;
   case       3376: c = "MQRCCF_REMOTE_CHL_TYPE_ERROR"; break;
   case       3377: c = "MQRCCF_TOPIC_RESTRICTED"; break;
   case       3378: c = "MQRCCF_CURRENT_LOG_EXTENT"; break;
   case       3379: c = "MQRCCF_LOG_EXTENT_NOT_FOUND"; break;
   case       3380: c = "MQRCCF_LOG_NOT_REDUCED"; break;
   case       3381: c = "MQRCCF_LOG_EXTENT_ERROR"; break;
   case       3382: c = "MQRCCF_ACCESS_BLOCKED"; break;
   case       4001: c = "MQRCCF_OBJECT_ALREADY_EXISTS"; break;
   case       4002: c = "MQRCCF_OBJECT_WRONG_TYPE"; break;
   case       4003: c = "MQRCCF_LIKE_OBJECT_WRONG_TYPE"; break;
   case       4004: c = "MQRCCF_OBJECT_OPEN"; break;
   case       4005: c = "MQRCCF_ATTR_VALUE_ERROR"; break;
   case       4006: c = "MQRCCF_UNKNOWN_Q_MGR"; break;
   case       4007: c = "MQRCCF_Q_WRONG_TYPE"; break;
   case       4008: c = "MQRCCF_OBJECT_NAME_ERROR"; break;
   case       4009: c = "MQRCCF_ALLOCATE_FAILED"; break;
   case       4010: c = "MQRCCF_HOST_NOT_AVAILABLE"; break;
   case       4011: c = "MQRCCF_CONFIGURATION_ERROR"; break;
   case       4012: c = "MQRCCF_CONNECTION_REFUSED"; break;
   case       4013: c = "MQRCCF_ENTRY_ERROR"; break;
   case       4014: c = "MQRCCF_SEND_FAILED"; break;
   case       4015: c = "MQRCCF_RECEIVED_DATA_ERROR"; break;
   case       4016: c = "MQRCCF_RECEIVE_FAILED"; break;
   case       4017: c = "MQRCCF_CONNECTION_CLOSED"; break;
   case       4018: c = "MQRCCF_NO_STORAGE"; break;
   case       4019: c = "MQRCCF_NO_COMMS_MANAGER"; break;
   case       4020: c = "MQRCCF_LISTENER_NOT_STARTED"; break;
   case       4024: c = "MQRCCF_BIND_FAILED"; break;
   case       4025: c = "MQRCCF_CHANNEL_INDOUBT"; break;
   case       4026: c = "MQRCCF_MQCONN_FAILED"; break;
   case       4027: c = "MQRCCF_MQOPEN_FAILED"; break;
   case       4028: c = "MQRCCF_MQGET_FAILED"; break;
   case       4029: c = "MQRCCF_MQPUT_FAILED"; break;
   case       4030: c = "MQRCCF_PING_ERROR"; break;
   case       4031: c = "MQRCCF_CHANNEL_IN_USE"; break;
   case       4032: c = "MQRCCF_CHANNEL_NOT_FOUND"; break;
   case       4033: c = "MQRCCF_UNKNOWN_REMOTE_CHANNEL"; break;
   case       4034: c = "MQRCCF_REMOTE_QM_UNAVAILABLE"; break;
   case       4035: c = "MQRCCF_REMOTE_QM_TERMINATING"; break;
   case       4036: c = "MQRCCF_MQINQ_FAILED"; break;
   case       4037: c = "MQRCCF_NOT_XMIT_Q"; break;
   case       4038: c = "MQRCCF_CHANNEL_DISABLED"; break;
   case       4039: c = "MQRCCF_USER_EXIT_NOT_AVAILABLE"; break;
   case       4040: c = "MQRCCF_COMMIT_FAILED"; break;
   case       4041: c = "MQRCCF_WRONG_CHANNEL_TYPE"; break;
   case       4042: c = "MQRCCF_CHANNEL_ALREADY_EXISTS"; break;
   case       4043: c = "MQRCCF_DATA_TOO_LARGE"; break;
   case       4044: c = "MQRCCF_CHANNEL_NAME_ERROR"; break;
   case       4045: c = "MQRCCF_XMIT_Q_NAME_ERROR"; break;
   case       4047: c = "MQRCCF_MCA_NAME_ERROR"; break;
   case       4048: c = "MQRCCF_SEND_EXIT_NAME_ERROR"; break;
   case       4049: c = "MQRCCF_SEC_EXIT_NAME_ERROR"; break;
   case       4050: c = "MQRCCF_MSG_EXIT_NAME_ERROR"; break;
   case       4051: c = "MQRCCF_RCV_EXIT_NAME_ERROR"; break;
   case       4052: c = "MQRCCF_XMIT_Q_NAME_WRONG_TYPE"; break;
   case       4053: c = "MQRCCF_MCA_NAME_WRONG_TYPE"; break;
   case       4054: c = "MQRCCF_DISC_INT_WRONG_TYPE"; break;
   case       4055: c = "MQRCCF_SHORT_RETRY_WRONG_TYPE"; break;
   case       4056: c = "MQRCCF_SHORT_TIMER_WRONG_TYPE"; break;
   case       4057: c = "MQRCCF_LONG_RETRY_WRONG_TYPE"; break;
   case       4058: c = "MQRCCF_LONG_TIMER_WRONG_TYPE"; break;
   case       4059: c = "MQRCCF_PUT_AUTH_WRONG_TYPE"; break;
   case       4060: c = "MQRCCF_KEEP_ALIVE_INT_ERROR"; break;
   case       4061: c = "MQRCCF_MISSING_CONN_NAME"; break;
   case       4062: c = "MQRCCF_CONN_NAME_ERROR"; break;
   case       4063: c = "MQRCCF_MQSET_FAILED"; break;
   case       4064: c = "MQRCCF_CHANNEL_NOT_ACTIVE"; break;
   case       4065: c = "MQRCCF_TERMINATED_BY_SEC_EXIT"; break;
   case       4067: c = "MQRCCF_DYNAMIC_Q_SCOPE_ERROR"; break;
   case       4068: c = "MQRCCF_CELL_DIR_NOT_AVAILABLE"; break;
   case       4069: c = "MQRCCF_MR_COUNT_ERROR"; break;
   case       4070: c = "MQRCCF_MR_COUNT_WRONG_TYPE"; break;
   case       4071: c = "MQRCCF_MR_EXIT_NAME_ERROR"; break;
   case       4072: c = "MQRCCF_MR_EXIT_NAME_WRONG_TYPE"; break;
   case       4073: c = "MQRCCF_MR_INTERVAL_ERROR"; break;
   case       4074: c = "MQRCCF_MR_INTERVAL_WRONG_TYPE"; break;
   case       4075: c = "MQRCCF_NPM_SPEED_ERROR"; break;
   case       4076: c = "MQRCCF_NPM_SPEED_WRONG_TYPE"; break;
   case       4077: c = "MQRCCF_HB_INTERVAL_ERROR"; break;
   case       4078: c = "MQRCCF_HB_INTERVAL_WRONG_TYPE"; break;
   case       4079: c = "MQRCCF_CHAD_ERROR"; break;
   case       4080: c = "MQRCCF_CHAD_WRONG_TYPE"; break;
   case       4081: c = "MQRCCF_CHAD_EVENT_ERROR"; break;
   case       4082: c = "MQRCCF_CHAD_EVENT_WRONG_TYPE"; break;
   case       4083: c = "MQRCCF_CHAD_EXIT_ERROR"; break;
   case       4084: c = "MQRCCF_CHAD_EXIT_WRONG_TYPE"; break;
   case       4085: c = "MQRCCF_SUPPRESSED_BY_EXIT"; break;
   case       4086: c = "MQRCCF_BATCH_INT_ERROR"; break;
   case       4087: c = "MQRCCF_BATCH_INT_WRONG_TYPE"; break;
   case       4088: c = "MQRCCF_NET_PRIORITY_ERROR"; break;
   case       4089: c = "MQRCCF_NET_PRIORITY_WRONG_TYPE"; break;
   case       4090: c = "MQRCCF_CHANNEL_CLOSED"; break;
   case       4091: c = "MQRCCF_Q_STATUS_NOT_FOUND"; break;
   case       4092: c = "MQRCCF_SSL_CIPHER_SPEC_ERROR"; break;
   case       4093: c = "MQRCCF_SSL_PEER_NAME_ERROR"; break;
   case       4094: c = "MQRCCF_SSL_CLIENT_AUTH_ERROR"; break;
   case       4095: c = "MQRCCF_RETAINED_NOT_SUPPORTED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRCN_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRCN_NO"; break;
   case          1: c = "MQRCN_YES"; break;
   case          2: c = "MQRCN_Q_MGR"; break;
   case          3: c = "MQRCN_DISABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRCVTIME_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRCVTIME_MULTIPLY"; break;
   case          1: c = "MQRCVTIME_ADD"; break;
   case          2: c = "MQRCVTIME_EQUAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRC_NONE"; break;
   case       2001: c = "MQRC_ALIAS_BASE_Q_TYPE_ERROR"; break;
   case       2002: c = "MQRC_ALREADY_CONNECTED"; break;
   case       2003: c = "MQRC_BACKED_OUT"; break;
   case       2004: c = "MQRC_BUFFER_ERROR"; break;
   case       2005: c = "MQRC_BUFFER_LENGTH_ERROR"; break;
   case       2006: c = "MQRC_CHAR_ATTR_LENGTH_ERROR"; break;
   case       2007: c = "MQRC_CHAR_ATTRS_ERROR"; break;
   case       2008: c = "MQRC_CHAR_ATTRS_TOO_SHORT"; break;
   case       2009: c = "MQRC_CONNECTION_BROKEN"; break;
   case       2010: c = "MQRC_DATA_LENGTH_ERROR"; break;
   case       2011: c = "MQRC_DYNAMIC_Q_NAME_ERROR"; break;
   case       2012: c = "MQRC_ENVIRONMENT_ERROR"; break;
   case       2013: c = "MQRC_EXPIRY_ERROR"; break;
   case       2014: c = "MQRC_FEEDBACK_ERROR"; break;
   case       2016: c = "MQRC_GET_INHIBITED"; break;
   case       2017: c = "MQRC_HANDLE_NOT_AVAILABLE"; break;
   case       2018: c = "MQRC_HCONN_ERROR"; break;
   case       2019: c = "MQRC_HOBJ_ERROR"; break;
   case       2020: c = "MQRC_INHIBIT_VALUE_ERROR"; break;
   case       2021: c = "MQRC_INT_ATTR_COUNT_ERROR"; break;
   case       2022: c = "MQRC_INT_ATTR_COUNT_TOO_SMALL"; break;
   case       2023: c = "MQRC_INT_ATTRS_ARRAY_ERROR"; break;
   case       2024: c = "MQRC_SYNCPOINT_LIMIT_REACHED"; break;
   case       2025: c = "MQRC_MAX_CONNS_LIMIT_REACHED"; break;
   case       2026: c = "MQRC_MD_ERROR"; break;
   case       2027: c = "MQRC_MISSING_REPLY_TO_Q"; break;
   case       2029: c = "MQRC_MSG_TYPE_ERROR"; break;
   case       2030: c = "MQRC_MSG_TOO_BIG_FOR_Q"; break;
   case       2031: c = "MQRC_MSG_TOO_BIG_FOR_Q_MGR"; break;
   case       2033: c = "MQRC_NO_MSG_AVAILABLE"; break;
   case       2034: c = "MQRC_NO_MSG_UNDER_CURSOR"; break;
   case       2035: c = "MQRC_NOT_AUTHORIZED"; break;
   case       2036: c = "MQRC_NOT_OPEN_FOR_BROWSE"; break;
   case       2037: c = "MQRC_NOT_OPEN_FOR_INPUT"; break;
   case       2038: c = "MQRC_NOT_OPEN_FOR_INQUIRE"; break;
   case       2039: c = "MQRC_NOT_OPEN_FOR_OUTPUT"; break;
   case       2040: c = "MQRC_NOT_OPEN_FOR_SET"; break;
   case       2041: c = "MQRC_OBJECT_CHANGED"; break;
   case       2042: c = "MQRC_OBJECT_IN_USE"; break;
   case       2043: c = "MQRC_OBJECT_TYPE_ERROR"; break;
   case       2044: c = "MQRC_OD_ERROR"; break;
   case       2045: c = "MQRC_OPTION_NOT_VALID_FOR_TYPE"; break;
   case       2046: c = "MQRC_OPTIONS_ERROR"; break;
   case       2047: c = "MQRC_PERSISTENCE_ERROR"; break;
   case       2048: c = "MQRC_PERSISTENT_NOT_ALLOWED"; break;
   case       2049: c = "MQRC_PRIORITY_EXCEEDS_MAXIMUM"; break;
   case       2050: c = "MQRC_PRIORITY_ERROR"; break;
   case       2051: c = "MQRC_PUT_INHIBITED"; break;
   case       2052: c = "MQRC_Q_DELETED"; break;
   case       2053: c = "MQRC_Q_FULL"; break;
   case       2055: c = "MQRC_Q_NOT_EMPTY"; break;
   case       2056: c = "MQRC_Q_SPACE_NOT_AVAILABLE"; break;
   case       2057: c = "MQRC_Q_TYPE_ERROR"; break;
   case       2058: c = "MQRC_Q_MGR_NAME_ERROR"; break;
   case       2059: c = "MQRC_Q_MGR_NOT_AVAILABLE"; break;
   case       2061: c = "MQRC_REPORT_OPTIONS_ERROR"; break;
   case       2062: c = "MQRC_SECOND_MARK_NOT_ALLOWED"; break;
   case       2063: c = "MQRC_SECURITY_ERROR"; break;
   case       2065: c = "MQRC_SELECTOR_COUNT_ERROR"; break;
   case       2066: c = "MQRC_SELECTOR_LIMIT_EXCEEDED"; break;
   case       2067: c = "MQRC_SELECTOR_ERROR"; break;
   case       2068: c = "MQRC_SELECTOR_NOT_FOR_TYPE"; break;
   case       2069: c = "MQRC_SIGNAL_OUTSTANDING"; break;
   case       2070: c = "MQRC_SIGNAL_REQUEST_ACCEPTED"; break;
   case       2071: c = "MQRC_STORAGE_NOT_AVAILABLE"; break;
   case       2072: c = "MQRC_SYNCPOINT_NOT_AVAILABLE"; break;
   case       2075: c = "MQRC_TRIGGER_CONTROL_ERROR"; break;
   case       2076: c = "MQRC_TRIGGER_DEPTH_ERROR"; break;
   case       2077: c = "MQRC_TRIGGER_MSG_PRIORITY_ERR"; break;
   case       2078: c = "MQRC_TRIGGER_TYPE_ERROR"; break;
   case       2079: c = "MQRC_TRUNCATED_MSG_ACCEPTED"; break;
   case       2080: c = "MQRC_TRUNCATED_MSG_FAILED"; break;
   case       2082: c = "MQRC_UNKNOWN_ALIAS_BASE_Q"; break;
   case       2085: c = "MQRC_UNKNOWN_OBJECT_NAME"; break;
   case       2086: c = "MQRC_UNKNOWN_OBJECT_Q_MGR"; break;
   case       2087: c = "MQRC_UNKNOWN_REMOTE_Q_MGR"; break;
   case       2090: c = "MQRC_WAIT_INTERVAL_ERROR"; break;
   case       2091: c = "MQRC_XMIT_Q_TYPE_ERROR"; break;
   case       2092: c = "MQRC_XMIT_Q_USAGE_ERROR"; break;
   case       2093: c = "MQRC_NOT_OPEN_FOR_PASS_ALL"; break;
   case       2094: c = "MQRC_NOT_OPEN_FOR_PASS_IDENT"; break;
   case       2095: c = "MQRC_NOT_OPEN_FOR_SET_ALL"; break;
   case       2096: c = "MQRC_NOT_OPEN_FOR_SET_IDENT"; break;
   case       2097: c = "MQRC_CONTEXT_HANDLE_ERROR"; break;
   case       2098: c = "MQRC_CONTEXT_NOT_AVAILABLE"; break;
   case       2099: c = "MQRC_SIGNAL1_ERROR"; break;
   case       2100: c = "MQRC_OBJECT_ALREADY_EXISTS"; break;
   case       2101: c = "MQRC_OBJECT_DAMAGED"; break;
   case       2102: c = "MQRC_RESOURCE_PROBLEM"; break;
   case       2103: c = "MQRC_ANOTHER_Q_MGR_CONNECTED"; break;
   case       2104: c = "MQRC_UNKNOWN_REPORT_OPTION"; break;
   case       2105: c = "MQRC_STORAGE_CLASS_ERROR"; break;
   case       2106: c = "MQRC_COD_NOT_VALID_FOR_XCF_Q"; break;
   case       2107: c = "MQRC_XWAIT_CANCELED"; break;
   case       2108: c = "MQRC_XWAIT_ERROR"; break;
   case       2109: c = "MQRC_SUPPRESSED_BY_EXIT"; break;
   case       2110: c = "MQRC_FORMAT_ERROR"; break;
   case       2111: c = "MQRC_SOURCE_CCSID_ERROR"; break;
   case       2112: c = "MQRC_SOURCE_INTEGER_ENC_ERROR"; break;
   case       2113: c = "MQRC_SOURCE_DECIMAL_ENC_ERROR"; break;
   case       2114: c = "MQRC_SOURCE_FLOAT_ENC_ERROR"; break;
   case       2115: c = "MQRC_TARGET_CCSID_ERROR"; break;
   case       2116: c = "MQRC_TARGET_INTEGER_ENC_ERROR"; break;
   case       2117: c = "MQRC_TARGET_DECIMAL_ENC_ERROR"; break;
   case       2118: c = "MQRC_TARGET_FLOAT_ENC_ERROR"; break;
   case       2119: c = "MQRC_NOT_CONVERTED"; break;
   case       2121: c = "MQRC_NO_EXTERNAL_PARTICIPANTS"; break;
   case       2122: c = "MQRC_PARTICIPANT_NOT_AVAILABLE"; break;
   case       2123: c = "MQRC_OUTCOME_MIXED"; break;
   case       2124: c = "MQRC_OUTCOME_PENDING"; break;
   case       2125: c = "MQRC_BRIDGE_STARTED"; break;
   case       2126: c = "MQRC_BRIDGE_STOPPED"; break;
   case       2127: c = "MQRC_ADAPTER_STORAGE_SHORTAGE"; break;
   case       2128: c = "MQRC_UOW_IN_PROGRESS"; break;
   case       2129: c = "MQRC_ADAPTER_CONN_LOAD_ERROR"; break;
   case       2130: c = "MQRC_ADAPTER_SERV_LOAD_ERROR"; break;
   case       2131: c = "MQRC_ADAPTER_DEFS_ERROR"; break;
   case       2132: c = "MQRC_ADAPTER_DEFS_LOAD_ERROR"; break;
   case       2133: c = "MQRC_ADAPTER_CONV_LOAD_ERROR"; break;
   case       2134: c = "MQRC_BO_ERROR"; break;
   case       2135: c = "MQRC_DH_ERROR"; break;
   case       2136: c = "MQRC_MULTIPLE_REASONS"; break;
   case       2137: c = "MQRC_OPEN_FAILED"; break;
   case       2138: c = "MQRC_ADAPTER_DISC_LOAD_ERROR"; break;
   case       2139: c = "MQRC_CNO_ERROR"; break;
   case       2140: c = "MQRC_CICS_WAIT_FAILED"; break;
   case       2141: c = "MQRC_DLH_ERROR"; break;
   case       2142: c = "MQRC_HEADER_ERROR"; break;
   case       2143: c = "MQRC_SOURCE_LENGTH_ERROR"; break;
   case       2144: c = "MQRC_TARGET_LENGTH_ERROR"; break;
   case       2145: c = "MQRC_SOURCE_BUFFER_ERROR"; break;
   case       2146: c = "MQRC_TARGET_BUFFER_ERROR"; break;
   case       2148: c = "MQRC_IIH_ERROR"; break;
   case       2149: c = "MQRC_PCF_ERROR"; break;
   case       2150: c = "MQRC_DBCS_ERROR"; break;
   case       2152: c = "MQRC_OBJECT_NAME_ERROR"; break;
   case       2153: c = "MQRC_OBJECT_Q_MGR_NAME_ERROR"; break;
   case       2154: c = "MQRC_RECS_PRESENT_ERROR"; break;
   case       2155: c = "MQRC_OBJECT_RECORDS_ERROR"; break;
   case       2156: c = "MQRC_RESPONSE_RECORDS_ERROR"; break;
   case       2157: c = "MQRC_ASID_MISMATCH"; break;
   case       2158: c = "MQRC_PMO_RECORD_FLAGS_ERROR"; break;
   case       2159: c = "MQRC_PUT_MSG_RECORDS_ERROR"; break;
   case       2160: c = "MQRC_CONN_ID_IN_USE"; break;
   case       2161: c = "MQRC_Q_MGR_QUIESCING"; break;
   case       2162: c = "MQRC_Q_MGR_STOPPING"; break;
   case       2163: c = "MQRC_DUPLICATE_RECOV_COORD"; break;
   case       2173: c = "MQRC_PMO_ERROR"; break;
   case       2182: c = "MQRC_API_EXIT_NOT_FOUND"; break;
   case       2183: c = "MQRC_API_EXIT_LOAD_ERROR"; break;
   case       2184: c = "MQRC_REMOTE_Q_NAME_ERROR"; break;
   case       2185: c = "MQRC_INCONSISTENT_PERSISTENCE"; break;
   case       2186: c = "MQRC_GMO_ERROR"; break;
   case       2187: c = "MQRC_CICS_BRIDGE_RESTRICTION"; break;
   case       2188: c = "MQRC_STOPPED_BY_CLUSTER_EXIT"; break;
   case       2189: c = "MQRC_CLUSTER_RESOLUTION_ERROR"; break;
   case       2190: c = "MQRC_CONVERTED_STRING_TOO_BIG"; break;
   case       2191: c = "MQRC_TMC_ERROR"; break;
   case       2192: c = "MQRC_STORAGE_MEDIUM_FULL"; break;
   case       2193: c = "MQRC_PAGESET_ERROR"; break;
   case       2194: c = "MQRC_NAME_NOT_VALID_FOR_TYPE"; break;
   case       2195: c = "MQRC_UNEXPECTED_ERROR"; break;
   case       2196: c = "MQRC_UNKNOWN_XMIT_Q"; break;
   case       2197: c = "MQRC_UNKNOWN_DEF_XMIT_Q"; break;
   case       2198: c = "MQRC_DEF_XMIT_Q_TYPE_ERROR"; break;
   case       2199: c = "MQRC_DEF_XMIT_Q_USAGE_ERROR"; break;
   case       2200: c = "MQRC_MSG_MARKED_BROWSE_CO_OP"; break;
   case       2201: c = "MQRC_NAME_IN_USE"; break;
   case       2202: c = "MQRC_CONNECTION_QUIESCING"; break;
   case       2203: c = "MQRC_CONNECTION_STOPPING"; break;
   case       2204: c = "MQRC_ADAPTER_NOT_AVAILABLE"; break;
   case       2206: c = "MQRC_MSG_ID_ERROR"; break;
   case       2207: c = "MQRC_CORREL_ID_ERROR"; break;
   case       2208: c = "MQRC_FILE_SYSTEM_ERROR"; break;
   case       2209: c = "MQRC_NO_MSG_LOCKED"; break;
   case       2210: c = "MQRC_SOAP_DOTNET_ERROR"; break;
   case       2211: c = "MQRC_SOAP_AXIS_ERROR"; break;
   case       2212: c = "MQRC_SOAP_URL_ERROR"; break;
   case       2216: c = "MQRC_FILE_NOT_AUDITED"; break;
   case       2217: c = "MQRC_CONNECTION_NOT_AUTHORIZED"; break;
   case       2218: c = "MQRC_MSG_TOO_BIG_FOR_CHANNEL"; break;
   case       2219: c = "MQRC_CALL_IN_PROGRESS"; break;
   case       2220: c = "MQRC_RMH_ERROR"; break;
   case       2222: c = "MQRC_Q_MGR_ACTIVE"; break;
   case       2223: c = "MQRC_Q_MGR_NOT_ACTIVE"; break;
   case       2224: c = "MQRC_Q_DEPTH_HIGH"; break;
   case       2225: c = "MQRC_Q_DEPTH_LOW"; break;
   case       2226: c = "MQRC_Q_SERVICE_INTERVAL_HIGH"; break;
   case       2227: c = "MQRC_Q_SERVICE_INTERVAL_OK"; break;
   case       2228: c = "MQRC_RFH_HEADER_FIELD_ERROR"; break;
   case       2229: c = "MQRC_RAS_PROPERTY_ERROR"; break;
   case       2232: c = "MQRC_UNIT_OF_WORK_NOT_STARTED"; break;
   case       2233: c = "MQRC_CHANNEL_AUTO_DEF_OK"; break;
   case       2234: c = "MQRC_CHANNEL_AUTO_DEF_ERROR"; break;
   case       2235: c = "MQRC_CFH_ERROR"; break;
   case       2236: c = "MQRC_CFIL_ERROR"; break;
   case       2237: c = "MQRC_CFIN_ERROR"; break;
   case       2238: c = "MQRC_CFSL_ERROR"; break;
   case       2239: c = "MQRC_CFST_ERROR"; break;
   case       2241: c = "MQRC_INCOMPLETE_GROUP"; break;
   case       2242: c = "MQRC_INCOMPLETE_MSG"; break;
   case       2243: c = "MQRC_INCONSISTENT_CCSIDS"; break;
   case       2244: c = "MQRC_INCONSISTENT_ENCODINGS"; break;
   case       2245: c = "MQRC_INCONSISTENT_UOW"; break;
   case       2246: c = "MQRC_INVALID_MSG_UNDER_CURSOR"; break;
   case       2247: c = "MQRC_MATCH_OPTIONS_ERROR"; break;
   case       2248: c = "MQRC_MDE_ERROR"; break;
   case       2249: c = "MQRC_MSG_FLAGS_ERROR"; break;
   case       2250: c = "MQRC_MSG_SEQ_NUMBER_ERROR"; break;
   case       2251: c = "MQRC_OFFSET_ERROR"; break;
   case       2252: c = "MQRC_ORIGINAL_LENGTH_ERROR"; break;
   case       2253: c = "MQRC_SEGMENT_LENGTH_ZERO"; break;
   case       2255: c = "MQRC_UOW_NOT_AVAILABLE"; break;
   case       2256: c = "MQRC_WRONG_GMO_VERSION"; break;
   case       2257: c = "MQRC_WRONG_MD_VERSION"; break;
   case       2258: c = "MQRC_GROUP_ID_ERROR"; break;
   case       2259: c = "MQRC_INCONSISTENT_BROWSE"; break;
   case       2260: c = "MQRC_XQH_ERROR"; break;
   case       2261: c = "MQRC_SRC_ENV_ERROR"; break;
   case       2262: c = "MQRC_SRC_NAME_ERROR"; break;
   case       2263: c = "MQRC_DEST_ENV_ERROR"; break;
   case       2264: c = "MQRC_DEST_NAME_ERROR"; break;
   case       2265: c = "MQRC_TM_ERROR"; break;
   case       2266: c = "MQRC_CLUSTER_EXIT_ERROR"; break;
   case       2267: c = "MQRC_CLUSTER_EXIT_LOAD_ERROR"; break;
   case       2268: c = "MQRC_CLUSTER_PUT_INHIBITED"; break;
   case       2269: c = "MQRC_CLUSTER_RESOURCE_ERROR"; break;
   case       2270: c = "MQRC_NO_DESTINATIONS_AVAILABLE"; break;
   case       2271: c = "MQRC_CONN_TAG_IN_USE"; break;
   case       2272: c = "MQRC_PARTIALLY_CONVERTED"; break;
   case       2273: c = "MQRC_CONNECTION_ERROR"; break;
   case       2274: c = "MQRC_OPTION_ENVIRONMENT_ERROR"; break;
   case       2277: c = "MQRC_CD_ERROR"; break;
   case       2278: c = "MQRC_CLIENT_CONN_ERROR"; break;
   case       2279: c = "MQRC_CHANNEL_STOPPED_BY_USER"; break;
   case       2280: c = "MQRC_HCONFIG_ERROR"; break;
   case       2281: c = "MQRC_FUNCTION_ERROR"; break;
   case       2282: c = "MQRC_CHANNEL_STARTED"; break;
   case       2283: c = "MQRC_CHANNEL_STOPPED"; break;
   case       2284: c = "MQRC_CHANNEL_CONV_ERROR"; break;
   case       2285: c = "MQRC_SERVICE_NOT_AVAILABLE"; break;
   case       2286: c = "MQRC_INITIALIZATION_FAILED"; break;
   case       2287: c = "MQRC_TERMINATION_FAILED"; break;
   case       2288: c = "MQRC_UNKNOWN_Q_NAME"; break;
   case       2289: c = "MQRC_SERVICE_ERROR"; break;
   case       2290: c = "MQRC_Q_ALREADY_EXISTS"; break;
   case       2291: c = "MQRC_USER_ID_NOT_AVAILABLE"; break;
   case       2292: c = "MQRC_UNKNOWN_ENTITY"; break;
   case       2293: c = "MQRC_UNKNOWN_AUTH_ENTITY"; break;
   case       2294: c = "MQRC_UNKNOWN_REF_OBJECT"; break;
   case       2295: c = "MQRC_CHANNEL_ACTIVATED"; break;
   case       2296: c = "MQRC_CHANNEL_NOT_ACTIVATED"; break;
   case       2297: c = "MQRC_UOW_CANCELED"; break;
   case       2298: c = "MQRC_FUNCTION_NOT_SUPPORTED"; break;
   case       2299: c = "MQRC_SELECTOR_TYPE_ERROR"; break;
   case       2300: c = "MQRC_COMMAND_TYPE_ERROR"; break;
   case       2301: c = "MQRC_MULTIPLE_INSTANCE_ERROR"; break;
   case       2302: c = "MQRC_SYSTEM_ITEM_NOT_ALTERABLE"; break;
   case       2303: c = "MQRC_BAG_CONVERSION_ERROR"; break;
   case       2304: c = "MQRC_SELECTOR_OUT_OF_RANGE"; break;
   case       2305: c = "MQRC_SELECTOR_NOT_UNIQUE"; break;
   case       2306: c = "MQRC_INDEX_NOT_PRESENT"; break;
   case       2307: c = "MQRC_STRING_ERROR"; break;
   case       2308: c = "MQRC_ENCODING_NOT_SUPPORTED"; break;
   case       2309: c = "MQRC_SELECTOR_NOT_PRESENT"; break;
   case       2310: c = "MQRC_OUT_SELECTOR_ERROR"; break;
   case       2311: c = "MQRC_STRING_TRUNCATED"; break;
   case       2312: c = "MQRC_SELECTOR_WRONG_TYPE"; break;
   case       2313: c = "MQRC_INCONSISTENT_ITEM_TYPE"; break;
   case       2314: c = "MQRC_INDEX_ERROR"; break;
   case       2315: c = "MQRC_SYSTEM_BAG_NOT_ALTERABLE"; break;
   case       2316: c = "MQRC_ITEM_COUNT_ERROR"; break;
   case       2317: c = "MQRC_FORMAT_NOT_SUPPORTED"; break;
   case       2318: c = "MQRC_SELECTOR_NOT_SUPPORTED"; break;
   case       2319: c = "MQRC_ITEM_VALUE_ERROR"; break;
   case       2320: c = "MQRC_HBAG_ERROR"; break;
   case       2321: c = "MQRC_PARAMETER_MISSING"; break;
   case       2322: c = "MQRC_CMD_SERVER_NOT_AVAILABLE"; break;
   case       2323: c = "MQRC_STRING_LENGTH_ERROR"; break;
   case       2324: c = "MQRC_INQUIRY_COMMAND_ERROR"; break;
   case       2325: c = "MQRC_NESTED_BAG_NOT_SUPPORTED"; break;
   case       2326: c = "MQRC_BAG_WRONG_TYPE"; break;
   case       2327: c = "MQRC_ITEM_TYPE_ERROR"; break;
   case       2328: c = "MQRC_SYSTEM_BAG_NOT_DELETABLE"; break;
   case       2329: c = "MQRC_SYSTEM_ITEM_NOT_DELETABLE"; break;
   case       2330: c = "MQRC_CODED_CHAR_SET_ID_ERROR"; break;
   case       2331: c = "MQRC_MSG_TOKEN_ERROR"; break;
   case       2332: c = "MQRC_MISSING_WIH"; break;
   case       2333: c = "MQRC_WIH_ERROR"; break;
   case       2334: c = "MQRC_RFH_ERROR"; break;
   case       2335: c = "MQRC_RFH_STRING_ERROR"; break;
   case       2336: c = "MQRC_RFH_COMMAND_ERROR"; break;
   case       2337: c = "MQRC_RFH_PARM_ERROR"; break;
   case       2338: c = "MQRC_RFH_DUPLICATE_PARM"; break;
   case       2339: c = "MQRC_RFH_PARM_MISSING"; break;
   case       2340: c = "MQRC_CHAR_CONVERSION_ERROR"; break;
   case       2341: c = "MQRC_UCS2_CONVERSION_ERROR"; break;
   case       2342: c = "MQRC_DB2_NOT_AVAILABLE"; break;
   case       2343: c = "MQRC_OBJECT_NOT_UNIQUE"; break;
   case       2344: c = "MQRC_CONN_TAG_NOT_RELEASED"; break;
   case       2345: c = "MQRC_CF_NOT_AVAILABLE"; break;
   case       2346: c = "MQRC_CF_STRUC_IN_USE"; break;
   case       2347: c = "MQRC_CF_STRUC_LIST_HDR_IN_USE"; break;
   case       2348: c = "MQRC_CF_STRUC_AUTH_FAILED"; break;
   case       2349: c = "MQRC_CF_STRUC_ERROR"; break;
   case       2350: c = "MQRC_CONN_TAG_NOT_USABLE"; break;
   case       2351: c = "MQRC_GLOBAL_UOW_CONFLICT"; break;
   case       2352: c = "MQRC_LOCAL_UOW_CONFLICT"; break;
   case       2353: c = "MQRC_HANDLE_IN_USE_FOR_UOW"; break;
   case       2354: c = "MQRC_UOW_ENLISTMENT_ERROR"; break;
   case       2355: c = "MQRC_UOW_MIX_NOT_SUPPORTED"; break;
   case       2356: c = "MQRC_WXP_ERROR"; break;
   case       2357: c = "MQRC_CURRENT_RECORD_ERROR"; break;
   case       2358: c = "MQRC_NEXT_OFFSET_ERROR"; break;
   case       2359: c = "MQRC_NO_RECORD_AVAILABLE"; break;
   case       2360: c = "MQRC_OBJECT_LEVEL_INCOMPATIBLE"; break;
   case       2361: c = "MQRC_NEXT_RECORD_ERROR"; break;
   case       2362: c = "MQRC_BACKOUT_THRESHOLD_REACHED"; break;
   case       2363: c = "MQRC_MSG_NOT_MATCHED"; break;
   case       2364: c = "MQRC_JMS_FORMAT_ERROR"; break;
   case       2365: c = "MQRC_SEGMENTS_NOT_SUPPORTED"; break;
   case       2366: c = "MQRC_WRONG_CF_LEVEL"; break;
   case       2367: c = "MQRC_CONFIG_CREATE_OBJECT"; break;
   case       2368: c = "MQRC_CONFIG_CHANGE_OBJECT"; break;
   case       2369: c = "MQRC_CONFIG_DELETE_OBJECT"; break;
   case       2370: c = "MQRC_CONFIG_REFRESH_OBJECT"; break;
   case       2371: c = "MQRC_CHANNEL_SSL_ERROR"; break;
   case       2372: c = "MQRC_PARTICIPANT_NOT_DEFINED"; break;
   case       2373: c = "MQRC_CF_STRUC_FAILED"; break;
   case       2374: c = "MQRC_API_EXIT_ERROR"; break;
   case       2375: c = "MQRC_API_EXIT_INIT_ERROR"; break;
   case       2376: c = "MQRC_API_EXIT_TERM_ERROR"; break;
   case       2377: c = "MQRC_EXIT_REASON_ERROR"; break;
   case       2378: c = "MQRC_RESERVED_VALUE_ERROR"; break;
   case       2379: c = "MQRC_NO_DATA_AVAILABLE"; break;
   case       2380: c = "MQRC_SCO_ERROR"; break;
   case       2381: c = "MQRC_KEY_REPOSITORY_ERROR"; break;
   case       2382: c = "MQRC_CRYPTO_HARDWARE_ERROR"; break;
   case       2383: c = "MQRC_AUTH_INFO_REC_COUNT_ERROR"; break;
   case       2384: c = "MQRC_AUTH_INFO_REC_ERROR"; break;
   case       2385: c = "MQRC_AIR_ERROR"; break;
   case       2386: c = "MQRC_AUTH_INFO_TYPE_ERROR"; break;
   case       2387: c = "MQRC_AUTH_INFO_CONN_NAME_ERROR"; break;
   case       2388: c = "MQRC_LDAP_USER_NAME_ERROR"; break;
   case       2389: c = "MQRC_LDAP_USER_NAME_LENGTH_ERR"; break;
   case       2390: c = "MQRC_LDAP_PASSWORD_ERROR"; break;
   case       2391: c = "MQRC_SSL_ALREADY_INITIALIZED"; break;
   case       2392: c = "MQRC_SSL_CONFIG_ERROR"; break;
   case       2393: c = "MQRC_SSL_INITIALIZATION_ERROR"; break;
   case       2394: c = "MQRC_Q_INDEX_TYPE_ERROR"; break;
   case       2395: c = "MQRC_CFBS_ERROR"; break;
   case       2396: c = "MQRC_SSL_NOT_ALLOWED"; break;
   case       2397: c = "MQRC_JSSE_ERROR"; break;
   case       2398: c = "MQRC_SSL_PEER_NAME_MISMATCH"; break;
   case       2399: c = "MQRC_SSL_PEER_NAME_ERROR"; break;
   case       2400: c = "MQRC_UNSUPPORTED_CIPHER_SUITE"; break;
   case       2401: c = "MQRC_SSL_CERTIFICATE_REVOKED"; break;
   case       2402: c = "MQRC_SSL_CERT_STORE_ERROR"; break;
   case       2406: c = "MQRC_CLIENT_EXIT_LOAD_ERROR"; break;
   case       2407: c = "MQRC_CLIENT_EXIT_ERROR"; break;
   case       2408: c = "MQRC_UOW_COMMITTED"; break;
   case       2409: c = "MQRC_SSL_KEY_RESET_ERROR"; break;
   case       2410: c = "MQRC_UNKNOWN_COMPONENT_NAME"; break;
   case       2411: c = "MQRC_LOGGER_STATUS"; break;
   case       2412: c = "MQRC_COMMAND_MQSC"; break;
   case       2413: c = "MQRC_COMMAND_PCF"; break;
   case       2414: c = "MQRC_CFIF_ERROR"; break;
   case       2415: c = "MQRC_CFSF_ERROR"; break;
   case       2416: c = "MQRC_CFGR_ERROR"; break;
   case       2417: c = "MQRC_MSG_NOT_ALLOWED_IN_GROUP"; break;
   case       2418: c = "MQRC_FILTER_OPERATOR_ERROR"; break;
   case       2419: c = "MQRC_NESTED_SELECTOR_ERROR"; break;
   case       2420: c = "MQRC_EPH_ERROR"; break;
   case       2421: c = "MQRC_RFH_FORMAT_ERROR"; break;
   case       2422: c = "MQRC_CFBF_ERROR"; break;
   case       2423: c = "MQRC_CLIENT_CHANNEL_CONFLICT"; break;
   case       2424: c = "MQRC_SD_ERROR"; break;
   case       2425: c = "MQRC_TOPIC_STRING_ERROR"; break;
   case       2426: c = "MQRC_STS_ERROR"; break;
   case       2428: c = "MQRC_NO_SUBSCRIPTION"; break;
   case       2429: c = "MQRC_SUBSCRIPTION_IN_USE"; break;
   case       2430: c = "MQRC_STAT_TYPE_ERROR"; break;
   case       2431: c = "MQRC_SUB_USER_DATA_ERROR"; break;
   case       2432: c = "MQRC_SUB_ALREADY_EXISTS"; break;
   case       2434: c = "MQRC_IDENTITY_MISMATCH"; break;
   case       2435: c = "MQRC_ALTER_SUB_ERROR"; break;
   case       2436: c = "MQRC_DURABILITY_NOT_ALLOWED"; break;
   case       2437: c = "MQRC_NO_RETAINED_MSG"; break;
   case       2438: c = "MQRC_SRO_ERROR"; break;
   case       2440: c = "MQRC_SUB_NAME_ERROR"; break;
   case       2441: c = "MQRC_OBJECT_STRING_ERROR"; break;
   case       2442: c = "MQRC_PROPERTY_NAME_ERROR"; break;
   case       2443: c = "MQRC_SEGMENTATION_NOT_ALLOWED"; break;
   case       2444: c = "MQRC_CBD_ERROR"; break;
   case       2445: c = "MQRC_CTLO_ERROR"; break;
   case       2446: c = "MQRC_NO_CALLBACKS_ACTIVE"; break;
   case       2448: c = "MQRC_CALLBACK_NOT_REGISTERED"; break;
   case       2457: c = "MQRC_OPTIONS_CHANGED"; break;
   case       2458: c = "MQRC_READ_AHEAD_MSGS"; break;
   case       2459: c = "MQRC_SELECTOR_SYNTAX_ERROR"; break;
   case       2460: c = "MQRC_HMSG_ERROR"; break;
   case       2461: c = "MQRC_CMHO_ERROR"; break;
   case       2462: c = "MQRC_DMHO_ERROR"; break;
   case       2463: c = "MQRC_SMPO_ERROR"; break;
   case       2464: c = "MQRC_IMPO_ERROR"; break;
   case       2465: c = "MQRC_PROPERTY_NAME_TOO_BIG"; break;
   case       2466: c = "MQRC_PROP_VALUE_NOT_CONVERTED"; break;
   case       2467: c = "MQRC_PROP_TYPE_NOT_SUPPORTED"; break;
   case       2469: c = "MQRC_PROPERTY_VALUE_TOO_BIG"; break;
   case       2470: c = "MQRC_PROP_CONV_NOT_SUPPORTED"; break;
   case       2471: c = "MQRC_PROPERTY_NOT_AVAILABLE"; break;
   case       2472: c = "MQRC_PROP_NUMBER_FORMAT_ERROR"; break;
   case       2473: c = "MQRC_PROPERTY_TYPE_ERROR"; break;
   case       2478: c = "MQRC_PROPERTIES_TOO_BIG"; break;
   case       2479: c = "MQRC_PUT_NOT_RETAINED"; break;
   case       2480: c = "MQRC_ALIAS_TARGTYPE_CHANGED"; break;
   case       2481: c = "MQRC_DMPO_ERROR"; break;
   case       2482: c = "MQRC_PD_ERROR"; break;
   case       2483: c = "MQRC_CALLBACK_TYPE_ERROR"; break;
   case       2484: c = "MQRC_CBD_OPTIONS_ERROR"; break;
   case       2485: c = "MQRC_MAX_MSG_LENGTH_ERROR"; break;
   case       2486: c = "MQRC_CALLBACK_ROUTINE_ERROR"; break;
   case       2487: c = "MQRC_CALLBACK_LINK_ERROR"; break;
   case       2488: c = "MQRC_OPERATION_ERROR"; break;
   case       2489: c = "MQRC_BMHO_ERROR"; break;
   case       2490: c = "MQRC_UNSUPPORTED_PROPERTY"; break;
   case       2492: c = "MQRC_PROP_NAME_NOT_CONVERTED"; break;
   case       2494: c = "MQRC_GET_ENABLED"; break;
   case       2495: c = "MQRC_MODULE_NOT_FOUND"; break;
   case       2496: c = "MQRC_MODULE_INVALID"; break;
   case       2497: c = "MQRC_MODULE_ENTRY_NOT_FOUND"; break;
   case       2498: c = "MQRC_MIXED_CONTENT_NOT_ALLOWED"; break;
   case       2499: c = "MQRC_MSG_HANDLE_IN_USE"; break;
   case       2500: c = "MQRC_HCONN_ASYNC_ACTIVE"; break;
   case       2501: c = "MQRC_MHBO_ERROR"; break;
   case       2502: c = "MQRC_PUBLICATION_FAILURE"; break;
   case       2503: c = "MQRC_SUB_INHIBITED"; break;
   case       2504: c = "MQRC_SELECTOR_ALWAYS_FALSE"; break;
   case       2507: c = "MQRC_XEPO_ERROR"; break;
   case       2509: c = "MQRC_DURABILITY_NOT_ALTERABLE"; break;
   case       2510: c = "MQRC_TOPIC_NOT_ALTERABLE"; break;
   case       2512: c = "MQRC_SUBLEVEL_NOT_ALTERABLE"; break;
   case       2513: c = "MQRC_PROPERTY_NAME_LENGTH_ERR"; break;
   case       2514: c = "MQRC_DUPLICATE_GROUP_SUB"; break;
   case       2515: c = "MQRC_GROUPING_NOT_ALTERABLE"; break;
   case       2516: c = "MQRC_SELECTOR_INVALID_FOR_TYPE"; break;
   case       2517: c = "MQRC_HOBJ_QUIESCED"; break;
   case       2518: c = "MQRC_HOBJ_QUIESCED_NO_MSGS"; break;
   case       2519: c = "MQRC_SELECTION_STRING_ERROR"; break;
   case       2520: c = "MQRC_RES_OBJECT_STRING_ERROR"; break;
   case       2521: c = "MQRC_CONNECTION_SUSPENDED"; break;
   case       2522: c = "MQRC_INVALID_DESTINATION"; break;
   case       2523: c = "MQRC_INVALID_SUBSCRIPTION"; break;
   case       2524: c = "MQRC_SELECTOR_NOT_ALTERABLE"; break;
   case       2525: c = "MQRC_RETAINED_MSG_Q_ERROR"; break;
   case       2526: c = "MQRC_RETAINED_NOT_DELIVERED"; break;
   case       2527: c = "MQRC_RFH_RESTRICTED_FORMAT_ERR"; break;
   case       2528: c = "MQRC_CONNECTION_STOPPED"; break;
   case       2529: c = "MQRC_ASYNC_UOW_CONFLICT"; break;
   case       2530: c = "MQRC_ASYNC_XA_CONFLICT"; break;
   case       2531: c = "MQRC_PUBSUB_INHIBITED"; break;
   case       2532: c = "MQRC_MSG_HANDLE_COPY_FAILURE"; break;
   case       2533: c = "MQRC_DEST_CLASS_NOT_ALTERABLE"; break;
   case       2534: c = "MQRC_OPERATION_NOT_ALLOWED"; break;
   case       2535: c = "MQRC_ACTION_ERROR"; break;
   case       2537: c = "MQRC_CHANNEL_NOT_AVAILABLE"; break;
   case       2538: c = "MQRC_HOST_NOT_AVAILABLE"; break;
   case       2539: c = "MQRC_CHANNEL_CONFIG_ERROR"; break;
   case       2540: c = "MQRC_UNKNOWN_CHANNEL_NAME"; break;
   case       2541: c = "MQRC_LOOPING_PUBLICATION"; break;
   case       2542: c = "MQRC_ALREADY_JOINED"; break;
   case       2543: c = "MQRC_STANDBY_Q_MGR"; break;
   case       2544: c = "MQRC_RECONNECTING"; break;
   case       2545: c = "MQRC_RECONNECTED"; break;
   case       2546: c = "MQRC_RECONNECT_QMID_MISMATCH"; break;
   case       2547: c = "MQRC_RECONNECT_INCOMPATIBLE"; break;
   case       2548: c = "MQRC_RECONNECT_FAILED"; break;
   case       2549: c = "MQRC_CALL_INTERRUPTED"; break;
   case       2550: c = "MQRC_NO_SUBS_MATCHED"; break;
   case       2551: c = "MQRC_SELECTION_NOT_AVAILABLE"; break;
   case       2552: c = "MQRC_CHANNEL_SSL_WARNING"; break;
   case       2553: c = "MQRC_OCSP_URL_ERROR"; break;
   case       2554: c = "MQRC_CONTENT_ERROR"; break;
   case       2555: c = "MQRC_RECONNECT_Q_MGR_REQD"; break;
   case       2556: c = "MQRC_RECONNECT_TIMED_OUT"; break;
   case       2557: c = "MQRC_PUBLISH_EXIT_ERROR"; break;
   case       2558: c = "MQRC_COMMINFO_ERROR"; break;
   case       2559: c = "MQRC_DEF_SYNCPOINT_INHIBITED"; break;
   case       2560: c = "MQRC_MULTICAST_ONLY"; break;
   case       2561: c = "MQRC_DATA_SET_NOT_AVAILABLE"; break;
   case       2562: c = "MQRC_GROUPING_NOT_ALLOWED"; break;
   case       2563: c = "MQRC_GROUP_ADDRESS_ERROR"; break;
   case       2564: c = "MQRC_MULTICAST_CONFIG_ERROR"; break;
   case       2565: c = "MQRC_MULTICAST_INTERFACE_ERROR"; break;
   case       2566: c = "MQRC_MULTICAST_SEND_ERROR"; break;
   case       2567: c = "MQRC_MULTICAST_INTERNAL_ERROR"; break;
   case       2568: c = "MQRC_CONNECTION_NOT_AVAILABLE"; break;
   case       2569: c = "MQRC_SYNCPOINT_NOT_ALLOWED"; break;
   case       2570: c = "MQRC_SSL_ALT_PROVIDER_REQUIRED"; break;
   case       2571: c = "MQRC_MCAST_PUB_STATUS"; break;
   case       2572: c = "MQRC_MCAST_SUB_STATUS"; break;
   case       2573: c = "MQRC_PRECONN_EXIT_LOAD_ERROR"; break;
   case       2574: c = "MQRC_PRECONN_EXIT_NOT_FOUND"; break;
   case       2575: c = "MQRC_PRECONN_EXIT_ERROR"; break;
   case       2576: c = "MQRC_CD_ARRAY_ERROR"; break;
   case       2577: c = "MQRC_CHANNEL_BLOCKED"; break;
   case       2578: c = "MQRC_CHANNEL_BLOCKED_WARNING"; break;
   case       2579: c = "MQRC_SUBSCRIPTION_CREATE"; break;
   case       2580: c = "MQRC_SUBSCRIPTION_DELETE"; break;
   case       2581: c = "MQRC_SUBSCRIPTION_CHANGE"; break;
   case       2582: c = "MQRC_SUBSCRIPTION_REFRESH"; break;
   case       2583: c = "MQRC_INSTALLATION_MISMATCH"; break;
   case       2584: c = "MQRC_NOT_PRIVILEGED"; break;
   case       2586: c = "MQRC_PROPERTIES_DISABLED"; break;
   case       2587: c = "MQRC_HMSG_NOT_AVAILABLE"; break;
   case       2588: c = "MQRC_EXIT_PROPS_NOT_SUPPORTED"; break;
   case       2589: c = "MQRC_INSTALLATION_MISSING"; break;
   case       2590: c = "MQRC_FASTPATH_NOT_AVAILABLE"; break;
   case       2591: c = "MQRC_CIPHER_SPEC_NOT_SUITE_B"; break;
   case       2592: c = "MQRC_SUITE_B_ERROR"; break;
   case       2593: c = "MQRC_CERT_VAL_POLICY_ERROR"; break;
   case       2594: c = "MQRC_PASSWORD_PROTECTION_ERROR"; break;
   case       2595: c = "MQRC_CSP_ERROR"; break;
   case       2596: c = "MQRC_CERT_LABEL_NOT_ALLOWED"; break;
   case       2598: c = "MQRC_ADMIN_TOPIC_STRING_ERROR"; break;
   case       2599: c = "MQRC_AMQP_NOT_AVAILABLE"; break;
   case       2600: c = "MQRC_CCDT_URL_ERROR"; break;
   case       6100: c = "MQRC_REOPEN_EXCL_INPUT_ERROR"; break;
   case       6101: c = "MQRC_REOPEN_INQUIRE_ERROR"; break;
   case       6102: c = "MQRC_REOPEN_SAVED_CONTEXT_ERR"; break;
   case       6103: c = "MQRC_REOPEN_TEMPORARY_Q_ERROR"; break;
   case       6104: c = "MQRC_ATTRIBUTE_LOCKED"; break;
   case       6105: c = "MQRC_CURSOR_NOT_VALID"; break;
   case       6106: c = "MQRC_ENCODING_ERROR"; break;
   case       6107: c = "MQRC_STRUC_ID_ERROR"; break;
   case       6108: c = "MQRC_NULL_POINTER"; break;
   case       6109: c = "MQRC_NO_CONNECTION_REFERENCE"; break;
   case       6110: c = "MQRC_NO_BUFFER"; break;
   case       6111: c = "MQRC_BINARY_DATA_LENGTH_ERROR"; break;
   case       6112: c = "MQRC_BUFFER_NOT_AUTOMATIC"; break;
   case       6113: c = "MQRC_INSUFFICIENT_BUFFER"; break;
   case       6114: c = "MQRC_INSUFFICIENT_DATA"; break;
   case       6115: c = "MQRC_DATA_TRUNCATED"; break;
   case       6116: c = "MQRC_ZERO_LENGTH"; break;
   case       6117: c = "MQRC_NEGATIVE_LENGTH"; break;
   case       6118: c = "MQRC_NEGATIVE_OFFSET"; break;
   case       6119: c = "MQRC_INCONSISTENT_FORMAT"; break;
   case       6120: c = "MQRC_INCONSISTENT_OBJECT_STATE"; break;
   case       6121: c = "MQRC_CONTEXT_OBJECT_NOT_VALID"; break;
   case       6122: c = "MQRC_CONTEXT_OPEN_ERROR"; break;
   case       6124: c = "MQRC_NOT_CONNECTED"; break;
   case       6125: c = "MQRC_NOT_OPEN"; break;
   case       6126: c = "MQRC_DISTRIBUTION_LIST_EMPTY"; break;
   case       6127: c = "MQRC_INCONSISTENT_OPEN_OPTIONS"; break;
   case       6128: c = "MQRC_WRONG_VERSION"; break;
   case       6129: c = "MQRC_REFERENCE_ERROR"; break;
   case       6130: c = "MQRC_XR_NOT_AVAILABLE"; break;
   case      29440: c = "MQRC_SUB_JOIN_NOT_ALTERABLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRDNS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRDNS_ENABLED"; break;
   case          1: c = "MQRDNS_DISABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRD_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQRD_NO_RECONNECT"; break;
   case          0: c = "MQRD_NO_DELAY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQREADA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQREADA_NO"; break;
   case          1: c = "MQREADA_YES"; break;
   case          2: c = "MQREADA_DISABLED"; break;
   case          3: c = "MQREADA_INHIBITED"; break;
   case          4: c = "MQREADA_BACKLOG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRECAUTO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRECAUTO_NO"; break;
   case          1: c = "MQRECAUTO_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRECORDING_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRECORDING_DISABLED"; break;
   case          1: c = "MQRECORDING_Q"; break;
   case          2: c = "MQRECORDING_MSG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQREGO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQREGO_NONE"; break;
   case          1: c = "MQREGO_CORREL_ID_AS_IDENTITY"; break;
   case          2: c = "MQREGO_ANONYMOUS"; break;
   case          4: c = "MQREGO_LOCAL"; break;
   case          8: c = "MQREGO_DIRECT_REQUESTS"; break;
   case         16: c = "MQREGO_NEW_PUBLICATIONS_ONLY"; break;
   case         32: c = "MQREGO_PUBLISH_ON_REQUEST_ONLY"; break;
   case         64: c = "MQREGO_DEREGISTER_ALL"; break;
   case        128: c = "MQREGO_INCLUDE_STREAM_NAME"; break;
   case        256: c = "MQREGO_INFORM_IF_RETAINED"; break;
   case        512: c = "MQREGO_DUPLICATES_OK"; break;
   case       1024: c = "MQREGO_NON_PERSISTENT"; break;
   case       2048: c = "MQREGO_PERSISTENT"; break;
   case       4096: c = "MQREGO_PERSISTENT_AS_PUBLISH"; break;
   case       8192: c = "MQREGO_PERSISTENT_AS_Q"; break;
   case      16384: c = "MQREGO_ADD_NAME"; break;
   case      32768: c = "MQREGO_NO_ALTERATION"; break;
   case      65536: c = "MQREGO_FULL_RESPONSE"; break;
   case     131072: c = "MQREGO_JOIN_SHARED"; break;
   case     262144: c = "MQREGO_JOIN_EXCLUSIVE"; break;
   case     524288: c = "MQREGO_LEAVE_ONLY"; break;
   case    1048576: c = "MQREGO_VARIABLE_USER_ID"; break;
   case    2097152: c = "MQREGO_LOCKED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQREORG_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQREORG_DISABLED"; break;
   case          1: c = "MQREORG_ENABLED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRFH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case     -65536: c = "MQRFH_FLAGS_RESTRICTED_MASK"; break;
   case          0: c = "MQRFH_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQRL_UNDEFINED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQROUTE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case     -65536: c = "MQROUTE_DELIVER_REJ_UNSUP_MASK"; break;
   case          0: c = "MQROUTE_UNLIMITED_ACTIVITIES"; break;
   case          2: c = "MQROUTE_DETAIL_LOW"; break;
   case          8: c = "MQROUTE_DETAIL_MEDIUM"; break;
   case         32: c = "MQROUTE_DETAIL_HIGH"; break;
   case        256: c = "MQROUTE_FORWARD_ALL"; break;
   case        512: c = "MQROUTE_FORWARD_IF_SUPPORTED"; break;
   case       4096: c = "MQROUTE_DELIVER_YES"; break;
   case       8192: c = "MQROUTE_DELIVER_NO"; break;
   case      65539: c = "MQROUTE_ACCUMULATE_NONE"; break;
   case      65540: c = "MQROUTE_ACCUMULATE_IN_MSG"; break;
   case      65541: c = "MQROUTE_ACCUMULATE_AND_REPLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case -270532353: c = "MQRO_ACCEPT_UNSUP_MASK"; break;
   case          0: c = "MQRO_NONE"; break;
   case          1: c = "MQRO_PAN"; break;
   case          2: c = "MQRO_NAN"; break;
   case          4: c = "MQRO_ACTIVITY"; break;
   case         64: c = "MQRO_PASS_CORREL_ID"; break;
   case        128: c = "MQRO_PASS_MSG_ID"; break;
   case        256: c = "MQRO_COA"; break;
   case        768: c = "MQRO_COA_WITH_DATA"; break;
   case       1792: c = "MQRO_COA_WITH_FULL_DATA"; break;
   case       2048: c = "MQRO_COD"; break;
   case       6144: c = "MQRO_COD_WITH_DATA"; break;
   case      14336: c = "MQRO_COD_WITH_FULL_DATA"; break;
   case      16384: c = "MQRO_PASS_DISCARD_AND_EXPIRY"; break;
   case     261888: c = "MQRO_ACCEPT_UNSUP_IF_XMIT_MASK"; break;
   case    2097152: c = "MQRO_EXPIRATION"; break;
   case    6291456: c = "MQRO_EXPIRATION_WITH_DATA"; break;
   case   14680064: c = "MQRO_EXPIRATION_WITH_FULL_DATA"; break;
   case   16777216: c = "MQRO_EXCEPTION"; break;
   case   50331648: c = "MQRO_EXCEPTION_WITH_DATA"; break;
   case  117440512: c = "MQRO_EXCEPTION_WITH_FULL_DATA"; break;
   case  134217728: c = "MQRO_DISCARD_MSG"; break;
   case  270270464: c = "MQRO_REJECT_UNSUP_MASK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQRP_NO"; break;
   case          1: c = "MQRP_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRQ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQRQ_CONN_NOT_AUTHORIZED"; break;
   case          2: c = "MQRQ_OPEN_NOT_AUTHORIZED"; break;
   case          3: c = "MQRQ_CLOSE_NOT_AUTHORIZED"; break;
   case          4: c = "MQRQ_CMD_NOT_AUTHORIZED"; break;
   case          5: c = "MQRQ_Q_MGR_STOPPING"; break;
   case          6: c = "MQRQ_Q_MGR_QUIESCING"; break;
   case          7: c = "MQRQ_CHANNEL_STOPPED_OK"; break;
   case          8: c = "MQRQ_CHANNEL_STOPPED_ERROR"; break;
   case          9: c = "MQRQ_CHANNEL_STOPPED_RETRY"; break;
   case         10: c = "MQRQ_CHANNEL_STOPPED_DISABLED"; break;
   case         11: c = "MQRQ_BRIDGE_STOPPED_OK"; break;
   case         12: c = "MQRQ_BRIDGE_STOPPED_ERROR"; break;
   case         13: c = "MQRQ_SSL_HANDSHAKE_ERROR"; break;
   case         14: c = "MQRQ_SSL_CIPHER_SPEC_ERROR"; break;
   case         15: c = "MQRQ_SSL_CLIENT_AUTH_ERROR"; break;
   case         16: c = "MQRQ_SSL_PEER_NAME_ERROR"; break;
   case         17: c = "MQRQ_SUB_NOT_AUTHORIZED"; break;
   case         18: c = "MQRQ_SUB_DEST_NOT_AUTHORIZED"; break;
   case         19: c = "MQRQ_SSL_UNKNOWN_REVOCATION"; break;
   case         20: c = "MQRQ_SYS_CONN_NOT_AUTHORIZED"; break;
   case         21: c = "MQRQ_CHANNEL_BLOCKED_ADDRESS"; break;
   case         22: c = "MQRQ_CHANNEL_BLOCKED_USERID"; break;
   case         23: c = "MQRQ_CHANNEL_BLOCKED_NOACCESS"; break;
   case         24: c = "MQRQ_MAX_ACTIVE_CHANNELS"; break;
   case         25: c = "MQRQ_MAX_CHANNELS"; break;
   case         26: c = "MQRQ_SVRCONN_INST_LIMIT"; break;
   case         27: c = "MQRQ_CLIENT_INST_LIMIT"; break;
   case         28: c = "MQRQ_CAF_NOT_INSTALLED"; break;
   case         29: c = "MQRQ_CSP_NOT_AUTHORIZED"; break;
   case         30: c = "MQRQ_FAILOVER_PERMITTED"; break;
   case         31: c = "MQRQ_FAILOVER_NOT_PERMITTED"; break;
   case         32: c = "MQRQ_STANDBY_ACTIVATED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQRT_CONFIGURATION"; break;
   case          2: c = "MQRT_EXPIRY"; break;
   case          3: c = "MQRT_NSPROC"; break;
   case          4: c = "MQRT_PROXYSUB"; break;
   case          5: c = "MQRT_SUB_CONFIGURATION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQRU_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQRU_PUBLISH_ON_REQUEST"; break;
   case          2: c = "MQRU_PUBLISH_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSCA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSCA_REQUIRED"; break;
   case          1: c = "MQSCA_OPTIONAL"; break;
   case          2: c = "MQSCA_NEVER_REQUIRED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSCOPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSCOPE_ALL"; break;
   case          1: c = "MQSCOPE_AS_PARENT"; break;
   case          4: c = "MQSCOPE_QMGR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSCO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQSCO_Q_MGR"; break;
   case          2: c = "MQSCO_CELL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSCYC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSCYC_UPPER"; break;
   case          1: c = "MQSCYC_MIXED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSECCOMM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSECCOMM_NO"; break;
   case          1: c = "MQSECCOMM_YES"; break;
   case          2: c = "MQSECCOMM_ANON"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSECITEM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSECITEM_ALL"; break;
   case          1: c = "MQSECITEM_MQADMIN"; break;
   case          2: c = "MQSECITEM_MQNLIST"; break;
   case          3: c = "MQSECITEM_MQPROC"; break;
   case          4: c = "MQSECITEM_MQQUEUE"; break;
   case          5: c = "MQSECITEM_MQCONN"; break;
   case          6: c = "MQSECITEM_MQCMDS"; break;
   case          7: c = "MQSECITEM_MXADMIN"; break;
   case          8: c = "MQSECITEM_MXNLIST"; break;
   case          9: c = "MQSECITEM_MXPROC"; break;
   case         10: c = "MQSECITEM_MXQUEUE"; break;
   case         11: c = "MQSECITEM_MXTOPIC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSECPROT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSECPROT_NONE"; break;
   case          1: c = "MQSECPROT_SSLV30"; break;
   case          2: c = "MQSECPROT_TLSV10"; break;
   case          4: c = "MQSECPROT_TLSV12"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSECSW_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQSECSW_PROCESS"; break;
   case          2: c = "MQSECSW_NAMELIST"; break;
   case          3: c = "MQSECSW_Q"; break;
   case          4: c = "MQSECSW_TOPIC"; break;
   case          6: c = "MQSECSW_CONTEXT"; break;
   case          7: c = "MQSECSW_ALTERNATE_USER"; break;
   case          8: c = "MQSECSW_COMMAND"; break;
   case          9: c = "MQSECSW_CONNECTION"; break;
   case         10: c = "MQSECSW_SUBSYSTEM"; break;
   case         11: c = "MQSECSW_COMMAND_RESOURCES"; break;
   case         15: c = "MQSECSW_Q_MGR"; break;
   case         16: c = "MQSECSW_QSG"; break;
   case         21: c = "MQSECSW_OFF_FOUND"; break;
   case         22: c = "MQSECSW_ON_FOUND"; break;
   case         23: c = "MQSECSW_OFF_NOT_FOUND"; break;
   case         24: c = "MQSECSW_ON_NOT_FOUND"; break;
   case         25: c = "MQSECSW_OFF_ERROR"; break;
   case         26: c = "MQSECSW_ON_OVERRIDDEN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSECTYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQSECTYPE_AUTHSERV"; break;
   case          2: c = "MQSECTYPE_SSL"; break;
   case          3: c = "MQSECTYPE_CLASSES"; break;
   case          4: c = "MQSECTYPE_CONNAUTH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSELTYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSELTYPE_NONE"; break;
   case          1: c = "MQSELTYPE_STANDARD"; break;
   case          2: c = "MQSELTYPE_EXTENDED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSEL_ALL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case     -30003: c = "MQSEL_ALL_SYSTEM_SELECTORS"; break;
   case     -30002: c = "MQSEL_ALL_USER_SELECTORS"; break;
   case     -30001: c = "MQSEL_ALL_SELECTORS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSEL_ANY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case     -30003: c = "MQSEL_ANY_SYSTEM_SELECTOR"; break;
   case     -30002: c = "MQSEL_ANY_USER_SELECTOR"; break;
   case     -30001: c = "MQSEL_ANY_SELECTOR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSMPO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSMPO_NONE"; break;
   case          1: c = "MQSMPO_SET_PROP_UNDER_CURSOR"; break;
   case          2: c = "MQSMPO_SET_PROP_AFTER_CURSOR"; break;
   case          4: c = "MQSMPO_APPEND_PROPERTY"; break;
   case          8: c = "MQSMPO_SET_PROP_BEFORE_CURSOR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSO_NONE"; break;
   case          1: c = "MQSO_ALTER"; break;
   case          2: c = "MQSO_CREATE"; break;
   case          4: c = "MQSO_RESUME"; break;
   case          8: c = "MQSO_DURABLE"; break;
   case         16: c = "MQSO_GROUP_SUB"; break;
   case         32: c = "MQSO_MANAGED"; break;
   case         64: c = "MQSO_SET_IDENTITY_CONTEXT"; break;
   case        128: c = "MQSO_NO_MULTICAST"; break;
   case        256: c = "MQSO_FIXED_USERID"; break;
   case        512: c = "MQSO_ANY_USERID"; break;
   case       2048: c = "MQSO_PUBLICATIONS_ON_REQUEST"; break;
   case       4096: c = "MQSO_NEW_PUBLICATIONS_ONLY"; break;
   case       8192: c = "MQSO_FAIL_IF_QUIESCING"; break;
   case     262144: c = "MQSO_ALTERNATE_USER_AUTHORITY"; break;
   case    1048576: c = "MQSO_WILDCARD_CHAR"; break;
   case    2097152: c = "MQSO_WILDCARD_TOPIC"; break;
   case    4194304: c = "MQSO_SET_CORREL_ID"; break;
   case   67108864: c = "MQSO_SCOPE_QMGR"; break;
   case  134217728: c = "MQSO_NO_READ_AHEAD"; break;
   case  268435456: c = "MQSO_READ_AHEAD"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSP_NOT_AVAILABLE"; break;
   case          1: c = "MQSP_AVAILABLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSQQM_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSQQM_USE"; break;
   case          1: c = "MQSQQM_IGNORE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSRO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSRO_NONE"; break;
   case       8192: c = "MQSRO_FAIL_IF_QUIESCING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQSR_ACTION_PUBLICATION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSSL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSSL_FIPS_NO"; break;
   case          1: c = "MQSSL_FIPS_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSTAT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSTAT_TYPE_ASYNC_ERROR"; break;
   case          1: c = "MQSTAT_TYPE_RECONNECTION"; break;
   case          2: c = "MQSTAT_TYPE_RECONNECTION_ERROR"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSTDBY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSTDBY_NOT_PERMITTED"; break;
   case          1: c = "MQSTDBY_PERMITTED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSUBTYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -2: c = "MQSUBTYPE_USER"; break;
   case         -1: c = "MQSUBTYPE_ALL"; break;
   case          1: c = "MQSUBTYPE_API"; break;
   case          2: c = "MQSUBTYPE_ADMIN"; break;
   case          3: c = "MQSUBTYPE_PROXY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSUB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQSUB_DURABLE_ALL"; break;
   case          0: c = "MQSUB_DURABLE_AS_PARENT"; break;
   case          1: c = "MQSUB_DURABLE_ALLOWED"; break;
   case          2: c = "MQSUB_DURABLE_INHIBITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSUS_NO"; break;
   case          1: c = "MQSUS_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSVC_CONTROL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSVC_CONTROL_Q_MGR"; break;
   case          1: c = "MQSVC_CONTROL_Q_MGR_START"; break;
   case          2: c = "MQSVC_CONTROL_MANUAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSVC_STATUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSVC_STATUS_STOPPED"; break;
   case          1: c = "MQSVC_STATUS_STARTING"; break;
   case          2: c = "MQSVC_STATUS_RUNNING"; break;
   case          3: c = "MQSVC_STATUS_STOPPING"; break;
   case          4: c = "MQSVC_STATUS_RETRYING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSVC_TYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSVC_TYPE_COMMAND"; break;
   case          1: c = "MQSVC_TYPE_SERVER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSYNCPOINT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSYNCPOINT_YES"; break;
   case          1: c = "MQSYNCPOINT_IFPER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSYSOBJ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSYSOBJ_YES"; break;
   case          1: c = "MQSYSOBJ_NO"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQSYSP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQSYSP_NO"; break;
   case          1: c = "MQSYSP_YES"; break;
   case          2: c = "MQSYSP_EXTENDED"; break;
   case         10: c = "MQSYSP_TYPE_INITIAL"; break;
   case         11: c = "MQSYSP_TYPE_SET"; break;
   case         12: c = "MQSYSP_TYPE_LOG_COPY"; break;
   case         13: c = "MQSYSP_TYPE_LOG_STATUS"; break;
   case         14: c = "MQSYSP_TYPE_ARCHIVE_TAPE"; break;
   case         20: c = "MQSYSP_ALLOC_BLK"; break;
   case         21: c = "MQSYSP_ALLOC_TRK"; break;
   case         22: c = "MQSYSP_ALLOC_CYL"; break;
   case         30: c = "MQSYSP_STATUS_BUSY"; break;
   case         31: c = "MQSYSP_STATUS_PREMOUNT"; break;
   case         32: c = "MQSYSP_STATUS_AVAILABLE"; break;
   case         33: c = "MQSYSP_STATUS_UNKNOWN"; break;
   case         34: c = "MQSYSP_STATUS_ALLOC_ARCHIVE"; break;
   case         35: c = "MQSYSP_STATUS_COPYING_BSDS"; break;
   case         36: c = "MQSYSP_STATUS_COPYING_LOG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQS_AVAIL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQS_AVAIL_NORMAL"; break;
   case          1: c = "MQS_AVAIL_ERROR"; break;
   case          2: c = "MQS_AVAIL_STOPPED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQS_EXPANDST_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQS_EXPANDST_NORMAL"; break;
   case          1: c = "MQS_EXPANDST_FAILED"; break;
   case          2: c = "MQS_EXPANDST_MAXIMUM"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQS_OPENMODE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQS_OPENMODE_NONE"; break;
   case          1: c = "MQS_OPENMODE_READONLY"; break;
   case          2: c = "MQS_OPENMODE_UPDATE"; break;
   case          3: c = "MQS_OPENMODE_RECOVERY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQS_STATUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQS_STATUS_CLOSED"; break;
   case          1: c = "MQS_STATUS_CLOSING"; break;
   case          2: c = "MQS_STATUS_OPENING"; break;
   case          3: c = "MQS_STATUS_OPEN"; break;
   case          4: c = "MQS_STATUS_NOTENABLED"; break;
   case          5: c = "MQS_STATUS_ALLOCFAIL"; break;
   case          6: c = "MQS_STATUS_OPENFAIL"; break;
   case          7: c = "MQS_STATUS_STGFAIL"; break;
   case          8: c = "MQS_STATUS_DATAFAIL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTA_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQTA_BLOCK"; break;
   case          2: c = "MQTA_PASSTHRU"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTA_PROXY_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQTA_PROXY_SUB_FORCE"; break;
   case          2: c = "MQTA_PROXY_SUB_FIRSTUSE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTA_PUB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTA_PUB_AS_PARENT"; break;
   case          1: c = "MQTA_PUB_INHIBITED"; break;
   case          2: c = "MQTA_PUB_ALLOWED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTA_SUB_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTA_SUB_AS_PARENT"; break;
   case          1: c = "MQTA_SUB_INHIBITED"; break;
   case          2: c = "MQTA_SUB_ALLOWED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTCPKEEP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTCPKEEP_NO"; break;
   case          1: c = "MQTCPKEEP_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTCPSTACK_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTCPSTACK_SINGLE"; break;
   case          1: c = "MQTCPSTACK_MULTIPLE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTC_OFF"; break;
   case          1: c = "MQTC_ON"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTIME_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTIME_UNIT_MINS"; break;
   case          1: c = "MQTIME_UNIT_SECS"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTOPT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTOPT_LOCAL"; break;
   case          1: c = "MQTOPT_CLUSTER"; break;
   case          2: c = "MQTOPT_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTRAXSTR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTRAXSTR_NO"; break;
   case          1: c = "MQTRAXSTR_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTRIGGER_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTRIGGER_RESTART_NO"; break;
   case          1: c = "MQTRIGGER_RESTART_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTSCOPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQTSCOPE_QMGR"; break;
   case          2: c = "MQTSCOPE_ALL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTT_NONE"; break;
   case          1: c = "MQTT_FIRST"; break;
   case          2: c = "MQTT_EVERY"; break;
   case          3: c = "MQTT_DEPTH"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQTYPE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQTYPE_AS_SET"; break;
   case          2: c = "MQTYPE_NULL"; break;
   case          4: c = "MQTYPE_BOOLEAN"; break;
   case          8: c = "MQTYPE_BYTE_STRING"; break;
   case         16: c = "MQTYPE_INT8"; break;
   case         32: c = "MQTYPE_INT16"; break;
   case         64: c = "MQTYPE_INT32"; break;
   case        128: c = "MQTYPE_INT64"; break;
   case        256: c = "MQTYPE_FLOAT32"; break;
   case        512: c = "MQTYPE_FLOAT64"; break;
   case       1024: c = "MQTYPE_STRING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUCI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUCI_NO"; break;
   case          1: c = "MQUCI_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUIDSUPP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUIDSUPP_NO"; break;
   case          1: c = "MQUIDSUPP_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUNDELIVERED_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUNDELIVERED_NORMAL"; break;
   case          1: c = "MQUNDELIVERED_SAFE"; break;
   case          2: c = "MQUNDELIVERED_DISCARD"; break;
   case          3: c = "MQUNDELIVERED_KEEP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUOWST_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUOWST_NONE"; break;
   case          1: c = "MQUOWST_ACTIVE"; break;
   case          2: c = "MQUOWST_PREPARED"; break;
   case          3: c = "MQUOWST_UNRESOLVED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUOWT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUOWT_Q_MGR"; break;
   case          1: c = "MQUOWT_CICS"; break;
   case          2: c = "MQUOWT_RRS"; break;
   case          3: c = "MQUOWT_IMS"; break;
   case          4: c = "MQUOWT_XA"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSAGE_DS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         10: c = "MQUSAGE_DS_OLDEST_ACTIVE_UOW"; break;
   case         11: c = "MQUSAGE_DS_OLDEST_PS_RECOVERY"; break;
   case         12: c = "MQUSAGE_DS_OLDEST_CF_RECOVERY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSAGE_EXPAND_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQUSAGE_EXPAND_USER"; break;
   case          2: c = "MQUSAGE_EXPAND_SYSTEM"; break;
   case          3: c = "MQUSAGE_EXPAND_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSAGE_PS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUSAGE_PS_AVAILABLE"; break;
   case          1: c = "MQUSAGE_PS_DEFINED"; break;
   case          2: c = "MQUSAGE_PS_OFFLINE"; break;
   case          3: c = "MQUSAGE_PS_NOT_DEFINED"; break;
   case          4: c = "MQUSAGE_PS_SUSPENDED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSAGE_SMDS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUSAGE_SMDS_AVAILABLE"; break;
   case          1: c = "MQUSAGE_SMDS_NO_DATA"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSEDLQ_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUSEDLQ_AS_PARENT"; break;
   case          1: c = "MQUSEDLQ_NO"; break;
   case          2: c = "MQUSEDLQ_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUSRC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUSRC_MAP"; break;
   case          1: c = "MQUSRC_NOACCESS"; break;
   case          2: c = "MQUSRC_CHANNEL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQUS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQUS_NORMAL"; break;
   case          1: c = "MQUS_TRANSMISSION"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQVL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQVL_NULL_TERMINATED"; break;
   case          0: c = "MQVL_EMPTY_STRING"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQVS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQVS_NULL_TERMINATED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQVU_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQVU_FIXED_USER"; break;
   case          2: c = "MQVU_ANY_USER"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQWARN_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQWARN_NO"; break;
   case          1: c = "MQWARN_YES"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQWIH_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQWIH_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQWI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQWI_UNLIMITED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQWS_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQWS_DEFAULT"; break;
   case          1: c = "MQWS_CHAR"; break;
   case          2: c = "MQWS_TOPIC"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQWXP_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          2: c = "MQWXP_PUT_BY_CLUSTER_CHL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXACT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQXACT_EXTERNAL"; break;
   case          2: c = "MQXACT_INTERNAL"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXCC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -8: c = "MQXCC_FAILED"; break;
   case         -7: c = "MQXCC_REQUEST_ACK"; break;
   case         -6: c = "MQXCC_CLOSE_CHANNEL"; break;
   case         -5: c = "MQXCC_SUPPRESS_EXIT"; break;
   case         -4: c = "MQXCC_SEND_SEC_MSG"; break;
   case         -3: c = "MQXCC_SEND_AND_REQUEST_SEC_MSG"; break;
   case         -2: c = "MQXCC_SKIP_FUNCTION"; break;
   case         -1: c = "MQXCC_SUPPRESS_FUNCTION"; break;
   case          0: c = "MQXCC_OK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXC_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQXC_MQOPEN"; break;
   case          2: c = "MQXC_MQCLOSE"; break;
   case          3: c = "MQXC_MQGET"; break;
   case          4: c = "MQXC_MQPUT"; break;
   case          5: c = "MQXC_MQPUT1"; break;
   case          6: c = "MQXC_MQINQ"; break;
   case          8: c = "MQXC_MQSET"; break;
   case          9: c = "MQXC_MQBACK"; break;
   case         10: c = "MQXC_MQCMIT"; break;
   case         42: c = "MQXC_MQSUB"; break;
   case         43: c = "MQXC_MQSUBRQ"; break;
   case         44: c = "MQXC_MQCB"; break;
   case         45: c = "MQXC_MQCTL"; break;
   case         46: c = "MQXC_MQSTAT"; break;
   case         48: c = "MQXC_CALLBACK"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXDR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQXDR_OK"; break;
   case          1: c = "MQXDR_CONVERSION_FAILED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXEPO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQXEPO_NONE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQXE_OTHER"; break;
   case          1: c = "MQXE_MCA"; break;
   case          2: c = "MQXE_MCA_SVRCONN"; break;
   case          3: c = "MQXE_COMMAND_SERVER"; break;
   case          4: c = "MQXE_MQSC"; break;
   case          5: c = "MQXE_MCA_CLNTCONN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXF_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQXF_INIT"; break;
   case          2: c = "MQXF_TERM"; break;
   case          3: c = "MQXF_CONN"; break;
   case          4: c = "MQXF_CONNX"; break;
   case          5: c = "MQXF_DISC"; break;
   case          6: c = "MQXF_OPEN"; break;
   case          7: c = "MQXF_CLOSE"; break;
   case          8: c = "MQXF_PUT1"; break;
   case          9: c = "MQXF_PUT"; break;
   case         10: c = "MQXF_GET"; break;
   case         11: c = "MQXF_DATA_CONV_ON_GET"; break;
   case         12: c = "MQXF_INQ"; break;
   case         13: c = "MQXF_SET"; break;
   case         14: c = "MQXF_BEGIN"; break;
   case         15: c = "MQXF_CMIT"; break;
   case         16: c = "MQXF_BACK"; break;
   case         18: c = "MQXF_STAT"; break;
   case         19: c = "MQXF_CB"; break;
   case         20: c = "MQXF_CTL"; break;
   case         21: c = "MQXF_CALLBACK"; break;
   case         22: c = "MQXF_SUB"; break;
   case         23: c = "MQXF_SUBRQ"; break;
   case         24: c = "MQXF_XACLOSE"; break;
   case         25: c = "MQXF_XACOMMIT"; break;
   case         26: c = "MQXF_XACOMPLETE"; break;
   case         27: c = "MQXF_XAEND"; break;
   case         28: c = "MQXF_XAFORGET"; break;
   case         29: c = "MQXF_XAOPEN"; break;
   case         30: c = "MQXF_XAPREPARE"; break;
   case         31: c = "MQXF_XARECOVER"; break;
   case         32: c = "MQXF_XAROLLBACK"; break;
   case         33: c = "MQXF_XASTART"; break;
   case         34: c = "MQXF_AXREG"; break;
   case         35: c = "MQXF_AXUNREG"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXPT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case         -1: c = "MQXPT_ALL"; break;
   case          0: c = "MQXPT_LOCAL"; break;
   case          1: c = "MQXPT_LU62"; break;
   case          2: c = "MQXPT_TCP"; break;
   case          3: c = "MQXPT_NETBIOS"; break;
   case          4: c = "MQXPT_SPX"; break;
   case          5: c = "MQXPT_DECNET"; break;
   case          6: c = "MQXPT_UDP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXR2_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQXR2_DEFAULT_CONTINUATION"; break;
   case          1: c = "MQXR2_PUT_WITH_DEF_USERID"; break;
   case          2: c = "MQXR2_PUT_WITH_MSG_USERID"; break;
   case          4: c = "MQXR2_USE_EXIT_BUFFER"; break;
   case          8: c = "MQXR2_CONTINUE_CHAIN"; break;
   case         16: c = "MQXR2_SUPPRESS_CHAIN"; break;
   case         32: c = "MQXR2_DYNAMIC_CACHE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXR_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQXR_BEFORE"; break;
   case          2: c = "MQXR_AFTER"; break;
   case          3: c = "MQXR_CONNECTION"; break;
   case          4: c = "MQXR_BEFORE_CONVERT"; break;
   case         11: c = "MQXR_INIT"; break;
   case         12: c = "MQXR_TERM"; break;
   case         13: c = "MQXR_MSG"; break;
   case         14: c = "MQXR_XMIT"; break;
   case         15: c = "MQXR_SEC_MSG"; break;
   case         16: c = "MQXR_INIT_SEC"; break;
   case         17: c = "MQXR_RETRY"; break;
   case         18: c = "MQXR_AUTO_CLUSSDR"; break;
   case         19: c = "MQXR_AUTO_RECEIVER"; break;
   case         20: c = "MQXR_CLWL_OPEN"; break;
   case         21: c = "MQXR_CLWL_PUT"; break;
   case         22: c = "MQXR_CLWL_MOVE"; break;
   case         23: c = "MQXR_CLWL_REPOS"; break;
   case         24: c = "MQXR_CLWL_REPOS_MOVE"; break;
   case         25: c = "MQXR_END_BATCH"; break;
   case         26: c = "MQXR_ACK_RECEIVED"; break;
   case         27: c = "MQXR_AUTO_SVRCONN"; break;
   case         28: c = "MQXR_AUTO_CLUSRCVR"; break;
   case         29: c = "MQXR_SEC_PARMS"; break;
   case         30: c = "MQXR_PUBLICATION"; break;
   case         31: c = "MQXR_PRECONNECT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQXT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          1: c = "MQXT_API_CROSSING_EXIT"; break;
   case          2: c = "MQXT_API_EXIT"; break;
   case         11: c = "MQXT_CHANNEL_SEC_EXIT"; break;
   case         12: c = "MQXT_CHANNEL_MSG_EXIT"; break;
   case         13: c = "MQXT_CHANNEL_SEND_EXIT"; break;
   case         14: c = "MQXT_CHANNEL_RCV_EXIT"; break;
   case         15: c = "MQXT_CHANNEL_MSG_RETRY_EXIT"; break;
   case         16: c = "MQXT_CHANNEL_AUTO_DEF_EXIT"; break;
   case         20: c = "MQXT_CLUSTER_WORKLOAD_EXIT"; break;
   case         21: c = "MQXT_PUBSUB_ROUTING_EXIT"; break;
   case         22: c = "MQXT_PUBLISH_EXIT"; break;
   case         23: c = "MQXT_PRECONNECT_EXIT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZAET_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZAET_NONE"; break;
   case          1: c = "MQZAET_PRINCIPAL"; break;
   case          2: c = "MQZAET_GROUP"; break;
   case          3: c = "MQZAET_UNKNOWN"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZAO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZAO_NONE"; break;
   case          1: c = "MQZAO_CONNECT"; break;
   case          2: c = "MQZAO_BROWSE"; break;
   case          4: c = "MQZAO_INPUT"; break;
   case          8: c = "MQZAO_OUTPUT"; break;
   case         16: c = "MQZAO_INQUIRE"; break;
   case         32: c = "MQZAO_SET"; break;
   case         64: c = "MQZAO_PASS_IDENTITY_CONTEXT"; break;
   case        128: c = "MQZAO_PASS_ALL_CONTEXT"; break;
   case        256: c = "MQZAO_SET_IDENTITY_CONTEXT"; break;
   case        512: c = "MQZAO_SET_ALL_CONTEXT"; break;
   case       1024: c = "MQZAO_ALTERNATE_USER_AUTHORITY"; break;
   case       2048: c = "MQZAO_PUBLISH"; break;
   case       4096: c = "MQZAO_SUBSCRIBE"; break;
   case       8192: c = "MQZAO_RESUME"; break;
   case      16383: c = "MQZAO_ALL_MQI"; break;
   case      65536: c = "MQZAO_CREATE"; break;
   case     131072: c = "MQZAO_DELETE"; break;
   case     262144: c = "MQZAO_DISPLAY"; break;
   case     524288: c = "MQZAO_CHANGE"; break;
   case    1048576: c = "MQZAO_CLEAR"; break;
   case    2097152: c = "MQZAO_CONTROL"; break;
   case    4194304: c = "MQZAO_CONTROL_EXTENDED"; break;
   case    8388608: c = "MQZAO_AUTHORIZE"; break;
   case   16646144: c = "MQZAO_ALL_ADMIN"; break;
   case   16777216: c = "MQZAO_REMOVE"; break;
   case   33554432: c = "MQZAO_SYSTEM"; break;
   case   50216959: c = "MQZAO_ALL"; break;
   case   67108864: c = "MQZAO_CREATE_ONLY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZAT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZAT_INITIAL_CONTEXT"; break;
   case          1: c = "MQZAT_CHANGE_CONTEXT"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZCI_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZCI_CONTINUE"; break;
   case          1: c = "MQZCI_STOP"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZIO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZIO_PRIMARY"; break;
   case          1: c = "MQZIO_SECONDARY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZSE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZSE_CONTINUE"; break;
   case          1: c = "MQZSE_START"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZSL_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZSL_NOT_RETURNED"; break;
   case          1: c = "MQZSL_RETURNED"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQZTO_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQZTO_PRIMARY"; break;
   case          1: c = "MQZTO_SECONDARY"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQ_CERT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQ_CERT_VAL_POLICY_ANY"; break;
   case          1: c = "MQ_CERT_VAL_POLICY_RFC5280"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQ_MQTT_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case      65536: c = "MQ_MQTT_MAX_KEEP_ALIVE"; break;
   default: c = ""; break;
   }
   return c;
 }

 char *MQ_SUITE_STR (MQLONG v) 
 {
   char *c;
   switch (v)
   {
   case          0: c = "MQ_SUITE_B_NOT_AVAILABLE"; break;
   case          1: c = "MQ_SUITE_B_NONE"; break;
   case          2: c = "MQ_SUITE_B_128_BIT"; break;
   case          4: c = "MQ_SUITE_B_192_BIT"; break;
   default: c = ""; break;
   }
   return c;
 }



 const struct MQI_BY_VALUE_STR { 
  MQLONG value;
  char *name;
 } MQI_BY_VALUE_STR[] = { 
   {  -270532353, "MQRO_ACCEPT_UNSUP_MASK" },
   {    -1048576, "MQMF_ACCEPT_UNSUP_MASK" },
   {    -1048576, "MQPD_REJECT_UNSUP_MASK" },
   {      -65536, "MQRFH_FLAGS_RESTRICTED_MASK" },
   {      -65536, "MQROUTE_DELIVER_REJ_UNSUP_MASK" },
   {      -65536, "MQROUTE_FORWARD_REJ_UNSUP_MASK" },
   {      -30003, "MQSEL_ALL_SYSTEM_SELECTORS" },
   {      -30003, "MQSEL_ANY_SYSTEM_SELECTOR" },
   {      -30002, "MQSEL_ALL_USER_SELECTORS" },
   {      -30002, "MQSEL_ANY_USER_SELECTOR" },
   {      -30001, "MQSEL_ALL_SELECTORS" },
   {      -30001, "MQSEL_ANY_SELECTOR" },
   {       -4096, "MQENC_RESERVED_MASK" },
   {       -2000, "MQIASY_LAST" },
   {          -9, "MQIASY_LAST_USED" },
   {          -9, "MQIASY_VERSION" },
   {          -8, "MQIASY_BAG_OPTIONS" },
   {          -8, "MQXCC_FAILED" },
   {          -7, "MQIASY_REASON" },
   {          -7, "MQXCC_REQUEST_ACK" },
   {          -6, "MQIASY_COMP_CODE" },
   {          -6, "MQXCC_CLOSE_CHANNEL" },
   {          -5, "MQIASY_CONTROL" },
   {          -5, "MQXCC_SUPPRESS_EXIT" },
   {          -4, "MQCCSI_AS_PUBLISHED" },
   {          -4, "MQIASY_MSG_SEQ_NUMBER" },
   {          -4, "MQXCC_SEND_SEC_MSG" },
   {          -3, "MQAUTH_ALL_MQI" },
   {          -3, "MQCCSI_APPL" },
   {          -3, "MQCLWL_USEQ_AS_Q_MGR" },
   {          -3, "MQHC_UNASSOCIATED_HCONN" },
   {          -3, "MQIASY_COMMAND" },
   {          -3, "MQMON_Q_MGR" },
   {          -3, "MQPRI_PRIORITY_AS_PUBLISHED" },
   {          -3, "MQXCC_SEND_AND_REQUEST_SEC_MSG" },
   {          -2, "MQAUTH_ALL_ADMIN" },
   {          -2, "MQCCSI_INHERIT" },
   {          -2, "MQCGWI_DEFAULT" },
   {          -2, "MQHB_NONE" },
   {          -2, "MQIASY_TYPE" },
   {          -2, "MQIAV_UNDEFINED" },
   {          -2, "MQIND_ALL" },
   {          -2, "MQLR_MAX" },
   {          -2, "MQMCP_COMPAT" },
   {          -2, "MQPRI_PRIORITY_AS_PARENT" },
   {          -2, "MQSUBTYPE_USER" },
   {          -2, "MQXCC_SKIP_FUNCTION" },
   {          -1, "MQAT_UNKNOWN" },
   {          -1, "MQAUTH_ALL" },
   {          -1, "MQBL_NULL_TERMINATED" },
   {          -1, "MQCBD_FULL_MSG_LENGTH" },
   {          -1, "MQCCSI_EMBEDDED" },
   {          -1, "MQCC_UNKNOWN" },
   {          -1, "MQCHLD_ALL" },
   {          -1, "MQCODL_AS_INPUT" },
   {          -1, "MQCOMPRESS_NOT_AVAILABLE" },
   {          -1, "MQEI_UNLIMITED" },
   {          -1, "MQENC_AS_PUBLISHED" },
   {          -1, "MQHB_UNUSABLE_HBAG" },
   {          -1, "MQHC_UNUSABLE_HCONN" },
   {          -1, "MQHM_UNUSABLE_HMSG" },
   {          -1, "MQHO_UNUSABLE_HOBJ" },
   {          -1, "MQIASY_CODED_CHAR_SET_ID" },
   {          -1, "MQIASY_FIRST" },
   {          -1, "MQIAV_NOT_APPLICABLE" },
   {          -1, "MQIND_NONE" },
   {          -1, "MQKAI_AUTO" },
   {          -1, "MQKEY_REUSE_UNLIMITED" },
   {          -1, "MQLR_AUTO" },
   {          -1, "MQMCP_ALL" },
   {          -1, "MQMMBI_UNLIMITED" },
   {          -1, "MQMON_NONE" },
   {          -1, "MQMON_NOT_AVAILABLE" },
   {          -1, "MQNSH_ALL" },
   {          -1, "MQOL_UNDEFINED" },
   {          -1, "MQPER_PERSISTENCE_AS_PARENT" },
   {          -1, "MQPRI_PRIORITY_AS_Q_DEF" },
   {          -1, "MQPRI_PRIORITY_AS_TOPIC_DEF" },
   {          -1, "MQPROP_UNRESTRICTED_LENGTH" },
   {          -1, "MQPSCT_NONE" },
   {          -1, "MQQSGD_ALL" },
   {          -1, "MQRD_NO_RECONNECT" },
   {          -1, "MQRL_UNDEFINED" },
   {          -1, "MQSUBTYPE_ALL" },
   {          -1, "MQSUB_DURABLE_ALL" },
   {          -1, "MQVL_NULL_TERMINATED" },
   {          -1, "MQVS_NULL_TERMINATED" },
   {          -1, "MQWI_UNLIMITED" },
   {          -1, "MQXCC_SUPPRESS_FUNCTION" },
   {          -1, "MQXPT_ALL" },
   {           0, "MQACTP_NEW" },
   {           0, "MQADOPT_CHECK_NONE" },
   {           0, "MQADOPT_TYPE_NO" },
   {           0, "MQADPCTX_NO" },
   {           0, "MQAIT_ALL" },
   {           0, "MQAS_NONE" },
   {           0, "MQAT_NO_CONTEXT" },
   {           0, "MQAUTHENTICATE_OS" },
   {           0, "MQAUTH_NONE" },
   {           0, "MQAUTO_START_NO" },
   {           0, "MQBMHO_NONE" },
   {           0, "MQBND_BIND_ON_OPEN" },
   {           0, "MQBO_NONE" },
   {           0, "MQBPLOCATION_BELOW" },
   {           0, "MQCADSD_NONE" },
   {           0, "MQCAFTY_NONE" },
   {           0, "MQCAP_NOT_SUPPORTED" },
   {           0, "MQCAUT_ALL" },
   {           0, "MQCBCF_NONE" },
   {           0, "MQCBDO_NONE" },
   {           0, "MQCBO_DO_NOT_CHECK_SELECTORS" },
   {           0, "MQCBO_DO_NOT_REORDER" },
   {           0, "MQCBO_LIST_FORM_INHIBITED" },
   {           0, "MQCBO_NONE" },
   {           0, "MQCBO_USER_BAG" },
   {           0, "MQCCSI_DEFAULT" },
   {           0, "MQCCSI_Q_MGR" },
   {           0, "MQCCSI_UNDEFINED" },
   {           0, "MQCCT_NO" },
   {           0, "MQCC_OK" },
   {           0, "MQCDC_NO_SENDER_CONVERSION" },
   {           0, "MQCFACCESS_ENABLED" },
   {           0, "MQCFCONLOS_TERMINATE" },
   {           0, "MQCFC_NOT_LAST" },
   {           0, "MQCFOFFLD_NONE" },
   {           0, "MQCFO_REFRESH_REPOSITORY_NO" },
   {           0, "MQCFO_REMOVE_QUEUES_NO" },
   {           0, "MQCFR_NO" },
   {           0, "MQCFSTATUS_NOT_FOUND" },
   {           0, "MQCFTYPE_APPL" },
   {           0, "MQCFT_NONE" },
   {           0, "MQCF_NONE" },
   {           0, "MQCHAD_DISABLED" },
   {           0, "MQCHIDS_NOT_INDOUBT" },
   {           0, "MQCHK_OPTIONAL" },
   {           0, "MQCHLA_DISABLED" },
   {           0, "MQCHRR_RESET_NOT_REQUESTED" },
   {           0, "MQCHSH_RESTART_NO" },
   {           0, "MQCHSR_STOP_NOT_REQUESTED" },
   {           0, "MQCHSSTATE_OTHER" },
   {           0, "MQCHS_INACTIVE" },
   {           0, "MQCIH_NONE" },
   {           0, "MQCIH_NO_SYNC_ON_RETURN" },
   {           0, "MQCIH_REPLY_WITH_NULLS" },
   {           0, "MQCIH_UNLIMITED_EXPIRATION" },
   {           0, "MQCLCT_STATIC" },
   {           0, "MQCLROUTE_DIRECT" },
   {           0, "MQCLST_ACTIVE" },
   {           0, "MQCLWL_USEQ_LOCAL" },
   {           0, "MQCLXQ_SCTQ" },
   {           0, "MQCMD_NONE" },
   {           0, "MQCMHO_DEFAULT_VALIDATION" },
   {           0, "MQCMHO_NONE" },
   {           0, "MQCNO_NONE" },
   {           0, "MQCNO_RECONNECT_AS_DEF" },
   {           0, "MQCNO_STANDARD_BINDING" },
   {           0, "MQCOMPRESS_NONE" },
   {           0, "MQCOPY_NONE" },
   {           0, "MQCO_IMMEDIATE" },
   {           0, "MQCO_NONE" },
   {           0, "MQCRC_OK" },
   {           0, "MQCSP_AUTH_NONE" },
   {           0, "MQCSRV_CONVERT_NO" },
   {           0, "MQCSRV_DLQ_NO" },
   {           0, "MQCS_NONE" },
   {           0, "MQCTES_NOSYNC" },
   {           0, "MQCTLO_NONE" },
   {           0, "MQDCC_NONE" },
   {           0, "MQDCC_SOURCE_ENC_UNDEFINED" },
   {           0, "MQDCC_TARGET_ENC_UNDEFINED" },
   {           0, "MQDELO_NONE" },
   {           0, "MQDHF_NONE" },
   {           0, "MQDISCONNECT_NORMAL" },
   {           0, "MQDLV_AS_PARENT" },
   {           0, "MQDL_NOT_SUPPORTED" },
   {           0, "MQDMHO_NONE" },
   {           0, "MQDMPO_DEL_FIRST" },
   {           0, "MQDMPO_NONE" },
   {           0, "MQDNSWLM_NO" },
   {           0, "MQDOPT_RESOLVED" },
   {           0, "MQDSB_DEFAULT" },
   {           0, "MQDSE_DEFAULT" },
   {           0, "MQENC_DECIMAL_UNDEFINED" },
   {           0, "MQENC_FLOAT_UNDEFINED" },
   {           0, "MQENC_INTEGER_UNDEFINED" },
   {           0, "MQEPH_NONE" },
   {           0, "MQEVO_OTHER" },
   {           0, "MQEVR_DISABLED" },
   {           0, "MQEXPI_OFF" },
   {           0, "MQEXTATTRS_ALL" },
   {           0, "MQEXT_ALL" },
   {           0, "MQFB_NONE" },
   {           0, "MQFC_NO" },
   {           0, "MQFUN_TYPE_UNKNOWN" },
   {           0, "MQGMO_NONE" },
   {           0, "MQGMO_NO_WAIT" },
   {           0, "MQGMO_PROPERTIES_AS_Q_DEF" },
   {           0, "MQGUR_DISABLED" },
   {           0, "MQHC_DEF_HCONN" },
   {           0, "MQHM_NONE" },
   {           0, "MQHO_NONE" },
   {           0, "MQHSTATE_INACTIVE" },
   {           0, "MQIAMO_MONITOR_FLAGS_NONE" },
   {           0, "MQIEPF_CLIENT_LIBRARY" },
   {           0, "MQIEPF_NONE" },
   {           0, "MQIEPF_NON_THREADED_LIBRARY" },
   {           0, "MQIGQ_DISABLED" },
   {           0, "MQIIH_NONE" },
   {           0, "MQIIH_UNLIMITED_EXPIRATION" },
   {           0, "MQIMGRCOV_NO" },
   {           0, "MQIMPO_INQ_FIRST" },
   {           0, "MQIMPO_NONE" },
   {           0, "MQINBD_Q_MGR" },
   {           0, "MQIPADDR_IPV4" },
   {           0, "MQIS_NO" },
   {           0, "MQIT_NONE" },
   {           0, "MQKEY_REUSE_DISABLED" },
   {           0, "MQLDAPC_INACTIVE" },
   {           0, "MQLDAP_AUTHORMD_OS" },
   {           0, "MQLDAP_NESTGRP_NO" },
   {           0, "MQMASTER_NO" },
   {           0, "MQMATCH_GENERIC" },
   {           0, "MQMCAS_STOPPED" },
   {           0, "MQMCB_DISABLED" },
   {           0, "MQMCP_NONE" },
   {           0, "MQMC_AS_PARENT" },
   {           0, "MQMDEF_NONE" },
   {           0, "MQMDS_PRIORITY" },
   {           0, "MQMEDIMGINTVL_OFF" },
   {           0, "MQMEDIMGLOGLN_OFF" },
   {           0, "MQMEDIMGSCHED_MANUAL" },
   {           0, "MQMF_NONE" },
   {           0, "MQMF_SEGMENTATION_INHIBITED" },
   {           0, "MQMHBO_NONE" },
   {           0, "MQMLP_ENCRYPTION_ALG_NONE" },
   {           0, "MQMLP_SIGN_ALG_NONE" },
   {           0, "MQMLP_TOLERATE_UNPROTECTED_NO" },
   {           0, "MQMODE_FORCE" },
   {           0, "MQMON_DISABLED" },
   {           0, "MQMON_OFF" },
   {           0, "MQMO_NONE" },
   {           0, "MQMULC_STANDARD" },
   {           0, "MQNPM_CLASS_NORMAL" },
   {           0, "MQNSH_NONE" },
   {           0, "MQNT_NONE" },
   {           0, "MQOM_NO" },
   {           0, "MQOO_BIND_AS_Q_DEF" },
   {           0, "MQOO_READ_AHEAD_AS_Q_DEF" },
   {           0, "MQOPER_SYSTEM_FIRST" },
   {           0, "MQOPER_UNKNOWN" },
   {           0, "MQOPMODE_COMPAT" },
   {           0, "MQOT_NONE" },
   {           0, "MQPAGECLAS_4KB" },
   {           0, "MQPD_NONE" },
   {           0, "MQPD_NO_CONTEXT" },
   {           0, "MQPER_NOT_PERSISTENT" },
   {           0, "MQPMO_NONE" },
   {           0, "MQPMO_RESPONSE_AS_Q_DEF" },
   {           0, "MQPMO_RESPONSE_AS_TOPIC_DEF" },
   {           0, "MQPMRF_NONE" },
   {           0, "MQPO_NO" },
   {           0, "MQPROP_COMPATIBILITY" },
   {           0, "MQPRT_RESPONSE_AS_PARENT" },
   {           0, "MQPSCLUS_DISABLED" },
   {           0, "MQPSM_DISABLED" },
   {           0, "MQPSPROP_NONE" },
   {           0, "MQPSST_ALL" },
   {           0, "MQPS_STATUS_INACTIVE" },
   {           0, "MQPUBO_NONE" },
   {           0, "MQQA_BACKOUT_NOT_HARDENED" },
   {           0, "MQQA_GET_ALLOWED" },
   {           0, "MQQA_NOT_SHAREABLE" },
   {           0, "MQQA_PUT_ALLOWED" },
   {           0, "MQQMOPT_DISABLED" },
   {           0, "MQQMT_NORMAL" },
   {           0, "MQQO_NO" },
   {           0, "MQQSGD_Q_MGR" },
   {           0, "MQQSGS_UNKNOWN" },
   {           0, "MQQSIE_NONE" },
   {           0, "MQQSO_NO" },
   {           0, "MQQSUM_NO" },
   {           0, "MQRAR_NO" },
   {           0, "MQRCN_NO" },
   {           0, "MQRCVTIME_MULTIPLY" },
   {           0, "MQRC_NONE" },
   {           0, "MQRDNS_ENABLED" },
   {           0, "MQRD_NO_DELAY" },
   {           0, "MQREADA_NO" },
   {           0, "MQRECAUTO_NO" },
   {           0, "MQRECORDING_DISABLED" },
   {           0, "MQREGO_NONE" },
   {           0, "MQREORG_DISABLED" },
   {           0, "MQRFH_NONE" },
   {           0, "MQRFH_NO_FLAGS" },
   {           0, "MQRMHF_NOT_LAST" },
   {           0, "MQROUTE_UNLIMITED_ACTIVITIES" },
   {           0, "MQRO_COPY_MSG_ID_TO_CORREL_ID" },
   {           0, "MQRO_DEAD_LETTER_Q" },
   {           0, "MQRO_NEW_MSG_ID" },
   {           0, "MQRO_NONE" },
   {           0, "MQRP_NO" },
   {           0, "MQSCA_REQUIRED" },
   {           0, "MQSCOPE_ALL" },
   {           0, "MQSCO_RESET_COUNT_DEFAULT" },
   {           0, "MQSCYC_UPPER" },
   {           0, "MQSECCOMM_NO" },
   {           0, "MQSECITEM_ALL" },
   {           0, "MQSECPROT_NONE" },
   {           0, "MQSELTYPE_NONE" },
   {           0, "MQSMPO_NONE" },
   {           0, "MQSMPO_SET_FIRST" },
   {           0, "MQSO_NONE" },
   {           0, "MQSO_NON_DURABLE" },
   {           0, "MQSO_READ_AHEAD_AS_Q_DEF" },
   {           0, "MQSP_NOT_AVAILABLE" },
   {           0, "MQSQQM_USE" },
   {           0, "MQSRO_NONE" },
   {           0, "MQSSL_FIPS_NO" },
   {           0, "MQSTAT_TYPE_ASYNC_ERROR" },
   {           0, "MQSTDBY_NOT_PERMITTED" },
   {           0, "MQSUB_DURABLE_AS_PARENT" },
   {           0, "MQSUS_NO" },
   {           0, "MQSVC_CONTROL_Q_MGR" },
   {           0, "MQSVC_STATUS_STOPPED" },
   {           0, "MQSVC_TYPE_COMMAND" },
   {           0, "MQSYNCPOINT_YES" },
   {           0, "MQSYSOBJ_YES" },
   {           0, "MQSYSP_NO" },
   {           0, "MQS_AVAIL_NORMAL" },
   {           0, "MQS_EXPANDST_NORMAL" },
   {           0, "MQS_OPENMODE_NONE" },
   {           0, "MQS_STATUS_CLOSED" },
   {           0, "MQTA_PUB_AS_PARENT" },
   {           0, "MQTA_SUB_AS_PARENT" },
   {           0, "MQTCPKEEP_NO" },
   {           0, "MQTCPSTACK_SINGLE" },
   {           0, "MQTC_OFF" },
   {           0, "MQTIME_UNIT_MINS" },
   {           0, "MQTOPT_LOCAL" },
   {           0, "MQTRAXSTR_NO" },
   {           0, "MQTRIGGER_RESTART_NO" },
   {           0, "MQTT_NONE" },
   {           0, "MQTYPE_AS_SET" },
   {           0, "MQUCI_NO" },
   {           0, "MQUIDSUPP_NO" },
   {           0, "MQUNDELIVERED_NORMAL" },
   {           0, "MQUOWST_NONE" },
   {           0, "MQUOWT_Q_MGR" },
   {           0, "MQUSAGE_PS_AVAILABLE" },
   {           0, "MQUSAGE_SMDS_AVAILABLE" },
   {           0, "MQUSEDLQ_AS_PARENT" },
   {           0, "MQUSRC_MAP" },
   {           0, "MQUS_NORMAL" },
   {           0, "MQVL_EMPTY_STRING" },
   {           0, "MQWARN_NO" },
   {           0, "MQWIH_NONE" },
   {           0, "MQWS_DEFAULT" },
   {           0, "MQXCC_OK" },
   {           0, "MQXDR_OK" },
   {           0, "MQXEPO_NONE" },
   {           0, "MQXE_OTHER" },
   {           0, "MQXPT_LOCAL" },
   {           0, "MQXR2_DEFAULT_CONTINUATION" },
   {           0, "MQXR2_PUT_WITH_DEF_ACTION" },
   {           0, "MQXR2_STATIC_CACHE" },
   {           0, "MQXR2_USE_AGENT_BUFFER" },
   {           0, "MQZAET_NONE" },
   {           0, "MQZAO_NONE" },
   {           0, "MQZAT_INITIAL_CONTEXT" },
   {           0, "MQZCI_CONTINUE" },
   {           0, "MQZCI_DEFAULT" },
   {           0, "MQZID_INIT" },
   {           0, "MQZID_INIT_AUTHORITY" },
   {           0, "MQZID_INIT_NAME" },
   {           0, "MQZID_INIT_USERID" },
   {           0, "MQZIO_PRIMARY" },
   {           0, "MQZSE_CONTINUE" },
   {           0, "MQZSL_NOT_RETURNED" },
   {           0, "MQZTO_PRIMARY" },
   {           0, "MQ_CERT_VAL_POLICY_ANY" },
   {           0, "MQ_CERT_VAL_POLICY_DEFAULT" },
   {           0, "MQ_SUITE_B_NOT_AVAILABLE" },
   {           1, "MQACH_CURRENT_VERSION" },
   {           1, "MQACH_VERSION_1" },
   {           1, "MQACTP_FORWARD" },
   {           1, "MQACTV_DETAIL_LOW" },
   {           1, "MQACT_FORCE_REMOVE" },
   {           1, "MQADOPT_CHECK_ALL" },
   {           1, "MQADOPT_TYPE_ALL" },
   {           1, "MQADPCTX_YES" },
   {           1, "MQAIR_VERSION_1" },
   {           1, "MQAIT_CRL_LDAP" },
   {           1, "MQAS_STARTED" },
   {           1, "MQAT_CICS" },
   {           1, "MQAUTHENTICATE_PAM" },
   {           1, "MQAUTHOPT_ENTITY_EXPLICIT" },
   {           1, "MQAUTH_ALT_USER_AUTHORITY" },
   {           1, "MQAUTO_START_YES" },
   {           1, "MQAXC_VERSION_1" },
   {           1, "MQAXP_VERSION_1" },
   {           1, "MQBMHO_CURRENT_VERSION" },
   {           1, "MQBMHO_DELETE_PROPERTIES" },
   {           1, "MQBMHO_VERSION_1" },
   {           1, "MQBND_BIND_NOT_FIXED" },
   {           1, "MQBO_CURRENT_VERSION" },
   {           1, "MQBO_VERSION_1" },
   {           1, "MQBPLOCATION_ABOVE" },
   {           1, "MQBT_OTMA" },
   {           1, "MQCADSD_SEND" },
   {           1, "MQCAFTY_PREFERRED" },
   {           1, "MQCAP_SUPPORTED" },
   {           1, "MQCAUT_BLOCKUSER" },
   {           1, "MQCBCF_READA_BUFFER_EMPTY" },
   {           1, "MQCBCT_START_CALL" },
   {           1, "MQCBC_VERSION_1" },
   {           1, "MQCBDO_START_CALL" },
   {           1, "MQCBD_CURRENT_VERSION" },
   {           1, "MQCBD_VERSION_1" },
   {           1, "MQCBO_ADMIN_BAG" },
   {           1, "MQCBT_MESSAGE_CONSUMER" },
   {           1, "MQCCT_YES" },
   {           1, "MQCC_WARNING" },
   {           1, "MQCDC_SENDER_CONVERSION" },
   {           1, "MQCDC_VERSION_1" },
   {           1, "MQCD_VERSION_1" },
   {           1, "MQCFACCESS_SUSPENDED" },
   {           1, "MQCFCONLOS_TOLERATE" },
   {           1, "MQCFC_LAST" },
   {           1, "MQCFH_VERSION_1" },
   {           1, "MQCFOFFLD_SMDS" },
   {           1, "MQCFOP_LESS" },
   {           1, "MQCFO_REFRESH_REPOSITORY_YES" },
   {           1, "MQCFO_REMOVE_QUEUES_YES" },
   {           1, "MQCFR_YES" },
   {           1, "MQCFSTATUS_ACTIVE" },
   {           1, "MQCFTYPE_ADMIN" },
   {           1, "MQCFT_COMMAND" },
   {           1, "MQCF_DIST_LISTS" },
   {           1, "MQCHAD_ENABLED" },
   {           1, "MQCHIDS_INDOUBT" },
   {           1, "MQCHK_NONE" },
   {           1, "MQCHLA_ENABLED" },
   {           1, "MQCHLD_DEFAULT" },
   {           1, "MQCHSH_RESTART_YES" },
   {           1, "MQCHSR_STOP_REQUESTED" },
   {           1, "MQCHS_BINDING" },
   {           1, "MQCHTAB_Q_MGR" },
   {           1, "MQCHT_SENDER" },
   {           1, "MQCIH_PASS_EXPIRATION" },
   {           1, "MQCIH_VERSION_1" },
   {           1, "MQCIT_MULTICAST" },
   {           1, "MQCLCT_DYNAMIC" },
   {           1, "MQCLROUTE_TOPIC_HOST" },
   {           1, "MQCLRS_LOCAL" },
   {           1, "MQCLRT_RETAINED" },
   {           1, "MQCLST_PENDING" },
   {           1, "MQCLT_PROGRAM" },
   {           1, "MQCLWL_USEQ_ANY" },
   {           1, "MQCLXQ_CHANNEL" },
   {           1, "MQCMDI_CMDSCOPE_ACCEPTED" },
   {           1, "MQCMD_CHANGE_Q_MGR" },
   {           1, "MQCMHO_CURRENT_VERSION" },
   {           1, "MQCMHO_NO_VALIDATION" },
   {           1, "MQCMHO_VERSION_1" },
   {           1, "MQCNO_FASTPATH_BINDING" },
   {           1, "MQCNO_VERSION_1" },
   {           1, "MQCOMPRESS_RLE" },
   {           1, "MQCOPY_ALL" },
   {           1, "MQCO_DELETE" },
   {           1, "MQCQT_LOCAL_Q" },
   {           1, "MQCRC_CICS_EXEC_ERROR" },
   {           1, "MQCSP_AUTH_USER_ID_AND_PWD" },
   {           1, "MQCSP_CURRENT_VERSION" },
   {           1, "MQCSP_VERSION_1" },
   {           1, "MQCSRV_CONVERT_YES" },
   {           1, "MQCSRV_DLQ_YES" },
   {           1, "MQCS_SUSPENDED_TEMPORARY" },
   {           1, "MQCTLO_CURRENT_VERSION" },
   {           1, "MQCTLO_THREAD_AFFINITY" },
   {           1, "MQCTLO_VERSION_1" },
   {           1, "MQCXP_VERSION_1" },
   {           1, "MQDCC_DEFAULT_CONVERSION" },
   {           1, "MQDC_MANAGED" },
   {           1, "MQDHF_NEW_MSG_IDS" },
   {           1, "MQDH_CURRENT_VERSION" },
   {           1, "MQDH_VERSION_1" },
   {           1, "MQDISCONNECT_IMPLICIT" },
   {           1, "MQDLH_CURRENT_VERSION" },
   {           1, "MQDLH_VERSION_1" },
   {           1, "MQDLV_ALL" },
   {           1, "MQDL_SUPPORTED" },
   {           1, "MQDMHO_CURRENT_VERSION" },
   {           1, "MQDMHO_VERSION_1" },
   {           1, "MQDMPO_CURRENT_VERSION" },
   {           1, "MQDMPO_DEL_PROP_UNDER_CURSOR" },
   {           1, "MQDMPO_VERSION_1" },
   {           1, "MQDNSWLM_YES" },
   {           1, "MQDOPT_DEFINED" },
   {           1, "MQDSB_8K" },
   {           1, "MQDSE_YES" },
   {           1, "MQDXP_VERSION_1" },
   {           1, "MQENC_INTEGER_NORMAL" },
   {           1, "MQEPH_CCSID_EMBEDDED" },
   {           1, "MQEPH_CURRENT_VERSION" },
   {           1, "MQEPH_VERSION_1" },
   {           1, "MQET_MQSC" },
   {           1, "MQEVO_CONSOLE" },
   {           1, "MQEVR_ENABLED" },
   {           1, "MQEXTATTRS_NONDEF" },
   {           1, "MQEXT_OBJECT" },
   {           1, "MQFB_SYSTEM_FIRST" },
   {           1, "MQFC_YES" },
   {           1, "MQFUN_TYPE_JVM" },
   {           1, "MQGMO_VERSION_1" },
   {           1, "MQGMO_WAIT" },
   {           1, "MQGUR_ENABLED" },
   {           1, "MQHSTATE_ACTIVE" },
   {           1, "MQIAMO_MONITOR_FLAGS_OBJNAME" },
   {           1, "MQIAMO_MONITOR_UNIT" },
   {           1, "MQIA_APPL_TYPE" },
   {           1, "MQIA_FIRST" },
   {           1, "MQIDO_COMMIT" },
   {           1, "MQIEPF_THREADED_LIBRARY" },
   {           1, "MQIEP_CURRENT_VERSION" },
   {           1, "MQIEP_VERSION_1" },
   {           1, "MQIGQPA_DEFAULT" },
   {           1, "MQIGQ_ENABLED" },
   {           1, "MQIIH_CURRENT_VERSION" },
   {           1, "MQIIH_PASS_EXPIRATION" },
   {           1, "MQIIH_VERSION_1" },
   {           1, "MQIMGRCOV_YES" },
   {           1, "MQIMPO_CURRENT_VERSION" },
   {           1, "MQIMPO_VERSION_1" },
   {           1, "MQIPADDR_IPV6" },
   {           1, "MQIS_YES" },
   {           1, "MQITEM_INTEGER" },
   {           1, "MQIT_INTEGER" },
   {           1, "MQIT_MSG_ID" },
   {           1, "MQLDAPC_CONNECTED" },
   {           1, "MQLDAP_AUTHORMD_SEARCHGRP" },
   {           1, "MQLDAP_NESTGRP_YES" },
   {           1, "MQLR_ONE" },
   {           1, "MQMASTER_YES" },
   {           1, "MQMATCH_RUNCHECK" },
   {           1, "MQMCAT_PROCESS" },
   {           1, "MQMCB_ENABLED" },
   {           1, "MQMCEV_PACKET_LOSS" },
   {           1, "MQMCP_USER" },
   {           1, "MQMC_ENABLED" },
   {           1, "MQMDS_FIFO" },
   {           1, "MQMD_VERSION_1" },
   {           1, "MQMEDIMGSCHED_AUTO" },
   {           1, "MQMF_SEGMENTATION_ALLOWED" },
   {           1, "MQMHBO_CURRENT_VERSION" },
   {           1, "MQMHBO_PROPERTIES_IN_MQRFH2" },
   {           1, "MQMHBO_VERSION_1" },
   {           1, "MQMLP_ENCRYPTION_ALG_RC2" },
   {           1, "MQMLP_SIGN_ALG_MD5" },
   {           1, "MQMLP_TOLERATE_UNPROTECTED_YES" },
   {           1, "MQMODE_QUIESCE" },
   {           1, "MQMON_ENABLED" },
   {           1, "MQMON_ON" },
   {           1, "MQMO_MATCH_MSG_ID" },
   {           1, "MQMT_REQUEST" },
   {           1, "MQMT_SYSTEM_FIRST" },
   {           1, "MQMULC_REFINED" },
   {           1, "MQNPMS_NORMAL" },
   {           1, "MQNT_Q" },
   {           1, "MQNXP_VERSION_1" },
   {           1, "MQOA_FIRST" },
   {           1, "MQOD_VERSION_1" },
   {           1, "MQOM_YES" },
   {           1, "MQOO_INPUT_AS_Q_DEF" },
   {           1, "MQOPER_BROWSE" },
   {           1, "MQOPMODE_NEW_FUNCTION" },
   {           1, "MQOP_START" },
   {           1, "MQOT_Q" },
   {           1, "MQPAGECLAS_FIXED4KB" },
   {           1, "MQPA_DEFAULT" },
   {           1, "MQPBC_VERSION_1" },
   {           1, "MQPD_CURRENT_VERSION" },
   {           1, "MQPD_SUPPORT_OPTIONAL" },
   {           1, "MQPD_USER_CONTEXT" },
   {           1, "MQPD_VERSION_1" },
   {           1, "MQPER_PERSISTENT" },
   {           1, "MQPL_MVS" },
   {           1, "MQPL_OS390" },
   {           1, "MQPL_ZOS" },
   {           1, "MQPMO_VERSION_1" },
   {           1, "MQPMRF_MSG_ID" },
   {           1, "MQPO_YES" },
   {           1, "MQPROP_NONE" },
   {           1, "MQPROTO_MQTTV3" },
   {           1, "MQPRT_SYNC_RESPONSE" },
   {           1, "MQPSCLUS_ENABLED" },
   {           1, "MQPSM_COMPAT" },
   {           1, "MQPSPROP_COMPAT" },
   {           1, "MQPSST_LOCAL" },
   {           1, "MQPSXP_VERSION_1" },
   {           1, "MQPS_STATUS_STARTING" },
   {           1, "MQPUBO_CORREL_ID_AS_IDENTITY" },
   {           1, "MQQA_BACKOUT_HARDENED" },
   {           1, "MQQA_GET_INHIBITED" },
   {           1, "MQQA_PUT_INHIBITED" },
   {           1, "MQQA_SHAREABLE" },
   {           1, "MQQDT_PREDEFINED" },
   {           1, "MQQF_LOCAL_Q" },
   {           1, "MQQMDT_EXPLICIT_CLUSTER_SENDER" },
   {           1, "MQQMFAC_IMS_BRIDGE" },
   {           1, "MQQMOPT_ENABLED" },
   {           1, "MQQMSTA_STARTING" },
   {           1, "MQQMT_REPOSITORY" },
   {           1, "MQQO_YES" },
   {           1, "MQQSGD_COPY" },
   {           1, "MQQSGS_CREATED" },
   {           1, "MQQSIE_HIGH" },
   {           1, "MQQSOT_ALL" },
   {           1, "MQQSO_SHARED" },
   {           1, "MQQSO_YES" },
   {           1, "MQQSUM_YES" },
   {           1, "MQQT_LOCAL" },
   {           1, "MQRAR_YES" },
   {           1, "MQRCN_YES" },
   {           1, "MQRCVTIME_ADD" },
   {           1, "MQRDNS_DISABLED" },
   {           1, "MQREADA_YES" },
   {           1, "MQRECAUTO_YES" },
   {           1, "MQRECORDING_Q" },
   {           1, "MQREGO_CORREL_ID_AS_IDENTITY" },
   {           1, "MQREORG_ENABLED" },
   {           1, "MQRFH_VERSION_1" },
   {           1, "MQRMHF_LAST" },
   {           1, "MQRMH_CURRENT_VERSION" },
   {           1, "MQRMH_VERSION_1" },
   {           1, "MQRO_PAN" },
   {           1, "MQRP_YES" },
   {           1, "MQRQ_CONN_NOT_AUTHORIZED" },
   {           1, "MQRT_CONFIGURATION" },
   {           1, "MQRU_PUBLISH_ON_REQUEST" },
   {           1, "MQSBC_CURRENT_VERSION" },
   {           1, "MQSBC_VERSION_1" },
   {           1, "MQSCA_OPTIONAL" },
   {           1, "MQSCOPE_AS_PARENT" },
   {           1, "MQSCO_Q_MGR" },
   {           1, "MQSCO_VERSION_1" },
   {           1, "MQSCYC_MIXED" },
   {           1, "MQSD_CURRENT_VERSION" },
   {           1, "MQSD_VERSION_1" },
   {           1, "MQSECCOMM_YES" },
   {           1, "MQSECITEM_MQADMIN" },
   {           1, "MQSECPROT_SSLV30" },
   {           1, "MQSECSW_PROCESS" },
   {           1, "MQSECTYPE_AUTHSERV" },
   {           1, "MQSELTYPE_STANDARD" },
   {           1, "MQSMPO_CURRENT_VERSION" },
   {           1, "MQSMPO_SET_PROP_UNDER_CURSOR" },
   {           1, "MQSMPO_VERSION_1" },
   {           1, "MQSO_ALTER" },
   {           1, "MQSP_AVAILABLE" },
   {           1, "MQSQQM_IGNORE" },
   {           1, "MQSRO_CURRENT_VERSION" },
   {           1, "MQSRO_VERSION_1" },
   {           1, "MQSR_ACTION_PUBLICATION" },
   {           1, "MQSSL_FIPS_YES" },
   {           1, "MQSTAT_TYPE_RECONNECTION" },
   {           1, "MQSTDBY_PERMITTED" },
   {           1, "MQSTS_VERSION_1" },
   {           1, "MQSUBTYPE_API" },
   {           1, "MQSUB_DURABLE_ALLOWED" },
   {           1, "MQSUB_DURABLE_YES" },
   {           1, "MQSUS_YES" },
   {           1, "MQSVC_CONTROL_Q_MGR_START" },
   {           1, "MQSVC_STATUS_STARTING" },
   {           1, "MQSVC_TYPE_SERVER" },
   {           1, "MQSYNCPOINT_IFPER" },
   {           1, "MQSYSOBJ_NO" },
   {           1, "MQSYSP_YES" },
   {           1, "MQS_AVAIL_ERROR" },
   {           1, "MQS_EXPANDST_FAILED" },
   {           1, "MQS_OPENMODE_READONLY" },
   {           1, "MQS_STATUS_CLOSING" },
   {           1, "MQTA_BLOCK" },
   {           1, "MQTA_PROXY_SUB_FORCE" },
   {           1, "MQTA_PUB_INHIBITED" },
   {           1, "MQTA_SUB_INHIBITED" },
   {           1, "MQTCPKEEP_YES" },
   {           1, "MQTCPSTACK_MULTIPLE" },
   {           1, "MQTC_ON" },
   {           1, "MQTIME_UNIT_SECS" },
   {           1, "MQTM_CURRENT_VERSION" },
   {           1, "MQTM_VERSION_1" },
   {           1, "MQTOPT_CLUSTER" },
   {           1, "MQTRAXSTR_YES" },
   {           1, "MQTRIGGER_RESTART_YES" },
   {           1, "MQTSCOPE_QMGR" },
   {           1, "MQTT_FIRST" },
   {           1, "MQUCI_YES" },
   {           1, "MQUIDSUPP_YES" },
   {           1, "MQUNDELIVERED_SAFE" },
   {           1, "MQUOWST_ACTIVE" },
   {           1, "MQUOWT_CICS" },
   {           1, "MQUSAGE_EXPAND_USER" },
   {           1, "MQUSAGE_PS_DEFINED" },
   {           1, "MQUSAGE_SMDS_NO_DATA" },
   {           1, "MQUSEDLQ_NO" },
   {           1, "MQUSRC_NOACCESS" },
   {           1, "MQUS_TRANSMISSION" },
   {           1, "MQVU_FIXED_USER" },
   {           1, "MQWARN_YES" },
   {           1, "MQWDR_VERSION_1" },
   {           1, "MQWIH_CURRENT_VERSION" },
   {           1, "MQWIH_VERSION_1" },
   {           1, "MQWQR_VERSION_1" },
   {           1, "MQWS_CHAR" },
   {           1, "MQWXP_VERSION_1" },
   {           1, "MQXACT_EXTERNAL" },
   {           1, "MQXC_MQOPEN" },
   {           1, "MQXDR_CONVERSION_FAILED" },
   {           1, "MQXEPO_CURRENT_VERSION" },
   {           1, "MQXEPO_VERSION_1" },
   {           1, "MQXE_MCA" },
   {           1, "MQXF_INIT" },
   {           1, "MQXPT_LU62" },
   {           1, "MQXP_VERSION_1" },
   {           1, "MQXQH_CURRENT_VERSION" },
   {           1, "MQXQH_VERSION_1" },
   {           1, "MQXR2_PUT_WITH_DEF_USERID" },
   {           1, "MQXR_BEFORE" },
   {           1, "MQXT_API_CROSSING_EXIT" },
   {           1, "MQXWD_VERSION_1" },
   {           1, "MQZAC_CURRENT_VERSION" },
   {           1, "MQZAC_VERSION_1" },
   {           1, "MQZAD_VERSION_1" },
   {           1, "MQZAET_PRINCIPAL" },
   {           1, "MQZAO_CONNECT" },
   {           1, "MQZAS_VERSION_1" },
   {           1, "MQZAT_CHANGE_CONTEXT" },
   {           1, "MQZCI_STOP" },
   {           1, "MQZED_VERSION_1" },
   {           1, "MQZFP_CURRENT_VERSION" },
   {           1, "MQZFP_VERSION_1" },
   {           1, "MQZIC_CURRENT_VERSION" },
   {           1, "MQZIC_VERSION_1" },
   {           1, "MQZID_TERM" },
   {           1, "MQZID_TERM_AUTHORITY" },
   {           1, "MQZID_TERM_NAME" },
   {           1, "MQZID_TERM_USERID" },
   {           1, "MQZIO_SECONDARY" },
   {           1, "MQZNS_VERSION_1" },
   {           1, "MQZSE_START" },
   {           1, "MQZSL_RETURNED" },
   {           1, "MQZTO_SECONDARY" },
   {           1, "MQZUS_VERSION_1" },
   {           1, "MQ_CERT_VAL_POLICY_RFC5280" },
   {           1, "MQ_SUITE_B_NONE" },
   {           2, "MQACTP_REPLY" },
   {           2, "MQACTV_DETAIL_MEDIUM" },
   {           2, "MQACT_ADVANCE_LOG" },
   {           2, "MQADOPT_CHECK_Q_MGR_NAME" },
   {           2, "MQADOPT_TYPE_SVR" },
   {           2, "MQAIR_CURRENT_VERSION" },
   {           2, "MQAIR_VERSION_2" },
   {           2, "MQAIT_OCSP" },
   {           2, "MQAS_START_WAIT" },
   {           2, "MQAT_MVS" },
   {           2, "MQAT_OS390" },
   {           2, "MQAT_ZOS" },
   {           2, "MQAUTHOPT_ENTITY_SET" },
   {           2, "MQAUTH_BROWSE" },
   {           2, "MQAXC_CURRENT_VERSION" },
   {           2, "MQAXC_VERSION_2" },
   {           2, "MQAXP_CURRENT_VERSION" },
   {           2, "MQAXP_VERSION_2" },
   {           2, "MQBND_BIND_ON_GROUP" },
   {           2, "MQBPLOCATION_SWITCHING_ABOVE" },
   {           2, "MQCAP_EXPIRED" },
   {           2, "MQCAUT_BLOCKADDR" },
   {           2, "MQCBCT_STOP_CALL" },
   {           2, "MQCBC_CURRENT_VERSION" },
   {           2, "MQCBC_VERSION_2" },
   {           2, "MQCBO_LIST_FORM_ALLOWED" },
   {           2, "MQCBT_EVENT_HANDLER" },
   {           2, "MQCC_FAILED" },
   {           2, "MQCDC_VERSION_2" },
   {           2, "MQCD_VERSION_2" },
   {           2, "MQCFACCESS_DISABLED" },
   {           2, "MQCFCONLOS_ASQMGR" },
   {           2, "MQCFH_VERSION_2" },
   {           2, "MQCFOFFLD_DB2" },
   {           2, "MQCFOP_EQUAL" },
   {           2, "MQCFSTATUS_IN_RECOVER" },
   {           2, "MQCFT_RESPONSE" },
   {           2, "MQCHK_REQUIRED_ADMIN" },
   {           2, "MQCHLD_SHARED" },
   {           2, "MQCHS_STARTING" },
   {           2, "MQCHTAB_CLNTCONN" },
   {           2, "MQCHT_SERVER" },
   {           2, "MQCIH_CURRENT_VERSION" },
   {           2, "MQCIH_REPLY_WITHOUT_NULLS" },
   {           2, "MQCIH_VERSION_2" },
   {           2, "MQCLROUTE_NONE" },
   {           2, "MQCLRS_GLOBAL" },
   {           2, "MQCLST_INVALID" },
   {           2, "MQCLT_TRANSACTION" },
   {           2, "MQCMDI_CMDSCOPE_GENERATED" },
   {           2, "MQCMD_INQUIRE_Q_MGR" },
   {           2, "MQCMHO_VALIDATE" },
   {           2, "MQCNO_SERIALIZE_CONN_TAG_Q_MGR" },
   {           2, "MQCNO_VERSION_2" },
   {           2, "MQCOMPRESS_ZLIBFAST" },
   {           2, "MQCOPY_FORWARD" },
   {           2, "MQCO_DELETE_PURGE" },
   {           2, "MQCQT_ALIAS_Q" },
   {           2, "MQCRC_MQ_API_ERROR" },
   {           2, "MQCS_SUSPENDED_USER_ACTION" },
   {           2, "MQCXP_VERSION_2" },
   {           2, "MQDCC_FILL_TARGET_BUFFER" },
   {           2, "MQDC_PROVIDED" },
   {           2, "MQDISCONNECT_Q_MGR" },
   {           2, "MQDLV_ALL_DUR" },
   {           2, "MQDSB_16K" },
   {           2, "MQDSE_NO" },
   {           2, "MQDXP_CURRENT_VERSION" },
   {           2, "MQDXP_VERSION_2" },
   {           2, "MQEC_MSG_ARRIVED" },
   {           2, "MQENC_INTEGER_REVERSED" },
   {           2, "MQEVO_INIT" },
   {           2, "MQEVR_EXCEPTION" },
   {           2, "MQEXT_AUTHORITY" },
   {           2, "MQFUN_TYPE_PROGRAM" },
   {           2, "MQGMO_SYNCPOINT" },
   {           2, "MQGMO_VERSION_2" },
   {           2, "MQIAMO_MONITOR_DELTA" },
   {           2, "MQIA_CODED_CHAR_SET_ID" },
   {           2, "MQIDO_BACKOUT" },
   {           2, "MQIEPF_LOCAL_LIBRARY" },
   {           2, "MQIGQPA_CONTEXT" },
   {           2, "MQIMGRCOV_AS_Q_MGR" },
   {           2, "MQIMPO_CONVERT_TYPE" },
   {           2, "MQITEM_STRING" },
   {           2, "MQIT_CORREL_ID" },
   {           2, "MQIT_STRING" },
   {           2, "MQLDAPC_ERROR" },
   {           2, "MQLDAP_AUTHORMD_SEARCHUSR" },
   {           2, "MQMATCH_EXACT" },
   {           2, "MQMCAT_THREAD" },
   {           2, "MQMCEV_HEARTBEAT_TIMEOUT" },
   {           2, "MQMCP_REPLY" },
   {           2, "MQMC_DISABLED" },
   {           2, "MQMDE_CURRENT_VERSION" },
   {           2, "MQMDE_VERSION_2" },
   {           2, "MQMD_CURRENT_VERSION" },
   {           2, "MQMD_VERSION_2" },
   {           2, "MQMF_SEGMENT" },
   {           2, "MQMHBO_DELETE_PROPERTIES" },
   {           2, "MQMLP_ENCRYPTION_ALG_DES" },
   {           2, "MQMLP_SIGN_ALG_SHA1" },
   {           2, "MQMODE_TERMINATE" },
   {           2, "MQMO_MATCH_CORREL_ID" },
   {           2, "MQMT_REPLY" },
   {           2, "MQNPMS_FAST" },
   {           2, "MQNT_CLUSTER" },
   {           2, "MQNXP_CURRENT_VERSION" },
   {           2, "MQNXP_VERSION_2" },
   {           2, "MQOD_VERSION_2" },
   {           2, "MQOO_INPUT_SHARED" },
   {           2, "MQOPER_DISCARD" },
   {           2, "MQOP_START_WAIT" },
   {           2, "MQOT_NAMELIST" },
   {           2, "MQPA_CONTEXT" },
   {           2, "MQPBC_CURRENT_VERSION" },
   {           2, "MQPBC_VERSION_2" },
   {           2, "MQPER_PERSISTENCE_AS_Q_DEF" },
   {           2, "MQPER_PERSISTENCE_AS_TOPIC_DEF" },
   {           2, "MQPL_OS2" },
   {           2, "MQPMO_SYNCPOINT" },
   {           2, "MQPMO_VERSION_2" },
   {           2, "MQPMRF_CORREL_ID" },
   {           2, "MQPROP_ALL" },
   {           2, "MQPROTO_HTTP" },
   {           2, "MQPRT_ASYNC_RESPONSE" },
   {           2, "MQPSM_ENABLED" },
   {           2, "MQPSPROP_RFH2" },
   {           2, "MQPSST_PARENT" },
   {           2, "MQPSXP_CURRENT_VERSION" },
   {           2, "MQPSXP_VERSION_2" },
   {           2, "MQPS_STATUS_STOPPING" },
   {           2, "MQPUBO_RETAIN_PUBLICATION" },
   {           2, "MQQDT_PERMANENT_DYNAMIC" },
   {           2, "MQQMDT_AUTO_CLUSTER_SENDER" },
   {           2, "MQQMFAC_DB2" },
   {           2, "MQQMF_REPOSITORY_Q_MGR" },
   {           2, "MQQMOPT_REPLY" },
   {           2, "MQQMSTA_RUNNING" },
   {           2, "MQQSGD_SHARED" },
   {           2, "MQQSGS_ACTIVE" },
   {           2, "MQQSIE_OK" },
   {           2, "MQQSOT_INPUT" },
   {           2, "MQQSO_EXCLUSIVE" },
   {           2, "MQQT_MODEL" },
   {           2, "MQRCN_Q_MGR" },
   {           2, "MQRCVTIME_EQUAL" },
   {           2, "MQREADA_DISABLED" },
   {           2, "MQRECORDING_MSG" },
   {           2, "MQREGO_ANONYMOUS" },
   {           2, "MQRFH_VERSION_2" },
   {           2, "MQROUTE_DETAIL_LOW" },
   {           2, "MQRO_NAN" },
   {           2, "MQRQ_OPEN_NOT_AUTHORIZED" },
   {           2, "MQRT_EXPIRY" },
   {           2, "MQRU_PUBLISH_ALL" },
   {           2, "MQSCA_NEVER_REQUIRED" },
   {           2, "MQSCO_CELL" },
   {           2, "MQSCO_VERSION_2" },
   {           2, "MQSECCOMM_ANON" },
   {           2, "MQSECITEM_MQNLIST" },
   {           2, "MQSECPROT_TLSV10" },
   {           2, "MQSECSW_NAMELIST" },
   {           2, "MQSECTYPE_SSL" },
   {           2, "MQSELTYPE_EXTENDED" },
   {           2, "MQSMPO_SET_PROP_AFTER_CURSOR" },
   {           2, "MQSO_CREATE" },
   {           2, "MQSTAT_TYPE_RECONNECTION_ERROR" },
   {           2, "MQSTS_CURRENT_VERSION" },
   {           2, "MQSTS_VERSION_2" },
   {           2, "MQSUBTYPE_ADMIN" },
   {           2, "MQSUB_DURABLE_INHIBITED" },
   {           2, "MQSUB_DURABLE_NO" },
   {           2, "MQSVC_CONTROL_MANUAL" },
   {           2, "MQSVC_STATUS_RUNNING" },
   {           2, "MQSYSP_EXTENDED" },
   {           2, "MQS_AVAIL_STOPPED" },
   {           2, "MQS_EXPANDST_MAXIMUM" },
   {           2, "MQS_OPENMODE_UPDATE" },
   {           2, "MQS_STATUS_OPENING" },
   {           2, "MQTA_PASSTHRU" },
   {           2, "MQTA_PROXY_SUB_FIRSTUSE" },
   {           2, "MQTA_PUB_ALLOWED" },
   {           2, "MQTA_SUB_ALLOWED" },
   {           2, "MQTOPT_ALL" },
   {           2, "MQTSCOPE_ALL" },
   {           2, "MQTT_EVERY" },
   {           2, "MQTYPE_NULL" },
   {           2, "MQUNDELIVERED_DISCARD" },
   {           2, "MQUOWST_PREPARED" },
   {           2, "MQUOWT_RRS" },
   {           2, "MQUSAGE_EXPAND_SYSTEM" },
   {           2, "MQUSAGE_PS_OFFLINE" },
   {           2, "MQUSEDLQ_YES" },
   {           2, "MQUSRC_CHANNEL" },
   {           2, "MQVU_ANY_USER" },
   {           2, "MQWDR_CURRENT_VERSION" },
   {           2, "MQWDR_VERSION_2" },
   {           2, "MQWQR_VERSION_2" },
   {           2, "MQWS_TOPIC" },
   {           2, "MQWXP_PUT_BY_CLUSTER_CHL" },
   {           2, "MQWXP_VERSION_2" },
   {           2, "MQXACT_INTERNAL" },
   {           2, "MQXC_MQCLOSE" },
   {           2, "MQXE_MCA_SVRCONN" },
   {           2, "MQXF_TERM" },
   {           2, "MQXPT_TCP" },
   {           2, "MQXR2_PUT_WITH_MSG_USERID" },
   {           2, "MQXR_AFTER" },
   {           2, "MQXT_API_EXIT" },
   {           2, "MQZAD_CURRENT_VERSION" },
   {           2, "MQZAD_VERSION_2" },
   {           2, "MQZAET_GROUP" },
   {           2, "MQZAO_BROWSE" },
   {           2, "MQZAS_VERSION_2" },
   {           2, "MQZED_CURRENT_VERSION" },
   {           2, "MQZED_VERSION_2" },
   {           2, "MQZID_CHECK_AUTHORITY" },
   {           2, "MQZID_FIND_USERID" },
   {           2, "MQZID_LOOKUP_NAME" },
   {           2, "MQ_ARM_SUFFIX_LENGTH" },
   {           2, "MQ_SUITE_B_128_BIT" },
   {           3, "MQACTP_REPORT" },
   {           3, "MQACTV_DETAIL_HIGH" },
   {           3, "MQACT_COLLECT_STATISTICS" },
   {           3, "MQAIT_IDPW_OS" },
   {           3, "MQAS_STOPPED" },
   {           3, "MQAT_IMS" },
   {           3, "MQAUTH_CHANGE" },
   {           3, "MQBPLOCATION_SWITCHING_BELOW" },
   {           3, "MQCAUT_SSLPEERMAP" },
   {           3, "MQCBCT_REGISTER_CALL" },
   {           3, "MQCDC_VERSION_3" },
   {           3, "MQCD_VERSION_3" },
   {           3, "MQCFH_CURRENT_VERSION" },
   {           3, "MQCFH_VERSION_3" },
   {           3, "MQCFOFFLD_BOTH" },
   {           3, "MQCFOP_NOT_GREATER" },
   {           3, "MQCFSTATUS_IN_BACKUP" },
   {           3, "MQCFT_INTEGER" },
   {           3, "MQCHK_REQUIRED" },
   {           3, "MQCHS_RUNNING" },
   {           3, "MQCHT_RECEIVER" },
   {           3, "MQCLST_ERROR" },
   {           3, "MQCMDI_CMDSCOPE_COMPLETED" },
   {           3, "MQCMD_CHANGE_PROCESS" },
   {           3, "MQCNO_VERSION_3" },
   {           3, "MQCQT_REMOTE_Q" },
   {           3, "MQCRC_BRIDGE_ERROR" },
   {           3, "MQCS_SUSPENDED" },
   {           3, "MQCXP_VERSION_3" },
   {           3, "MQDLV_ALL_AVAIL" },
   {           3, "MQDSB_32K" },
   {           3, "MQEC_WAIT_INTERVAL_EXPIRED" },
   {           3, "MQEVO_MSG" },
   {           3, "MQEVR_NO_DISPLAY" },
   {           3, "MQFUN_TYPE_PROCEDURE" },
   {           3, "MQGMO_VERSION_3" },
   {           3, "MQIA_CURRENT_Q_DEPTH" },
   {           3, "MQIGQPA_ONLY_IGQ" },
   {           3, "MQINBD_GROUP" },
   {           3, "MQITEM_BAG" },
   {           3, "MQIT_BAG" },
   {           3, "MQLDAP_AUTHORMD_SRCHGRPSN" },
   {           3, "MQMATCH_ALL" },
   {           3, "MQMCAS_RUNNING" },
   {           3, "MQMCEV_VERSION_CONFLICT" },
   {           3, "MQMC_ONLY" },
   {           3, "MQMLP_ENCRYPTION_ALG_3DES" },
   {           3, "MQMLP_SIGN_ALG_SHA224" },
   {           3, "MQOD_VERSION_3" },
   {           3, "MQOPER_GET" },
   {           3, "MQOT_PROCESS" },
   {           3, "MQPA_ONLY_MCA" },
   {           3, "MQPL_AIX" },
   {           3, "MQPL_UNIX" },
   {           3, "MQPMO_CURRENT_VERSION" },
   {           3, "MQPMO_VERSION_3" },
   {           3, "MQPROP_FORCE_MQRFH2" },
   {           3, "MQPROTO_AMQP" },
   {           3, "MQPSPROP_MSGPROP" },
   {           3, "MQPSST_CHILD" },
   {           3, "MQPS_STATUS_ACTIVE" },
   {           3, "MQQDT_TEMPORARY_DYNAMIC" },
   {           3, "MQQMDT_CLUSTER_RECEIVER" },
   {           3, "MQQMSTA_QUIESCING" },
   {           3, "MQQSGD_GROUP" },
   {           3, "MQQSGS_INACTIVE" },
   {           3, "MQQSOT_OUTPUT" },
   {           3, "MQQT_ALIAS" },
   {           3, "MQRCN_DISABLED" },
   {           3, "MQREADA_INHIBITED" },
   {           3, "MQRQ_CLOSE_NOT_AUTHORIZED" },
   {           3, "MQRT_NSPROC" },
   {           3, "MQSCO_VERSION_3" },
   {           3, "MQSECITEM_MQPROC" },
   {           3, "MQSECSW_Q" },
   {           3, "MQSECTYPE_CLASSES" },
   {           3, "MQSUBTYPE_PROXY" },
   {           3, "MQSVC_STATUS_STOPPING" },
   {           3, "MQS_OPENMODE_RECOVERY" },
   {           3, "MQS_STATUS_OPEN" },
   {           3, "MQTT_DEPTH" },
   {           3, "MQUNDELIVERED_KEEP" },
   {           3, "MQUOWST_UNRESOLVED" },
   {           3, "MQUOWT_IMS" },
   {           3, "MQUSAGE_EXPAND_NONE" },
   {           3, "MQUSAGE_PS_NOT_DEFINED" },
   {           3, "MQWQR_CURRENT_VERSION" },
   {           3, "MQWQR_VERSION_3" },
   {           3, "MQWXP_VERSION_3" },
   {           3, "MQXC_MQGET" },
   {           3, "MQXE_COMMAND_SERVER" },
   {           3, "MQXF_CONN" },
   {           3, "MQXPT_NETBIOS" },
   {           3, "MQXR_CONNECTION" },
   {           3, "MQZAET_UNKNOWN" },
   {           3, "MQZAS_VERSION_3" },
   {           3, "MQZID_COPY_ALL_AUTHORITY" },
   {           3, "MQZID_INSERT_NAME" },
   {           4, "MQACT_PUBSUB" },
   {           4, "MQADOPT_CHECK_NET_ADDR" },
   {           4, "MQADOPT_TYPE_SDR" },
   {           4, "MQAIT_IDPW_LDAP" },
   {           4, "MQAS_SUSPENDED" },
   {           4, "MQAT_OS2" },
   {           4, "MQAUTH_CLEAR" },
   {           4, "MQCAUT_ADDRESSMAP" },
   {           4, "MQCBCT_DEREGISTER_CALL" },
   {           4, "MQCBDO_STOP_CALL" },
   {           4, "MQCBO_REORDER_AS_REQUIRED" },
   {           4, "MQCDC_VERSION_4" },
   {           4, "MQCD_VERSION_4" },
   {           4, "MQCFOP_GREATER" },
   {           4, "MQCFSTATUS_FAILED" },
   {           4, "MQCFT_STRING" },
   {           4, "MQCHK_AS_Q_MGR" },
   {           4, "MQCHLD_PRIVATE" },
   {           4, "MQCHS_STOPPING" },
   {           4, "MQCHT_REQUESTER" },
   {           4, "MQCIH_SYNC_ON_RETURN" },
   {           4, "MQCMDI_QSG_DISP_COMPLETED" },
   {           4, "MQCMD_COPY_PROCESS" },
   {           4, "MQCNO_SERIALIZE_CONN_TAG_QSG" },
   {           4, "MQCNO_VERSION_4" },
   {           4, "MQCOMPRESS_ZLIBHIGH" },
   {           4, "MQCOPY_PUBLISH" },
   {           4, "MQCO_KEEP_SUB" },
   {           4, "MQCQT_Q_MGR_ALIAS" },
   {           4, "MQCRC_BRIDGE_ABEND" },
   {           4, "MQCS_STOPPED" },
   {           4, "MQCXP_VERSION_4" },
   {           4, "MQDCC_INT_DEFAULT_CONVERSION" },
   {           4, "MQDELO_LOCAL" },
   {           4, "MQDSB_64K" },
   {           4, "MQEC_WAIT_CANCELED" },
   {           4, "MQEVO_MQSET" },
   {           4, "MQEVR_API_ONLY" },
   {           4, "MQFUN_TYPE_USERDEF" },
   {           4, "MQGMO_CURRENT_VERSION" },
   {           4, "MQGMO_NO_SYNCPOINT" },
   {           4, "MQGMO_VERSION_4" },
   {           4, "MQIA_DEF_INPUT_OPEN_OPTION" },
   {           4, "MQIGQPA_ALTERNATE_OR_IGQ" },
   {           4, "MQIMPO_QUERY_LENGTH" },
   {           4, "MQITEM_BYTE_STRING" },
   {           4, "MQIT_MSG_TOKEN" },
   {           4, "MQMCEV_RELIABILITY" },
   {           4, "MQMF_LAST_SEGMENT" },
   {           4, "MQMLP_ENCRYPTION_ALG_AES128" },
   {           4, "MQMLP_SIGN_ALG_SHA256" },
   {           4, "MQMO_MATCH_GROUP_ID" },
   {           4, "MQMT_REPORT" },
   {           4, "MQNT_AUTH_INFO" },
   {           4, "MQOD_CURRENT_VERSION" },
   {           4, "MQOD_VERSION_4" },
   {           4, "MQOO_INPUT_EXCLUSIVE" },
   {           4, "MQOPER_PUT" },
   {           4, "MQOP_STOP" },
   {           4, "MQOT_STORAGE_CLASS" },
   {           4, "MQPA_ALTERNATE_OR_MCA" },
   {           4, "MQPL_OS400" },
   {           4, "MQPMO_NO_SYNCPOINT" },
   {           4, "MQPMRF_GROUP_ID" },
   {           4, "MQPROP_V6COMPAT" },
   {           4, "MQPROTO_MQTTV311" },
   {           4, "MQPS_STATUS_COMPAT" },
   {           4, "MQPUBO_OTHER_SUBSCRIBERS_ONLY" },
   {           4, "MQQDT_SHARED_DYNAMIC" },
   {           4, "MQQMDT_AUTO_EXP_CLUSTER_SENDER" },
   {           4, "MQQMSTA_STANDBY" },
   {           4, "MQQSGD_PRIVATE" },
   {           4, "MQQSGS_FAILED" },
   {           4, "MQREADA_BACKLOG" },
   {           4, "MQREGO_LOCAL" },
   {           4, "MQRO_ACTIVITY" },
   {           4, "MQRQ_CMD_NOT_AUTHORIZED" },
   {           4, "MQRT_PROXYSUB" },
   {           4, "MQSCOPE_QMGR" },
   {           4, "MQSCO_VERSION_4" },
   {           4, "MQSECITEM_MQQUEUE" },
   {           4, "MQSECPROT_TLSV12" },
   {           4, "MQSECSW_TOPIC" },
   {           4, "MQSECTYPE_CONNAUTH" },
   {           4, "MQSMPO_APPEND_PROPERTY" },
   {           4, "MQSO_RESUME" },
   {           4, "MQSVC_STATUS_RETRYING" },
   {           4, "MQS_STATUS_NOTENABLED" },
   {           4, "MQTYPE_BOOLEAN" },
   {           4, "MQUOWT_XA" },
   {           4, "MQUSAGE_PS_SUSPENDED" },
   {           4, "MQWXP_CURRENT_VERSION" },
   {           4, "MQWXP_VERSION_4" },
   {           4, "MQXC_MQPUT" },
   {           4, "MQXE_MQSC" },
   {           4, "MQXF_CONNX" },
   {           4, "MQXPT_SPX" },
   {           4, "MQXR2_USE_EXIT_BUFFER" },
   {           4, "MQXR_BEFORE_CONVERT" },
   {           4, "MQZAO_INPUT" },
   {           4, "MQZAS_VERSION_4" },
   {           4, "MQZID_DELETE_AUTHORITY" },
   {           4, "MQZID_DELETE_NAME" },
   {           4, "MQ_ABEND_CODE_LENGTH" },
   {           4, "MQ_APPL_ORIGIN_DATA_LENGTH" },
   {           4, "MQ_ASID_LENGTH" },
   {           4, "MQ_ATTENTION_ID_LENGTH" },
   {           4, "MQ_AUTO_REORG_TIME_LENGTH" },
   {           4, "MQ_CANCEL_CODE_LENGTH" },
   {           4, "MQ_DB2_NAME_LENGTH" },
   {           4, "MQ_FACILITY_LIKE_LENGTH" },
   {           4, "MQ_FUNCTION_LENGTH" },
   {           4, "MQ_OPERATOR_MESSAGE_LENGTH" },
   {           4, "MQ_QSG_NAME_LENGTH" },
   {           4, "MQ_Q_MGR_CPF_LENGTH" },
   {           4, "MQ_REMOTE_PRODUCT_LENGTH" },
   {           4, "MQ_REMOTE_SYS_ID_LENGTH" },
   {           4, "MQ_SMDS_NAME_LENGTH" },
   {           4, "MQ_START_CODE_LENGTH" },
   {           4, "MQ_SUITE_B_192_BIT" },
   {           4, "MQ_SUITE_B_SIZE" },
   {           4, "MQ_TPIPE_PFX_LENGTH" },
   {           4, "MQ_TRANSACTION_ID_LENGTH" },
   {           4, "MQ_TRIGGER_TERM_ID_LENGTH" },
   {           4, "MQ_TRIGGER_TRANS_ID_LENGTH" },
   {           5, "MQACT_ADD" },
   {           5, "MQAS_SUSPENDED_TEMPORARY" },
   {           5, "MQAT_DOS" },
   {           5, "MQAUTH_CONNECT" },
   {           5, "MQCAUT_USERMAP" },
   {           5, "MQCBCT_EVENT_CALL" },
   {           5, "MQCDC_VERSION_5" },
   {           5, "MQCD_VERSION_5" },
   {           5, "MQCFOP_NOT_EQUAL" },
   {           5, "MQCFSTATUS_NONE" },
   {           5, "MQCFT_INTEGER_LIST" },
   {           5, "MQCHLD_FIXSHARED" },
   {           5, "MQCHS_RETRYING" },
   {           5, "MQCHT_ALL" },
   {           5, "MQCMDI_COMMAND_ACCEPTED" },
   {           5, "MQCMD_CREATE_PROCESS" },
   {           5, "MQCNO_VERSION_5" },
   {           5, "MQCRC_APPLICATION_ABEND" },
   {           5, "MQCXP_VERSION_5" },
   {           5, "MQDSB_128K" },
   {           5, "MQEC_Q_MGR_QUIESCING" },
   {           5, "MQEVO_INTERNAL" },
   {           5, "MQEVR_ADMIN_ONLY" },
   {           5, "MQFUN_TYPE_COMMAND" },
   {           5, "MQIA_DEF_PERSISTENCE" },
   {           5, "MQITEM_INTEGER_FILTER" },
   {           5, "MQIT_GROUP_ID" },
   {           5, "MQMCEV_CLOSED_TRANS" },
   {           5, "MQMLP_ENCRYPTION_ALG_AES256" },
   {           5, "MQMLP_SIGN_ALG_SHA384" },
   {           5, "MQOPER_PUT_REPLY" },
   {           5, "MQOT_Q_MGR" },
   {           5, "MQPL_WINDOWS" },
   {           5, "MQPS_STATUS_ERROR" },
   {           5, "MQQSGS_PENDING" },
   {           5, "MQRQ_Q_MGR_STOPPING" },
   {           5, "MQRT_SUB_CONFIGURATION" },
   {           5, "MQSCO_CURRENT_VERSION" },
   {           5, "MQSCO_VERSION_5" },
   {           5, "MQSECITEM_MQCONN" },
   {           5, "MQS_STATUS_ALLOCFAIL" },
   {           5, "MQXC_MQPUT1" },
   {           5, "MQXE_MCA_CLNTCONN" },
   {           5, "MQXF_DISC" },
   {           5, "MQXPT_DECNET" },
   {           5, "MQZAS_VERSION_5" },
   {           5, "MQZID_SET_AUTHORITY" },
   {           6, "MQACT_REPLACE" },
   {           6, "MQAS_ACTIVE" },
   {           6, "MQAT_AIX" },
   {           6, "MQAT_DEFAULT" },
   {           6, "MQAT_UNIX" },
   {           6, "MQAUTH_CREATE" },
   {           6, "MQCAUT_QMGRMAP" },
   {           6, "MQCBCT_MSG_REMOVED" },
   {           6, "MQCDC_VERSION_6" },
   {           6, "MQCD_VERSION_6" },
   {           6, "MQCFOP_NOT_LESS" },
   {           6, "MQCFSTATUS_UNKNOWN" },
   {           6, "MQCFT_STRING_LIST" },
   {           6, "MQCHS_STOPPED" },
   {           6, "MQCHT_CLNTCONN" },
   {           6, "MQCMDI_CLUSTER_REQUEST_QUEUED" },
   {           6, "MQCMD_DELETE_PROCESS" },
   {           6, "MQCNO_CURRENT_VERSION" },
   {           6, "MQCNO_VERSION_6" },
   {           6, "MQCRC_SECURITY_ERROR" },
   {           6, "MQCXP_VERSION_6" },
   {           6, "MQDSB_256K" },
   {           6, "MQEC_CONNECTION_QUIESCING" },
   {           6, "MQEVO_MQSUB" },
   {           6, "MQEVR_USER_ONLY" },
   {           6, "MQIA_DEF_PRIORITY" },
   {           6, "MQITEM_STRING_FILTER" },
   {           6, "MQMCEV_STREAM_ERROR" },
   {           6, "MQMLP_SIGN_ALG_SHA512" },
   {           6, "MQOPER_PUT_REPORT" },
   {           6, "MQOT_CHANNEL" },
   {           6, "MQPS_STATUS_REFUSED" },
   {           6, "MQQSGD_LIVE" },
   {           6, "MQQT_REMOTE" },
   {           6, "MQRQ_Q_MGR_QUIESCING" },
   {           6, "MQSECITEM_MQCMDS" },
   {           6, "MQSECSW_CONTEXT" },
   {           6, "MQS_STATUS_OPENFAIL" },
   {           6, "MQXC_MQINQ" },
   {           6, "MQXF_OPEN" },
   {           6, "MQXPT_UDP" },
   {           6, "MQZAS_VERSION_6" },
   {           6, "MQZID_GET_AUTHORITY" },
   {           6, "MQ_VOLSER_LENGTH" },
   {           7, "MQACT_REMOVE" },
   {           7, "MQAS_INACTIVE" },
   {           7, "MQAT_QMGR" },
   {           7, "MQAUTH_DELETE" },
   {           7, "MQCBCT_MSG_NOT_REMOVED" },
   {           7, "MQCDC_VERSION_7" },
   {           7, "MQCD_VERSION_7" },
   {           7, "MQCFSTATUS_RECOVERED" },
   {           7, "MQCFT_EVENT" },
   {           7, "MQCHS_REQUESTING" },
   {           7, "MQCHT_SVRCONN" },
   {           7, "MQCMDI_CHANNEL_INIT_STARTED" },
   {           7, "MQCMD_INQUIRE_PROCESS" },
   {           7, "MQCRC_PROGRAM_NOT_AVAILABLE" },
   {           7, "MQCXP_VERSION_7" },
   {           7, "MQDSB_512K" },
   {           7, "MQEVO_CTLMSG" },
   {           7, "MQIA_DEFINITION_TYPE" },
   {           7, "MQITEM_INTEGER64" },
   {           7, "MQOPER_RECEIVE" },
   {           7, "MQOT_AUTH_INFO" },
   {           7, "MQQT_CLUSTER" },
   {           7, "MQRQ_CHANNEL_STOPPED_OK" },
   {           7, "MQSECITEM_MXADMIN" },
   {           7, "MQSECSW_ALTERNATE_USER" },
   {           7, "MQS_STATUS_STGFAIL" },
   {           7, "MQXF_CLOSE" },
   {           7, "MQZID_GET_EXPLICIT_AUTHORITY" },
   {           8, "MQACT_REMOVEALL" },
   {           8, "MQADOPT_CHECK_CHANNEL_NAME" },
   {           8, "MQADOPT_TYPE_RCVR" },
   {           8, "MQAT_OS400" },
   {           8, "MQAUTH_DISPLAY" },
   {           8, "MQCBCT_MC_EVENT_CALL" },
   {           8, "MQCBO_CHECK_SELECTORS" },
   {           8, "MQCDC_VERSION_8" },
   {           8, "MQCD_VERSION_8" },
   {           8, "MQCFSTATUS_EMPTY" },
   {           8, "MQCFT_USER" },
   {           8, "MQCHS_PAUSED" },
   {           8, "MQCHT_CLUSRCVR" },
   {           8, "MQCMD_CHANGE_Q" },
   {           8, "MQCNO_RESTRICT_CONN_TAG_Q_MGR" },
   {           8, "MQCOMPRESS_SYSTEM" },
   {           8, "MQCOPY_REPLY" },
   {           8, "MQCO_REMOVE_SUB" },
   {           8, "MQCRC_BRIDGE_TIMEOUT" },
   {           8, "MQCXP_VERSION_8" },
   {           8, "MQDSB_1024K" },
   {           8, "MQDSB_1M" },
   {           8, "MQEVO_REST" },
   {           8, "MQGMO_SET_SIGNAL" },
   {           8, "MQIA_HARDEN_GET_BACKOUT" },
   {           8, "MQIIH_REPLY_FORMAT_NONE" },
   {           8, "MQIMPO_INQ_NEXT" },
   {           8, "MQITEM_BYTE_STRING_FILTER" },
   {           8, "MQMF_MSG_IN_GROUP" },
   {           8, "MQMO_MATCH_MSG_SEQ_NUMBER" },
   {           8, "MQMT_DATAGRAM" },
   {           8, "MQOO_BROWSE" },
   {           8, "MQOPER_SEND" },
   {           8, "MQOT_TOPIC" },
   {           8, "MQPMRF_FEEDBACK" },
   {           8, "MQPUBO_NO_REGISTRATION" },
   {           8, "MQQMF_CLUSSDR_USER_DEFINED" },
   {           8, "MQREGO_DIRECT_REQUESTS" },
   {           8, "MQROUTE_DETAIL_MEDIUM" },
   {           8, "MQRQ_CHANNEL_STOPPED_ERROR" },
   {           8, "MQSECITEM_MXNLIST" },
   {           8, "MQSECSW_COMMAND" },
   {           8, "MQSMPO_SET_PROP_BEFORE_CURSOR" },
   {           8, "MQSO_DURABLE" },
   {           8, "MQS_STATUS_DATAFAIL" },
   {           8, "MQTYPE_BYTE_STRING" },
   {           8, "MQXC_MQSET" },
   {           8, "MQXF_PUT1" },
   {           8, "MQXR2_CONTINUE_CHAIN" },
   {           8, "MQZAO_OUTPUT" },
   {           8, "MQZID_REFRESH_CACHE" },
   {           8, "MQ_ARCHIVE_UNIT_LENGTH" },
   {           8, "MQ_AUTHENTICATOR_LENGTH" },
   {           8, "MQ_BATCH_INTERFACE_ID_LENGTH" },
   {           8, "MQ_CHANNEL_TIME_LENGTH" },
   {           8, "MQ_CICS_FILE_NAME_LENGTH" },
   {           8, "MQ_CREATION_TIME_LENGTH" },
   {           8, "MQ_DSG_NAME_LENGTH" },
   {           8, "MQ_FACILITY_LENGTH" },
   {           8, "MQ_FORMAT_LENGTH" },
   {           8, "MQ_LOG_CORREL_ID_LENGTH" },
   {           8, "MQ_LTERM_OVERRIDE_LENGTH" },
   {           8, "MQ_LU_NAME_LENGTH" },
   {           8, "MQ_MFS_MAP_NAME_LENGTH" },
   {           8, "MQ_MODE_NAME_LENGTH" },
   {           8, "MQ_ORIGIN_NAME_LENGTH" },
   {           8, "MQ_PASS_TICKET_APPL_LENGTH" },
   {           8, "MQ_PSB_NAME_LENGTH" },
   {           8, "MQ_PST_ID_LENGTH" },
   {           8, "MQ_PUT_DATE_LENGTH" },
   {           8, "MQ_PUT_TIME_LENGTH" },
   {           8, "MQ_REMOTE_VERSION_LENGTH" },
   {           8, "MQ_SERVICE_STEP_LENGTH" },
   {           8, "MQ_SSL_KEY_MEMBER_LENGTH" },
   {           8, "MQ_STORAGE_CLASS_LENGTH" },
   {           8, "MQ_SYSTEM_NAME_LENGTH" },
   {           8, "MQ_TASK_NUMBER_LENGTH" },
   {           8, "MQ_TCP_NAME_LENGTH" },
   {           8, "MQ_TIME_LENGTH" },
   {           8, "MQ_TPIPE_NAME_LENGTH" },
   {           8, "MQ_TRIGGER_PROGRAM_NAME_LENGTH" },
   {           8, "MQ_VERSION_LENGTH" },
   {           8, "MQ_XCF_GROUP_NAME_LENGTH" },
   {           9, "MQACT_FAIL" },
   {           9, "MQAT_WINDOWS" },
   {           9, "MQAUTH_INPUT" },
   {           9, "MQCDC_VERSION_9" },
   {           9, "MQCD_VERSION_9" },
   {           9, "MQCFSTATUS_NEW" },
   {           9, "MQCFT_BYTE_STRING" },
   {           9, "MQCHS_DISCONNECTED" },
   {           9, "MQCHT_CLUSSDR" },
   {           9, "MQCMD_CLEAR_Q" },
   {           9, "MQCRC_TRANSID_NOT_AVAILABLE" },
   {           9, "MQCXP_CURRENT_VERSION" },
   {           9, "MQCXP_VERSION_9" },
   {           9, "MQIA_INHIBIT_GET" },
   {           9, "MQOPER_TRANSFORM" },
   {           9, "MQOT_COMM_INFO" },
   {           9, "MQRQ_CHANNEL_STOPPED_RETRY" },
   {           9, "MQSECITEM_MXPROC" },
   {           9, "MQSECSW_CONNECTION" },
   {           9, "MQXC_MQBACK" },
   {           9, "MQXF_PUT" },
   {           9, "MQZID_ENUMERATE_AUTHORITY_DATA" },
   {          10, "MQACT_REDUCE_LOG" },
   {          10, "MQAT_CICS_VSE" },
   {          10, "MQAUTH_INQUIRE" },
   {          10, "MQCDC_VERSION_10" },
   {          10, "MQCD_VERSION_10" },
   {          10, "MQCFOP_CONTAINS" },
   {          10, "MQCFT_TRACE_ROUTE" },
   {          10, "MQCHT_MQTT" },
   {          10, "MQCMD_COPY_Q" },
   {          10, "MQIA_INHIBIT_PUT" },
   {          10, "MQMCEV_NEW_SOURCE" },
   {          10, "MQNPM_CLASS_HIGH" },
   {          10, "MQOPER_PUBLISH" },
   {          10, "MQOT_CF_STRUC" },
   {          10, "MQRQ_CHANNEL_STOPPED_DISABLED" },
   {          10, "MQSECITEM_MXQUEUE" },
   {          10, "MQSECSW_SUBSYSTEM" },
   {          10, "MQSYSP_TYPE_INITIAL" },
   {          10, "MQUSAGE_DS_OLDEST_ACTIVE_UOW" },
   {          10, "MQXC_MQCMIT" },
   {          10, "MQXF_GET" },
   {          10, "MQZID_AUTHENTICATE_USER" },
   {          10, "MQ_APPL_FUNCTION_NAME_LENGTH" },
   {          11, "MQACT_ARCHIVE_LOG" },
   {          11, "MQAT_WINDOWS_NT" },
   {          11, "MQAUTH_OUTPUT" },
   {          11, "MQCDC_CURRENT_VERSION" },
   {          11, "MQCDC_VERSION_11" },
   {          11, "MQCD_CURRENT_VERSION" },
   {          11, "MQCD_VERSION_11" },
   {          11, "MQCHT_AMQP" },
   {          11, "MQCMDI_RECOVER_STARTED" },
   {          11, "MQCMD_CREATE_Q" },
   {          11, "MQIA_MAX_HANDLES" },
   {          11, "MQMCEV_RECEIVE_QUEUE_TRIMMED" },
   {          11, "MQOPER_EXCLUDED_PUBLISH" },
   {          11, "MQOT_LISTENER" },
   {          11, "MQPL_WINDOWS_NT" },
   {          11, "MQRQ_BRIDGE_STOPPED_OK" },
   {          11, "MQSECITEM_MXTOPIC" },
   {          11, "MQSECSW_COMMAND_RESOURCES" },
   {          11, "MQSYSP_TYPE_SET" },
   {          11, "MQUSAGE_DS_OLDEST_PS_RECOVERY" },
   {          11, "MQXF_DATA_CONV_ON_GET" },
   {          11, "MQXR_INIT" },
   {          11, "MQXT_CHANNEL_SEC_EXIT" },
   {          11, "MQZID_FREE_USER" },
   {          12, "MQAT_VMS" },
   {          12, "MQAUTH_PASS_ALL_CONTEXT" },
   {          12, "MQBMHO_CURRENT_LENGTH" },
   {          12, "MQBMHO_LENGTH_1" },
   {          12, "MQBO_CURRENT_LENGTH" },
   {          12, "MQBO_LENGTH_1" },
   {          12, "MQCFT_REPORT" },
   {          12, "MQCMDI_BACKUP_STARTED" },
   {          12, "MQCMD_DELETE_Q" },
   {          12, "MQCMHO_CURRENT_LENGTH" },
   {          12, "MQCMHO_LENGTH_1" },
   {          12, "MQCNO_LENGTH_1" },
   {          12, "MQDMHO_CURRENT_LENGTH" },
   {          12, "MQDMHO_LENGTH_1" },
   {          12, "MQDMPO_CURRENT_LENGTH" },
   {          12, "MQDMPO_LENGTH_1" },
   {          12, "MQIA_USAGE" },
   {          12, "MQMCEV_PACKET_LOSS_NACK_EXPIRE" },
   {          12, "MQMHBO_CURRENT_LENGTH" },
   {          12, "MQMHBO_LENGTH_1" },
   {          12, "MQOPER_DISCARDED_PUBLISH" },
   {          12, "MQOT_SERVICE" },
   {          12, "MQPL_VMS" },
   {          12, "MQRQ_BRIDGE_STOPPED_ERROR" },
   {          12, "MQSYSP_TYPE_LOG_COPY" },
   {          12, "MQUSAGE_DS_OLDEST_CF_RECOVERY" },
   {          12, "MQXF_INQ" },
   {          12, "MQXR_TERM" },
   {          12, "MQXT_CHANNEL_MSG_EXIT" },
   {          12, "MQZID_INQUIRE" },
   {          12, "MQ_CF_LEID_LENGTH" },
   {          12, "MQ_CF_STRUC_NAME_LENGTH" },
   {          12, "MQ_CHANNEL_DATE_LENGTH" },
   {          12, "MQ_CREATION_DATE_LENGTH" },
   {          12, "MQ_DATE_LENGTH" },
   {          12, "MQ_LRSN_LENGTH" },
   {          12, "MQ_MCA_USER_ID_LENGTH" },
   {          12, "MQ_PASSWORD_LENGTH" },
   {          12, "MQ_USER_ID_LENGTH" },
   {          13, "MQAT_GUARDIAN" },
   {          13, "MQAT_NSK" },
   {          13, "MQAUTH_PASS_IDENTITY_CONTEXT" },
   {          13, "MQCFOP_EXCLUDES" },
   {          13, "MQCFT_INTEGER_FILTER" },
   {          13, "MQCHS_INITIALIZING" },
   {          13, "MQCMDI_RECOVER_COMPLETED" },
   {          13, "MQCMD_INQUIRE_Q" },
   {          13, "MQIA_MAX_MSG_LENGTH" },
   {          13, "MQMCEV_ACK_RETRIES_EXCEEDED" },
   {          13, "MQPL_NSK" },
   {          13, "MQPL_NSS" },
   {          13, "MQRQ_SSL_HANDSHAKE_ERROR" },
   {          13, "MQSYSP_TYPE_LOG_STATUS" },
   {          13, "MQXF_SET" },
   {          13, "MQXR_MSG" },
   {          13, "MQXT_CHANNEL_SEND_EXIT" },
   {          13, "MQZID_CHECK_PRIVILEGED" },
   {          14, "MQAT_VOS" },
   {          14, "MQAUTH_SET" },
   {          14, "MQCFT_STRING_FILTER" },
   {          14, "MQCHS_SWITCHING" },
   {          14, "MQCMDI_SEC_TIMER_ZERO" },
   {          14, "MQIA_MAX_PRIORITY" },
   {          14, "MQMCEV_STREAM_SUSPEND_NACK" },
   {          14, "MQRQ_SSL_CIPHER_SPEC_ERROR" },
   {          14, "MQSYSP_TYPE_ARCHIVE_TAPE" },
   {          14, "MQXF_BEGIN" },
   {          14, "MQXR_XMIT" },
   {          14, "MQXT_CHANNEL_RCV_EXIT" },
   {          15, "MQAT_OPEN_TP1" },
   {          15, "MQAUTH_SET_ALL_CONTEXT" },
   {          15, "MQCFT_BYTE_STRING_FILTER" },
   {          15, "MQENC_INTEGER_MASK" },
   {          15, "MQIA_MAX_Q_DEPTH" },
   {          15, "MQMCEV_STREAM_RESUME_NACK" },
   {          15, "MQPL_OPEN_TP1" },
   {          15, "MQRQ_SSL_CLIENT_AUTH_ERROR" },
   {          15, "MQSECSW_Q_MGR" },
   {          15, "MQXF_CMIT" },
   {          15, "MQXR_SEC_MSG" },
   {          15, "MQXT_CHANNEL_MSG_RETRY_EXIT" },
   {          16, "MQADOPT_TYPE_CLUSRCVR" },
   {          16, "MQAUTHOPT_NAME_EXPLICIT" },
   {          16, "MQAUTH_SET_IDENTITY_CONTEXT" },
   {          16, "MQCADSD_RECV" },
   {          16, "MQCBO_COMMAND_BAG" },
   {          16, "MQCFBS_STRUC_LENGTH_FIXED" },
   {          16, "MQCFGR_STRUC_LENGTH" },
   {          16, "MQCFIL64_STRUC_LENGTH_FIXED" },
   {          16, "MQCFIL_STRUC_LENGTH_FIXED" },
   {          16, "MQCFIN_STRUC_LENGTH" },
   {          16, "MQCFT_COMMAND_XR" },
   {          16, "MQCMDI_REFRESH_CONFIGURATION" },
   {          16, "MQCMD_REFRESH_Q_MGR" },
   {          16, "MQCNO_RESTRICT_CONN_TAG_QSG" },
   {          16, "MQCOPY_REPORT" },
   {          16, "MQCUOWC_MIDDLE" },
   {          16, "MQDCC_SOURCE_ENC_FACTOR" },
   {          16, "MQDCC_SOURCE_ENC_NORMAL" },
   {          16, "MQENC_DECIMAL_NORMAL" },
   {          16, "MQGMO_BROWSE_FIRST" },
   {          16, "MQIA_MSG_DELIVERY_SEQUENCE" },
   {          16, "MQIIH_IGNORE_PURG" },
   {          16, "MQIMPO_INQ_PROP_UNDER_CURSOR" },
   {          16, "MQMCEV_STREAM_EXPELLED" },
   {          16, "MQMF_LAST_MSG_IN_GROUP" },
   {          16, "MQMO_MATCH_OFFSET" },
   {          16, "MQOO_OUTPUT" },
   {          16, "MQPMRF_ACCOUNTING_TOKEN" },
   {          16, "MQPUBO_IS_RETAINED_PUBLICATION" },
   {          16, "MQQMF_CLUSSDR_AUTO_DEFINED" },
   {          16, "MQREGO_NEW_PUBLICATIONS_ONLY" },
   {          16, "MQRQ_SSL_PEER_NAME_ERROR" },
   {          16, "MQSECSW_QSG" },
   {          16, "MQSO_GROUP_SUB" },
   {          16, "MQSRO_CURRENT_LENGTH" },
   {          16, "MQSRO_LENGTH_1" },
   {          16, "MQTYPE_INT8" },
   {          16, "MQXF_BACK" },
   {          16, "MQXR2_SUPPRESS_CHAIN" },
   {          16, "MQXR_INIT_SEC" },
   {          16, "MQXT_CHANNEL_AUTO_DEF_EXIT" },
   {          16, "MQZAO_INQUIRE" },
   {          16, "MQ_EXIT_USER_AREA_LENGTH" },
   {          16, "MQ_INSTALLATION_NAME_LENGTH" },
   {          16, "MQ_LUWID_LENGTH" },
   {          16, "MQ_MSG_TOKEN_LENGTH" },
   {          16, "MQ_RBA_LENGTH" },
   {          16, "MQ_TRAN_INSTANCE_ID_LENGTH" },
   {          16, "MQ_XCF_MEMBER_NAME_LENGTH" },
   {          17, "MQAUTH_CONTROL" },
   {          17, "MQCFT_XR_MSG" },
   {          17, "MQCMDI_SEC_SIGNOFF_ERROR" },
   {          17, "MQCMD_RESET_Q_STATS" },
   {          17, "MQCUOWC_FIRST" },
   {          17, "MQIA_OPEN_INPUT_COUNT" },
   {          17, "MQMON_LOW" },
   {          17, "MQRQ_SUB_NOT_AUTHORIZED" },
   {          17, "MQXR_RETRY" },
   {          18, "MQAT_VM" },
   {          18, "MQAUTH_CONTROL_EXTENDED" },
   {          18, "MQCFOP_LIKE" },
   {          18, "MQCFT_XR_ITEM" },
   {          18, "MQCMDI_IMS_BRIDGE_SUSPENDED" },
   {          18, "MQCMD_INQUIRE_Q_NAMES" },
   {          18, "MQIA_OPEN_OUTPUT_COUNT" },
   {          18, "MQPL_VM" },
   {          18, "MQRQ_SUB_DEST_NOT_AUTHORIZED" },
   {          18, "MQXF_STAT" },
   {          18, "MQXR_AUTO_CLUSSDR" },
   {          18, "MQ_DNS_GROUP_NAME_LENGTH" },
   {          19, "MQAT_IMS_BRIDGE" },
   {          19, "MQAUTH_PUBLISH" },
   {          19, "MQCFT_XR_SUMMARY" },
   {          19, "MQCMDI_DB2_SUSPENDED" },
   {          19, "MQCMD_INQUIRE_PROCESS_NAMES" },
   {          19, "MQIA_NAME_COUNT" },
   {          19, "MQRQ_SSL_UNKNOWN_REVOCATION" },
   {          19, "MQXF_CB" },
   {          19, "MQXR_AUTO_RECEIVER" },
   {          20, "MQAT_XCF" },
   {          20, "MQAUTH_SUBSCRIBE" },
   {          20, "MQCFBF_STRUC_LENGTH_FIXED" },
   {          20, "MQCFIF_STRUC_LENGTH" },
   {          20, "MQCFSTATUS_ADMIN_INCOMPLETE" },
   {          20, "MQCFST_STRUC_LENGTH_FIXED" },
   {          20, "MQCFT_GROUP" },
   {          20, "MQCMDI_DB2_OBSOLETE_MSGS" },
   {          20, "MQCMD_INQUIRE_CHANNEL_NAMES" },
   {          20, "MQCNO_LENGTH_2 (4 byte)" },
   {          20, "MQCTLO_CURRENT_LENGTH (4 byte)" },
   {          20, "MQCTLO_LENGTH_1 (4 byte)" },
   {          20, "MQIA_Q_TYPE" },
   {          20, "MQMCEV_FIRST_MESSAGE" },
   {          20, "MQRQ_SYS_CONN_NOT_AUTHORIZED" },
   {          20, "MQSMPO_CURRENT_LENGTH" },
   {          20, "MQSMPO_LENGTH_1" },
   {          20, "MQSYSP_ALLOC_BLK" },
   {          20, "MQXF_CTL" },
   {          20, "MQXR_CLWL_OPEN" },
   {          20, "MQXT_CLUSTER_WORKLOAD_EXIT" },
   {          20, "MQZFP_CURRENT_LENGTH (4 byte)" },
   {          20, "MQZFP_LENGTH_1 (4 byte)" },
   {          20, "MQ_CHANNEL_NAME_LENGTH" },
   {          20, "MQ_MCA_NAME_LENGTH" },
   {          20, "MQ_PROGRAM_NAME_LENGTH" },
   {          20, "MQ_SHORT_CONN_NAME_LENGTH" },
   {          21, "MQAT_CICS_BRIDGE" },
   {          21, "MQAUTH_RESUME" },
   {          21, "MQCFOP_NOT_LIKE" },
   {          21, "MQCFSTATUS_NEVER_USED" },
   {          21, "MQCFT_STATISTICS" },
   {          21, "MQCMDI_SEC_UPPERCASE" },
   {          21, "MQCMD_CHANGE_CHANNEL" },
   {          21, "MQIA_RETENTION_INTERVAL" },
   {          21, "MQMCEV_LATE_JOIN_FAILURE" },
   {          21, "MQRQ_CHANNEL_BLOCKED_ADDRESS" },
   {          21, "MQSECSW_OFF_FOUND" },
   {          21, "MQSYSP_ALLOC_TRK" },
   {          21, "MQXF_CALLBACK" },
   {          21, "MQXR_CLWL_PUT" },
   {          21, "MQXT_PUBSUB_ROUTING_EXIT" },
   {          22, "MQAT_NOTES_AGENT" },
   {          22, "MQAUTH_SYSTEM" },
   {          22, "MQCFSTATUS_NO_BACKUP" },
   {          22, "MQCFT_ACCOUNTING" },
   {          22, "MQCMDI_SEC_MIXEDCASE" },
   {          22, "MQCMD_COPY_CHANNEL" },
   {          22, "MQCOPY_DEFAULT" },
   {          22, "MQIA_BACKOUT_THRESHOLD" },
   {          22, "MQMCEV_MESSAGE_LOSS" },
   {          22, "MQRQ_CHANNEL_BLOCKED_USERID" },
   {          22, "MQSECSW_ON_FOUND" },
   {          22, "MQSYSP_ALLOC_CYL" },
   {          22, "MQXF_SUB" },
   {          22, "MQXR_CLWL_MOVE" },
   {          22, "MQXT_PUBLISH_EXIT" },
   {          23, "MQAT_TPF" },
   {          23, "MQCFSTATUS_NOT_FAILED" },
   {          23, "MQCFT_INTEGER64" },
   {          23, "MQCMD_CREATE_CHANNEL" },
   {          23, "MQIA_SHAREABILITY" },
   {          23, "MQMCEV_SEND_PACKET_FAILURE" },
   {          23, "MQPL_TPF" },
   {          23, "MQRQ_CHANNEL_BLOCKED_NOACCESS" },
   {          23, "MQSECSW_OFF_NOT_FOUND" },
   {          23, "MQXF_SUBRQ" },
   {          23, "MQXR_CLWL_REPOS" },
   {          23, "MQXT_PRECONNECT_EXIT" },
   {          23, "MQ_CLIENT_ID_LENGTH" },
   {          24, "MQCFIN64_STRUC_LENGTH" },
   {          24, "MQCFSF_STRUC_LENGTH_FIXED" },
   {          24, "MQCFSL_STRUC_LENGTH_FIXED" },
   {          24, "MQCFSTATUS_NOT_RECOVERABLE" },
   {          24, "MQCMD_DELETE_CHANNEL" },
   {          24, "MQCNO_LENGTH_2 (8 byte)" },
   {          24, "MQCTLO_CURRENT_LENGTH (8 byte)" },
   {          24, "MQCTLO_LENGTH_1 (8 byte)" },
   {          24, "MQIA_TRIGGER_CONTROL" },
   {          24, "MQMCEV_REPAIR_DELAY" },
   {          24, "MQPD_CURRENT_LENGTH" },
   {          24, "MQPD_LENGTH_1" },
   {          24, "MQRQ_MAX_ACTIVE_CHANNELS" },
   {          24, "MQSECSW_ON_NOT_FOUND" },
   {          24, "MQXF_XACLOSE" },
   {          24, "MQXR_CLWL_REPOS_MOVE" },
   {          24, "MQXWD_CURRENT_LENGTH" },
   {          24, "MQXWD_LENGTH_1" },
   {          24, "MQZFP_CURRENT_LENGTH (8 byte)" },
   {          24, "MQZFP_LENGTH_1 (8 byte)" },
   {          24, "MQ_BRIDGE_NAME_LENGTH" },
   {          24, "MQ_CONNECTION_ID_LENGTH" },
   {          24, "MQ_CORREL_ID_LENGTH" },
   {          24, "MQ_GROUP_ID_LENGTH" },
   {          24, "MQ_LOG_EXTENT_NAME_LENGTH" },
   {          24, "MQ_MSG_ID_LENGTH" },
   {          24, "MQ_OBJECT_INSTANCE_ID_LENGTH" },
   {          24, "MQ_RESPONSE_ID_LENGTH" },
   {          25, "MQAT_USER" },
   {          25, "MQCFSTATUS_XES_ERROR" },
   {          25, "MQCFT_INTEGER64_LIST" },
   {          25, "MQCMD_INQUIRE_CHANNEL" },
   {          25, "MQIA_TRIGGER_INTERVAL" },
   {          25, "MQMCEV_MEMORY_ALERT_ON" },
   {          25, "MQRQ_MAX_CHANNELS" },
   {          25, "MQSECSW_OFF_ERROR" },
   {          25, "MQXF_XACOMMIT" },
   {          25, "MQXR_END_BATCH" },
   {          26, "MQAT_BROKER" },
   {          26, "MQAT_QMGR_PUBLISH" },
   {          26, "MQCFOP_CONTAINS_GEN" },
   {          26, "MQCFT_APP_ACTIVITY" },
   {          26, "MQCMD_PING_CHANNEL" },
   {          26, "MQIA_TRIGGER_MSG_PRIORITY" },
   {          26, "MQMCEV_MEMORY_ALERT_OFF" },
   {          26, "MQRQ_SVRCONN_INST_LIMIT" },
   {          26, "MQSECSW_ON_OVERRIDDEN" },
   {          26, "MQXF_XACOMPLETE" },
   {          26, "MQXR_ACK_RECEIVED" },
   {          27, "MQCMD_RESET_CHANNEL" },
   {          27, "MQIA_CPI_LEVEL" },
   {          27, "MQMCEV_NACK_ALERT_ON" },
   {          27, "MQPL_VSE" },
   {          27, "MQRQ_CLIENT_INST_LIMIT" },
   {          27, "MQXF_XAEND" },
   {          27, "MQXR_AUTO_SVRCONN" },
   {          28, "MQAT_JAVA" },
   {          28, "MQCMD_START_CHANNEL" },
   {          28, "MQIA_TRIGGER_TYPE" },
   {          28, "MQMCEV_NACK_ALERT_OFF" },
   {          28, "MQPBC_LENGTH_1 (4 byte)" },
   {          28, "MQPL_APPLIANCE" },
   {          28, "MQRQ_CAF_NOT_INSTALLED" },
   {          28, "MQXF_XAFORGET" },
   {          28, "MQXR_AUTO_CLUSRCVR" },
   {          28, "MQ_APPL_NAME_LENGTH" },
   {          28, "MQ_APPL_TAG_LENGTH" },
   {          28, "MQ_MCA_JOB_NAME_LENGTH" },
   {          28, "MQ_PUT_APPL_NAME_LENGTH" },
   {          29, "MQAT_DQM" },
   {          29, "MQCFOP_EXCLUDES_GEN" },
   {          29, "MQCMD_STOP_CHANNEL" },
   {          29, "MQIA_TRIGGER_DEPTH" },
   {          29, "MQMCEV_REPAIR_ALERT_ON" },
   {          29, "MQRQ_CSP_NOT_AUTHORIZED" },
   {          29, "MQXF_XAOPEN" },
   {          29, "MQXR_SEC_PARMS" },
   {          30, "MQAT_CHANNEL_INITIATOR" },
   {          30, "MQCMD_START_CHANNEL_INIT" },
   {          30, "MQIA_SYNCPOINT" },
   {          30, "MQMCEV_REPAIR_ALERT_OFF" },
   {          30, "MQRQ_FAILOVER_PERMITTED" },
   {          30, "MQSYSP_STATUS_BUSY" },
   {          30, "MQXF_XAPREPARE" },
   {          30, "MQXR_PUBLICATION" },
   {          31, "MQAT_WLM" },
   {          31, "MQCMD_START_CHANNEL_LISTENER" },
   {          31, "MQIA_COMMAND_LEVEL" },
   {          31, "MQMCEV_RELIABILITY_CHANGED" },
   {          31, "MQRQ_FAILOVER_NOT_PERMITTED" },
   {          31, "MQSYSP_STATUS_PREMOUNT" },
   {          31, "MQXF_XARECOVER" },
   {          31, "MQXR_PRECONNECT" },
   {          32, "MQAT_BATCH" },
   {          32, "MQAUTHOPT_NAME_ALL_MATCHING" },
   {          32, "MQCBO_SYSTEM_BAG" },
   {          32, "MQCMD_CHANGE_NAMELIST" },
   {          32, "MQCNO_HANDLE_SHARE_NONE" },
   {          32, "MQCO_QUIESCE" },
   {          32, "MQDCC_SOURCE_ENC_NATIVE" },
   {          32, "MQDCC_SOURCE_ENC_REVERSED" },
   {          32, "MQENC_DECIMAL_REVERSED" },
   {          32, "MQGMO_BROWSE_NEXT" },
   {          32, "MQIA_PLATFORM" },
   {          32, "MQIIH_CM0_REQUEST_RESPONSE" },
   {          32, "MQIMPO_CONVERT_VALUE" },
   {          32, "MQMO_MATCH_MSG_TOKEN" },
   {          32, "MQOO_INQUIRE" },
   {          32, "MQPBC_CURRENT_LENGTH (4 byte)" },
   {          32, "MQPBC_LENGTH_1 (8 byte)" },
   {          32, "MQPBC_LENGTH_2 (4 byte)" },
   {          32, "MQPMO_DEFAULT_CONTEXT" },
   {          32, "MQQMF_AVAILABLE" },
   {          32, "MQREGO_PUBLISH_ON_REQUEST_ONLY" },
   {          32, "MQRFH_CURRENT_LENGTH" },
   {          32, "MQRFH_LENGTH_1" },
   {          32, "MQRFH_STRUC_LENGTH_FIXED" },
   {          32, "MQROUTE_DETAIL_HIGH" },
   {          32, "MQRQ_STANDBY_ACTIVATED" },
   {          32, "MQSO_MANAGED" },
   {          32, "MQSYSP_STATUS_AVAILABLE" },
   {          32, "MQTYPE_INT16" },
   {          32, "MQXEPO_CURRENT_LENGTH (4 byte)" },
   {          32, "MQXEPO_LENGTH_1 (4 byte)" },
   {          32, "MQXF_XAROLLBACK" },
   {          32, "MQXR2_DYNAMIC_CACHE" },
   {          32, "MQZAO_SET" },
   {          32, "MQ_ACCOUNTING_TOKEN_LENGTH" },
   {          32, "MQ_APPL_IDENTITY_DATA_LENGTH" },
   {          32, "MQ_CHINIT_SERVICE_PARM_LENGTH" },
   {          32, "MQ_EXIT_DATA_LENGTH" },
   {          32, "MQ_LDAP_PASSWORD_LENGTH" },
   {          32, "MQ_MCA_USER_DATA_LENGTH" },
   {          32, "MQ_SERVICE_NAME_LENGTH" },
   {          32, "MQ_SSL_CIPHER_SPEC_LENGTH" },
   {          32, "MQ_SSL_CIPHER_SUITE_LENGTH" },
   {          32, "MQ_SSL_HANDSHAKE_STAGE_LENGTH" },
   {          32, "MQ_SYSP_SERVICE_LENGTH" },
   {          33, "MQAT_RRS_BATCH" },
   {          33, "MQCMD_COPY_NAMELIST" },
   {          33, "MQIA_MAX_UNCOMMITTED_MSGS" },
   {          33, "MQMON_MEDIUM" },
   {          33, "MQSYSP_STATUS_UNKNOWN" },
   {          33, "MQXF_XASTART" },
   {          34, "MQAT_SIB" },
   {          34, "MQCMD_CREATE_NAMELIST" },
   {          34, "MQIA_DIST_LISTS" },
   {          34, "MQSYSP_STATUS_ALLOC_ARCHIVE" },
   {          34, "MQXF_AXREG" },
   {          35, "MQAT_SYSTEM_EXTENSION" },
   {          35, "MQCMD_DELETE_NAMELIST" },
   {          35, "MQIA_TIME_SINCE_RESET" },
   {          35, "MQSYSP_STATUS_COPYING_BSDS" },
   {          35, "MQXF_AXUNREG" },
   {          36, "MQAT_MCAST_PUBLISH" },
   {          36, "MQCFH_STRUC_LENGTH" },
   {          36, "MQCMD_INQUIRE_NAMELIST" },
   {          36, "MQIA_HIGH_Q_DEPTH" },
   {          36, "MQRFH2_CURRENT_LENGTH" },
   {          36, "MQRFH2_LENGTH_2" },
   {          36, "MQRFH_STRUC_LENGTH_FIXED_2" },
   {          36, "MQSYSP_STATUS_COPYING_LOG" },
   {          36, "MQ_ARCHIVE_PFX_LENGTH" },
   {          37, "MQAT_AMQP" },
   {          37, "MQCMD_INQUIRE_NAMELIST_NAMES" },
   {          37, "MQIA_MSG_ENQ_COUNT" },
   {          38, "MQCMD_ESCAPE" },
   {          38, "MQIA_MSG_DEQ_COUNT" },
   {          39, "MQCMD_RESOLVE_CHANNEL" },
   {          39, "MQIA_EXPIRY_INTERVAL" },
   {          40, "MQCMD_PING_Q_MGR" },
   {          40, "MQIA_Q_DEPTH_HIGH_LIMIT" },
   {          40, "MQPBC_CURRENT_LENGTH (8 byte)" },
   {          40, "MQPBC_LENGTH_2 (8 byte)" },
   {          40, "MQXEPO_CURRENT_LENGTH (8 byte)" },
   {          40, "MQXEPO_LENGTH_1 (8 byte)" },
   {          40, "MQ_SECURITY_ID_LENGTH" },
   {          40, "MQ_SECURITY_PROFILE_LENGTH" },
   {          41, "MQCMD_INQUIRE_Q_STATUS" },
   {          41, "MQIA_Q_DEPTH_LOW_LIMIT" },
   {          42, "MQCMD_INQUIRE_CHANNEL_STATUS" },
   {          42, "MQIA_Q_DEPTH_MAX_EVENT" },
   {          42, "MQXC_MQSUB" },
   {          43, "MQCMD_CONFIG_EVENT" },
   {          43, "MQIA_Q_DEPTH_HIGH_EVENT" },
   {          43, "MQXC_MQSUBRQ" },
   {          44, "MQCMD_Q_MGR_EVENT" },
   {          44, "MQDXP_LENGTH_1" },
   {          44, "MQIA_Q_DEPTH_LOW_EVENT" },
   {          44, "MQXC_MQCB" },
   {          44, "MQXP_CURRENT_LENGTH" },
   {          44, "MQXP_LENGTH_1" },
   {          44, "MQ_AUTO_REORG_CATALOG_LENGTH" },
   {          44, "MQ_DATA_SET_NAME_LENGTH" },
   {          44, "MQ_SSL_KEY_LIBRARY_LENGTH" },
   {          45, "MQCMD_PERFM_EVENT" },
   {          45, "MQIA_SCOPE" },
   {          45, "MQXC_MQCTL" },
   {          46, "MQCMD_CHANNEL_EVENT" },
   {          46, "MQIA_Q_SERVICE_INTERVAL_EVENT" },
   {          46, "MQXC_MQSTAT" },
   {          47, "MQIA_AUTHORITY_EVENT" },
   {          48, "MQCBC_LENGTH_1 (4 byte)" },
   {          48, "MQCSP_CURRENT_LENGTH (4 byte)" },
   {          48, "MQCSP_LENGTH_1 (4 byte)" },
   {          48, "MQDH_CURRENT_LENGTH" },
   {          48, "MQDH_LENGTH_1" },
   {          48, "MQDXP_CURRENT_LENGTH (4 byte)" },
   {          48, "MQDXP_LENGTH_2 (4 byte)" },
   {          48, "MQIA_INHIBIT_EVENT" },
   {          48, "MQXC_CALLBACK" },
   {          48, "MQ_AUTH_INFO_NAME_LENGTH" },
   {          48, "MQ_AUTH_PROFILE_NAME_LENGTH" },
   {          48, "MQ_CLUSTER_NAME_LENGTH" },
   {          48, "MQ_COMM_INFO_NAME_LENGTH" },
   {          48, "MQ_EXIT_INFO_NAME_LENGTH" },
   {          48, "MQ_EXIT_PD_AREA_LENGTH" },
   {          48, "MQ_IP_ADDRESS_LENGTH" },
   {          48, "MQ_LISTENER_NAME_LENGTH" },
   {          48, "MQ_LOCAL_ADDRESS_LENGTH" },
   {          48, "MQ_NAMELIST_NAME_LENGTH" },
   {          48, "MQ_OBJECT_NAME_LENGTH" },
   {          48, "MQ_PROCESS_NAME_LENGTH" },
   {          48, "MQ_Q_MGR_IDENTIFIER_LENGTH" },
   {          48, "MQ_Q_MGR_NAME_LENGTH" },
   {          48, "MQ_Q_NAME_LENGTH" },
   {          48, "MQ_SERVICE_COMPONENT_LENGTH" },
   {          48, "MQ_TOPIC_NAME_LENGTH" },
   {          49, "MQIA_LOCAL_EVENT" },
   {          50, "MQIA_REMOTE_EVENT" },
   {          51, "MQIA_CONFIGURATION_EVENT" },
   {          52, "MQCBC_CURRENT_LENGTH (4 byte)" },
   {          52, "MQCBC_LENGTH_2 (4 byte)" },
   {          52, "MQIA_START_STOP_EVENT" },
   {          52, "MQNXP_LENGTH_1 (4 byte)" },
   {          53, "MQIA_PERFORMANCE_EVENT" },
   {          54, "MQIA_Q_SERVICE_INTERVAL" },
   {          55, "MQIA_CHANNEL_AUTO_DEF" },
   {          56, "MQCBC_LENGTH_1 (8 byte)" },
   {          56, "MQCSP_CURRENT_LENGTH (8 byte)" },
   {          56, "MQCSP_LENGTH_1 (8 byte)" },
   {          56, "MQDXP_CURRENT_LENGTH (8 byte)" },
   {          56, "MQDXP_LENGTH_2 (8 byte)" },
   {          56, "MQIA_CHANNEL_AUTO_DEF_EVENT" },
   {          56, "MQNXP_CURRENT_LENGTH (4 byte)" },
   {          56, "MQNXP_LENGTH_2 (4 byte)" },
   {          56, "MQZED_LENGTH_1 (4 byte)" },
   {          57, "MQIA_INDEX_TYPE" },
   {          58, "MQIA_CLUSTER_WORKLOAD_LENGTH" },
   {          59, "MQIA_CLUSTER_Q_TYPE" },
   {          60, "MQCMD_DELETE_PUBLICATION" },
   {          60, "MQIA_ARCHIVE" },
   {          60, "MQIMPO_CURRENT_LENGTH (4 byte)" },
   {          60, "MQIMPO_LENGTH_1 (4 byte)" },
   {          60, "MQZED_CURRENT_LENGTH (4 byte)" },
   {          60, "MQZED_LENGTH_2 (4 byte)" },
   {          61, "MQCMD_DEREGISTER_PUBLISHER" },
   {          61, "MQIA_DEF_BIND" },
   {          62, "MQCMD_DEREGISTER_SUBSCRIBER" },
   {          62, "MQIA_PAGESET_ID" },
   {          63, "MQCMD_PUBLISH" },
   {          63, "MQIA_QSG_DISP" },
   {          64, "MQAUTHOPT_NAME_AS_WILDCARD" },
   {          64, "MQCBC_CURRENT_LENGTH (8 byte)" },
   {          64, "MQCBC_LENGTH_2 (8 byte)" },
   {          64, "MQCBO_GROUP_BAG" },
   {          64, "MQCMD_REGISTER_PUBLISHER" },
   {          64, "MQCNO_HANDLE_SHARE_BLOCK" },
   {          64, "MQGMO_ACCEPT_TRUNCATED_MSG" },
   {          64, "MQIA_INTRA_GROUP_QUEUING" },
   {          64, "MQIMPO_CURRENT_LENGTH (8 byte)" },
   {          64, "MQIMPO_LENGTH_1 (8 byte)" },
   {          64, "MQNXP_LENGTH_1 (8 byte)" },
   {          64, "MQOO_SET" },
   {          64, "MQPMO_NEW_MSG_ID" },
   {          64, "MQQF_CLWL_USEQ_ANY" },
   {          64, "MQREGO_DEREGISTER_ALL" },
   {          64, "MQRO_PASS_CORREL_ID" },
   {          64, "MQSO_SET_IDENTITY_CONTEXT" },
   {          64, "MQTYPE_INT32" },
   {          64, "MQTYPE_LONG" },
   {          64, "MQZAO_PASS_IDENTITY_CONTEXT" },
   {          64, "MQZED_LENGTH_1 (8 byte)" },
   {          64, "MQ_APPL_DESC_LENGTH" },
   {          64, "MQ_AUTH_INFO_DESC_LENGTH" },
   {          64, "MQ_CERT_LABEL_LENGTH" },
   {          64, "MQ_CF_STRUC_DESC_LENGTH" },
   {          64, "MQ_CHANNEL_DESC_LENGTH" },
   {          64, "MQ_CHLAUTH_DESC_LENGTH" },
   {          64, "MQ_COMM_INFO_DESC_LENGTH" },
   {          64, "MQ_INSTALLATION_DESC_LENGTH" },
   {          64, "MQ_LISTENER_DESC_LENGTH" },
   {          64, "MQ_MAX_MCA_USER_ID_LENGTH" },
   {          64, "MQ_MAX_USER_ID_LENGTH" },
   {          64, "MQ_NAMELIST_DESC_LENGTH" },
   {          64, "MQ_PROCESS_DESC_LENGTH" },
   {          64, "MQ_Q_DESC_LENGTH" },
   {          64, "MQ_Q_MGR_DESC_LENGTH" },
   {          64, "MQ_SERVICE_DESC_LENGTH" },
   {          64, "MQ_STORAGE_CLASS_DESC_LENGTH" },
   {          64, "MQ_TOPIC_DESC_LENGTH" },
   {          64, "MQ_TP_NAME_LENGTH" },
   {          64, "MQ_TRIGGER_DATA_LENGTH" },
   {          65, "MQCMD_REGISTER_SUBSCRIBER" },
   {          65, "MQIA_IGQ_PUT_AUTHORITY" },
   {          65, "MQMON_HIGH" },
   {          66, "MQCMD_REQUEST_UPDATE" },
   {          66, "MQIA_AUTH_INFO_TYPE" },
   {          67, "MQCMD_BROKER_INTERNAL" },
   {          68, "MQACH_CURRENT_LENGTH (4 byte)" },
   {          68, "MQACH_LENGTH_1 (4 byte)" },
   {          68, "MQEPH_CURRENT_LENGTH" },
   {          68, "MQEPH_LENGTH_1" },
   {          68, "MQEPH_STRUC_LENGTH_FIXED" },
   {          68, "MQIA_MSG_MARK_BROWSE_INTERVAL" },
   {          69, "MQCMD_ACTIVITY_MSG" },
   {          69, "MQIA_SSL_TASKS" },
   {          70, "MQCMD_INQUIRE_CLUSTER_Q_MGR" },
   {          70, "MQIA_CF_LEVEL" },
   {          71, "MQCMD_RESUME_Q_MGR_CLUSTER" },
   {          71, "MQIA_CF_RECOVER" },
   {          72, "MQACH_CURRENT_LENGTH (8 byte)" },
   {          72, "MQACH_LENGTH_1 (8 byte)" },
   {          72, "MQCMD_SUSPEND_Q_MGR_CLUSTER" },
   {          72, "MQGMO_LENGTH_1" },
   {          72, "MQIA_NAMELIST_TYPE" },
   {          72, "MQMDE_CURRENT_LENGTH" },
   {          72, "MQMDE_LENGTH_2" },
   {          72, "MQNXP_CURRENT_LENGTH (8 byte)" },
   {          72, "MQNXP_LENGTH_2 (8 byte)" },
   {          72, "MQZAD_LENGTH_1 (4 byte)" },
   {          72, "MQZED_CURRENT_LENGTH (8 byte)" },
   {          72, "MQZED_LENGTH_2 (8 byte)" },
   {          73, "MQCMD_REFRESH_CLUSTER" },
   {          73, "MQIA_CHANNEL_EVENT" },
   {          74, "MQCMD_RESET_CLUSTER" },
   {          74, "MQIA_BRIDGE_EVENT" },
   {          75, "MQCMD_TRACE_ROUTE" },
   {          75, "MQIA_SSL_EVENT" },
   {          76, "MQIA_SSL_RESET_COUNT" },
   {          76, "MQZAD_CURRENT_LENGTH (4 byte)" },
   {          76, "MQZAD_LENGTH_2 (4 byte)" },
   {          77, "MQIA_SHARED_Q_Q_MGR_NAME" },
   {          78, "MQCMD_REFRESH_SECURITY" },
   {          78, "MQIA_NPM_CLASS" },
   {          79, "MQCMD_CHANGE_AUTH_INFO" },
   {          80, "MQCMD_COPY_AUTH_INFO" },
   {          80, "MQGMO_LENGTH_2" },
   {          80, "MQIA_MAX_OPEN_Q" },
   {          80, "MQMCEV_SHM_DEST_UNUSABLE" },
   {          80, "MQZAD_CURRENT_LENGTH (8 byte)" },
   {          80, "MQZAD_LENGTH_1 (8 byte)" },
   {          80, "MQZAD_LENGTH_2 (8 byte)" },
   {          81, "MQCMD_CREATE_AUTH_INFO" },
   {          81, "MQIA_MONITOR_INTERVAL" },
   {          81, "MQMCEV_SHM_PORT_UNUSABLE" },
   {          82, "MQCMD_DELETE_AUTH_INFO" },
   {          82, "MQIA_Q_USERS" },
   {          83, "MQCMD_INQUIRE_AUTH_INFO" },
   {          83, "MQIA_MAX_GLOBAL_LOCKS" },
   {          84, "MQCMD_INQUIRE_AUTH_INFO_NAMES" },
   {          84, "MQIA_MAX_LOCAL_LOCKS" },
   {          84, "MQIIH_CURRENT_LENGTH" },
   {          84, "MQIIH_LENGTH_1" },
   {          84, "MQZAC_CURRENT_LENGTH" },
   {          84, "MQZAC_LENGTH_1" },
   {          84, "MQZIC_CURRENT_LENGTH" },
   {          84, "MQZIC_LENGTH_1" },
   {          85, "MQCMD_INQUIRE_CONNECTION" },
   {          85, "MQIA_LISTENER_PORT_NUMBER" },
   {          86, "MQCMD_STOP_CONNECTION" },
   {          86, "MQIA_BATCH_INTERFACE_AUTO" },
   {          87, "MQCMD_INQUIRE_AUTH_RECS" },
   {          87, "MQIA_CMD_SERVER_AUTO" },
   {          88, "MQCMD_INQUIRE_ENTITY_AUTH" },
   {          88, "MQIA_CMD_SERVER_CONVERT_MSG" },
   {          89, "MQCMD_DELETE_AUTH_REC" },
   {          89, "MQIA_CMD_SERVER_DLQ_MSG" },
   {          90, "MQCMD_SET_AUTH_REC" },
   {          90, "MQIA_MAX_Q_TRIGGERS" },
   {          91, "MQCMD_LOGGER_EVENT" },
   {          91, "MQIA_TRIGGER_RESTART" },
   {          92, "MQCMD_RESET_Q_MGR" },
   {          92, "MQIA_SSL_FIPS_REQUIRED" },
   {          93, "MQCMD_CHANGE_LISTENER" },
   {          93, "MQIA_IP_ADDRESS_VERSION" },
   {          94, "MQCMD_COPY_LISTENER" },
   {          94, "MQIA_LOGGER_EVENT" },
   {          95, "MQCMD_CREATE_LISTENER" },
   {          95, "MQIA_CLWL_Q_RANK" },
   {          96, "MQCMD_DELETE_LISTENER" },
   {          96, "MQIA_CLWL_Q_PRIORITY" },
   {          96, "MQ_ENV_INFO_LENGTH" },
   {          97, "MQCMD_INQUIRE_LISTENER" },
   {          97, "MQIA_CLWL_MRU_CHANNELS" },
   {          98, "MQCMD_INQUIRE_LISTENER_STATUS" },
   {          98, "MQIA_CLWL_USEQ" },
   {          99, "MQCMD_COMMAND_EVENT" },
   {          99, "MQIA_COMMAND_EVENT" },
   {         100, "MQCHSSTATE_END_OF_BATCH" },
   {         100, "MQCMDL_LEVEL_1" },
   {         100, "MQCMD_CHANGE_SECURITY" },
   {         100, "MQGMO_LENGTH_3" },
   {         100, "MQIAMO_MONITOR_HUNDREDTHS" },
   {         100, "MQIA_ACTIVE_CHANNELS" },
   {         101, "MQCMDL_LEVEL_101" },
   {         101, "MQCMD_CHANGE_CF_STRUC" },
   {         101, "MQIA_CHINIT_ADAPTERS" },
   {         102, "MQCMD_CHANGE_STG_CLASS" },
   {         102, "MQIA_ADOPTNEWMCA_CHECK" },
   {         103, "MQCMD_CHANGE_TRACE" },
   {         103, "MQIA_ADOPTNEWMCA_TYPE" },
   {         104, "MQCMD_ARCHIVE_LOG" },
   {         104, "MQIA_ADOPTNEWMCA_INTERVAL" },
   {         105, "MQCMD_BACKUP_CF_STRUC" },
   {         105, "MQIA_CHINIT_DISPATCHERS" },
   {         106, "MQCMD_CREATE_BUFFER_POOL" },
   {         106, "MQIA_DNS_WLM" },
   {         107, "MQCMD_CREATE_PAGE_SET" },
   {         107, "MQIA_LISTENER_TIMER" },
   {         108, "MQCMD_CREATE_CF_STRUC" },
   {         108, "MQIA_LU62_CHANNELS" },
   {         108, "MQRMH_CURRENT_LENGTH" },
   {         108, "MQRMH_LENGTH_1" },
   {         109, "MQCMD_CREATE_STG_CLASS" },
   {         109, "MQIA_MAX_CHANNELS" },
   {         110, "MQCMDL_LEVEL_110" },
   {         110, "MQCMD_COPY_CF_STRUC" },
   {         110, "MQIA_OUTBOUND_PORT_MIN" },
   {         110, "MQMCEV_CCT_GETTIME_FAILED" },
   {         111, "MQCMD_COPY_STG_CLASS" },
   {         111, "MQIA_RECEIVE_TIMEOUT" },
   {         112, "MQCMD_DELETE_CF_STRUC" },
   {         112, "MQGMO_CURRENT_LENGTH" },
   {         112, "MQGMO_LENGTH_4" },
   {         112, "MQIA_RECEIVE_TIMEOUT_TYPE" },
   {         112, "MQMT_MQE_FIELDS_FROM_MQE" },
   {         113, "MQCMD_DELETE_STG_CLASS" },
   {         113, "MQIA_RECEIVE_TIMEOUT_MIN" },
   {         113, "MQMT_MQE_FIELDS" },
   {         114, "MQCMDL_LEVEL_114" },
   {         114, "MQCMD_INQUIRE_ARCHIVE" },
   {         114, "MQIA_TCP_CHANNELS" },
   {         115, "MQCMD_INQUIRE_CF_STRUC" },
   {         115, "MQIA_TCP_KEEP_ALIVE" },
   {         116, "MQCMD_INQUIRE_CF_STRUC_STATUS" },
   {         116, "MQIA_TCP_STACK_TYPE" },
   {         117, "MQCMD_INQUIRE_CMD_SERVER" },
   {         117, "MQIA_CHINIT_TRACE_AUTO_START" },
   {         118, "MQCMD_INQUIRE_CHANNEL_INIT" },
   {         118, "MQIA_CHINIT_TRACE_TABLE_SIZE" },
   {         119, "MQCMD_INQUIRE_QSG" },
   {         119, "MQIA_CHINIT_CONTROL" },
   {         120, "MQCMDL_LEVEL_120" },
   {         120, "MQCMD_INQUIRE_LOG" },
   {         120, "MQIA_CMD_SERVER_CONTROL" },
   {         120, "MQMCEV_DEST_INTERFACE_FAILURE" },
   {         120, "MQWIH_CURRENT_LENGTH" },
   {         120, "MQWIH_LENGTH_1" },
   {         121, "MQCMD_INQUIRE_SECURITY" },
   {         121, "MQIA_SERVICE_TYPE" },
   {         121, "MQMCEV_DEST_INTERFACE_FAILOVER" },
   {         122, "MQCMD_INQUIRE_STG_CLASS" },
   {         122, "MQIA_MONITORING_CHANNEL" },
   {         122, "MQMCEV_PORT_INTERFACE_FAILURE" },
   {         123, "MQCMD_INQUIRE_SYSTEM" },
   {         123, "MQIA_MONITORING_Q" },
   {         123, "MQMCEV_PORT_INTERFACE_FAILOVER" },
   {         124, "MQCMD_INQUIRE_THREAD" },
   {         124, "MQIA_MONITORING_AUTO_CLUSSDR" },
   {         124, "MQWDR1_CURRENT_LENGTH" },
   {         124, "MQWDR1_LENGTH_1" },
   {         124, "MQWDR2_LENGTH_1" },
   {         124, "MQWDR_LENGTH_1" },
   {         125, "MQCMD_INQUIRE_TRACE" },
   {         126, "MQCMD_INQUIRE_USAGE" },
   {         127, "MQCMD_MOVE_Q" },
   {         127, "MQIA_STATISTICS_MQI" },
   {         128, "MQCMD_RECOVER_BSDS" },
   {         128, "MQCNO_HANDLE_SHARE_NO_BLOCK" },
   {         128, "MQGMO_MARK_SKIP_BACKOUT" },
   {         128, "MQIA_STATISTICS_Q" },
   {         128, "MQOO_SAVE_ALL_CONTEXT" },
   {         128, "MQPMO_LENGTH_1" },
   {         128, "MQPMO_NEW_CORREL_ID" },
   {         128, "MQQF_CLWL_USEQ_LOCAL" },
   {         128, "MQREGO_INCLUDE_STREAM_NAME" },
   {         128, "MQRO_PASS_MSG_ID" },
   {         128, "MQSO_NO_MULTICAST" },
   {         128, "MQTYPE_INT64" },
   {         128, "MQZAO_PASS_ALL_CONTEXT" },
   {         128, "MQ_CONN_TAG_LENGTH" },
   {         128, "MQ_CUSTOM_LENGTH" },
   {         128, "MQ_EXIT_NAME_LENGTH" },
   {         128, "MQ_LDAP_CLASS_LENGTH" },
   {         128, "MQ_LDAP_FIELD_LENGTH" },
   {         128, "MQ_MAX_EXIT_NAME_LENGTH" },
   {         128, "MQ_PROCESS_ENV_DATA_LENGTH" },
   {         128, "MQ_PROCESS_USER_DATA_LENGTH" },
   {         128, "MQ_SUB_IDENTITY_LENGTH" },
   {         128, "MQ_SUB_POINT_LENGTH" },
   {         129, "MQCMD_RECOVER_CF_STRUC" },
   {         129, "MQIA_STATISTICS_CHANNEL" },
   {         130, "MQCMD_RESET_TPIPE" },
   {         130, "MQIA_STATISTICS_AUTO_CLUSSDR" },
   {         131, "MQCMD_RESOLVE_INDOUBT" },
   {         131, "MQIA_STATISTICS_INTERVAL" },
   {         132, "MQCMD_RESUME_Q_MGR" },
   {         133, "MQCMD_REVERIFY_SECURITY" },
   {         133, "MQIA_ACCOUNTING_MQI" },
   {         134, "MQCMD_SET_ARCHIVE" },
   {         134, "MQIA_ACCOUNTING_Q" },
   {         135, "MQIA_ACCOUNTING_INTERVAL" },
   {         136, "MQCMD_SET_LOG" },
   {         136, "MQIA_ACCOUNTING_CONN_OVERRIDE" },
   {         136, "MQWDR2_CURRENT_LENGTH" },
   {         136, "MQWDR2_LENGTH_2" },
   {         136, "MQWDR_CURRENT_LENGTH" },
   {         136, "MQWDR_LENGTH_2" },
   {         137, "MQCMD_SET_SYSTEM" },
   {         137, "MQIA_TRACE_ROUTE_RECORDING" },
   {         138, "MQCMD_START_CMD_SERVER" },
   {         138, "MQIA_ACTIVITY_RECORDING" },
   {         139, "MQCMD_START_Q_MGR" },
   {         139, "MQIA_SERVICE_CONTROL" },
   {         140, "MQCMD_START_TRACE" },
   {         140, "MQIA_OUTBOUND_PORT_MAX" },
   {         140, "MQIEP_CURRENT_LENGTH (4 byte)" },
   {         140, "MQIEP_LENGTH_1 (4 byte)" },
   {         141, "MQCMD_STOP_CHANNEL_INIT" },
   {         141, "MQIA_SECURITY_CASE" },
   {         142, "MQCMD_STOP_CHANNEL_LISTENER" },
   {         143, "MQCMD_STOP_CMD_SERVER" },
   {         144, "MQCMD_STOP_Q_MGR" },
   {         145, "MQCMD_STOP_TRACE" },
   {         146, "MQCMD_SUSPEND_Q_MGR" },
   {         147, "MQCMD_INQUIRE_CF_STRUC_NAMES" },
   {         148, "MQCMD_INQUIRE_STG_CLASS_NAMES" },
   {         148, "MQCNO_LENGTH_3 (4 byte)" },
   {         149, "MQCMD_CHANGE_SERVICE" },
   {         150, "MQCMD_COPY_SERVICE" },
   {         150, "MQIA_QMOPT_CSMT_ON_ERROR" },
   {         151, "MQCMD_CREATE_SERVICE" },
   {         151, "MQIA_QMOPT_CONS_INFO_MSGS" },
   {         152, "MQCMD_DELETE_SERVICE" },
   {         152, "MQCNO_LENGTH_3 (8 byte)" },
   {         152, "MQIA_QMOPT_CONS_WARNING_MSGS" },
   {         152, "MQPMO_LENGTH_2 (4 byte)" },
   {         153, "MQCMD_INQUIRE_SERVICE" },
   {         153, "MQIA_QMOPT_CONS_ERROR_MSGS" },
   {         154, "MQCMD_INQUIRE_SERVICE_STATUS" },
   {         154, "MQIA_QMOPT_CONS_CRITICAL_MSGS" },
   {         155, "MQCMD_START_SERVICE" },
   {         155, "MQIA_QMOPT_CONS_COMMS_MSGS" },
   {         156, "MQCBD_CURRENT_LENGTH (4 byte)" },
   {         156, "MQCBD_LENGTH_1 (4 byte)" },
   {         156, "MQCMD_STOP_SERVICE" },
   {         156, "MQCNO_LENGTH_4 (4 byte)" },
   {         156, "MQCXP_LENGTH_3" },
   {         156, "MQCXP_LENGTH_4" },
   {         156, "MQIA_QMOPT_CONS_REORG_MSGS" },
   {         156, "MQPSXP_LENGTH_1 (4 byte)" },
   {         157, "MQCMD_DELETE_BUFFER_POOL" },
   {         157, "MQIA_QMOPT_CONS_SYSTEM_MSGS" },
   {         158, "MQCMD_DELETE_PAGE_SET" },
   {         158, "MQIA_QMOPT_LOG_INFO_MSGS" },
   {         159, "MQCMD_CHANGE_BUFFER_POOL" },
   {         159, "MQIA_QMOPT_LOG_WARNING_MSGS" },
   {         160, "MQCMD_CHANGE_PAGE_SET" },
   {         160, "MQCXP_LENGTH_5" },
   {         160, "MQIA_QMOPT_LOG_ERROR_MSGS" },
   {         160, "MQPMO_LENGTH_2 (8 byte)" },
   {         160, "MQPSXP_CURRENT_LENGTH (4 byte)" },
   {         160, "MQPSXP_LENGTH_2 (4 byte)" },
   {         161, "MQCMD_INQUIRE_Q_MGR_STATUS" },
   {         161, "MQIA_QMOPT_LOG_CRITICAL_MSGS" },
   {         162, "MQCMD_CREATE_LOG" },
   {         162, "MQIA_QMOPT_LOG_COMMS_MSGS" },
   {         163, "MQIA_QMOPT_LOG_REORG_MSGS" },
   {         164, "MQCIH_LENGTH_1" },
   {         164, "MQCMD_STATISTICS_MQI" },
   {         164, "MQIA_QMOPT_LOG_SYSTEM_MSGS" },
   {         165, "MQCMD_STATISTICS_Q" },
   {         165, "MQIA_QMOPT_TRACE_MQI_CALLS" },
   {         166, "MQCMD_STATISTICS_CHANNEL" },
   {         166, "MQIA_QMOPT_TRACE_COMMS" },
   {         167, "MQCMD_ACCOUNTING_MQI" },
   {         167, "MQIA_QMOPT_TRACE_REORG" },
   {         168, "MQCBD_CURRENT_LENGTH (8 byte)" },
   {         168, "MQCBD_LENGTH_1 (8 byte)" },
   {         168, "MQCMD_ACCOUNTING_Q" },
   {         168, "MQCNO_LENGTH_4 (8 byte)" },
   {         168, "MQIA_QMOPT_TRACE_CONVERSION" },
   {         168, "MQOD_LENGTH_1" },
   {         169, "MQCMD_INQUIRE_AUTH_SERVICE" },
   {         169, "MQIA_QMOPT_TRACE_SYSTEM" },
   {         170, "MQCMD_CHANGE_TOPIC" },
   {         170, "MQIA_QMOPT_INTERNAL_DUMP" },
   {         171, "MQCMD_COPY_TOPIC" },
   {         171, "MQIA_MAX_RECOVERY_TASKS" },
   {         172, "MQCMD_CREATE_TOPIC" },
   {         172, "MQDLH_CURRENT_LENGTH" },
   {         172, "MQDLH_LENGTH_1" },
   {         172, "MQIA_MAX_CLIENTS" },
   {         173, "MQCMD_DELETE_TOPIC" },
   {         173, "MQIA_AUTO_REORGANIZATION" },
   {         174, "MQCMD_INQUIRE_TOPIC" },
   {         174, "MQIA_AUTO_REORG_INTERVAL" },
   {         175, "MQCMD_INQUIRE_TOPIC_NAMES" },
   {         175, "MQIA_DURABLE_SUB" },
   {         176, "MQCMD_INQUIRE_SUBSCRIPTION" },
   {         176, "MQIA_MULTICAST" },
   {         176, "MQPMO_CURRENT_LENGTH (4 byte)" },
   {         176, "MQPMO_LENGTH_3 (4 byte)" },
   {         176, "MQPSXP_LENGTH_1 (8 byte)" },
   {         177, "MQCMD_CREATE_SUBSCRIPTION" },
   {         178, "MQCMD_CHANGE_SUBSCRIPTION" },
   {         179, "MQCMD_DELETE_SUBSCRIPTION" },
   {         180, "MQCIH_CURRENT_LENGTH" },
   {         180, "MQCIH_LENGTH_2" },
   {         181, "MQCMD_COPY_SUBSCRIPTION" },
   {         181, "MQIA_INHIBIT_PUB" },
   {         182, "MQCMD_INQUIRE_SUB_STATUS" },
   {         182, "MQIA_INHIBIT_SUB" },
   {         183, "MQCMD_INQUIRE_TOPIC_STATUS" },
   {         183, "MQIA_TREE_LIFE_TIME" },
   {         184, "MQCMD_CLEAR_TOPIC_STRING" },
   {         184, "MQIA_DEF_PUT_RESPONSE_TYPE" },
   {         184, "MQPMO_CURRENT_LENGTH (8 byte)" },
   {         184, "MQPMO_LENGTH_3 (8 byte)" },
   {         184, "MQPSXP_CURRENT_LENGTH (8 byte)" },
   {         184, "MQPSXP_LENGTH_2 (8 byte)" },
   {         185, "MQCMD_INQUIRE_PUBSUB_STATUS" },
   {         185, "MQIA_TOPIC_DEF_PERSISTENCE" },
   {         186, "MQCMD_INQUIRE_SMDS" },
   {         186, "MQIA_MASTER_ADMIN" },
   {         187, "MQCMD_CHANGE_SMDS" },
   {         187, "MQIA_PUBSUB_MODE" },
   {         188, "MQCMD_RESET_SMDS" },
   {         188, "MQCNO_LENGTH_5 (4 byte)" },
   {         188, "MQIA_DEF_READ_AHEAD" },
   {         189, "MQIA_READ_AHEAD" },
   {         190, "MQCMD_CREATE_COMM_INFO" },
   {         190, "MQIA_PROPERTY_CONTROL" },
   {         191, "MQCMD_INQUIRE_COMM_INFO" },
   {         192, "MQCMD_CHANGE_COMM_INFO" },
   {         192, "MQCXP_LENGTH_6 (4 byte)" },
   {         192, "MQIA_MAX_PROPERTIES_LENGTH" },
   {         193, "MQCMD_COPY_COMM_INFO" },
   {         193, "MQIA_BASE_TYPE" },
   {         194, "MQCMD_DELETE_COMM_INFO" },
   {         195, "MQCMD_PURGE_CHANNEL" },
   {         195, "MQIA_PM_DELIVERY" },
   {         196, "MQCMD_MQXR_DIAGNOSTICS" },
   {         196, "MQIA_NPM_DELIVERY" },
   {         197, "MQCMD_START_SMDSCONN" },
   {         198, "MQCMD_STOP_SMDSCONN" },
   {         199, "MQCMD_INQUIRE_SMDSCONN" },
   {         199, "MQIA_PROXY_SUB" },
   {         200, "MQCHSSTATE_SENDING" },
   {         200, "MQCMDL_LEVEL_200" },
   {         200, "MQCMD_INQUIRE_MQXR_STATUS" },
   {         200, "MQCNO_LENGTH_5 (8 byte)" },
   {         200, "MQCXP_LENGTH_6 (8 byte)" },
   {         200, "MQCXP_LENGTH_7 (4 byte)" },
   {         200, "MQOD_LENGTH_2 (4 byte)" },
   {         200, "MQWQR1_CURRENT_LENGTH" },
   {         200, "MQWQR1_LENGTH_1" },
   {         200, "MQWQR2_LENGTH_1" },
   {         200, "MQWQR3_LENGTH_1" },
   {         200, "MQWQR_LENGTH_1" },
   {         201, "MQCMDL_LEVEL_201" },
   {         201, "MQCMD_START_CLIENT_TRACE" },
   {         202, "MQCMD_STOP_CLIENT_TRACE" },
   {         203, "MQCMD_SET_CHLAUTH_REC" },
   {         203, "MQIA_PUBSUB_NP_MSG" },
   {         204, "MQCMD_INQUIRE_CHLAUTH_RECS" },
   {         204, "MQIA_SUB_COUNT" },
   {         205, "MQCMD_INQUIRE_PROT_POLICY" },
   {         205, "MQIA_PUBSUB_NP_RESP" },
   {         206, "MQCMD_CREATE_PROT_POLICY" },
   {         206, "MQIA_PUBSUB_MAXMSG_RETRY_COUNT" },
   {         207, "MQCMD_DELETE_PROT_POLICY" },
   {         207, "MQIA_PUBSUB_SYNC_PT" },
   {         208, "MQCMD_CHANGE_PROT_POLICY" },
   {         208, "MQCMD_SET_PROT_POLICY" },
   {         208, "MQCNO_CURRENT_LENGTH (4 byte)" },
   {         208, "MQCNO_LENGTH_6 (4 byte)" },
   {         208, "MQCXP_LENGTH_7 (8 byte)" },
   {         208, "MQCXP_LENGTH_8 (4 byte)" },
   {         208, "MQIA_TOPIC_TYPE" },
   {         208, "MQOD_LENGTH_2 (8 byte)" },
   {         208, "MQWQR2_CURRENT_LENGTH" },
   {         208, "MQWQR2_LENGTH_2" },
   {         208, "MQWQR3_LENGTH_2" },
   {         208, "MQWQR_LENGTH_2" },
   {         208, "MQWXP1_CURRENT_LENGTH (4 byte)" },
   {         208, "MQWXP1_LENGTH_1 (4 byte)" },
   {         208, "MQWXP2_LENGTH_1 (4 byte)" },
   {         208, "MQWXP3_LENGTH_1 (4 byte)" },
   {         208, "MQWXP4_LENGTH_1 (4 byte)" },
   {         208, "MQWXP_LENGTH_1 (4 byte)" },
   {         209, "MQCMD_ACTIVITY_TRACE" },
   {         210, "MQCMDL_LEVEL_210" },
   {         211, "MQCMDL_LEVEL_211" },
   {         212, "MQWQR3_CURRENT_LENGTH" },
   {         212, "MQWQR3_LENGTH_3" },
   {         212, "MQWQR_CURRENT_LENGTH" },
   {         212, "MQWQR_LENGTH_3" },
   {         213, "MQCMD_RESET_CF_STRUC" },
   {         214, "MQCMD_INQUIRE_XR_CAPABILITY" },
   {         215, "MQIA_PUB_COUNT" },
   {         216, "MQCMD_INQUIRE_AMQP_CAPABILITY" },
   {         216, "MQIA_WILDCARD_OPERATION" },
   {         216, "MQWXP2_CURRENT_LENGTH (4 byte)" },
   {         216, "MQWXP2_LENGTH_2 (4 byte)" },
   {         216, "MQWXP3_LENGTH_2 (4 byte)" },
   {         216, "MQWXP4_LENGTH_2 (4 byte)" },
   {         216, "MQWXP_LENGTH_2 (4 byte)" },
   {         217, "MQCMD_AMQP_DIAGNOSTICS" },
   {         218, "MQIA_SUB_SCOPE" },
   {         219, "MQIA_PUB_SCOPE" },
   {         220, "MQCMDL_LEVEL_220" },
   {         220, "MQCXP_CURRENT_LENGTH (4 byte)" },
   {         220, "MQCXP_LENGTH_9 (4 byte)" },
   {         220, "MQWXP3_CURRENT_LENGTH (4 byte)" },
   {         220, "MQWXP3_LENGTH_3 (4 byte)" },
   {         220, "MQWXP4_LENGTH_3 (4 byte)" },
   {         220, "MQWXP_LENGTH_3 (4 byte)" },
   {         221, "MQCMDL_LEVEL_221" },
   {         221, "MQIA_GROUP_UR" },
   {         222, "MQIA_UR_DISP" },
   {         223, "MQIA_COMM_INFO_TYPE" },
   {         224, "MQCNO_CURRENT_LENGTH (8 byte)" },
   {         224, "MQCNO_LENGTH_6 (8 byte)" },
   {         224, "MQCXP_LENGTH_8 (8 byte)" },
   {         224, "MQIA_CF_OFFLOAD" },
   {         224, "MQSTS_LENGTH_1" },
   {         224, "MQWXP1_CURRENT_LENGTH (8 byte)" },
   {         224, "MQWXP1_LENGTH_1 (8 byte)" },
   {         224, "MQWXP2_LENGTH_1 (8 byte)" },
   {         224, "MQWXP3_LENGTH_1 (8 byte)" },
   {         224, "MQWXP4_CURRENT_LENGTH (4 byte)" },
   {         224, "MQWXP4_LENGTH_1 (8 byte)" },
   {         224, "MQWXP4_LENGTH_4 (4 byte)" },
   {         224, "MQWXP_CURRENT_LENGTH (4 byte)" },
   {         224, "MQWXP_LENGTH_1 (8 byte)" },
   {         224, "MQWXP_LENGTH_4 (4 byte)" },
   {         225, "MQIA_CF_OFFLOAD_THRESHOLD1" },
   {         226, "MQIA_CF_OFFLOAD_THRESHOLD2" },
   {         227, "MQIA_CF_OFFLOAD_THRESHOLD3" },
   {         228, "MQIA_CF_SMDS_BUFFERS" },
   {         229, "MQIA_CF_OFFLDUSE" },
   {         230, "MQCMDL_LEVEL_230" },
   {         230, "MQIA_MAX_RESPONSES" },
   {         231, "MQIA_RESPONSE_RESTART_POINT" },
   {         232, "MQIA_COMM_EVENT" },
   {         233, "MQIA_MCAST_BRIDGE" },
   {         234, "MQIA_USE_DEAD_LETTER_Q" },
   {         235, "MQIA_TOLERATE_UNPROTECTED" },
   {         236, "MQIA_SIGNATURE_ALGORITHM" },
   {         237, "MQIA_ENCRYPTION_ALGORITHM" },
   {         238, "MQIA_POLICY_VERSION" },
   {         239, "MQIA_ACTIVITY_CONN_OVERRIDE" },
   {         240, "MQCXP_CURRENT_LENGTH (8 byte)" },
   {         240, "MQCXP_LENGTH_9 (8 byte)" },
   {         240, "MQDCC_SOURCE_ENC_MASK" },
   {         240, "MQENC_DECIMAL_MASK" },
   {         240, "MQIA_ACTIVITY_TRACE" },
   {         240, "MQWXP2_CURRENT_LENGTH (8 byte)" },
   {         240, "MQWXP2_LENGTH_2 (8 byte)" },
   {         240, "MQWXP3_CURRENT_LENGTH (8 byte)" },
   {         240, "MQWXP3_LENGTH_2 (8 byte)" },
   {         240, "MQWXP3_LENGTH_3 (8 byte)" },
   {         240, "MQWXP4_LENGTH_2 (8 byte)" },
   {         240, "MQWXP4_LENGTH_3 (8 byte)" },
   {         240, "MQWXP_LENGTH_2 (8 byte)" },
   {         240, "MQWXP_LENGTH_3 (8 byte)" },
   {         242, "MQIA_SUB_CONFIGURATION_EVENT" },
   {         243, "MQIA_XR_CAPABILITY" },
   {         244, "MQAXP_CURRENT_LENGTH (4 byte)" },
   {         244, "MQAXP_LENGTH_1 (4 byte)" },
   {         244, "MQIA_CF_RECAUTO" },
   {         245, "MQIA_QMGR_CFCONLOS" },
   {         246, "MQIA_CF_CFCONLOS" },
   {         247, "MQIA_SUITE_B_STRENGTH" },
   {         248, "MQIA_CHLAUTH_RECORDS" },
   {         248, "MQWXP4_CURRENT_LENGTH (8 byte)" },
   {         248, "MQWXP4_LENGTH_4 (8 byte)" },
   {         248, "MQWXP_CURRENT_LENGTH (8 byte)" },
   {         248, "MQWXP_LENGTH_4 (8 byte)" },
   {         249, "MQIA_PUBSUB_CLUSTER" },
   {         250, "MQIA_DEF_CLUSTER_XMIT_Q_TYPE" },
   {         251, "MQIA_PROT_POLICY_CAPABILITY" },
   {         252, "MQIA_CERT_VAL_POLICY" },
   {         253, "MQIA_TOPIC_NODE_COUNT" },
   {         254, "MQIA_REVERSE_DNS_LOOKUP" },
   {         255, "MQIA_CLUSTER_PUB_ROUTE" },
   {         255, "MQ_SERVICE_ARGS_LENGTH" },
   {         255, "MQ_SERVICE_COMMAND_LENGTH" },
   {         255, "MQ_SERVICE_PATH_LENGTH" },
   {         256, "MQAUTHOPT_CUMULATIVE" },
   {         256, "MQAXP_CURRENT_LENGTH (8 byte)" },
   {         256, "MQAXP_LENGTH_1 (8 byte)" },
   {         256, "MQCADSD_MSGFORMAT" },
   {         256, "MQCBDO_REGISTER_CALL" },
   {         256, "MQCNO_SHARED_BINDING" },
   {         256, "MQCTES_COMMIT" },
   {         256, "MQCUOWC_COMMIT" },
   {         256, "MQDCC_TARGET_ENC_FACTOR" },
   {         256, "MQDCC_TARGET_ENC_NORMAL" },
   {         256, "MQENC_FLOAT_IEEE_NORMAL" },
   {         256, "MQFB_QUIT" },
   {         256, "MQGMO_MSG_UNDER_CURSOR" },
   {         256, "MQIA_CLUSTER_OBJECT_STATE" },
   {         256, "MQNC_MAX_NAMELIST_NAME_COUNT" },
   {         256, "MQOO_PASS_IDENTITY_CONTEXT" },
   {         256, "MQOP_REGISTER" },
   {         256, "MQPMO_PASS_IDENTITY_CONTEXT" },
   {         256, "MQREGO_INFORM_IF_RETAINED" },
   {         256, "MQROUTE_FORWARD_ALL" },
   {         256, "MQRO_COA" },
   {         256, "MQSO_FIXED_USERID" },
   {         256, "MQTYPE_FLOAT32" },
   {         256, "MQZAO_SET_IDENTITY_CONTEXT" },
   {         256, "MQ_AMQP_CLIENT_ID_LENGTH" },
   {         256, "MQ_AUTH_INFO_OCSP_URL_LENGTH" },
   {         256, "MQ_CSP_PASSWORD_LENGTH" },
   {         256, "MQ_INSTALLATION_PATH_LENGTH" },
   {         256, "MQ_PROCESS_APPL_ID_LENGTH" },
   {         256, "MQ_SHORT_DNAME_LENGTH" },
   {         256, "MQ_SSL_CRYPTO_HARDWARE_LENGTH" },
   {         256, "MQ_SSL_KEY_REPOSITORY_LENGTH" },
   {         256, "MQ_SSL_SHORT_PEER_NAME_LENGTH" },
   {         256, "MQ_UOW_ID_LENGTH" },
   {         257, "MQIA_CHECK_LOCAL_BINDING" },
   {         258, "MQFB_EXPIRATION" },
   {         258, "MQIA_CHECK_CLIENT_BINDING" },
   {         259, "MQFB_COA" },
   {         259, "MQIA_AUTHENTICATION_FAIL_DELAY" },
   {         260, "MQFB_COD" },
   {         260, "MQIA_ADOPT_CONTEXT" },
   {         261, "MQIA_LDAP_SECURE_COMM" },
   {         262, "MQFB_CHANNEL_COMPLETED" },
   {         262, "MQIA_DISPLAY_TYPE" },
   {         263, "MQFB_CHANNEL_FAIL_RETRY" },
   {         263, "MQIA_LDAP_AUTHORMD" },
   {         264, "MQFB_CHANNEL_FAIL" },
   {         264, "MQIA_LDAP_NESTGRP" },
   {         264, "MQIEP_CURRENT_LENGTH (8 byte)" },
   {         264, "MQIEP_LENGTH_1 (8 byte)" },
   {         264, "MQ_AUTH_INFO_CONN_NAME_LENGTH" },
   {         264, "MQ_CONN_NAME_LENGTH" },
   {         264, "MQ_GROUP_ADDRESS_LENGTH" },
   {         265, "MQFB_APPL_CANNOT_BE_STARTED" },
   {         265, "MQIA_AMQP_CAPABILITY" },
   {         266, "MQFB_TM_ERROR" },
   {         266, "MQIA_AUTHENTICATION_METHOD" },
   {         267, "MQFB_APPL_TYPE_ERROR" },
   {         267, "MQIA_KEY_REUSE_COUNT" },
   {         268, "MQFB_STOPPED_BY_MSG_EXIT" },
   {         268, "MQIA_MEDIA_IMAGE_SCHEDULING" },
   {         269, "MQFB_ACTIVITY" },
   {         269, "MQIA_MEDIA_IMAGE_INTERVAL" },
   {         270, "MQIA_MEDIA_IMAGE_LOG_LENGTH" },
   {         271, "MQFB_XMIT_Q_MSG_ERROR" },
   {         271, "MQIA_MEDIA_IMAGE_RECOVER_OBJ" },
   {         272, "MQCUOWC_LAST" },
   {         272, "MQIA_MEDIA_IMAGE_RECOVER_Q" },
   {         272, "MQSBC_CURRENT_LENGTH (4 byte)" },
   {         272, "MQSBC_LENGTH_1 (4 byte)" },
   {         272, "MQSTS_CURRENT_LENGTH (4 byte)" },
   {         272, "MQSTS_LENGTH_2 (4 byte)" },
   {         273, "MQCUOWC_ONLY" },
   {         273, "MQENC_NORMAL" },
   {         273, "MQIA_ADVANCED_CAPABILITY" },
   {         273, "MQIA_LAST_USED" },
   {         275, "MQFB_PAN" },
   {         276, "MQFB_NAN" },
   {         277, "MQFB_STOPPED_BY_CHAD_EXIT" },
   {         279, "MQFB_STOPPED_BY_PUBSUB_EXIT" },
   {         280, "MQFB_NOT_A_REPOSITORY_MSG" },
   {         280, "MQSTS_CURRENT_LENGTH (8 byte)" },
   {         280, "MQSTS_LENGTH_2 (8 byte)" },
   {         281, "MQFB_BIND_OPEN_CLUSRCVR_DEL" },
   {         282, "MQFB_MAX_ACTIVITIES" },
   {         283, "MQFB_NOT_FORWARDED" },
   {         284, "MQFB_NOT_DELIVERED" },
   {         285, "MQFB_UNSUPPORTED_FORWARDING" },
   {         286, "MQFB_UNSUPPORTED_DELIVERY" },
   {         288, "MQSBC_CURRENT_LENGTH (8 byte)" },
   {         288, "MQSBC_LENGTH_1 (8 byte)" },
   {         291, "MQFB_DATA_LENGTH_ZERO" },
   {         292, "MQFB_DATA_LENGTH_NEGATIVE" },
   {         293, "MQFB_DATA_LENGTH_TOO_BIG" },
   {         294, "MQFB_BUFFER_OVERFLOW" },
   {         295, "MQFB_LENGTH_OFF_BY_ONE" },
   {         296, "MQFB_IIH_ERROR" },
   {         298, "MQFB_NOT_AUTHORIZED_FOR_IMS" },
   {         300, "MQCHSSTATE_RECEIVING" },
   {         300, "MQFB_IMS_ERROR" },
   {         301, "MQFB_IMS_FIRST" },
   {         312, "MQSD_CURRENT_LENGTH (4 byte)" },
   {         312, "MQSD_LENGTH_1 (4 byte)" },
   {         320, "MQAIR_LENGTH_1 (4 byte)" },
   {         320, "MQCMDL_LEVEL_320" },
   {         324, "MQMD1_CURRENT_LENGTH" },
   {         324, "MQMD1_LENGTH_1" },
   {         324, "MQMD2_LENGTH_1" },
   {         324, "MQMD_LENGTH_1" },
   {         328, "MQAIR_LENGTH_1 (8 byte)" },
   {         336, "MQOD_LENGTH_3 (4 byte)" },
   {         344, "MQOD_LENGTH_3 (8 byte)" },
   {         344, "MQSD_CURRENT_LENGTH (8 byte)" },
   {         344, "MQSD_LENGTH_1 (8 byte)" },
   {         364, "MQMD2_CURRENT_LENGTH" },
   {         364, "MQMD2_LENGTH_2" },
   {         364, "MQMD_CURRENT_LENGTH" },
   {         364, "MQMD_LENGTH_2" },
   {         384, "MQAXC_LENGTH_1 (4 byte)" },
   {         392, "MQAXC_LENGTH_1 (8 byte)" },
   {         399, "MQFB_IMS_LAST" },
   {         400, "MQCHSSTATE_SERIALIZING" },
   {         400, "MQOD_CURRENT_LENGTH (4 byte)" },
   {         400, "MQOD_LENGTH_4 (4 byte)" },
   {         401, "MQFB_CICS_INTERNAL_ERROR" },
   {         402, "MQFB_CICS_NOT_AUTHORIZED" },
   {         403, "MQFB_CICS_BRIDGE_FAILURE" },
   {         404, "MQFB_CICS_CORREL_ID_ERROR" },
   {         405, "MQFB_CICS_CCSID_ERROR" },
   {         406, "MQFB_CICS_ENCODING_ERROR" },
   {         407, "MQFB_CICS_CIH_ERROR" },
   {         408, "MQFB_CICS_UOW_ERROR" },
   {         409, "MQFB_CICS_COMMAREA_ERROR" },
   {         410, "MQFB_CICS_APPL_NOT_STARTED" },
   {         411, "MQFB_CICS_APPL_ABENDED" },
   {         412, "MQAXC_CURRENT_LENGTH (4 byte)" },
   {         412, "MQAXC_LENGTH_2 (4 byte)" },
   {         412, "MQFB_CICS_DLQ_ERROR" },
   {         413, "MQFB_CICS_UOW_BACKED_OUT" },
   {         420, "MQCMDL_LEVEL_420" },
   {         424, "MQAXC_CURRENT_LENGTH (8 byte)" },
   {         424, "MQAXC_LENGTH_2 (8 byte)" },
   {         424, "MQOD_CURRENT_LENGTH (8 byte)" },
   {         424, "MQOD_LENGTH_4 (8 byte)" },
   {         428, "MQXQH_CURRENT_LENGTH" },
   {         428, "MQXQH_LENGTH_1" },
   {         500, "MQCHSSTATE_RESYNCHING" },
   {         500, "MQCMDL_LEVEL_500" },
   {         501, "MQFB_PUBLICATIONS_ON_REQUEST" },
   {         502, "MQFB_SUBSCRIBER_IS_PUBLISHER" },
   {         503, "MQFB_MSG_SCOPE_MISMATCH" },
   {         504, "MQFB_SELECTOR_MISMATCH" },
   {         505, "MQFB_NOT_A_GROUPUR_MSG" },
   {         510, "MQCMDL_LEVEL_510" },
   {         512, "MQAUTHOPT_EXCLUDE_TEMP" },
   {         512, "MQCBDO_DEREGISTER_CALL" },
   {         512, "MQCNO_ISOLATED_BINDING" },
   {         512, "MQDCC_TARGET_ENC_NATIVE" },
   {         512, "MQDCC_TARGET_ENC_REVERSED" },
   {         512, "MQENC_FLOAT_IEEE_REVERSED" },
   {         512, "MQGMO_LOCK" },
   {         512, "MQOO_PASS_ALL_CONTEXT" },
   {         512, "MQOP_DEREGISTER" },
   {         512, "MQPMO_PASS_ALL_CONTEXT" },
   {         512, "MQREGO_DUPLICATES_OK" },
   {         512, "MQROUTE_FORWARD_IF_SUPPORTED" },
   {         512, "MQSO_ANY_USERID" },
   {         512, "MQTYPE_FLOAT64" },
   {         512, "MQZAO_SET_ALL_CONTEXT" },
   {         520, "MQCMDL_LEVEL_520" },
   {         530, "MQCMDL_LEVEL_530" },
   {         531, "MQCMDL_LEVEL_531" },
   {         532, "MQSCO_LENGTH_1 (4 byte)" },
   {         536, "MQSCO_LENGTH_1 (8 byte)" },
   {         540, "MQSCO_LENGTH_2 (4 byte)" },
   {         544, "MQSCO_LENGTH_2 (8 byte)" },
   {         546, "MQENC_NATIVE" },
   {         546, "MQENC_REVERSED" },
   {         556, "MQSCO_LENGTH_3 (4 byte)" },
   {         560, "MQSCO_LENGTH_3 (8 byte)" },
   {         560, "MQSCO_LENGTH_4 (4 byte)" },
   {         568, "MQSCO_LENGTH_4 (8 byte)" },
   {         576, "MQAIR_CURRENT_LENGTH (4 byte)" },
   {         576, "MQAIR_LENGTH_2 (4 byte)" },
   {         584, "MQAIR_CURRENT_LENGTH (8 byte)" },
   {         584, "MQAIR_LENGTH_2 (8 byte)" },
   {         600, "MQCHSSTATE_HEARTBEATING" },
   {         600, "MQCMDL_LEVEL_600" },
   {         600, "MQFB_IMS_NACK_1A_REASON_FIRST" },
   {         624, "MQSCO_CURRENT_LENGTH (4 byte)" },
   {         624, "MQSCO_LENGTH_5 (4 byte)" },
   {         632, "MQSCO_CURRENT_LENGTH (8 byte)" },
   {         632, "MQSCO_LENGTH_5 (8 byte)" },
   {         684, "MQTMC2_LENGTH_1" },
   {         684, "MQTMC_CURRENT_LENGTH" },
   {         684, "MQTMC_LENGTH_1" },
   {         684, "MQTM_CURRENT_LENGTH" },
   {         684, "MQTM_LENGTH_1" },
   {         700, "MQCHSSTATE_IN_SCYEXIT" },
   {         700, "MQCMDL_LEVEL_700" },
   {         701, "MQCMDL_LEVEL_701" },
   {         701, "MQIAMO_FIRST" },
   {         702, "MQIAMO_AVG_BATCH_SIZE" },
   {         703, "MQIAMO64_AVG_Q_TIME" },
   {         703, "MQIAMO_AVG_Q_TIME" },
   {         704, "MQIAMO_BACKOUTS" },
   {         705, "MQIAMO_BROWSES" },
   {         706, "MQIAMO_BROWSE_MAX_BYTES" },
   {         707, "MQIAMO_BROWSE_MIN_BYTES" },
   {         708, "MQIAMO_BROWSES_FAILED" },
   {         709, "MQIAMO_CLOSES" },
   {         710, "MQCMDL_LEVEL_710" },
   {         710, "MQIAMO_COMMITS" },
   {         711, "MQCMDL_LEVEL_711" },
   {         711, "MQIAMO_COMMITS_FAILED" },
   {         712, "MQIAMO_CONNS" },
   {         713, "MQIAMO_CONNS_MAX" },
   {         714, "MQIAMO_DISCS" },
   {         715, "MQIAMO_DISCS_IMPLICIT" },
   {         716, "MQIAMO_DISC_TYPE" },
   {         717, "MQIAMO_EXIT_TIME_AVG" },
   {         718, "MQIAMO_EXIT_TIME_MAX" },
   {         719, "MQIAMO_EXIT_TIME_MIN" },
   {         720, "MQIAMO_FULL_BATCHES" },
   {         721, "MQIAMO_GENERATED_MSGS" },
   {         722, "MQIAMO_GETS" },
   {         723, "MQIAMO_GET_MAX_BYTES" },
   {         724, "MQIAMO_GET_MIN_BYTES" },
   {         725, "MQIAMO_GETS_FAILED" },
   {         726, "MQIAMO_INCOMPLETE_BATCHES" },
   {         727, "MQIAMO_INQS" },
   {         728, "MQIAMO_MSGS" },
   {         729, "MQIAMO_NET_TIME_AVG" },
   {         730, "MQIAMO_NET_TIME_MAX" },
   {         731, "MQIAMO_NET_TIME_MIN" },
   {         732, "MQIAMO_OBJECT_COUNT" },
   {         732, "MQTMC2_CURRENT_LENGTH" },
   {         732, "MQTMC2_LENGTH_2" },
   {         733, "MQIAMO_OPENS" },
   {         734, "MQIAMO_PUT1S" },
   {         735, "MQIAMO_PUTS" },
   {         736, "MQIAMO_PUT_MAX_BYTES" },
   {         737, "MQIAMO_PUT_MIN_BYTES" },
   {         738, "MQIAMO_PUT_RETRIES" },
   {         739, "MQIAMO_Q_MAX_DEPTH" },
   {         740, "MQIAMO_Q_MIN_DEPTH" },
   {         741, "MQIAMO64_Q_TIME_AVG" },
   {         741, "MQIAMO_Q_TIME_AVG" },
   {         742, "MQIAMO64_Q_TIME_MAX" },
   {         742, "MQIAMO_Q_TIME_MAX" },
   {         743, "MQIAMO64_Q_TIME_MIN" },
   {         743, "MQIAMO_Q_TIME_MIN" },
   {         744, "MQIAMO_SETS" },
   {         745, "MQIAMO64_BROWSE_BYTES" },
   {         746, "MQIAMO64_BYTES" },
   {         747, "MQIAMO64_GET_BYTES" },
   {         748, "MQIAMO64_PUT_BYTES" },
   {         749, "MQIAMO_CONNS_FAILED" },
   {         750, "MQCMDL_LEVEL_750" },
   {         751, "MQIAMO_OPENS_FAILED" },
   {         752, "MQIAMO_INQS_FAILED" },
   {         753, "MQIAMO_SETS_FAILED" },
   {         754, "MQIAMO_PUTS_FAILED" },
   {         755, "MQIAMO_PUT1S_FAILED" },
   {         757, "MQIAMO_CLOSES_FAILED" },
   {         758, "MQIAMO_MSGS_EXPIRED" },
   {         759, "MQIAMO_MSGS_NOT_QUEUED" },
   {         760, "MQIAMO_MSGS_PURGED" },
   {         764, "MQIAMO_SUBS_DUR" },
   {         765, "MQIAMO_SUBS_NDUR" },
   {         766, "MQIAMO_SUBS_FAILED" },
   {         767, "MQIAMO_SUBRQS" },
   {         768, "MQENC_FLOAT_S390" },
   {         768, "MQIAMO_SUBRQS_FAILED" },
   {         768, "MQRO_COA_WITH_DATA" },
   {         769, "MQIAMO_CBS" },
   {         770, "MQIAMO_CBS_FAILED" },
   {         771, "MQIAMO_CTLS" },
   {         772, "MQIAMO_CTLS_FAILED" },
   {         773, "MQIAMO_STATS" },
   {         774, "MQIAMO_STATS_FAILED" },
   {         775, "MQIAMO_SUB_DUR_HIGHWATER" },
   {         776, "MQIAMO_SUB_DUR_LOWWATER" },
   {         777, "MQIAMO_SUB_NDUR_HIGHWATER" },
   {         778, "MQIAMO_SUB_NDUR_LOWWATER" },
   {         779, "MQIAMO_TOPIC_PUTS" },
   {         780, "MQIAMO_TOPIC_PUTS_FAILED" },
   {         781, "MQIAMO_TOPIC_PUT1S" },
   {         782, "MQIAMO_TOPIC_PUT1S_FAILED" },
   {         783, "MQIAMO64_TOPIC_PUT_BYTES" },
   {         784, "MQIAMO_PUBLISH_MSG_COUNT" },
   {         785, "MQENC_S390" },
   {         785, "MQIAMO64_PUBLISH_MSG_BYTES" },
   {         786, "MQIAMO_UNSUBS_DUR" },
   {         787, "MQIAMO_UNSUBS_NDUR" },
   {         788, "MQIAMO_UNSUBS_FAILED" },
   {         789, "MQIAMO_INTERVAL" },
   {         790, "MQIAMO_MSGS_SENT" },
   {         791, "MQIAMO_BYTES_SENT" },
   {         792, "MQIAMO_REPAIR_BYTES" },
   {         793, "MQIAMO_FEEDBACK_MODE" },
   {         794, "MQIAMO_RELIABILITY_TYPE" },
   {         795, "MQIAMO_LATE_JOIN_MARK" },
   {         796, "MQIAMO_NACKS_RCVD" },
   {         797, "MQIAMO_REPAIR_PKTS" },
   {         798, "MQIAMO_HISTORY_PKTS" },
   {         799, "MQIAMO_PENDING_PKTS" },
   {         800, "MQCHSSTATE_IN_RCVEXIT" },
   {         800, "MQCMDL_LEVEL_800" },
   {         800, "MQIAMO_PKT_RATE" },
   {         801, "MQCMDL_LEVEL_801" },
   {         801, "MQIAMO_MCAST_XMIT_RATE" },
   {         802, "MQCMDL_LEVEL_802" },
   {         802, "MQIAMO_MCAST_BATCH_TIME" },
   {         803, "MQIAMO_MCAST_HEARTBEAT" },
   {         804, "MQIAMO_DEST_DATA_PORT" },
   {         805, "MQIAMO_DEST_REPAIR_PORT" },
   {         806, "MQIAMO_ACKS_RCVD" },
   {         807, "MQIAMO_ACTIVE_ACKERS" },
   {         808, "MQIAMO_PKTS_SENT" },
   {         809, "MQIAMO_TOTAL_REPAIR_PKTS" },
   {         810, "MQIAMO_TOTAL_PKTS_SENT" },
   {         811, "MQIAMO_TOTAL_MSGS_SENT" },
   {         812, "MQIAMO_TOTAL_BYTES_SENT" },
   {         813, "MQIAMO_NUM_STREAMS" },
   {         814, "MQIAMO_ACK_FEEDBACK" },
   {         815, "MQIAMO_NACK_FEEDBACK" },
   {         816, "MQIAMO_PKTS_LOST" },
   {         817, "MQIAMO_MSGS_RCVD" },
   {         818, "MQIAMO_MSG_BYTES_RCVD" },
   {         819, "MQIAMO_MSGS_DELIVERED" },
   {         820, "MQIAMO_PKTS_PROCESSED" },
   {         821, "MQIAMO_PKTS_DELIVERED" },
   {         822, "MQIAMO_PKTS_DROPPED" },
   {         823, "MQIAMO_PKTS_DUPLICATED" },
   {         824, "MQIAMO_NACKS_CREATED" },
   {         825, "MQIAMO_NACK_PKTS_SENT" },
   {         826, "MQIAMO_REPAIR_PKTS_RQSTD" },
   {         827, "MQIAMO_REPAIR_PKTS_RCVD" },
   {         828, "MQIAMO_PKTS_REPAIRED" },
   {         829, "MQIAMO_TOTAL_MSGS_RCVD" },
   {         830, "MQIAMO_TOTAL_MSG_BYTES_RCVD" },
   {         831, "MQIAMO_TOTAL_REPAIR_PKTS_RCVD" },
   {         832, "MQIAMO_TOTAL_REPAIR_PKTS_RQSTD" },
   {         833, "MQIAMO_TOTAL_MSGS_PROCESSED" },
   {         834, "MQIAMO_TOTAL_MSGS_SELECTED" },
   {         835, "MQIAMO_TOTAL_MSGS_EXPIRED" },
   {         836, "MQIAMO_TOTAL_MSGS_DELIVERED" },
   {         837, "MQIAMO_TOTAL_MSGS_RETURNED" },
   {         838, "MQIAMO64_HIGHRES_TIME" },
   {         839, "MQIAMO_MONITOR_CLASS" },
   {         840, "MQIAMO_MONITOR_TYPE" },
   {         841, "MQIAMO_MONITOR_ELEMENT" },
   {         842, "MQIAMO_MONITOR_DATATYPE" },
   {         843, "MQIAMO_MONITOR_FLAGS" },
   {         844, "MQIAMO64_QMGR_OP_DURATION" },
   {         845, "MQIAMO64_MONITOR_INTERVAL" },
   {         845, "MQIAMO_LAST_USED" },
   {         855, "MQFB_IMS_NACK_1A_REASON_LAST" },
   {         900, "MQCHSSTATE_IN_SENDEXIT" },
   {         900, "MQCMDL_LEVEL_900" },
   {         900, "MQRC_APPL_FIRST" },
   {         901, "MQCMDL_LEVEL_901" },
   {         902, "MQCMDL_LEVEL_902" },
   {         903, "MQCMDL_LEVEL_903" },
   {         904, "MQCMDL_LEVEL_904" },
   {         905, "MQCMDL_LEVEL_905" },
   {         910, "MQCMDL_LEVEL_910" },
   {         911, "MQCMDL_CURRENT_LEVEL" },
   {         911, "MQCMDL_LEVEL_911" },
   {         984, "MQCDC_LENGTH_1" },
   {         984, "MQCD_LENGTH_1" },
   {         999, "MQOT_RESERVED_1" },
   {         999, "MQRC_APPL_LAST" },
   {         999, "MQ_TOTAL_EXIT_DATA_LENGTH" },
   {         999, "MQ_TOTAL_EXIT_NAME_LENGTH" },
   {        1000, "MQCHSSTATE_IN_MSGEXIT" },
   {        1001, "MQIACF_FIRST" },
   {        1001, "MQIACF_Q_MGR_ATTRS" },
   {        1001, "MQNT_ALL" },
   {        1001, "MQOT_ALL" },
   {        1001, "MQQT_ALL" },
   {        1002, "MQIACF_Q_ATTRS" },
   {        1002, "MQOT_ALIAS_Q" },
   {        1003, "MQIACF_PROCESS_ATTRS" },
   {        1003, "MQOT_MODEL_Q" },
   {        1004, "MQIACF_NAMELIST_ATTRS" },
   {        1004, "MQOT_LOCAL_Q" },
   {        1005, "MQIACF_FORCE" },
   {        1005, "MQOT_REMOTE_Q" },
   {        1006, "MQIACF_REPLACE" },
   {        1007, "MQIACF_PURGE" },
   {        1007, "MQOT_SENDER_CHANNEL" },
   {        1008, "MQIACF_MODE" },
   {        1008, "MQIACF_QUIESCE" },
   {        1008, "MQOT_SERVER_CHANNEL" },
   {        1009, "MQIACF_ALL" },
   {        1009, "MQOT_REQUESTER_CHANNEL" },
   {        1010, "MQIACF_EVENT_APPL_TYPE" },
   {        1010, "MQOT_RECEIVER_CHANNEL" },
   {        1011, "MQIACF_EVENT_ORIGIN" },
   {        1011, "MQOT_CURRENT_CHANNEL" },
   {        1012, "MQIACF_PARAMETER_ID" },
   {        1012, "MQOT_SAVED_CHANNEL" },
   {        1013, "MQIACF_ERROR_ID" },
   {        1013, "MQIACF_ERROR_IDENTIFIER" },
   {        1013, "MQOT_SVRCONN_CHANNEL" },
   {        1014, "MQIACF_SELECTOR" },
   {        1014, "MQOT_CLNTCONN_CHANNEL" },
   {        1015, "MQIACF_CHANNEL_ATTRS" },
   {        1015, "MQOT_SHORT_CHANNEL" },
   {        1016, "MQIACF_OBJECT_TYPE" },
   {        1016, "MQOT_CHLAUTH" },
   {        1017, "MQIACF_ESCAPE_TYPE" },
   {        1017, "MQOT_REMOTE_Q_MGR_NAME" },
   {        1018, "MQIACF_ERROR_OFFSET" },
   {        1019, "MQIACF_AUTH_INFO_ATTRS" },
   {        1019, "MQOT_PROT_POLICY" },
   {        1020, "MQIACF_REASON_QUALIFIER" },
   {        1020, "MQOT_TT_CHANNEL" },
   {        1021, "MQIACF_COMMAND" },
   {        1021, "MQOT_AMQP_CHANNEL" },
   {        1022, "MQIACF_OPEN_OPTIONS" },
   {        1022, "MQOT_AUTH_REC" },
   {        1023, "MQIACF_OPEN_TYPE" },
   {        1023, "MQPD_ACCEPT_UNSUP_MASK" },
   {        1024, "MQCNO_LOCAL_BINDING" },
   {        1024, "MQENC_FLOAT_TNS" },
   {        1024, "MQGMO_UNLOCK" },
   {        1024, "MQIACF_PROCESS_ID" },
   {        1024, "MQIAMO_MONITOR_KB" },
   {        1024, "MQOO_SET_IDENTITY_CONTEXT" },
   {        1024, "MQPD_SUPPORT_REQUIRED_IF_LOCAL" },
   {        1024, "MQPMO_SET_IDENTITY_CONTEXT" },
   {        1024, "MQREGO_NON_PERSISTENT" },
   {        1024, "MQTYPE_STRING" },
   {        1024, "MQZAO_ALTERNATE_USER_AUTHORITY" },
   {        1024, "MQ_CLIENT_USER_ID_LENGTH" },
   {        1024, "MQ_DISTINGUISHED_NAME_LENGTH" },
   {        1024, "MQ_ENTITY_NAME_LENGTH" },
   {        1024, "MQ_JAAS_CONFIG_LENGTH" },
   {        1024, "MQ_LDAP_BASE_DN_LENGTH" },
   {        1024, "MQ_LDAP_MCA_USER_ID_LENGTH" },
   {        1024, "MQ_LOG_PATH_LENGTH" },
   {        1024, "MQ_MAX_LDAP_MCA_USER_ID_LENGTH" },
   {        1024, "MQ_SSL_KEY_PASSPHRASE_LENGTH" },
   {        1024, "MQ_SSL_PEER_NAME_LENGTH" },
   {        1025, "MQIACF_THREAD_ID" },
   {        1026, "MQIACF_Q_STATUS_ATTRS" },
   {        1027, "MQIACF_UNCOMMITTED_MSGS" },
   {        1028, "MQIACF_HANDLE_STATE" },
   {        1041, "MQENC_TNS" },
   {        1070, "MQIACF_AUX_ERROR_DATA_INT_1" },
   {        1071, "MQIACF_AUX_ERROR_DATA_INT_2" },
   {        1072, "MQIACF_CONV_REASON_CODE" },
   {        1073, "MQIACF_BRIDGE_TYPE" },
   {        1074, "MQIACF_INQUIRY" },
   {        1075, "MQIACF_WAIT_INTERVAL" },
   {        1076, "MQIACF_OPTIONS" },
   {        1077, "MQIACF_BROKER_OPTIONS" },
   {        1078, "MQIACF_REFRESH_TYPE" },
   {        1079, "MQIACF_SEQUENCE_NUMBER" },
   {        1080, "MQIACF_INTEGER_DATA" },
   {        1081, "MQIACF_REGISTRATION_OPTIONS" },
   {        1082, "MQIACF_PUBLICATION_OPTIONS" },
   {        1083, "MQIACF_CLUSTER_INFO" },
   {        1084, "MQIACF_Q_MGR_DEFINITION_TYPE" },
   {        1085, "MQIACF_Q_MGR_TYPE" },
   {        1086, "MQIACF_ACTION" },
   {        1087, "MQIACF_SUSPEND" },
   {        1088, "MQIACF_BROKER_COUNT" },
   {        1089, "MQIACF_APPL_COUNT" },
   {        1090, "MQIACF_ANONYMOUS_COUNT" },
   {        1091, "MQIACF_REG_REG_OPTIONS" },
   {        1092, "MQIACF_DELETE_OPTIONS" },
   {        1093, "MQIACF_CLUSTER_Q_MGR_ATTRS" },
   {        1094, "MQIACF_REFRESH_INTERVAL" },
   {        1095, "MQIACF_REFRESH_REPOSITORY" },
   {        1096, "MQIACF_REMOVE_QUEUES" },
   {        1098, "MQIACF_OPEN_INPUT_TYPE" },
   {        1099, "MQIACF_OPEN_OUTPUT" },
   {        1100, "MQCHSSTATE_IN_MREXIT" },
   {        1100, "MQIACF_OPEN_SET" },
   {        1101, "MQIACF_OPEN_INQUIRE" },
   {        1102, "MQIACF_OPEN_BROWSE" },
   {        1103, "MQIACF_Q_STATUS_TYPE" },
   {        1104, "MQIACF_Q_HANDLE" },
   {        1105, "MQIACF_Q_STATUS" },
   {        1106, "MQIACF_SECURITY_TYPE" },
   {        1107, "MQIACF_CONNECTION_ATTRS" },
   {        1108, "MQIACF_CONNECT_OPTIONS" },
   {        1110, "MQIACF_CONN_INFO_TYPE" },
   {        1111, "MQIACF_CONN_INFO_CONN" },
   {        1112, "MQIACF_CONN_INFO_HANDLE" },
   {        1113, "MQIACF_CONN_INFO_ALL" },
   {        1114, "MQIACF_AUTH_PROFILE_ATTRS" },
   {        1115, "MQIACF_AUTHORIZATION_LIST" },
   {        1116, "MQIACF_AUTH_ADD_AUTHS" },
   {        1117, "MQIACF_AUTH_REMOVE_AUTHS" },
   {        1118, "MQIACF_ENTITY_TYPE" },
   {        1120, "MQIACF_COMMAND_INFO" },
   {        1121, "MQIACF_CMDSCOPE_Q_MGR_COUNT" },
   {        1122, "MQIACF_Q_MGR_SYSTEM" },
   {        1123, "MQIACF_Q_MGR_EVENT" },
   {        1124, "MQIACF_Q_MGR_DQM" },
   {        1125, "MQIACF_Q_MGR_CLUSTER" },
   {        1126, "MQIACF_QSG_DISPS" },
   {        1128, "MQIACF_UOW_STATE" },
   {        1129, "MQIACF_SECURITY_ITEM" },
   {        1130, "MQIACF_CF_STRUC_STATUS" },
   {        1132, "MQIACF_UOW_TYPE" },
   {        1133, "MQIACF_CF_STRUC_ATTRS" },
   {        1134, "MQIACF_EXCLUDE_INTERVAL" },
   {        1135, "MQIACF_CF_STATUS_TYPE" },
   {        1136, "MQIACF_CF_STATUS_SUMMARY" },
   {        1137, "MQIACF_CF_STATUS_CONNECT" },
   {        1138, "MQIACF_CF_STATUS_BACKUP" },
   {        1139, "MQIACF_CF_STRUC_TYPE" },
   {        1140, "MQIACF_CF_STRUC_SIZE_MAX" },
   {        1141, "MQIACF_CF_STRUC_SIZE_USED" },
   {        1142, "MQIACF_CF_STRUC_ENTRIES_MAX" },
   {        1143, "MQIACF_CF_STRUC_ENTRIES_USED" },
   {        1144, "MQIACF_CF_STRUC_BACKUP_SIZE" },
   {        1145, "MQIACF_MOVE_TYPE" },
   {        1146, "MQIACF_MOVE_TYPE_MOVE" },
   {        1147, "MQIACF_MOVE_TYPE_ADD" },
   {        1148, "MQIACF_Q_MGR_NUMBER" },
   {        1149, "MQIACF_Q_MGR_STATUS" },
   {        1150, "MQIACF_DB2_CONN_STATUS" },
   {        1151, "MQIACF_SECURITY_ATTRS" },
   {        1152, "MQIACF_SECURITY_TIMEOUT" },
   {        1153, "MQIACF_SECURITY_INTERVAL" },
   {        1154, "MQIACF_SECURITY_SWITCH" },
   {        1155, "MQIACF_SECURITY_SETTING" },
   {        1156, "MQIACF_STORAGE_CLASS_ATTRS" },
   {        1157, "MQIACF_USAGE_TYPE" },
   {        1158, "MQIACF_BUFFER_POOL_ID" },
   {        1159, "MQIACF_USAGE_TOTAL_PAGES" },
   {        1160, "MQIACF_USAGE_UNUSED_PAGES" },
   {        1161, "MQIACF_USAGE_PERSIST_PAGES" },
   {        1162, "MQIACF_USAGE_NONPERSIST_PAGES" },
   {        1163, "MQIACF_USAGE_RESTART_EXTENTS" },
   {        1164, "MQIACF_USAGE_EXPAND_COUNT" },
   {        1165, "MQIACF_PAGESET_STATUS" },
   {        1166, "MQIACF_USAGE_TOTAL_BUFFERS" },
   {        1167, "MQIACF_USAGE_DATA_SET_TYPE" },
   {        1168, "MQIACF_USAGE_PAGESET" },
   {        1169, "MQIACF_USAGE_DATA_SET" },
   {        1170, "MQIACF_USAGE_BUFFER_POOL" },
   {        1171, "MQIACF_MOVE_COUNT" },
   {        1172, "MQIACF_EXPIRY_Q_COUNT" },
   {        1173, "MQIACF_CONFIGURATION_OBJECTS" },
   {        1174, "MQIACF_CONFIGURATION_EVENTS" },
   {        1175, "MQIACF_SYSP_TYPE" },
   {        1176, "MQIACF_SYSP_DEALLOC_INTERVAL" },
   {        1177, "MQIACF_SYSP_MAX_ARCHIVE" },
   {        1178, "MQIACF_SYSP_MAX_READ_TAPES" },
   {        1179, "MQIACF_SYSP_IN_BUFFER_SIZE" },
   {        1180, "MQIACF_SYSP_OUT_BUFFER_SIZE" },
   {        1181, "MQIACF_SYSP_OUT_BUFFER_COUNT" },
   {        1182, "MQIACF_SYSP_ARCHIVE" },
   {        1183, "MQIACF_SYSP_DUAL_ACTIVE" },
   {        1184, "MQIACF_SYSP_DUAL_ARCHIVE" },
   {        1185, "MQIACF_SYSP_DUAL_BSDS" },
   {        1186, "MQIACF_SYSP_MAX_CONNS" },
   {        1187, "MQIACF_SYSP_MAX_CONNS_FORE" },
   {        1188, "MQIACF_SYSP_MAX_CONNS_BACK" },
   {        1189, "MQIACF_SYSP_EXIT_INTERVAL" },
   {        1190, "MQIACF_SYSP_EXIT_TASKS" },
   {        1191, "MQIACF_SYSP_CHKPOINT_COUNT" },
   {        1192, "MQIACF_SYSP_OTMA_INTERVAL" },
   {        1193, "MQIACF_SYSP_Q_INDEX_DEFER" },
   {        1194, "MQIACF_SYSP_DB2_TASKS" },
   {        1195, "MQIACF_SYSP_RESLEVEL_AUDIT" },
   {        1196, "MQIACF_SYSP_ROUTING_CODE" },
   {        1197, "MQIACF_SYSP_SMF_ACCOUNTING" },
   {        1198, "MQIACF_SYSP_SMF_STATS" },
   {        1199, "MQIACF_SYSP_SMF_INTERVAL" },
   {        1200, "MQCHSSTATE_IN_CHADEXIT" },
   {        1200, "MQIACF_SYSP_TRACE_CLASS" },
   {        1201, "MQIACF_SYSP_TRACE_SIZE" },
   {        1202, "MQIACF_SYSP_WLM_INTERVAL" },
   {        1203, "MQIACF_SYSP_ALLOC_UNIT" },
   {        1204, "MQIACF_SYSP_ARCHIVE_RETAIN" },
   {        1205, "MQIACF_SYSP_ARCHIVE_WTOR" },
   {        1206, "MQIACF_SYSP_BLOCK_SIZE" },
   {        1207, "MQIACF_SYSP_CATALOG" },
   {        1208, "MQIACF_SYSP_COMPACT" },
   {        1209, "MQIACF_SYSP_ALLOC_PRIMARY" },
   {        1210, "MQIACF_SYSP_ALLOC_SECONDARY" },
   {        1211, "MQIACF_SYSP_PROTECT" },
   {        1212, "MQIACF_SYSP_QUIESCE_INTERVAL" },
   {        1213, "MQIACF_SYSP_TIMESTAMP" },
   {        1214, "MQIACF_SYSP_UNIT_ADDRESS" },
   {        1215, "MQIACF_SYSP_UNIT_STATUS" },
   {        1216, "MQIACF_SYSP_LOG_COPY" },
   {        1217, "MQIACF_SYSP_LOG_USED" },
   {        1218, "MQIACF_SYSP_LOG_SUSPEND" },
   {        1219, "MQIACF_SYSP_OFFLOAD_STATUS" },
   {        1220, "MQIACF_SYSP_TOTAL_LOGS" },
   {        1221, "MQIACF_SYSP_FULL_LOGS" },
   {        1222, "MQIACF_LISTENER_ATTRS" },
   {        1223, "MQIACF_LISTENER_STATUS_ATTRS" },
   {        1224, "MQIACF_SERVICE_ATTRS" },
   {        1225, "MQIACF_SERVICE_STATUS_ATTRS" },
   {        1226, "MQIACF_Q_TIME_INDICATOR" },
   {        1227, "MQIACF_OLDEST_MSG_AGE" },
   {        1228, "MQIACF_AUTH_OPTIONS" },
   {        1229, "MQIACF_Q_MGR_STATUS_ATTRS" },
   {        1230, "MQIACF_CONNECTION_COUNT" },
   {        1231, "MQIACF_Q_MGR_FACILITY" },
   {        1232, "MQIACF_CHINIT_STATUS" },
   {        1233, "MQIACF_CMD_SERVER_STATUS" },
   {        1234, "MQIACF_ROUTE_DETAIL" },
   {        1235, "MQIACF_RECORDED_ACTIVITIES" },
   {        1236, "MQIACF_MAX_ACTIVITIES" },
   {        1237, "MQIACF_DISCONTINUITY_COUNT" },
   {        1238, "MQIACF_ROUTE_ACCUMULATION" },
   {        1239, "MQIACF_ROUTE_DELIVERY" },
   {        1240, "MQIACF_OPERATION_TYPE" },
   {        1241, "MQIACF_BACKOUT_COUNT" },
   {        1242, "MQIACF_COMP_CODE" },
   {        1243, "MQIACF_ENCODING" },
   {        1244, "MQIACF_EXPIRY" },
   {        1245, "MQIACF_FEEDBACK" },
   {        1247, "MQIACF_MSG_FLAGS" },
   {        1248, "MQIACF_MSG_LENGTH" },
   {        1249, "MQIACF_MSG_TYPE" },
   {        1250, "MQCHSSTATE_NET_CONNECTING" },
   {        1250, "MQIACF_OFFSET" },
   {        1251, "MQIACF_ORIGINAL_LENGTH" },
   {        1252, "MQIACF_PERSISTENCE" },
   {        1253, "MQIACF_PRIORITY" },
   {        1254, "MQIACF_REASON_CODE" },
   {        1255, "MQIACF_REPORT" },
   {        1256, "MQIACF_VERSION" },
   {        1257, "MQIACF_UNRECORDED_ACTIVITIES" },
   {        1258, "MQIACF_MONITORING" },
   {        1259, "MQIACF_ROUTE_FORWARDING" },
   {        1260, "MQIACF_SERVICE_STATUS" },
   {        1261, "MQIACF_Q_TYPES" },
   {        1262, "MQIACF_USER_ID_SUPPORT" },
   {        1263, "MQIACF_INTERFACE_VERSION" },
   {        1264, "MQIACF_AUTH_SERVICE_ATTRS" },
   {        1265, "MQIACF_USAGE_EXPAND_TYPE" },
   {        1266, "MQIACF_SYSP_CLUSTER_CACHE" },
   {        1267, "MQIACF_SYSP_DB2_BLOB_TASKS" },
   {        1268, "MQIACF_SYSP_WLM_INT_UNITS" },
   {        1269, "MQIACF_TOPIC_ATTRS" },
   {        1271, "MQIACF_PUBSUB_PROPERTIES" },
   {        1273, "MQIACF_DESTINATION_CLASS" },
   {        1274, "MQIACF_DURABLE_SUBSCRIPTION" },
   {        1275, "MQIACF_SUBSCRIPTION_SCOPE" },
   {        1277, "MQIACF_VARIABLE_USER_ID" },
   {        1280, "MQIACF_REQUEST_ONLY" },
   {        1283, "MQIACF_PUB_PRIORITY" },
   {        1287, "MQIACF_SUB_ATTRS" },
   {        1288, "MQIACF_WILDCARD_SCHEMA" },
   {        1289, "MQIACF_SUB_TYPE" },
   {        1290, "MQIACF_MESSAGE_COUNT" },
   {        1291, "MQIACF_Q_MGR_PUBSUB" },
   {        1292, "MQIACF_Q_MGR_VERSION" },
   {        1294, "MQIACF_SUB_STATUS_ATTRS" },
   {        1295, "MQIACF_TOPIC_STATUS" },
   {        1296, "MQIACF_TOPIC_SUB" },
   {        1297, "MQIACF_TOPIC_PUB" },
   {        1300, "MQCHSSTATE_SSL_HANDSHAKING" },
   {        1300, "MQIACF_RETAINED_PUBLICATION" },
   {        1301, "MQIACF_TOPIC_STATUS_ATTRS" },
   {        1302, "MQIACF_TOPIC_STATUS_TYPE" },
   {        1303, "MQIACF_SUB_OPTIONS" },
   {        1304, "MQIACF_PUBLISH_COUNT" },
   {        1305, "MQIACF_CLEAR_TYPE" },
   {        1306, "MQIACF_CLEAR_SCOPE" },
   {        1307, "MQIACF_SUB_LEVEL" },
   {        1308, "MQIACF_ASYNC_STATE" },
   {        1309, "MQIACF_SUB_SUMMARY" },
   {        1310, "MQIACF_OBSOLETE_MSGS" },
   {        1311, "MQIACF_PUBSUB_STATUS" },
   {        1312, "MQCDC_LENGTH_2" },
   {        1312, "MQCD_LENGTH_2" },
   {        1314, "MQIACF_PS_STATUS_TYPE" },
   {        1318, "MQIACF_PUBSUB_STATUS_ATTRS" },
   {        1321, "MQIACF_SELECTOR_TYPE" },
   {        1322, "MQIACF_LOG_COMPRESSION" },
   {        1323, "MQIACF_GROUPUR_CHECK_ID" },
   {        1324, "MQIACF_MULC_CAPTURE" },
   {        1325, "MQIACF_PERMIT_STANDBY" },
   {        1326, "MQIACF_OPERATION_MODE" },
   {        1327, "MQIACF_COMM_INFO_ATTRS" },
   {        1328, "MQIACF_CF_SMDS_BLOCK_SIZE" },
   {        1329, "MQIACF_CF_SMDS_EXPAND" },
   {        1330, "MQIACF_USAGE_FREE_BUFF" },
   {        1331, "MQIACF_USAGE_FREE_BUFF_PERC" },
   {        1332, "MQIACF_CF_STRUC_ACCESS" },
   {        1333, "MQIACF_CF_STATUS_SMDS" },
   {        1334, "MQIACF_SMDS_ATTRS" },
   {        1335, "MQIACF_USAGE_SMDS" },
   {        1336, "MQIACF_USAGE_BLOCK_SIZE" },
   {        1337, "MQIACF_USAGE_DATA_BLOCKS" },
   {        1338, "MQIACF_USAGE_EMPTY_BUFFERS" },
   {        1339, "MQIACF_USAGE_INUSE_BUFFERS" },
   {        1340, "MQIACF_USAGE_LOWEST_FREE" },
   {        1341, "MQIACF_USAGE_OFFLOAD_MSGS" },
   {        1342, "MQIACF_USAGE_READS_SAVED" },
   {        1343, "MQIACF_USAGE_SAVED_BUFFERS" },
   {        1344, "MQIACF_USAGE_TOTAL_BLOCKS" },
   {        1345, "MQIACF_USAGE_USED_BLOCKS" },
   {        1346, "MQIACF_USAGE_USED_RATE" },
   {        1347, "MQIACF_USAGE_WAIT_RATE" },
   {        1348, "MQIACF_SMDS_OPENMODE" },
   {        1349, "MQIACF_SMDS_STATUS" },
   {        1350, "MQIACF_SMDS_AVAIL" },
   {        1351, "MQIACF_MCAST_REL_INDICATOR" },
   {        1352, "MQIACF_CHLAUTH_TYPE" },
   {        1354, "MQIACF_MQXR_DIAGNOSTICS_TYPE" },
   {        1355, "MQIACF_CHLAUTH_ATTRS" },
   {        1356, "MQIACF_OPERATION_ID" },
   {        1357, "MQIACF_API_CALLER_TYPE" },
   {        1358, "MQIACF_API_ENVIRONMENT" },
   {        1359, "MQIACF_TRACE_DETAIL" },
   {        1360, "MQIACF_HOBJ" },
   {        1361, "MQIACF_CALL_TYPE" },
   {        1362, "MQIACF_MQCB_OPERATION" },
   {        1363, "MQIACF_MQCB_TYPE" },
   {        1364, "MQIACF_MQCB_OPTIONS" },
   {        1365, "MQIACF_CLOSE_OPTIONS" },
   {        1366, "MQIACF_CTL_OPERATION" },
   {        1367, "MQIACF_GET_OPTIONS" },
   {        1368, "MQIACF_RECS_PRESENT" },
   {        1369, "MQIACF_KNOWN_DEST_COUNT" },
   {        1370, "MQIACF_UNKNOWN_DEST_COUNT" },
   {        1371, "MQIACF_INVALID_DEST_COUNT" },
   {        1372, "MQIACF_RESOLVED_TYPE" },
   {        1373, "MQIACF_PUT_OPTIONS" },
   {        1374, "MQIACF_BUFFER_LENGTH" },
   {        1375, "MQIACF_TRACE_DATA_LENGTH" },
   {        1376, "MQIACF_SMDS_EXPANDST" },
   {        1377, "MQIACF_STRUC_LENGTH" },
   {        1378, "MQIACF_ITEM_COUNT" },
   {        1379, "MQIACF_EXPIRY_TIME" },
   {        1380, "MQIACF_CONNECT_TIME" },
   {        1381, "MQIACF_DISCONNECT_TIME" },
   {        1382, "MQIACF_HSUB" },
   {        1383, "MQIACF_SUBRQ_OPTIONS" },
   {        1384, "MQIACF_XA_RMID" },
   {        1385, "MQIACF_XA_FLAGS" },
   {        1386, "MQIACF_XA_RETCODE" },
   {        1387, "MQIACF_XA_HANDLE" },
   {        1388, "MQIACF_XA_RETVAL" },
   {        1389, "MQIACF_STATUS_TYPE" },
   {        1390, "MQIACF_XA_COUNT" },
   {        1391, "MQIACF_SELECTOR_COUNT" },
   {        1392, "MQIACF_SELECTORS" },
   {        1393, "MQIACF_INTATTR_COUNT" },
   {        1394, "MQIACF_INT_ATTRS" },
   {        1395, "MQIACF_SUBRQ_ACTION" },
   {        1396, "MQIACF_NUM_PUBS" },
   {        1397, "MQIACF_POINTER_SIZE" },
   {        1398, "MQIACF_REMOVE_AUTHREC" },
   {        1399, "MQIACF_XR_ATTRS" },
   {        1400, "MQCHSSTATE_NAME_SERVER" },
   {        1400, "MQIACF_APPL_FUNCTION_TYPE" },
   {        1401, "MQIACF_AMQP_ATTRS" },
   {        1402, "MQIACF_EXPORT_TYPE" },
   {        1403, "MQIACF_EXPORT_ATTRS" },
   {        1404, "MQIACF_SYSTEM_OBJECTS" },
   {        1405, "MQIACF_CONNECTION_SWAP" },
   {        1406, "MQIACF_AMQP_DIAGNOSTICS_TYPE" },
   {        1408, "MQIACF_BUFFER_POOL_LOCATION" },
   {        1409, "MQIACF_LDAP_CONNECTION_STATUS" },
   {        1410, "MQIACF_SYSP_MAX_ACE_POOL" },
   {        1411, "MQIACF_PAGECLAS" },
   {        1412, "MQIACF_AUTH_REC_TYPE" },
   {        1413, "MQIACF_SYSP_MAX_CONC_OFFLOADS" },
   {        1414, "MQIACF_SYSP_ZHYPERWRITE" },
   {        1415, "MQIACF_Q_MGR_STATUS_LOG" },
   {        1416, "MQIACF_ARCHIVE_LOG_SIZE" },
   {        1417, "MQIACF_MEDIA_LOG_SIZE" },
   {        1418, "MQIACF_RESTART_LOG_SIZE" },
   {        1419, "MQIACF_REUSABLE_LOG_SIZE" },
   {        1420, "MQIACF_LOG_IN_USE" },
   {        1421, "MQIACF_LOG_UTILIZATION" },
   {        1422, "MQIACF_LOG_REDUCTION" },
   {        1423, "MQIACF_IGNORE_STATE" },
   {        1423, "MQIACF_LAST_USED" },
   {        1480, "MQCDC_LENGTH_3" },
   {        1480, "MQCD_LENGTH_3" },
   {        1500, "MQCHSSTATE_IN_MQPUT" },
   {        1501, "MQIACH_FIRST" },
   {        1501, "MQIACH_XMIT_PROTOCOL_TYPE" },
   {        1502, "MQIACH_BATCH_SIZE" },
   {        1503, "MQIACH_DISC_INTERVAL" },
   {        1504, "MQIACH_SHORT_TIMER" },
   {        1505, "MQIACH_SHORT_RETRY" },
   {        1506, "MQIACH_LONG_TIMER" },
   {        1507, "MQIACH_LONG_RETRY" },
   {        1508, "MQIACH_PUT_AUTHORITY" },
   {        1509, "MQIACH_SEQUENCE_NUMBER_WRAP" },
   {        1510, "MQIACH_MAX_MSG_LENGTH" },
   {        1511, "MQIACH_CHANNEL_TYPE" },
   {        1512, "MQIACH_DATA_COUNT" },
   {        1513, "MQIACH_NAME_COUNT" },
   {        1514, "MQIACH_MSG_SEQUENCE_NUMBER" },
   {        1515, "MQIACH_DATA_CONVERSION" },
   {        1516, "MQIACH_IN_DOUBT" },
   {        1517, "MQIACH_MCA_TYPE" },
   {        1518, "MQIACH_SESSION_COUNT" },
   {        1519, "MQIACH_ADAPTER" },
   {        1520, "MQIACH_COMMAND_COUNT" },
   {        1521, "MQIACH_SOCKET" },
   {        1522, "MQIACH_PORT" },
   {        1523, "MQIACH_CHANNEL_INSTANCE_TYPE" },
   {        1524, "MQIACH_CHANNEL_INSTANCE_ATTRS" },
   {        1525, "MQIACH_CHANNEL_ERROR_DATA" },
   {        1526, "MQIACH_CHANNEL_TABLE" },
   {        1527, "MQIACH_CHANNEL_STATUS" },
   {        1528, "MQIACH_INDOUBT_STATUS" },
   {        1529, "MQIACH_LAST_SEQUENCE_NUMBER" },
   {        1529, "MQIACH_LAST_SEQ_NUMBER" },
   {        1531, "MQIACH_CURRENT_MSGS" },
   {        1532, "MQIACH_CURRENT_SEQUENCE_NUMBER" },
   {        1532, "MQIACH_CURRENT_SEQ_NUMBER" },
   {        1533, "MQIACH_SSL_RETURN_CODE" },
   {        1534, "MQIACH_MSGS" },
   {        1535, "MQIACH_BYTES_SENT" },
   {        1536, "MQIACH_BYTES_RCVD" },
   {        1536, "MQIACH_BYTES_RECEIVED" },
   {        1537, "MQIACH_BATCHES" },
   {        1538, "MQIACH_BUFFERS_SENT" },
   {        1539, "MQIACH_BUFFERS_RCVD" },
   {        1539, "MQIACH_BUFFERS_RECEIVED" },
   {        1540, "MQCDC_LENGTH_4 (4 byte)" },
   {        1540, "MQCD_LENGTH_4 (4 byte)" },
   {        1540, "MQIACH_LONG_RETRIES_LEFT" },
   {        1541, "MQIACH_SHORT_RETRIES_LEFT" },
   {        1542, "MQIACH_MCA_STATUS" },
   {        1543, "MQIACH_STOP_REQUESTED" },
   {        1544, "MQIACH_MR_COUNT" },
   {        1545, "MQIACH_MR_INTERVAL" },
   {        1552, "MQCDC_LENGTH_5 (4 byte)" },
   {        1552, "MQCD_LENGTH_5 (4 byte)" },
   {        1562, "MQIACH_NPM_SPEED" },
   {        1563, "MQIACH_HB_INTERVAL" },
   {        1564, "MQIACH_BATCH_INTERVAL" },
   {        1565, "MQIACH_NETWORK_PRIORITY" },
   {        1566, "MQIACH_KEEP_ALIVE_INTERVAL" },
   {        1567, "MQIACH_BATCH_HB" },
   {        1568, "MQCDC_LENGTH_4 (8 byte)" },
   {        1568, "MQCD_LENGTH_4 (8 byte)" },
   {        1568, "MQIACH_SSL_CLIENT_AUTH" },
   {        1570, "MQIACH_ALLOC_RETRY" },
   {        1571, "MQIACH_ALLOC_FAST_TIMER" },
   {        1572, "MQIACH_ALLOC_SLOW_TIMER" },
   {        1573, "MQIACH_DISC_RETRY" },
   {        1574, "MQIACH_PORT_NUMBER" },
   {        1575, "MQIACH_HDR_COMPRESSION" },
   {        1576, "MQIACH_MSG_COMPRESSION" },
   {        1577, "MQIACH_CLWL_CHANNEL_RANK" },
   {        1578, "MQIACH_CLWL_CHANNEL_PRIORITY" },
   {        1579, "MQIACH_CLWL_CHANNEL_WEIGHT" },
   {        1580, "MQIACH_CHANNEL_DISP" },
   {        1581, "MQIACH_INBOUND_DISP" },
   {        1582, "MQIACH_CHANNEL_TYPES" },
   {        1583, "MQIACH_ADAPS_STARTED" },
   {        1584, "MQCDC_LENGTH_5 (8 byte)" },
   {        1584, "MQCD_LENGTH_5 (8 byte)" },
   {        1584, "MQIACH_ADAPS_MAX" },
   {        1585, "MQIACH_DISPS_STARTED" },
   {        1586, "MQIACH_DISPS_MAX" },
   {        1587, "MQIACH_SSLTASKS_STARTED" },
   {        1588, "MQIACH_SSLTASKS_MAX" },
   {        1589, "MQIACH_CURRENT_CHL" },
   {        1590, "MQIACH_CURRENT_CHL_MAX" },
   {        1591, "MQIACH_CURRENT_CHL_TCP" },
   {        1592, "MQIACH_CURRENT_CHL_LU62" },
   {        1593, "MQIACH_ACTIVE_CHL" },
   {        1594, "MQIACH_ACTIVE_CHL_MAX" },
   {        1595, "MQIACH_ACTIVE_CHL_PAUSED" },
   {        1596, "MQIACH_ACTIVE_CHL_STARTED" },
   {        1597, "MQIACH_ACTIVE_CHL_STOPPED" },
   {        1598, "MQIACH_ACTIVE_CHL_RETRY" },
   {        1599, "MQIACH_LISTENER_STATUS" },
   {        1600, "MQCHSSTATE_IN_MQGET" },
   {        1600, "MQIACH_SHARED_CHL_RESTART" },
   {        1601, "MQIACH_LISTENER_CONTROL" },
   {        1602, "MQIACH_BACKLOG" },
   {        1604, "MQIACH_XMITQ_TIME_INDICATOR" },
   {        1605, "MQIACH_NETWORK_TIME_INDICATOR" },
   {        1606, "MQIACH_EXIT_TIME_INDICATOR" },
   {        1607, "MQIACH_BATCH_SIZE_INDICATOR" },
   {        1608, "MQIACH_XMITQ_MSGS_AVAILABLE" },
   {        1609, "MQIACH_CHANNEL_SUBSTATE" },
   {        1610, "MQIACH_SSL_KEY_RESETS" },
   {        1611, "MQIACH_COMPRESSION_RATE" },
   {        1612, "MQIACH_COMPRESSION_TIME" },
   {        1613, "MQIACH_MAX_XMIT_SIZE" },
   {        1614, "MQIACH_DEF_CHANNEL_DISP" },
   {        1615, "MQIACH_SHARING_CONVERSATIONS" },
   {        1616, "MQIACH_MAX_SHARING_CONVS" },
   {        1617, "MQIACH_CURRENT_SHARING_CONVS" },
   {        1618, "MQIACH_MAX_INSTANCES" },
   {        1619, "MQIACH_MAX_INSTS_PER_CLIENT" },
   {        1620, "MQIACH_CLIENT_CHANNEL_WEIGHT" },
   {        1621, "MQIACH_CONNECTION_AFFINITY" },
   {        1623, "MQIACH_RESET_REQUESTED" },
   {        1624, "MQIACH_BATCH_DATA_LIMIT" },
   {        1625, "MQIACH_MSG_HISTORY" },
   {        1626, "MQIACH_MULTICAST_PROPERTIES" },
   {        1627, "MQIACH_NEW_SUBSCRIBER_HISTORY" },
   {        1628, "MQIACH_MC_HB_INTERVAL" },
   {        1629, "MQIACH_USE_CLIENT_ID" },
   {        1630, "MQIACH_MQTT_KEEP_ALIVE" },
   {        1631, "MQIACH_IN_DOUBT_IN" },
   {        1632, "MQIACH_IN_DOUBT_OUT" },
   {        1633, "MQIACH_MSGS_SENT" },
   {        1634, "MQIACH_MSGS_RCVD" },
   {        1634, "MQIACH_MSGS_RECEIVED" },
   {        1635, "MQIACH_PENDING_OUT" },
   {        1636, "MQIACH_AVAILABLE_CIPHERSPECS" },
   {        1637, "MQIACH_MATCH" },
   {        1638, "MQIACH_USER_SOURCE" },
   {        1639, "MQIACH_WARNING" },
   {        1640, "MQIACH_DEF_RECONNECT" },
   {        1642, "MQIACH_CHANNEL_SUMMARY_ATTRS" },
   {        1643, "MQIACH_PROTOCOL" },
   {        1644, "MQIACH_AMQP_KEEP_ALIVE" },
   {        1645, "MQIACH_LAST_USED" },
   {        1645, "MQIACH_SECURITY_PROTOCOL" },
   {        1648, "MQCDC_LENGTH_6 (4 byte)" },
   {        1648, "MQCD_LENGTH_6 (4 byte)" },
   {        1688, "MQCDC_LENGTH_6 (8 byte)" },
   {        1688, "MQCD_LENGTH_6 (8 byte)" },
   {        1700, "MQCHSSTATE_IN_MQI_CALL" },
   {        1748, "MQCDC_LENGTH_7 (4 byte)" },
   {        1748, "MQCD_LENGTH_7 (4 byte)" },
   {        1792, "MQCDC_LENGTH_7 (8 byte)" },
   {        1792, "MQCD_LENGTH_7 (8 byte)" },
   {        1792, "MQRO_COA_WITH_FULL_DATA" },
   {        1800, "MQCHSSTATE_COMPRESSING" },
   {        1840, "MQCDC_LENGTH_8 (4 byte)" },
   {        1840, "MQCD_LENGTH_8 (4 byte)" },
   {        1864, "MQCDC_LENGTH_9 (4 byte)" },
   {        1864, "MQCD_LENGTH_9 (4 byte)" },
   {        1876, "MQCDC_LENGTH_10 (4 byte)" },
   {        1876, "MQCD_LENGTH_10 (4 byte)" },
   {        1888, "MQCDC_LENGTH_8 (8 byte)" },
   {        1888, "MQCD_LENGTH_8 (8 byte)" },
   {        1912, "MQCDC_LENGTH_9 (8 byte)" },
   {        1912, "MQCD_LENGTH_9 (8 byte)" },
   {        1920, "MQCDC_LENGTH_10 (8 byte)" },
   {        1920, "MQCD_LENGTH_10 (8 byte)" },
   {        1940, "MQCDC_CURRENT_LENGTH (4 byte)" },
   {        1940, "MQCDC_LENGTH_11 (4 byte)" },
   {        1940, "MQCD_CURRENT_LENGTH (4 byte)" },
   {        1940, "MQCD_LENGTH_11 (4 byte)" },
   {        1984, "MQCDC_CURRENT_LENGTH (8 byte)" },
   {        1984, "MQCDC_LENGTH_11 (8 byte)" },
   {        1984, "MQCD_CURRENT_LENGTH (8 byte)" },
   {        1984, "MQCD_LENGTH_11 (8 byte)" },
   {        2000, "MQIA_LAST" },
   {        2000, "MQIA_USER_LIST" },
   {        2001, "MQCA_APPL_ID" },
   {        2001, "MQCA_FIRST" },
   {        2001, "MQRC_ALIAS_BASE_Q_TYPE_ERROR" },
   {        2002, "MQCA_BASE_OBJECT_NAME" },
   {        2002, "MQCA_BASE_Q_NAME" },
   {        2002, "MQRC_ALREADY_CONNECTED" },
   {        2003, "MQCA_COMMAND_INPUT_Q_NAME" },
   {        2003, "MQRC_BACKED_OUT" },
   {        2004, "MQCA_CREATION_DATE" },
   {        2004, "MQRC_BUFFER_ERROR" },
   {        2005, "MQCA_CREATION_TIME" },
   {        2005, "MQRC_BUFFER_LENGTH_ERROR" },
   {        2006, "MQCA_DEAD_LETTER_Q_NAME" },
   {        2006, "MQRC_CHAR_ATTR_LENGTH_ERROR" },
   {        2007, "MQCA_ENV_DATA" },
   {        2007, "MQRC_CHAR_ATTRS_ERROR" },
   {        2008, "MQCA_INITIATION_Q_NAME" },
   {        2008, "MQRC_CHAR_ATTRS_TOO_SHORT" },
   {        2009, "MQCA_NAMELIST_DESC" },
   {        2009, "MQRC_CONNECTION_BROKEN" },
   {        2010, "MQCA_NAMELIST_NAME" },
   {        2010, "MQRC_DATA_LENGTH_ERROR" },
   {        2011, "MQCA_PROCESS_DESC" },
   {        2011, "MQRC_DYNAMIC_Q_NAME_ERROR" },
   {        2012, "MQCA_PROCESS_NAME" },
   {        2012, "MQRC_ENVIRONMENT_ERROR" },
   {        2013, "MQCA_Q_DESC" },
   {        2013, "MQRC_EXPIRY_ERROR" },
   {        2014, "MQCA_Q_MGR_DESC" },
   {        2014, "MQRC_FEEDBACK_ERROR" },
   {        2015, "MQCA_Q_MGR_NAME" },
   {        2016, "MQCA_Q_NAME" },
   {        2016, "MQRC_GET_INHIBITED" },
   {        2017, "MQCA_REMOTE_Q_MGR_NAME" },
   {        2017, "MQRC_HANDLE_NOT_AVAILABLE" },
   {        2018, "MQCA_REMOTE_Q_NAME" },
   {        2018, "MQRC_HCONN_ERROR" },
   {        2019, "MQCA_BACKOUT_REQ_Q_NAME" },
   {        2019, "MQRC_HOBJ_ERROR" },
   {        2020, "MQCA_NAMES" },
   {        2020, "MQRC_INHIBIT_VALUE_ERROR" },
   {        2021, "MQCA_USER_DATA" },
   {        2021, "MQRC_INT_ATTR_COUNT_ERROR" },
   {        2022, "MQCA_STORAGE_CLASS" },
   {        2022, "MQRC_INT_ATTR_COUNT_TOO_SMALL" },
   {        2023, "MQCA_TRIGGER_DATA" },
   {        2023, "MQRC_INT_ATTRS_ARRAY_ERROR" },
   {        2024, "MQCA_XMIT_Q_NAME" },
   {        2024, "MQRC_SYNCPOINT_LIMIT_REACHED" },
   {        2025, "MQCA_DEF_XMIT_Q_NAME" },
   {        2025, "MQRC_MAX_CONNS_LIMIT_REACHED" },
   {        2026, "MQCA_CHANNEL_AUTO_DEF_EXIT" },
   {        2026, "MQRC_MD_ERROR" },
   {        2027, "MQCA_ALTERATION_DATE" },
   {        2027, "MQRC_MISSING_REPLY_TO_Q" },
   {        2028, "MQCA_ALTERATION_TIME" },
   {        2029, "MQCA_CLUSTER_NAME" },
   {        2029, "MQRC_MSG_TYPE_ERROR" },
   {        2030, "MQCA_CLUSTER_NAMELIST" },
   {        2030, "MQRC_MSG_TOO_BIG_FOR_Q" },
   {        2031, "MQCA_CLUSTER_Q_MGR_NAME" },
   {        2031, "MQRC_MSG_TOO_BIG_FOR_Q_MGR" },
   {        2032, "MQCA_Q_MGR_IDENTIFIER" },
   {        2033, "MQCA_CLUSTER_WORKLOAD_EXIT" },
   {        2033, "MQRC_NO_MSG_AVAILABLE" },
   {        2034, "MQCA_CLUSTER_WORKLOAD_DATA" },
   {        2034, "MQRC_NO_MSG_UNDER_CURSOR" },
   {        2035, "MQCA_REPOSITORY_NAME" },
   {        2035, "MQRC_NOT_AUTHORIZED" },
   {        2036, "MQCA_REPOSITORY_NAMELIST" },
   {        2036, "MQRC_NOT_OPEN_FOR_BROWSE" },
   {        2037, "MQCA_CLUSTER_DATE" },
   {        2037, "MQRC_NOT_OPEN_FOR_INPUT" },
   {        2038, "MQCA_CLUSTER_TIME" },
   {        2038, "MQRC_NOT_OPEN_FOR_INQUIRE" },
   {        2039, "MQCA_CF_STRUC_NAME" },
   {        2039, "MQRC_NOT_OPEN_FOR_OUTPUT" },
   {        2040, "MQCA_QSG_NAME" },
   {        2040, "MQRC_NOT_OPEN_FOR_SET" },
   {        2041, "MQCA_IGQ_USER_ID" },
   {        2041, "MQRC_OBJECT_CHANGED" },
   {        2042, "MQCA_STORAGE_CLASS_DESC" },
   {        2042, "MQRC_OBJECT_IN_USE" },
   {        2043, "MQCA_XCF_GROUP_NAME" },
   {        2043, "MQRC_OBJECT_TYPE_ERROR" },
   {        2044, "MQCA_XCF_MEMBER_NAME" },
   {        2044, "MQRC_OD_ERROR" },
   {        2045, "MQCA_AUTH_INFO_NAME" },
   {        2045, "MQRC_OPTION_NOT_VALID_FOR_TYPE" },
   {        2046, "MQCA_AUTH_INFO_DESC" },
   {        2046, "MQRC_OPTIONS_ERROR" },
   {        2047, "MQCA_LDAP_USER_NAME" },
   {        2047, "MQRC_PERSISTENCE_ERROR" },
   {        2048, "MQCA_LDAP_PASSWORD" },
   {        2048, "MQCNO_CLIENT_BINDING" },
   {        2048, "MQGMO_BROWSE_MSG_UNDER_CURSOR" },
   {        2048, "MQOO_SET_ALL_CONTEXT" },
   {        2048, "MQPMO_SET_ALL_CONTEXT" },
   {        2048, "MQRC_PERSISTENT_NOT_ALLOWED" },
   {        2048, "MQREGO_PERSISTENT" },
   {        2048, "MQRO_COD" },
   {        2048, "MQSO_PUBLICATIONS_ON_REQUEST" },
   {        2048, "MQZAO_PUBLISH" },
   {        2049, "MQCA_SSL_KEY_REPOSITORY" },
   {        2049, "MQRC_PRIORITY_EXCEEDS_MAXIMUM" },
   {        2050, "MQCA_SSL_CRL_NAMELIST" },
   {        2050, "MQRC_PRIORITY_ERROR" },
   {        2051, "MQCA_SSL_CRYPTO_HARDWARE" },
   {        2051, "MQRC_PUT_INHIBITED" },
   {        2052, "MQCA_CF_STRUC_DESC" },
   {        2052, "MQRC_Q_DELETED" },
   {        2053, "MQCA_AUTH_INFO_CONN_NAME" },
   {        2053, "MQRC_Q_FULL" },
   {        2055, "MQRC_Q_NOT_EMPTY" },
   {        2056, "MQRC_Q_SPACE_NOT_AVAILABLE" },
   {        2057, "MQRC_Q_TYPE_ERROR" },
   {        2058, "MQRC_Q_MGR_NAME_ERROR" },
   {        2059, "MQRC_Q_MGR_NOT_AVAILABLE" },
   {        2060, "MQCA_CICS_FILE_NAME" },
   {        2061, "MQCA_TRIGGER_TRANS_ID" },
   {        2061, "MQRC_REPORT_OPTIONS_ERROR" },
   {        2062, "MQCA_TRIGGER_PROGRAM_NAME" },
   {        2062, "MQRC_SECOND_MARK_NOT_ALLOWED" },
   {        2063, "MQCA_TRIGGER_TERM_ID" },
   {        2063, "MQRC_SECURITY_ERROR" },
   {        2064, "MQCA_TRIGGER_CHANNEL_NAME" },
   {        2065, "MQCA_SYSTEM_LOG_Q_NAME" },
   {        2065, "MQRC_SELECTOR_COUNT_ERROR" },
   {        2066, "MQCA_MONITOR_Q_NAME" },
   {        2066, "MQRC_SELECTOR_LIMIT_EXCEEDED" },
   {        2067, "MQCA_COMMAND_REPLY_Q_NAME" },
   {        2067, "MQRC_SELECTOR_ERROR" },
   {        2068, "MQCA_BATCH_INTERFACE_ID" },
   {        2068, "MQRC_SELECTOR_NOT_FOR_TYPE" },
   {        2069, "MQCA_SSL_KEY_LIBRARY" },
   {        2069, "MQRC_SIGNAL_OUTSTANDING" },
   {        2070, "MQCA_SSL_KEY_MEMBER" },
   {        2070, "MQRC_SIGNAL_REQUEST_ACCEPTED" },
   {        2071, "MQCA_DNS_GROUP" },
   {        2071, "MQRC_STORAGE_NOT_AVAILABLE" },
   {        2072, "MQCA_LU_GROUP_NAME" },
   {        2072, "MQRC_SYNCPOINT_NOT_AVAILABLE" },
   {        2073, "MQCA_LU_NAME" },
   {        2074, "MQCA_LU62_ARM_SUFFIX" },
   {        2075, "MQCA_TCP_NAME" },
   {        2075, "MQRC_TRIGGER_CONTROL_ERROR" },
   {        2076, "MQCA_CHINIT_SERVICE_PARM" },
   {        2076, "MQRC_TRIGGER_DEPTH_ERROR" },
   {        2077, "MQCA_SERVICE_NAME" },
   {        2077, "MQRC_TRIGGER_MSG_PRIORITY_ERR" },
   {        2078, "MQCA_SERVICE_DESC" },
   {        2078, "MQRC_TRIGGER_TYPE_ERROR" },
   {        2079, "MQCA_SERVICE_START_COMMAND" },
   {        2079, "MQRC_TRUNCATED_MSG_ACCEPTED" },
   {        2080, "MQCA_SERVICE_START_ARGS" },
   {        2080, "MQRC_TRUNCATED_MSG_FAILED" },
   {        2081, "MQCA_SERVICE_STOP_COMMAND" },
   {        2082, "MQCA_SERVICE_STOP_ARGS" },
   {        2082, "MQRC_UNKNOWN_ALIAS_BASE_Q" },
   {        2083, "MQCA_STDOUT_DESTINATION" },
   {        2084, "MQCA_STDERR_DESTINATION" },
   {        2085, "MQCA_TPIPE_NAME" },
   {        2085, "MQRC_UNKNOWN_OBJECT_NAME" },
   {        2086, "MQCA_PASS_TICKET_APPL" },
   {        2086, "MQRC_UNKNOWN_OBJECT_Q_MGR" },
   {        2087, "MQRC_UNKNOWN_REMOTE_Q_MGR" },
   {        2090, "MQCA_AUTO_REORG_START_TIME" },
   {        2090, "MQRC_WAIT_INTERVAL_ERROR" },
   {        2091, "MQCA_AUTO_REORG_CATALOG" },
   {        2091, "MQRC_XMIT_Q_TYPE_ERROR" },
   {        2092, "MQCA_TOPIC_NAME" },
   {        2092, "MQRC_XMIT_Q_USAGE_ERROR" },
   {        2093, "MQCA_TOPIC_DESC" },
   {        2093, "MQRC_NOT_OPEN_FOR_PASS_ALL" },
   {        2094, "MQCA_TOPIC_STRING" },
   {        2094, "MQRC_NOT_OPEN_FOR_PASS_IDENT" },
   {        2095, "MQRC_NOT_OPEN_FOR_SET_ALL" },
   {        2096, "MQCA_MODEL_DURABLE_Q" },
   {        2096, "MQRC_NOT_OPEN_FOR_SET_IDENT" },
   {        2097, "MQCA_MODEL_NON_DURABLE_Q" },
   {        2097, "MQRC_CONTEXT_HANDLE_ERROR" },
   {        2098, "MQCA_RESUME_DATE" },
   {        2098, "MQRC_CONTEXT_NOT_AVAILABLE" },
   {        2099, "MQCA_RESUME_TIME" },
   {        2099, "MQRC_SIGNAL1_ERROR" },
   {        2100, "MQRC_OBJECT_ALREADY_EXISTS" },
   {        2101, "MQCA_CHILD" },
   {        2101, "MQRC_OBJECT_DAMAGED" },
   {        2102, "MQCA_PARENT" },
   {        2102, "MQRC_RESOURCE_PROBLEM" },
   {        2103, "MQRC_ANOTHER_Q_MGR_CONNECTED" },
   {        2104, "MQRC_UNKNOWN_REPORT_OPTION" },
   {        2105, "MQCA_ADMIN_TOPIC_NAME" },
   {        2105, "MQRC_STORAGE_CLASS_ERROR" },
   {        2106, "MQRC_COD_NOT_VALID_FOR_XCF_Q" },
   {        2107, "MQRC_XWAIT_CANCELED" },
   {        2108, "MQCA_TOPIC_STRING_FILTER" },
   {        2108, "MQRC_XWAIT_ERROR" },
   {        2109, "MQCA_AUTH_INFO_OCSP_URL" },
   {        2109, "MQRC_SUPPRESSED_BY_EXIT" },
   {        2110, "MQCA_COMM_INFO_NAME" },
   {        2110, "MQRC_FORMAT_ERROR" },
   {        2111, "MQCA_COMM_INFO_DESC" },
   {        2111, "MQRC_SOURCE_CCSID_ERROR" },
   {        2112, "MQCA_POLICY_NAME" },
   {        2112, "MQRC_SOURCE_INTEGER_ENC_ERROR" },
   {        2113, "MQCA_SIGNER_DN" },
   {        2113, "MQRC_SOURCE_DECIMAL_ENC_ERROR" },
   {        2114, "MQCA_RECIPIENT_DN" },
   {        2114, "MQRC_SOURCE_FLOAT_ENC_ERROR" },
   {        2115, "MQCA_INSTALLATION_DESC" },
   {        2115, "MQRC_TARGET_CCSID_ERROR" },
   {        2116, "MQCA_INSTALLATION_NAME" },
   {        2116, "MQRC_TARGET_INTEGER_ENC_ERROR" },
   {        2117, "MQCA_INSTALLATION_PATH" },
   {        2117, "MQRC_TARGET_DECIMAL_ENC_ERROR" },
   {        2118, "MQCA_CHLAUTH_DESC" },
   {        2118, "MQRC_TARGET_FLOAT_ENC_ERROR" },
   {        2119, "MQCA_CUSTOM" },
   {        2119, "MQRC_NOT_CONVERTED" },
   {        2120, "MQCA_VERSION" },
   {        2120, "MQRC_CONVERTED_MSG_TOO_BIG" },
   {        2121, "MQCA_CERT_LABEL" },
   {        2121, "MQRC_NO_EXTERNAL_PARTICIPANTS" },
   {        2122, "MQCA_XR_VERSION" },
   {        2122, "MQRC_PARTICIPANT_NOT_AVAILABLE" },
   {        2123, "MQCA_XR_SSL_CIPHER_SUITES" },
   {        2123, "MQRC_OUTCOME_MIXED" },
   {        2124, "MQCA_CLUS_CHL_NAME" },
   {        2124, "MQRC_OUTCOME_PENDING" },
   {        2125, "MQCA_CONN_AUTH" },
   {        2125, "MQRC_BRIDGE_STARTED" },
   {        2126, "MQCA_LDAP_BASE_DN_USERS" },
   {        2126, "MQRC_BRIDGE_STOPPED" },
   {        2127, "MQCA_LDAP_SHORT_USER_FIELD" },
   {        2127, "MQRC_ADAPTER_STORAGE_SHORTAGE" },
   {        2128, "MQCA_LDAP_USER_OBJECT_CLASS" },
   {        2128, "MQRC_UOW_IN_PROGRESS" },
   {        2129, "MQCA_LDAP_USER_ATTR_FIELD" },
   {        2129, "MQRC_ADAPTER_CONN_LOAD_ERROR" },
   {        2130, "MQCA_SSL_CERT_ISSUER_NAME" },
   {        2130, "MQRC_ADAPTER_SERV_LOAD_ERROR" },
   {        2131, "MQCA_QSG_CERT_LABEL" },
   {        2131, "MQRC_ADAPTER_DEFS_ERROR" },
   {        2132, "MQCA_LDAP_BASE_DN_GROUPS" },
   {        2132, "MQRC_ADAPTER_DEFS_LOAD_ERROR" },
   {        2133, "MQCA_LDAP_GROUP_OBJECT_CLASS" },
   {        2133, "MQRC_ADAPTER_CONV_LOAD_ERROR" },
   {        2134, "MQCA_LDAP_GROUP_ATTR_FIELD" },
   {        2134, "MQRC_BO_ERROR" },
   {        2135, "MQCA_LDAP_FIND_GROUP_FIELD" },
   {        2135, "MQRC_DH_ERROR" },
   {        2136, "MQCA_AMQP_VERSION" },
   {        2136, "MQRC_MULTIPLE_REASONS" },
   {        2137, "MQCA_AMQP_SSL_CIPHER_SUITES" },
   {        2137, "MQCA_LAST_USED" },
   {        2137, "MQRC_OPEN_FAILED" },
   {        2138, "MQRC_ADAPTER_DISC_LOAD_ERROR" },
   {        2139, "MQRC_CNO_ERROR" },
   {        2140, "MQRC_CICS_WAIT_FAILED" },
   {        2141, "MQRC_DLH_ERROR" },
   {        2142, "MQRC_HEADER_ERROR" },
   {        2143, "MQRC_SOURCE_LENGTH_ERROR" },
   {        2144, "MQRC_TARGET_LENGTH_ERROR" },
   {        2145, "MQRC_SOURCE_BUFFER_ERROR" },
   {        2146, "MQRC_TARGET_BUFFER_ERROR" },
   {        2148, "MQRC_IIH_ERROR" },
   {        2149, "MQRC_PCF_ERROR" },
   {        2150, "MQRC_DBCS_ERROR" },
   {        2152, "MQRC_OBJECT_NAME_ERROR" },
   {        2153, "MQRC_OBJECT_Q_MGR_NAME_ERROR" },
   {        2154, "MQRC_RECS_PRESENT_ERROR" },
   {        2155, "MQRC_OBJECT_RECORDS_ERROR" },
   {        2156, "MQRC_RESPONSE_RECORDS_ERROR" },
   {        2157, "MQRC_ASID_MISMATCH" },
   {        2158, "MQRC_PMO_RECORD_FLAGS_ERROR" },
   {        2159, "MQRC_PUT_MSG_RECORDS_ERROR" },
   {        2160, "MQRC_CONN_ID_IN_USE" },
   {        2161, "MQRC_Q_MGR_QUIESCING" },
   {        2162, "MQRC_Q_MGR_STOPPING" },
   {        2163, "MQRC_DUPLICATE_RECOV_COORD" },
   {        2173, "MQRC_PMO_ERROR" },
   {        2182, "MQRC_API_EXIT_NOT_FOUND" },
   {        2183, "MQRC_API_EXIT_LOAD_ERROR" },
   {        2184, "MQRC_REMOTE_Q_NAME_ERROR" },
   {        2185, "MQRC_INCONSISTENT_PERSISTENCE" },
   {        2186, "MQRC_GMO_ERROR" },
   {        2187, "MQRC_CICS_BRIDGE_RESTRICTION" },
   {        2188, "MQRC_STOPPED_BY_CLUSTER_EXIT" },
   {        2189, "MQRC_CLUSTER_RESOLUTION_ERROR" },
   {        2190, "MQRC_CONVERTED_STRING_TOO_BIG" },
   {        2191, "MQRC_TMC_ERROR" },
   {        2192, "MQRC_STORAGE_MEDIUM_FULL" },
   {        2193, "MQRC_PAGESET_ERROR" },
   {        2194, "MQRC_NAME_NOT_VALID_FOR_TYPE" },
   {        2195, "MQRC_UNEXPECTED_ERROR" },
   {        2196, "MQRC_UNKNOWN_XMIT_Q" },
   {        2197, "MQRC_UNKNOWN_DEF_XMIT_Q" },
   {        2198, "MQRC_DEF_XMIT_Q_TYPE_ERROR" },
   {        2199, "MQRC_DEF_XMIT_Q_USAGE_ERROR" },
   {        2200, "MQRC_MSG_MARKED_BROWSE_CO_OP" },
   {        2201, "MQRC_NAME_IN_USE" },
   {        2202, "MQRC_CONNECTION_QUIESCING" },
   {        2203, "MQRC_CONNECTION_STOPPING" },
   {        2204, "MQRC_ADAPTER_NOT_AVAILABLE" },
   {        2206, "MQRC_MSG_ID_ERROR" },
   {        2207, "MQRC_CORREL_ID_ERROR" },
   {        2208, "MQRC_FILE_SYSTEM_ERROR" },
   {        2209, "MQRC_NO_MSG_LOCKED" },
   {        2210, "MQRC_SOAP_DOTNET_ERROR" },
   {        2211, "MQRC_SOAP_AXIS_ERROR" },
   {        2212, "MQRC_SOAP_URL_ERROR" },
   {        2216, "MQRC_FILE_NOT_AUDITED" },
   {        2217, "MQRC_CONNECTION_NOT_AUTHORIZED" },
   {        2218, "MQRC_MSG_TOO_BIG_FOR_CHANNEL" },
   {        2219, "MQRC_CALL_IN_PROGRESS" },
   {        2220, "MQRC_RMH_ERROR" },
   {        2222, "MQRC_Q_MGR_ACTIVE" },
   {        2223, "MQRC_Q_MGR_NOT_ACTIVE" },
   {        2224, "MQRC_Q_DEPTH_HIGH" },
   {        2225, "MQRC_Q_DEPTH_LOW" },
   {        2226, "MQRC_Q_SERVICE_INTERVAL_HIGH" },
   {        2227, "MQRC_Q_SERVICE_INTERVAL_OK" },
   {        2228, "MQRC_RFH_HEADER_FIELD_ERROR" },
   {        2229, "MQRC_RAS_PROPERTY_ERROR" },
   {        2232, "MQRC_UNIT_OF_WORK_NOT_STARTED" },
   {        2233, "MQRC_CHANNEL_AUTO_DEF_OK" },
   {        2234, "MQRC_CHANNEL_AUTO_DEF_ERROR" },
   {        2235, "MQRC_CFH_ERROR" },
   {        2236, "MQRC_CFIL_ERROR" },
   {        2237, "MQRC_CFIN_ERROR" },
   {        2238, "MQRC_CFSL_ERROR" },
   {        2239, "MQRC_CFST_ERROR" },
   {        2241, "MQRC_INCOMPLETE_GROUP" },
   {        2242, "MQRC_INCOMPLETE_MSG" },
   {        2243, "MQRC_INCONSISTENT_CCSIDS" },
   {        2244, "MQRC_INCONSISTENT_ENCODINGS" },
   {        2245, "MQRC_INCONSISTENT_UOW" },
   {        2246, "MQRC_INVALID_MSG_UNDER_CURSOR" },
   {        2247, "MQRC_MATCH_OPTIONS_ERROR" },
   {        2248, "MQRC_MDE_ERROR" },
   {        2249, "MQRC_MSG_FLAGS_ERROR" },
   {        2250, "MQRC_MSG_SEQ_NUMBER_ERROR" },
   {        2251, "MQRC_OFFSET_ERROR" },
   {        2252, "MQRC_ORIGINAL_LENGTH_ERROR" },
   {        2253, "MQRC_SEGMENT_LENGTH_ZERO" },
   {        2255, "MQRC_UOW_NOT_AVAILABLE" },
   {        2256, "MQRC_WRONG_GMO_VERSION" },
   {        2257, "MQRC_WRONG_MD_VERSION" },
   {        2258, "MQRC_GROUP_ID_ERROR" },
   {        2259, "MQRC_INCONSISTENT_BROWSE" },
   {        2260, "MQRC_XQH_ERROR" },
   {        2261, "MQRC_SRC_ENV_ERROR" },
   {        2262, "MQRC_SRC_NAME_ERROR" },
   {        2263, "MQRC_DEST_ENV_ERROR" },
   {        2264, "MQRC_DEST_NAME_ERROR" },
   {        2265, "MQRC_TM_ERROR" },
   {        2266, "MQRC_CLUSTER_EXIT_ERROR" },
   {        2267, "MQRC_CLUSTER_EXIT_LOAD_ERROR" },
   {        2268, "MQRC_CLUSTER_PUT_INHIBITED" },
   {        2269, "MQRC_CLUSTER_RESOURCE_ERROR" },
   {        2270, "MQRC_NO_DESTINATIONS_AVAILABLE" },
   {        2271, "MQRC_CONN_TAG_IN_USE" },
   {        2272, "MQRC_PARTIALLY_CONVERTED" },
   {        2273, "MQRC_CONNECTION_ERROR" },
   {        2274, "MQRC_OPTION_ENVIRONMENT_ERROR" },
   {        2277, "MQRC_CD_ERROR" },
   {        2278, "MQRC_CLIENT_CONN_ERROR" },
   {        2279, "MQRC_CHANNEL_STOPPED_BY_USER" },
   {        2280, "MQRC_HCONFIG_ERROR" },
   {        2281, "MQRC_FUNCTION_ERROR" },
   {        2282, "MQRC_CHANNEL_STARTED" },
   {        2283, "MQRC_CHANNEL_STOPPED" },
   {        2284, "MQRC_CHANNEL_CONV_ERROR" },
   {        2285, "MQRC_SERVICE_NOT_AVAILABLE" },
   {        2286, "MQRC_INITIALIZATION_FAILED" },
   {        2287, "MQRC_TERMINATION_FAILED" },
   {        2288, "MQRC_UNKNOWN_Q_NAME" },
   {        2289, "MQRC_SERVICE_ERROR" },
   {        2290, "MQRC_Q_ALREADY_EXISTS" },
   {        2291, "MQRC_USER_ID_NOT_AVAILABLE" },
   {        2292, "MQRC_UNKNOWN_ENTITY" },
   {        2293, "MQRC_UNKNOWN_AUTH_ENTITY" },
   {        2294, "MQRC_UNKNOWN_REF_OBJECT" },
   {        2295, "MQRC_CHANNEL_ACTIVATED" },
   {        2296, "MQRC_CHANNEL_NOT_ACTIVATED" },
   {        2297, "MQRC_UOW_CANCELED" },
   {        2298, "MQRC_FUNCTION_NOT_SUPPORTED" },
   {        2299, "MQRC_SELECTOR_TYPE_ERROR" },
   {        2300, "MQRC_COMMAND_TYPE_ERROR" },
   {        2301, "MQRC_MULTIPLE_INSTANCE_ERROR" },
   {        2302, "MQRC_SYSTEM_ITEM_NOT_ALTERABLE" },
   {        2303, "MQRC_BAG_CONVERSION_ERROR" },
   {        2304, "MQRC_SELECTOR_OUT_OF_RANGE" },
   {        2305, "MQRC_SELECTOR_NOT_UNIQUE" },
   {        2306, "MQRC_INDEX_NOT_PRESENT" },
   {        2307, "MQRC_STRING_ERROR" },
   {        2308, "MQRC_ENCODING_NOT_SUPPORTED" },
   {        2309, "MQRC_SELECTOR_NOT_PRESENT" },
   {        2310, "MQRC_OUT_SELECTOR_ERROR" },
   {        2311, "MQRC_STRING_TRUNCATED" },
   {        2312, "MQRC_SELECTOR_WRONG_TYPE" },
   {        2313, "MQRC_INCONSISTENT_ITEM_TYPE" },
   {        2314, "MQRC_INDEX_ERROR" },
   {        2315, "MQRC_SYSTEM_BAG_NOT_ALTERABLE" },
   {        2316, "MQRC_ITEM_COUNT_ERROR" },
   {        2317, "MQRC_FORMAT_NOT_SUPPORTED" },
   {        2318, "MQRC_SELECTOR_NOT_SUPPORTED" },
   {        2319, "MQRC_ITEM_VALUE_ERROR" },
   {        2320, "MQRC_HBAG_ERROR" },
   {        2321, "MQRC_PARAMETER_MISSING" },
   {        2322, "MQRC_CMD_SERVER_NOT_AVAILABLE" },
   {        2323, "MQRC_STRING_LENGTH_ERROR" },
   {        2324, "MQRC_INQUIRY_COMMAND_ERROR" },
   {        2325, "MQRC_NESTED_BAG_NOT_SUPPORTED" },
   {        2326, "MQRC_BAG_WRONG_TYPE" },
   {        2327, "MQRC_ITEM_TYPE_ERROR" },
   {        2328, "MQRC_SYSTEM_BAG_NOT_DELETABLE" },
   {        2329, "MQRC_SYSTEM_ITEM_NOT_DELETABLE" },
   {        2330, "MQRC_CODED_CHAR_SET_ID_ERROR" },
   {        2331, "MQRC_MSG_TOKEN_ERROR" },
   {        2332, "MQRC_MISSING_WIH" },
   {        2333, "MQRC_WIH_ERROR" },
   {        2334, "MQRC_RFH_ERROR" },
   {        2335, "MQRC_RFH_STRING_ERROR" },
   {        2336, "MQRC_RFH_COMMAND_ERROR" },
   {        2337, "MQRC_RFH_PARM_ERROR" },
   {        2338, "MQRC_RFH_DUPLICATE_PARM" },
   {        2339, "MQRC_RFH_PARM_MISSING" },
   {        2340, "MQRC_CHAR_CONVERSION_ERROR" },
   {        2341, "MQRC_UCS2_CONVERSION_ERROR" },
   {        2342, "MQRC_DB2_NOT_AVAILABLE" },
   {        2343, "MQRC_OBJECT_NOT_UNIQUE" },
   {        2344, "MQRC_CONN_TAG_NOT_RELEASED" },
   {        2345, "MQRC_CF_NOT_AVAILABLE" },
   {        2346, "MQRC_CF_STRUC_IN_USE" },
   {        2347, "MQRC_CF_STRUC_LIST_HDR_IN_USE" },
   {        2348, "MQRC_CF_STRUC_AUTH_FAILED" },
   {        2349, "MQRC_CF_STRUC_ERROR" },
   {        2350, "MQRC_CONN_TAG_NOT_USABLE" },
   {        2351, "MQRC_GLOBAL_UOW_CONFLICT" },
   {        2352, "MQRC_LOCAL_UOW_CONFLICT" },
   {        2353, "MQRC_HANDLE_IN_USE_FOR_UOW" },
   {        2354, "MQRC_UOW_ENLISTMENT_ERROR" },
   {        2355, "MQRC_UOW_MIX_NOT_SUPPORTED" },
   {        2356, "MQRC_WXP_ERROR" },
   {        2357, "MQRC_CURRENT_RECORD_ERROR" },
   {        2358, "MQRC_NEXT_OFFSET_ERROR" },
   {        2359, "MQRC_NO_RECORD_AVAILABLE" },
   {        2360, "MQRC_OBJECT_LEVEL_INCOMPATIBLE" },
   {        2361, "MQRC_NEXT_RECORD_ERROR" },
   {        2362, "MQRC_BACKOUT_THRESHOLD_REACHED" },
   {        2363, "MQRC_MSG_NOT_MATCHED" },
   {        2364, "MQRC_JMS_FORMAT_ERROR" },
   {        2365, "MQRC_SEGMENTS_NOT_SUPPORTED" },
   {        2366, "MQRC_WRONG_CF_LEVEL" },
   {        2367, "MQRC_CONFIG_CREATE_OBJECT" },
   {        2368, "MQRC_CONFIG_CHANGE_OBJECT" },
   {        2369, "MQRC_CONFIG_DELETE_OBJECT" },
   {        2370, "MQRC_CONFIG_REFRESH_OBJECT" },
   {        2371, "MQRC_CHANNEL_SSL_ERROR" },
   {        2372, "MQRC_PARTICIPANT_NOT_DEFINED" },
   {        2373, "MQRC_CF_STRUC_FAILED" },
   {        2374, "MQRC_API_EXIT_ERROR" },
   {        2375, "MQRC_API_EXIT_INIT_ERROR" },
   {        2376, "MQRC_API_EXIT_TERM_ERROR" },
   {        2377, "MQRC_EXIT_REASON_ERROR" },
   {        2378, "MQRC_RESERVED_VALUE_ERROR" },
   {        2379, "MQRC_NO_DATA_AVAILABLE" },
   {        2380, "MQRC_SCO_ERROR" },
   {        2381, "MQRC_KEY_REPOSITORY_ERROR" },
   {        2382, "MQRC_CRYPTO_HARDWARE_ERROR" },
   {        2383, "MQRC_AUTH_INFO_REC_COUNT_ERROR" },
   {        2384, "MQRC_AUTH_INFO_REC_ERROR" },
   {        2385, "MQRC_AIR_ERROR" },
   {        2386, "MQRC_AUTH_INFO_TYPE_ERROR" },
   {        2387, "MQRC_AUTH_INFO_CONN_NAME_ERROR" },
   {        2388, "MQRC_LDAP_USER_NAME_ERROR" },
   {        2389, "MQRC_LDAP_USER_NAME_LENGTH_ERR" },
   {        2390, "MQRC_LDAP_PASSWORD_ERROR" },
   {        2391, "MQRC_SSL_ALREADY_INITIALIZED" },
   {        2392, "MQRC_SSL_CONFIG_ERROR" },
   {        2393, "MQRC_SSL_INITIALIZATION_ERROR" },
   {        2394, "MQRC_Q_INDEX_TYPE_ERROR" },
   {        2395, "MQRC_CFBS_ERROR" },
   {        2396, "MQRC_SSL_NOT_ALLOWED" },
   {        2397, "MQRC_JSSE_ERROR" },
   {        2398, "MQRC_SSL_PEER_NAME_MISMATCH" },
   {        2399, "MQRC_SSL_PEER_NAME_ERROR" },
   {        2400, "MQRC_UNSUPPORTED_CIPHER_SUITE" },
   {        2401, "MQRC_SSL_CERTIFICATE_REVOKED" },
   {        2402, "MQRC_SSL_CERT_STORE_ERROR" },
   {        2406, "MQRC_CLIENT_EXIT_LOAD_ERROR" },
   {        2407, "MQRC_CLIENT_EXIT_ERROR" },
   {        2408, "MQRC_UOW_COMMITTED" },
   {        2409, "MQRC_SSL_KEY_RESET_ERROR" },
   {        2410, "MQRC_UNKNOWN_COMPONENT_NAME" },
   {        2411, "MQRC_LOGGER_STATUS" },
   {        2412, "MQRC_COMMAND_MQSC" },
   {        2413, "MQRC_COMMAND_PCF" },
   {        2414, "MQRC_CFIF_ERROR" },
   {        2415, "MQRC_CFSF_ERROR" },
   {        2416, "MQRC_CFGR_ERROR" },
   {        2417, "MQRC_MSG_NOT_ALLOWED_IN_GROUP" },
   {        2418, "MQRC_FILTER_OPERATOR_ERROR" },
   {        2419, "MQRC_NESTED_SELECTOR_ERROR" },
   {        2420, "MQRC_EPH_ERROR" },
   {        2421, "MQRC_RFH_FORMAT_ERROR" },
   {        2422, "MQRC_CFBF_ERROR" },
   {        2423, "MQRC_CLIENT_CHANNEL_CONFLICT" },
   {        2424, "MQRC_SD_ERROR" },
   {        2425, "MQRC_TOPIC_STRING_ERROR" },
   {        2426, "MQRC_STS_ERROR" },
   {        2428, "MQRC_NO_SUBSCRIPTION" },
   {        2429, "MQRC_SUBSCRIPTION_IN_USE" },
   {        2430, "MQRC_STAT_TYPE_ERROR" },
   {        2431, "MQRC_SUB_USER_DATA_ERROR" },
   {        2432, "MQRC_SUB_ALREADY_EXISTS" },
   {        2434, "MQRC_IDENTITY_MISMATCH" },
   {        2435, "MQRC_ALTER_SUB_ERROR" },
   {        2436, "MQRC_DURABILITY_NOT_ALLOWED" },
   {        2437, "MQRC_NO_RETAINED_MSG" },
   {        2438, "MQRC_SRO_ERROR" },
   {        2440, "MQRC_SUB_NAME_ERROR" },
   {        2441, "MQRC_OBJECT_STRING_ERROR" },
   {        2442, "MQRC_PROPERTY_NAME_ERROR" },
   {        2443, "MQRC_SEGMENTATION_NOT_ALLOWED" },
   {        2444, "MQRC_CBD_ERROR" },
   {        2445, "MQRC_CTLO_ERROR" },
   {        2446, "MQRC_NO_CALLBACKS_ACTIVE" },
   {        2448, "MQRC_CALLBACK_NOT_REGISTERED" },
   {        2457, "MQRC_OPTIONS_CHANGED" },
   {        2458, "MQRC_READ_AHEAD_MSGS" },
   {        2459, "MQRC_SELECTOR_SYNTAX_ERROR" },
   {        2460, "MQRC_HMSG_ERROR" },
   {        2461, "MQRC_CMHO_ERROR" },
   {        2462, "MQRC_DMHO_ERROR" },
   {        2463, "MQRC_SMPO_ERROR" },
   {        2464, "MQRC_IMPO_ERROR" },
   {        2465, "MQRC_PROPERTY_NAME_TOO_BIG" },
   {        2466, "MQRC_PROP_VALUE_NOT_CONVERTED" },
   {        2467, "MQRC_PROP_TYPE_NOT_SUPPORTED" },
   {        2469, "MQRC_PROPERTY_VALUE_TOO_BIG" },
   {        2470, "MQRC_PROP_CONV_NOT_SUPPORTED" },
   {        2471, "MQRC_PROPERTY_NOT_AVAILABLE" },
   {        2472, "MQRC_PROP_NUMBER_FORMAT_ERROR" },
   {        2473, "MQRC_PROPERTY_TYPE_ERROR" },
   {        2478, "MQRC_PROPERTIES_TOO_BIG" },
   {        2479, "MQRC_PUT_NOT_RETAINED" },
   {        2480, "MQRC_ALIAS_TARGTYPE_CHANGED" },
   {        2481, "MQRC_DMPO_ERROR" },
   {        2482, "MQRC_PD_ERROR" },
   {        2483, "MQRC_CALLBACK_TYPE_ERROR" },
   {        2484, "MQRC_CBD_OPTIONS_ERROR" },
   {        2485, "MQRC_MAX_MSG_LENGTH_ERROR" },
   {        2486, "MQRC_CALLBACK_ROUTINE_ERROR" },
   {        2487, "MQRC_CALLBACK_LINK_ERROR" },
   {        2488, "MQRC_OPERATION_ERROR" },
   {        2489, "MQRC_BMHO_ERROR" },
   {        2490, "MQRC_UNSUPPORTED_PROPERTY" },
   {        2492, "MQRC_PROP_NAME_NOT_CONVERTED" },
   {        2494, "MQRC_GET_ENABLED" },
   {        2495, "MQRC_MODULE_NOT_FOUND" },
   {        2496, "MQRC_MODULE_INVALID" },
   {        2497, "MQRC_MODULE_ENTRY_NOT_FOUND" },
   {        2498, "MQRC_MIXED_CONTENT_NOT_ALLOWED" },
   {        2499, "MQRC_MSG_HANDLE_IN_USE" },
   {        2500, "MQRC_HCONN_ASYNC_ACTIVE" },
   {        2501, "MQRC_MHBO_ERROR" },
   {        2502, "MQRC_PUBLICATION_FAILURE" },
   {        2503, "MQRC_SUB_INHIBITED" },
   {        2504, "MQRC_SELECTOR_ALWAYS_FALSE" },
   {        2507, "MQRC_XEPO_ERROR" },
   {        2509, "MQRC_DURABILITY_NOT_ALTERABLE" },
   {        2510, "MQRC_TOPIC_NOT_ALTERABLE" },
   {        2512, "MQRC_SUBLEVEL_NOT_ALTERABLE" },
   {        2513, "MQRC_PROPERTY_NAME_LENGTH_ERR" },
   {        2514, "MQRC_DUPLICATE_GROUP_SUB" },
   {        2515, "MQRC_GROUPING_NOT_ALTERABLE" },
   {        2516, "MQRC_SELECTOR_INVALID_FOR_TYPE" },
   {        2517, "MQRC_HOBJ_QUIESCED" },
   {        2518, "MQRC_HOBJ_QUIESCED_NO_MSGS" },
   {        2519, "MQRC_SELECTION_STRING_ERROR" },
   {        2520, "MQRC_RES_OBJECT_STRING_ERROR" },
   {        2521, "MQRC_CONNECTION_SUSPENDED" },
   {        2522, "MQRC_INVALID_DESTINATION" },
   {        2523, "MQRC_INVALID_SUBSCRIPTION" },
   {        2524, "MQRC_SELECTOR_NOT_ALTERABLE" },
   {        2525, "MQRC_RETAINED_MSG_Q_ERROR" },
   {        2526, "MQRC_RETAINED_NOT_DELIVERED" },
   {        2527, "MQRC_RFH_RESTRICTED_FORMAT_ERR" },
   {        2528, "MQRC_CONNECTION_STOPPED" },
   {        2529, "MQRC_ASYNC_UOW_CONFLICT" },
   {        2530, "MQRC_ASYNC_XA_CONFLICT" },
   {        2531, "MQRC_PUBSUB_INHIBITED" },
   {        2532, "MQRC_MSG_HANDLE_COPY_FAILURE" },
   {        2533, "MQRC_DEST_CLASS_NOT_ALTERABLE" },
   {        2534, "MQRC_OPERATION_NOT_ALLOWED" },
   {        2535, "MQRC_ACTION_ERROR" },
   {        2537, "MQRC_CHANNEL_NOT_AVAILABLE" },
   {        2538, "MQRC_HOST_NOT_AVAILABLE" },
   {        2539, "MQRC_CHANNEL_CONFIG_ERROR" },
   {        2540, "MQRC_UNKNOWN_CHANNEL_NAME" },
   {        2541, "MQRC_LOOPING_PUBLICATION" },
   {        2542, "MQRC_ALREADY_JOINED" },
   {        2543, "MQRC_STANDBY_Q_MGR" },
   {        2544, "MQRC_RECONNECTING" },
   {        2545, "MQRC_RECONNECTED" },
   {        2546, "MQRC_RECONNECT_QMID_MISMATCH" },
   {        2547, "MQRC_RECONNECT_INCOMPATIBLE" },
   {        2548, "MQRC_RECONNECT_FAILED" },
   {        2549, "MQRC_CALL_INTERRUPTED" },
   {        2550, "MQRC_NO_SUBS_MATCHED" },
   {        2551, "MQRC_SELECTION_NOT_AVAILABLE" },
   {        2552, "MQRC_CHANNEL_SSL_WARNING" },
   {        2553, "MQRC_OCSP_URL_ERROR" },
   {        2554, "MQRC_CONTENT_ERROR" },
   {        2555, "MQRC_RECONNECT_Q_MGR_REQD" },
   {        2556, "MQRC_RECONNECT_TIMED_OUT" },
   {        2557, "MQRC_PUBLISH_EXIT_ERROR" },
   {        2558, "MQRC_COMMINFO_ERROR" },
   {        2559, "MQRC_DEF_SYNCPOINT_INHIBITED" },
   {        2560, "MQRC_MULTICAST_ONLY" },
   {        2561, "MQRC_DATA_SET_NOT_AVAILABLE" },
   {        2562, "MQRC_GROUPING_NOT_ALLOWED" },
   {        2563, "MQRC_GROUP_ADDRESS_ERROR" },
   {        2564, "MQRC_MULTICAST_CONFIG_ERROR" },
   {        2565, "MQRC_MULTICAST_INTERFACE_ERROR" },
   {        2566, "MQRC_MULTICAST_SEND_ERROR" },
   {        2567, "MQRC_MULTICAST_INTERNAL_ERROR" },
   {        2568, "MQRC_CONNECTION_NOT_AVAILABLE" },
   {        2569, "MQRC_SYNCPOINT_NOT_ALLOWED" },
   {        2570, "MQRC_SSL_ALT_PROVIDER_REQUIRED" },
   {        2571, "MQRC_MCAST_PUB_STATUS" },
   {        2572, "MQRC_MCAST_SUB_STATUS" },
   {        2573, "MQRC_PRECONN_EXIT_LOAD_ERROR" },
   {        2574, "MQRC_PRECONN_EXIT_NOT_FOUND" },
   {        2575, "MQRC_PRECONN_EXIT_ERROR" },
   {        2576, "MQRC_CD_ARRAY_ERROR" },
   {        2577, "MQRC_CHANNEL_BLOCKED" },
   {        2578, "MQRC_CHANNEL_BLOCKED_WARNING" },
   {        2579, "MQRC_SUBSCRIPTION_CREATE" },
   {        2580, "MQRC_SUBSCRIPTION_DELETE" },
   {        2581, "MQRC_SUBSCRIPTION_CHANGE" },
   {        2582, "MQRC_SUBSCRIPTION_REFRESH" },
   {        2583, "MQRC_INSTALLATION_MISMATCH" },
   {        2584, "MQRC_NOT_PRIVILEGED" },
   {        2586, "MQRC_PROPERTIES_DISABLED" },
   {        2587, "MQRC_HMSG_NOT_AVAILABLE" },
   {        2588, "MQRC_EXIT_PROPS_NOT_SUPPORTED" },
   {        2589, "MQRC_INSTALLATION_MISSING" },
   {        2590, "MQRC_FASTPATH_NOT_AVAILABLE" },
   {        2591, "MQRC_CIPHER_SPEC_NOT_SUITE_B" },
   {        2592, "MQRC_SUITE_B_ERROR" },
   {        2593, "MQRC_CERT_VAL_POLICY_ERROR" },
   {        2594, "MQRC_PASSWORD_PROTECTION_ERROR" },
   {        2595, "MQRC_CSP_ERROR" },
   {        2596, "MQRC_CERT_LABEL_NOT_ALLOWED" },
   {        2598, "MQRC_ADMIN_TOPIC_STRING_ERROR" },
   {        2599, "MQRC_AMQP_NOT_AVAILABLE" },
   {        2600, "MQRC_CCDT_URL_ERROR" },
   {        2701, "MQCAMO_CLOSE_DATE" },
   {        2701, "MQCAMO_FIRST" },
   {        2702, "MQCAMO_CLOSE_TIME" },
   {        2703, "MQCAMO_CONN_DATE" },
   {        2704, "MQCAMO_CONN_TIME" },
   {        2705, "MQCAMO_DISC_DATE" },
   {        2706, "MQCAMO_DISC_TIME" },
   {        2707, "MQCAMO_END_DATE" },
   {        2708, "MQCAMO_END_TIME" },
   {        2709, "MQCAMO_OPEN_DATE" },
   {        2710, "MQCAMO_OPEN_TIME" },
   {        2711, "MQCAMO_START_DATE" },
   {        2712, "MQCAMO_START_TIME" },
   {        2713, "MQCAMO_MONITOR_CLASS" },
   {        2714, "MQCAMO_MONITOR_TYPE" },
   {        2715, "MQCAMO_LAST_USED" },
   {        2715, "MQCAMO_MONITOR_DESC" },
   {        3001, "MQCACF_FIRST" },
   {        3001, "MQCACF_FROM_Q_NAME" },
   {        3001, "MQRCCF_CFH_TYPE_ERROR" },
   {        3002, "MQCACF_TO_Q_NAME" },
   {        3002, "MQRCCF_CFH_LENGTH_ERROR" },
   {        3003, "MQCACF_FROM_PROCESS_NAME" },
   {        3003, "MQRCCF_CFH_VERSION_ERROR" },
   {        3004, "MQCACF_TO_PROCESS_NAME" },
   {        3004, "MQRCCF_CFH_MSG_SEQ_NUMBER_ERR" },
   {        3005, "MQCACF_FROM_NAMELIST_NAME" },
   {        3005, "MQRCCF_CFH_CONTROL_ERROR" },
   {        3006, "MQCACF_TO_NAMELIST_NAME" },
   {        3006, "MQRCCF_CFH_PARM_COUNT_ERROR" },
   {        3007, "MQCACF_FROM_CHANNEL_NAME" },
   {        3007, "MQRCCF_CFH_COMMAND_ERROR" },
   {        3008, "MQCACF_TO_CHANNEL_NAME" },
   {        3008, "MQRCCF_COMMAND_FAILED" },
   {        3009, "MQCACF_FROM_AUTH_INFO_NAME" },
   {        3009, "MQRCCF_CFIN_LENGTH_ERROR" },
   {        3010, "MQCACF_TO_AUTH_INFO_NAME" },
   {        3010, "MQRCCF_CFST_LENGTH_ERROR" },
   {        3011, "MQCACF_Q_NAMES" },
   {        3011, "MQRCCF_CFST_STRING_LENGTH_ERR" },
   {        3012, "MQCACF_PROCESS_NAMES" },
   {        3012, "MQRCCF_FORCE_VALUE_ERROR" },
   {        3013, "MQCACF_NAMELIST_NAMES" },
   {        3013, "MQRCCF_STRUCTURE_TYPE_ERROR" },
   {        3014, "MQCACF_ESCAPE_TEXT" },
   {        3014, "MQRCCF_CFIN_PARM_ID_ERROR" },
   {        3015, "MQCACF_LOCAL_Q_NAMES" },
   {        3015, "MQRCCF_CFST_PARM_ID_ERROR" },
   {        3016, "MQCACF_MODEL_Q_NAMES" },
   {        3016, "MQRCCF_MSG_LENGTH_ERROR" },
   {        3017, "MQCACF_ALIAS_Q_NAMES" },
   {        3017, "MQRCCF_CFIN_DUPLICATE_PARM" },
   {        3018, "MQCACF_REMOTE_Q_NAMES" },
   {        3018, "MQRCCF_CFST_DUPLICATE_PARM" },
   {        3019, "MQCACF_SENDER_CHANNEL_NAMES" },
   {        3019, "MQRCCF_PARM_COUNT_TOO_SMALL" },
   {        3020, "MQCACF_SERVER_CHANNEL_NAMES" },
   {        3020, "MQRCCF_PARM_COUNT_TOO_BIG" },
   {        3021, "MQCACF_REQUESTER_CHANNEL_NAMES" },
   {        3021, "MQRCCF_Q_ALREADY_IN_CELL" },
   {        3022, "MQCACF_RECEIVER_CHANNEL_NAMES" },
   {        3022, "MQRCCF_Q_TYPE_ERROR" },
   {        3023, "MQCACF_OBJECT_Q_MGR_NAME" },
   {        3023, "MQRCCF_MD_FORMAT_ERROR" },
   {        3024, "MQCACF_APPL_NAME" },
   {        3024, "MQRCCF_CFSL_LENGTH_ERROR" },
   {        3025, "MQCACF_USER_IDENTIFIER" },
   {        3025, "MQRCCF_REPLACE_VALUE_ERROR" },
   {        3026, "MQCACF_AUX_ERROR_DATA_STR_1" },
   {        3026, "MQRCCF_CFIL_DUPLICATE_VALUE" },
   {        3027, "MQCACF_AUX_ERROR_DATA_STR_2" },
   {        3027, "MQRCCF_CFIL_COUNT_ERROR" },
   {        3028, "MQCACF_AUX_ERROR_DATA_STR_3" },
   {        3028, "MQRCCF_CFIL_LENGTH_ERROR" },
   {        3029, "MQCACF_BRIDGE_NAME" },
   {        3029, "MQRCCF_MODE_VALUE_ERROR" },
   {        3029, "MQRCCF_QUIESCE_VALUE_ERROR" },
   {        3030, "MQCACF_STREAM_NAME" },
   {        3030, "MQRCCF_MSG_SEQ_NUMBER_ERROR" },
   {        3031, "MQCACF_TOPIC" },
   {        3031, "MQRCCF_PING_DATA_COUNT_ERROR" },
   {        3032, "MQCACF_PARENT_Q_MGR_NAME" },
   {        3032, "MQRCCF_PING_DATA_COMPARE_ERROR" },
   {        3033, "MQCACF_CORREL_ID" },
   {        3033, "MQRCCF_CFSL_PARM_ID_ERROR" },
   {        3034, "MQCACF_PUBLISH_TIMESTAMP" },
   {        3034, "MQRCCF_CHANNEL_TYPE_ERROR" },
   {        3035, "MQCACF_STRING_DATA" },
   {        3035, "MQRCCF_PARM_SEQUENCE_ERROR" },
   {        3036, "MQCACF_SUPPORTED_STREAM_NAME" },
   {        3036, "MQRCCF_XMIT_PROTOCOL_TYPE_ERR" },
   {        3037, "MQCACF_REG_TOPIC" },
   {        3037, "MQRCCF_BATCH_SIZE_ERROR" },
   {        3038, "MQCACF_REG_TIME" },
   {        3038, "MQRCCF_DISC_INT_ERROR" },
   {        3039, "MQCACF_REG_USER_ID" },
   {        3039, "MQRCCF_SHORT_RETRY_ERROR" },
   {        3040, "MQCACF_CHILD_Q_MGR_NAME" },
   {        3040, "MQRCCF_SHORT_TIMER_ERROR" },
   {        3041, "MQCACF_REG_STREAM_NAME" },
   {        3041, "MQRCCF_LONG_RETRY_ERROR" },
   {        3042, "MQCACF_REG_Q_MGR_NAME" },
   {        3042, "MQRCCF_LONG_TIMER_ERROR" },
   {        3043, "MQCACF_REG_Q_NAME" },
   {        3043, "MQRCCF_SEQ_NUMBER_WRAP_ERROR" },
   {        3044, "MQCACF_REG_CORREL_ID" },
   {        3044, "MQRCCF_MAX_MSG_LENGTH_ERROR" },
   {        3045, "MQCACF_EVENT_USER_ID" },
   {        3045, "MQRCCF_PUT_AUTH_ERROR" },
   {        3046, "MQCACF_OBJECT_NAME" },
   {        3046, "MQRCCF_PURGE_VALUE_ERROR" },
   {        3047, "MQCACF_EVENT_Q_MGR" },
   {        3047, "MQRCCF_CFIL_PARM_ID_ERROR" },
   {        3048, "MQCACF_AUTH_INFO_NAMES" },
   {        3048, "MQRCCF_MSG_TRUNCATED" },
   {        3049, "MQCACF_EVENT_APPL_IDENTITY" },
   {        3049, "MQRCCF_CCSID_ERROR" },
   {        3050, "MQCACF_EVENT_APPL_NAME" },
   {        3050, "MQRCCF_ENCODING_ERROR" },
   {        3051, "MQCACF_EVENT_APPL_ORIGIN" },
   {        3051, "MQRCCF_QUEUES_VALUE_ERROR" },
   {        3052, "MQCACF_SUBSCRIPTION_NAME" },
   {        3052, "MQRCCF_DATA_CONV_VALUE_ERROR" },
   {        3053, "MQCACF_REG_SUB_NAME" },
   {        3053, "MQRCCF_INDOUBT_VALUE_ERROR" },
   {        3054, "MQCACF_SUBSCRIPTION_IDENTITY" },
   {        3054, "MQRCCF_ESCAPE_TYPE_ERROR" },
   {        3055, "MQCACF_REG_SUB_IDENTITY" },
   {        3055, "MQRCCF_REPOS_VALUE_ERROR" },
   {        3056, "MQCACF_SUBSCRIPTION_USER_DATA" },
   {        3057, "MQCACF_REG_SUB_USER_DATA" },
   {        3058, "MQCACF_APPL_TAG" },
   {        3059, "MQCACF_DATA_SET_NAME" },
   {        3060, "MQCACF_UOW_START_DATE" },
   {        3061, "MQCACF_UOW_START_TIME" },
   {        3062, "MQCACF_UOW_LOG_START_DATE" },
   {        3062, "MQRCCF_CHANNEL_TABLE_ERROR" },
   {        3063, "MQCACF_UOW_LOG_START_TIME" },
   {        3063, "MQRCCF_MCA_TYPE_ERROR" },
   {        3064, "MQCACF_UOW_LOG_EXTENT_NAME" },
   {        3064, "MQRCCF_CHL_INST_TYPE_ERROR" },
   {        3065, "MQCACF_PRINCIPAL_ENTITY_NAMES" },
   {        3065, "MQRCCF_CHL_STATUS_NOT_FOUND" },
   {        3066, "MQCACF_GROUP_ENTITY_NAMES" },
   {        3066, "MQRCCF_CFSL_DUPLICATE_PARM" },
   {        3067, "MQCACF_AUTH_PROFILE_NAME" },
   {        3067, "MQRCCF_CFSL_TOTAL_LENGTH_ERROR" },
   {        3068, "MQCACF_ENTITY_NAME" },
   {        3068, "MQRCCF_CFSL_COUNT_ERROR" },
   {        3069, "MQCACF_SERVICE_COMPONENT" },
   {        3069, "MQRCCF_CFSL_STRING_LENGTH_ERR" },
   {        3070, "MQCACF_RESPONSE_Q_MGR_NAME" },
   {        3070, "MQRCCF_BROKER_DELETED" },
   {        3071, "MQCACF_CURRENT_LOG_EXTENT_NAME" },
   {        3071, "MQRCCF_STREAM_ERROR" },
   {        3072, "MQCACF_RESTART_LOG_EXTENT_NAME" },
   {        3072, "MQRCCF_TOPIC_ERROR" },
   {        3073, "MQCACF_MEDIA_LOG_EXTENT_NAME" },
   {        3073, "MQRCCF_NOT_REGISTERED" },
   {        3074, "MQCACF_LOG_PATH" },
   {        3074, "MQRCCF_Q_MGR_NAME_ERROR" },
   {        3075, "MQCACF_COMMAND_MQSC" },
   {        3075, "MQRCCF_INCORRECT_STREAM" },
   {        3076, "MQCACF_Q_MGR_CPF" },
   {        3076, "MQRCCF_Q_NAME_ERROR" },
   {        3077, "MQRCCF_NO_RETAINED_MSG" },
   {        3078, "MQCACF_USAGE_LOG_RBA" },
   {        3078, "MQRCCF_DUPLICATE_IDENTITY" },
   {        3079, "MQCACF_USAGE_LOG_LRSN" },
   {        3079, "MQRCCF_INCORRECT_Q" },
   {        3080, "MQCACF_COMMAND_SCOPE" },
   {        3080, "MQRCCF_CORREL_ID_ERROR" },
   {        3081, "MQCACF_ASID" },
   {        3081, "MQRCCF_NOT_AUTHORIZED" },
   {        3082, "MQCACF_PSB_NAME" },
   {        3082, "MQRCCF_UNKNOWN_STREAM" },
   {        3083, "MQCACF_PST_ID" },
   {        3083, "MQRCCF_REG_OPTIONS_ERROR" },
   {        3084, "MQCACF_TASK_NUMBER" },
   {        3084, "MQRCCF_PUB_OPTIONS_ERROR" },
   {        3085, "MQCACF_TRANSACTION_ID" },
   {        3085, "MQRCCF_UNKNOWN_BROKER" },
   {        3086, "MQCACF_Q_MGR_UOW_ID" },
   {        3086, "MQRCCF_Q_MGR_CCSID_ERROR" },
   {        3087, "MQRCCF_DEL_OPTIONS_ERROR" },
   {        3088, "MQCACF_ORIGIN_NAME" },
   {        3088, "MQRCCF_CLUSTER_NAME_CONFLICT" },
   {        3089, "MQCACF_ENV_INFO" },
   {        3089, "MQRCCF_REPOS_NAME_CONFLICT" },
   {        3090, "MQCACF_SECURITY_PROFILE" },
   {        3090, "MQRCCF_CLUSTER_Q_USAGE_ERROR" },
   {        3091, "MQCACF_CONFIGURATION_DATE" },
   {        3091, "MQRCCF_ACTION_VALUE_ERROR" },
   {        3092, "MQCACF_CONFIGURATION_TIME" },
   {        3092, "MQRCCF_COMMS_LIBRARY_ERROR" },
   {        3093, "MQCACF_FROM_CF_STRUC_NAME" },
   {        3093, "MQRCCF_NETBIOS_NAME_ERROR" },
   {        3094, "MQCACF_TO_CF_STRUC_NAME" },
   {        3094, "MQRCCF_BROKER_COMMAND_FAILED" },
   {        3095, "MQCACF_CF_STRUC_NAMES" },
   {        3095, "MQRCCF_CFST_CONFLICTING_PARM" },
   {        3096, "MQCACF_FAIL_DATE" },
   {        3096, "MQRCCF_PATH_NOT_VALID" },
   {        3097, "MQCACF_FAIL_TIME" },
   {        3097, "MQRCCF_PARM_SYNTAX_ERROR" },
   {        3098, "MQCACF_BACKUP_DATE" },
   {        3098, "MQRCCF_PWD_LENGTH_ERROR" },
   {        3099, "MQCACF_BACKUP_TIME" },
   {        3100, "MQCACF_SYSTEM_NAME" },
   {        3101, "MQCACF_CF_STRUC_BACKUP_START" },
   {        3102, "MQCACF_CF_STRUC_BACKUP_END" },
   {        3103, "MQCACF_CF_STRUC_LOG_Q_MGRS" },
   {        3104, "MQCACF_FROM_STORAGE_CLASS" },
   {        3105, "MQCACF_TO_STORAGE_CLASS" },
   {        3106, "MQCACF_STORAGE_CLASS_NAMES" },
   {        3108, "MQCACF_DSG_NAME" },
   {        3109, "MQCACF_DB2_NAME" },
   {        3110, "MQCACF_SYSP_CMD_USER_ID" },
   {        3111, "MQCACF_SYSP_OTMA_GROUP" },
   {        3112, "MQCACF_SYSP_OTMA_MEMBER" },
   {        3113, "MQCACF_SYSP_OTMA_DRU_EXIT" },
   {        3114, "MQCACF_SYSP_OTMA_TPIPE_PFX" },
   {        3115, "MQCACF_SYSP_ARCHIVE_PFX1" },
   {        3116, "MQCACF_SYSP_ARCHIVE_UNIT1" },
   {        3117, "MQCACF_SYSP_LOG_CORREL_ID" },
   {        3118, "MQCACF_SYSP_UNIT_VOLSER" },
   {        3119, "MQCACF_SYSP_Q_MGR_TIME" },
   {        3120, "MQCACF_SYSP_Q_MGR_DATE" },
   {        3121, "MQCACF_SYSP_Q_MGR_RBA" },
   {        3122, "MQCACF_SYSP_LOG_RBA" },
   {        3123, "MQCACF_SYSP_SERVICE" },
   {        3124, "MQCACF_FROM_LISTENER_NAME" },
   {        3125, "MQCACF_TO_LISTENER_NAME" },
   {        3126, "MQCACF_FROM_SERVICE_NAME" },
   {        3127, "MQCACF_TO_SERVICE_NAME" },
   {        3128, "MQCACF_LAST_PUT_DATE" },
   {        3129, "MQCACF_LAST_PUT_TIME" },
   {        3130, "MQCACF_LAST_GET_DATE" },
   {        3131, "MQCACF_LAST_GET_TIME" },
   {        3132, "MQCACF_OPERATION_DATE" },
   {        3133, "MQCACF_OPERATION_TIME" },
   {        3134, "MQCACF_ACTIVITY_DESC" },
   {        3135, "MQCACF_APPL_IDENTITY_DATA" },
   {        3136, "MQCACF_APPL_ORIGIN_DATA" },
   {        3137, "MQCACF_PUT_DATE" },
   {        3138, "MQCACF_PUT_TIME" },
   {        3139, "MQCACF_REPLY_TO_Q" },
   {        3140, "MQCACF_REPLY_TO_Q_MGR" },
   {        3141, "MQCACF_RESOLVED_Q_NAME" },
   {        3142, "MQCACF_STRUC_ID" },
   {        3143, "MQCACF_VALUE_NAME" },
   {        3144, "MQCACF_SERVICE_START_DATE" },
   {        3145, "MQCACF_SERVICE_START_TIME" },
   {        3146, "MQCACF_SYSP_OFFLINE_RBA" },
   {        3147, "MQCACF_SYSP_ARCHIVE_PFX2" },
   {        3148, "MQCACF_SYSP_ARCHIVE_UNIT2" },
   {        3149, "MQCACF_TO_TOPIC_NAME" },
   {        3150, "MQCACF_FROM_TOPIC_NAME" },
   {        3150, "MQRCCF_FILTER_ERROR" },
   {        3151, "MQCACF_TOPIC_NAMES" },
   {        3151, "MQRCCF_WRONG_USER" },
   {        3152, "MQCACF_SUB_NAME" },
   {        3152, "MQRCCF_DUPLICATE_SUBSCRIPTION" },
   {        3153, "MQCACF_DESTINATION_Q_MGR" },
   {        3153, "MQRCCF_SUB_NAME_ERROR" },
   {        3154, "MQCACF_DESTINATION" },
   {        3154, "MQRCCF_SUB_IDENTITY_ERROR" },
   {        3155, "MQRCCF_SUBSCRIPTION_IN_USE" },
   {        3156, "MQCACF_SUB_USER_ID" },
   {        3156, "MQRCCF_SUBSCRIPTION_LOCKED" },
   {        3157, "MQRCCF_ALREADY_JOINED" },
   {        3159, "MQCACF_SUB_USER_DATA" },
   {        3160, "MQCACF_SUB_SELECTOR" },
   {        3160, "MQRCCF_OBJECT_IN_USE" },
   {        3161, "MQCACF_LAST_PUB_DATE" },
   {        3161, "MQRCCF_UNKNOWN_FILE_NAME" },
   {        3162, "MQCACF_LAST_PUB_TIME" },
   {        3162, "MQRCCF_FILE_NOT_AVAILABLE" },
   {        3163, "MQCACF_FROM_SUB_NAME" },
   {        3163, "MQRCCF_DISC_RETRY_ERROR" },
   {        3164, "MQCACF_TO_SUB_NAME" },
   {        3164, "MQRCCF_ALLOC_RETRY_ERROR" },
   {        3165, "MQRCCF_ALLOC_SLOW_TIMER_ERROR" },
   {        3166, "MQRCCF_ALLOC_FAST_TIMER_ERROR" },
   {        3167, "MQCACF_LAST_MSG_TIME" },
   {        3167, "MQRCCF_PORT_NUMBER_ERROR" },
   {        3168, "MQCACF_LAST_MSG_DATE" },
   {        3168, "MQRCCF_CHL_SYSTEM_NOT_ACTIVE" },
   {        3169, "MQCACF_SUBSCRIPTION_POINT" },
   {        3169, "MQRCCF_ENTITY_NAME_MISSING" },
   {        3170, "MQCACF_FILTER" },
   {        3170, "MQRCCF_PROFILE_NAME_ERROR" },
   {        3171, "MQCACF_NONE" },
   {        3171, "MQRCCF_AUTH_VALUE_ERROR" },
   {        3172, "MQCACF_ADMIN_TOPIC_NAMES" },
   {        3172, "MQRCCF_AUTH_VALUE_MISSING" },
   {        3173, "MQCACF_ROUTING_FINGER_PRINT" },
   {        3173, "MQRCCF_OBJECT_TYPE_MISSING" },
   {        3174, "MQCACF_APPL_DESC" },
   {        3174, "MQRCCF_CONNECTION_ID_ERROR" },
   {        3175, "MQCACF_Q_MGR_START_DATE" },
   {        3175, "MQRCCF_LOG_TYPE_ERROR" },
   {        3176, "MQCACF_Q_MGR_START_TIME" },
   {        3176, "MQRCCF_PROGRAM_NOT_AVAILABLE" },
   {        3177, "MQCACF_FROM_COMM_INFO_NAME" },
   {        3177, "MQRCCF_PROGRAM_AUTH_FAILED" },
   {        3178, "MQCACF_TO_COMM_INFO_NAME" },
   {        3179, "MQCACF_CF_OFFLOAD_SIZE1" },
   {        3180, "MQCACF_CF_OFFLOAD_SIZE2" },
   {        3181, "MQCACF_CF_OFFLOAD_SIZE3" },
   {        3182, "MQCACF_CF_SMDS_GENERIC_NAME" },
   {        3183, "MQCACF_CF_SMDS" },
   {        3184, "MQCACF_RECOVERY_DATE" },
   {        3185, "MQCACF_RECOVERY_TIME" },
   {        3186, "MQCACF_CF_SMDSCONN" },
   {        3187, "MQCACF_CF_STRUC_NAME" },
   {        3188, "MQCACF_ALTERNATE_USERID" },
   {        3189, "MQCACF_CHAR_ATTRS" },
   {        3190, "MQCACF_DYNAMIC_Q_NAME" },
   {        3191, "MQCACF_HOST_NAME" },
   {        3192, "MQCACF_MQCB_NAME" },
   {        3193, "MQCACF_OBJECT_STRING" },
   {        3194, "MQCACF_RESOLVED_LOCAL_Q_MGR" },
   {        3195, "MQCACF_RESOLVED_LOCAL_Q_NAME" },
   {        3196, "MQCACF_RESOLVED_OBJECT_STRING" },
   {        3197, "MQCACF_RESOLVED_Q_MGR" },
   {        3198, "MQCACF_SELECTION_STRING" },
   {        3199, "MQCACF_XA_INFO" },
   {        3200, "MQCACF_APPL_FUNCTION" },
   {        3200, "MQRCCF_NONE_FOUND" },
   {        3201, "MQCACF_XQH_REMOTE_Q_NAME" },
   {        3201, "MQRCCF_SECURITY_SWITCH_OFF" },
   {        3202, "MQCACF_XQH_REMOTE_Q_MGR" },
   {        3202, "MQRCCF_SECURITY_REFRESH_FAILED" },
   {        3203, "MQCACF_XQH_PUT_TIME" },
   {        3203, "MQRCCF_PARM_CONFLICT" },
   {        3204, "MQCACF_XQH_PUT_DATE" },
   {        3204, "MQRCCF_COMMAND_INHIBITED" },
   {        3205, "MQCACF_EXCL_OPERATOR_MESSAGES" },
   {        3205, "MQRCCF_OBJECT_BEING_DELETED" },
   {        3206, "MQCACF_CSP_USER_IDENTIFIER" },
   {        3207, "MQCACF_AMQP_CLIENT_ID" },
   {        3207, "MQRCCF_STORAGE_CLASS_IN_USE" },
   {        3208, "MQCACF_ARCHIVE_LOG_EXTENT_NAME" },
   {        3208, "MQCACF_LAST_USED" },
   {        3208, "MQRCCF_OBJECT_NAME_RESTRICTED" },
   {        3209, "MQRCCF_OBJECT_LIMIT_EXCEEDED" },
   {        3210, "MQRCCF_OBJECT_OPEN_FORCE" },
   {        3211, "MQRCCF_DISPOSITION_CONFLICT" },
   {        3212, "MQRCCF_Q_MGR_NOT_IN_QSG" },
   {        3213, "MQRCCF_ATTR_VALUE_FIXED" },
   {        3215, "MQRCCF_NAMELIST_ERROR" },
   {        3217, "MQRCCF_NO_CHANNEL_INITIATOR" },
   {        3218, "MQRCCF_CHANNEL_INITIATOR_ERROR" },
   {        3222, "MQRCCF_COMMAND_LEVEL_CONFLICT" },
   {        3223, "MQRCCF_Q_ATTR_CONFLICT" },
   {        3224, "MQRCCF_EVENTS_DISABLED" },
   {        3225, "MQRCCF_COMMAND_SCOPE_ERROR" },
   {        3226, "MQRCCF_COMMAND_REPLY_ERROR" },
   {        3227, "MQRCCF_FUNCTION_RESTRICTED" },
   {        3228, "MQRCCF_PARM_MISSING" },
   {        3229, "MQRCCF_PARM_VALUE_ERROR" },
   {        3230, "MQRCCF_COMMAND_LENGTH_ERROR" },
   {        3231, "MQRCCF_COMMAND_ORIGIN_ERROR" },
   {        3232, "MQRCCF_LISTENER_CONFLICT" },
   {        3233, "MQRCCF_LISTENER_STARTED" },
   {        3234, "MQRCCF_LISTENER_STOPPED" },
   {        3235, "MQRCCF_CHANNEL_ERROR" },
   {        3236, "MQRCCF_CF_STRUC_ERROR" },
   {        3237, "MQRCCF_UNKNOWN_USER_ID" },
   {        3238, "MQRCCF_UNEXPECTED_ERROR" },
   {        3239, "MQRCCF_NO_XCF_PARTNER" },
   {        3240, "MQRCCF_CFGR_PARM_ID_ERROR" },
   {        3241, "MQRCCF_CFIF_LENGTH_ERROR" },
   {        3242, "MQRCCF_CFIF_OPERATOR_ERROR" },
   {        3243, "MQRCCF_CFIF_PARM_ID_ERROR" },
   {        3244, "MQRCCF_CFSF_FILTER_VAL_LEN_ERR" },
   {        3245, "MQRCCF_CFSF_LENGTH_ERROR" },
   {        3246, "MQRCCF_CFSF_OPERATOR_ERROR" },
   {        3247, "MQRCCF_CFSF_PARM_ID_ERROR" },
   {        3248, "MQRCCF_TOO_MANY_FILTERS" },
   {        3249, "MQRCCF_LISTENER_RUNNING" },
   {        3250, "MQRCCF_LSTR_STATUS_NOT_FOUND" },
   {        3251, "MQRCCF_SERVICE_RUNNING" },
   {        3252, "MQRCCF_SERV_STATUS_NOT_FOUND" },
   {        3253, "MQRCCF_SERVICE_STOPPED" },
   {        3254, "MQRCCF_CFBS_DUPLICATE_PARM" },
   {        3255, "MQRCCF_CFBS_LENGTH_ERROR" },
   {        3256, "MQRCCF_CFBS_PARM_ID_ERROR" },
   {        3257, "MQRCCF_CFBS_STRING_LENGTH_ERR" },
   {        3258, "MQRCCF_CFGR_LENGTH_ERROR" },
   {        3259, "MQRCCF_CFGR_PARM_COUNT_ERROR" },
   {        3260, "MQRCCF_CONN_NOT_STOPPED" },
   {        3261, "MQRCCF_SERVICE_REQUEST_PENDING" },
   {        3262, "MQRCCF_NO_START_CMD" },
   {        3263, "MQRCCF_NO_STOP_CMD" },
   {        3264, "MQRCCF_CFBF_LENGTH_ERROR" },
   {        3265, "MQRCCF_CFBF_PARM_ID_ERROR" },
   {        3266, "MQRCCF_CFBF_OPERATOR_ERROR" },
   {        3267, "MQRCCF_CFBF_FILTER_VAL_LEN_ERR" },
   {        3268, "MQRCCF_LISTENER_STILL_ACTIVE" },
   {        3269, "MQRCCF_DEF_XMIT_Q_CLUS_ERROR" },
   {        3300, "MQRCCF_TOPICSTR_ALREADY_EXISTS" },
   {        3301, "MQRCCF_SHARING_CONVS_ERROR" },
   {        3302, "MQRCCF_SHARING_CONVS_TYPE" },
   {        3303, "MQRCCF_SECURITY_CASE_CONFLICT" },
   {        3305, "MQRCCF_TOPIC_TYPE_ERROR" },
   {        3306, "MQRCCF_MAX_INSTANCES_ERROR" },
   {        3307, "MQRCCF_MAX_INSTS_PER_CLNT_ERR" },
   {        3308, "MQRCCF_TOPIC_STRING_NOT_FOUND" },
   {        3309, "MQRCCF_SUBSCRIPTION_POINT_ERR" },
   {        3311, "MQRCCF_SUB_ALREADY_EXISTS" },
   {        3312, "MQRCCF_UNKNOWN_OBJECT_NAME" },
   {        3313, "MQRCCF_REMOTE_Q_NAME_ERROR" },
   {        3314, "MQRCCF_DURABILITY_NOT_ALLOWED" },
   {        3315, "MQRCCF_HOBJ_ERROR" },
   {        3316, "MQRCCF_DEST_NAME_ERROR" },
   {        3317, "MQRCCF_INVALID_DESTINATION" },
   {        3318, "MQRCCF_PUBSUB_INHIBITED" },
   {        3319, "MQRCCF_GROUPUR_CHECKS_FAILED" },
   {        3320, "MQRCCF_COMM_INFO_TYPE_ERROR" },
   {        3321, "MQRCCF_USE_CLIENT_ID_ERROR" },
   {        3322, "MQRCCF_CLIENT_ID_NOT_FOUND" },
   {        3323, "MQRCCF_CLIENT_ID_ERROR" },
   {        3324, "MQRCCF_PORT_IN_USE" },
   {        3325, "MQRCCF_SSL_ALT_PROVIDER_REQD" },
   {        3326, "MQRCCF_CHLAUTH_TYPE_ERROR" },
   {        3327, "MQRCCF_CHLAUTH_ACTION_ERROR" },
   {        3328, "MQRCCF_POLICY_NOT_FOUND" },
   {        3329, "MQRCCF_ENCRYPTION_ALG_ERROR" },
   {        3330, "MQRCCF_SIGNATURE_ALG_ERROR" },
   {        3331, "MQRCCF_TOLERATION_POL_ERROR" },
   {        3332, "MQRCCF_POLICY_VERSION_ERROR" },
   {        3333, "MQRCCF_RECIPIENT_DN_MISSING" },
   {        3334, "MQRCCF_POLICY_NAME_MISSING" },
   {        3335, "MQRCCF_CHLAUTH_USERSRC_ERROR" },
   {        3336, "MQRCCF_WRONG_CHLAUTH_TYPE" },
   {        3337, "MQRCCF_CHLAUTH_ALREADY_EXISTS" },
   {        3338, "MQRCCF_CHLAUTH_NOT_FOUND" },
   {        3339, "MQRCCF_WRONG_CHLAUTH_ACTION" },
   {        3340, "MQRCCF_WRONG_CHLAUTH_USERSRC" },
   {        3341, "MQRCCF_CHLAUTH_WARN_ERROR" },
   {        3342, "MQRCCF_WRONG_CHLAUTH_MATCH" },
   {        3343, "MQRCCF_IPADDR_RANGE_CONFLICT" },
   {        3344, "MQRCCF_CHLAUTH_MAX_EXCEEDED" },
   {        3345, "MQRCCF_ADDRESS_ERROR" },
   {        3345, "MQRCCF_IPADDR_ERROR" },
   {        3346, "MQRCCF_IPADDR_RANGE_ERROR" },
   {        3347, "MQRCCF_PROFILE_NAME_MISSING" },
   {        3348, "MQRCCF_CHLAUTH_CLNTUSER_ERROR" },
   {        3349, "MQRCCF_CHLAUTH_NAME_ERROR" },
   {        3350, "MQRCCF_CHLAUTH_RUNCHECK_ERROR" },
   {        3351, "MQRCCF_CF_STRUC_ALREADY_FAILED" },
   {        3352, "MQRCCF_CFCONLOS_CHECKS_FAILED" },
   {        3353, "MQRCCF_SUITE_B_ERROR" },
   {        3354, "MQRCCF_CHANNEL_NOT_STARTED" },
   {        3355, "MQRCCF_CUSTOM_ERROR" },
   {        3356, "MQRCCF_BACKLOG_OUT_OF_RANGE" },
   {        3357, "MQRCCF_CHLAUTH_DISABLED" },
   {        3358, "MQRCCF_SMDS_REQUIRES_DSGROUP" },
   {        3359, "MQRCCF_PSCLUS_DISABLED_TOPDEF" },
   {        3360, "MQRCCF_PSCLUS_TOPIC_EXISTS" },
   {        3361, "MQRCCF_SSL_CIPHER_SUITE_ERROR" },
   {        3362, "MQRCCF_SOCKET_ERROR" },
   {        3363, "MQRCCF_CLUS_XMIT_Q_USAGE_ERROR" },
   {        3364, "MQRCCF_CERT_VAL_POLICY_ERROR" },
   {        3365, "MQRCCF_INVALID_PROTOCOL" },
   {        3366, "MQRCCF_REVDNS_DISABLED" },
   {        3367, "MQRCCF_CLROUTE_NOT_ALTERABLE" },
   {        3368, "MQRCCF_CLUSTER_TOPIC_CONFLICT" },
   {        3369, "MQRCCF_DEFCLXQ_MODEL_Q_ERROR" },
   {        3370, "MQRCCF_CHLAUTH_CHKCLI_ERROR" },
   {        3371, "MQRCCF_CERT_LABEL_NOT_ALLOWED" },
   {        3372, "MQRCCF_Q_MGR_ATTR_CONFLICT" },
   {        3373, "MQRCCF_ENTITY_TYPE_MISSING" },
   {        3374, "MQRCCF_CLWL_EXIT_NAME_ERROR" },
   {        3375, "MQRCCF_SERVICE_NAME_ERROR" },
   {        3376, "MQRCCF_REMOTE_CHL_TYPE_ERROR" },
   {        3377, "MQRCCF_TOPIC_RESTRICTED" },
   {        3378, "MQRCCF_CURRENT_LOG_EXTENT" },
   {        3379, "MQRCCF_LOG_EXTENT_NOT_FOUND" },
   {        3380, "MQRCCF_LOG_NOT_REDUCED" },
   {        3381, "MQRCCF_LOG_EXTENT_ERROR" },
   {        3382, "MQRCCF_ACCESS_BLOCKED" },
   {        3501, "MQCACH_CHANNEL_NAME" },
   {        3501, "MQCACH_FIRST" },
   {        3502, "MQCACH_DESC" },
   {        3503, "MQCACH_MODE_NAME" },
   {        3504, "MQCACH_TP_NAME" },
   {        3505, "MQCACH_XMIT_Q_NAME" },
   {        3506, "MQCACH_CONNECTION_NAME" },
   {        3507, "MQCACH_MCA_NAME" },
   {        3508, "MQCACH_SEC_EXIT_NAME" },
   {        3509, "MQCACH_MSG_EXIT_NAME" },
   {        3510, "MQCACH_SEND_EXIT_NAME" },
   {        3511, "MQCACH_RCV_EXIT_NAME" },
   {        3512, "MQCACH_CHANNEL_NAMES" },
   {        3513, "MQCACH_SEC_EXIT_USER_DATA" },
   {        3514, "MQCACH_MSG_EXIT_USER_DATA" },
   {        3515, "MQCACH_SEND_EXIT_USER_DATA" },
   {        3516, "MQCACH_RCV_EXIT_USER_DATA" },
   {        3517, "MQCACH_USER_ID" },
   {        3518, "MQCACH_PASSWORD" },
   {        3520, "MQCACH_LOCAL_ADDRESS" },
   {        3521, "MQCACH_LOCAL_NAME" },
   {        3524, "MQCACH_LAST_MSG_TIME" },
   {        3525, "MQCACH_LAST_MSG_DATE" },
   {        3527, "MQCACH_MCA_USER_ID" },
   {        3528, "MQCACH_CHANNEL_START_TIME" },
   {        3529, "MQCACH_CHANNEL_START_DATE" },
   {        3530, "MQCACH_MCA_JOB_NAME" },
   {        3531, "MQCACH_LAST_LUWID" },
   {        3532, "MQCACH_CURRENT_LUWID" },
   {        3533, "MQCACH_FORMAT_NAME" },
   {        3534, "MQCACH_MR_EXIT_NAME" },
   {        3535, "MQCACH_MR_EXIT_USER_DATA" },
   {        3544, "MQCACH_SSL_CIPHER_SPEC" },
   {        3545, "MQCACH_SSL_PEER_NAME" },
   {        3546, "MQCACH_SSL_HANDSHAKE_STAGE" },
   {        3547, "MQCACH_SSL_SHORT_PEER_NAME" },
   {        3548, "MQCACH_REMOTE_APPL_TAG" },
   {        3549, "MQCACH_SSL_CERT_USER_ID" },
   {        3550, "MQCACH_SSL_CERT_ISSUER_NAME" },
   {        3551, "MQCACH_LU_NAME" },
   {        3552, "MQCACH_IP_ADDRESS" },
   {        3553, "MQCACH_TCP_NAME" },
   {        3554, "MQCACH_LISTENER_NAME" },
   {        3555, "MQCACH_LISTENER_DESC" },
   {        3556, "MQCACH_LISTENER_START_DATE" },
   {        3557, "MQCACH_LISTENER_START_TIME" },
   {        3558, "MQCACH_SSL_KEY_RESET_DATE" },
   {        3559, "MQCACH_SSL_KEY_RESET_TIME" },
   {        3560, "MQCACH_REMOTE_VERSION" },
   {        3561, "MQCACH_REMOTE_PRODUCT" },
   {        3562, "MQCACH_GROUP_ADDRESS" },
   {        3563, "MQCACH_JAAS_CONFIG" },
   {        3564, "MQCACH_CLIENT_ID" },
   {        3565, "MQCACH_SSL_KEY_PASSPHRASE" },
   {        3566, "MQCACH_CONNECTION_NAME_LIST" },
   {        3567, "MQCACH_CLIENT_USER_ID" },
   {        3568, "MQCACH_MCA_USER_ID_LIST" },
   {        3569, "MQCACH_SSL_CIPHER_SUITE" },
   {        3570, "MQCACH_WEBCONTENT_PATH" },
   {        3571, "MQCACH_LAST_USED" },
   {        3571, "MQCACH_TOPIC_ROOT" },
   {        3840, "MQDCC_TARGET_ENC_MASK" },
   {        3840, "MQENC_FLOAT_MASK" },
   {        4000, "MQCA_LAST" },
   {        4000, "MQCA_USER_LIST" },
   {        4000, "MQ_MSG_HEADER_LENGTH" },
   {        4001, "MQHA_BAG_HANDLE" },
   {        4001, "MQHA_FIRST" },
   {        4001, "MQHA_LAST_USED" },
   {        4001, "MQRCCF_OBJECT_ALREADY_EXISTS" },
   {        4002, "MQRCCF_OBJECT_WRONG_TYPE" },
   {        4003, "MQRCCF_LIKE_OBJECT_WRONG_TYPE" },
   {        4004, "MQRCCF_OBJECT_OPEN" },
   {        4005, "MQRCCF_ATTR_VALUE_ERROR" },
   {        4006, "MQRCCF_UNKNOWN_Q_MGR" },
   {        4007, "MQRCCF_Q_WRONG_TYPE" },
   {        4008, "MQRCCF_OBJECT_NAME_ERROR" },
   {        4009, "MQRCCF_ALLOCATE_FAILED" },
   {        4010, "MQRCCF_HOST_NOT_AVAILABLE" },
   {        4011, "MQRCCF_CONFIGURATION_ERROR" },
   {        4012, "MQRCCF_CONNECTION_REFUSED" },
   {        4013, "MQRCCF_ENTRY_ERROR" },
   {        4014, "MQRCCF_SEND_FAILED" },
   {        4015, "MQRCCF_RECEIVED_DATA_ERROR" },
   {        4016, "MQRCCF_RECEIVE_FAILED" },
   {        4017, "MQRCCF_CONNECTION_CLOSED" },
   {        4018, "MQRCCF_NO_STORAGE" },
   {        4019, "MQRCCF_NO_COMMS_MANAGER" },
   {        4020, "MQRCCF_LISTENER_NOT_STARTED" },
   {        4024, "MQRCCF_BIND_FAILED" },
   {        4025, "MQRCCF_CHANNEL_INDOUBT" },
   {        4026, "MQRCCF_MQCONN_FAILED" },
   {        4027, "MQRCCF_MQOPEN_FAILED" },
   {        4028, "MQRCCF_MQGET_FAILED" },
   {        4029, "MQRCCF_MQPUT_FAILED" },
   {        4030, "MQRCCF_PING_ERROR" },
   {        4031, "MQRCCF_CHANNEL_IN_USE" },
   {        4032, "MQRCCF_CHANNEL_NOT_FOUND" },
   {        4033, "MQRCCF_UNKNOWN_REMOTE_CHANNEL" },
   {        4034, "MQRCCF_REMOTE_QM_UNAVAILABLE" },
   {        4035, "MQRCCF_REMOTE_QM_TERMINATING" },
   {        4036, "MQRCCF_MQINQ_FAILED" },
   {        4037, "MQRCCF_NOT_XMIT_Q" },
   {        4038, "MQRCCF_CHANNEL_DISABLED" },
   {        4039, "MQRCCF_USER_EXIT_NOT_AVAILABLE" },
   {        4040, "MQRCCF_COMMIT_FAILED" },
   {        4041, "MQRCCF_WRONG_CHANNEL_TYPE" },
   {        4042, "MQRCCF_CHANNEL_ALREADY_EXISTS" },
   {        4043, "MQRCCF_DATA_TOO_LARGE" },
   {        4044, "MQRCCF_CHANNEL_NAME_ERROR" },
   {        4045, "MQRCCF_XMIT_Q_NAME_ERROR" },
   {        4047, "MQRCCF_MCA_NAME_ERROR" },
   {        4048, "MQRCCF_SEND_EXIT_NAME_ERROR" },
   {        4049, "MQRCCF_SEC_EXIT_NAME_ERROR" },
   {        4050, "MQRCCF_MSG_EXIT_NAME_ERROR" },
   {        4051, "MQRCCF_RCV_EXIT_NAME_ERROR" },
   {        4052, "MQRCCF_XMIT_Q_NAME_WRONG_TYPE" },
   {        4053, "MQRCCF_MCA_NAME_WRONG_TYPE" },
   {        4054, "MQRCCF_DISC_INT_WRONG_TYPE" },
   {        4055, "MQRCCF_SHORT_RETRY_WRONG_TYPE" },
   {        4056, "MQRCCF_SHORT_TIMER_WRONG_TYPE" },
   {        4057, "MQRCCF_LONG_RETRY_WRONG_TYPE" },
   {        4058, "MQRCCF_LONG_TIMER_WRONG_TYPE" },
   {        4059, "MQRCCF_PUT_AUTH_WRONG_TYPE" },
   {        4060, "MQRCCF_KEEP_ALIVE_INT_ERROR" },
   {        4061, "MQRCCF_MISSING_CONN_NAME" },
   {        4062, "MQRCCF_CONN_NAME_ERROR" },
   {        4063, "MQRCCF_MQSET_FAILED" },
   {        4064, "MQRCCF_CHANNEL_NOT_ACTIVE" },
   {        4065, "MQRCCF_TERMINATED_BY_SEC_EXIT" },
   {        4067, "MQRCCF_DYNAMIC_Q_SCOPE_ERROR" },
   {        4068, "MQRCCF_CELL_DIR_NOT_AVAILABLE" },
   {        4069, "MQRCCF_MR_COUNT_ERROR" },
   {        4070, "MQRCCF_MR_COUNT_WRONG_TYPE" },
   {        4071, "MQRCCF_MR_EXIT_NAME_ERROR" },
   {        4072, "MQRCCF_MR_EXIT_NAME_WRONG_TYPE" },
   {        4073, "MQRCCF_MR_INTERVAL_ERROR" },
   {        4074, "MQRCCF_MR_INTERVAL_WRONG_TYPE" },
   {        4075, "MQRCCF_NPM_SPEED_ERROR" },
   {        4076, "MQRCCF_NPM_SPEED_WRONG_TYPE" },
   {        4077, "MQRCCF_HB_INTERVAL_ERROR" },
   {        4078, "MQRCCF_HB_INTERVAL_WRONG_TYPE" },
   {        4079, "MQRCCF_CHAD_ERROR" },
   {        4080, "MQRCCF_CHAD_WRONG_TYPE" },
   {        4081, "MQRCCF_CHAD_EVENT_ERROR" },
   {        4082, "MQRCCF_CHAD_EVENT_WRONG_TYPE" },
   {        4083, "MQRCCF_CHAD_EXIT_ERROR" },
   {        4084, "MQRCCF_CHAD_EXIT_WRONG_TYPE" },
   {        4085, "MQRCCF_SUPPRESSED_BY_EXIT" },
   {        4086, "MQRCCF_BATCH_INT_ERROR" },
   {        4087, "MQRCCF_BATCH_INT_WRONG_TYPE" },
   {        4088, "MQRCCF_NET_PRIORITY_ERROR" },
   {        4089, "MQRCCF_NET_PRIORITY_WRONG_TYPE" },
   {        4090, "MQRCCF_CHANNEL_CLOSED" },
   {        4091, "MQRCCF_Q_STATUS_NOT_FOUND" },
   {        4092, "MQRCCF_SSL_CIPHER_SPEC_ERROR" },
   {        4093, "MQRCCF_SSL_PEER_NAME_ERROR" },
   {        4094, "MQRCCF_SSL_CLIENT_AUTH_ERROR" },
   {        4095, "MQMF_REJECT_UNSUP_MASK" },
   {        4095, "MQRCCF_RETAINED_NOT_SUPPORTED" },
   {        4095, "MQ_MAX_PROPERTY_NAME_LENGTH" },
   {        4096, "MQCNO_ACCOUNTING_MQI_ENABLED" },
   {        4096, "MQGMO_SYNCPOINT_IF_PERSISTENT" },
   {        4096, "MQOO_ALTERNATE_USER_AUTHORITY" },
   {        4096, "MQPMO_ALTERNATE_USER_AUTHORITY" },
   {        4096, "MQREGO_PERSISTENT_AS_PUBLISH" },
   {        4096, "MQROUTE_DELIVER_YES" },
   {        4096, "MQSO_NEW_PUBLICATIONS_ONLY" },
   {        4096, "MQZAO_SUBSCRIBE" },
   {        4352, "MQCTES_BACKOUT" },
   {        4352, "MQCUOWC_BACKOUT" },
   {        5507, "MQCACF_CLUS_CHAN_Q_MGR_NAME" },
   {        5508, "MQCACF_CLUS_SHORT_CONN_NAME" },
   {        6000, "MQHA_LAST" },
   {        6001, "MQBA_FIRST" },
   {        6100, "MQRC_REOPEN_EXCL_INPUT_ERROR" },
   {        6101, "MQRC_REOPEN_INQUIRE_ERROR" },
   {        6102, "MQRC_REOPEN_SAVED_CONTEXT_ERR" },
   {        6103, "MQRC_REOPEN_TEMPORARY_Q_ERROR" },
   {        6104, "MQRC_ATTRIBUTE_LOCKED" },
   {        6105, "MQRC_CURSOR_NOT_VALID" },
   {        6106, "MQRC_ENCODING_ERROR" },
   {        6107, "MQRC_STRUC_ID_ERROR" },
   {        6108, "MQRC_NULL_POINTER" },
   {        6109, "MQRC_NO_CONNECTION_REFERENCE" },
   {        6110, "MQRC_NO_BUFFER" },
   {        6111, "MQRC_BINARY_DATA_LENGTH_ERROR" },
   {        6112, "MQRC_BUFFER_NOT_AUTOMATIC" },
   {        6113, "MQRC_INSUFFICIENT_BUFFER" },
   {        6114, "MQRC_INSUFFICIENT_DATA" },
   {        6115, "MQRC_DATA_TRUNCATED" },
   {        6116, "MQRC_ZERO_LENGTH" },
   {        6117, "MQRC_NEGATIVE_LENGTH" },
   {        6118, "MQRC_NEGATIVE_OFFSET" },
   {        6119, "MQRC_INCONSISTENT_FORMAT" },
   {        6120, "MQRC_INCONSISTENT_OBJECT_STATE" },
   {        6121, "MQRC_CONTEXT_OBJECT_NOT_VALID" },
   {        6122, "MQRC_CONTEXT_OPEN_ERROR" },
   {        6123, "MQRC_STRUC_LENGTH_ERROR" },
   {        6124, "MQRC_NOT_CONNECTED" },
   {        6125, "MQRC_NOT_OPEN" },
   {        6126, "MQRC_DISTRIBUTION_LIST_EMPTY" },
   {        6127, "MQRC_INCONSISTENT_OPEN_OPTIONS" },
   {        6128, "MQRC_WRONG_VERSION" },
   {        6129, "MQRC_REFERENCE_ERROR" },
   {        6130, "MQRC_XR_NOT_AVAILABLE" },
   {        6144, "MQRO_COD_WITH_DATA" },
   {        7001, "MQBACF_EVENT_ACCOUNTING_TOKEN" },
   {        7001, "MQBACF_FIRST" },
   {        7002, "MQBACF_EVENT_SECURITY_ID" },
   {        7003, "MQBACF_RESPONSE_SET" },
   {        7004, "MQBACF_RESPONSE_ID" },
   {        7005, "MQBACF_EXTERNAL_UOW_ID" },
   {        7006, "MQBACF_CONNECTION_ID" },
   {        7007, "MQBACF_GENERIC_CONNECTION_ID" },
   {        7008, "MQBACF_ORIGIN_UOW_ID" },
   {        7009, "MQBACF_Q_MGR_UOW_ID" },
   {        7010, "MQBACF_ACCOUNTING_TOKEN" },
   {        7011, "MQBACF_CORREL_ID" },
   {        7012, "MQBACF_GROUP_ID" },
   {        7013, "MQBACF_MSG_ID" },
   {        7014, "MQBACF_CF_LEID" },
   {        7015, "MQBACF_DESTINATION_CORREL_ID" },
   {        7016, "MQBACF_SUB_ID" },
   {        7019, "MQBACF_ALTERNATE_SECURITYID" },
   {        7020, "MQBACF_MESSAGE_DATA" },
   {        7021, "MQBACF_MQBO_STRUCT" },
   {        7022, "MQBACF_MQCB_FUNCTION" },
   {        7023, "MQBACF_MQCBC_STRUCT" },
   {        7024, "MQBACF_MQCBD_STRUCT" },
   {        7025, "MQBACF_MQCD_STRUCT" },
   {        7026, "MQBACF_MQCNO_STRUCT" },
   {        7027, "MQBACF_MQGMO_STRUCT" },
   {        7028, "MQBACF_MQMD_STRUCT" },
   {        7029, "MQBACF_MQPMO_STRUCT" },
   {        7030, "MQBACF_MQSD_STRUCT" },
   {        7031, "MQBACF_MQSTS_STRUCT" },
   {        7032, "MQBACF_SUB_CORREL_ID" },
   {        7033, "MQBACF_XA_XID" },
   {        7034, "MQBACF_XQH_CORREL_ID" },
   {        7035, "MQBACF_LAST_USED" },
   {        7035, "MQBACF_XQH_MSG_ID" },
   {        8000, "MQBA_LAST" },
   {        8000, "MQFIELD_WQR_StrucId" },
   {        8001, "MQFIELD_WQR_Version" },
   {        8001, "MQGACF_COMMAND_CONTEXT" },
   {        8001, "MQGACF_FIRST" },
   {        8001, "MQGA_FIRST" },
   {        8002, "MQFIELD_WQR_StrucLength" },
   {        8002, "MQGACF_COMMAND_DATA" },
   {        8003, "MQFIELD_WQR_QFlags" },
   {        8003, "MQGACF_TRACE_ROUTE" },
   {        8004, "MQFIELD_WQR_QName" },
   {        8004, "MQGACF_OPERATION" },
   {        8005, "MQFIELD_WQR_QMgrIdentifier" },
   {        8005, "MQGACF_ACTIVITY" },
   {        8006, "MQFIELD_WQR_ClusterRecOffset" },
   {        8006, "MQGACF_EMBEDDED_MQMD" },
   {        8007, "MQFIELD_WQR_QType" },
   {        8007, "MQGACF_MESSAGE" },
   {        8008, "MQFIELD_WQR_QDesc" },
   {        8008, "MQGACF_MQMD" },
   {        8009, "MQFIELD_WQR_DefBind" },
   {        8009, "MQGACF_VALUE_NAMING" },
   {        8010, "MQFIELD_WQR_DefPersistence" },
   {        8010, "MQGACF_Q_ACCOUNTING_DATA" },
   {        8011, "MQFIELD_WQR_DefPriority" },
   {        8011, "MQGACF_Q_STATISTICS_DATA" },
   {        8012, "MQFIELD_WQR_InhibitPut" },
   {        8012, "MQGACF_CHL_STATISTICS_DATA" },
   {        8013, "MQFIELD_WQR_CLWLQueuePriority" },
   {        8013, "MQGACF_ACTIVITY_TRACE" },
   {        8014, "MQFIELD_WQR_CLWLQueueRank" },
   {        8014, "MQGACF_APP_DIST_LIST" },
   {        8015, "MQFIELD_WQR_DefPutResponse" },
   {        8015, "MQGACF_MONITOR_CLASS" },
   {        8016, "MQGACF_MONITOR_TYPE" },
   {        8017, "MQGACF_LAST_USED" },
   {        8017, "MQGACF_MONITOR_ELEMENT" },
   {        8192, "MQCBDO_FAIL_IF_QUIESCING" },
   {        8192, "MQCNO_ACCOUNTING_MQI_DISABLED" },
   {        8192, "MQCTLO_FAIL_IF_QUIESCING" },
   {        8192, "MQGMO_FAIL_IF_QUIESCING" },
   {        8192, "MQOO_FAIL_IF_QUIESCING" },
   {        8192, "MQPMO_FAIL_IF_QUIESCING" },
   {        8192, "MQREGO_PERSISTENT_AS_Q" },
   {        8192, "MQROUTE_DELIVER_NO" },
   {        8192, "MQSO_FAIL_IF_QUIESCING" },
   {        8192, "MQSRO_FAIL_IF_QUIESCING" },
   {        8192, "MQZAO_RESUME" },
   {        9000, "MQGA_LAST" },
   {        9000, "MQOA_LAST" },
   {       10000, "MQIAMO_MONITOR_PERCENT" },
   {       10240, "MQ_SELECTOR_LENGTH" },
   {       10240, "MQ_SUB_NAME_LENGTH" },
   {       10240, "MQ_TOPIC_STR_LENGTH" },
   {       10240, "MQ_USER_DATA_LENGTH" },
   {       14336, "MQRO_COD_WITH_FULL_DATA" },
   {       16383, "MQZAO_ALL_MQI" },
   {       16384, "MQCBDO_EVENT_CALL" },
   {       16384, "MQCNO_ACCOUNTING_Q_ENABLED" },
   {       16384, "MQGMO_CONVERT" },
   {       16384, "MQOO_BIND_ON_OPEN" },
   {       16384, "MQPMO_NO_CONTEXT" },
   {       16384, "MQREGO_ADD_NAME" },
   {       16384, "MQRO_PASS_DISCARD_AND_EXPIRY" },
   {       29440, "MQRC_SUB_JOIN_NOT_ALTERABLE" },
   {       32768, "MQCBDO_MC_EVENT_CALL" },
   {       32768, "MQCNO_ACCOUNTING_Q_DISABLED" },
   {       32768, "MQGMO_LOGICAL_ORDER" },
   {       32768, "MQOO_BIND_NOT_FIXED" },
   {       32768, "MQPMO_LOGICAL_ORDER" },
   {       32768, "MQREGO_NO_ALTERATION" },
   {       32768, "MQ_COMMAND_MQSC_LENGTH" },
   {       65535, "MQFB_SYSTEM_LAST" },
   {       65535, "MQMT_SYSTEM_LAST" },
   {       65535, "MQOPER_SYSTEM_LAST" },
   {       65536, "MQAT_USER_FIRST" },
   {       65536, "MQCNO_NO_CONV_SHARING" },
   {       65536, "MQCTES_ENDTASK" },
   {       65536, "MQCUOWC_CONTINUE" },
   {       65536, "MQFB_APPL_FIRST" },
   {       65536, "MQGMO_COMPLETE_MSG" },
   {       65536, "MQMT_APPL_FIRST" },
   {       65536, "MQOO_RESOLVE_NAMES" },
   {       65536, "MQOPER_APPL_FIRST" },
   {       65536, "MQOP_SUSPEND" },
   {       65536, "MQPMO_ASYNC_RESPONSE" },
   {       65536, "MQREGO_FULL_RESPONSE" },
   {       65536, "MQUA_FIRST" },
   {       65536, "MQZAO_CREATE" },
   {       65536, "MQ_MQTT_MAX_KEEP_ALIVE" },
   {       65539, "MQROUTE_ACCUMULATE_NONE" },
   {       65540, "MQROUTE_ACCUMULATE_IN_MSG" },
   {       65541, "MQROUTE_ACCUMULATE_AND_REPLY" },
   {      131072, "MQGMO_ALL_MSGS_AVAILABLE" },
   {      131072, "MQOO_CO_OP" },
   {      131072, "MQOP_RESUME" },
   {      131072, "MQPMO_SYNC_RESPONSE" },
   {      131072, "MQREGO_JOIN_SHARED" },
   {      131072, "MQZAO_DELETE" },
   {      261888, "MQRO_ACCEPT_UNSUP_IF_XMIT_MASK" },
   {      262144, "MQCNO_ALL_CONVS_SHARE" },
   {      262144, "MQGMO_ALL_SEGMENTS_AVAILABLE" },
   {      262144, "MQOO_RESOLVE_LOCAL_Q" },
   {      262144, "MQOO_RESOLVE_LOCAL_TOPIC" },
   {      262144, "MQPMO_RESOLVE_LOCAL_Q" },
   {      262144, "MQREGO_JOIN_EXCLUSIVE" },
   {      262144, "MQSO_ALTERNATE_USER_AUTHORITY" },
   {      262144, "MQZAO_DISPLAY" },
   {      524288, "MQCNO_CD_FOR_OUTPUT_ONLY" },
   {      524288, "MQOO_NO_READ_AHEAD" },
   {      524288, "MQPMO_WARN_IF_NO_SUBS_MATCHED" },
   {      524288, "MQREGO_LEAVE_ONLY" },
   {      524288, "MQZAO_CHANGE" },
   {     1000000, "MQIAMO_MONITOR_MICROSEC" },
   {     1044480, "MQMF_ACCEPT_UNSUP_IF_XMIT_MASK" },
   {     1047552, "MQPD_ACCEPT_UNSUP_IF_XMIT_MASK" },
   {     1048576, "MQCNO_USE_CD_SELECTION" },
   {     1048576, "MQGMO_MARK_BROWSE_HANDLE" },
   {     1048576, "MQIAMO_MONITOR_MB" },
   {     1048576, "MQOO_READ_AHEAD" },
   {     1048576, "MQPD_SUPPORT_REQUIRED" },
   {     1048576, "MQREGO_VARIABLE_USER_ID" },
   {     1048576, "MQSO_WILDCARD_CHAR" },
   {     1048576, "MQZAO_CLEAR" },
   {     2097152, "MQGMO_MARK_BROWSE_CO_OP" },
   {     2097152, "MQOO_NO_MULTICAST" },
   {     2097152, "MQPMO_PUB_OPTIONS_MASK" },
   {     2097152, "MQPMO_RETAIN" },
   {     2097152, "MQREGO_LOCKED" },
   {     2097152, "MQRO_EXPIRATION" },
   {     2097152, "MQSO_WILDCARD_TOPIC" },
   {     2097152, "MQZAO_CONTROL" },
   {     4194304, "MQGMO_UNMARK_BROWSE_CO_OP" },
   {     4194304, "MQOO_BIND_ON_GROUP" },
   {     4194304, "MQSO_SET_CORREL_ID" },
   {     4194304, "MQZAO_CONTROL_EXTENDED" },
   {     6291456, "MQRO_EXPIRATION_WITH_DATA" },
   {     8388608, "MQGMO_UNMARK_BROWSE_HANDLE" },
   {     8388608, "MQPMO_MD_FOR_OUTPUT_ONLY" },
   {     8388608, "MQZAO_AUTHORIZE" },
   {    14680064, "MQRO_EXPIRATION_WITH_FULL_DATA" },
   {    16646144, "MQZAO_ALL_ADMIN" },
   {    16777216, "MQCNO_RECONNECT" },
   {    16777216, "MQGMO_UNMARKED_BROWSE_MSG" },
   {    16777216, "MQRO_EXCEPTION" },
   {    16777216, "MQZAO_REMOVE" },
   {    17825808, "MQGMO_BROWSE_HANDLE" },
   {    18874384, "MQGMO_BROWSE_CO_OP" },
   {    33554432, "MQCNO_RECONNECT_DISABLED" },
   {    33554432, "MQGMO_PROPERTIES_FORCE_MQRFH2" },
   {    33554432, "MQZAO_SYSTEM" },
   {    50216959, "MQZAO_ALL" },
   {    50331648, "MQRO_EXCEPTION_WITH_DATA" },
   {    67108864, "MQCNO_RECONNECT_Q_MGR" },
   {    67108864, "MQGMO_NO_PROPERTIES" },
   {    67108864, "MQPMO_SCOPE_QMGR" },
   {    67108864, "MQSO_SCOPE_QMGR" },
   {    67108864, "MQZAO_CREATE_ONLY" },
   {   100000000, "MQIAMO_MONITOR_GB" },
   {   117440512, "MQRO_EXCEPTION_WITH_FULL_DATA" },
   {   134217728, "MQCNO_ACTIVITY_TRACE_ENABLED" },
   {   134217728, "MQGMO_PROPERTIES_IN_HANDLE" },
   {   134217728, "MQPMO_SUPPRESS_REPLYTO" },
   {   134217728, "MQRO_DISCARD_MSG" },
   {   134217728, "MQSO_NO_READ_AHEAD" },
   {   268435455, "MQCOMPRESS_ANY" },
   {   268435456, "MQCNO_ACTIVITY_TRACE_DISABLED" },
   {   268435456, "MQGMO_PROPERTIES_COMPATIBILITY" },
   {   268435456, "MQPMO_NOT_OWN_SUBS" },
   {   268435456, "MQSO_READ_AHEAD" },
   {   270270464, "MQRO_REJECT_UNSUP_MASK" },
   {   999999999, "MQAT_USER_LAST" },
   {   999999999, "MQFB_APPL_LAST" },
   {   999999999, "MQMT_APPL_LAST" },
   {   999999999, "MQOPER_APPL_LAST" },
   {   999999999, "MQUA_LAST" },
   {  0, "" }
 };



 const struct MQI_BY_NAME_STR { 
  char *name;
  MQLONG value;
 } MQI_BY_NAME_STR[] = { 
   {  "MQACH_CURRENT_LENGTH (4 byte)"   ,         68 },
   {  "MQACH_CURRENT_LENGTH (8 byte)"   ,         72 },
   {  "MQACH_CURRENT_VERSION"           ,          1 },
   {  "MQACH_LENGTH_1 (4 byte)"         ,         68 },
   {  "MQACH_LENGTH_1 (8 byte)"         ,         72 },
   {  "MQACH_VERSION_1"                 ,          1 },
   {  "MQACTP_FORWARD"                  ,          1 },
   {  "MQACTP_NEW"                      ,          0 },
   {  "MQACTP_REPLY"                    ,          2 },
   {  "MQACTP_REPORT"                   ,          3 },
   {  "MQACTV_DETAIL_HIGH"              ,          3 },
   {  "MQACTV_DETAIL_LOW"               ,          1 },
   {  "MQACTV_DETAIL_MEDIUM"            ,          2 },
   {  "MQACT_ADD"                       ,          5 },
   {  "MQACT_ADVANCE_LOG"               ,          2 },
   {  "MQACT_ARCHIVE_LOG"               ,         11 },
   {  "MQACT_COLLECT_STATISTICS"        ,          3 },
   {  "MQACT_FAIL"                      ,          9 },
   {  "MQACT_FORCE_REMOVE"              ,          1 },
   {  "MQACT_PUBSUB"                    ,          4 },
   {  "MQACT_REDUCE_LOG"                ,         10 },
   {  "MQACT_REMOVE"                    ,          7 },
   {  "MQACT_REMOVEALL"                 ,          8 },
   {  "MQACT_REPLACE"                   ,          6 },
   {  "MQADOPT_CHECK_ALL"               ,          1 },
   {  "MQADOPT_CHECK_CHANNEL_NAME"      ,          8 },
   {  "MQADOPT_CHECK_NET_ADDR"          ,          4 },
   {  "MQADOPT_CHECK_NONE"              ,          0 },
   {  "MQADOPT_CHECK_Q_MGR_NAME"        ,          2 },
   {  "MQADOPT_TYPE_ALL"                ,          1 },
   {  "MQADOPT_TYPE_CLUSRCVR"           ,         16 },
   {  "MQADOPT_TYPE_NO"                 ,          0 },
   {  "MQADOPT_TYPE_RCVR"               ,          8 },
   {  "MQADOPT_TYPE_SDR"                ,          4 },
   {  "MQADOPT_TYPE_SVR"                ,          2 },
   {  "MQADPCTX_NO"                     ,          0 },
   {  "MQADPCTX_YES"                    ,          1 },
   {  "MQAIR_CURRENT_LENGTH (4 byte)"   ,        576 },
   {  "MQAIR_CURRENT_LENGTH (8 byte)"   ,        584 },
   {  "MQAIR_CURRENT_VERSION"           ,          2 },
   {  "MQAIR_LENGTH_1 (4 byte)"         ,        320 },
   {  "MQAIR_LENGTH_1 (8 byte)"         ,        328 },
   {  "MQAIR_LENGTH_2 (4 byte)"         ,        576 },
   {  "MQAIR_LENGTH_2 (8 byte)"         ,        584 },
   {  "MQAIR_VERSION_1"                 ,          1 },
   {  "MQAIR_VERSION_2"                 ,          2 },
   {  "MQAIT_ALL"                       ,          0 },
   {  "MQAIT_CRL_LDAP"                  ,          1 },
   {  "MQAIT_IDPW_LDAP"                 ,          4 },
   {  "MQAIT_IDPW_OS"                   ,          3 },
   {  "MQAIT_OCSP"                      ,          2 },
   {  "MQAS_ACTIVE"                     ,          6 },
   {  "MQAS_INACTIVE"                   ,          7 },
   {  "MQAS_NONE"                       ,          0 },
   {  "MQAS_STARTED"                    ,          1 },
   {  "MQAS_START_WAIT"                 ,          2 },
   {  "MQAS_STOPPED"                    ,          3 },
   {  "MQAS_SUSPENDED"                  ,          4 },
   {  "MQAS_SUSPENDED_TEMPORARY"        ,          5 },
   {  "MQAT_AIX"                        ,          6 },
   {  "MQAT_AMQP"                       ,         37 },
   {  "MQAT_BATCH"                      ,         32 },
   {  "MQAT_BROKER"                     ,         26 },
   {  "MQAT_CHANNEL_INITIATOR"          ,         30 },
   {  "MQAT_CICS"                       ,          1 },
   {  "MQAT_CICS_BRIDGE"                ,         21 },
   {  "MQAT_CICS_VSE"                   ,         10 },
   {  "MQAT_DEFAULT"                    ,          6 },
   {  "MQAT_DOS"                        ,          5 },
   {  "MQAT_DQM"                        ,         29 },
   {  "MQAT_GUARDIAN"                   ,         13 },
   {  "MQAT_IMS"                        ,          3 },
   {  "MQAT_IMS_BRIDGE"                 ,         19 },
   {  "MQAT_JAVA"                       ,         28 },
   {  "MQAT_MCAST_PUBLISH"              ,         36 },
   {  "MQAT_MVS"                        ,          2 },
   {  "MQAT_NOTES_AGENT"                ,         22 },
   {  "MQAT_NO_CONTEXT"                 ,          0 },
   {  "MQAT_NSK"                        ,         13 },
   {  "MQAT_OPEN_TP1"                   ,         15 },
   {  "MQAT_OS2"                        ,          4 },
   {  "MQAT_OS390"                      ,          2 },
   {  "MQAT_OS400"                      ,          8 },
   {  "MQAT_QMGR"                       ,          7 },
   {  "MQAT_QMGR_PUBLISH"               ,         26 },
   {  "MQAT_RRS_BATCH"                  ,         33 },
   {  "MQAT_SIB"                        ,         34 },
   {  "MQAT_SYSTEM_EXTENSION"           ,         35 },
   {  "MQAT_TPF"                        ,         23 },
   {  "MQAT_UNIX"                       ,          6 },
   {  "MQAT_UNKNOWN"                    ,         -1 },
   {  "MQAT_USER"                       ,         25 },
   {  "MQAT_USER_FIRST"                 ,      65536 },
   {  "MQAT_USER_LAST"                  ,  999999999 },
   {  "MQAT_VM"                         ,         18 },
   {  "MQAT_VMS"                        ,         12 },
   {  "MQAT_VOS"                        ,         14 },
   {  "MQAT_WINDOWS"                    ,          9 },
   {  "MQAT_WINDOWS_NT"                 ,         11 },
   {  "MQAT_WLM"                        ,         31 },
   {  "MQAT_XCF"                        ,         20 },
   {  "MQAT_ZOS"                        ,          2 },
   {  "MQAUTHENTICATE_OS"               ,          0 },
   {  "MQAUTHENTICATE_PAM"              ,          1 },
   {  "MQAUTHOPT_CUMULATIVE"            ,        256 },
   {  "MQAUTHOPT_ENTITY_EXPLICIT"       ,          1 },
   {  "MQAUTHOPT_ENTITY_SET"            ,          2 },
   {  "MQAUTHOPT_EXCLUDE_TEMP"          ,        512 },
   {  "MQAUTHOPT_NAME_ALL_MATCHING"     ,         32 },
   {  "MQAUTHOPT_NAME_AS_WILDCARD"      ,         64 },
   {  "MQAUTHOPT_NAME_EXPLICIT"         ,         16 },
   {  "MQAUTH_ALL"                      ,         -1 },
   {  "MQAUTH_ALL_ADMIN"                ,         -2 },
   {  "MQAUTH_ALL_MQI"                  ,         -3 },
   {  "MQAUTH_ALT_USER_AUTHORITY"       ,          1 },
   {  "MQAUTH_BROWSE"                   ,          2 },
   {  "MQAUTH_CHANGE"                   ,          3 },
   {  "MQAUTH_CLEAR"                    ,          4 },
   {  "MQAUTH_CONNECT"                  ,          5 },
   {  "MQAUTH_CONTROL"                  ,         17 },
   {  "MQAUTH_CONTROL_EXTENDED"         ,         18 },
   {  "MQAUTH_CREATE"                   ,          6 },
   {  "MQAUTH_DELETE"                   ,          7 },
   {  "MQAUTH_DISPLAY"                  ,          8 },
   {  "MQAUTH_INPUT"                    ,          9 },
   {  "MQAUTH_INQUIRE"                  ,         10 },
   {  "MQAUTH_NONE"                     ,          0 },
   {  "MQAUTH_OUTPUT"                   ,         11 },
   {  "MQAUTH_PASS_ALL_CONTEXT"         ,         12 },
   {  "MQAUTH_PASS_IDENTITY_CONTEXT"    ,         13 },
   {  "MQAUTH_PUBLISH"                  ,         19 },
   {  "MQAUTH_RESUME"                   ,         21 },
   {  "MQAUTH_SET"                      ,         14 },
   {  "MQAUTH_SET_ALL_CONTEXT"          ,         15 },
   {  "MQAUTH_SET_IDENTITY_CONTEXT"     ,         16 },
   {  "MQAUTH_SUBSCRIBE"                ,         20 },
   {  "MQAUTH_SYSTEM"                   ,         22 },
   {  "MQAUTO_START_NO"                 ,          0 },
   {  "MQAUTO_START_YES"                ,          1 },
   {  "MQAXC_CURRENT_LENGTH (4 byte)"   ,        412 },
   {  "MQAXC_CURRENT_LENGTH (8 byte)"   ,        424 },
   {  "MQAXC_CURRENT_VERSION"           ,          2 },
   {  "MQAXC_LENGTH_1 (4 byte)"         ,        384 },
   {  "MQAXC_LENGTH_1 (8 byte)"         ,        392 },
   {  "MQAXC_LENGTH_2 (4 byte)"         ,        412 },
   {  "MQAXC_LENGTH_2 (8 byte)"         ,        424 },
   {  "MQAXC_VERSION_1"                 ,          1 },
   {  "MQAXC_VERSION_2"                 ,          2 },
   {  "MQAXP_CURRENT_LENGTH (4 byte)"   ,        244 },
   {  "MQAXP_CURRENT_LENGTH (8 byte)"   ,        256 },
   {  "MQAXP_CURRENT_VERSION"           ,          2 },
   {  "MQAXP_LENGTH_1 (4 byte)"         ,        244 },
   {  "MQAXP_LENGTH_1 (8 byte)"         ,        256 },
   {  "MQAXP_VERSION_1"                 ,          1 },
   {  "MQAXP_VERSION_2"                 ,          2 },
   {  "MQBACF_ACCOUNTING_TOKEN"         ,       7010 },
   {  "MQBACF_ALTERNATE_SECURITYID"     ,       7019 },
   {  "MQBACF_CF_LEID"                  ,       7014 },
   {  "MQBACF_CONNECTION_ID"            ,       7006 },
   {  "MQBACF_CORREL_ID"                ,       7011 },
   {  "MQBACF_DESTINATION_CORREL_ID"    ,       7015 },
   {  "MQBACF_EVENT_ACCOUNTING_TOKEN"   ,       7001 },
   {  "MQBACF_EVENT_SECURITY_ID"        ,       7002 },
   {  "MQBACF_EXTERNAL_UOW_ID"          ,       7005 },
   {  "MQBACF_FIRST"                    ,       7001 },
   {  "MQBACF_GENERIC_CONNECTION_ID"    ,       7007 },
   {  "MQBACF_GROUP_ID"                 ,       7012 },
   {  "MQBACF_LAST_USED"                ,       7035 },
   {  "MQBACF_MESSAGE_DATA"             ,       7020 },
   {  "MQBACF_MQBO_STRUCT"              ,       7021 },
   {  "MQBACF_MQCBC_STRUCT"             ,       7023 },
   {  "MQBACF_MQCBD_STRUCT"             ,       7024 },
   {  "MQBACF_MQCB_FUNCTION"            ,       7022 },
   {  "MQBACF_MQCD_STRUCT"              ,       7025 },
   {  "MQBACF_MQCNO_STRUCT"             ,       7026 },
   {  "MQBACF_MQGMO_STRUCT"             ,       7027 },
   {  "MQBACF_MQMD_STRUCT"              ,       7028 },
   {  "MQBACF_MQPMO_STRUCT"             ,       7029 },
   {  "MQBACF_MQSD_STRUCT"              ,       7030 },
   {  "MQBACF_MQSTS_STRUCT"             ,       7031 },
   {  "MQBACF_MSG_ID"                   ,       7013 },
   {  "MQBACF_ORIGIN_UOW_ID"            ,       7008 },
   {  "MQBACF_Q_MGR_UOW_ID"             ,       7009 },
   {  "MQBACF_RESPONSE_ID"              ,       7004 },
   {  "MQBACF_RESPONSE_SET"             ,       7003 },
   {  "MQBACF_SUB_CORREL_ID"            ,       7032 },
   {  "MQBACF_SUB_ID"                   ,       7016 },
   {  "MQBACF_XA_XID"                   ,       7033 },
   {  "MQBACF_XQH_CORREL_ID"            ,       7034 },
   {  "MQBACF_XQH_MSG_ID"               ,       7035 },
   {  "MQBA_FIRST"                      ,       6001 },
   {  "MQBA_LAST"                       ,       8000 },
   {  "MQBL_NULL_TERMINATED"            ,         -1 },
   {  "MQBMHO_CURRENT_LENGTH"           ,         12 },
   {  "MQBMHO_CURRENT_VERSION"          ,          1 },
   {  "MQBMHO_DELETE_PROPERTIES"        ,          1 },
   {  "MQBMHO_LENGTH_1"                 ,         12 },
   {  "MQBMHO_NONE"                     ,          0 },
   {  "MQBMHO_VERSION_1"                ,          1 },
   {  "MQBND_BIND_NOT_FIXED"            ,          1 },
   {  "MQBND_BIND_ON_GROUP"             ,          2 },
   {  "MQBND_BIND_ON_OPEN"              ,          0 },
   {  "MQBO_CURRENT_LENGTH"             ,         12 },
   {  "MQBO_CURRENT_VERSION"            ,          1 },
   {  "MQBO_LENGTH_1"                   ,         12 },
   {  "MQBO_NONE"                       ,          0 },
   {  "MQBO_VERSION_1"                  ,          1 },
   {  "MQBPLOCATION_ABOVE"              ,          1 },
   {  "MQBPLOCATION_BELOW"              ,          0 },
   {  "MQBPLOCATION_SWITCHING_ABOVE"    ,          2 },
   {  "MQBPLOCATION_SWITCHING_BELOW"    ,          3 },
   {  "MQBT_OTMA"                       ,          1 },
   {  "MQCACF_ACTIVITY_DESC"            ,       3134 },
   {  "MQCACF_ADMIN_TOPIC_NAMES"        ,       3172 },
   {  "MQCACF_ALIAS_Q_NAMES"            ,       3017 },
   {  "MQCACF_ALTERNATE_USERID"         ,       3188 },
   {  "MQCACF_AMQP_CLIENT_ID"           ,       3207 },
   {  "MQCACF_APPL_DESC"                ,       3174 },
   {  "MQCACF_APPL_FUNCTION"            ,       3200 },
   {  "MQCACF_APPL_IDENTITY_DATA"       ,       3135 },
   {  "MQCACF_APPL_NAME"                ,       3024 },
   {  "MQCACF_APPL_ORIGIN_DATA"         ,       3136 },
   {  "MQCACF_APPL_TAG"                 ,       3058 },
   {  "MQCACF_ARCHIVE_LOG_EXTENT_NAME"  ,       3208 },
   {  "MQCACF_ASID"                     ,       3081 },
   {  "MQCACF_AUTH_INFO_NAMES"          ,       3048 },
   {  "MQCACF_AUTH_PROFILE_NAME"        ,       3067 },
   {  "MQCACF_AUX_ERROR_DATA_STR_1"     ,       3026 },
   {  "MQCACF_AUX_ERROR_DATA_STR_2"     ,       3027 },
   {  "MQCACF_AUX_ERROR_DATA_STR_3"     ,       3028 },
   {  "MQCACF_BACKUP_DATE"              ,       3098 },
   {  "MQCACF_BACKUP_TIME"              ,       3099 },
   {  "MQCACF_BRIDGE_NAME"              ,       3029 },
   {  "MQCACF_CF_OFFLOAD_SIZE1"         ,       3179 },
   {  "MQCACF_CF_OFFLOAD_SIZE2"         ,       3180 },
   {  "MQCACF_CF_OFFLOAD_SIZE3"         ,       3181 },
   {  "MQCACF_CF_SMDS"                  ,       3183 },
   {  "MQCACF_CF_SMDSCONN"              ,       3186 },
   {  "MQCACF_CF_SMDS_GENERIC_NAME"     ,       3182 },
   {  "MQCACF_CF_STRUC_BACKUP_END"      ,       3102 },
   {  "MQCACF_CF_STRUC_BACKUP_START"    ,       3101 },
   {  "MQCACF_CF_STRUC_LOG_Q_MGRS"      ,       3103 },
   {  "MQCACF_CF_STRUC_NAME"            ,       3187 },
   {  "MQCACF_CF_STRUC_NAMES"           ,       3095 },
   {  "MQCACF_CHAR_ATTRS"               ,       3189 },
   {  "MQCACF_CHILD_Q_MGR_NAME"         ,       3040 },
   {  "MQCACF_CLUS_CHAN_Q_MGR_NAME"     ,       5507 },
   {  "MQCACF_CLUS_SHORT_CONN_NAME"     ,       5508 },
   {  "MQCACF_COMMAND_MQSC"             ,       3075 },
   {  "MQCACF_COMMAND_SCOPE"            ,       3080 },
   {  "MQCACF_CONFIGURATION_DATE"       ,       3091 },
   {  "MQCACF_CONFIGURATION_TIME"       ,       3092 },
   {  "MQCACF_CORREL_ID"                ,       3033 },
   {  "MQCACF_CSP_USER_IDENTIFIER"      ,       3206 },
   {  "MQCACF_CURRENT_LOG_EXTENT_NAME"  ,       3071 },
   {  "MQCACF_DATA_SET_NAME"            ,       3059 },
   {  "MQCACF_DB2_NAME"                 ,       3109 },
   {  "MQCACF_DESTINATION"              ,       3154 },
   {  "MQCACF_DESTINATION_Q_MGR"        ,       3153 },
   {  "MQCACF_DSG_NAME"                 ,       3108 },
   {  "MQCACF_DYNAMIC_Q_NAME"           ,       3190 },
   {  "MQCACF_ENTITY_NAME"              ,       3068 },
   {  "MQCACF_ENV_INFO"                 ,       3089 },
   {  "MQCACF_ESCAPE_TEXT"              ,       3014 },
   {  "MQCACF_EVENT_APPL_IDENTITY"      ,       3049 },
   {  "MQCACF_EVENT_APPL_NAME"          ,       3050 },
   {  "MQCACF_EVENT_APPL_ORIGIN"        ,       3051 },
   {  "MQCACF_EVENT_Q_MGR"              ,       3047 },
   {  "MQCACF_EVENT_USER_ID"            ,       3045 },
   {  "MQCACF_EXCL_OPERATOR_MESSAGES"   ,       3205 },
   {  "MQCACF_FAIL_DATE"                ,       3096 },
   {  "MQCACF_FAIL_TIME"                ,       3097 },
   {  "MQCACF_FILTER"                   ,       3170 },
   {  "MQCACF_FIRST"                    ,       3001 },
   {  "MQCACF_FROM_AUTH_INFO_NAME"      ,       3009 },
   {  "MQCACF_FROM_CF_STRUC_NAME"       ,       3093 },
   {  "MQCACF_FROM_CHANNEL_NAME"        ,       3007 },
   {  "MQCACF_FROM_COMM_INFO_NAME"      ,       3177 },
   {  "MQCACF_FROM_LISTENER_NAME"       ,       3124 },
   {  "MQCACF_FROM_NAMELIST_NAME"       ,       3005 },
   {  "MQCACF_FROM_PROCESS_NAME"        ,       3003 },
   {  "MQCACF_FROM_Q_NAME"              ,       3001 },
   {  "MQCACF_FROM_SERVICE_NAME"        ,       3126 },
   {  "MQCACF_FROM_STORAGE_CLASS"       ,       3104 },
   {  "MQCACF_FROM_SUB_NAME"            ,       3163 },
   {  "MQCACF_FROM_TOPIC_NAME"          ,       3150 },
   {  "MQCACF_GROUP_ENTITY_NAMES"       ,       3066 },
   {  "MQCACF_HOST_NAME"                ,       3191 },
   {  "MQCACF_LAST_GET_DATE"            ,       3130 },
   {  "MQCACF_LAST_GET_TIME"            ,       3131 },
   {  "MQCACF_LAST_MSG_DATE"            ,       3168 },
   {  "MQCACF_LAST_MSG_TIME"            ,       3167 },
   {  "MQCACF_LAST_PUB_DATE"            ,       3161 },
   {  "MQCACF_LAST_PUB_TIME"            ,       3162 },
   {  "MQCACF_LAST_PUT_DATE"            ,       3128 },
   {  "MQCACF_LAST_PUT_TIME"            ,       3129 },
   {  "MQCACF_LAST_USED"                ,       3208 },
   {  "MQCACF_LOCAL_Q_NAMES"            ,       3015 },
   {  "MQCACF_LOG_PATH"                 ,       3074 },
   {  "MQCACF_MEDIA_LOG_EXTENT_NAME"    ,       3073 },
   {  "MQCACF_MODEL_Q_NAMES"            ,       3016 },
   {  "MQCACF_MQCB_NAME"                ,       3192 },
   {  "MQCACF_NAMELIST_NAMES"           ,       3013 },
   {  "MQCACF_NONE"                     ,       3171 },
   {  "MQCACF_OBJECT_NAME"              ,       3046 },
   {  "MQCACF_OBJECT_Q_MGR_NAME"        ,       3023 },
   {  "MQCACF_OBJECT_STRING"            ,       3193 },
   {  "MQCACF_OPERATION_DATE"           ,       3132 },
   {  "MQCACF_OPERATION_TIME"           ,       3133 },
   {  "MQCACF_ORIGIN_NAME"              ,       3088 },
   {  "MQCACF_PARENT_Q_MGR_NAME"        ,       3032 },
   {  "MQCACF_PRINCIPAL_ENTITY_NAMES"   ,       3065 },
   {  "MQCACF_PROCESS_NAMES"            ,       3012 },
   {  "MQCACF_PSB_NAME"                 ,       3082 },
   {  "MQCACF_PST_ID"                   ,       3083 },
   {  "MQCACF_PUBLISH_TIMESTAMP"        ,       3034 },
   {  "MQCACF_PUT_DATE"                 ,       3137 },
   {  "MQCACF_PUT_TIME"                 ,       3138 },
   {  "MQCACF_Q_MGR_CPF"                ,       3076 },
   {  "MQCACF_Q_MGR_START_DATE"         ,       3175 },
   {  "MQCACF_Q_MGR_START_TIME"         ,       3176 },
   {  "MQCACF_Q_MGR_UOW_ID"             ,       3086 },
   {  "MQCACF_Q_NAMES"                  ,       3011 },
   {  "MQCACF_RECEIVER_CHANNEL_NAMES"   ,       3022 },
   {  "MQCACF_RECOVERY_DATE"            ,       3184 },
   {  "MQCACF_RECOVERY_TIME"            ,       3185 },
   {  "MQCACF_REG_CORREL_ID"            ,       3044 },
   {  "MQCACF_REG_Q_MGR_NAME"           ,       3042 },
   {  "MQCACF_REG_Q_NAME"               ,       3043 },
   {  "MQCACF_REG_STREAM_NAME"          ,       3041 },
   {  "MQCACF_REG_SUB_IDENTITY"         ,       3055 },
   {  "MQCACF_REG_SUB_NAME"             ,       3053 },
   {  "MQCACF_REG_SUB_USER_DATA"        ,       3057 },
   {  "MQCACF_REG_TIME"                 ,       3038 },
   {  "MQCACF_REG_TOPIC"                ,       3037 },
   {  "MQCACF_REG_USER_ID"              ,       3039 },
   {  "MQCACF_REMOTE_Q_NAMES"           ,       3018 },
   {  "MQCACF_REPLY_TO_Q"               ,       3139 },
   {  "MQCACF_REPLY_TO_Q_MGR"           ,       3140 },
   {  "MQCACF_REQUESTER_CHANNEL_NAMES"  ,       3021 },
   {  "MQCACF_RESOLVED_LOCAL_Q_MGR"     ,       3194 },
   {  "MQCACF_RESOLVED_LOCAL_Q_NAME"    ,       3195 },
   {  "MQCACF_RESOLVED_OBJECT_STRING"   ,       3196 },
   {  "MQCACF_RESOLVED_Q_MGR"           ,       3197 },
   {  "MQCACF_RESOLVED_Q_NAME"          ,       3141 },
   {  "MQCACF_RESPONSE_Q_MGR_NAME"      ,       3070 },
   {  "MQCACF_RESTART_LOG_EXTENT_NAME"  ,       3072 },
   {  "MQCACF_ROUTING_FINGER_PRINT"     ,       3173 },
   {  "MQCACF_SECURITY_PROFILE"         ,       3090 },
   {  "MQCACF_SELECTION_STRING"         ,       3198 },
   {  "MQCACF_SENDER_CHANNEL_NAMES"     ,       3019 },
   {  "MQCACF_SERVER_CHANNEL_NAMES"     ,       3020 },
   {  "MQCACF_SERVICE_COMPONENT"        ,       3069 },
   {  "MQCACF_SERVICE_START_DATE"       ,       3144 },
   {  "MQCACF_SERVICE_START_TIME"       ,       3145 },
   {  "MQCACF_STORAGE_CLASS_NAMES"      ,       3106 },
   {  "MQCACF_STREAM_NAME"              ,       3030 },
   {  "MQCACF_STRING_DATA"              ,       3035 },
   {  "MQCACF_STRUC_ID"                 ,       3142 },
   {  "MQCACF_SUBSCRIPTION_IDENTITY"    ,       3054 },
   {  "MQCACF_SUBSCRIPTION_NAME"        ,       3052 },
   {  "MQCACF_SUBSCRIPTION_POINT"       ,       3169 },
   {  "MQCACF_SUBSCRIPTION_USER_DATA"   ,       3056 },
   {  "MQCACF_SUB_NAME"                 ,       3152 },
   {  "MQCACF_SUB_SELECTOR"             ,       3160 },
   {  "MQCACF_SUB_USER_DATA"            ,       3159 },
   {  "MQCACF_SUB_USER_ID"              ,       3156 },
   {  "MQCACF_SUPPORTED_STREAM_NAME"    ,       3036 },
   {  "MQCACF_SYSP_ARCHIVE_PFX1"        ,       3115 },
   {  "MQCACF_SYSP_ARCHIVE_PFX2"        ,       3147 },
   {  "MQCACF_SYSP_ARCHIVE_UNIT1"       ,       3116 },
   {  "MQCACF_SYSP_ARCHIVE_UNIT2"       ,       3148 },
   {  "MQCACF_SYSP_CMD_USER_ID"         ,       3110 },
   {  "MQCACF_SYSP_LOG_CORREL_ID"       ,       3117 },
   {  "MQCACF_SYSP_LOG_RBA"             ,       3122 },
   {  "MQCACF_SYSP_OFFLINE_RBA"         ,       3146 },
   {  "MQCACF_SYSP_OTMA_DRU_EXIT"       ,       3113 },
   {  "MQCACF_SYSP_OTMA_GROUP"          ,       3111 },
   {  "MQCACF_SYSP_OTMA_MEMBER"         ,       3112 },
   {  "MQCACF_SYSP_OTMA_TPIPE_PFX"      ,       3114 },
   {  "MQCACF_SYSP_Q_MGR_DATE"          ,       3120 },
   {  "MQCACF_SYSP_Q_MGR_RBA"           ,       3121 },
   {  "MQCACF_SYSP_Q_MGR_TIME"          ,       3119 },
   {  "MQCACF_SYSP_SERVICE"             ,       3123 },
   {  "MQCACF_SYSP_UNIT_VOLSER"         ,       3118 },
   {  "MQCACF_SYSTEM_NAME"              ,       3100 },
   {  "MQCACF_TASK_NUMBER"              ,       3084 },
   {  "MQCACF_TOPIC"                    ,       3031 },
   {  "MQCACF_TOPIC_NAMES"              ,       3151 },
   {  "MQCACF_TO_AUTH_INFO_NAME"        ,       3010 },
   {  "MQCACF_TO_CF_STRUC_NAME"         ,       3094 },
   {  "MQCACF_TO_CHANNEL_NAME"          ,       3008 },
   {  "MQCACF_TO_COMM_INFO_NAME"        ,       3178 },
   {  "MQCACF_TO_LISTENER_NAME"         ,       3125 },
   {  "MQCACF_TO_NAMELIST_NAME"         ,       3006 },
   {  "MQCACF_TO_PROCESS_NAME"          ,       3004 },
   {  "MQCACF_TO_Q_NAME"                ,       3002 },
   {  "MQCACF_TO_SERVICE_NAME"          ,       3127 },
   {  "MQCACF_TO_STORAGE_CLASS"         ,       3105 },
   {  "MQCACF_TO_SUB_NAME"              ,       3164 },
   {  "MQCACF_TO_TOPIC_NAME"            ,       3149 },
   {  "MQCACF_TRANSACTION_ID"           ,       3085 },
   {  "MQCACF_UOW_LOG_EXTENT_NAME"      ,       3064 },
   {  "MQCACF_UOW_LOG_START_DATE"       ,       3062 },
   {  "MQCACF_UOW_LOG_START_TIME"       ,       3063 },
   {  "MQCACF_UOW_START_DATE"           ,       3060 },
   {  "MQCACF_UOW_START_TIME"           ,       3061 },
   {  "MQCACF_USAGE_LOG_LRSN"           ,       3079 },
   {  "MQCACF_USAGE_LOG_RBA"            ,       3078 },
   {  "MQCACF_USER_IDENTIFIER"          ,       3025 },
   {  "MQCACF_VALUE_NAME"               ,       3143 },
   {  "MQCACF_XA_INFO"                  ,       3199 },
   {  "MQCACF_XQH_PUT_DATE"             ,       3204 },
   {  "MQCACF_XQH_PUT_TIME"             ,       3203 },
   {  "MQCACF_XQH_REMOTE_Q_MGR"         ,       3202 },
   {  "MQCACF_XQH_REMOTE_Q_NAME"        ,       3201 },
   {  "MQCACH_CHANNEL_NAME"             ,       3501 },
   {  "MQCACH_CHANNEL_NAMES"            ,       3512 },
   {  "MQCACH_CHANNEL_START_DATE"       ,       3529 },
   {  "MQCACH_CHANNEL_START_TIME"       ,       3528 },
   {  "MQCACH_CLIENT_ID"                ,       3564 },
   {  "MQCACH_CLIENT_USER_ID"           ,       3567 },
   {  "MQCACH_CONNECTION_NAME"          ,       3506 },
   {  "MQCACH_CONNECTION_NAME_LIST"     ,       3566 },
   {  "MQCACH_CURRENT_LUWID"            ,       3532 },
   {  "MQCACH_DESC"                     ,       3502 },
   {  "MQCACH_FIRST"                    ,       3501 },
   {  "MQCACH_FORMAT_NAME"              ,       3533 },
   {  "MQCACH_GROUP_ADDRESS"            ,       3562 },
   {  "MQCACH_IP_ADDRESS"               ,       3552 },
   {  "MQCACH_JAAS_CONFIG"              ,       3563 },
   {  "MQCACH_LAST_LUWID"               ,       3531 },
   {  "MQCACH_LAST_MSG_DATE"            ,       3525 },
   {  "MQCACH_LAST_MSG_TIME"            ,       3524 },
   {  "MQCACH_LAST_USED"                ,       3571 },
   {  "MQCACH_LISTENER_DESC"            ,       3555 },
   {  "MQCACH_LISTENER_NAME"            ,       3554 },
   {  "MQCACH_LISTENER_START_DATE"      ,       3556 },
   {  "MQCACH_LISTENER_START_TIME"      ,       3557 },
   {  "MQCACH_LOCAL_ADDRESS"            ,       3520 },
   {  "MQCACH_LOCAL_NAME"               ,       3521 },
   {  "MQCACH_LU_NAME"                  ,       3551 },
   {  "MQCACH_MCA_JOB_NAME"             ,       3530 },
   {  "MQCACH_MCA_NAME"                 ,       3507 },
   {  "MQCACH_MCA_USER_ID"              ,       3527 },
   {  "MQCACH_MCA_USER_ID_LIST"         ,       3568 },
   {  "MQCACH_MODE_NAME"                ,       3503 },
   {  "MQCACH_MR_EXIT_NAME"             ,       3534 },
   {  "MQCACH_MR_EXIT_USER_DATA"        ,       3535 },
   {  "MQCACH_MSG_EXIT_NAME"            ,       3509 },
   {  "MQCACH_MSG_EXIT_USER_DATA"       ,       3514 },
   {  "MQCACH_PASSWORD"                 ,       3518 },
   {  "MQCACH_RCV_EXIT_NAME"            ,       3511 },
   {  "MQCACH_RCV_EXIT_USER_DATA"       ,       3516 },
   {  "MQCACH_REMOTE_APPL_TAG"          ,       3548 },
   {  "MQCACH_REMOTE_PRODUCT"           ,       3561 },
   {  "MQCACH_REMOTE_VERSION"           ,       3560 },
   {  "MQCACH_SEC_EXIT_NAME"            ,       3508 },
   {  "MQCACH_SEC_EXIT_USER_DATA"       ,       3513 },
   {  "MQCACH_SEND_EXIT_NAME"           ,       3510 },
   {  "MQCACH_SEND_EXIT_USER_DATA"      ,       3515 },
   {  "MQCACH_SSL_CERT_ISSUER_NAME"     ,       3550 },
   {  "MQCACH_SSL_CERT_USER_ID"         ,       3549 },
   {  "MQCACH_SSL_CIPHER_SPEC"          ,       3544 },
   {  "MQCACH_SSL_CIPHER_SUITE"         ,       3569 },
   {  "MQCACH_SSL_HANDSHAKE_STAGE"      ,       3546 },
   {  "MQCACH_SSL_KEY_PASSPHRASE"       ,       3565 },
   {  "MQCACH_SSL_KEY_RESET_DATE"       ,       3558 },
   {  "MQCACH_SSL_KEY_RESET_TIME"       ,       3559 },
   {  "MQCACH_SSL_PEER_NAME"            ,       3545 },
   {  "MQCACH_SSL_SHORT_PEER_NAME"      ,       3547 },
   {  "MQCACH_TCP_NAME"                 ,       3553 },
   {  "MQCACH_TOPIC_ROOT"               ,       3571 },
   {  "MQCACH_TP_NAME"                  ,       3504 },
   {  "MQCACH_USER_ID"                  ,       3517 },
   {  "MQCACH_WEBCONTENT_PATH"          ,       3570 },
   {  "MQCACH_XMIT_Q_NAME"              ,       3505 },
   {  "MQCADSD_MSGFORMAT"               ,        256 },
   {  "MQCADSD_NONE"                    ,          0 },
   {  "MQCADSD_RECV"                    ,         16 },
   {  "MQCADSD_SEND"                    ,          1 },
   {  "MQCAFTY_NONE"                    ,          0 },
   {  "MQCAFTY_PREFERRED"               ,          1 },
   {  "MQCAMO_CLOSE_DATE"               ,       2701 },
   {  "MQCAMO_CLOSE_TIME"               ,       2702 },
   {  "MQCAMO_CONN_DATE"                ,       2703 },
   {  "MQCAMO_CONN_TIME"                ,       2704 },
   {  "MQCAMO_DISC_DATE"                ,       2705 },
   {  "MQCAMO_DISC_TIME"                ,       2706 },
   {  "MQCAMO_END_DATE"                 ,       2707 },
   {  "MQCAMO_END_TIME"                 ,       2708 },
   {  "MQCAMO_FIRST"                    ,       2701 },
   {  "MQCAMO_LAST_USED"                ,       2715 },
   {  "MQCAMO_MONITOR_CLASS"            ,       2713 },
   {  "MQCAMO_MONITOR_DESC"             ,       2715 },
   {  "MQCAMO_MONITOR_TYPE"             ,       2714 },
   {  "MQCAMO_OPEN_DATE"                ,       2709 },
   {  "MQCAMO_OPEN_TIME"                ,       2710 },
   {  "MQCAMO_START_DATE"               ,       2711 },
   {  "MQCAMO_START_TIME"               ,       2712 },
   {  "MQCAP_EXPIRED"                   ,          2 },
   {  "MQCAP_NOT_SUPPORTED"             ,          0 },
   {  "MQCAP_SUPPORTED"                 ,          1 },
   {  "MQCAUT_ADDRESSMAP"               ,          4 },
   {  "MQCAUT_ALL"                      ,          0 },
   {  "MQCAUT_BLOCKADDR"                ,          2 },
   {  "MQCAUT_BLOCKUSER"                ,          1 },
   {  "MQCAUT_QMGRMAP"                  ,          6 },
   {  "MQCAUT_SSLPEERMAP"               ,          3 },
   {  "MQCAUT_USERMAP"                  ,          5 },
   {  "MQCA_ADMIN_TOPIC_NAME"           ,       2105 },
   {  "MQCA_ALTERATION_DATE"            ,       2027 },
   {  "MQCA_ALTERATION_TIME"            ,       2028 },
   {  "MQCA_AMQP_SSL_CIPHER_SUITES"     ,       2137 },
   {  "MQCA_AMQP_VERSION"               ,       2136 },
   {  "MQCA_APPL_ID"                    ,       2001 },
   {  "MQCA_AUTH_INFO_CONN_NAME"        ,       2053 },
   {  "MQCA_AUTH_INFO_DESC"             ,       2046 },
   {  "MQCA_AUTH_INFO_NAME"             ,       2045 },
   {  "MQCA_AUTH_INFO_OCSP_URL"         ,       2109 },
   {  "MQCA_AUTO_REORG_CATALOG"         ,       2091 },
   {  "MQCA_AUTO_REORG_START_TIME"      ,       2090 },
   {  "MQCA_BACKOUT_REQ_Q_NAME"         ,       2019 },
   {  "MQCA_BASE_OBJECT_NAME"           ,       2002 },
   {  "MQCA_BASE_Q_NAME"                ,       2002 },
   {  "MQCA_BATCH_INTERFACE_ID"         ,       2068 },
   {  "MQCA_CERT_LABEL"                 ,       2121 },
   {  "MQCA_CF_STRUC_DESC"              ,       2052 },
   {  "MQCA_CF_STRUC_NAME"              ,       2039 },
   {  "MQCA_CHANNEL_AUTO_DEF_EXIT"      ,       2026 },
   {  "MQCA_CHILD"                      ,       2101 },
   {  "MQCA_CHINIT_SERVICE_PARM"        ,       2076 },
   {  "MQCA_CHLAUTH_DESC"               ,       2118 },
   {  "MQCA_CICS_FILE_NAME"             ,       2060 },
   {  "MQCA_CLUSTER_DATE"               ,       2037 },
   {  "MQCA_CLUSTER_NAME"               ,       2029 },
   {  "MQCA_CLUSTER_NAMELIST"           ,       2030 },
   {  "MQCA_CLUSTER_Q_MGR_NAME"         ,       2031 },
   {  "MQCA_CLUSTER_TIME"               ,       2038 },
   {  "MQCA_CLUSTER_WORKLOAD_DATA"      ,       2034 },
   {  "MQCA_CLUSTER_WORKLOAD_EXIT"      ,       2033 },
   {  "MQCA_CLUS_CHL_NAME"              ,       2124 },
   {  "MQCA_COMMAND_INPUT_Q_NAME"       ,       2003 },
   {  "MQCA_COMMAND_REPLY_Q_NAME"       ,       2067 },
   {  "MQCA_COMM_INFO_DESC"             ,       2111 },
   {  "MQCA_COMM_INFO_NAME"             ,       2110 },
   {  "MQCA_CONN_AUTH"                  ,       2125 },
   {  "MQCA_CREATION_DATE"              ,       2004 },
   {  "MQCA_CREATION_TIME"              ,       2005 },
   {  "MQCA_CUSTOM"                     ,       2119 },
   {  "MQCA_DEAD_LETTER_Q_NAME"         ,       2006 },
   {  "MQCA_DEF_XMIT_Q_NAME"            ,       2025 },
   {  "MQCA_DNS_GROUP"                  ,       2071 },
   {  "MQCA_ENV_DATA"                   ,       2007 },
   {  "MQCA_FIRST"                      ,       2001 },
   {  "MQCA_IGQ_USER_ID"                ,       2041 },
   {  "MQCA_INITIATION_Q_NAME"          ,       2008 },
   {  "MQCA_INSTALLATION_DESC"          ,       2115 },
   {  "MQCA_INSTALLATION_NAME"          ,       2116 },
   {  "MQCA_INSTALLATION_PATH"          ,       2117 },
   {  "MQCA_LAST"                       ,       4000 },
   {  "MQCA_LAST_USED"                  ,       2137 },
   {  "MQCA_LDAP_BASE_DN_GROUPS"        ,       2132 },
   {  "MQCA_LDAP_BASE_DN_USERS"         ,       2126 },
   {  "MQCA_LDAP_FIND_GROUP_FIELD"      ,       2135 },
   {  "MQCA_LDAP_GROUP_ATTR_FIELD"      ,       2134 },
   {  "MQCA_LDAP_GROUP_OBJECT_CLASS"    ,       2133 },
   {  "MQCA_LDAP_PASSWORD"              ,       2048 },
   {  "MQCA_LDAP_SHORT_USER_FIELD"      ,       2127 },
   {  "MQCA_LDAP_USER_ATTR_FIELD"       ,       2129 },
   {  "MQCA_LDAP_USER_NAME"             ,       2047 },
   {  "MQCA_LDAP_USER_OBJECT_CLASS"     ,       2128 },
   {  "MQCA_LU62_ARM_SUFFIX"            ,       2074 },
   {  "MQCA_LU_GROUP_NAME"              ,       2072 },
   {  "MQCA_LU_NAME"                    ,       2073 },
   {  "MQCA_MODEL_DURABLE_Q"            ,       2096 },
   {  "MQCA_MODEL_NON_DURABLE_Q"        ,       2097 },
   {  "MQCA_MONITOR_Q_NAME"             ,       2066 },
   {  "MQCA_NAMELIST_DESC"              ,       2009 },
   {  "MQCA_NAMELIST_NAME"              ,       2010 },
   {  "MQCA_NAMES"                      ,       2020 },
   {  "MQCA_PARENT"                     ,       2102 },
   {  "MQCA_PASS_TICKET_APPL"           ,       2086 },
   {  "MQCA_POLICY_NAME"                ,       2112 },
   {  "MQCA_PROCESS_DESC"               ,       2011 },
   {  "MQCA_PROCESS_NAME"               ,       2012 },
   {  "MQCA_QSG_CERT_LABEL"             ,       2131 },
   {  "MQCA_QSG_NAME"                   ,       2040 },
   {  "MQCA_Q_DESC"                     ,       2013 },
   {  "MQCA_Q_MGR_DESC"                 ,       2014 },
   {  "MQCA_Q_MGR_IDENTIFIER"           ,       2032 },
   {  "MQCA_Q_MGR_NAME"                 ,       2015 },
   {  "MQCA_Q_NAME"                     ,       2016 },
   {  "MQCA_RECIPIENT_DN"               ,       2114 },
   {  "MQCA_REMOTE_Q_MGR_NAME"          ,       2017 },
   {  "MQCA_REMOTE_Q_NAME"              ,       2018 },
   {  "MQCA_REPOSITORY_NAME"            ,       2035 },
   {  "MQCA_REPOSITORY_NAMELIST"        ,       2036 },
   {  "MQCA_RESUME_DATE"                ,       2098 },
   {  "MQCA_RESUME_TIME"                ,       2099 },
   {  "MQCA_SERVICE_DESC"               ,       2078 },
   {  "MQCA_SERVICE_NAME"               ,       2077 },
   {  "MQCA_SERVICE_START_ARGS"         ,       2080 },
   {  "MQCA_SERVICE_START_COMMAND"      ,       2079 },
   {  "MQCA_SERVICE_STOP_ARGS"          ,       2082 },
   {  "MQCA_SERVICE_STOP_COMMAND"       ,       2081 },
   {  "MQCA_SIGNER_DN"                  ,       2113 },
   {  "MQCA_SSL_CERT_ISSUER_NAME"       ,       2130 },
   {  "MQCA_SSL_CRL_NAMELIST"           ,       2050 },
   {  "MQCA_SSL_CRYPTO_HARDWARE"        ,       2051 },
   {  "MQCA_SSL_KEY_LIBRARY"            ,       2069 },
   {  "MQCA_SSL_KEY_MEMBER"             ,       2070 },
   {  "MQCA_SSL_KEY_REPOSITORY"         ,       2049 },
   {  "MQCA_STDERR_DESTINATION"         ,       2084 },
   {  "MQCA_STDOUT_DESTINATION"         ,       2083 },
   {  "MQCA_STORAGE_CLASS"              ,       2022 },
   {  "MQCA_STORAGE_CLASS_DESC"         ,       2042 },
   {  "MQCA_SYSTEM_LOG_Q_NAME"          ,       2065 },
   {  "MQCA_TCP_NAME"                   ,       2075 },
   {  "MQCA_TOPIC_DESC"                 ,       2093 },
   {  "MQCA_TOPIC_NAME"                 ,       2092 },
   {  "MQCA_TOPIC_STRING"               ,       2094 },
   {  "MQCA_TOPIC_STRING_FILTER"        ,       2108 },
   {  "MQCA_TPIPE_NAME"                 ,       2085 },
   {  "MQCA_TRIGGER_CHANNEL_NAME"       ,       2064 },
   {  "MQCA_TRIGGER_DATA"               ,       2023 },
   {  "MQCA_TRIGGER_PROGRAM_NAME"       ,       2062 },
   {  "MQCA_TRIGGER_TERM_ID"            ,       2063 },
   {  "MQCA_TRIGGER_TRANS_ID"           ,       2061 },
   {  "MQCA_USER_DATA"                  ,       2021 },
   {  "MQCA_USER_LIST"                  ,       4000 },
   {  "MQCA_VERSION"                    ,       2120 },
   {  "MQCA_XCF_GROUP_NAME"             ,       2043 },
   {  "MQCA_XCF_MEMBER_NAME"            ,       2044 },
   {  "MQCA_XMIT_Q_NAME"                ,       2024 },
   {  "MQCA_XR_SSL_CIPHER_SUITES"       ,       2123 },
   {  "MQCA_XR_VERSION"                 ,       2122 },
   {  "MQCBCF_NONE"                     ,          0 },
   {  "MQCBCF_READA_BUFFER_EMPTY"       ,          1 },
   {  "MQCBCT_DEREGISTER_CALL"          ,          4 },
   {  "MQCBCT_EVENT_CALL"               ,          5 },
   {  "MQCBCT_MC_EVENT_CALL"            ,          8 },
   {  "MQCBCT_MSG_NOT_REMOVED"          ,          7 },
   {  "MQCBCT_MSG_REMOVED"              ,          6 },
   {  "MQCBCT_REGISTER_CALL"            ,          3 },
   {  "MQCBCT_START_CALL"               ,          1 },
   {  "MQCBCT_STOP_CALL"                ,          2 },
   {  "MQCBC_CURRENT_LENGTH (4 byte)"   ,         52 },
   {  "MQCBC_CURRENT_LENGTH (8 byte)"   ,         64 },
   {  "MQCBC_CURRENT_VERSION"           ,          2 },
   {  "MQCBC_LENGTH_1 (4 byte)"         ,         48 },
   {  "MQCBC_LENGTH_1 (8 byte)"         ,         56 },
   {  "MQCBC_LENGTH_2 (4 byte)"         ,         52 },
   {  "MQCBC_LENGTH_2 (8 byte)"         ,         64 },
   {  "MQCBC_VERSION_1"                 ,          1 },
   {  "MQCBC_VERSION_2"                 ,          2 },
   {  "MQCBDO_DEREGISTER_CALL"          ,        512 },
   {  "MQCBDO_EVENT_CALL"               ,      16384 },
   {  "MQCBDO_FAIL_IF_QUIESCING"        ,       8192 },
   {  "MQCBDO_MC_EVENT_CALL"            ,      32768 },
   {  "MQCBDO_NONE"                     ,          0 },
   {  "MQCBDO_REGISTER_CALL"            ,        256 },
   {  "MQCBDO_START_CALL"               ,          1 },
   {  "MQCBDO_STOP_CALL"                ,          4 },
   {  "MQCBD_CURRENT_LENGTH (4 byte)"   ,        156 },
   {  "MQCBD_CURRENT_LENGTH (8 byte)"   ,        168 },
   {  "MQCBD_CURRENT_VERSION"           ,          1 },
   {  "MQCBD_FULL_MSG_LENGTH"           ,         -1 },
   {  "MQCBD_LENGTH_1 (4 byte)"         ,        156 },
   {  "MQCBD_LENGTH_1 (8 byte)"         ,        168 },
   {  "MQCBD_VERSION_1"                 ,          1 },
   {  "MQCBO_ADMIN_BAG"                 ,          1 },
   {  "MQCBO_CHECK_SELECTORS"           ,          8 },
   {  "MQCBO_COMMAND_BAG"               ,         16 },
   {  "MQCBO_DO_NOT_CHECK_SELECTORS"    ,          0 },
   {  "MQCBO_DO_NOT_REORDER"            ,          0 },
   {  "MQCBO_GROUP_BAG"                 ,         64 },
   {  "MQCBO_LIST_FORM_ALLOWED"         ,          2 },
   {  "MQCBO_LIST_FORM_INHIBITED"       ,          0 },
   {  "MQCBO_NONE"                      ,          0 },
   {  "MQCBO_REORDER_AS_REQUIRED"       ,          4 },
   {  "MQCBO_SYSTEM_BAG"                ,         32 },
   {  "MQCBO_USER_BAG"                  ,          0 },
   {  "MQCBT_EVENT_HANDLER"             ,          2 },
   {  "MQCBT_MESSAGE_CONSUMER"          ,          1 },
   {  "MQCCSI_APPL"                     ,         -3 },
   {  "MQCCSI_AS_PUBLISHED"             ,         -4 },
   {  "MQCCSI_DEFAULT"                  ,          0 },
   {  "MQCCSI_EMBEDDED"                 ,         -1 },
   {  "MQCCSI_INHERIT"                  ,         -2 },
   {  "MQCCSI_Q_MGR"                    ,          0 },
   {  "MQCCSI_UNDEFINED"                ,          0 },
   {  "MQCCT_NO"                        ,          0 },
   {  "MQCCT_YES"                       ,          1 },
   {  "MQCC_FAILED"                     ,          2 },
   {  "MQCC_OK"                         ,          0 },
   {  "MQCC_UNKNOWN"                    ,         -1 },
   {  "MQCC_WARNING"                    ,          1 },
   {  "MQCDC_CURRENT_LENGTH (4 byte)"   ,       1940 },
   {  "MQCDC_CURRENT_LENGTH (8 byte)"   ,       1984 },
   {  "MQCDC_CURRENT_VERSION"           ,         11 },
   {  "MQCDC_LENGTH_1"                  ,        984 },
   {  "MQCDC_LENGTH_10 (4 byte)"        ,       1876 },
   {  "MQCDC_LENGTH_10 (8 byte)"        ,       1920 },
   {  "MQCDC_LENGTH_11 (4 byte)"        ,       1940 },
   {  "MQCDC_LENGTH_11 (8 byte)"        ,       1984 },
   {  "MQCDC_LENGTH_2"                  ,       1312 },
   {  "MQCDC_LENGTH_3"                  ,       1480 },
   {  "MQCDC_LENGTH_4 (4 byte)"         ,       1540 },
   {  "MQCDC_LENGTH_4 (8 byte)"         ,       1568 },
   {  "MQCDC_LENGTH_5 (4 byte)"         ,       1552 },
   {  "MQCDC_LENGTH_5 (8 byte)"         ,       1584 },
   {  "MQCDC_LENGTH_6 (4 byte)"         ,       1648 },
   {  "MQCDC_LENGTH_6 (8 byte)"         ,       1688 },
   {  "MQCDC_LENGTH_7 (4 byte)"         ,       1748 },
   {  "MQCDC_LENGTH_7 (8 byte)"         ,       1792 },
   {  "MQCDC_LENGTH_8 (4 byte)"         ,       1840 },
   {  "MQCDC_LENGTH_8 (8 byte)"         ,       1888 },
   {  "MQCDC_LENGTH_9 (4 byte)"         ,       1864 },
   {  "MQCDC_LENGTH_9 (8 byte)"         ,       1912 },
   {  "MQCDC_NO_SENDER_CONVERSION"      ,          0 },
   {  "MQCDC_SENDER_CONVERSION"         ,          1 },
   {  "MQCDC_VERSION_1"                 ,          1 },
   {  "MQCDC_VERSION_10"                ,         10 },
   {  "MQCDC_VERSION_11"                ,         11 },
   {  "MQCDC_VERSION_2"                 ,          2 },
   {  "MQCDC_VERSION_3"                 ,          3 },
   {  "MQCDC_VERSION_4"                 ,          4 },
   {  "MQCDC_VERSION_5"                 ,          5 },
   {  "MQCDC_VERSION_6"                 ,          6 },
   {  "MQCDC_VERSION_7"                 ,          7 },
   {  "MQCDC_VERSION_8"                 ,          8 },
   {  "MQCDC_VERSION_9"                 ,          9 },
   {  "MQCD_CURRENT_LENGTH (4 byte)"    ,       1940 },
   {  "MQCD_CURRENT_LENGTH (8 byte)"    ,       1984 },
   {  "MQCD_CURRENT_VERSION"            ,         11 },
   {  "MQCD_LENGTH_1"                   ,        984 },
   {  "MQCD_LENGTH_10 (4 byte)"         ,       1876 },
   {  "MQCD_LENGTH_10 (8 byte)"         ,       1920 },
   {  "MQCD_LENGTH_11 (4 byte)"         ,       1940 },
   {  "MQCD_LENGTH_11 (8 byte)"         ,       1984 },
   {  "MQCD_LENGTH_2"                   ,       1312 },
   {  "MQCD_LENGTH_3"                   ,       1480 },
   {  "MQCD_LENGTH_4 (4 byte)"          ,       1540 },
   {  "MQCD_LENGTH_4 (8 byte)"          ,       1568 },
   {  "MQCD_LENGTH_5 (4 byte)"          ,       1552 },
   {  "MQCD_LENGTH_5 (8 byte)"          ,       1584 },
   {  "MQCD_LENGTH_6 (4 byte)"          ,       1648 },
   {  "MQCD_LENGTH_6 (8 byte)"          ,       1688 },
   {  "MQCD_LENGTH_7 (4 byte)"          ,       1748 },
   {  "MQCD_LENGTH_7 (8 byte)"          ,       1792 },
   {  "MQCD_LENGTH_8 (4 byte)"          ,       1840 },
   {  "MQCD_LENGTH_8 (8 byte)"          ,       1888 },
   {  "MQCD_LENGTH_9 (4 byte)"          ,       1864 },
   {  "MQCD_LENGTH_9 (8 byte)"          ,       1912 },
   {  "MQCD_VERSION_1"                  ,          1 },
   {  "MQCD_VERSION_10"                 ,         10 },
   {  "MQCD_VERSION_11"                 ,         11 },
   {  "MQCD_VERSION_2"                  ,          2 },
   {  "MQCD_VERSION_3"                  ,          3 },
   {  "MQCD_VERSION_4"                  ,          4 },
   {  "MQCD_VERSION_5"                  ,          5 },
   {  "MQCD_VERSION_6"                  ,          6 },
   {  "MQCD_VERSION_7"                  ,          7 },
   {  "MQCD_VERSION_8"                  ,          8 },
   {  "MQCD_VERSION_9"                  ,          9 },
   {  "MQCFACCESS_DISABLED"             ,          2 },
   {  "MQCFACCESS_ENABLED"              ,          0 },
   {  "MQCFACCESS_SUSPENDED"            ,          1 },
   {  "MQCFBF_STRUC_LENGTH_FIXED"       ,         20 },
   {  "MQCFBS_STRUC_LENGTH_FIXED"       ,         16 },
   {  "MQCFCONLOS_ASQMGR"               ,          2 },
   {  "MQCFCONLOS_TERMINATE"            ,          0 },
   {  "MQCFCONLOS_TOLERATE"             ,          1 },
   {  "MQCFC_LAST"                      ,          1 },
   {  "MQCFC_NOT_LAST"                  ,          0 },
   {  "MQCFGR_STRUC_LENGTH"             ,         16 },
   {  "MQCFH_CURRENT_VERSION"           ,          3 },
   {  "MQCFH_STRUC_LENGTH"              ,         36 },
   {  "MQCFH_VERSION_1"                 ,          1 },
   {  "MQCFH_VERSION_2"                 ,          2 },
   {  "MQCFH_VERSION_3"                 ,          3 },
   {  "MQCFIF_STRUC_LENGTH"             ,         20 },
   {  "MQCFIL64_STRUC_LENGTH_FIXED"     ,         16 },
   {  "MQCFIL_STRUC_LENGTH_FIXED"       ,         16 },
   {  "MQCFIN64_STRUC_LENGTH"           ,         24 },
   {  "MQCFIN_STRUC_LENGTH"             ,         16 },
   {  "MQCFOFFLD_BOTH"                  ,          3 },
   {  "MQCFOFFLD_DB2"                   ,          2 },
   {  "MQCFOFFLD_NONE"                  ,          0 },
   {  "MQCFOFFLD_SMDS"                  ,          1 },
   {  "MQCFOP_CONTAINS"                 ,         10 },
   {  "MQCFOP_CONTAINS_GEN"             ,         26 },
   {  "MQCFOP_EQUAL"                    ,          2 },
   {  "MQCFOP_EXCLUDES"                 ,         13 },
   {  "MQCFOP_EXCLUDES_GEN"             ,         29 },
   {  "MQCFOP_GREATER"                  ,          4 },
   {  "MQCFOP_LESS"                     ,          1 },
   {  "MQCFOP_LIKE"                     ,         18 },
   {  "MQCFOP_NOT_EQUAL"                ,          5 },
   {  "MQCFOP_NOT_GREATER"              ,          3 },
   {  "MQCFOP_NOT_LESS"                 ,          6 },
   {  "MQCFOP_NOT_LIKE"                 ,         21 },
   {  "MQCFO_REFRESH_REPOSITORY_NO"     ,          0 },
   {  "MQCFO_REFRESH_REPOSITORY_YES"    ,          1 },
   {  "MQCFO_REMOVE_QUEUES_NO"          ,          0 },
   {  "MQCFO_REMOVE_QUEUES_YES"         ,          1 },
   {  "MQCFR_NO"                        ,          0 },
   {  "MQCFR_YES"                       ,          1 },
   {  "MQCFSF_STRUC_LENGTH_FIXED"       ,         24 },
   {  "MQCFSL_STRUC_LENGTH_FIXED"       ,         24 },
   {  "MQCFSTATUS_ACTIVE"               ,          1 },
   {  "MQCFSTATUS_ADMIN_INCOMPLETE"     ,         20 },
   {  "MQCFSTATUS_EMPTY"                ,          8 },
   {  "MQCFSTATUS_FAILED"               ,          4 },
   {  "MQCFSTATUS_IN_BACKUP"            ,          3 },
   {  "MQCFSTATUS_IN_RECOVER"           ,          2 },
   {  "MQCFSTATUS_NEVER_USED"           ,         21 },
   {  "MQCFSTATUS_NEW"                  ,          9 },
   {  "MQCFSTATUS_NONE"                 ,          5 },
   {  "MQCFSTATUS_NOT_FAILED"           ,         23 },
   {  "MQCFSTATUS_NOT_FOUND"            ,          0 },
   {  "MQCFSTATUS_NOT_RECOVERABLE"      ,         24 },
   {  "MQCFSTATUS_NO_BACKUP"            ,         22 },
   {  "MQCFSTATUS_RECOVERED"            ,          7 },
   {  "MQCFSTATUS_UNKNOWN"              ,          6 },
   {  "MQCFSTATUS_XES_ERROR"            ,         25 },
   {  "MQCFST_STRUC_LENGTH_FIXED"       ,         20 },
   {  "MQCFTYPE_ADMIN"                  ,          1 },
   {  "MQCFTYPE_APPL"                   ,          0 },
   {  "MQCFT_ACCOUNTING"                ,         22 },
   {  "MQCFT_APP_ACTIVITY"              ,         26 },
   {  "MQCFT_BYTE_STRING"               ,          9 },
   {  "MQCFT_BYTE_STRING_FILTER"        ,         15 },
   {  "MQCFT_COMMAND"                   ,          1 },
   {  "MQCFT_COMMAND_XR"                ,         16 },
   {  "MQCFT_EVENT"                     ,          7 },
   {  "MQCFT_GROUP"                     ,         20 },
   {  "MQCFT_INTEGER"                   ,          3 },
   {  "MQCFT_INTEGER64"                 ,         23 },
   {  "MQCFT_INTEGER64_LIST"            ,         25 },
   {  "MQCFT_INTEGER_FILTER"            ,         13 },
   {  "MQCFT_INTEGER_LIST"              ,          5 },
   {  "MQCFT_NONE"                      ,          0 },
   {  "MQCFT_REPORT"                    ,         12 },
   {  "MQCFT_RESPONSE"                  ,          2 },
   {  "MQCFT_STATISTICS"                ,         21 },
   {  "MQCFT_STRING"                    ,          4 },
   {  "MQCFT_STRING_FILTER"             ,         14 },
   {  "MQCFT_STRING_LIST"               ,          6 },
   {  "MQCFT_TRACE_ROUTE"               ,         10 },
   {  "MQCFT_USER"                      ,          8 },
   {  "MQCFT_XR_ITEM"                   ,         18 },
   {  "MQCFT_XR_MSG"                    ,         17 },
   {  "MQCFT_XR_SUMMARY"                ,         19 },
   {  "MQCF_DIST_LISTS"                 ,          1 },
   {  "MQCF_NONE"                       ,          0 },
   {  "MQCGWI_DEFAULT"                  ,         -2 },
   {  "MQCHAD_DISABLED"                 ,          0 },
   {  "MQCHAD_ENABLED"                  ,          1 },
   {  "MQCHIDS_INDOUBT"                 ,          1 },
   {  "MQCHIDS_NOT_INDOUBT"             ,          0 },
   {  "MQCHK_AS_Q_MGR"                  ,          4 },
   {  "MQCHK_NONE"                      ,          1 },
   {  "MQCHK_OPTIONAL"                  ,          0 },
   {  "MQCHK_REQUIRED"                  ,          3 },
   {  "MQCHK_REQUIRED_ADMIN"            ,          2 },
   {  "MQCHLA_DISABLED"                 ,          0 },
   {  "MQCHLA_ENABLED"                  ,          1 },
   {  "MQCHLD_ALL"                      ,         -1 },
   {  "MQCHLD_DEFAULT"                  ,          1 },
   {  "MQCHLD_FIXSHARED"                ,          5 },
   {  "MQCHLD_PRIVATE"                  ,          4 },
   {  "MQCHLD_SHARED"                   ,          2 },
   {  "MQCHRR_RESET_NOT_REQUESTED"      ,          0 },
   {  "MQCHSH_RESTART_NO"               ,          0 },
   {  "MQCHSH_RESTART_YES"              ,          1 },
   {  "MQCHSR_STOP_NOT_REQUESTED"       ,          0 },
   {  "MQCHSR_STOP_REQUESTED"           ,          1 },
   {  "MQCHSSTATE_COMPRESSING"          ,       1800 },
   {  "MQCHSSTATE_END_OF_BATCH"         ,        100 },
   {  "MQCHSSTATE_HEARTBEATING"         ,        600 },
   {  "MQCHSSTATE_IN_CHADEXIT"          ,       1200 },
   {  "MQCHSSTATE_IN_MQGET"             ,       1600 },
   {  "MQCHSSTATE_IN_MQI_CALL"          ,       1700 },
   {  "MQCHSSTATE_IN_MQPUT"             ,       1500 },
   {  "MQCHSSTATE_IN_MREXIT"            ,       1100 },
   {  "MQCHSSTATE_IN_MSGEXIT"           ,       1000 },
   {  "MQCHSSTATE_IN_RCVEXIT"           ,        800 },
   {  "MQCHSSTATE_IN_SCYEXIT"           ,        700 },
   {  "MQCHSSTATE_IN_SENDEXIT"          ,        900 },
   {  "MQCHSSTATE_NAME_SERVER"          ,       1400 },
   {  "MQCHSSTATE_NET_CONNECTING"       ,       1250 },
   {  "MQCHSSTATE_OTHER"                ,          0 },
   {  "MQCHSSTATE_RECEIVING"            ,        300 },
   {  "MQCHSSTATE_RESYNCHING"           ,        500 },
   {  "MQCHSSTATE_SENDING"              ,        200 },
   {  "MQCHSSTATE_SERIALIZING"          ,        400 },
   {  "MQCHSSTATE_SSL_HANDSHAKING"      ,       1300 },
   {  "MQCHS_BINDING"                   ,          1 },
   {  "MQCHS_DISCONNECTED"              ,          9 },
   {  "MQCHS_INACTIVE"                  ,          0 },
   {  "MQCHS_INITIALIZING"              ,         13 },
   {  "MQCHS_PAUSED"                    ,          8 },
   {  "MQCHS_REQUESTING"                ,          7 },
   {  "MQCHS_RETRYING"                  ,          5 },
   {  "MQCHS_RUNNING"                   ,          3 },
   {  "MQCHS_STARTING"                  ,          2 },
   {  "MQCHS_STOPPED"                   ,          6 },
   {  "MQCHS_STOPPING"                  ,          4 },
   {  "MQCHS_SWITCHING"                 ,         14 },
   {  "MQCHTAB_CLNTCONN"                ,          2 },
   {  "MQCHTAB_Q_MGR"                   ,          1 },
   {  "MQCHT_ALL"                       ,          5 },
   {  "MQCHT_AMQP"                      ,         11 },
   {  "MQCHT_CLNTCONN"                  ,          6 },
   {  "MQCHT_CLUSRCVR"                  ,          8 },
   {  "MQCHT_CLUSSDR"                   ,          9 },
   {  "MQCHT_MQTT"                      ,         10 },
   {  "MQCHT_RECEIVER"                  ,          3 },
   {  "MQCHT_REQUESTER"                 ,          4 },
   {  "MQCHT_SENDER"                    ,          1 },
   {  "MQCHT_SERVER"                    ,          2 },
   {  "MQCHT_SVRCONN"                   ,          7 },
   {  "MQCIH_CURRENT_LENGTH"            ,        180 },
   {  "MQCIH_CURRENT_VERSION"           ,          2 },
   {  "MQCIH_LENGTH_1"                  ,        164 },
   {  "MQCIH_LENGTH_2"                  ,        180 },
   {  "MQCIH_NONE"                      ,          0 },
   {  "MQCIH_NO_SYNC_ON_RETURN"         ,          0 },
   {  "MQCIH_PASS_EXPIRATION"           ,          1 },
   {  "MQCIH_REPLY_WITHOUT_NULLS"       ,          2 },
   {  "MQCIH_REPLY_WITH_NULLS"          ,          0 },
   {  "MQCIH_SYNC_ON_RETURN"            ,          4 },
   {  "MQCIH_UNLIMITED_EXPIRATION"      ,          0 },
   {  "MQCIH_VERSION_1"                 ,          1 },
   {  "MQCIH_VERSION_2"                 ,          2 },
   {  "MQCIT_MULTICAST"                 ,          1 },
   {  "MQCLCT_DYNAMIC"                  ,          1 },
   {  "MQCLCT_STATIC"                   ,          0 },
   {  "MQCLROUTE_DIRECT"                ,          0 },
   {  "MQCLROUTE_NONE"                  ,          2 },
   {  "MQCLROUTE_TOPIC_HOST"            ,          1 },
   {  "MQCLRS_GLOBAL"                   ,          2 },
   {  "MQCLRS_LOCAL"                    ,          1 },
   {  "MQCLRT_RETAINED"                 ,          1 },
   {  "MQCLST_ACTIVE"                   ,          0 },
   {  "MQCLST_ERROR"                    ,          3 },
   {  "MQCLST_INVALID"                  ,          2 },
   {  "MQCLST_PENDING"                  ,          1 },
   {  "MQCLT_PROGRAM"                   ,          1 },
   {  "MQCLT_TRANSACTION"               ,          2 },
   {  "MQCLWL_USEQ_ANY"                 ,          1 },
   {  "MQCLWL_USEQ_AS_Q_MGR"            ,         -3 },
   {  "MQCLWL_USEQ_LOCAL"               ,          0 },
   {  "MQCLXQ_CHANNEL"                  ,          1 },
   {  "MQCLXQ_SCTQ"                     ,          0 },
   {  "MQCMDI_BACKUP_STARTED"           ,         12 },
   {  "MQCMDI_CHANNEL_INIT_STARTED"     ,          7 },
   {  "MQCMDI_CLUSTER_REQUEST_QUEUED"   ,          6 },
   {  "MQCMDI_CMDSCOPE_ACCEPTED"        ,          1 },
   {  "MQCMDI_CMDSCOPE_COMPLETED"       ,          3 },
   {  "MQCMDI_CMDSCOPE_GENERATED"       ,          2 },
   {  "MQCMDI_COMMAND_ACCEPTED"         ,          5 },
   {  "MQCMDI_DB2_OBSOLETE_MSGS"        ,         20 },
   {  "MQCMDI_DB2_SUSPENDED"            ,         19 },
   {  "MQCMDI_IMS_BRIDGE_SUSPENDED"     ,         18 },
   {  "MQCMDI_QSG_DISP_COMPLETED"       ,          4 },
   {  "MQCMDI_RECOVER_COMPLETED"        ,         13 },
   {  "MQCMDI_RECOVER_STARTED"          ,         11 },
   {  "MQCMDI_REFRESH_CONFIGURATION"    ,         16 },
   {  "MQCMDI_SEC_MIXEDCASE"            ,         22 },
   {  "MQCMDI_SEC_SIGNOFF_ERROR"        ,         17 },
   {  "MQCMDI_SEC_TIMER_ZERO"           ,         14 },
   {  "MQCMDI_SEC_UPPERCASE"            ,         21 },
   {  "MQCMDL_CURRENT_LEVEL"            ,        911 },
   {  "MQCMDL_LEVEL_1"                  ,        100 },
   {  "MQCMDL_LEVEL_101"                ,        101 },
   {  "MQCMDL_LEVEL_110"                ,        110 },
   {  "MQCMDL_LEVEL_114"                ,        114 },
   {  "MQCMDL_LEVEL_120"                ,        120 },
   {  "MQCMDL_LEVEL_200"                ,        200 },
   {  "MQCMDL_LEVEL_201"                ,        201 },
   {  "MQCMDL_LEVEL_210"                ,        210 },
   {  "MQCMDL_LEVEL_211"                ,        211 },
   {  "MQCMDL_LEVEL_220"                ,        220 },
   {  "MQCMDL_LEVEL_221"                ,        221 },
   {  "MQCMDL_LEVEL_230"                ,        230 },
   {  "MQCMDL_LEVEL_320"                ,        320 },
   {  "MQCMDL_LEVEL_420"                ,        420 },
   {  "MQCMDL_LEVEL_500"                ,        500 },
   {  "MQCMDL_LEVEL_510"                ,        510 },
   {  "MQCMDL_LEVEL_520"                ,        520 },
   {  "MQCMDL_LEVEL_530"                ,        530 },
   {  "MQCMDL_LEVEL_531"                ,        531 },
   {  "MQCMDL_LEVEL_600"                ,        600 },
   {  "MQCMDL_LEVEL_700"                ,        700 },
   {  "MQCMDL_LEVEL_701"                ,        701 },
   {  "MQCMDL_LEVEL_710"                ,        710 },
   {  "MQCMDL_LEVEL_711"                ,        711 },
   {  "MQCMDL_LEVEL_750"                ,        750 },
   {  "MQCMDL_LEVEL_800"                ,        800 },
   {  "MQCMDL_LEVEL_801"                ,        801 },
   {  "MQCMDL_LEVEL_802"                ,        802 },
   {  "MQCMDL_LEVEL_900"                ,        900 },
   {  "MQCMDL_LEVEL_901"                ,        901 },
   {  "MQCMDL_LEVEL_902"                ,        902 },
   {  "MQCMDL_LEVEL_903"                ,        903 },
   {  "MQCMDL_LEVEL_904"                ,        904 },
   {  "MQCMDL_LEVEL_905"                ,        905 },
   {  "MQCMDL_LEVEL_910"                ,        910 },
   {  "MQCMDL_LEVEL_911"                ,        911 },
   {  "MQCMD_ACCOUNTING_MQI"            ,        167 },
   {  "MQCMD_ACCOUNTING_Q"              ,        168 },
   {  "MQCMD_ACTIVITY_MSG"              ,         69 },
   {  "MQCMD_ACTIVITY_TRACE"            ,        209 },
   {  "MQCMD_AMQP_DIAGNOSTICS"          ,        217 },
   {  "MQCMD_ARCHIVE_LOG"               ,        104 },
   {  "MQCMD_BACKUP_CF_STRUC"           ,        105 },
   {  "MQCMD_BROKER_INTERNAL"           ,         67 },
   {  "MQCMD_CHANGE_AUTH_INFO"          ,         79 },
   {  "MQCMD_CHANGE_BUFFER_POOL"        ,        159 },
   {  "MQCMD_CHANGE_CF_STRUC"           ,        101 },
   {  "MQCMD_CHANGE_CHANNEL"            ,         21 },
   {  "MQCMD_CHANGE_COMM_INFO"          ,        192 },
   {  "MQCMD_CHANGE_LISTENER"           ,         93 },
   {  "MQCMD_CHANGE_NAMELIST"           ,         32 },
   {  "MQCMD_CHANGE_PAGE_SET"           ,        160 },
   {  "MQCMD_CHANGE_PROCESS"            ,          3 },
   {  "MQCMD_CHANGE_PROT_POLICY"        ,        208 },
   {  "MQCMD_CHANGE_Q"                  ,          8 },
   {  "MQCMD_CHANGE_Q_MGR"              ,          1 },
   {  "MQCMD_CHANGE_SECURITY"           ,        100 },
   {  "MQCMD_CHANGE_SERVICE"            ,        149 },
   {  "MQCMD_CHANGE_SMDS"               ,        187 },
   {  "MQCMD_CHANGE_STG_CLASS"          ,        102 },
   {  "MQCMD_CHANGE_SUBSCRIPTION"       ,        178 },
   {  "MQCMD_CHANGE_TOPIC"              ,        170 },
   {  "MQCMD_CHANGE_TRACE"              ,        103 },
   {  "MQCMD_CHANNEL_EVENT"             ,         46 },
   {  "MQCMD_CLEAR_Q"                   ,          9 },
   {  "MQCMD_CLEAR_TOPIC_STRING"        ,        184 },
   {  "MQCMD_COMMAND_EVENT"             ,         99 },
   {  "MQCMD_CONFIG_EVENT"              ,         43 },
   {  "MQCMD_COPY_AUTH_INFO"            ,         80 },
   {  "MQCMD_COPY_CF_STRUC"             ,        110 },
   {  "MQCMD_COPY_CHANNEL"              ,         22 },
   {  "MQCMD_COPY_COMM_INFO"            ,        193 },
   {  "MQCMD_COPY_LISTENER"             ,         94 },
   {  "MQCMD_COPY_NAMELIST"             ,         33 },
   {  "MQCMD_COPY_PROCESS"              ,          4 },
   {  "MQCMD_COPY_Q"                    ,         10 },
   {  "MQCMD_COPY_SERVICE"              ,        150 },
   {  "MQCMD_COPY_STG_CLASS"            ,        111 },
   {  "MQCMD_COPY_SUBSCRIPTION"         ,        181 },
   {  "MQCMD_COPY_TOPIC"                ,        171 },
   {  "MQCMD_CREATE_AUTH_INFO"          ,         81 },
   {  "MQCMD_CREATE_BUFFER_POOL"        ,        106 },
   {  "MQCMD_CREATE_CF_STRUC"           ,        108 },
   {  "MQCMD_CREATE_CHANNEL"            ,         23 },
   {  "MQCMD_CREATE_COMM_INFO"          ,        190 },
   {  "MQCMD_CREATE_LISTENER"           ,         95 },
   {  "MQCMD_CREATE_LOG"                ,        162 },
   {  "MQCMD_CREATE_NAMELIST"           ,         34 },
   {  "MQCMD_CREATE_PAGE_SET"           ,        107 },
   {  "MQCMD_CREATE_PROCESS"            ,          5 },
   {  "MQCMD_CREATE_PROT_POLICY"        ,        206 },
   {  "MQCMD_CREATE_Q"                  ,         11 },
   {  "MQCMD_CREATE_SERVICE"            ,        151 },
   {  "MQCMD_CREATE_STG_CLASS"          ,        109 },
   {  "MQCMD_CREATE_SUBSCRIPTION"       ,        177 },
   {  "MQCMD_CREATE_TOPIC"              ,        172 },
   {  "MQCMD_DELETE_AUTH_INFO"          ,         82 },
   {  "MQCMD_DELETE_AUTH_REC"           ,         89 },
   {  "MQCMD_DELETE_BUFFER_POOL"        ,        157 },
   {  "MQCMD_DELETE_CF_STRUC"           ,        112 },
   {  "MQCMD_DELETE_CHANNEL"            ,         24 },
   {  "MQCMD_DELETE_COMM_INFO"          ,        194 },
   {  "MQCMD_DELETE_LISTENER"           ,         96 },
   {  "MQCMD_DELETE_NAMELIST"           ,         35 },
   {  "MQCMD_DELETE_PAGE_SET"           ,        158 },
   {  "MQCMD_DELETE_PROCESS"            ,          6 },
   {  "MQCMD_DELETE_PROT_POLICY"        ,        207 },
   {  "MQCMD_DELETE_PUBLICATION"        ,         60 },
   {  "MQCMD_DELETE_Q"                  ,         12 },
   {  "MQCMD_DELETE_SERVICE"            ,        152 },
   {  "MQCMD_DELETE_STG_CLASS"          ,        113 },
   {  "MQCMD_DELETE_SUBSCRIPTION"       ,        179 },
   {  "MQCMD_DELETE_TOPIC"              ,        173 },
   {  "MQCMD_DEREGISTER_PUBLISHER"      ,         61 },
   {  "MQCMD_DEREGISTER_SUBSCRIBER"     ,         62 },
   {  "MQCMD_ESCAPE"                    ,         38 },
   {  "MQCMD_INQUIRE_AMQP_CAPABILITY"   ,        216 },
   {  "MQCMD_INQUIRE_ARCHIVE"           ,        114 },
   {  "MQCMD_INQUIRE_AUTH_INFO"         ,         83 },
   {  "MQCMD_INQUIRE_AUTH_INFO_NAMES"   ,         84 },
   {  "MQCMD_INQUIRE_AUTH_RECS"         ,         87 },
   {  "MQCMD_INQUIRE_AUTH_SERVICE"      ,        169 },
   {  "MQCMD_INQUIRE_CF_STRUC"          ,        115 },
   {  "MQCMD_INQUIRE_CF_STRUC_NAMES"    ,        147 },
   {  "MQCMD_INQUIRE_CF_STRUC_STATUS"   ,        116 },
   {  "MQCMD_INQUIRE_CHANNEL"           ,         25 },
   {  "MQCMD_INQUIRE_CHANNEL_INIT"      ,        118 },
   {  "MQCMD_INQUIRE_CHANNEL_NAMES"     ,         20 },
   {  "MQCMD_INQUIRE_CHANNEL_STATUS"    ,         42 },
   {  "MQCMD_INQUIRE_CHLAUTH_RECS"      ,        204 },
   {  "MQCMD_INQUIRE_CLUSTER_Q_MGR"     ,         70 },
   {  "MQCMD_INQUIRE_CMD_SERVER"        ,        117 },
   {  "MQCMD_INQUIRE_COMM_INFO"         ,        191 },
   {  "MQCMD_INQUIRE_CONNECTION"        ,         85 },
   {  "MQCMD_INQUIRE_ENTITY_AUTH"       ,         88 },
   {  "MQCMD_INQUIRE_LISTENER"          ,         97 },
   {  "MQCMD_INQUIRE_LISTENER_STATUS"   ,         98 },
   {  "MQCMD_INQUIRE_LOG"               ,        120 },
   {  "MQCMD_INQUIRE_MQXR_STATUS"       ,        200 },
   {  "MQCMD_INQUIRE_NAMELIST"          ,         36 },
   {  "MQCMD_INQUIRE_NAMELIST_NAMES"    ,         37 },
   {  "MQCMD_INQUIRE_PROCESS"           ,          7 },
   {  "MQCMD_INQUIRE_PROCESS_NAMES"     ,         19 },
   {  "MQCMD_INQUIRE_PROT_POLICY"       ,        205 },
   {  "MQCMD_INQUIRE_PUBSUB_STATUS"     ,        185 },
   {  "MQCMD_INQUIRE_Q"                 ,         13 },
   {  "MQCMD_INQUIRE_QSG"               ,        119 },
   {  "MQCMD_INQUIRE_Q_MGR"             ,          2 },
   {  "MQCMD_INQUIRE_Q_MGR_STATUS"      ,        161 },
   {  "MQCMD_INQUIRE_Q_NAMES"           ,         18 },
   {  "MQCMD_INQUIRE_Q_STATUS"          ,         41 },
   {  "MQCMD_INQUIRE_SECURITY"          ,        121 },
   {  "MQCMD_INQUIRE_SERVICE"           ,        153 },
   {  "MQCMD_INQUIRE_SERVICE_STATUS"    ,        154 },
   {  "MQCMD_INQUIRE_SMDS"              ,        186 },
   {  "MQCMD_INQUIRE_SMDSCONN"          ,        199 },
   {  "MQCMD_INQUIRE_STG_CLASS"         ,        122 },
   {  "MQCMD_INQUIRE_STG_CLASS_NAMES"   ,        148 },
   {  "MQCMD_INQUIRE_SUBSCRIPTION"      ,        176 },
   {  "MQCMD_INQUIRE_SUB_STATUS"        ,        182 },
   {  "MQCMD_INQUIRE_SYSTEM"            ,        123 },
   {  "MQCMD_INQUIRE_THREAD"            ,        124 },
   {  "MQCMD_INQUIRE_TOPIC"             ,        174 },
   {  "MQCMD_INQUIRE_TOPIC_NAMES"       ,        175 },
   {  "MQCMD_INQUIRE_TOPIC_STATUS"      ,        183 },
   {  "MQCMD_INQUIRE_TRACE"             ,        125 },
   {  "MQCMD_INQUIRE_USAGE"             ,        126 },
   {  "MQCMD_INQUIRE_XR_CAPABILITY"     ,        214 },
   {  "MQCMD_LOGGER_EVENT"              ,         91 },
   {  "MQCMD_MOVE_Q"                    ,        127 },
   {  "MQCMD_MQXR_DIAGNOSTICS"          ,        196 },
   {  "MQCMD_NONE"                      ,          0 },
   {  "MQCMD_PERFM_EVENT"               ,         45 },
   {  "MQCMD_PING_CHANNEL"              ,         26 },
   {  "MQCMD_PING_Q_MGR"                ,         40 },
   {  "MQCMD_PUBLISH"                   ,         63 },
   {  "MQCMD_PURGE_CHANNEL"             ,        195 },
   {  "MQCMD_Q_MGR_EVENT"               ,         44 },
   {  "MQCMD_RECOVER_BSDS"              ,        128 },
   {  "MQCMD_RECOVER_CF_STRUC"          ,        129 },
   {  "MQCMD_REFRESH_CLUSTER"           ,         73 },
   {  "MQCMD_REFRESH_Q_MGR"             ,         16 },
   {  "MQCMD_REFRESH_SECURITY"          ,         78 },
   {  "MQCMD_REGISTER_PUBLISHER"        ,         64 },
   {  "MQCMD_REGISTER_SUBSCRIBER"       ,         65 },
   {  "MQCMD_REQUEST_UPDATE"            ,         66 },
   {  "MQCMD_RESET_CF_STRUC"            ,        213 },
   {  "MQCMD_RESET_CHANNEL"             ,         27 },
   {  "MQCMD_RESET_CLUSTER"             ,         74 },
   {  "MQCMD_RESET_Q_MGR"               ,         92 },
   {  "MQCMD_RESET_Q_STATS"             ,         17 },
   {  "MQCMD_RESET_SMDS"                ,        188 },
   {  "MQCMD_RESET_TPIPE"               ,        130 },
   {  "MQCMD_RESOLVE_CHANNEL"           ,         39 },
   {  "MQCMD_RESOLVE_INDOUBT"           ,        131 },
   {  "MQCMD_RESUME_Q_MGR"              ,        132 },
   {  "MQCMD_RESUME_Q_MGR_CLUSTER"      ,         71 },
   {  "MQCMD_REVERIFY_SECURITY"         ,        133 },
   {  "MQCMD_SET_ARCHIVE"               ,        134 },
   {  "MQCMD_SET_AUTH_REC"              ,         90 },
   {  "MQCMD_SET_CHLAUTH_REC"           ,        203 },
   {  "MQCMD_SET_LOG"                   ,        136 },
   {  "MQCMD_SET_PROT_POLICY"           ,        208 },
   {  "MQCMD_SET_SYSTEM"                ,        137 },
   {  "MQCMD_START_CHANNEL"             ,         28 },
   {  "MQCMD_START_CHANNEL_INIT"        ,         30 },
   {  "MQCMD_START_CHANNEL_LISTENER"    ,         31 },
   {  "MQCMD_START_CLIENT_TRACE"        ,        201 },
   {  "MQCMD_START_CMD_SERVER"          ,        138 },
   {  "MQCMD_START_Q_MGR"               ,        139 },
   {  "MQCMD_START_SERVICE"             ,        155 },
   {  "MQCMD_START_SMDSCONN"            ,        197 },
   {  "MQCMD_START_TRACE"               ,        140 },
   {  "MQCMD_STATISTICS_CHANNEL"        ,        166 },
   {  "MQCMD_STATISTICS_MQI"            ,        164 },
   {  "MQCMD_STATISTICS_Q"              ,        165 },
   {  "MQCMD_STOP_CHANNEL"              ,         29 },
   {  "MQCMD_STOP_CHANNEL_INIT"         ,        141 },
   {  "MQCMD_STOP_CHANNEL_LISTENER"     ,        142 },
   {  "MQCMD_STOP_CLIENT_TRACE"         ,        202 },
   {  "MQCMD_STOP_CMD_SERVER"           ,        143 },
   {  "MQCMD_STOP_CONNECTION"           ,         86 },
   {  "MQCMD_STOP_Q_MGR"                ,        144 },
   {  "MQCMD_STOP_SERVICE"              ,        156 },
   {  "MQCMD_STOP_SMDSCONN"             ,        198 },
   {  "MQCMD_STOP_TRACE"                ,        145 },
   {  "MQCMD_SUSPEND_Q_MGR"             ,        146 },
   {  "MQCMD_SUSPEND_Q_MGR_CLUSTER"     ,         72 },
   {  "MQCMD_TRACE_ROUTE"               ,         75 },
   {  "MQCMHO_CURRENT_LENGTH"           ,         12 },
   {  "MQCMHO_CURRENT_VERSION"          ,          1 },
   {  "MQCMHO_DEFAULT_VALIDATION"       ,          0 },
   {  "MQCMHO_LENGTH_1"                 ,         12 },
   {  "MQCMHO_NONE"                     ,          0 },
   {  "MQCMHO_NO_VALIDATION"            ,          1 },
   {  "MQCMHO_VALIDATE"                 ,          2 },
   {  "MQCMHO_VERSION_1"                ,          1 },
   {  "MQCNO_ACCOUNTING_MQI_DISABLED"   ,       8192 },
   {  "MQCNO_ACCOUNTING_MQI_ENABLED"    ,       4096 },
   {  "MQCNO_ACCOUNTING_Q_DISABLED"     ,      32768 },
   {  "MQCNO_ACCOUNTING_Q_ENABLED"      ,      16384 },
   {  "MQCNO_ACTIVITY_TRACE_DISABLED"   ,  268435456 },
   {  "MQCNO_ACTIVITY_TRACE_ENABLED"    ,  134217728 },
   {  "MQCNO_ALL_CONVS_SHARE"           ,     262144 },
   {  "MQCNO_CD_FOR_OUTPUT_ONLY"        ,     524288 },
   {  "MQCNO_CLIENT_BINDING"            ,       2048 },
   {  "MQCNO_CURRENT_LENGTH (4 byte)"   ,        208 },
   {  "MQCNO_CURRENT_LENGTH (8 byte)"   ,        224 },
   {  "MQCNO_CURRENT_VERSION"           ,          6 },
   {  "MQCNO_FASTPATH_BINDING"          ,          1 },
   {  "MQCNO_HANDLE_SHARE_BLOCK"        ,         64 },
   {  "MQCNO_HANDLE_SHARE_NONE"         ,         32 },
   {  "MQCNO_HANDLE_SHARE_NO_BLOCK"     ,        128 },
   {  "MQCNO_ISOLATED_BINDING"          ,        512 },
   {  "MQCNO_LENGTH_1"                  ,         12 },
   {  "MQCNO_LENGTH_2 (4 byte)"         ,         20 },
   {  "MQCNO_LENGTH_2 (8 byte)"         ,         24 },
   {  "MQCNO_LENGTH_3 (4 byte)"         ,        148 },
   {  "MQCNO_LENGTH_3 (8 byte)"         ,        152 },
   {  "MQCNO_LENGTH_4 (4 byte)"         ,        156 },
   {  "MQCNO_LENGTH_4 (8 byte)"         ,        168 },
   {  "MQCNO_LENGTH_5 (4 byte)"         ,        188 },
   {  "MQCNO_LENGTH_5 (8 byte)"         ,        200 },
   {  "MQCNO_LENGTH_6 (4 byte)"         ,        208 },
   {  "MQCNO_LENGTH_6 (8 byte)"         ,        224 },
   {  "MQCNO_LOCAL_BINDING"             ,       1024 },
   {  "MQCNO_NONE"                      ,          0 },
   {  "MQCNO_NO_CONV_SHARING"           ,      65536 },
   {  "MQCNO_RECONNECT"                 ,   16777216 },
   {  "MQCNO_RECONNECT_AS_DEF"          ,          0 },
   {  "MQCNO_RECONNECT_DISABLED"        ,   33554432 },
   {  "MQCNO_RECONNECT_Q_MGR"           ,   67108864 },
   {  "MQCNO_RESTRICT_CONN_TAG_QSG"     ,         16 },
   {  "MQCNO_RESTRICT_CONN_TAG_Q_MGR"   ,          8 },
   {  "MQCNO_SERIALIZE_CONN_TAG_QSG"    ,          4 },
   {  "MQCNO_SERIALIZE_CONN_TAG_Q_MGR"  ,          2 },
   {  "MQCNO_SHARED_BINDING"            ,        256 },
   {  "MQCNO_STANDARD_BINDING"          ,          0 },
   {  "MQCNO_USE_CD_SELECTION"          ,    1048576 },
   {  "MQCNO_VERSION_1"                 ,          1 },
   {  "MQCNO_VERSION_2"                 ,          2 },
   {  "MQCNO_VERSION_3"                 ,          3 },
   {  "MQCNO_VERSION_4"                 ,          4 },
   {  "MQCNO_VERSION_5"                 ,          5 },
   {  "MQCNO_VERSION_6"                 ,          6 },
   {  "MQCODL_AS_INPUT"                 ,         -1 },
   {  "MQCOMPRESS_ANY"                  ,  268435455 },
   {  "MQCOMPRESS_NONE"                 ,          0 },
   {  "MQCOMPRESS_NOT_AVAILABLE"        ,         -1 },
   {  "MQCOMPRESS_RLE"                  ,          1 },
   {  "MQCOMPRESS_SYSTEM"               ,          8 },
   {  "MQCOMPRESS_ZLIBFAST"             ,          2 },
   {  "MQCOMPRESS_ZLIBHIGH"             ,          4 },
   {  "MQCOPY_ALL"                      ,          1 },
   {  "MQCOPY_DEFAULT"                  ,         22 },
   {  "MQCOPY_FORWARD"                  ,          2 },
   {  "MQCOPY_NONE"                     ,          0 },
   {  "MQCOPY_PUBLISH"                  ,          4 },
   {  "MQCOPY_REPLY"                    ,          8 },
   {  "MQCOPY_REPORT"                   ,         16 },
   {  "MQCO_DELETE"                     ,          1 },
   {  "MQCO_DELETE_PURGE"               ,          2 },
   {  "MQCO_IMMEDIATE"                  ,          0 },
   {  "MQCO_KEEP_SUB"                   ,          4 },
   {  "MQCO_NONE"                       ,          0 },
   {  "MQCO_QUIESCE"                    ,         32 },
   {  "MQCO_REMOVE_SUB"                 ,          8 },
   {  "MQCQT_ALIAS_Q"                   ,          2 },
   {  "MQCQT_LOCAL_Q"                   ,          1 },
   {  "MQCQT_Q_MGR_ALIAS"               ,          4 },
   {  "MQCQT_REMOTE_Q"                  ,          3 },
   {  "MQCRC_APPLICATION_ABEND"         ,          5 },
   {  "MQCRC_BRIDGE_ABEND"              ,          4 },
   {  "MQCRC_BRIDGE_ERROR"              ,          3 },
   {  "MQCRC_BRIDGE_TIMEOUT"            ,          8 },
   {  "MQCRC_CICS_EXEC_ERROR"           ,          1 },
   {  "MQCRC_MQ_API_ERROR"              ,          2 },
   {  "MQCRC_OK"                        ,          0 },
   {  "MQCRC_PROGRAM_NOT_AVAILABLE"     ,          7 },
   {  "MQCRC_SECURITY_ERROR"            ,          6 },
   {  "MQCRC_TRANSID_NOT_AVAILABLE"     ,          9 },
   {  "MQCSP_AUTH_NONE"                 ,          0 },
   {  "MQCSP_AUTH_USER_ID_AND_PWD"      ,          1 },
   {  "MQCSP_CURRENT_LENGTH (4 byte)"   ,         48 },
   {  "MQCSP_CURRENT_LENGTH (8 byte)"   ,         56 },
   {  "MQCSP_CURRENT_VERSION"           ,          1 },
   {  "MQCSP_LENGTH_1 (4 byte)"         ,         48 },
   {  "MQCSP_LENGTH_1 (8 byte)"         ,         56 },
   {  "MQCSP_VERSION_1"                 ,          1 },
   {  "MQCSRV_CONVERT_NO"               ,          0 },
   {  "MQCSRV_CONVERT_YES"              ,          1 },
   {  "MQCSRV_DLQ_NO"                   ,          0 },
   {  "MQCSRV_DLQ_YES"                  ,          1 },
   {  "MQCS_NONE"                       ,          0 },
   {  "MQCS_STOPPED"                    ,          4 },
   {  "MQCS_SUSPENDED"                  ,          3 },
   {  "MQCS_SUSPENDED_TEMPORARY"        ,          1 },
   {  "MQCS_SUSPENDED_USER_ACTION"      ,          2 },
   {  "MQCTES_BACKOUT"                  ,       4352 },
   {  "MQCTES_COMMIT"                   ,        256 },
   {  "MQCTES_ENDTASK"                  ,      65536 },
   {  "MQCTES_NOSYNC"                   ,          0 },
   {  "MQCTLO_CURRENT_LENGTH (4 byte)"  ,         20 },
   {  "MQCTLO_CURRENT_LENGTH (8 byte)"  ,         24 },
   {  "MQCTLO_CURRENT_VERSION"          ,          1 },
   {  "MQCTLO_FAIL_IF_QUIESCING"        ,       8192 },
   {  "MQCTLO_LENGTH_1 (4 byte)"        ,         20 },
   {  "MQCTLO_LENGTH_1 (8 byte)"        ,         24 },
   {  "MQCTLO_NONE"                     ,          0 },
   {  "MQCTLO_THREAD_AFFINITY"          ,          1 },
   {  "MQCTLO_VERSION_1"                ,          1 },
   {  "MQCUOWC_BACKOUT"                 ,       4352 },
   {  "MQCUOWC_COMMIT"                  ,        256 },
   {  "MQCUOWC_CONTINUE"                ,      65536 },
   {  "MQCUOWC_FIRST"                   ,         17 },
   {  "MQCUOWC_LAST"                    ,        272 },
   {  "MQCUOWC_MIDDLE"                  ,         16 },
   {  "MQCUOWC_ONLY"                    ,        273 },
   {  "MQCXP_CURRENT_LENGTH (4 byte)"   ,        220 },
   {  "MQCXP_CURRENT_LENGTH (8 byte)"   ,        240 },
   {  "MQCXP_CURRENT_VERSION"           ,          9 },
   {  "MQCXP_LENGTH_3"                  ,        156 },
   {  "MQCXP_LENGTH_4"                  ,        156 },
   {  "MQCXP_LENGTH_5"                  ,        160 },
   {  "MQCXP_LENGTH_6 (4 byte)"         ,        192 },
   {  "MQCXP_LENGTH_6 (8 byte)"         ,        200 },
   {  "MQCXP_LENGTH_7 (4 byte)"         ,        200 },
   {  "MQCXP_LENGTH_7 (8 byte)"         ,        208 },
   {  "MQCXP_LENGTH_8 (4 byte)"         ,        208 },
   {  "MQCXP_LENGTH_8 (8 byte)"         ,        224 },
   {  "MQCXP_LENGTH_9 (4 byte)"         ,        220 },
   {  "MQCXP_LENGTH_9 (8 byte)"         ,        240 },
   {  "MQCXP_VERSION_1"                 ,          1 },
   {  "MQCXP_VERSION_2"                 ,          2 },
   {  "MQCXP_VERSION_3"                 ,          3 },
   {  "MQCXP_VERSION_4"                 ,          4 },
   {  "MQCXP_VERSION_5"                 ,          5 },
   {  "MQCXP_VERSION_6"                 ,          6 },
   {  "MQCXP_VERSION_7"                 ,          7 },
   {  "MQCXP_VERSION_8"                 ,          8 },
   {  "MQCXP_VERSION_9"                 ,          9 },
   {  "MQDCC_DEFAULT_CONVERSION"        ,          1 },
   {  "MQDCC_FILL_TARGET_BUFFER"        ,          2 },
   {  "MQDCC_INT_DEFAULT_CONVERSION"    ,          4 },
   {  "MQDCC_NONE"                      ,          0 },
   {  "MQDCC_SOURCE_ENC_FACTOR"         ,         16 },
   {  "MQDCC_SOURCE_ENC_MASK"           ,        240 },
   {  "MQDCC_SOURCE_ENC_NATIVE"         ,         32 },
   {  "MQDCC_SOURCE_ENC_NORMAL"         ,         16 },
   {  "MQDCC_SOURCE_ENC_REVERSED"       ,         32 },
   {  "MQDCC_SOURCE_ENC_UNDEFINED"      ,          0 },
   {  "MQDCC_TARGET_ENC_FACTOR"         ,        256 },
   {  "MQDCC_TARGET_ENC_MASK"           ,       3840 },
   {  "MQDCC_TARGET_ENC_NATIVE"         ,        512 },
   {  "MQDCC_TARGET_ENC_NORMAL"         ,        256 },
   {  "MQDCC_TARGET_ENC_REVERSED"       ,        512 },
   {  "MQDCC_TARGET_ENC_UNDEFINED"      ,          0 },
   {  "MQDC_MANAGED"                    ,          1 },
   {  "MQDC_PROVIDED"                   ,          2 },
   {  "MQDELO_LOCAL"                    ,          4 },
   {  "MQDELO_NONE"                     ,          0 },
   {  "MQDHF_NEW_MSG_IDS"               ,          1 },
   {  "MQDHF_NONE"                      ,          0 },
   {  "MQDH_CURRENT_LENGTH"             ,         48 },
   {  "MQDH_CURRENT_VERSION"            ,          1 },
   {  "MQDH_LENGTH_1"                   ,         48 },
   {  "MQDH_VERSION_1"                  ,          1 },
   {  "MQDISCONNECT_IMPLICIT"           ,          1 },
   {  "MQDISCONNECT_NORMAL"             ,          0 },
   {  "MQDISCONNECT_Q_MGR"              ,          2 },
   {  "MQDLH_CURRENT_LENGTH"            ,        172 },
   {  "MQDLH_CURRENT_VERSION"           ,          1 },
   {  "MQDLH_LENGTH_1"                  ,        172 },
   {  "MQDLH_VERSION_1"                 ,          1 },
   {  "MQDLV_ALL"                       ,          1 },
   {  "MQDLV_ALL_AVAIL"                 ,          3 },
   {  "MQDLV_ALL_DUR"                   ,          2 },
   {  "MQDLV_AS_PARENT"                 ,          0 },
   {  "MQDL_NOT_SUPPORTED"              ,          0 },
   {  "MQDL_SUPPORTED"                  ,          1 },
   {  "MQDMHO_CURRENT_LENGTH"           ,         12 },
   {  "MQDMHO_CURRENT_VERSION"          ,          1 },
   {  "MQDMHO_LENGTH_1"                 ,         12 },
   {  "MQDMHO_NONE"                     ,          0 },
   {  "MQDMHO_VERSION_1"                ,          1 },
   {  "MQDMPO_CURRENT_LENGTH"           ,         12 },
   {  "MQDMPO_CURRENT_VERSION"          ,          1 },
   {  "MQDMPO_DEL_FIRST"                ,          0 },
   {  "MQDMPO_DEL_PROP_UNDER_CURSOR"    ,          1 },
   {  "MQDMPO_LENGTH_1"                 ,         12 },
   {  "MQDMPO_NONE"                     ,          0 },
   {  "MQDMPO_VERSION_1"                ,          1 },
   {  "MQDNSWLM_NO"                     ,          0 },
   {  "MQDNSWLM_YES"                    ,          1 },
   {  "MQDOPT_DEFINED"                  ,          1 },
   {  "MQDOPT_RESOLVED"                 ,          0 },
   {  "MQDSB_1024K"                     ,          8 },
   {  "MQDSB_128K"                      ,          5 },
   {  "MQDSB_16K"                       ,          2 },
   {  "MQDSB_1M"                        ,          8 },
   {  "MQDSB_256K"                      ,          6 },
   {  "MQDSB_32K"                       ,          3 },
   {  "MQDSB_512K"                      ,          7 },
   {  "MQDSB_64K"                       ,          4 },
   {  "MQDSB_8K"                        ,          1 },
   {  "MQDSB_DEFAULT"                   ,          0 },
   {  "MQDSE_DEFAULT"                   ,          0 },
   {  "MQDSE_NO"                        ,          2 },
   {  "MQDSE_YES"                       ,          1 },
   {  "MQDXP_CURRENT_LENGTH (4 byte)"   ,         48 },
   {  "MQDXP_CURRENT_LENGTH (8 byte)"   ,         56 },
   {  "MQDXP_CURRENT_VERSION"           ,          2 },
   {  "MQDXP_LENGTH_1"                  ,         44 },
   {  "MQDXP_LENGTH_2 (4 byte)"         ,         48 },
   {  "MQDXP_LENGTH_2 (8 byte)"         ,         56 },
   {  "MQDXP_VERSION_1"                 ,          1 },
   {  "MQDXP_VERSION_2"                 ,          2 },
   {  "MQEC_CONNECTION_QUIESCING"       ,          6 },
   {  "MQEC_MSG_ARRIVED"                ,          2 },
   {  "MQEC_Q_MGR_QUIESCING"            ,          5 },
   {  "MQEC_WAIT_CANCELED"              ,          4 },
   {  "MQEC_WAIT_INTERVAL_EXPIRED"      ,          3 },
   {  "MQEI_UNLIMITED"                  ,         -1 },
   {  "MQENC_AS_PUBLISHED"              ,         -1 },
   {  "MQENC_DECIMAL_MASK"              ,        240 },
   {  "MQENC_DECIMAL_NORMAL"            ,         16 },
   {  "MQENC_DECIMAL_REVERSED"          ,         32 },
   {  "MQENC_DECIMAL_UNDEFINED"         ,          0 },
   {  "MQENC_FLOAT_IEEE_NORMAL"         ,        256 },
   {  "MQENC_FLOAT_IEEE_REVERSED"       ,        512 },
   {  "MQENC_FLOAT_MASK"                ,       3840 },
   {  "MQENC_FLOAT_S390"                ,        768 },
   {  "MQENC_FLOAT_TNS"                 ,       1024 },
   {  "MQENC_FLOAT_UNDEFINED"           ,          0 },
   {  "MQENC_INTEGER_MASK"              ,         15 },
   {  "MQENC_INTEGER_NORMAL"            ,          1 },
   {  "MQENC_INTEGER_REVERSED"          ,          2 },
   {  "MQENC_INTEGER_UNDEFINED"         ,          0 },
   {  "MQENC_NATIVE"                    ,        546 },
   {  "MQENC_NORMAL"                    ,        273 },
   {  "MQENC_RESERVED_MASK"             ,      -4096 },
   {  "MQENC_REVERSED"                  ,        546 },
   {  "MQENC_S390"                      ,        785 },
   {  "MQENC_TNS"                       ,       1041 },
   {  "MQEPH_CCSID_EMBEDDED"            ,          1 },
   {  "MQEPH_CURRENT_LENGTH"            ,         68 },
   {  "MQEPH_CURRENT_VERSION"           ,          1 },
   {  "MQEPH_LENGTH_1"                  ,         68 },
   {  "MQEPH_NONE"                      ,          0 },
   {  "MQEPH_STRUC_LENGTH_FIXED"        ,         68 },
   {  "MQEPH_VERSION_1"                 ,          1 },
   {  "MQET_MQSC"                       ,          1 },
   {  "MQEVO_CONSOLE"                   ,          1 },
   {  "MQEVO_CTLMSG"                    ,          7 },
   {  "MQEVO_INIT"                      ,          2 },
   {  "MQEVO_INTERNAL"                  ,          5 },
   {  "MQEVO_MQSET"                     ,          4 },
   {  "MQEVO_MQSUB"                     ,          6 },
   {  "MQEVO_MSG"                       ,          3 },
   {  "MQEVO_OTHER"                     ,          0 },
   {  "MQEVO_REST"                      ,          8 },
   {  "MQEVR_ADMIN_ONLY"                ,          5 },
   {  "MQEVR_API_ONLY"                  ,          4 },
   {  "MQEVR_DISABLED"                  ,          0 },
   {  "MQEVR_ENABLED"                   ,          1 },
   {  "MQEVR_EXCEPTION"                 ,          2 },
   {  "MQEVR_NO_DISPLAY"                ,          3 },
   {  "MQEVR_USER_ONLY"                 ,          6 },
   {  "MQEXPI_OFF"                      ,          0 },
   {  "MQEXTATTRS_ALL"                  ,          0 },
   {  "MQEXTATTRS_NONDEF"               ,          1 },
   {  "MQEXT_ALL"                       ,          0 },
   {  "MQEXT_AUTHORITY"                 ,          2 },
   {  "MQEXT_OBJECT"                    ,          1 },
   {  "MQFB_ACTIVITY"                   ,        269 },
   {  "MQFB_APPL_CANNOT_BE_STARTED"     ,        265 },
   {  "MQFB_APPL_FIRST"                 ,      65536 },
   {  "MQFB_APPL_LAST"                  ,  999999999 },
   {  "MQFB_APPL_TYPE_ERROR"            ,        267 },
   {  "MQFB_BIND_OPEN_CLUSRCVR_DEL"     ,        281 },
   {  "MQFB_BUFFER_OVERFLOW"            ,        294 },
   {  "MQFB_CHANNEL_COMPLETED"          ,        262 },
   {  "MQFB_CHANNEL_FAIL"               ,        264 },
   {  "MQFB_CHANNEL_FAIL_RETRY"         ,        263 },
   {  "MQFB_CICS_APPL_ABENDED"          ,        411 },
   {  "MQFB_CICS_APPL_NOT_STARTED"      ,        410 },
   {  "MQFB_CICS_BRIDGE_FAILURE"        ,        403 },
   {  "MQFB_CICS_CCSID_ERROR"           ,        405 },
   {  "MQFB_CICS_CIH_ERROR"             ,        407 },
   {  "MQFB_CICS_COMMAREA_ERROR"        ,        409 },
   {  "MQFB_CICS_CORREL_ID_ERROR"       ,        404 },
   {  "MQFB_CICS_DLQ_ERROR"             ,        412 },
   {  "MQFB_CICS_ENCODING_ERROR"        ,        406 },
   {  "MQFB_CICS_INTERNAL_ERROR"        ,        401 },
   {  "MQFB_CICS_NOT_AUTHORIZED"        ,        402 },
   {  "MQFB_CICS_UOW_BACKED_OUT"        ,        413 },
   {  "MQFB_CICS_UOW_ERROR"             ,        408 },
   {  "MQFB_COA"                        ,        259 },
   {  "MQFB_COD"                        ,        260 },
   {  "MQFB_DATA_LENGTH_NEGATIVE"       ,        292 },
   {  "MQFB_DATA_LENGTH_TOO_BIG"        ,        293 },
   {  "MQFB_DATA_LENGTH_ZERO"           ,        291 },
   {  "MQFB_EXPIRATION"                 ,        258 },
   {  "MQFB_IIH_ERROR"                  ,        296 },
   {  "MQFB_IMS_ERROR"                  ,        300 },
   {  "MQFB_IMS_FIRST"                  ,        301 },
   {  "MQFB_IMS_LAST"                   ,        399 },
   {  "MQFB_IMS_NACK_1A_REASON_FIRST"   ,        600 },
   {  "MQFB_IMS_NACK_1A_REASON_LAST"    ,        855 },
   {  "MQFB_LENGTH_OFF_BY_ONE"          ,        295 },
   {  "MQFB_MAX_ACTIVITIES"             ,        282 },
   {  "MQFB_MSG_SCOPE_MISMATCH"         ,        503 },
   {  "MQFB_NAN"                        ,        276 },
   {  "MQFB_NONE"                       ,          0 },
   {  "MQFB_NOT_AUTHORIZED_FOR_IMS"     ,        298 },
   {  "MQFB_NOT_A_GROUPUR_MSG"          ,        505 },
   {  "MQFB_NOT_A_REPOSITORY_MSG"       ,        280 },
   {  "MQFB_NOT_DELIVERED"              ,        284 },
   {  "MQFB_NOT_FORWARDED"              ,        283 },
   {  "MQFB_PAN"                        ,        275 },
   {  "MQFB_PUBLICATIONS_ON_REQUEST"    ,        501 },
   {  "MQFB_QUIT"                       ,        256 },
   {  "MQFB_SELECTOR_MISMATCH"          ,        504 },
   {  "MQFB_STOPPED_BY_CHAD_EXIT"       ,        277 },
   {  "MQFB_STOPPED_BY_MSG_EXIT"        ,        268 },
   {  "MQFB_STOPPED_BY_PUBSUB_EXIT"     ,        279 },
   {  "MQFB_SUBSCRIBER_IS_PUBLISHER"    ,        502 },
   {  "MQFB_SYSTEM_FIRST"               ,          1 },
   {  "MQFB_SYSTEM_LAST"                ,      65535 },
   {  "MQFB_TM_ERROR"                   ,        266 },
   {  "MQFB_UNSUPPORTED_DELIVERY"       ,        286 },
   {  "MQFB_UNSUPPORTED_FORWARDING"     ,        285 },
   {  "MQFB_XMIT_Q_MSG_ERROR"           ,        271 },
   {  "MQFC_NO"                         ,          0 },
   {  "MQFC_YES"                        ,          1 },
   {  "MQFIELD_WQR_CLWLQueuePriority"   ,       8013 },
   {  "MQFIELD_WQR_CLWLQueueRank"       ,       8014 },
   {  "MQFIELD_WQR_ClusterRecOffset"    ,       8006 },
   {  "MQFIELD_WQR_DefBind"             ,       8009 },
   {  "MQFIELD_WQR_DefPersistence"      ,       8010 },
   {  "MQFIELD_WQR_DefPriority"         ,       8011 },
   {  "MQFIELD_WQR_DefPutResponse"      ,       8015 },
   {  "MQFIELD_WQR_InhibitPut"          ,       8012 },
   {  "MQFIELD_WQR_QDesc"               ,       8008 },
   {  "MQFIELD_WQR_QFlags"              ,       8003 },
   {  "MQFIELD_WQR_QMgrIdentifier"      ,       8005 },
   {  "MQFIELD_WQR_QName"               ,       8004 },
   {  "MQFIELD_WQR_QType"               ,       8007 },
   {  "MQFIELD_WQR_StrucId"             ,       8000 },
   {  "MQFIELD_WQR_StrucLength"         ,       8002 },
   {  "MQFIELD_WQR_Version"             ,       8001 },
   {  "MQFUN_TYPE_COMMAND"              ,          5 },
   {  "MQFUN_TYPE_JVM"                  ,          1 },
   {  "MQFUN_TYPE_PROCEDURE"            ,          3 },
   {  "MQFUN_TYPE_PROGRAM"              ,          2 },
   {  "MQFUN_TYPE_UNKNOWN"              ,          0 },
   {  "MQFUN_TYPE_USERDEF"              ,          4 },
   {  "MQGACF_ACTIVITY"                 ,       8005 },
   {  "MQGACF_ACTIVITY_TRACE"           ,       8013 },
   {  "MQGACF_APP_DIST_LIST"            ,       8014 },
   {  "MQGACF_CHL_STATISTICS_DATA"      ,       8012 },
   {  "MQGACF_COMMAND_CONTEXT"          ,       8001 },
   {  "MQGACF_COMMAND_DATA"             ,       8002 },
   {  "MQGACF_EMBEDDED_MQMD"            ,       8006 },
   {  "MQGACF_FIRST"                    ,       8001 },
   {  "MQGACF_LAST_USED"                ,       8017 },
   {  "MQGACF_MESSAGE"                  ,       8007 },
   {  "MQGACF_MONITOR_CLASS"            ,       8015 },
   {  "MQGACF_MONITOR_ELEMENT"          ,       8017 },
   {  "MQGACF_MONITOR_TYPE"             ,       8016 },
   {  "MQGACF_MQMD"                     ,       8008 },
   {  "MQGACF_OPERATION"                ,       8004 },
   {  "MQGACF_Q_ACCOUNTING_DATA"        ,       8010 },
   {  "MQGACF_Q_STATISTICS_DATA"        ,       8011 },
   {  "MQGACF_TRACE_ROUTE"              ,       8003 },
   {  "MQGACF_VALUE_NAMING"             ,       8009 },
   {  "MQGA_FIRST"                      ,       8001 },
   {  "MQGA_LAST"                       ,       9000 },
   {  "MQGMO_ACCEPT_TRUNCATED_MSG"      ,         64 },
   {  "MQGMO_ALL_MSGS_AVAILABLE"        ,     131072 },
   {  "MQGMO_ALL_SEGMENTS_AVAILABLE"    ,     262144 },
   {  "MQGMO_BROWSE_CO_OP"              ,   18874384 },
   {  "MQGMO_BROWSE_FIRST"              ,         16 },
   {  "MQGMO_BROWSE_HANDLE"             ,   17825808 },
   {  "MQGMO_BROWSE_MSG_UNDER_CURSOR"   ,       2048 },
   {  "MQGMO_BROWSE_NEXT"               ,         32 },
   {  "MQGMO_COMPLETE_MSG"              ,      65536 },
   {  "MQGMO_CONVERT"                   ,      16384 },
   {  "MQGMO_CURRENT_LENGTH"            ,        112 },
   {  "MQGMO_CURRENT_VERSION"           ,          4 },
   {  "MQGMO_FAIL_IF_QUIESCING"         ,       8192 },
   {  "MQGMO_LENGTH_1"                  ,         72 },
   {  "MQGMO_LENGTH_2"                  ,         80 },
   {  "MQGMO_LENGTH_3"                  ,        100 },
   {  "MQGMO_LENGTH_4"                  ,        112 },
   {  "MQGMO_LOCK"                      ,        512 },
   {  "MQGMO_LOGICAL_ORDER"             ,      32768 },
   {  "MQGMO_MARK_BROWSE_CO_OP"         ,    2097152 },
   {  "MQGMO_MARK_BROWSE_HANDLE"        ,    1048576 },
   {  "MQGMO_MARK_SKIP_BACKOUT"         ,        128 },
   {  "MQGMO_MSG_UNDER_CURSOR"          ,        256 },
   {  "MQGMO_NONE"                      ,          0 },
   {  "MQGMO_NO_PROPERTIES"             ,   67108864 },
   {  "MQGMO_NO_SYNCPOINT"              ,          4 },
   {  "MQGMO_NO_WAIT"                   ,          0 },
   {  "MQGMO_PROPERTIES_AS_Q_DEF"       ,          0 },
   {  "MQGMO_PROPERTIES_COMPATIBILITY"  ,  268435456 },
   {  "MQGMO_PROPERTIES_FORCE_MQRFH2"   ,   33554432 },
   {  "MQGMO_PROPERTIES_IN_HANDLE"      ,  134217728 },
   {  "MQGMO_SET_SIGNAL"                ,          8 },
   {  "MQGMO_SYNCPOINT"                 ,          2 },
   {  "MQGMO_SYNCPOINT_IF_PERSISTENT"   ,       4096 },
   {  "MQGMO_UNLOCK"                    ,       1024 },
   {  "MQGMO_UNMARKED_BROWSE_MSG"       ,   16777216 },
   {  "MQGMO_UNMARK_BROWSE_CO_OP"       ,    4194304 },
   {  "MQGMO_UNMARK_BROWSE_HANDLE"      ,    8388608 },
   {  "MQGMO_VERSION_1"                 ,          1 },
   {  "MQGMO_VERSION_2"                 ,          2 },
   {  "MQGMO_VERSION_3"                 ,          3 },
   {  "MQGMO_VERSION_4"                 ,          4 },
   {  "MQGMO_WAIT"                      ,          1 },
   {  "MQGUR_DISABLED"                  ,          0 },
   {  "MQGUR_ENABLED"                   ,          1 },
   {  "MQHA_BAG_HANDLE"                 ,       4001 },
   {  "MQHA_FIRST"                      ,       4001 },
   {  "MQHA_LAST"                       ,       6000 },
   {  "MQHA_LAST_USED"                  ,       4001 },
   {  "MQHB_NONE"                       ,         -2 },
   {  "MQHB_UNUSABLE_HBAG"              ,         -1 },
   {  "MQHC_DEF_HCONN"                  ,          0 },
   {  "MQHC_UNASSOCIATED_HCONN"         ,         -3 },
   {  "MQHC_UNUSABLE_HCONN"             ,         -1 },
   {  "MQHM_NONE"                       ,          0 },
   {  "MQHM_UNUSABLE_HMSG"              ,         -1 },
   {  "MQHO_NONE"                       ,          0 },
   {  "MQHO_UNUSABLE_HOBJ"              ,         -1 },
   {  "MQHSTATE_ACTIVE"                 ,          1 },
   {  "MQHSTATE_INACTIVE"               ,          0 },
   {  "MQIACF_ACTION"                   ,       1086 },
   {  "MQIACF_ALL"                      ,       1009 },
   {  "MQIACF_AMQP_ATTRS"               ,       1401 },
   {  "MQIACF_AMQP_DIAGNOSTICS_TYPE"    ,       1406 },
   {  "MQIACF_ANONYMOUS_COUNT"          ,       1090 },
   {  "MQIACF_API_CALLER_TYPE"          ,       1357 },
   {  "MQIACF_API_ENVIRONMENT"          ,       1358 },
   {  "MQIACF_APPL_COUNT"               ,       1089 },
   {  "MQIACF_APPL_FUNCTION_TYPE"       ,       1400 },
   {  "MQIACF_ARCHIVE_LOG_SIZE"         ,       1416 },
   {  "MQIACF_ASYNC_STATE"              ,       1308 },
   {  "MQIACF_AUTHORIZATION_LIST"       ,       1115 },
   {  "MQIACF_AUTH_ADD_AUTHS"           ,       1116 },
   {  "MQIACF_AUTH_INFO_ATTRS"          ,       1019 },
   {  "MQIACF_AUTH_OPTIONS"             ,       1228 },
   {  "MQIACF_AUTH_PROFILE_ATTRS"       ,       1114 },
   {  "MQIACF_AUTH_REC_TYPE"            ,       1412 },
   {  "MQIACF_AUTH_REMOVE_AUTHS"        ,       1117 },
   {  "MQIACF_AUTH_SERVICE_ATTRS"       ,       1264 },
   {  "MQIACF_AUX_ERROR_DATA_INT_1"     ,       1070 },
   {  "MQIACF_AUX_ERROR_DATA_INT_2"     ,       1071 },
   {  "MQIACF_BACKOUT_COUNT"            ,       1241 },
   {  "MQIACF_BRIDGE_TYPE"              ,       1073 },
   {  "MQIACF_BROKER_COUNT"             ,       1088 },
   {  "MQIACF_BROKER_OPTIONS"           ,       1077 },
   {  "MQIACF_BUFFER_LENGTH"            ,       1374 },
   {  "MQIACF_BUFFER_POOL_ID"           ,       1158 },
   {  "MQIACF_BUFFER_POOL_LOCATION"     ,       1408 },
   {  "MQIACF_CALL_TYPE"                ,       1361 },
   {  "MQIACF_CF_SMDS_BLOCK_SIZE"       ,       1328 },
   {  "MQIACF_CF_SMDS_EXPAND"           ,       1329 },
   {  "MQIACF_CF_STATUS_BACKUP"         ,       1138 },
   {  "MQIACF_CF_STATUS_CONNECT"        ,       1137 },
   {  "MQIACF_CF_STATUS_SMDS"           ,       1333 },
   {  "MQIACF_CF_STATUS_SUMMARY"        ,       1136 },
   {  "MQIACF_CF_STATUS_TYPE"           ,       1135 },
   {  "MQIACF_CF_STRUC_ACCESS"          ,       1332 },
   {  "MQIACF_CF_STRUC_ATTRS"           ,       1133 },
   {  "MQIACF_CF_STRUC_BACKUP_SIZE"     ,       1144 },
   {  "MQIACF_CF_STRUC_ENTRIES_MAX"     ,       1142 },
   {  "MQIACF_CF_STRUC_ENTRIES_USED"    ,       1143 },
   {  "MQIACF_CF_STRUC_SIZE_MAX"        ,       1140 },
   {  "MQIACF_CF_STRUC_SIZE_USED"       ,       1141 },
   {  "MQIACF_CF_STRUC_STATUS"          ,       1130 },
   {  "MQIACF_CF_STRUC_TYPE"            ,       1139 },
   {  "MQIACF_CHANNEL_ATTRS"            ,       1015 },
   {  "MQIACF_CHINIT_STATUS"            ,       1232 },
   {  "MQIACF_CHLAUTH_ATTRS"            ,       1355 },
   {  "MQIACF_CHLAUTH_TYPE"             ,       1352 },
   {  "MQIACF_CLEAR_SCOPE"              ,       1306 },
   {  "MQIACF_CLEAR_TYPE"               ,       1305 },
   {  "MQIACF_CLOSE_OPTIONS"            ,       1365 },
   {  "MQIACF_CLUSTER_INFO"             ,       1083 },
   {  "MQIACF_CLUSTER_Q_MGR_ATTRS"      ,       1093 },
   {  "MQIACF_CMDSCOPE_Q_MGR_COUNT"     ,       1121 },
   {  "MQIACF_CMD_SERVER_STATUS"        ,       1233 },
   {  "MQIACF_COMMAND"                  ,       1021 },
   {  "MQIACF_COMMAND_INFO"             ,       1120 },
   {  "MQIACF_COMM_INFO_ATTRS"          ,       1327 },
   {  "MQIACF_COMP_CODE"                ,       1242 },
   {  "MQIACF_CONFIGURATION_EVENTS"     ,       1174 },
   {  "MQIACF_CONFIGURATION_OBJECTS"    ,       1173 },
   {  "MQIACF_CONNECTION_ATTRS"         ,       1107 },
   {  "MQIACF_CONNECTION_COUNT"         ,       1230 },
   {  "MQIACF_CONNECTION_SWAP"          ,       1405 },
   {  "MQIACF_CONNECT_OPTIONS"          ,       1108 },
   {  "MQIACF_CONNECT_TIME"             ,       1380 },
   {  "MQIACF_CONN_INFO_ALL"            ,       1113 },
   {  "MQIACF_CONN_INFO_CONN"           ,       1111 },
   {  "MQIACF_CONN_INFO_HANDLE"         ,       1112 },
   {  "MQIACF_CONN_INFO_TYPE"           ,       1110 },
   {  "MQIACF_CONV_REASON_CODE"         ,       1072 },
   {  "MQIACF_CTL_OPERATION"            ,       1366 },
   {  "MQIACF_DB2_CONN_STATUS"          ,       1150 },
   {  "MQIACF_DELETE_OPTIONS"           ,       1092 },
   {  "MQIACF_DESTINATION_CLASS"        ,       1273 },
   {  "MQIACF_DISCONNECT_TIME"          ,       1381 },
   {  "MQIACF_DISCONTINUITY_COUNT"      ,       1237 },
   {  "MQIACF_DURABLE_SUBSCRIPTION"     ,       1274 },
   {  "MQIACF_ENCODING"                 ,       1243 },
   {  "MQIACF_ENTITY_TYPE"              ,       1118 },
   {  "MQIACF_ERROR_ID"                 ,       1013 },
   {  "MQIACF_ERROR_IDENTIFIER"         ,       1013 },
   {  "MQIACF_ERROR_OFFSET"             ,       1018 },
   {  "MQIACF_ESCAPE_TYPE"              ,       1017 },
   {  "MQIACF_EVENT_APPL_TYPE"          ,       1010 },
   {  "MQIACF_EVENT_ORIGIN"             ,       1011 },
   {  "MQIACF_EXCLUDE_INTERVAL"         ,       1134 },
   {  "MQIACF_EXPIRY"                   ,       1244 },
   {  "MQIACF_EXPIRY_Q_COUNT"           ,       1172 },
   {  "MQIACF_EXPIRY_TIME"              ,       1379 },
   {  "MQIACF_EXPORT_ATTRS"             ,       1403 },
   {  "MQIACF_EXPORT_TYPE"              ,       1402 },
   {  "MQIACF_FEEDBACK"                 ,       1245 },
   {  "MQIACF_FIRST"                    ,       1001 },
   {  "MQIACF_FORCE"                    ,       1005 },
   {  "MQIACF_GET_OPTIONS"              ,       1367 },
   {  "MQIACF_GROUPUR_CHECK_ID"         ,       1323 },
   {  "MQIACF_HANDLE_STATE"             ,       1028 },
   {  "MQIACF_HOBJ"                     ,       1360 },
   {  "MQIACF_HSUB"                     ,       1382 },
   {  "MQIACF_IGNORE_STATE"             ,       1423 },
   {  "MQIACF_INQUIRY"                  ,       1074 },
   {  "MQIACF_INTATTR_COUNT"            ,       1393 },
   {  "MQIACF_INTEGER_DATA"             ,       1080 },
   {  "MQIACF_INTERFACE_VERSION"        ,       1263 },
   {  "MQIACF_INT_ATTRS"                ,       1394 },
   {  "MQIACF_INVALID_DEST_COUNT"       ,       1371 },
   {  "MQIACF_ITEM_COUNT"               ,       1378 },
   {  "MQIACF_KNOWN_DEST_COUNT"         ,       1369 },
   {  "MQIACF_LAST_USED"                ,       1423 },
   {  "MQIACF_LDAP_CONNECTION_STATUS"   ,       1409 },
   {  "MQIACF_LISTENER_ATTRS"           ,       1222 },
   {  "MQIACF_LISTENER_STATUS_ATTRS"    ,       1223 },
   {  "MQIACF_LOG_COMPRESSION"          ,       1322 },
   {  "MQIACF_LOG_IN_USE"               ,       1420 },
   {  "MQIACF_LOG_REDUCTION"            ,       1422 },
   {  "MQIACF_LOG_UTILIZATION"          ,       1421 },
   {  "MQIACF_MAX_ACTIVITIES"           ,       1236 },
   {  "MQIACF_MCAST_REL_INDICATOR"      ,       1351 },
   {  "MQIACF_MEDIA_LOG_SIZE"           ,       1417 },
   {  "MQIACF_MESSAGE_COUNT"            ,       1290 },
   {  "MQIACF_MODE"                     ,       1008 },
   {  "MQIACF_MONITORING"               ,       1258 },
   {  "MQIACF_MOVE_COUNT"               ,       1171 },
   {  "MQIACF_MOVE_TYPE"                ,       1145 },
   {  "MQIACF_MOVE_TYPE_ADD"            ,       1147 },
   {  "MQIACF_MOVE_TYPE_MOVE"           ,       1146 },
   {  "MQIACF_MQCB_OPERATION"           ,       1362 },
   {  "MQIACF_MQCB_OPTIONS"             ,       1364 },
   {  "MQIACF_MQCB_TYPE"                ,       1363 },
   {  "MQIACF_MQXR_DIAGNOSTICS_TYPE"    ,       1354 },
   {  "MQIACF_MSG_FLAGS"                ,       1247 },
   {  "MQIACF_MSG_LENGTH"               ,       1248 },
   {  "MQIACF_MSG_TYPE"                 ,       1249 },
   {  "MQIACF_MULC_CAPTURE"             ,       1324 },
   {  "MQIACF_NAMELIST_ATTRS"           ,       1004 },
   {  "MQIACF_NUM_PUBS"                 ,       1396 },
   {  "MQIACF_OBJECT_TYPE"              ,       1016 },
   {  "MQIACF_OBSOLETE_MSGS"            ,       1310 },
   {  "MQIACF_OFFSET"                   ,       1250 },
   {  "MQIACF_OLDEST_MSG_AGE"           ,       1227 },
   {  "MQIACF_OPEN_BROWSE"              ,       1102 },
   {  "MQIACF_OPEN_INPUT_TYPE"          ,       1098 },
   {  "MQIACF_OPEN_INQUIRE"             ,       1101 },
   {  "MQIACF_OPEN_OPTIONS"             ,       1022 },
   {  "MQIACF_OPEN_OUTPUT"              ,       1099 },
   {  "MQIACF_OPEN_SET"                 ,       1100 },
   {  "MQIACF_OPEN_TYPE"                ,       1023 },
   {  "MQIACF_OPERATION_ID"             ,       1356 },
   {  "MQIACF_OPERATION_MODE"           ,       1326 },
   {  "MQIACF_OPERATION_TYPE"           ,       1240 },
   {  "MQIACF_OPTIONS"                  ,       1076 },
   {  "MQIACF_ORIGINAL_LENGTH"          ,       1251 },
   {  "MQIACF_PAGECLAS"                 ,       1411 },
   {  "MQIACF_PAGESET_STATUS"           ,       1165 },
   {  "MQIACF_PARAMETER_ID"             ,       1012 },
   {  "MQIACF_PERMIT_STANDBY"           ,       1325 },
   {  "MQIACF_PERSISTENCE"              ,       1252 },
   {  "MQIACF_POINTER_SIZE"             ,       1397 },
   {  "MQIACF_PRIORITY"                 ,       1253 },
   {  "MQIACF_PROCESS_ATTRS"            ,       1003 },
   {  "MQIACF_PROCESS_ID"               ,       1024 },
   {  "MQIACF_PS_STATUS_TYPE"           ,       1314 },
   {  "MQIACF_PUBLICATION_OPTIONS"      ,       1082 },
   {  "MQIACF_PUBLISH_COUNT"            ,       1304 },
   {  "MQIACF_PUBSUB_PROPERTIES"        ,       1271 },
   {  "MQIACF_PUBSUB_STATUS"            ,       1311 },
   {  "MQIACF_PUBSUB_STATUS_ATTRS"      ,       1318 },
   {  "MQIACF_PUB_PRIORITY"             ,       1283 },
   {  "MQIACF_PURGE"                    ,       1007 },
   {  "MQIACF_PUT_OPTIONS"              ,       1373 },
   {  "MQIACF_QSG_DISPS"                ,       1126 },
   {  "MQIACF_QUIESCE"                  ,       1008 },
   {  "MQIACF_Q_ATTRS"                  ,       1002 },
   {  "MQIACF_Q_HANDLE"                 ,       1104 },
   {  "MQIACF_Q_MGR_ATTRS"              ,       1001 },
   {  "MQIACF_Q_MGR_CLUSTER"            ,       1125 },
   {  "MQIACF_Q_MGR_DEFINITION_TYPE"    ,       1084 },
   {  "MQIACF_Q_MGR_DQM"                ,       1124 },
   {  "MQIACF_Q_MGR_EVENT"              ,       1123 },
   {  "MQIACF_Q_MGR_FACILITY"           ,       1231 },
   {  "MQIACF_Q_MGR_NUMBER"             ,       1148 },
   {  "MQIACF_Q_MGR_PUBSUB"             ,       1291 },
   {  "MQIACF_Q_MGR_STATUS"             ,       1149 },
   {  "MQIACF_Q_MGR_STATUS_ATTRS"       ,       1229 },
   {  "MQIACF_Q_MGR_STATUS_LOG"         ,       1415 },
   {  "MQIACF_Q_MGR_SYSTEM"             ,       1122 },
   {  "MQIACF_Q_MGR_TYPE"               ,       1085 },
   {  "MQIACF_Q_MGR_VERSION"            ,       1292 },
   {  "MQIACF_Q_STATUS"                 ,       1105 },
   {  "MQIACF_Q_STATUS_ATTRS"           ,       1026 },
   {  "MQIACF_Q_STATUS_TYPE"            ,       1103 },
   {  "MQIACF_Q_TIME_INDICATOR"         ,       1226 },
   {  "MQIACF_Q_TYPES"                  ,       1261 },
   {  "MQIACF_REASON_CODE"              ,       1254 },
   {  "MQIACF_REASON_QUALIFIER"         ,       1020 },
   {  "MQIACF_RECORDED_ACTIVITIES"      ,       1235 },
   {  "MQIACF_RECS_PRESENT"             ,       1368 },
   {  "MQIACF_REFRESH_INTERVAL"         ,       1094 },
   {  "MQIACF_REFRESH_REPOSITORY"       ,       1095 },
   {  "MQIACF_REFRESH_TYPE"             ,       1078 },
   {  "MQIACF_REGISTRATION_OPTIONS"     ,       1081 },
   {  "MQIACF_REG_REG_OPTIONS"          ,       1091 },
   {  "MQIACF_REMOVE_AUTHREC"           ,       1398 },
   {  "MQIACF_REMOVE_QUEUES"            ,       1096 },
   {  "MQIACF_REPLACE"                  ,       1006 },
   {  "MQIACF_REPORT"                   ,       1255 },
   {  "MQIACF_REQUEST_ONLY"             ,       1280 },
   {  "MQIACF_RESOLVED_TYPE"            ,       1372 },
   {  "MQIACF_RESTART_LOG_SIZE"         ,       1418 },
   {  "MQIACF_RETAINED_PUBLICATION"     ,       1300 },
   {  "MQIACF_REUSABLE_LOG_SIZE"        ,       1419 },
   {  "MQIACF_ROUTE_ACCUMULATION"       ,       1238 },
   {  "MQIACF_ROUTE_DELIVERY"           ,       1239 },
   {  "MQIACF_ROUTE_DETAIL"             ,       1234 },
   {  "MQIACF_ROUTE_FORWARDING"         ,       1259 },
   {  "MQIACF_SECURITY_ATTRS"           ,       1151 },
   {  "MQIACF_SECURITY_INTERVAL"        ,       1153 },
   {  "MQIACF_SECURITY_ITEM"            ,       1129 },
   {  "MQIACF_SECURITY_SETTING"         ,       1155 },
   {  "MQIACF_SECURITY_SWITCH"          ,       1154 },
   {  "MQIACF_SECURITY_TIMEOUT"         ,       1152 },
   {  "MQIACF_SECURITY_TYPE"            ,       1106 },
   {  "MQIACF_SELECTOR"                 ,       1014 },
   {  "MQIACF_SELECTORS"                ,       1392 },
   {  "MQIACF_SELECTOR_COUNT"           ,       1391 },
   {  "MQIACF_SELECTOR_TYPE"            ,       1321 },
   {  "MQIACF_SEQUENCE_NUMBER"          ,       1079 },
   {  "MQIACF_SERVICE_ATTRS"            ,       1224 },
   {  "MQIACF_SERVICE_STATUS"           ,       1260 },
   {  "MQIACF_SERVICE_STATUS_ATTRS"     ,       1225 },
   {  "MQIACF_SMDS_ATTRS"               ,       1334 },
   {  "MQIACF_SMDS_AVAIL"               ,       1350 },
   {  "MQIACF_SMDS_EXPANDST"            ,       1376 },
   {  "MQIACF_SMDS_OPENMODE"            ,       1348 },
   {  "MQIACF_SMDS_STATUS"              ,       1349 },
   {  "MQIACF_STATUS_TYPE"              ,       1389 },
   {  "MQIACF_STORAGE_CLASS_ATTRS"      ,       1156 },
   {  "MQIACF_STRUC_LENGTH"             ,       1377 },
   {  "MQIACF_SUBRQ_ACTION"             ,       1395 },
   {  "MQIACF_SUBRQ_OPTIONS"            ,       1383 },
   {  "MQIACF_SUBSCRIPTION_SCOPE"       ,       1275 },
   {  "MQIACF_SUB_ATTRS"                ,       1287 },
   {  "MQIACF_SUB_LEVEL"                ,       1307 },
   {  "MQIACF_SUB_OPTIONS"              ,       1303 },
   {  "MQIACF_SUB_STATUS_ATTRS"         ,       1294 },
   {  "MQIACF_SUB_SUMMARY"              ,       1309 },
   {  "MQIACF_SUB_TYPE"                 ,       1289 },
   {  "MQIACF_SUSPEND"                  ,       1087 },
   {  "MQIACF_SYSP_ALLOC_PRIMARY"       ,       1209 },
   {  "MQIACF_SYSP_ALLOC_SECONDARY"     ,       1210 },
   {  "MQIACF_SYSP_ALLOC_UNIT"          ,       1203 },
   {  "MQIACF_SYSP_ARCHIVE"             ,       1182 },
   {  "MQIACF_SYSP_ARCHIVE_RETAIN"      ,       1204 },
   {  "MQIACF_SYSP_ARCHIVE_WTOR"        ,       1205 },
   {  "MQIACF_SYSP_BLOCK_SIZE"          ,       1206 },
   {  "MQIACF_SYSP_CATALOG"             ,       1207 },
   {  "MQIACF_SYSP_CHKPOINT_COUNT"      ,       1191 },
   {  "MQIACF_SYSP_CLUSTER_CACHE"       ,       1266 },
   {  "MQIACF_SYSP_COMPACT"             ,       1208 },
   {  "MQIACF_SYSP_DB2_BLOB_TASKS"      ,       1267 },
   {  "MQIACF_SYSP_DB2_TASKS"           ,       1194 },
   {  "MQIACF_SYSP_DEALLOC_INTERVAL"    ,       1176 },
   {  "MQIACF_SYSP_DUAL_ACTIVE"         ,       1183 },
   {  "MQIACF_SYSP_DUAL_ARCHIVE"        ,       1184 },
   {  "MQIACF_SYSP_DUAL_BSDS"           ,       1185 },
   {  "MQIACF_SYSP_EXIT_INTERVAL"       ,       1189 },
   {  "MQIACF_SYSP_EXIT_TASKS"          ,       1190 },
   {  "MQIACF_SYSP_FULL_LOGS"           ,       1221 },
   {  "MQIACF_SYSP_IN_BUFFER_SIZE"      ,       1179 },
   {  "MQIACF_SYSP_LOG_COPY"            ,       1216 },
   {  "MQIACF_SYSP_LOG_SUSPEND"         ,       1218 },
   {  "MQIACF_SYSP_LOG_USED"            ,       1217 },
   {  "MQIACF_SYSP_MAX_ACE_POOL"        ,       1410 },
   {  "MQIACF_SYSP_MAX_ARCHIVE"         ,       1177 },
   {  "MQIACF_SYSP_MAX_CONC_OFFLOADS"   ,       1413 },
   {  "MQIACF_SYSP_MAX_CONNS"           ,       1186 },
   {  "MQIACF_SYSP_MAX_CONNS_BACK"      ,       1188 },
   {  "MQIACF_SYSP_MAX_CONNS_FORE"      ,       1187 },
   {  "MQIACF_SYSP_MAX_READ_TAPES"      ,       1178 },
   {  "MQIACF_SYSP_OFFLOAD_STATUS"      ,       1219 },
   {  "MQIACF_SYSP_OTMA_INTERVAL"       ,       1192 },
   {  "MQIACF_SYSP_OUT_BUFFER_COUNT"    ,       1181 },
   {  "MQIACF_SYSP_OUT_BUFFER_SIZE"     ,       1180 },
   {  "MQIACF_SYSP_PROTECT"             ,       1211 },
   {  "MQIACF_SYSP_QUIESCE_INTERVAL"    ,       1212 },
   {  "MQIACF_SYSP_Q_INDEX_DEFER"       ,       1193 },
   {  "MQIACF_SYSP_RESLEVEL_AUDIT"      ,       1195 },
   {  "MQIACF_SYSP_ROUTING_CODE"        ,       1196 },
   {  "MQIACF_SYSP_SMF_ACCOUNTING"      ,       1197 },
   {  "MQIACF_SYSP_SMF_INTERVAL"        ,       1199 },
   {  "MQIACF_SYSP_SMF_STATS"           ,       1198 },
   {  "MQIACF_SYSP_TIMESTAMP"           ,       1213 },
   {  "MQIACF_SYSP_TOTAL_LOGS"          ,       1220 },
   {  "MQIACF_SYSP_TRACE_CLASS"         ,       1200 },
   {  "MQIACF_SYSP_TRACE_SIZE"          ,       1201 },
   {  "MQIACF_SYSP_TYPE"                ,       1175 },
   {  "MQIACF_SYSP_UNIT_ADDRESS"        ,       1214 },
   {  "MQIACF_SYSP_UNIT_STATUS"         ,       1215 },
   {  "MQIACF_SYSP_WLM_INTERVAL"        ,       1202 },
   {  "MQIACF_SYSP_WLM_INT_UNITS"       ,       1268 },
   {  "MQIACF_SYSP_ZHYPERWRITE"         ,       1414 },
   {  "MQIACF_SYSTEM_OBJECTS"           ,       1404 },
   {  "MQIACF_THREAD_ID"                ,       1025 },
   {  "MQIACF_TOPIC_ATTRS"              ,       1269 },
   {  "MQIACF_TOPIC_PUB"                ,       1297 },
   {  "MQIACF_TOPIC_STATUS"             ,       1295 },
   {  "MQIACF_TOPIC_STATUS_ATTRS"       ,       1301 },
   {  "MQIACF_TOPIC_STATUS_TYPE"        ,       1302 },
   {  "MQIACF_TOPIC_SUB"                ,       1296 },
   {  "MQIACF_TRACE_DATA_LENGTH"        ,       1375 },
   {  "MQIACF_TRACE_DETAIL"             ,       1359 },
   {  "MQIACF_UNCOMMITTED_MSGS"         ,       1027 },
   {  "MQIACF_UNKNOWN_DEST_COUNT"       ,       1370 },
   {  "MQIACF_UNRECORDED_ACTIVITIES"    ,       1257 },
   {  "MQIACF_UOW_STATE"                ,       1128 },
   {  "MQIACF_UOW_TYPE"                 ,       1132 },
   {  "MQIACF_USAGE_BLOCK_SIZE"         ,       1336 },
   {  "MQIACF_USAGE_BUFFER_POOL"        ,       1170 },
   {  "MQIACF_USAGE_DATA_BLOCKS"        ,       1337 },
   {  "MQIACF_USAGE_DATA_SET"           ,       1169 },
   {  "MQIACF_USAGE_DATA_SET_TYPE"      ,       1167 },
   {  "MQIACF_USAGE_EMPTY_BUFFERS"      ,       1338 },
   {  "MQIACF_USAGE_EXPAND_COUNT"       ,       1164 },
   {  "MQIACF_USAGE_EXPAND_TYPE"        ,       1265 },
   {  "MQIACF_USAGE_FREE_BUFF"          ,       1330 },
   {  "MQIACF_USAGE_FREE_BUFF_PERC"     ,       1331 },
   {  "MQIACF_USAGE_INUSE_BUFFERS"      ,       1339 },
   {  "MQIACF_USAGE_LOWEST_FREE"        ,       1340 },
   {  "MQIACF_USAGE_NONPERSIST_PAGES"   ,       1162 },
   {  "MQIACF_USAGE_OFFLOAD_MSGS"       ,       1341 },
   {  "MQIACF_USAGE_PAGESET"            ,       1168 },
   {  "MQIACF_USAGE_PERSIST_PAGES"      ,       1161 },
   {  "MQIACF_USAGE_READS_SAVED"        ,       1342 },
   {  "MQIACF_USAGE_RESTART_EXTENTS"    ,       1163 },
   {  "MQIACF_USAGE_SAVED_BUFFERS"      ,       1343 },
   {  "MQIACF_USAGE_SMDS"               ,       1335 },
   {  "MQIACF_USAGE_TOTAL_BLOCKS"       ,       1344 },
   {  "MQIACF_USAGE_TOTAL_BUFFERS"      ,       1166 },
   {  "MQIACF_USAGE_TOTAL_PAGES"        ,       1159 },
   {  "MQIACF_USAGE_TYPE"               ,       1157 },
   {  "MQIACF_USAGE_UNUSED_PAGES"       ,       1160 },
   {  "MQIACF_USAGE_USED_BLOCKS"        ,       1345 },
   {  "MQIACF_USAGE_USED_RATE"          ,       1346 },
   {  "MQIACF_USAGE_WAIT_RATE"          ,       1347 },
   {  "MQIACF_USER_ID_SUPPORT"          ,       1262 },
   {  "MQIACF_VARIABLE_USER_ID"         ,       1277 },
   {  "MQIACF_VERSION"                  ,       1256 },
   {  "MQIACF_WAIT_INTERVAL"            ,       1075 },
   {  "MQIACF_WILDCARD_SCHEMA"          ,       1288 },
   {  "MQIACF_XA_COUNT"                 ,       1390 },
   {  "MQIACF_XA_FLAGS"                 ,       1385 },
   {  "MQIACF_XA_HANDLE"                ,       1387 },
   {  "MQIACF_XA_RETCODE"               ,       1386 },
   {  "MQIACF_XA_RETVAL"                ,       1388 },
   {  "MQIACF_XA_RMID"                  ,       1384 },
   {  "MQIACF_XR_ATTRS"                 ,       1399 },
   {  "MQIACH_ACTIVE_CHL"               ,       1593 },
   {  "MQIACH_ACTIVE_CHL_MAX"           ,       1594 },
   {  "MQIACH_ACTIVE_CHL_PAUSED"        ,       1595 },
   {  "MQIACH_ACTIVE_CHL_RETRY"         ,       1598 },
   {  "MQIACH_ACTIVE_CHL_STARTED"       ,       1596 },
   {  "MQIACH_ACTIVE_CHL_STOPPED"       ,       1597 },
   {  "MQIACH_ADAPS_MAX"                ,       1584 },
   {  "MQIACH_ADAPS_STARTED"            ,       1583 },
   {  "MQIACH_ADAPTER"                  ,       1519 },
   {  "MQIACH_ALLOC_FAST_TIMER"         ,       1571 },
   {  "MQIACH_ALLOC_RETRY"              ,       1570 },
   {  "MQIACH_ALLOC_SLOW_TIMER"         ,       1572 },
   {  "MQIACH_AMQP_KEEP_ALIVE"          ,       1644 },
   {  "MQIACH_AVAILABLE_CIPHERSPECS"    ,       1636 },
   {  "MQIACH_BACKLOG"                  ,       1602 },
   {  "MQIACH_BATCHES"                  ,       1537 },
   {  "MQIACH_BATCH_DATA_LIMIT"         ,       1624 },
   {  "MQIACH_BATCH_HB"                 ,       1567 },
   {  "MQIACH_BATCH_INTERVAL"           ,       1564 },
   {  "MQIACH_BATCH_SIZE"               ,       1502 },
   {  "MQIACH_BATCH_SIZE_INDICATOR"     ,       1607 },
   {  "MQIACH_BUFFERS_RCVD"             ,       1539 },
   {  "MQIACH_BUFFERS_RECEIVED"         ,       1539 },
   {  "MQIACH_BUFFERS_SENT"             ,       1538 },
   {  "MQIACH_BYTES_RCVD"               ,       1536 },
   {  "MQIACH_BYTES_RECEIVED"           ,       1536 },
   {  "MQIACH_BYTES_SENT"               ,       1535 },
   {  "MQIACH_CHANNEL_DISP"             ,       1580 },
   {  "MQIACH_CHANNEL_ERROR_DATA"       ,       1525 },
   {  "MQIACH_CHANNEL_INSTANCE_ATTRS"   ,       1524 },
   {  "MQIACH_CHANNEL_INSTANCE_TYPE"    ,       1523 },
   {  "MQIACH_CHANNEL_STATUS"           ,       1527 },
   {  "MQIACH_CHANNEL_SUBSTATE"         ,       1609 },
   {  "MQIACH_CHANNEL_SUMMARY_ATTRS"    ,       1642 },
   {  "MQIACH_CHANNEL_TABLE"            ,       1526 },
   {  "MQIACH_CHANNEL_TYPE"             ,       1511 },
   {  "MQIACH_CHANNEL_TYPES"            ,       1582 },
   {  "MQIACH_CLIENT_CHANNEL_WEIGHT"    ,       1620 },
   {  "MQIACH_CLWL_CHANNEL_PRIORITY"    ,       1578 },
   {  "MQIACH_CLWL_CHANNEL_RANK"        ,       1577 },
   {  "MQIACH_CLWL_CHANNEL_WEIGHT"      ,       1579 },
   {  "MQIACH_COMMAND_COUNT"            ,       1520 },
   {  "MQIACH_COMPRESSION_RATE"         ,       1611 },
   {  "MQIACH_COMPRESSION_TIME"         ,       1612 },
   {  "MQIACH_CONNECTION_AFFINITY"      ,       1621 },
   {  "MQIACH_CURRENT_CHL"              ,       1589 },
   {  "MQIACH_CURRENT_CHL_LU62"         ,       1592 },
   {  "MQIACH_CURRENT_CHL_MAX"          ,       1590 },
   {  "MQIACH_CURRENT_CHL_TCP"          ,       1591 },
   {  "MQIACH_CURRENT_MSGS"             ,       1531 },
   {  "MQIACH_CURRENT_SEQUENCE_NUMBER"  ,       1532 },
   {  "MQIACH_CURRENT_SEQ_NUMBER"       ,       1532 },
   {  "MQIACH_CURRENT_SHARING_CONVS"    ,       1617 },
   {  "MQIACH_DATA_CONVERSION"          ,       1515 },
   {  "MQIACH_DATA_COUNT"               ,       1512 },
   {  "MQIACH_DEF_CHANNEL_DISP"         ,       1614 },
   {  "MQIACH_DEF_RECONNECT"            ,       1640 },
   {  "MQIACH_DISC_INTERVAL"            ,       1503 },
   {  "MQIACH_DISC_RETRY"               ,       1573 },
   {  "MQIACH_DISPS_MAX"                ,       1586 },
   {  "MQIACH_DISPS_STARTED"            ,       1585 },
   {  "MQIACH_EXIT_TIME_INDICATOR"      ,       1606 },
   {  "MQIACH_FIRST"                    ,       1501 },
   {  "MQIACH_HB_INTERVAL"              ,       1563 },
   {  "MQIACH_HDR_COMPRESSION"          ,       1575 },
   {  "MQIACH_INBOUND_DISP"             ,       1581 },
   {  "MQIACH_INDOUBT_STATUS"           ,       1528 },
   {  "MQIACH_IN_DOUBT"                 ,       1516 },
   {  "MQIACH_IN_DOUBT_IN"              ,       1631 },
   {  "MQIACH_IN_DOUBT_OUT"             ,       1632 },
   {  "MQIACH_KEEP_ALIVE_INTERVAL"      ,       1566 },
   {  "MQIACH_LAST_SEQUENCE_NUMBER"     ,       1529 },
   {  "MQIACH_LAST_SEQ_NUMBER"          ,       1529 },
   {  "MQIACH_LAST_USED"                ,       1645 },
   {  "MQIACH_LISTENER_CONTROL"         ,       1601 },
   {  "MQIACH_LISTENER_STATUS"          ,       1599 },
   {  "MQIACH_LONG_RETRIES_LEFT"        ,       1540 },
   {  "MQIACH_LONG_RETRY"               ,       1507 },
   {  "MQIACH_LONG_TIMER"               ,       1506 },
   {  "MQIACH_MATCH"                    ,       1637 },
   {  "MQIACH_MAX_INSTANCES"            ,       1618 },
   {  "MQIACH_MAX_INSTS_PER_CLIENT"     ,       1619 },
   {  "MQIACH_MAX_MSG_LENGTH"           ,       1510 },
   {  "MQIACH_MAX_SHARING_CONVS"        ,       1616 },
   {  "MQIACH_MAX_XMIT_SIZE"            ,       1613 },
   {  "MQIACH_MCA_STATUS"               ,       1542 },
   {  "MQIACH_MCA_TYPE"                 ,       1517 },
   {  "MQIACH_MC_HB_INTERVAL"           ,       1628 },
   {  "MQIACH_MQTT_KEEP_ALIVE"          ,       1630 },
   {  "MQIACH_MR_COUNT"                 ,       1544 },
   {  "MQIACH_MR_INTERVAL"              ,       1545 },
   {  "MQIACH_MSGS"                     ,       1534 },
   {  "MQIACH_MSGS_RCVD"                ,       1634 },
   {  "MQIACH_MSGS_RECEIVED"            ,       1634 },
   {  "MQIACH_MSGS_SENT"                ,       1633 },
   {  "MQIACH_MSG_COMPRESSION"          ,       1576 },
   {  "MQIACH_MSG_HISTORY"              ,       1625 },
   {  "MQIACH_MSG_SEQUENCE_NUMBER"      ,       1514 },
   {  "MQIACH_MULTICAST_PROPERTIES"     ,       1626 },
   {  "MQIACH_NAME_COUNT"               ,       1513 },
   {  "MQIACH_NETWORK_PRIORITY"         ,       1565 },
   {  "MQIACH_NETWORK_TIME_INDICATOR"   ,       1605 },
   {  "MQIACH_NEW_SUBSCRIBER_HISTORY"   ,       1627 },
   {  "MQIACH_NPM_SPEED"                ,       1562 },
   {  "MQIACH_PENDING_OUT"              ,       1635 },
   {  "MQIACH_PORT"                     ,       1522 },
   {  "MQIACH_PORT_NUMBER"              ,       1574 },
   {  "MQIACH_PROTOCOL"                 ,       1643 },
   {  "MQIACH_PUT_AUTHORITY"            ,       1508 },
   {  "MQIACH_RESET_REQUESTED"          ,       1623 },
   {  "MQIACH_SECURITY_PROTOCOL"        ,       1645 },
   {  "MQIACH_SEQUENCE_NUMBER_WRAP"     ,       1509 },
   {  "MQIACH_SESSION_COUNT"            ,       1518 },
   {  "MQIACH_SHARED_CHL_RESTART"       ,       1600 },
   {  "MQIACH_SHARING_CONVERSATIONS"    ,       1615 },
   {  "MQIACH_SHORT_RETRIES_LEFT"       ,       1541 },
   {  "MQIACH_SHORT_RETRY"              ,       1505 },
   {  "MQIACH_SHORT_TIMER"              ,       1504 },
   {  "MQIACH_SOCKET"                   ,       1521 },
   {  "MQIACH_SSLTASKS_MAX"             ,       1588 },
   {  "MQIACH_SSLTASKS_STARTED"         ,       1587 },
   {  "MQIACH_SSL_CLIENT_AUTH"          ,       1568 },
   {  "MQIACH_SSL_KEY_RESETS"           ,       1610 },
   {  "MQIACH_SSL_RETURN_CODE"          ,       1533 },
   {  "MQIACH_STOP_REQUESTED"           ,       1543 },
   {  "MQIACH_USER_SOURCE"              ,       1638 },
   {  "MQIACH_USE_CLIENT_ID"            ,       1629 },
   {  "MQIACH_WARNING"                  ,       1639 },
   {  "MQIACH_XMITQ_MSGS_AVAILABLE"     ,       1608 },
   {  "MQIACH_XMITQ_TIME_INDICATOR"     ,       1604 },
   {  "MQIACH_XMIT_PROTOCOL_TYPE"       ,       1501 },
   {  "MQIAMO64_AVG_Q_TIME"             ,        703 },
   {  "MQIAMO64_BROWSE_BYTES"           ,        745 },
   {  "MQIAMO64_BYTES"                  ,        746 },
   {  "MQIAMO64_GET_BYTES"              ,        747 },
   {  "MQIAMO64_HIGHRES_TIME"           ,        838 },
   {  "MQIAMO64_MONITOR_INTERVAL"       ,        845 },
   {  "MQIAMO64_PUBLISH_MSG_BYTES"      ,        785 },
   {  "MQIAMO64_PUT_BYTES"              ,        748 },
   {  "MQIAMO64_QMGR_OP_DURATION"       ,        844 },
   {  "MQIAMO64_Q_TIME_AVG"             ,        741 },
   {  "MQIAMO64_Q_TIME_MAX"             ,        742 },
   {  "MQIAMO64_Q_TIME_MIN"             ,        743 },
   {  "MQIAMO64_TOPIC_PUT_BYTES"        ,        783 },
   {  "MQIAMO_ACKS_RCVD"                ,        806 },
   {  "MQIAMO_ACK_FEEDBACK"             ,        814 },
   {  "MQIAMO_ACTIVE_ACKERS"            ,        807 },
   {  "MQIAMO_AVG_BATCH_SIZE"           ,        702 },
   {  "MQIAMO_AVG_Q_TIME"               ,        703 },
   {  "MQIAMO_BACKOUTS"                 ,        704 },
   {  "MQIAMO_BROWSES"                  ,        705 },
   {  "MQIAMO_BROWSES_FAILED"           ,        708 },
   {  "MQIAMO_BROWSE_MAX_BYTES"         ,        706 },
   {  "MQIAMO_BROWSE_MIN_BYTES"         ,        707 },
   {  "MQIAMO_BYTES_SENT"               ,        791 },
   {  "MQIAMO_CBS"                      ,        769 },
   {  "MQIAMO_CBS_FAILED"               ,        770 },
   {  "MQIAMO_CLOSES"                   ,        709 },
   {  "MQIAMO_CLOSES_FAILED"            ,        757 },
   {  "MQIAMO_COMMITS"                  ,        710 },
   {  "MQIAMO_COMMITS_FAILED"           ,        711 },
   {  "MQIAMO_CONNS"                    ,        712 },
   {  "MQIAMO_CONNS_FAILED"             ,        749 },
   {  "MQIAMO_CONNS_MAX"                ,        713 },
   {  "MQIAMO_CTLS"                     ,        771 },
   {  "MQIAMO_CTLS_FAILED"              ,        772 },
   {  "MQIAMO_DEST_DATA_PORT"           ,        804 },
   {  "MQIAMO_DEST_REPAIR_PORT"         ,        805 },
   {  "MQIAMO_DISCS"                    ,        714 },
   {  "MQIAMO_DISCS_IMPLICIT"           ,        715 },
   {  "MQIAMO_DISC_TYPE"                ,        716 },
   {  "MQIAMO_EXIT_TIME_AVG"            ,        717 },
   {  "MQIAMO_EXIT_TIME_MAX"            ,        718 },
   {  "MQIAMO_EXIT_TIME_MIN"            ,        719 },
   {  "MQIAMO_FEEDBACK_MODE"            ,        793 },
   {  "MQIAMO_FIRST"                    ,        701 },
   {  "MQIAMO_FULL_BATCHES"             ,        720 },
   {  "MQIAMO_GENERATED_MSGS"           ,        721 },
   {  "MQIAMO_GETS"                     ,        722 },
   {  "MQIAMO_GETS_FAILED"              ,        725 },
   {  "MQIAMO_GET_MAX_BYTES"            ,        723 },
   {  "MQIAMO_GET_MIN_BYTES"            ,        724 },
   {  "MQIAMO_HISTORY_PKTS"             ,        798 },
   {  "MQIAMO_INCOMPLETE_BATCHES"       ,        726 },
   {  "MQIAMO_INQS"                     ,        727 },
   {  "MQIAMO_INQS_FAILED"              ,        752 },
   {  "MQIAMO_INTERVAL"                 ,        789 },
   {  "MQIAMO_LAST_USED"                ,        845 },
   {  "MQIAMO_LATE_JOIN_MARK"           ,        795 },
   {  "MQIAMO_MCAST_BATCH_TIME"         ,        802 },
   {  "MQIAMO_MCAST_HEARTBEAT"          ,        803 },
   {  "MQIAMO_MCAST_XMIT_RATE"          ,        801 },
   {  "MQIAMO_MONITOR_CLASS"            ,        839 },
   {  "MQIAMO_MONITOR_DATATYPE"         ,        842 },
   {  "MQIAMO_MONITOR_DELTA"            ,          2 },
   {  "MQIAMO_MONITOR_ELEMENT"          ,        841 },
   {  "MQIAMO_MONITOR_FLAGS"            ,        843 },
   {  "MQIAMO_MONITOR_FLAGS_NONE"       ,          0 },
   {  "MQIAMO_MONITOR_FLAGS_OBJNAME"    ,          1 },
   {  "MQIAMO_MONITOR_GB"               ,  100000000 },
   {  "MQIAMO_MONITOR_HUNDREDTHS"       ,        100 },
   {  "MQIAMO_MONITOR_KB"               ,       1024 },
   {  "MQIAMO_MONITOR_MB"               ,    1048576 },
   {  "MQIAMO_MONITOR_MICROSEC"         ,    1000000 },
   {  "MQIAMO_MONITOR_PERCENT"          ,      10000 },
   {  "MQIAMO_MONITOR_TYPE"             ,        840 },
   {  "MQIAMO_MONITOR_UNIT"             ,          1 },
   {  "MQIAMO_MSGS"                     ,        728 },
   {  "MQIAMO_MSGS_DELIVERED"           ,        819 },
   {  "MQIAMO_MSGS_EXPIRED"             ,        758 },
   {  "MQIAMO_MSGS_NOT_QUEUED"          ,        759 },
   {  "MQIAMO_MSGS_PURGED"              ,        760 },
   {  "MQIAMO_MSGS_RCVD"                ,        817 },
   {  "MQIAMO_MSGS_SENT"                ,        790 },
   {  "MQIAMO_MSG_BYTES_RCVD"           ,        818 },
   {  "MQIAMO_NACKS_CREATED"            ,        824 },
   {  "MQIAMO_NACKS_RCVD"               ,        796 },
   {  "MQIAMO_NACK_FEEDBACK"            ,        815 },
   {  "MQIAMO_NACK_PKTS_SENT"           ,        825 },
   {  "MQIAMO_NET_TIME_AVG"             ,        729 },
   {  "MQIAMO_NET_TIME_MAX"             ,        730 },
   {  "MQIAMO_NET_TIME_MIN"             ,        731 },
   {  "MQIAMO_NUM_STREAMS"              ,        813 },
   {  "MQIAMO_OBJECT_COUNT"             ,        732 },
   {  "MQIAMO_OPENS"                    ,        733 },
   {  "MQIAMO_OPENS_FAILED"             ,        751 },
   {  "MQIAMO_PENDING_PKTS"             ,        799 },
   {  "MQIAMO_PKTS_DELIVERED"           ,        821 },
   {  "MQIAMO_PKTS_DROPPED"             ,        822 },
   {  "MQIAMO_PKTS_DUPLICATED"          ,        823 },
   {  "MQIAMO_PKTS_LOST"                ,        816 },
   {  "MQIAMO_PKTS_PROCESSED"           ,        820 },
   {  "MQIAMO_PKTS_REPAIRED"            ,        828 },
   {  "MQIAMO_PKTS_SENT"                ,        808 },
   {  "MQIAMO_PKT_RATE"                 ,        800 },
   {  "MQIAMO_PUBLISH_MSG_COUNT"        ,        784 },
   {  "MQIAMO_PUT1S"                    ,        734 },
   {  "MQIAMO_PUT1S_FAILED"             ,        755 },
   {  "MQIAMO_PUTS"                     ,        735 },
   {  "MQIAMO_PUTS_FAILED"              ,        754 },
   {  "MQIAMO_PUT_MAX_BYTES"            ,        736 },
   {  "MQIAMO_PUT_MIN_BYTES"            ,        737 },
   {  "MQIAMO_PUT_RETRIES"              ,        738 },
   {  "MQIAMO_Q_MAX_DEPTH"              ,        739 },
   {  "MQIAMO_Q_MIN_DEPTH"              ,        740 },
   {  "MQIAMO_Q_TIME_AVG"               ,        741 },
   {  "MQIAMO_Q_TIME_MAX"               ,        742 },
   {  "MQIAMO_Q_TIME_MIN"               ,        743 },
   {  "MQIAMO_RELIABILITY_TYPE"         ,        794 },
   {  "MQIAMO_REPAIR_BYTES"             ,        792 },
   {  "MQIAMO_REPAIR_PKTS"              ,        797 },
   {  "MQIAMO_REPAIR_PKTS_RCVD"         ,        827 },
   {  "MQIAMO_REPAIR_PKTS_RQSTD"        ,        826 },
   {  "MQIAMO_SETS"                     ,        744 },
   {  "MQIAMO_SETS_FAILED"              ,        753 },
   {  "MQIAMO_STATS"                    ,        773 },
   {  "MQIAMO_STATS_FAILED"             ,        774 },
   {  "MQIAMO_SUBRQS"                   ,        767 },
   {  "MQIAMO_SUBRQS_FAILED"            ,        768 },
   {  "MQIAMO_SUBS_DUR"                 ,        764 },
   {  "MQIAMO_SUBS_FAILED"              ,        766 },
   {  "MQIAMO_SUBS_NDUR"                ,        765 },
   {  "MQIAMO_SUB_DUR_HIGHWATER"        ,        775 },
   {  "MQIAMO_SUB_DUR_LOWWATER"         ,        776 },
   {  "MQIAMO_SUB_NDUR_HIGHWATER"       ,        777 },
   {  "MQIAMO_SUB_NDUR_LOWWATER"        ,        778 },
   {  "MQIAMO_TOPIC_PUT1S"              ,        781 },
   {  "MQIAMO_TOPIC_PUT1S_FAILED"       ,        782 },
   {  "MQIAMO_TOPIC_PUTS"               ,        779 },
   {  "MQIAMO_TOPIC_PUTS_FAILED"        ,        780 },
   {  "MQIAMO_TOTAL_BYTES_SENT"         ,        812 },
   {  "MQIAMO_TOTAL_MSGS_DELIVERED"     ,        836 },
   {  "MQIAMO_TOTAL_MSGS_EXPIRED"       ,        835 },
   {  "MQIAMO_TOTAL_MSGS_PROCESSED"     ,        833 },
   {  "MQIAMO_TOTAL_MSGS_RCVD"          ,        829 },
   {  "MQIAMO_TOTAL_MSGS_RETURNED"      ,        837 },
   {  "MQIAMO_TOTAL_MSGS_SELECTED"      ,        834 },
   {  "MQIAMO_TOTAL_MSGS_SENT"          ,        811 },
   {  "MQIAMO_TOTAL_MSG_BYTES_RCVD"     ,        830 },
   {  "MQIAMO_TOTAL_PKTS_SENT"          ,        810 },
   {  "MQIAMO_TOTAL_REPAIR_PKTS"        ,        809 },
   {  "MQIAMO_TOTAL_REPAIR_PKTS_RCVD"   ,        831 },
   {  "MQIAMO_TOTAL_REPAIR_PKTS_RQSTD"  ,        832 },
   {  "MQIAMO_UNSUBS_DUR"               ,        786 },
   {  "MQIAMO_UNSUBS_FAILED"            ,        788 },
   {  "MQIAMO_UNSUBS_NDUR"              ,        787 },
   {  "MQIASY_BAG_OPTIONS"              ,         -8 },
   {  "MQIASY_CODED_CHAR_SET_ID"        ,         -1 },
   {  "MQIASY_COMMAND"                  ,         -3 },
   {  "MQIASY_COMP_CODE"                ,         -6 },
   {  "MQIASY_CONTROL"                  ,         -5 },
   {  "MQIASY_FIRST"                    ,         -1 },
   {  "MQIASY_LAST"                     ,      -2000 },
   {  "MQIASY_LAST_USED"                ,         -9 },
   {  "MQIASY_MSG_SEQ_NUMBER"           ,         -4 },
   {  "MQIASY_REASON"                   ,         -7 },
   {  "MQIASY_TYPE"                     ,         -2 },
   {  "MQIASY_VERSION"                  ,         -9 },
   {  "MQIAV_NOT_APPLICABLE"            ,         -1 },
   {  "MQIAV_UNDEFINED"                 ,         -2 },
   {  "MQIA_ACCOUNTING_CONN_OVERRIDE"   ,        136 },
   {  "MQIA_ACCOUNTING_INTERVAL"        ,        135 },
   {  "MQIA_ACCOUNTING_MQI"             ,        133 },
   {  "MQIA_ACCOUNTING_Q"               ,        134 },
   {  "MQIA_ACTIVE_CHANNELS"            ,        100 },
   {  "MQIA_ACTIVITY_CONN_OVERRIDE"     ,        239 },
   {  "MQIA_ACTIVITY_RECORDING"         ,        138 },
   {  "MQIA_ACTIVITY_TRACE"             ,        240 },
   {  "MQIA_ADOPTNEWMCA_CHECK"          ,        102 },
   {  "MQIA_ADOPTNEWMCA_INTERVAL"       ,        104 },
   {  "MQIA_ADOPTNEWMCA_TYPE"           ,        103 },
   {  "MQIA_ADOPT_CONTEXT"              ,        260 },
   {  "MQIA_ADVANCED_CAPABILITY"        ,        273 },
   {  "MQIA_AMQP_CAPABILITY"            ,        265 },
   {  "MQIA_APPL_TYPE"                  ,          1 },
   {  "MQIA_ARCHIVE"                    ,         60 },
   {  "MQIA_AUTHENTICATION_FAIL_DELAY"  ,        259 },
   {  "MQIA_AUTHENTICATION_METHOD"      ,        266 },
   {  "MQIA_AUTHORITY_EVENT"            ,         47 },
   {  "MQIA_AUTH_INFO_TYPE"             ,         66 },
   {  "MQIA_AUTO_REORGANIZATION"        ,        173 },
   {  "MQIA_AUTO_REORG_INTERVAL"        ,        174 },
   {  "MQIA_BACKOUT_THRESHOLD"          ,         22 },
   {  "MQIA_BASE_TYPE"                  ,        193 },
   {  "MQIA_BATCH_INTERFACE_AUTO"       ,         86 },
   {  "MQIA_BRIDGE_EVENT"               ,         74 },
   {  "MQIA_CERT_VAL_POLICY"            ,        252 },
   {  "MQIA_CF_CFCONLOS"                ,        246 },
   {  "MQIA_CF_LEVEL"                   ,         70 },
   {  "MQIA_CF_OFFLDUSE"                ,        229 },
   {  "MQIA_CF_OFFLOAD"                 ,        224 },
   {  "MQIA_CF_OFFLOAD_THRESHOLD1"      ,        225 },
   {  "MQIA_CF_OFFLOAD_THRESHOLD2"      ,        226 },
   {  "MQIA_CF_OFFLOAD_THRESHOLD3"      ,        227 },
   {  "MQIA_CF_RECAUTO"                 ,        244 },
   {  "MQIA_CF_RECOVER"                 ,         71 },
   {  "MQIA_CF_SMDS_BUFFERS"            ,        228 },
   {  "MQIA_CHANNEL_AUTO_DEF"           ,         55 },
   {  "MQIA_CHANNEL_AUTO_DEF_EVENT"     ,         56 },
   {  "MQIA_CHANNEL_EVENT"              ,         73 },
   {  "MQIA_CHECK_CLIENT_BINDING"       ,        258 },
   {  "MQIA_CHECK_LOCAL_BINDING"        ,        257 },
   {  "MQIA_CHINIT_ADAPTERS"            ,        101 },
   {  "MQIA_CHINIT_CONTROL"             ,        119 },
   {  "MQIA_CHINIT_DISPATCHERS"         ,        105 },
   {  "MQIA_CHINIT_TRACE_AUTO_START"    ,        117 },
   {  "MQIA_CHINIT_TRACE_TABLE_SIZE"    ,        118 },
   {  "MQIA_CHLAUTH_RECORDS"            ,        248 },
   {  "MQIA_CLUSTER_OBJECT_STATE"       ,        256 },
   {  "MQIA_CLUSTER_PUB_ROUTE"          ,        255 },
   {  "MQIA_CLUSTER_Q_TYPE"             ,         59 },
   {  "MQIA_CLUSTER_WORKLOAD_LENGTH"    ,         58 },
   {  "MQIA_CLWL_MRU_CHANNELS"          ,         97 },
   {  "MQIA_CLWL_Q_PRIORITY"            ,         96 },
   {  "MQIA_CLWL_Q_RANK"                ,         95 },
   {  "MQIA_CLWL_USEQ"                  ,         98 },
   {  "MQIA_CMD_SERVER_AUTO"            ,         87 },
   {  "MQIA_CMD_SERVER_CONTROL"         ,        120 },
   {  "MQIA_CMD_SERVER_CONVERT_MSG"     ,         88 },
   {  "MQIA_CMD_SERVER_DLQ_MSG"         ,         89 },
   {  "MQIA_CODED_CHAR_SET_ID"          ,          2 },
   {  "MQIA_COMMAND_EVENT"              ,         99 },
   {  "MQIA_COMMAND_LEVEL"              ,         31 },
   {  "MQIA_COMM_EVENT"                 ,        232 },
   {  "MQIA_COMM_INFO_TYPE"             ,        223 },
   {  "MQIA_CONFIGURATION_EVENT"        ,         51 },
   {  "MQIA_CPI_LEVEL"                  ,         27 },
   {  "MQIA_CURRENT_Q_DEPTH"            ,          3 },
   {  "MQIA_DEFINITION_TYPE"            ,          7 },
   {  "MQIA_DEF_BIND"                   ,         61 },
   {  "MQIA_DEF_CLUSTER_XMIT_Q_TYPE"    ,        250 },
   {  "MQIA_DEF_INPUT_OPEN_OPTION"      ,          4 },
   {  "MQIA_DEF_PERSISTENCE"            ,          5 },
   {  "MQIA_DEF_PRIORITY"               ,          6 },
   {  "MQIA_DEF_PUT_RESPONSE_TYPE"      ,        184 },
   {  "MQIA_DEF_READ_AHEAD"             ,        188 },
   {  "MQIA_DISPLAY_TYPE"               ,        262 },
   {  "MQIA_DIST_LISTS"                 ,         34 },
   {  "MQIA_DNS_WLM"                    ,        106 },
   {  "MQIA_DURABLE_SUB"                ,        175 },
   {  "MQIA_ENCRYPTION_ALGORITHM"       ,        237 },
   {  "MQIA_EXPIRY_INTERVAL"            ,         39 },
   {  "MQIA_FIRST"                      ,          1 },
   {  "MQIA_GROUP_UR"                   ,        221 },
   {  "MQIA_HARDEN_GET_BACKOUT"         ,          8 },
   {  "MQIA_HIGH_Q_DEPTH"               ,         36 },
   {  "MQIA_IGQ_PUT_AUTHORITY"          ,         65 },
   {  "MQIA_INDEX_TYPE"                 ,         57 },
   {  "MQIA_INHIBIT_EVENT"              ,         48 },
   {  "MQIA_INHIBIT_GET"                ,          9 },
   {  "MQIA_INHIBIT_PUB"                ,        181 },
   {  "MQIA_INHIBIT_PUT"                ,         10 },
   {  "MQIA_INHIBIT_SUB"                ,        182 },
   {  "MQIA_INTRA_GROUP_QUEUING"        ,         64 },
   {  "MQIA_IP_ADDRESS_VERSION"         ,         93 },
   {  "MQIA_KEY_REUSE_COUNT"            ,        267 },
   {  "MQIA_LAST"                       ,       2000 },
   {  "MQIA_LAST_USED"                  ,        273 },
   {  "MQIA_LDAP_AUTHORMD"              ,        263 },
   {  "MQIA_LDAP_NESTGRP"               ,        264 },
   {  "MQIA_LDAP_SECURE_COMM"           ,        261 },
   {  "MQIA_LISTENER_PORT_NUMBER"       ,         85 },
   {  "MQIA_LISTENER_TIMER"             ,        107 },
   {  "MQIA_LOCAL_EVENT"                ,         49 },
   {  "MQIA_LOGGER_EVENT"               ,         94 },
   {  "MQIA_LU62_CHANNELS"              ,        108 },
   {  "MQIA_MASTER_ADMIN"               ,        186 },
   {  "MQIA_MAX_CHANNELS"               ,        109 },
   {  "MQIA_MAX_CLIENTS"                ,        172 },
   {  "MQIA_MAX_GLOBAL_LOCKS"           ,         83 },
   {  "MQIA_MAX_HANDLES"                ,         11 },
   {  "MQIA_MAX_LOCAL_LOCKS"            ,         84 },
   {  "MQIA_MAX_MSG_LENGTH"             ,         13 },
   {  "MQIA_MAX_OPEN_Q"                 ,         80 },
   {  "MQIA_MAX_PRIORITY"               ,         14 },
   {  "MQIA_MAX_PROPERTIES_LENGTH"      ,        192 },
   {  "MQIA_MAX_Q_DEPTH"                ,         15 },
   {  "MQIA_MAX_Q_TRIGGERS"             ,         90 },
   {  "MQIA_MAX_RECOVERY_TASKS"         ,        171 },
   {  "MQIA_MAX_RESPONSES"              ,        230 },
   {  "MQIA_MAX_UNCOMMITTED_MSGS"       ,         33 },
   {  "MQIA_MCAST_BRIDGE"               ,        233 },
   {  "MQIA_MEDIA_IMAGE_INTERVAL"       ,        269 },
   {  "MQIA_MEDIA_IMAGE_LOG_LENGTH"     ,        270 },
   {  "MQIA_MEDIA_IMAGE_RECOVER_OBJ"    ,        271 },
   {  "MQIA_MEDIA_IMAGE_RECOVER_Q"      ,        272 },
   {  "MQIA_MEDIA_IMAGE_SCHEDULING"     ,        268 },
   {  "MQIA_MONITORING_AUTO_CLUSSDR"    ,        124 },
   {  "MQIA_MONITORING_CHANNEL"         ,        122 },
   {  "MQIA_MONITORING_Q"               ,        123 },
   {  "MQIA_MONITOR_INTERVAL"           ,         81 },
   {  "MQIA_MSG_DELIVERY_SEQUENCE"      ,         16 },
   {  "MQIA_MSG_DEQ_COUNT"              ,         38 },
   {  "MQIA_MSG_ENQ_COUNT"              ,         37 },
   {  "MQIA_MSG_MARK_BROWSE_INTERVAL"   ,         68 },
   {  "MQIA_MULTICAST"                  ,        176 },
   {  "MQIA_NAMELIST_TYPE"              ,         72 },
   {  "MQIA_NAME_COUNT"                 ,         19 },
   {  "MQIA_NPM_CLASS"                  ,         78 },
   {  "MQIA_NPM_DELIVERY"               ,        196 },
   {  "MQIA_OPEN_INPUT_COUNT"           ,         17 },
   {  "MQIA_OPEN_OUTPUT_COUNT"          ,         18 },
   {  "MQIA_OUTBOUND_PORT_MAX"          ,        140 },
   {  "MQIA_OUTBOUND_PORT_MIN"          ,        110 },
   {  "MQIA_PAGESET_ID"                 ,         62 },
   {  "MQIA_PERFORMANCE_EVENT"          ,         53 },
   {  "MQIA_PLATFORM"                   ,         32 },
   {  "MQIA_PM_DELIVERY"                ,        195 },
   {  "MQIA_POLICY_VERSION"             ,        238 },
   {  "MQIA_PROPERTY_CONTROL"           ,        190 },
   {  "MQIA_PROT_POLICY_CAPABILITY"     ,        251 },
   {  "MQIA_PROXY_SUB"                  ,        199 },
   {  "MQIA_PUBSUB_CLUSTER"             ,        249 },
   {  "MQIA_PUBSUB_MAXMSG_RETRY_COUNT"  ,        206 },
   {  "MQIA_PUBSUB_MODE"                ,        187 },
   {  "MQIA_PUBSUB_NP_MSG"              ,        203 },
   {  "MQIA_PUBSUB_NP_RESP"             ,        205 },
   {  "MQIA_PUBSUB_SYNC_PT"             ,        207 },
   {  "MQIA_PUB_COUNT"                  ,        215 },
   {  "MQIA_PUB_SCOPE"                  ,        219 },
   {  "MQIA_QMGR_CFCONLOS"              ,        245 },
   {  "MQIA_QMOPT_CONS_COMMS_MSGS"      ,        155 },
   {  "MQIA_QMOPT_CONS_CRITICAL_MSGS"   ,        154 },
   {  "MQIA_QMOPT_CONS_ERROR_MSGS"      ,        153 },
   {  "MQIA_QMOPT_CONS_INFO_MSGS"       ,        151 },
   {  "MQIA_QMOPT_CONS_REORG_MSGS"      ,        156 },
   {  "MQIA_QMOPT_CONS_SYSTEM_MSGS"     ,        157 },
   {  "MQIA_QMOPT_CONS_WARNING_MSGS"    ,        152 },
   {  "MQIA_QMOPT_CSMT_ON_ERROR"        ,        150 },
   {  "MQIA_QMOPT_INTERNAL_DUMP"        ,        170 },
   {  "MQIA_QMOPT_LOG_COMMS_MSGS"       ,        162 },
   {  "MQIA_QMOPT_LOG_CRITICAL_MSGS"    ,        161 },
   {  "MQIA_QMOPT_LOG_ERROR_MSGS"       ,        160 },
   {  "MQIA_QMOPT_LOG_INFO_MSGS"        ,        158 },
   {  "MQIA_QMOPT_LOG_REORG_MSGS"       ,        163 },
   {  "MQIA_QMOPT_LOG_SYSTEM_MSGS"      ,        164 },
   {  "MQIA_QMOPT_LOG_WARNING_MSGS"     ,        159 },
   {  "MQIA_QMOPT_TRACE_COMMS"          ,        166 },
   {  "MQIA_QMOPT_TRACE_CONVERSION"     ,        168 },
   {  "MQIA_QMOPT_TRACE_MQI_CALLS"      ,        165 },
   {  "MQIA_QMOPT_TRACE_REORG"          ,        167 },
   {  "MQIA_QMOPT_TRACE_SYSTEM"         ,        169 },
   {  "MQIA_QSG_DISP"                   ,         63 },
   {  "MQIA_Q_DEPTH_HIGH_EVENT"         ,         43 },
   {  "MQIA_Q_DEPTH_HIGH_LIMIT"         ,         40 },
   {  "MQIA_Q_DEPTH_LOW_EVENT"          ,         44 },
   {  "MQIA_Q_DEPTH_LOW_LIMIT"          ,         41 },
   {  "MQIA_Q_DEPTH_MAX_EVENT"          ,         42 },
   {  "MQIA_Q_SERVICE_INTERVAL"         ,         54 },
   {  "MQIA_Q_SERVICE_INTERVAL_EVENT"   ,         46 },
   {  "MQIA_Q_TYPE"                     ,         20 },
   {  "MQIA_Q_USERS"                    ,         82 },
   {  "MQIA_READ_AHEAD"                 ,        189 },
   {  "MQIA_RECEIVE_TIMEOUT"            ,        111 },
   {  "MQIA_RECEIVE_TIMEOUT_MIN"        ,        113 },
   {  "MQIA_RECEIVE_TIMEOUT_TYPE"       ,        112 },
   {  "MQIA_REMOTE_EVENT"               ,         50 },
   {  "MQIA_RESPONSE_RESTART_POINT"     ,        231 },
   {  "MQIA_RETENTION_INTERVAL"         ,         21 },
   {  "MQIA_REVERSE_DNS_LOOKUP"         ,        254 },
   {  "MQIA_SCOPE"                      ,         45 },
   {  "MQIA_SECURITY_CASE"              ,        141 },
   {  "MQIA_SERVICE_CONTROL"            ,        139 },
   {  "MQIA_SERVICE_TYPE"               ,        121 },
   {  "MQIA_SHAREABILITY"               ,         23 },
   {  "MQIA_SHARED_Q_Q_MGR_NAME"        ,         77 },
   {  "MQIA_SIGNATURE_ALGORITHM"        ,        236 },
   {  "MQIA_SSL_EVENT"                  ,         75 },
   {  "MQIA_SSL_FIPS_REQUIRED"          ,         92 },
   {  "MQIA_SSL_RESET_COUNT"            ,         76 },
   {  "MQIA_SSL_TASKS"                  ,         69 },
   {  "MQIA_START_STOP_EVENT"           ,         52 },
   {  "MQIA_STATISTICS_AUTO_CLUSSDR"    ,        130 },
   {  "MQIA_STATISTICS_CHANNEL"         ,        129 },
   {  "MQIA_STATISTICS_INTERVAL"        ,        131 },
   {  "MQIA_STATISTICS_MQI"             ,        127 },
   {  "MQIA_STATISTICS_Q"               ,        128 },
   {  "MQIA_SUB_CONFIGURATION_EVENT"    ,        242 },
   {  "MQIA_SUB_COUNT"                  ,        204 },
   {  "MQIA_SUB_SCOPE"                  ,        218 },
   {  "MQIA_SUITE_B_STRENGTH"           ,        247 },
   {  "MQIA_SYNCPOINT"                  ,         30 },
   {  "MQIA_TCP_CHANNELS"               ,        114 },
   {  "MQIA_TCP_KEEP_ALIVE"             ,        115 },
   {  "MQIA_TCP_STACK_TYPE"             ,        116 },
   {  "MQIA_TIME_SINCE_RESET"           ,         35 },
   {  "MQIA_TOLERATE_UNPROTECTED"       ,        235 },
   {  "MQIA_TOPIC_DEF_PERSISTENCE"      ,        185 },
   {  "MQIA_TOPIC_NODE_COUNT"           ,        253 },
   {  "MQIA_TOPIC_TYPE"                 ,        208 },
   {  "MQIA_TRACE_ROUTE_RECORDING"      ,        137 },
   {  "MQIA_TREE_LIFE_TIME"             ,        183 },
   {  "MQIA_TRIGGER_CONTROL"            ,         24 },
   {  "MQIA_TRIGGER_DEPTH"              ,         29 },
   {  "MQIA_TRIGGER_INTERVAL"           ,         25 },
   {  "MQIA_TRIGGER_MSG_PRIORITY"       ,         26 },
   {  "MQIA_TRIGGER_RESTART"            ,         91 },
   {  "MQIA_TRIGGER_TYPE"               ,         28 },
   {  "MQIA_UR_DISP"                    ,        222 },
   {  "MQIA_USAGE"                      ,         12 },
   {  "MQIA_USER_LIST"                  ,       2000 },
   {  "MQIA_USE_DEAD_LETTER_Q"          ,        234 },
   {  "MQIA_WILDCARD_OPERATION"         ,        216 },
   {  "MQIA_XR_CAPABILITY"              ,        243 },
   {  "MQIDO_BACKOUT"                   ,          2 },
   {  "MQIDO_COMMIT"                    ,          1 },
   {  "MQIEPF_CLIENT_LIBRARY"           ,          0 },
   {  "MQIEPF_LOCAL_LIBRARY"            ,          2 },
   {  "MQIEPF_NONE"                     ,          0 },
   {  "MQIEPF_NON_THREADED_LIBRARY"     ,          0 },
   {  "MQIEPF_THREADED_LIBRARY"         ,          1 },
   {  "MQIEP_CURRENT_LENGTH (4 byte)"   ,        140 },
   {  "MQIEP_CURRENT_LENGTH (8 byte)"   ,        264 },
   {  "MQIEP_CURRENT_VERSION"           ,          1 },
   {  "MQIEP_LENGTH_1 (4 byte)"         ,        140 },
   {  "MQIEP_LENGTH_1 (8 byte)"         ,        264 },
   {  "MQIEP_VERSION_1"                 ,          1 },
   {  "MQIGQPA_ALTERNATE_OR_IGQ"        ,          4 },
   {  "MQIGQPA_CONTEXT"                 ,          2 },
   {  "MQIGQPA_DEFAULT"                 ,          1 },
   {  "MQIGQPA_ONLY_IGQ"                ,          3 },
   {  "MQIGQ_DISABLED"                  ,          0 },
   {  "MQIGQ_ENABLED"                   ,          1 },
   {  "MQIIH_CM0_REQUEST_RESPONSE"      ,         32 },
   {  "MQIIH_CURRENT_LENGTH"            ,         84 },
   {  "MQIIH_CURRENT_VERSION"           ,          1 },
   {  "MQIIH_IGNORE_PURG"               ,         16 },
   {  "MQIIH_LENGTH_1"                  ,         84 },
   {  "MQIIH_NONE"                      ,          0 },
   {  "MQIIH_PASS_EXPIRATION"           ,          1 },
   {  "MQIIH_REPLY_FORMAT_NONE"         ,          8 },
   {  "MQIIH_UNLIMITED_EXPIRATION"      ,          0 },
   {  "MQIIH_VERSION_1"                 ,          1 },
   {  "MQIMGRCOV_AS_Q_MGR"              ,          2 },
   {  "MQIMGRCOV_NO"                    ,          0 },
   {  "MQIMGRCOV_YES"                   ,          1 },
   {  "MQIMPO_CONVERT_TYPE"             ,          2 },
   {  "MQIMPO_CONVERT_VALUE"            ,         32 },
   {  "MQIMPO_CURRENT_LENGTH (4 byte)"  ,         60 },
   {  "MQIMPO_CURRENT_LENGTH (8 byte)"  ,         64 },
   {  "MQIMPO_CURRENT_VERSION"          ,          1 },
   {  "MQIMPO_INQ_FIRST"                ,          0 },
   {  "MQIMPO_INQ_NEXT"                 ,          8 },
   {  "MQIMPO_INQ_PROP_UNDER_CURSOR"    ,         16 },
   {  "MQIMPO_LENGTH_1 (4 byte)"        ,         60 },
   {  "MQIMPO_LENGTH_1 (8 byte)"        ,         64 },
   {  "MQIMPO_NONE"                     ,          0 },
   {  "MQIMPO_QUERY_LENGTH"             ,          4 },
   {  "MQIMPO_VERSION_1"                ,          1 },
   {  "MQINBD_GROUP"                    ,          3 },
   {  "MQINBD_Q_MGR"                    ,          0 },
   {  "MQIND_ALL"                       ,         -2 },
   {  "MQIND_NONE"                      ,         -1 },
   {  "MQIPADDR_IPV4"                   ,          0 },
   {  "MQIPADDR_IPV6"                   ,          1 },
   {  "MQIS_NO"                         ,          0 },
   {  "MQIS_YES"                        ,          1 },
   {  "MQITEM_BAG"                      ,          3 },
   {  "MQITEM_BYTE_STRING"              ,          4 },
   {  "MQITEM_BYTE_STRING_FILTER"       ,          8 },
   {  "MQITEM_INTEGER"                  ,          1 },
   {  "MQITEM_INTEGER64"                ,          7 },
   {  "MQITEM_INTEGER_FILTER"           ,          5 },
   {  "MQITEM_STRING"                   ,          2 },
   {  "MQITEM_STRING_FILTER"            ,          6 },
   {  "MQIT_BAG"                        ,          3 },
   {  "MQIT_CORREL_ID"                  ,          2 },
   {  "MQIT_GROUP_ID"                   ,          5 },
   {  "MQIT_INTEGER"                    ,          1 },
   {  "MQIT_MSG_ID"                     ,          1 },
   {  "MQIT_MSG_TOKEN"                  ,          4 },
   {  "MQIT_NONE"                       ,          0 },
   {  "MQIT_STRING"                     ,          2 },
   {  "MQKAI_AUTO"                      ,         -1 },
   {  "MQKEY_REUSE_DISABLED"            ,          0 },
   {  "MQKEY_REUSE_UNLIMITED"           ,         -1 },
   {  "MQLDAPC_CONNECTED"               ,          1 },
   {  "MQLDAPC_ERROR"                   ,          2 },
   {  "MQLDAPC_INACTIVE"                ,          0 },
   {  "MQLDAP_AUTHORMD_OS"              ,          0 },
   {  "MQLDAP_AUTHORMD_SEARCHGRP"       ,          1 },
   {  "MQLDAP_AUTHORMD_SEARCHUSR"       ,          2 },
   {  "MQLDAP_AUTHORMD_SRCHGRPSN"       ,          3 },
   {  "MQLDAP_NESTGRP_NO"               ,          0 },
   {  "MQLDAP_NESTGRP_YES"              ,          1 },
   {  "MQLR_AUTO"                       ,         -1 },
   {  "MQLR_MAX"                        ,         -2 },
   {  "MQLR_ONE"                        ,          1 },
   {  "MQMASTER_NO"                     ,          0 },
   {  "MQMASTER_YES"                    ,          1 },
   {  "MQMATCH_ALL"                     ,          3 },
   {  "MQMATCH_EXACT"                   ,          2 },
   {  "MQMATCH_GENERIC"                 ,          0 },
   {  "MQMATCH_RUNCHECK"                ,          1 },
   {  "MQMCAS_RUNNING"                  ,          3 },
   {  "MQMCAS_STOPPED"                  ,          0 },
   {  "MQMCAT_PROCESS"                  ,          1 },
   {  "MQMCAT_THREAD"                   ,          2 },
   {  "MQMCB_DISABLED"                  ,          0 },
   {  "MQMCB_ENABLED"                   ,          1 },
   {  "MQMCEV_ACK_RETRIES_EXCEEDED"     ,         13 },
   {  "MQMCEV_CCT_GETTIME_FAILED"       ,        110 },
   {  "MQMCEV_CLOSED_TRANS"             ,          5 },
   {  "MQMCEV_DEST_INTERFACE_FAILOVER"  ,        121 },
   {  "MQMCEV_DEST_INTERFACE_FAILURE"   ,        120 },
   {  "MQMCEV_FIRST_MESSAGE"            ,         20 },
   {  "MQMCEV_HEARTBEAT_TIMEOUT"        ,          2 },
   {  "MQMCEV_LATE_JOIN_FAILURE"        ,         21 },
   {  "MQMCEV_MEMORY_ALERT_OFF"         ,         26 },
   {  "MQMCEV_MEMORY_ALERT_ON"          ,         25 },
   {  "MQMCEV_MESSAGE_LOSS"             ,         22 },
   {  "MQMCEV_NACK_ALERT_OFF"           ,         28 },
   {  "MQMCEV_NACK_ALERT_ON"            ,         27 },
   {  "MQMCEV_NEW_SOURCE"               ,         10 },
   {  "MQMCEV_PACKET_LOSS"              ,          1 },
   {  "MQMCEV_PACKET_LOSS_NACK_EXPIRE"  ,         12 },
   {  "MQMCEV_PORT_INTERFACE_FAILOVER"  ,        123 },
   {  "MQMCEV_PORT_INTERFACE_FAILURE"   ,        122 },
   {  "MQMCEV_RECEIVE_QUEUE_TRIMMED"    ,         11 },
   {  "MQMCEV_RELIABILITY"              ,          4 },
   {  "MQMCEV_RELIABILITY_CHANGED"      ,         31 },
   {  "MQMCEV_REPAIR_ALERT_OFF"         ,         30 },
   {  "MQMCEV_REPAIR_ALERT_ON"          ,         29 },
   {  "MQMCEV_REPAIR_DELAY"             ,         24 },
   {  "MQMCEV_SEND_PACKET_FAILURE"      ,         23 },
   {  "MQMCEV_SHM_DEST_UNUSABLE"        ,         80 },
   {  "MQMCEV_SHM_PORT_UNUSABLE"        ,         81 },
   {  "MQMCEV_STREAM_ERROR"             ,          6 },
   {  "MQMCEV_STREAM_EXPELLED"          ,         16 },
   {  "MQMCEV_STREAM_RESUME_NACK"       ,         15 },
   {  "MQMCEV_STREAM_SUSPEND_NACK"      ,         14 },
   {  "MQMCEV_VERSION_CONFLICT"         ,          3 },
   {  "MQMCP_ALL"                       ,         -1 },
   {  "MQMCP_COMPAT"                    ,         -2 },
   {  "MQMCP_NONE"                      ,          0 },
   {  "MQMCP_REPLY"                     ,          2 },
   {  "MQMCP_USER"                      ,          1 },
   {  "MQMC_AS_PARENT"                  ,          0 },
   {  "MQMC_DISABLED"                   ,          2 },
   {  "MQMC_ENABLED"                    ,          1 },
   {  "MQMC_ONLY"                       ,          3 },
   {  "MQMD1_CURRENT_LENGTH"            ,        324 },
   {  "MQMD1_LENGTH_1"                  ,        324 },
   {  "MQMD2_CURRENT_LENGTH"            ,        364 },
   {  "MQMD2_LENGTH_1"                  ,        324 },
   {  "MQMD2_LENGTH_2"                  ,        364 },
   {  "MQMDEF_NONE"                     ,          0 },
   {  "MQMDE_CURRENT_LENGTH"            ,         72 },
   {  "MQMDE_CURRENT_VERSION"           ,          2 },
   {  "MQMDE_LENGTH_2"                  ,         72 },
   {  "MQMDE_VERSION_2"                 ,          2 },
   {  "MQMDS_FIFO"                      ,          1 },
   {  "MQMDS_PRIORITY"                  ,          0 },
   {  "MQMD_CURRENT_LENGTH"             ,        364 },
   {  "MQMD_CURRENT_VERSION"            ,          2 },
   {  "MQMD_LENGTH_1"                   ,        324 },
   {  "MQMD_LENGTH_2"                   ,        364 },
   {  "MQMD_VERSION_1"                  ,          1 },
   {  "MQMD_VERSION_2"                  ,          2 },
   {  "MQMEDIMGINTVL_OFF"               ,          0 },
   {  "MQMEDIMGLOGLN_OFF"               ,          0 },
   {  "MQMEDIMGSCHED_AUTO"              ,          1 },
   {  "MQMEDIMGSCHED_MANUAL"            ,          0 },
   {  "MQMF_ACCEPT_UNSUP_IF_XMIT_MASK"  ,    1044480 },
   {  "MQMF_ACCEPT_UNSUP_MASK"          ,   -1048576 },
   {  "MQMF_LAST_MSG_IN_GROUP"          ,         16 },
   {  "MQMF_LAST_SEGMENT"               ,          4 },
   {  "MQMF_MSG_IN_GROUP"               ,          8 },
   {  "MQMF_NONE"                       ,          0 },
   {  "MQMF_REJECT_UNSUP_MASK"          ,       4095 },
   {  "MQMF_SEGMENT"                    ,          2 },
   {  "MQMF_SEGMENTATION_ALLOWED"       ,          1 },
   {  "MQMF_SEGMENTATION_INHIBITED"     ,          0 },
   {  "MQMHBO_CURRENT_LENGTH"           ,         12 },
   {  "MQMHBO_CURRENT_VERSION"          ,          1 },
   {  "MQMHBO_DELETE_PROPERTIES"        ,          2 },
   {  "MQMHBO_LENGTH_1"                 ,         12 },
   {  "MQMHBO_NONE"                     ,          0 },
   {  "MQMHBO_PROPERTIES_IN_MQRFH2"     ,          1 },
   {  "MQMHBO_VERSION_1"                ,          1 },
   {  "MQMLP_ENCRYPTION_ALG_3DES"       ,          3 },
   {  "MQMLP_ENCRYPTION_ALG_AES128"     ,          4 },
   {  "MQMLP_ENCRYPTION_ALG_AES256"     ,          5 },
   {  "MQMLP_ENCRYPTION_ALG_DES"        ,          2 },
   {  "MQMLP_ENCRYPTION_ALG_NONE"       ,          0 },
   {  "MQMLP_ENCRYPTION_ALG_RC2"        ,          1 },
   {  "MQMLP_SIGN_ALG_MD5"              ,          1 },
   {  "MQMLP_SIGN_ALG_NONE"             ,          0 },
   {  "MQMLP_SIGN_ALG_SHA1"             ,          2 },
   {  "MQMLP_SIGN_ALG_SHA224"           ,          3 },
   {  "MQMLP_SIGN_ALG_SHA256"           ,          4 },
   {  "MQMLP_SIGN_ALG_SHA384"           ,          5 },
   {  "MQMLP_SIGN_ALG_SHA512"           ,          6 },
   {  "MQMLP_TOLERATE_UNPROTECTED_NO"   ,          0 },
   {  "MQMLP_TOLERATE_UNPROTECTED_YES"  ,          1 },
   {  "MQMMBI_UNLIMITED"                ,         -1 },
   {  "MQMODE_FORCE"                    ,          0 },
   {  "MQMODE_QUIESCE"                  ,          1 },
   {  "MQMODE_TERMINATE"                ,          2 },
   {  "MQMON_DISABLED"                  ,          0 },
   {  "MQMON_ENABLED"                   ,          1 },
   {  "MQMON_HIGH"                      ,         65 },
   {  "MQMON_LOW"                       ,         17 },
   {  "MQMON_MEDIUM"                    ,         33 },
   {  "MQMON_NONE"                      ,         -1 },
   {  "MQMON_NOT_AVAILABLE"             ,         -1 },
   {  "MQMON_OFF"                       ,          0 },
   {  "MQMON_ON"                        ,          1 },
   {  "MQMON_Q_MGR"                     ,         -3 },
   {  "MQMO_MATCH_CORREL_ID"            ,          2 },
   {  "MQMO_MATCH_GROUP_ID"             ,          4 },
   {  "MQMO_MATCH_MSG_ID"               ,          1 },
   {  "MQMO_MATCH_MSG_SEQ_NUMBER"       ,          8 },
   {  "MQMO_MATCH_MSG_TOKEN"            ,         32 },
   {  "MQMO_MATCH_OFFSET"               ,         16 },
   {  "MQMO_NONE"                       ,          0 },
   {  "MQMT_APPL_FIRST"                 ,      65536 },
   {  "MQMT_APPL_LAST"                  ,  999999999 },
   {  "MQMT_DATAGRAM"                   ,          8 },
   {  "MQMT_MQE_FIELDS"                 ,        113 },
   {  "MQMT_MQE_FIELDS_FROM_MQE"        ,        112 },
   {  "MQMT_REPLY"                      ,          2 },
   {  "MQMT_REPORT"                     ,          4 },
   {  "MQMT_REQUEST"                    ,          1 },
   {  "MQMT_SYSTEM_FIRST"               ,          1 },
   {  "MQMT_SYSTEM_LAST"                ,      65535 },
   {  "MQMULC_REFINED"                  ,          1 },
   {  "MQMULC_STANDARD"                 ,          0 },
   {  "MQNC_MAX_NAMELIST_NAME_COUNT"    ,        256 },
   {  "MQNPMS_FAST"                     ,          2 },
   {  "MQNPMS_NORMAL"                   ,          1 },
   {  "MQNPM_CLASS_HIGH"                ,         10 },
   {  "MQNPM_CLASS_NORMAL"              ,          0 },
   {  "MQNSH_ALL"                       ,         -1 },
   {  "MQNSH_NONE"                      ,          0 },
   {  "MQNT_ALL"                        ,       1001 },
   {  "MQNT_AUTH_INFO"                  ,          4 },
   {  "MQNT_CLUSTER"                    ,          2 },
   {  "MQNT_NONE"                       ,          0 },
   {  "MQNT_Q"                          ,          1 },
   {  "MQNXP_CURRENT_LENGTH (4 byte)"   ,         56 },
   {  "MQNXP_CURRENT_LENGTH (8 byte)"   ,         72 },
   {  "MQNXP_CURRENT_VERSION"           ,          2 },
   {  "MQNXP_LENGTH_1 (4 byte)"         ,         52 },
   {  "MQNXP_LENGTH_1 (8 byte)"         ,         64 },
   {  "MQNXP_LENGTH_2 (4 byte)"         ,         56 },
   {  "MQNXP_LENGTH_2 (8 byte)"         ,         72 },
   {  "MQNXP_VERSION_1"                 ,          1 },
   {  "MQNXP_VERSION_2"                 ,          2 },
   {  "MQOA_FIRST"                      ,          1 },
   {  "MQOA_LAST"                       ,       9000 },
   {  "MQOD_CURRENT_LENGTH (4 byte)"    ,        400 },
   {  "MQOD_CURRENT_LENGTH (8 byte)"    ,        424 },
   {  "MQOD_CURRENT_VERSION"            ,          4 },
   {  "MQOD_LENGTH_1"                   ,        168 },
   {  "MQOD_LENGTH_2 (4 byte)"          ,        200 },
   {  "MQOD_LENGTH_2 (8 byte)"          ,        208 },
   {  "MQOD_LENGTH_3 (4 byte)"          ,        336 },
   {  "MQOD_LENGTH_3 (8 byte)"          ,        344 },
   {  "MQOD_LENGTH_4 (4 byte)"          ,        400 },
   {  "MQOD_LENGTH_4 (8 byte)"          ,        424 },
   {  "MQOD_VERSION_1"                  ,          1 },
   {  "MQOD_VERSION_2"                  ,          2 },
   {  "MQOD_VERSION_3"                  ,          3 },
   {  "MQOD_VERSION_4"                  ,          4 },
   {  "MQOL_UNDEFINED"                  ,         -1 },
   {  "MQOM_NO"                         ,          0 },
   {  "MQOM_YES"                        ,          1 },
   {  "MQOO_ALTERNATE_USER_AUTHORITY"   ,       4096 },
   {  "MQOO_BIND_AS_Q_DEF"              ,          0 },
   {  "MQOO_BIND_NOT_FIXED"             ,      32768 },
   {  "MQOO_BIND_ON_GROUP"              ,    4194304 },
   {  "MQOO_BIND_ON_OPEN"               ,      16384 },
   {  "MQOO_BROWSE"                     ,          8 },
   {  "MQOO_CO_OP"                      ,     131072 },
   {  "MQOO_FAIL_IF_QUIESCING"          ,       8192 },
   {  "MQOO_INPUT_AS_Q_DEF"             ,          1 },
   {  "MQOO_INPUT_EXCLUSIVE"            ,          4 },
   {  "MQOO_INPUT_SHARED"               ,          2 },
   {  "MQOO_INQUIRE"                    ,         32 },
   {  "MQOO_NO_MULTICAST"               ,    2097152 },
   {  "MQOO_NO_READ_AHEAD"              ,     524288 },
   {  "MQOO_OUTPUT"                     ,         16 },
   {  "MQOO_PASS_ALL_CONTEXT"           ,        512 },
   {  "MQOO_PASS_IDENTITY_CONTEXT"      ,        256 },
   {  "MQOO_READ_AHEAD"                 ,    1048576 },
   {  "MQOO_READ_AHEAD_AS_Q_DEF"        ,          0 },
   {  "MQOO_RESOLVE_LOCAL_Q"            ,     262144 },
   {  "MQOO_RESOLVE_LOCAL_TOPIC"        ,     262144 },
   {  "MQOO_RESOLVE_NAMES"              ,      65536 },
   {  "MQOO_SAVE_ALL_CONTEXT"           ,        128 },
   {  "MQOO_SET"                        ,         64 },
   {  "MQOO_SET_ALL_CONTEXT"            ,       2048 },
   {  "MQOO_SET_IDENTITY_CONTEXT"       ,       1024 },
   {  "MQOPER_APPL_FIRST"               ,      65536 },
   {  "MQOPER_APPL_LAST"                ,  999999999 },
   {  "MQOPER_BROWSE"                   ,          1 },
   {  "MQOPER_DISCARD"                  ,          2 },
   {  "MQOPER_DISCARDED_PUBLISH"        ,         12 },
   {  "MQOPER_EXCLUDED_PUBLISH"         ,         11 },
   {  "MQOPER_GET"                      ,          3 },
   {  "MQOPER_PUBLISH"                  ,         10 },
   {  "MQOPER_PUT"                      ,          4 },
   {  "MQOPER_PUT_REPLY"                ,          5 },
   {  "MQOPER_PUT_REPORT"               ,          6 },
   {  "MQOPER_RECEIVE"                  ,          7 },
   {  "MQOPER_SEND"                     ,          8 },
   {  "MQOPER_SYSTEM_FIRST"             ,          0 },
   {  "MQOPER_SYSTEM_LAST"              ,      65535 },
   {  "MQOPER_TRANSFORM"                ,          9 },
   {  "MQOPER_UNKNOWN"                  ,          0 },
   {  "MQOPMODE_COMPAT"                 ,          0 },
   {  "MQOPMODE_NEW_FUNCTION"           ,          1 },
   {  "MQOP_DEREGISTER"                 ,        512 },
   {  "MQOP_REGISTER"                   ,        256 },
   {  "MQOP_RESUME"                     ,     131072 },
   {  "MQOP_START"                      ,          1 },
   {  "MQOP_START_WAIT"                 ,          2 },
   {  "MQOP_STOP"                       ,          4 },
   {  "MQOP_SUSPEND"                    ,      65536 },
   {  "MQOT_ALIAS_Q"                    ,       1002 },
   {  "MQOT_ALL"                        ,       1001 },
   {  "MQOT_AMQP_CHANNEL"               ,       1021 },
   {  "MQOT_AUTH_INFO"                  ,          7 },
   {  "MQOT_AUTH_REC"                   ,       1022 },
   {  "MQOT_CF_STRUC"                   ,         10 },
   {  "MQOT_CHANNEL"                    ,          6 },
   {  "MQOT_CHLAUTH"                    ,       1016 },
   {  "MQOT_CLNTCONN_CHANNEL"           ,       1014 },
   {  "MQOT_COMM_INFO"                  ,          9 },
   {  "MQOT_CURRENT_CHANNEL"            ,       1011 },
   {  "MQOT_LISTENER"                   ,         11 },
   {  "MQOT_LOCAL_Q"                    ,       1004 },
   {  "MQOT_MODEL_Q"                    ,       1003 },
   {  "MQOT_NAMELIST"                   ,          2 },
   {  "MQOT_NONE"                       ,          0 },
   {  "MQOT_PROCESS"                    ,          3 },
   {  "MQOT_PROT_POLICY"                ,       1019 },
   {  "MQOT_Q"                          ,          1 },
   {  "MQOT_Q_MGR"                      ,          5 },
   {  "MQOT_RECEIVER_CHANNEL"           ,       1010 },
   {  "MQOT_REMOTE_Q"                   ,       1005 },
   {  "MQOT_REMOTE_Q_MGR_NAME"          ,       1017 },
   {  "MQOT_REQUESTER_CHANNEL"          ,       1009 },
   {  "MQOT_RESERVED_1"                 ,        999 },
   {  "MQOT_SAVED_CHANNEL"              ,       1012 },
   {  "MQOT_SENDER_CHANNEL"             ,       1007 },
   {  "MQOT_SERVER_CHANNEL"             ,       1008 },
   {  "MQOT_SERVICE"                    ,         12 },
   {  "MQOT_SHORT_CHANNEL"              ,       1015 },
   {  "MQOT_STORAGE_CLASS"              ,          4 },
   {  "MQOT_SVRCONN_CHANNEL"            ,       1013 },
   {  "MQOT_TOPIC"                      ,          8 },
   {  "MQOT_TT_CHANNEL"                 ,       1020 },
   {  "MQPAGECLAS_4KB"                  ,          0 },
   {  "MQPAGECLAS_FIXED4KB"             ,          1 },
   {  "MQPA_ALTERNATE_OR_MCA"           ,          4 },
   {  "MQPA_CONTEXT"                    ,          2 },
   {  "MQPA_DEFAULT"                    ,          1 },
   {  "MQPA_ONLY_MCA"                   ,          3 },
   {  "MQPBC_CURRENT_LENGTH (4 byte)"   ,         32 },
   {  "MQPBC_CURRENT_LENGTH (8 byte)"   ,         40 },
   {  "MQPBC_CURRENT_VERSION"           ,          2 },
   {  "MQPBC_LENGTH_1 (4 byte)"         ,         28 },
   {  "MQPBC_LENGTH_1 (8 byte)"         ,         32 },
   {  "MQPBC_LENGTH_2 (4 byte)"         ,         32 },
   {  "MQPBC_LENGTH_2 (8 byte)"         ,         40 },
   {  "MQPBC_VERSION_1"                 ,          1 },
   {  "MQPBC_VERSION_2"                 ,          2 },
   {  "MQPD_ACCEPT_UNSUP_IF_XMIT_MASK"  ,    1047552 },
   {  "MQPD_ACCEPT_UNSUP_MASK"          ,       1023 },
   {  "MQPD_CURRENT_LENGTH"             ,         24 },
   {  "MQPD_CURRENT_VERSION"            ,          1 },
   {  "MQPD_LENGTH_1"                   ,         24 },
   {  "MQPD_NONE"                       ,          0 },
   {  "MQPD_NO_CONTEXT"                 ,          0 },
   {  "MQPD_REJECT_UNSUP_MASK"          ,   -1048576 },
   {  "MQPD_SUPPORT_OPTIONAL"           ,          1 },
   {  "MQPD_SUPPORT_REQUIRED"           ,    1048576 },
   {  "MQPD_SUPPORT_REQUIRED_IF_LOCAL"  ,       1024 },
   {  "MQPD_USER_CONTEXT"               ,          1 },
   {  "MQPD_VERSION_1"                  ,          1 },
   {  "MQPER_NOT_PERSISTENT"            ,          0 },
   {  "MQPER_PERSISTENCE_AS_PARENT"     ,         -1 },
   {  "MQPER_PERSISTENCE_AS_Q_DEF"      ,          2 },
   {  "MQPER_PERSISTENCE_AS_TOPIC_DEF"  ,          2 },
   {  "MQPER_PERSISTENT"                ,          1 },
   {  "MQPL_AIX"                        ,          3 },
   {  "MQPL_APPLIANCE"                  ,         28 },
   {  "MQPL_MVS"                        ,          1 },
   {  "MQPL_NSK"                        ,         13 },
   {  "MQPL_NSS"                        ,         13 },
   {  "MQPL_OPEN_TP1"                   ,         15 },
   {  "MQPL_OS2"                        ,          2 },
   {  "MQPL_OS390"                      ,          1 },
   {  "MQPL_OS400"                      ,          4 },
   {  "MQPL_TPF"                        ,         23 },
   {  "MQPL_UNIX"                       ,          3 },
   {  "MQPL_VM"                         ,         18 },
   {  "MQPL_VMS"                        ,         12 },
   {  "MQPL_VSE"                        ,         27 },
   {  "MQPL_WINDOWS"                    ,          5 },
   {  "MQPL_WINDOWS_NT"                 ,         11 },
   {  "MQPL_ZOS"                        ,          1 },
   {  "MQPMO_ALTERNATE_USER_AUTHORITY"  ,       4096 },
   {  "MQPMO_ASYNC_RESPONSE"            ,      65536 },
   {  "MQPMO_CURRENT_LENGTH (4 byte)"   ,        176 },
   {  "MQPMO_CURRENT_LENGTH (8 byte)"   ,        184 },
   {  "MQPMO_CURRENT_VERSION"           ,          3 },
   {  "MQPMO_DEFAULT_CONTEXT"           ,         32 },
   {  "MQPMO_FAIL_IF_QUIESCING"         ,       8192 },
   {  "MQPMO_LENGTH_1"                  ,        128 },
   {  "MQPMO_LENGTH_2 (4 byte)"         ,        152 },
   {  "MQPMO_LENGTH_2 (8 byte)"         ,        160 },
   {  "MQPMO_LENGTH_3 (4 byte)"         ,        176 },
   {  "MQPMO_LENGTH_3 (8 byte)"         ,        184 },
   {  "MQPMO_LOGICAL_ORDER"             ,      32768 },
   {  "MQPMO_MD_FOR_OUTPUT_ONLY"        ,    8388608 },
   {  "MQPMO_NEW_CORREL_ID"             ,        128 },
   {  "MQPMO_NEW_MSG_ID"                ,         64 },
   {  "MQPMO_NONE"                      ,          0 },
   {  "MQPMO_NOT_OWN_SUBS"              ,  268435456 },
   {  "MQPMO_NO_CONTEXT"                ,      16384 },
   {  "MQPMO_NO_SYNCPOINT"              ,          4 },
   {  "MQPMO_PASS_ALL_CONTEXT"          ,        512 },
   {  "MQPMO_PASS_IDENTITY_CONTEXT"     ,        256 },
   {  "MQPMO_PUB_OPTIONS_MASK"          ,    2097152 },
   {  "MQPMO_RESOLVE_LOCAL_Q"           ,     262144 },
   {  "MQPMO_RESPONSE_AS_Q_DEF"         ,          0 },
   {  "MQPMO_RESPONSE_AS_TOPIC_DEF"     ,          0 },
   {  "MQPMO_RETAIN"                    ,    2097152 },
   {  "MQPMO_SCOPE_QMGR"                ,   67108864 },
   {  "MQPMO_SET_ALL_CONTEXT"           ,       2048 },
   {  "MQPMO_SET_IDENTITY_CONTEXT"      ,       1024 },
   {  "MQPMO_SUPPRESS_REPLYTO"          ,  134217728 },
   {  "MQPMO_SYNCPOINT"                 ,          2 },
   {  "MQPMO_SYNC_RESPONSE"             ,     131072 },
   {  "MQPMO_VERSION_1"                 ,          1 },
   {  "MQPMO_VERSION_2"                 ,          2 },
   {  "MQPMO_VERSION_3"                 ,          3 },
   {  "MQPMO_WARN_IF_NO_SUBS_MATCHED"   ,     524288 },
   {  "MQPMRF_ACCOUNTING_TOKEN"         ,         16 },
   {  "MQPMRF_CORREL_ID"                ,          2 },
   {  "MQPMRF_FEEDBACK"                 ,          8 },
   {  "MQPMRF_GROUP_ID"                 ,          4 },
   {  "MQPMRF_MSG_ID"                   ,          1 },
   {  "MQPMRF_NONE"                     ,          0 },
   {  "MQPO_NO"                         ,          0 },
   {  "MQPO_YES"                        ,          1 },
   {  "MQPRI_PRIORITY_AS_PARENT"        ,         -2 },
   {  "MQPRI_PRIORITY_AS_PUBLISHED"     ,         -3 },
   {  "MQPRI_PRIORITY_AS_Q_DEF"         ,         -1 },
   {  "MQPRI_PRIORITY_AS_TOPIC_DEF"     ,         -1 },
   {  "MQPROP_ALL"                      ,          2 },
   {  "MQPROP_COMPATIBILITY"            ,          0 },
   {  "MQPROP_FORCE_MQRFH2"             ,          3 },
   {  "MQPROP_NONE"                     ,          1 },
   {  "MQPROP_UNRESTRICTED_LENGTH"      ,         -1 },
   {  "MQPROP_V6COMPAT"                 ,          4 },
   {  "MQPROTO_AMQP"                    ,          3 },
   {  "MQPROTO_HTTP"                    ,          2 },
   {  "MQPROTO_MQTTV3"                  ,          1 },
   {  "MQPROTO_MQTTV311"                ,          4 },
   {  "MQPRT_ASYNC_RESPONSE"            ,          2 },
   {  "MQPRT_RESPONSE_AS_PARENT"        ,          0 },
   {  "MQPRT_SYNC_RESPONSE"             ,          1 },
   {  "MQPSCLUS_DISABLED"               ,          0 },
   {  "MQPSCLUS_ENABLED"                ,          1 },
   {  "MQPSCT_NONE"                     ,         -1 },
   {  "MQPSM_COMPAT"                    ,          1 },
   {  "MQPSM_DISABLED"                  ,          0 },
   {  "MQPSM_ENABLED"                   ,          2 },
   {  "MQPSPROP_COMPAT"                 ,          1 },
   {  "MQPSPROP_MSGPROP"                ,          3 },
   {  "MQPSPROP_NONE"                   ,          0 },
   {  "MQPSPROP_RFH2"                   ,          2 },
   {  "MQPSST_ALL"                      ,          0 },
   {  "MQPSST_CHILD"                    ,          3 },
   {  "MQPSST_LOCAL"                    ,          1 },
   {  "MQPSST_PARENT"                   ,          2 },
   {  "MQPSXP_CURRENT_LENGTH (4 byte)"  ,        160 },
   {  "MQPSXP_CURRENT_LENGTH (8 byte)"  ,        184 },
   {  "MQPSXP_CURRENT_VERSION"          ,          2 },
   {  "MQPSXP_LENGTH_1 (4 byte)"        ,        156 },
   {  "MQPSXP_LENGTH_1 (8 byte)"        ,        176 },
   {  "MQPSXP_LENGTH_2 (4 byte)"        ,        160 },
   {  "MQPSXP_LENGTH_2 (8 byte)"        ,        184 },
   {  "MQPSXP_VERSION_1"                ,          1 },
   {  "MQPSXP_VERSION_2"                ,          2 },
   {  "MQPS_STATUS_ACTIVE"              ,          3 },
   {  "MQPS_STATUS_COMPAT"              ,          4 },
   {  "MQPS_STATUS_ERROR"               ,          5 },
   {  "MQPS_STATUS_INACTIVE"            ,          0 },
   {  "MQPS_STATUS_REFUSED"             ,          6 },
   {  "MQPS_STATUS_STARTING"            ,          1 },
   {  "MQPS_STATUS_STOPPING"            ,          2 },
   {  "MQPUBO_CORREL_ID_AS_IDENTITY"    ,          1 },
   {  "MQPUBO_IS_RETAINED_PUBLICATION"  ,         16 },
   {  "MQPUBO_NONE"                     ,          0 },
   {  "MQPUBO_NO_REGISTRATION"          ,          8 },
   {  "MQPUBO_OTHER_SUBSCRIBERS_ONLY"   ,          4 },
   {  "MQPUBO_RETAIN_PUBLICATION"       ,          2 },
   {  "MQQA_BACKOUT_HARDENED"           ,          1 },
   {  "MQQA_BACKOUT_NOT_HARDENED"       ,          0 },
   {  "MQQA_GET_ALLOWED"                ,          0 },
   {  "MQQA_GET_INHIBITED"              ,          1 },
   {  "MQQA_NOT_SHAREABLE"              ,          0 },
   {  "MQQA_PUT_ALLOWED"                ,          0 },
   {  "MQQA_PUT_INHIBITED"              ,          1 },
   {  "MQQA_SHAREABLE"                  ,          1 },
   {  "MQQDT_PERMANENT_DYNAMIC"         ,          2 },
   {  "MQQDT_PREDEFINED"                ,          1 },
   {  "MQQDT_SHARED_DYNAMIC"            ,          4 },
   {  "MQQDT_TEMPORARY_DYNAMIC"         ,          3 },
   {  "MQQF_CLWL_USEQ_ANY"              ,         64 },
   {  "MQQF_CLWL_USEQ_LOCAL"            ,        128 },
   {  "MQQF_LOCAL_Q"                    ,          1 },
   {  "MQQMDT_AUTO_CLUSTER_SENDER"      ,          2 },
   {  "MQQMDT_AUTO_EXP_CLUSTER_SENDER"  ,          4 },
   {  "MQQMDT_CLUSTER_RECEIVER"         ,          3 },
   {  "MQQMDT_EXPLICIT_CLUSTER_SENDER"  ,          1 },
   {  "MQQMFAC_DB2"                     ,          2 },
   {  "MQQMFAC_IMS_BRIDGE"              ,          1 },
   {  "MQQMF_AVAILABLE"                 ,         32 },
   {  "MQQMF_CLUSSDR_AUTO_DEFINED"      ,         16 },
   {  "MQQMF_CLUSSDR_USER_DEFINED"      ,          8 },
   {  "MQQMF_REPOSITORY_Q_MGR"          ,          2 },
   {  "MQQMOPT_DISABLED"                ,          0 },
   {  "MQQMOPT_ENABLED"                 ,          1 },
   {  "MQQMOPT_REPLY"                   ,          2 },
   {  "MQQMSTA_QUIESCING"               ,          3 },
   {  "MQQMSTA_RUNNING"                 ,          2 },
   {  "MQQMSTA_STANDBY"                 ,          4 },
   {  "MQQMSTA_STARTING"                ,          1 },
   {  "MQQMT_NORMAL"                    ,          0 },
   {  "MQQMT_REPOSITORY"                ,          1 },
   {  "MQQO_NO"                         ,          0 },
   {  "MQQO_YES"                        ,          1 },
   {  "MQQSGD_ALL"                      ,         -1 },
   {  "MQQSGD_COPY"                     ,          1 },
   {  "MQQSGD_GROUP"                    ,          3 },
   {  "MQQSGD_LIVE"                     ,          6 },
   {  "MQQSGD_PRIVATE"                  ,          4 },
   {  "MQQSGD_Q_MGR"                    ,          0 },
   {  "MQQSGD_SHARED"                   ,          2 },
   {  "MQQSGS_ACTIVE"                   ,          2 },
   {  "MQQSGS_CREATED"                  ,          1 },
   {  "MQQSGS_FAILED"                   ,          4 },
   {  "MQQSGS_INACTIVE"                 ,          3 },
   {  "MQQSGS_PENDING"                  ,          5 },
   {  "MQQSGS_UNKNOWN"                  ,          0 },
   {  "MQQSIE_HIGH"                     ,          1 },
   {  "MQQSIE_NONE"                     ,          0 },
   {  "MQQSIE_OK"                       ,          2 },
   {  "MQQSOT_ALL"                      ,          1 },
   {  "MQQSOT_INPUT"                    ,          2 },
   {  "MQQSOT_OUTPUT"                   ,          3 },
   {  "MQQSO_EXCLUSIVE"                 ,          2 },
   {  "MQQSO_NO"                        ,          0 },
   {  "MQQSO_SHARED"                    ,          1 },
   {  "MQQSO_YES"                       ,          1 },
   {  "MQQSUM_NO"                       ,          0 },
   {  "MQQSUM_YES"                      ,          1 },
   {  "MQQT_ALIAS"                      ,          3 },
   {  "MQQT_ALL"                        ,       1001 },
   {  "MQQT_CLUSTER"                    ,          7 },
   {  "MQQT_LOCAL"                      ,          1 },
   {  "MQQT_MODEL"                      ,          2 },
   {  "MQQT_REMOTE"                     ,          6 },
   {  "MQRAR_NO"                        ,          0 },
   {  "MQRAR_YES"                       ,          1 },
   {  "MQRCCF_ACCESS_BLOCKED"           ,       3382 },
   {  "MQRCCF_ACTION_VALUE_ERROR"       ,       3091 },
   {  "MQRCCF_ADDRESS_ERROR"            ,       3345 },
   {  "MQRCCF_ALLOCATE_FAILED"          ,       4009 },
   {  "MQRCCF_ALLOC_FAST_TIMER_ERROR"   ,       3166 },
   {  "MQRCCF_ALLOC_RETRY_ERROR"        ,       3164 },
   {  "MQRCCF_ALLOC_SLOW_TIMER_ERROR"   ,       3165 },
   {  "MQRCCF_ALREADY_JOINED"           ,       3157 },
   {  "MQRCCF_ATTR_VALUE_ERROR"         ,       4005 },
   {  "MQRCCF_ATTR_VALUE_FIXED"         ,       3213 },
   {  "MQRCCF_AUTH_VALUE_ERROR"         ,       3171 },
   {  "MQRCCF_AUTH_VALUE_MISSING"       ,       3172 },
   {  "MQRCCF_BACKLOG_OUT_OF_RANGE"     ,       3356 },
   {  "MQRCCF_BATCH_INT_ERROR"          ,       4086 },
   {  "MQRCCF_BATCH_INT_WRONG_TYPE"     ,       4087 },
   {  "MQRCCF_BATCH_SIZE_ERROR"         ,       3037 },
   {  "MQRCCF_BIND_FAILED"              ,       4024 },
   {  "MQRCCF_BROKER_COMMAND_FAILED"    ,       3094 },
   {  "MQRCCF_BROKER_DELETED"           ,       3070 },
   {  "MQRCCF_CCSID_ERROR"              ,       3049 },
   {  "MQRCCF_CELL_DIR_NOT_AVAILABLE"   ,       4068 },
   {  "MQRCCF_CERT_LABEL_NOT_ALLOWED"   ,       3371 },
   {  "MQRCCF_CERT_VAL_POLICY_ERROR"    ,       3364 },
   {  "MQRCCF_CFBF_FILTER_VAL_LEN_ERR"  ,       3267 },
   {  "MQRCCF_CFBF_LENGTH_ERROR"        ,       3264 },
   {  "MQRCCF_CFBF_OPERATOR_ERROR"      ,       3266 },
   {  "MQRCCF_CFBF_PARM_ID_ERROR"       ,       3265 },
   {  "MQRCCF_CFBS_DUPLICATE_PARM"      ,       3254 },
   {  "MQRCCF_CFBS_LENGTH_ERROR"        ,       3255 },
   {  "MQRCCF_CFBS_PARM_ID_ERROR"       ,       3256 },
   {  "MQRCCF_CFBS_STRING_LENGTH_ERR"   ,       3257 },
   {  "MQRCCF_CFCONLOS_CHECKS_FAILED"   ,       3352 },
   {  "MQRCCF_CFGR_LENGTH_ERROR"        ,       3258 },
   {  "MQRCCF_CFGR_PARM_COUNT_ERROR"    ,       3259 },
   {  "MQRCCF_CFGR_PARM_ID_ERROR"       ,       3240 },
   {  "MQRCCF_CFH_COMMAND_ERROR"        ,       3007 },
   {  "MQRCCF_CFH_CONTROL_ERROR"        ,       3005 },
   {  "MQRCCF_CFH_LENGTH_ERROR"         ,       3002 },
   {  "MQRCCF_CFH_MSG_SEQ_NUMBER_ERR"   ,       3004 },
   {  "MQRCCF_CFH_PARM_COUNT_ERROR"     ,       3006 },
   {  "MQRCCF_CFH_TYPE_ERROR"           ,       3001 },
   {  "MQRCCF_CFH_VERSION_ERROR"        ,       3003 },
   {  "MQRCCF_CFIF_LENGTH_ERROR"        ,       3241 },
   {  "MQRCCF_CFIF_OPERATOR_ERROR"      ,       3242 },
   {  "MQRCCF_CFIF_PARM_ID_ERROR"       ,       3243 },
   {  "MQRCCF_CFIL_COUNT_ERROR"         ,       3027 },
   {  "MQRCCF_CFIL_DUPLICATE_VALUE"     ,       3026 },
   {  "MQRCCF_CFIL_LENGTH_ERROR"        ,       3028 },
   {  "MQRCCF_CFIL_PARM_ID_ERROR"       ,       3047 },
   {  "MQRCCF_CFIN_DUPLICATE_PARM"      ,       3017 },
   {  "MQRCCF_CFIN_LENGTH_ERROR"        ,       3009 },
   {  "MQRCCF_CFIN_PARM_ID_ERROR"       ,       3014 },
   {  "MQRCCF_CFSF_FILTER_VAL_LEN_ERR"  ,       3244 },
   {  "MQRCCF_CFSF_LENGTH_ERROR"        ,       3245 },
   {  "MQRCCF_CFSF_OPERATOR_ERROR"      ,       3246 },
   {  "MQRCCF_CFSF_PARM_ID_ERROR"       ,       3247 },
   {  "MQRCCF_CFSL_COUNT_ERROR"         ,       3068 },
   {  "MQRCCF_CFSL_DUPLICATE_PARM"      ,       3066 },
   {  "MQRCCF_CFSL_LENGTH_ERROR"        ,       3024 },
   {  "MQRCCF_CFSL_PARM_ID_ERROR"       ,       3033 },
   {  "MQRCCF_CFSL_STRING_LENGTH_ERR"   ,       3069 },
   {  "MQRCCF_CFSL_TOTAL_LENGTH_ERROR"  ,       3067 },
   {  "MQRCCF_CFST_CONFLICTING_PARM"    ,       3095 },
   {  "MQRCCF_CFST_DUPLICATE_PARM"      ,       3018 },
   {  "MQRCCF_CFST_LENGTH_ERROR"        ,       3010 },
   {  "MQRCCF_CFST_PARM_ID_ERROR"       ,       3015 },
   {  "MQRCCF_CFST_STRING_LENGTH_ERR"   ,       3011 },
   {  "MQRCCF_CF_STRUC_ALREADY_FAILED"  ,       3351 },
   {  "MQRCCF_CF_STRUC_ERROR"           ,       3236 },
   {  "MQRCCF_CHAD_ERROR"               ,       4079 },
   {  "MQRCCF_CHAD_EVENT_ERROR"         ,       4081 },
   {  "MQRCCF_CHAD_EVENT_WRONG_TYPE"    ,       4082 },
   {  "MQRCCF_CHAD_EXIT_ERROR"          ,       4083 },
   {  "MQRCCF_CHAD_EXIT_WRONG_TYPE"     ,       4084 },
   {  "MQRCCF_CHAD_WRONG_TYPE"          ,       4080 },
   {  "MQRCCF_CHANNEL_ALREADY_EXISTS"   ,       4042 },
   {  "MQRCCF_CHANNEL_CLOSED"           ,       4090 },
   {  "MQRCCF_CHANNEL_DISABLED"         ,       4038 },
   {  "MQRCCF_CHANNEL_ERROR"            ,       3235 },
   {  "MQRCCF_CHANNEL_INDOUBT"          ,       4025 },
   {  "MQRCCF_CHANNEL_INITIATOR_ERROR"  ,       3218 },
   {  "MQRCCF_CHANNEL_IN_USE"           ,       4031 },
   {  "MQRCCF_CHANNEL_NAME_ERROR"       ,       4044 },
   {  "MQRCCF_CHANNEL_NOT_ACTIVE"       ,       4064 },
   {  "MQRCCF_CHANNEL_NOT_FOUND"        ,       4032 },
   {  "MQRCCF_CHANNEL_NOT_STARTED"      ,       3354 },
   {  "MQRCCF_CHANNEL_TABLE_ERROR"      ,       3062 },
   {  "MQRCCF_CHANNEL_TYPE_ERROR"       ,       3034 },
   {  "MQRCCF_CHLAUTH_ACTION_ERROR"     ,       3327 },
   {  "MQRCCF_CHLAUTH_ALREADY_EXISTS"   ,       3337 },
   {  "MQRCCF_CHLAUTH_CHKCLI_ERROR"     ,       3370 },
   {  "MQRCCF_CHLAUTH_CLNTUSER_ERROR"   ,       3348 },
   {  "MQRCCF_CHLAUTH_DISABLED"         ,       3357 },
   {  "MQRCCF_CHLAUTH_MAX_EXCEEDED"     ,       3344 },
   {  "MQRCCF_CHLAUTH_NAME_ERROR"       ,       3349 },
   {  "MQRCCF_CHLAUTH_NOT_FOUND"        ,       3338 },
   {  "MQRCCF_CHLAUTH_RUNCHECK_ERROR"   ,       3350 },
   {  "MQRCCF_CHLAUTH_TYPE_ERROR"       ,       3326 },
   {  "MQRCCF_CHLAUTH_USERSRC_ERROR"    ,       3335 },
   {  "MQRCCF_CHLAUTH_WARN_ERROR"       ,       3341 },
   {  "MQRCCF_CHL_INST_TYPE_ERROR"      ,       3064 },
   {  "MQRCCF_CHL_STATUS_NOT_FOUND"     ,       3065 },
   {  "MQRCCF_CHL_SYSTEM_NOT_ACTIVE"    ,       3168 },
   {  "MQRCCF_CLIENT_ID_ERROR"          ,       3323 },
   {  "MQRCCF_CLIENT_ID_NOT_FOUND"      ,       3322 },
   {  "MQRCCF_CLROUTE_NOT_ALTERABLE"    ,       3367 },
   {  "MQRCCF_CLUSTER_NAME_CONFLICT"    ,       3088 },
   {  "MQRCCF_CLUSTER_Q_USAGE_ERROR"    ,       3090 },
   {  "MQRCCF_CLUSTER_TOPIC_CONFLICT"   ,       3368 },
   {  "MQRCCF_CLUS_XMIT_Q_USAGE_ERROR"  ,       3363 },
   {  "MQRCCF_CLWL_EXIT_NAME_ERROR"     ,       3374 },
   {  "MQRCCF_COMMAND_FAILED"           ,       3008 },
   {  "MQRCCF_COMMAND_INHIBITED"        ,       3204 },
   {  "MQRCCF_COMMAND_LENGTH_ERROR"     ,       3230 },
   {  "MQRCCF_COMMAND_LEVEL_CONFLICT"   ,       3222 },
   {  "MQRCCF_COMMAND_ORIGIN_ERROR"     ,       3231 },
   {  "MQRCCF_COMMAND_REPLY_ERROR"      ,       3226 },
   {  "MQRCCF_COMMAND_SCOPE_ERROR"      ,       3225 },
   {  "MQRCCF_COMMIT_FAILED"            ,       4040 },
   {  "MQRCCF_COMMS_LIBRARY_ERROR"      ,       3092 },
   {  "MQRCCF_COMM_INFO_TYPE_ERROR"     ,       3320 },
   {  "MQRCCF_CONFIGURATION_ERROR"      ,       4011 },
   {  "MQRCCF_CONNECTION_CLOSED"        ,       4017 },
   {  "MQRCCF_CONNECTION_ID_ERROR"      ,       3174 },
   {  "MQRCCF_CONNECTION_REFUSED"       ,       4012 },
   {  "MQRCCF_CONN_NAME_ERROR"          ,       4062 },
   {  "MQRCCF_CONN_NOT_STOPPED"         ,       3260 },
   {  "MQRCCF_CORREL_ID_ERROR"          ,       3080 },
   {  "MQRCCF_CURRENT_LOG_EXTENT"       ,       3378 },
   {  "MQRCCF_CUSTOM_ERROR"             ,       3355 },
   {  "MQRCCF_DATA_CONV_VALUE_ERROR"    ,       3052 },
   {  "MQRCCF_DATA_TOO_LARGE"           ,       4043 },
   {  "MQRCCF_DEFCLXQ_MODEL_Q_ERROR"    ,       3369 },
   {  "MQRCCF_DEF_XMIT_Q_CLUS_ERROR"    ,       3269 },
   {  "MQRCCF_DEL_OPTIONS_ERROR"        ,       3087 },
   {  "MQRCCF_DEST_NAME_ERROR"          ,       3316 },
   {  "MQRCCF_DISC_INT_ERROR"           ,       3038 },
   {  "MQRCCF_DISC_INT_WRONG_TYPE"      ,       4054 },
   {  "MQRCCF_DISC_RETRY_ERROR"         ,       3163 },
   {  "MQRCCF_DISPOSITION_CONFLICT"     ,       3211 },
   {  "MQRCCF_DUPLICATE_IDENTITY"       ,       3078 },
   {  "MQRCCF_DUPLICATE_SUBSCRIPTION"   ,       3152 },
   {  "MQRCCF_DURABILITY_NOT_ALLOWED"   ,       3314 },
   {  "MQRCCF_DYNAMIC_Q_SCOPE_ERROR"    ,       4067 },
   {  "MQRCCF_ENCODING_ERROR"           ,       3050 },
   {  "MQRCCF_ENCRYPTION_ALG_ERROR"     ,       3329 },
   {  "MQRCCF_ENTITY_NAME_MISSING"      ,       3169 },
   {  "MQRCCF_ENTITY_TYPE_MISSING"      ,       3373 },
   {  "MQRCCF_ENTRY_ERROR"              ,       4013 },
   {  "MQRCCF_ESCAPE_TYPE_ERROR"        ,       3054 },
   {  "MQRCCF_EVENTS_DISABLED"          ,       3224 },
   {  "MQRCCF_FILE_NOT_AVAILABLE"       ,       3162 },
   {  "MQRCCF_FILTER_ERROR"             ,       3150 },
   {  "MQRCCF_FORCE_VALUE_ERROR"        ,       3012 },
   {  "MQRCCF_FUNCTION_RESTRICTED"      ,       3227 },
   {  "MQRCCF_GROUPUR_CHECKS_FAILED"    ,       3319 },
   {  "MQRCCF_HB_INTERVAL_ERROR"        ,       4077 },
   {  "MQRCCF_HB_INTERVAL_WRONG_TYPE"   ,       4078 },
   {  "MQRCCF_HOBJ_ERROR"               ,       3315 },
   {  "MQRCCF_HOST_NOT_AVAILABLE"       ,       4010 },
   {  "MQRCCF_INCORRECT_Q"              ,       3079 },
   {  "MQRCCF_INCORRECT_STREAM"         ,       3075 },
   {  "MQRCCF_INDOUBT_VALUE_ERROR"      ,       3053 },
   {  "MQRCCF_INVALID_DESTINATION"      ,       3317 },
   {  "MQRCCF_INVALID_PROTOCOL"         ,       3365 },
   {  "MQRCCF_IPADDR_ERROR"             ,       3345 },
   {  "MQRCCF_IPADDR_RANGE_CONFLICT"    ,       3343 },
   {  "MQRCCF_IPADDR_RANGE_ERROR"       ,       3346 },
   {  "MQRCCF_KEEP_ALIVE_INT_ERROR"     ,       4060 },
   {  "MQRCCF_LIKE_OBJECT_WRONG_TYPE"   ,       4003 },
   {  "MQRCCF_LISTENER_CONFLICT"        ,       3232 },
   {  "MQRCCF_LISTENER_NOT_STARTED"     ,       4020 },
   {  "MQRCCF_LISTENER_RUNNING"         ,       3249 },
   {  "MQRCCF_LISTENER_STARTED"         ,       3233 },
   {  "MQRCCF_LISTENER_STILL_ACTIVE"    ,       3268 },
   {  "MQRCCF_LISTENER_STOPPED"         ,       3234 },
   {  "MQRCCF_LOG_EXTENT_ERROR"         ,       3381 },
   {  "MQRCCF_LOG_EXTENT_NOT_FOUND"     ,       3379 },
   {  "MQRCCF_LOG_NOT_REDUCED"          ,       3380 },
   {  "MQRCCF_LOG_TYPE_ERROR"           ,       3175 },
   {  "MQRCCF_LONG_RETRY_ERROR"         ,       3041 },
   {  "MQRCCF_LONG_RETRY_WRONG_TYPE"    ,       4057 },
   {  "MQRCCF_LONG_TIMER_ERROR"         ,       3042 },
   {  "MQRCCF_LONG_TIMER_WRONG_TYPE"    ,       4058 },
   {  "MQRCCF_LSTR_STATUS_NOT_FOUND"    ,       3250 },
   {  "MQRCCF_MAX_INSTANCES_ERROR"      ,       3306 },
   {  "MQRCCF_MAX_INSTS_PER_CLNT_ERR"   ,       3307 },
   {  "MQRCCF_MAX_MSG_LENGTH_ERROR"     ,       3044 },
   {  "MQRCCF_MCA_NAME_ERROR"           ,       4047 },
   {  "MQRCCF_MCA_NAME_WRONG_TYPE"      ,       4053 },
   {  "MQRCCF_MCA_TYPE_ERROR"           ,       3063 },
   {  "MQRCCF_MD_FORMAT_ERROR"          ,       3023 },
   {  "MQRCCF_MISSING_CONN_NAME"        ,       4061 },
   {  "MQRCCF_MODE_VALUE_ERROR"         ,       3029 },
   {  "MQRCCF_MQCONN_FAILED"            ,       4026 },
   {  "MQRCCF_MQGET_FAILED"             ,       4028 },
   {  "MQRCCF_MQINQ_FAILED"             ,       4036 },
   {  "MQRCCF_MQOPEN_FAILED"            ,       4027 },
   {  "MQRCCF_MQPUT_FAILED"             ,       4029 },
   {  "MQRCCF_MQSET_FAILED"             ,       4063 },
   {  "MQRCCF_MR_COUNT_ERROR"           ,       4069 },
   {  "MQRCCF_MR_COUNT_WRONG_TYPE"      ,       4070 },
   {  "MQRCCF_MR_EXIT_NAME_ERROR"       ,       4071 },
   {  "MQRCCF_MR_EXIT_NAME_WRONG_TYPE"  ,       4072 },
   {  "MQRCCF_MR_INTERVAL_ERROR"        ,       4073 },
   {  "MQRCCF_MR_INTERVAL_WRONG_TYPE"   ,       4074 },
   {  "MQRCCF_MSG_EXIT_NAME_ERROR"      ,       4050 },
   {  "MQRCCF_MSG_LENGTH_ERROR"         ,       3016 },
   {  "MQRCCF_MSG_SEQ_NUMBER_ERROR"     ,       3030 },
   {  "MQRCCF_MSG_TRUNCATED"            ,       3048 },
   {  "MQRCCF_NAMELIST_ERROR"           ,       3215 },
   {  "MQRCCF_NETBIOS_NAME_ERROR"       ,       3093 },
   {  "MQRCCF_NET_PRIORITY_ERROR"       ,       4088 },
   {  "MQRCCF_NET_PRIORITY_WRONG_TYPE"  ,       4089 },
   {  "MQRCCF_NONE_FOUND"               ,       3200 },
   {  "MQRCCF_NOT_AUTHORIZED"           ,       3081 },
   {  "MQRCCF_NOT_REGISTERED"           ,       3073 },
   {  "MQRCCF_NOT_XMIT_Q"               ,       4037 },
   {  "MQRCCF_NO_CHANNEL_INITIATOR"     ,       3217 },
   {  "MQRCCF_NO_COMMS_MANAGER"         ,       4019 },
   {  "MQRCCF_NO_RETAINED_MSG"          ,       3077 },
   {  "MQRCCF_NO_START_CMD"             ,       3262 },
   {  "MQRCCF_NO_STOP_CMD"              ,       3263 },
   {  "MQRCCF_NO_STORAGE"               ,       4018 },
   {  "MQRCCF_NO_XCF_PARTNER"           ,       3239 },
   {  "MQRCCF_NPM_SPEED_ERROR"          ,       4075 },
   {  "MQRCCF_NPM_SPEED_WRONG_TYPE"     ,       4076 },
   {  "MQRCCF_OBJECT_ALREADY_EXISTS"    ,       4001 },
   {  "MQRCCF_OBJECT_BEING_DELETED"     ,       3205 },
   {  "MQRCCF_OBJECT_IN_USE"            ,       3160 },
   {  "MQRCCF_OBJECT_LIMIT_EXCEEDED"    ,       3209 },
   {  "MQRCCF_OBJECT_NAME_ERROR"        ,       4008 },
   {  "MQRCCF_OBJECT_NAME_RESTRICTED"   ,       3208 },
   {  "MQRCCF_OBJECT_OPEN"              ,       4004 },
   {  "MQRCCF_OBJECT_OPEN_FORCE"        ,       3210 },
   {  "MQRCCF_OBJECT_TYPE_MISSING"      ,       3173 },
   {  "MQRCCF_OBJECT_WRONG_TYPE"        ,       4002 },
   {  "MQRCCF_PARM_CONFLICT"            ,       3203 },
   {  "MQRCCF_PARM_COUNT_TOO_BIG"       ,       3020 },
   {  "MQRCCF_PARM_COUNT_TOO_SMALL"     ,       3019 },
   {  "MQRCCF_PARM_MISSING"             ,       3228 },
   {  "MQRCCF_PARM_SEQUENCE_ERROR"      ,       3035 },
   {  "MQRCCF_PARM_SYNTAX_ERROR"        ,       3097 },
   {  "MQRCCF_PARM_VALUE_ERROR"         ,       3229 },
   {  "MQRCCF_PATH_NOT_VALID"           ,       3096 },
   {  "MQRCCF_PING_DATA_COMPARE_ERROR"  ,       3032 },
   {  "MQRCCF_PING_DATA_COUNT_ERROR"    ,       3031 },
   {  "MQRCCF_PING_ERROR"               ,       4030 },
   {  "MQRCCF_POLICY_NAME_MISSING"      ,       3334 },
   {  "MQRCCF_POLICY_NOT_FOUND"         ,       3328 },
   {  "MQRCCF_POLICY_VERSION_ERROR"     ,       3332 },
   {  "MQRCCF_PORT_IN_USE"              ,       3324 },
   {  "MQRCCF_PORT_NUMBER_ERROR"        ,       3167 },
   {  "MQRCCF_PROFILE_NAME_ERROR"       ,       3170 },
   {  "MQRCCF_PROFILE_NAME_MISSING"     ,       3347 },
   {  "MQRCCF_PROGRAM_AUTH_FAILED"      ,       3177 },
   {  "MQRCCF_PROGRAM_NOT_AVAILABLE"    ,       3176 },
   {  "MQRCCF_PSCLUS_DISABLED_TOPDEF"   ,       3359 },
   {  "MQRCCF_PSCLUS_TOPIC_EXISTS"      ,       3360 },
   {  "MQRCCF_PUBSUB_INHIBITED"         ,       3318 },
   {  "MQRCCF_PUB_OPTIONS_ERROR"        ,       3084 },
   {  "MQRCCF_PURGE_VALUE_ERROR"        ,       3046 },
   {  "MQRCCF_PUT_AUTH_ERROR"           ,       3045 },
   {  "MQRCCF_PUT_AUTH_WRONG_TYPE"      ,       4059 },
   {  "MQRCCF_PWD_LENGTH_ERROR"         ,       3098 },
   {  "MQRCCF_QUEUES_VALUE_ERROR"       ,       3051 },
   {  "MQRCCF_QUIESCE_VALUE_ERROR"      ,       3029 },
   {  "MQRCCF_Q_ALREADY_IN_CELL"        ,       3021 },
   {  "MQRCCF_Q_ATTR_CONFLICT"          ,       3223 },
   {  "MQRCCF_Q_MGR_ATTR_CONFLICT"      ,       3372 },
   {  "MQRCCF_Q_MGR_CCSID_ERROR"        ,       3086 },
   {  "MQRCCF_Q_MGR_NAME_ERROR"         ,       3074 },
   {  "MQRCCF_Q_MGR_NOT_IN_QSG"         ,       3212 },
   {  "MQRCCF_Q_NAME_ERROR"             ,       3076 },
   {  "MQRCCF_Q_STATUS_NOT_FOUND"       ,       4091 },
   {  "MQRCCF_Q_TYPE_ERROR"             ,       3022 },
   {  "MQRCCF_Q_WRONG_TYPE"             ,       4007 },
   {  "MQRCCF_RCV_EXIT_NAME_ERROR"      ,       4051 },
   {  "MQRCCF_RECEIVED_DATA_ERROR"      ,       4015 },
   {  "MQRCCF_RECEIVE_FAILED"           ,       4016 },
   {  "MQRCCF_RECIPIENT_DN_MISSING"     ,       3333 },
   {  "MQRCCF_REG_OPTIONS_ERROR"        ,       3083 },
   {  "MQRCCF_REMOTE_CHL_TYPE_ERROR"    ,       3376 },
   {  "MQRCCF_REMOTE_QM_TERMINATING"    ,       4035 },
   {  "MQRCCF_REMOTE_QM_UNAVAILABLE"    ,       4034 },
   {  "MQRCCF_REMOTE_Q_NAME_ERROR"      ,       3313 },
   {  "MQRCCF_REPLACE_VALUE_ERROR"      ,       3025 },
   {  "MQRCCF_REPOS_NAME_CONFLICT"      ,       3089 },
   {  "MQRCCF_REPOS_VALUE_ERROR"        ,       3055 },
   {  "MQRCCF_RETAINED_NOT_SUPPORTED"   ,       4095 },
   {  "MQRCCF_REVDNS_DISABLED"          ,       3366 },
   {  "MQRCCF_SECURITY_CASE_CONFLICT"   ,       3303 },
   {  "MQRCCF_SECURITY_REFRESH_FAILED"  ,       3202 },
   {  "MQRCCF_SECURITY_SWITCH_OFF"      ,       3201 },
   {  "MQRCCF_SEC_EXIT_NAME_ERROR"      ,       4049 },
   {  "MQRCCF_SEND_EXIT_NAME_ERROR"     ,       4048 },
   {  "MQRCCF_SEND_FAILED"              ,       4014 },
   {  "MQRCCF_SEQ_NUMBER_WRAP_ERROR"    ,       3043 },
   {  "MQRCCF_SERVICE_NAME_ERROR"       ,       3375 },
   {  "MQRCCF_SERVICE_REQUEST_PENDING"  ,       3261 },
   {  "MQRCCF_SERVICE_RUNNING"          ,       3251 },
   {  "MQRCCF_SERVICE_STOPPED"          ,       3253 },
   {  "MQRCCF_SERV_STATUS_NOT_FOUND"    ,       3252 },
   {  "MQRCCF_SHARING_CONVS_ERROR"      ,       3301 },
   {  "MQRCCF_SHARING_CONVS_TYPE"       ,       3302 },
   {  "MQRCCF_SHORT_RETRY_ERROR"        ,       3039 },
   {  "MQRCCF_SHORT_RETRY_WRONG_TYPE"   ,       4055 },
   {  "MQRCCF_SHORT_TIMER_ERROR"        ,       3040 },
   {  "MQRCCF_SHORT_TIMER_WRONG_TYPE"   ,       4056 },
   {  "MQRCCF_SIGNATURE_ALG_ERROR"      ,       3330 },
   {  "MQRCCF_SMDS_REQUIRES_DSGROUP"    ,       3358 },
   {  "MQRCCF_SOCKET_ERROR"             ,       3362 },
   {  "MQRCCF_SSL_ALT_PROVIDER_REQD"    ,       3325 },
   {  "MQRCCF_SSL_CIPHER_SPEC_ERROR"    ,       4092 },
   {  "MQRCCF_SSL_CIPHER_SUITE_ERROR"   ,       3361 },
   {  "MQRCCF_SSL_CLIENT_AUTH_ERROR"    ,       4094 },
   {  "MQRCCF_SSL_PEER_NAME_ERROR"      ,       4093 },
   {  "MQRCCF_STORAGE_CLASS_IN_USE"     ,       3207 },
   {  "MQRCCF_STREAM_ERROR"             ,       3071 },
   {  "MQRCCF_STRUCTURE_TYPE_ERROR"     ,       3013 },
   {  "MQRCCF_SUBSCRIPTION_IN_USE"      ,       3155 },
   {  "MQRCCF_SUBSCRIPTION_LOCKED"      ,       3156 },
   {  "MQRCCF_SUBSCRIPTION_POINT_ERR"   ,       3309 },
   {  "MQRCCF_SUB_ALREADY_EXISTS"       ,       3311 },
   {  "MQRCCF_SUB_IDENTITY_ERROR"       ,       3154 },
   {  "MQRCCF_SUB_NAME_ERROR"           ,       3153 },
   {  "MQRCCF_SUITE_B_ERROR"            ,       3353 },
   {  "MQRCCF_SUPPRESSED_BY_EXIT"       ,       4085 },
   {  "MQRCCF_TERMINATED_BY_SEC_EXIT"   ,       4065 },
   {  "MQRCCF_TOLERATION_POL_ERROR"     ,       3331 },
   {  "MQRCCF_TOO_MANY_FILTERS"         ,       3248 },
   {  "MQRCCF_TOPICSTR_ALREADY_EXISTS"  ,       3300 },
   {  "MQRCCF_TOPIC_ERROR"              ,       3072 },
   {  "MQRCCF_TOPIC_RESTRICTED"         ,       3377 },
   {  "MQRCCF_TOPIC_STRING_NOT_FOUND"   ,       3308 },
   {  "MQRCCF_TOPIC_TYPE_ERROR"         ,       3305 },
   {  "MQRCCF_UNEXPECTED_ERROR"         ,       3238 },
   {  "MQRCCF_UNKNOWN_BROKER"           ,       3085 },
   {  "MQRCCF_UNKNOWN_FILE_NAME"        ,       3161 },
   {  "MQRCCF_UNKNOWN_OBJECT_NAME"      ,       3312 },
   {  "MQRCCF_UNKNOWN_Q_MGR"            ,       4006 },
   {  "MQRCCF_UNKNOWN_REMOTE_CHANNEL"   ,       4033 },
   {  "MQRCCF_UNKNOWN_STREAM"           ,       3082 },
   {  "MQRCCF_UNKNOWN_USER_ID"          ,       3237 },
   {  "MQRCCF_USER_EXIT_NOT_AVAILABLE"  ,       4039 },
   {  "MQRCCF_USE_CLIENT_ID_ERROR"      ,       3321 },
   {  "MQRCCF_WRONG_CHANNEL_TYPE"       ,       4041 },
   {  "MQRCCF_WRONG_CHLAUTH_ACTION"     ,       3339 },
   {  "MQRCCF_WRONG_CHLAUTH_MATCH"      ,       3342 },
   {  "MQRCCF_WRONG_CHLAUTH_TYPE"       ,       3336 },
   {  "MQRCCF_WRONG_CHLAUTH_USERSRC"    ,       3340 },
   {  "MQRCCF_WRONG_USER"               ,       3151 },
   {  "MQRCCF_XMIT_PROTOCOL_TYPE_ERR"   ,       3036 },
   {  "MQRCCF_XMIT_Q_NAME_ERROR"        ,       4045 },
   {  "MQRCCF_XMIT_Q_NAME_WRONG_TYPE"   ,       4052 },
   {  "MQRCN_DISABLED"                  ,          3 },
   {  "MQRCN_NO"                        ,          0 },
   {  "MQRCN_Q_MGR"                     ,          2 },
   {  "MQRCN_YES"                       ,          1 },
   {  "MQRCVTIME_ADD"                   ,          1 },
   {  "MQRCVTIME_EQUAL"                 ,          2 },
   {  "MQRCVTIME_MULTIPLY"              ,          0 },
   {  "MQRC_ACTION_ERROR"               ,       2535 },
   {  "MQRC_ADAPTER_CONN_LOAD_ERROR"    ,       2129 },
   {  "MQRC_ADAPTER_CONV_LOAD_ERROR"    ,       2133 },
   {  "MQRC_ADAPTER_DEFS_ERROR"         ,       2131 },
   {  "MQRC_ADAPTER_DEFS_LOAD_ERROR"    ,       2132 },
   {  "MQRC_ADAPTER_DISC_LOAD_ERROR"    ,       2138 },
   {  "MQRC_ADAPTER_NOT_AVAILABLE"      ,       2204 },
   {  "MQRC_ADAPTER_SERV_LOAD_ERROR"    ,       2130 },
   {  "MQRC_ADAPTER_STORAGE_SHORTAGE"   ,       2127 },
   {  "MQRC_ADMIN_TOPIC_STRING_ERROR"   ,       2598 },
   {  "MQRC_AIR_ERROR"                  ,       2385 },
   {  "MQRC_ALIAS_BASE_Q_TYPE_ERROR"    ,       2001 },
   {  "MQRC_ALIAS_TARGTYPE_CHANGED"     ,       2480 },
   {  "MQRC_ALREADY_CONNECTED"          ,       2002 },
   {  "MQRC_ALREADY_JOINED"             ,       2542 },
   {  "MQRC_ALTER_SUB_ERROR"            ,       2435 },
   {  "MQRC_AMQP_NOT_AVAILABLE"         ,       2599 },
   {  "MQRC_ANOTHER_Q_MGR_CONNECTED"    ,       2103 },
   {  "MQRC_API_EXIT_ERROR"             ,       2374 },
   {  "MQRC_API_EXIT_INIT_ERROR"        ,       2375 },
   {  "MQRC_API_EXIT_LOAD_ERROR"        ,       2183 },
   {  "MQRC_API_EXIT_NOT_FOUND"         ,       2182 },
   {  "MQRC_API_EXIT_TERM_ERROR"        ,       2376 },
   {  "MQRC_APPL_FIRST"                 ,        900 },
   {  "MQRC_APPL_LAST"                  ,        999 },
   {  "MQRC_ASID_MISMATCH"              ,       2157 },
   {  "MQRC_ASYNC_UOW_CONFLICT"         ,       2529 },
   {  "MQRC_ASYNC_XA_CONFLICT"          ,       2530 },
   {  "MQRC_ATTRIBUTE_LOCKED"           ,       6104 },
   {  "MQRC_AUTH_INFO_CONN_NAME_ERROR"  ,       2387 },
   {  "MQRC_AUTH_INFO_REC_COUNT_ERROR"  ,       2383 },
   {  "MQRC_AUTH_INFO_REC_ERROR"        ,       2384 },
   {  "MQRC_AUTH_INFO_TYPE_ERROR"       ,       2386 },
   {  "MQRC_BACKED_OUT"                 ,       2003 },
   {  "MQRC_BACKOUT_THRESHOLD_REACHED"  ,       2362 },
   {  "MQRC_BAG_CONVERSION_ERROR"       ,       2303 },
   {  "MQRC_BAG_WRONG_TYPE"             ,       2326 },
   {  "MQRC_BINARY_DATA_LENGTH_ERROR"   ,       6111 },
   {  "MQRC_BMHO_ERROR"                 ,       2489 },
   {  "MQRC_BO_ERROR"                   ,       2134 },
   {  "MQRC_BRIDGE_STARTED"             ,       2125 },
   {  "MQRC_BRIDGE_STOPPED"             ,       2126 },
   {  "MQRC_BUFFER_ERROR"               ,       2004 },
   {  "MQRC_BUFFER_LENGTH_ERROR"        ,       2005 },
   {  "MQRC_BUFFER_NOT_AUTOMATIC"       ,       6112 },
   {  "MQRC_CALLBACK_LINK_ERROR"        ,       2487 },
   {  "MQRC_CALLBACK_NOT_REGISTERED"    ,       2448 },
   {  "MQRC_CALLBACK_ROUTINE_ERROR"     ,       2486 },
   {  "MQRC_CALLBACK_TYPE_ERROR"        ,       2483 },
   {  "MQRC_CALL_INTERRUPTED"           ,       2549 },
   {  "MQRC_CALL_IN_PROGRESS"           ,       2219 },
   {  "MQRC_CBD_ERROR"                  ,       2444 },
   {  "MQRC_CBD_OPTIONS_ERROR"          ,       2484 },
   {  "MQRC_CCDT_URL_ERROR"             ,       2600 },
   {  "MQRC_CD_ARRAY_ERROR"             ,       2576 },
   {  "MQRC_CD_ERROR"                   ,       2277 },
   {  "MQRC_CERT_LABEL_NOT_ALLOWED"     ,       2596 },
   {  "MQRC_CERT_VAL_POLICY_ERROR"      ,       2593 },
   {  "MQRC_CFBF_ERROR"                 ,       2422 },
   {  "MQRC_CFBS_ERROR"                 ,       2395 },
   {  "MQRC_CFGR_ERROR"                 ,       2416 },
   {  "MQRC_CFH_ERROR"                  ,       2235 },
   {  "MQRC_CFIF_ERROR"                 ,       2414 },
   {  "MQRC_CFIL_ERROR"                 ,       2236 },
   {  "MQRC_CFIN_ERROR"                 ,       2237 },
   {  "MQRC_CFSF_ERROR"                 ,       2415 },
   {  "MQRC_CFSL_ERROR"                 ,       2238 },
   {  "MQRC_CFST_ERROR"                 ,       2239 },
   {  "MQRC_CF_NOT_AVAILABLE"           ,       2345 },
   {  "MQRC_CF_STRUC_AUTH_FAILED"       ,       2348 },
   {  "MQRC_CF_STRUC_ERROR"             ,       2349 },
   {  "MQRC_CF_STRUC_FAILED"            ,       2373 },
   {  "MQRC_CF_STRUC_IN_USE"            ,       2346 },
   {  "MQRC_CF_STRUC_LIST_HDR_IN_USE"   ,       2347 },
   {  "MQRC_CHANNEL_ACTIVATED"          ,       2295 },
   {  "MQRC_CHANNEL_AUTO_DEF_ERROR"     ,       2234 },
   {  "MQRC_CHANNEL_AUTO_DEF_OK"        ,       2233 },
   {  "MQRC_CHANNEL_BLOCKED"            ,       2577 },
   {  "MQRC_CHANNEL_BLOCKED_WARNING"    ,       2578 },
   {  "MQRC_CHANNEL_CONFIG_ERROR"       ,       2539 },
   {  "MQRC_CHANNEL_CONV_ERROR"         ,       2284 },
   {  "MQRC_CHANNEL_NOT_ACTIVATED"      ,       2296 },
   {  "MQRC_CHANNEL_NOT_AVAILABLE"      ,       2537 },
   {  "MQRC_CHANNEL_SSL_ERROR"          ,       2371 },
   {  "MQRC_CHANNEL_SSL_WARNING"        ,       2552 },
   {  "MQRC_CHANNEL_STARTED"            ,       2282 },
   {  "MQRC_CHANNEL_STOPPED"            ,       2283 },
   {  "MQRC_CHANNEL_STOPPED_BY_USER"    ,       2279 },
   {  "MQRC_CHAR_ATTRS_ERROR"           ,       2007 },
   {  "MQRC_CHAR_ATTRS_TOO_SHORT"       ,       2008 },
   {  "MQRC_CHAR_ATTR_LENGTH_ERROR"     ,       2006 },
   {  "MQRC_CHAR_CONVERSION_ERROR"      ,       2340 },
   {  "MQRC_CICS_BRIDGE_RESTRICTION"    ,       2187 },
   {  "MQRC_CICS_WAIT_FAILED"           ,       2140 },
   {  "MQRC_CIPHER_SPEC_NOT_SUITE_B"    ,       2591 },
   {  "MQRC_CLIENT_CHANNEL_CONFLICT"    ,       2423 },
   {  "MQRC_CLIENT_CONN_ERROR"          ,       2278 },
   {  "MQRC_CLIENT_EXIT_ERROR"          ,       2407 },
   {  "MQRC_CLIENT_EXIT_LOAD_ERROR"     ,       2406 },
   {  "MQRC_CLUSTER_EXIT_ERROR"         ,       2266 },
   {  "MQRC_CLUSTER_EXIT_LOAD_ERROR"    ,       2267 },
   {  "MQRC_CLUSTER_PUT_INHIBITED"      ,       2268 },
   {  "MQRC_CLUSTER_RESOLUTION_ERROR"   ,       2189 },
   {  "MQRC_CLUSTER_RESOURCE_ERROR"     ,       2269 },
   {  "MQRC_CMD_SERVER_NOT_AVAILABLE"   ,       2322 },
   {  "MQRC_CMHO_ERROR"                 ,       2461 },
   {  "MQRC_CNO_ERROR"                  ,       2139 },
   {  "MQRC_CODED_CHAR_SET_ID_ERROR"    ,       2330 },
   {  "MQRC_COD_NOT_VALID_FOR_XCF_Q"    ,       2106 },
   {  "MQRC_COMMAND_MQSC"               ,       2412 },
   {  "MQRC_COMMAND_PCF"                ,       2413 },
   {  "MQRC_COMMAND_TYPE_ERROR"         ,       2300 },
   {  "MQRC_COMMINFO_ERROR"             ,       2558 },
   {  "MQRC_CONFIG_CHANGE_OBJECT"       ,       2368 },
   {  "MQRC_CONFIG_CREATE_OBJECT"       ,       2367 },
   {  "MQRC_CONFIG_DELETE_OBJECT"       ,       2369 },
   {  "MQRC_CONFIG_REFRESH_OBJECT"      ,       2370 },
   {  "MQRC_CONNECTION_BROKEN"          ,       2009 },
   {  "MQRC_CONNECTION_ERROR"           ,       2273 },
   {  "MQRC_CONNECTION_NOT_AUTHORIZED"  ,       2217 },
   {  "MQRC_CONNECTION_NOT_AVAILABLE"   ,       2568 },
   {  "MQRC_CONNECTION_QUIESCING"       ,       2202 },
   {  "MQRC_CONNECTION_STOPPED"         ,       2528 },
   {  "MQRC_CONNECTION_STOPPING"        ,       2203 },
   {  "MQRC_CONNECTION_SUSPENDED"       ,       2521 },
   {  "MQRC_CONN_ID_IN_USE"             ,       2160 },
   {  "MQRC_CONN_TAG_IN_USE"            ,       2271 },
   {  "MQRC_CONN_TAG_NOT_RELEASED"      ,       2344 },
   {  "MQRC_CONN_TAG_NOT_USABLE"        ,       2350 },
   {  "MQRC_CONTENT_ERROR"              ,       2554 },
   {  "MQRC_CONTEXT_HANDLE_ERROR"       ,       2097 },
   {  "MQRC_CONTEXT_NOT_AVAILABLE"      ,       2098 },
   {  "MQRC_CONTEXT_OBJECT_NOT_VALID"   ,       6121 },
   {  "MQRC_CONTEXT_OPEN_ERROR"         ,       6122 },
   {  "MQRC_CONVERTED_MSG_TOO_BIG"      ,       2120 },
   {  "MQRC_CONVERTED_STRING_TOO_BIG"   ,       2190 },
   {  "MQRC_CORREL_ID_ERROR"            ,       2207 },
   {  "MQRC_CRYPTO_HARDWARE_ERROR"      ,       2382 },
   {  "MQRC_CSP_ERROR"                  ,       2595 },
   {  "MQRC_CTLO_ERROR"                 ,       2445 },
   {  "MQRC_CURRENT_RECORD_ERROR"       ,       2357 },
   {  "MQRC_CURSOR_NOT_VALID"           ,       6105 },
   {  "MQRC_DATA_LENGTH_ERROR"          ,       2010 },
   {  "MQRC_DATA_SET_NOT_AVAILABLE"     ,       2561 },
   {  "MQRC_DATA_TRUNCATED"             ,       6115 },
   {  "MQRC_DB2_NOT_AVAILABLE"          ,       2342 },
   {  "MQRC_DBCS_ERROR"                 ,       2150 },
   {  "MQRC_DEF_SYNCPOINT_INHIBITED"    ,       2559 },
   {  "MQRC_DEF_XMIT_Q_TYPE_ERROR"      ,       2198 },
   {  "MQRC_DEF_XMIT_Q_USAGE_ERROR"     ,       2199 },
   {  "MQRC_DEST_CLASS_NOT_ALTERABLE"   ,       2533 },
   {  "MQRC_DEST_ENV_ERROR"             ,       2263 },
   {  "MQRC_DEST_NAME_ERROR"            ,       2264 },
   {  "MQRC_DH_ERROR"                   ,       2135 },
   {  "MQRC_DISTRIBUTION_LIST_EMPTY"    ,       6126 },
   {  "MQRC_DLH_ERROR"                  ,       2141 },
   {  "MQRC_DMHO_ERROR"                 ,       2462 },
   {  "MQRC_DMPO_ERROR"                 ,       2481 },
   {  "MQRC_DUPLICATE_GROUP_SUB"        ,       2514 },
   {  "MQRC_DUPLICATE_RECOV_COORD"      ,       2163 },
   {  "MQRC_DURABILITY_NOT_ALLOWED"     ,       2436 },
   {  "MQRC_DURABILITY_NOT_ALTERABLE"   ,       2509 },
   {  "MQRC_DYNAMIC_Q_NAME_ERROR"       ,       2011 },
   {  "MQRC_ENCODING_ERROR"             ,       6106 },
   {  "MQRC_ENCODING_NOT_SUPPORTED"     ,       2308 },
   {  "MQRC_ENVIRONMENT_ERROR"          ,       2012 },
   {  "MQRC_EPH_ERROR"                  ,       2420 },
   {  "MQRC_EXIT_PROPS_NOT_SUPPORTED"   ,       2588 },
   {  "MQRC_EXIT_REASON_ERROR"          ,       2377 },
   {  "MQRC_EXPIRY_ERROR"               ,       2013 },
   {  "MQRC_FASTPATH_NOT_AVAILABLE"     ,       2590 },
   {  "MQRC_FEEDBACK_ERROR"             ,       2014 },
   {  "MQRC_FILE_NOT_AUDITED"           ,       2216 },
   {  "MQRC_FILE_SYSTEM_ERROR"          ,       2208 },
   {  "MQRC_FILTER_OPERATOR_ERROR"      ,       2418 },
   {  "MQRC_FORMAT_ERROR"               ,       2110 },
   {  "MQRC_FORMAT_NOT_SUPPORTED"       ,       2317 },
   {  "MQRC_FUNCTION_ERROR"             ,       2281 },
   {  "MQRC_FUNCTION_NOT_SUPPORTED"     ,       2298 },
   {  "MQRC_GET_ENABLED"                ,       2494 },
   {  "MQRC_GET_INHIBITED"              ,       2016 },
   {  "MQRC_GLOBAL_UOW_CONFLICT"        ,       2351 },
   {  "MQRC_GMO_ERROR"                  ,       2186 },
   {  "MQRC_GROUPING_NOT_ALLOWED"       ,       2562 },
   {  "MQRC_GROUPING_NOT_ALTERABLE"     ,       2515 },
   {  "MQRC_GROUP_ADDRESS_ERROR"        ,       2563 },
   {  "MQRC_GROUP_ID_ERROR"             ,       2258 },
   {  "MQRC_HANDLE_IN_USE_FOR_UOW"      ,       2353 },
   {  "MQRC_HANDLE_NOT_AVAILABLE"       ,       2017 },
   {  "MQRC_HBAG_ERROR"                 ,       2320 },
   {  "MQRC_HCONFIG_ERROR"              ,       2280 },
   {  "MQRC_HCONN_ASYNC_ACTIVE"         ,       2500 },
   {  "MQRC_HCONN_ERROR"                ,       2018 },
   {  "MQRC_HEADER_ERROR"               ,       2142 },
   {  "MQRC_HMSG_ERROR"                 ,       2460 },
   {  "MQRC_HMSG_NOT_AVAILABLE"         ,       2587 },
   {  "MQRC_HOBJ_ERROR"                 ,       2019 },
   {  "MQRC_HOBJ_QUIESCED"              ,       2517 },
   {  "MQRC_HOBJ_QUIESCED_NO_MSGS"      ,       2518 },
   {  "MQRC_HOST_NOT_AVAILABLE"         ,       2538 },
   {  "MQRC_IDENTITY_MISMATCH"          ,       2434 },
   {  "MQRC_IIH_ERROR"                  ,       2148 },
   {  "MQRC_IMPO_ERROR"                 ,       2464 },
   {  "MQRC_INCOMPLETE_GROUP"           ,       2241 },
   {  "MQRC_INCOMPLETE_MSG"             ,       2242 },
   {  "MQRC_INCONSISTENT_BROWSE"        ,       2259 },
   {  "MQRC_INCONSISTENT_CCSIDS"        ,       2243 },
   {  "MQRC_INCONSISTENT_ENCODINGS"     ,       2244 },
   {  "MQRC_INCONSISTENT_FORMAT"        ,       6119 },
   {  "MQRC_INCONSISTENT_ITEM_TYPE"     ,       2313 },
   {  "MQRC_INCONSISTENT_OBJECT_STATE"  ,       6120 },
   {  "MQRC_INCONSISTENT_OPEN_OPTIONS"  ,       6127 },
   {  "MQRC_INCONSISTENT_PERSISTENCE"   ,       2185 },
   {  "MQRC_INCONSISTENT_UOW"           ,       2245 },
   {  "MQRC_INDEX_ERROR"                ,       2314 },
   {  "MQRC_INDEX_NOT_PRESENT"          ,       2306 },
   {  "MQRC_INHIBIT_VALUE_ERROR"        ,       2020 },
   {  "MQRC_INITIALIZATION_FAILED"      ,       2286 },
   {  "MQRC_INQUIRY_COMMAND_ERROR"      ,       2324 },
   {  "MQRC_INSTALLATION_MISMATCH"      ,       2583 },
   {  "MQRC_INSTALLATION_MISSING"       ,       2589 },
   {  "MQRC_INSUFFICIENT_BUFFER"        ,       6113 },
   {  "MQRC_INSUFFICIENT_DATA"          ,       6114 },
   {  "MQRC_INT_ATTRS_ARRAY_ERROR"      ,       2023 },
   {  "MQRC_INT_ATTR_COUNT_ERROR"       ,       2021 },
   {  "MQRC_INT_ATTR_COUNT_TOO_SMALL"   ,       2022 },
   {  "MQRC_INVALID_DESTINATION"        ,       2522 },
   {  "MQRC_INVALID_MSG_UNDER_CURSOR"   ,       2246 },
   {  "MQRC_INVALID_SUBSCRIPTION"       ,       2523 },
   {  "MQRC_ITEM_COUNT_ERROR"           ,       2316 },
   {  "MQRC_ITEM_TYPE_ERROR"            ,       2327 },
   {  "MQRC_ITEM_VALUE_ERROR"           ,       2319 },
   {  "MQRC_JMS_FORMAT_ERROR"           ,       2364 },
   {  "MQRC_JSSE_ERROR"                 ,       2397 },
   {  "MQRC_KEY_REPOSITORY_ERROR"       ,       2381 },
   {  "MQRC_LDAP_PASSWORD_ERROR"        ,       2390 },
   {  "MQRC_LDAP_USER_NAME_ERROR"       ,       2388 },
   {  "MQRC_LDAP_USER_NAME_LENGTH_ERR"  ,       2389 },
   {  "MQRC_LOCAL_UOW_CONFLICT"         ,       2352 },
   {  "MQRC_LOGGER_STATUS"              ,       2411 },
   {  "MQRC_LOOPING_PUBLICATION"        ,       2541 },
   {  "MQRC_MATCH_OPTIONS_ERROR"        ,       2247 },
   {  "MQRC_MAX_CONNS_LIMIT_REACHED"    ,       2025 },
   {  "MQRC_MAX_MSG_LENGTH_ERROR"       ,       2485 },
   {  "MQRC_MCAST_PUB_STATUS"           ,       2571 },
   {  "MQRC_MCAST_SUB_STATUS"           ,       2572 },
   {  "MQRC_MDE_ERROR"                  ,       2248 },
   {  "MQRC_MD_ERROR"                   ,       2026 },
   {  "MQRC_MHBO_ERROR"                 ,       2501 },
   {  "MQRC_MISSING_REPLY_TO_Q"         ,       2027 },
   {  "MQRC_MISSING_WIH"                ,       2332 },
   {  "MQRC_MIXED_CONTENT_NOT_ALLOWED"  ,       2498 },
   {  "MQRC_MODULE_ENTRY_NOT_FOUND"     ,       2497 },
   {  "MQRC_MODULE_INVALID"             ,       2496 },
   {  "MQRC_MODULE_NOT_FOUND"           ,       2495 },
   {  "MQRC_MSG_FLAGS_ERROR"            ,       2249 },
   {  "MQRC_MSG_HANDLE_COPY_FAILURE"    ,       2532 },
   {  "MQRC_MSG_HANDLE_IN_USE"          ,       2499 },
   {  "MQRC_MSG_ID_ERROR"               ,       2206 },
   {  "MQRC_MSG_MARKED_BROWSE_CO_OP"    ,       2200 },
   {  "MQRC_MSG_NOT_ALLOWED_IN_GROUP"   ,       2417 },
   {  "MQRC_MSG_NOT_MATCHED"            ,       2363 },
   {  "MQRC_MSG_SEQ_NUMBER_ERROR"       ,       2250 },
   {  "MQRC_MSG_TOKEN_ERROR"            ,       2331 },
   {  "MQRC_MSG_TOO_BIG_FOR_CHANNEL"    ,       2218 },
   {  "MQRC_MSG_TOO_BIG_FOR_Q"          ,       2030 },
   {  "MQRC_MSG_TOO_BIG_FOR_Q_MGR"      ,       2031 },
   {  "MQRC_MSG_TYPE_ERROR"             ,       2029 },
   {  "MQRC_MULTICAST_CONFIG_ERROR"     ,       2564 },
   {  "MQRC_MULTICAST_INTERFACE_ERROR"  ,       2565 },
   {  "MQRC_MULTICAST_INTERNAL_ERROR"   ,       2567 },
   {  "MQRC_MULTICAST_ONLY"             ,       2560 },
   {  "MQRC_MULTICAST_SEND_ERROR"       ,       2566 },
   {  "MQRC_MULTIPLE_INSTANCE_ERROR"    ,       2301 },
   {  "MQRC_MULTIPLE_REASONS"           ,       2136 },
   {  "MQRC_NAME_IN_USE"                ,       2201 },
   {  "MQRC_NAME_NOT_VALID_FOR_TYPE"    ,       2194 },
   {  "MQRC_NEGATIVE_LENGTH"            ,       6117 },
   {  "MQRC_NEGATIVE_OFFSET"            ,       6118 },
   {  "MQRC_NESTED_BAG_NOT_SUPPORTED"   ,       2325 },
   {  "MQRC_NESTED_SELECTOR_ERROR"      ,       2419 },
   {  "MQRC_NEXT_OFFSET_ERROR"          ,       2358 },
   {  "MQRC_NEXT_RECORD_ERROR"          ,       2361 },
   {  "MQRC_NONE"                       ,          0 },
   {  "MQRC_NOT_AUTHORIZED"             ,       2035 },
   {  "MQRC_NOT_CONNECTED"              ,       6124 },
   {  "MQRC_NOT_CONVERTED"              ,       2119 },
   {  "MQRC_NOT_OPEN"                   ,       6125 },
   {  "MQRC_NOT_OPEN_FOR_BROWSE"        ,       2036 },
   {  "MQRC_NOT_OPEN_FOR_INPUT"         ,       2037 },
   {  "MQRC_NOT_OPEN_FOR_INQUIRE"       ,       2038 },
   {  "MQRC_NOT_OPEN_FOR_OUTPUT"        ,       2039 },
   {  "MQRC_NOT_OPEN_FOR_PASS_ALL"      ,       2093 },
   {  "MQRC_NOT_OPEN_FOR_PASS_IDENT"    ,       2094 },
   {  "MQRC_NOT_OPEN_FOR_SET"           ,       2040 },
   {  "MQRC_NOT_OPEN_FOR_SET_ALL"       ,       2095 },
   {  "MQRC_NOT_OPEN_FOR_SET_IDENT"     ,       2096 },
   {  "MQRC_NOT_PRIVILEGED"             ,       2584 },
   {  "MQRC_NO_BUFFER"                  ,       6110 },
   {  "MQRC_NO_CALLBACKS_ACTIVE"        ,       2446 },
   {  "MQRC_NO_CONNECTION_REFERENCE"    ,       6109 },
   {  "MQRC_NO_DATA_AVAILABLE"          ,       2379 },
   {  "MQRC_NO_DESTINATIONS_AVAILABLE"  ,       2270 },
   {  "MQRC_NO_EXTERNAL_PARTICIPANTS"   ,       2121 },
   {  "MQRC_NO_MSG_AVAILABLE"           ,       2033 },
   {  "MQRC_NO_MSG_LOCKED"              ,       2209 },
   {  "MQRC_NO_MSG_UNDER_CURSOR"        ,       2034 },
   {  "MQRC_NO_RECORD_AVAILABLE"        ,       2359 },
   {  "MQRC_NO_RETAINED_MSG"            ,       2437 },
   {  "MQRC_NO_SUBSCRIPTION"            ,       2428 },
   {  "MQRC_NO_SUBS_MATCHED"            ,       2550 },
   {  "MQRC_NULL_POINTER"               ,       6108 },
   {  "MQRC_OBJECT_ALREADY_EXISTS"      ,       2100 },
   {  "MQRC_OBJECT_CHANGED"             ,       2041 },
   {  "MQRC_OBJECT_DAMAGED"             ,       2101 },
   {  "MQRC_OBJECT_IN_USE"              ,       2042 },
   {  "MQRC_OBJECT_LEVEL_INCOMPATIBLE"  ,       2360 },
   {  "MQRC_OBJECT_NAME_ERROR"          ,       2152 },
   {  "MQRC_OBJECT_NOT_UNIQUE"          ,       2343 },
   {  "MQRC_OBJECT_Q_MGR_NAME_ERROR"    ,       2153 },
   {  "MQRC_OBJECT_RECORDS_ERROR"       ,       2155 },
   {  "MQRC_OBJECT_STRING_ERROR"        ,       2441 },
   {  "MQRC_OBJECT_TYPE_ERROR"          ,       2043 },
   {  "MQRC_OCSP_URL_ERROR"             ,       2553 },
   {  "MQRC_OD_ERROR"                   ,       2044 },
   {  "MQRC_OFFSET_ERROR"               ,       2251 },
   {  "MQRC_OPEN_FAILED"                ,       2137 },
   {  "MQRC_OPERATION_ERROR"            ,       2488 },
   {  "MQRC_OPERATION_NOT_ALLOWED"      ,       2534 },
   {  "MQRC_OPTIONS_CHANGED"            ,       2457 },
   {  "MQRC_OPTIONS_ERROR"              ,       2046 },
   {  "MQRC_OPTION_ENVIRONMENT_ERROR"   ,       2274 },
   {  "MQRC_OPTION_NOT_VALID_FOR_TYPE"  ,       2045 },
   {  "MQRC_ORIGINAL_LENGTH_ERROR"      ,       2252 },
   {  "MQRC_OUTCOME_MIXED"              ,       2123 },
   {  "MQRC_OUTCOME_PENDING"            ,       2124 },
   {  "MQRC_OUT_SELECTOR_ERROR"         ,       2310 },
   {  "MQRC_PAGESET_ERROR"              ,       2193 },
   {  "MQRC_PARAMETER_MISSING"          ,       2321 },
   {  "MQRC_PARTIALLY_CONVERTED"        ,       2272 },
   {  "MQRC_PARTICIPANT_NOT_AVAILABLE"  ,       2122 },
   {  "MQRC_PARTICIPANT_NOT_DEFINED"    ,       2372 },
   {  "MQRC_PASSWORD_PROTECTION_ERROR"  ,       2594 },
   {  "MQRC_PCF_ERROR"                  ,       2149 },
   {  "MQRC_PD_ERROR"                   ,       2482 },
   {  "MQRC_PERSISTENCE_ERROR"          ,       2047 },
   {  "MQRC_PERSISTENT_NOT_ALLOWED"     ,       2048 },
   {  "MQRC_PMO_ERROR"                  ,       2173 },
   {  "MQRC_PMO_RECORD_FLAGS_ERROR"     ,       2158 },
   {  "MQRC_PRECONN_EXIT_ERROR"         ,       2575 },
   {  "MQRC_PRECONN_EXIT_LOAD_ERROR"    ,       2573 },
   {  "MQRC_PRECONN_EXIT_NOT_FOUND"     ,       2574 },
   {  "MQRC_PRIORITY_ERROR"             ,       2050 },
   {  "MQRC_PRIORITY_EXCEEDS_MAXIMUM"   ,       2049 },
   {  "MQRC_PROPERTIES_DISABLED"        ,       2586 },
   {  "MQRC_PROPERTIES_TOO_BIG"         ,       2478 },
   {  "MQRC_PROPERTY_NAME_ERROR"        ,       2442 },
   {  "MQRC_PROPERTY_NAME_LENGTH_ERR"   ,       2513 },
   {  "MQRC_PROPERTY_NAME_TOO_BIG"      ,       2465 },
   {  "MQRC_PROPERTY_NOT_AVAILABLE"     ,       2471 },
   {  "MQRC_PROPERTY_TYPE_ERROR"        ,       2473 },
   {  "MQRC_PROPERTY_VALUE_TOO_BIG"     ,       2469 },
   {  "MQRC_PROP_CONV_NOT_SUPPORTED"    ,       2470 },
   {  "MQRC_PROP_NAME_NOT_CONVERTED"    ,       2492 },
   {  "MQRC_PROP_NUMBER_FORMAT_ERROR"   ,       2472 },
   {  "MQRC_PROP_TYPE_NOT_SUPPORTED"    ,       2467 },
   {  "MQRC_PROP_VALUE_NOT_CONVERTED"   ,       2466 },
   {  "MQRC_PUBLICATION_FAILURE"        ,       2502 },
   {  "MQRC_PUBLISH_EXIT_ERROR"         ,       2557 },
   {  "MQRC_PUBSUB_INHIBITED"           ,       2531 },
   {  "MQRC_PUT_INHIBITED"              ,       2051 },
   {  "MQRC_PUT_MSG_RECORDS_ERROR"      ,       2159 },
   {  "MQRC_PUT_NOT_RETAINED"           ,       2479 },
   {  "MQRC_Q_ALREADY_EXISTS"           ,       2290 },
   {  "MQRC_Q_DELETED"                  ,       2052 },
   {  "MQRC_Q_DEPTH_HIGH"               ,       2224 },
   {  "MQRC_Q_DEPTH_LOW"                ,       2225 },
   {  "MQRC_Q_FULL"                     ,       2053 },
   {  "MQRC_Q_INDEX_TYPE_ERROR"         ,       2394 },
   {  "MQRC_Q_MGR_ACTIVE"               ,       2222 },
   {  "MQRC_Q_MGR_NAME_ERROR"           ,       2058 },
   {  "MQRC_Q_MGR_NOT_ACTIVE"           ,       2223 },
   {  "MQRC_Q_MGR_NOT_AVAILABLE"        ,       2059 },
   {  "MQRC_Q_MGR_QUIESCING"            ,       2161 },
   {  "MQRC_Q_MGR_STOPPING"             ,       2162 },
   {  "MQRC_Q_NOT_EMPTY"                ,       2055 },
   {  "MQRC_Q_SERVICE_INTERVAL_HIGH"    ,       2226 },
   {  "MQRC_Q_SERVICE_INTERVAL_OK"      ,       2227 },
   {  "MQRC_Q_SPACE_NOT_AVAILABLE"      ,       2056 },
   {  "MQRC_Q_TYPE_ERROR"               ,       2057 },
   {  "MQRC_RAS_PROPERTY_ERROR"         ,       2229 },
   {  "MQRC_READ_AHEAD_MSGS"            ,       2458 },
   {  "MQRC_RECONNECTED"                ,       2545 },
   {  "MQRC_RECONNECTING"               ,       2544 },
   {  "MQRC_RECONNECT_FAILED"           ,       2548 },
   {  "MQRC_RECONNECT_INCOMPATIBLE"     ,       2547 },
   {  "MQRC_RECONNECT_QMID_MISMATCH"    ,       2546 },
   {  "MQRC_RECONNECT_Q_MGR_REQD"       ,       2555 },
   {  "MQRC_RECONNECT_TIMED_OUT"        ,       2556 },
   {  "MQRC_RECS_PRESENT_ERROR"         ,       2154 },
   {  "MQRC_REFERENCE_ERROR"            ,       6129 },
   {  "MQRC_REMOTE_Q_NAME_ERROR"        ,       2184 },
   {  "MQRC_REOPEN_EXCL_INPUT_ERROR"    ,       6100 },
   {  "MQRC_REOPEN_INQUIRE_ERROR"       ,       6101 },
   {  "MQRC_REOPEN_SAVED_CONTEXT_ERR"   ,       6102 },
   {  "MQRC_REOPEN_TEMPORARY_Q_ERROR"   ,       6103 },
   {  "MQRC_REPORT_OPTIONS_ERROR"       ,       2061 },
   {  "MQRC_RESERVED_VALUE_ERROR"       ,       2378 },
   {  "MQRC_RESOURCE_PROBLEM"           ,       2102 },
   {  "MQRC_RESPONSE_RECORDS_ERROR"     ,       2156 },
   {  "MQRC_RES_OBJECT_STRING_ERROR"    ,       2520 },
   {  "MQRC_RETAINED_MSG_Q_ERROR"       ,       2525 },
   {  "MQRC_RETAINED_NOT_DELIVERED"     ,       2526 },
   {  "MQRC_RFH_COMMAND_ERROR"          ,       2336 },
   {  "MQRC_RFH_DUPLICATE_PARM"         ,       2338 },
   {  "MQRC_RFH_ERROR"                  ,       2334 },
   {  "MQRC_RFH_FORMAT_ERROR"           ,       2421 },
   {  "MQRC_RFH_HEADER_FIELD_ERROR"     ,       2228 },
   {  "MQRC_RFH_PARM_ERROR"             ,       2337 },
   {  "MQRC_RFH_PARM_MISSING"           ,       2339 },
   {  "MQRC_RFH_RESTRICTED_FORMAT_ERR"  ,       2527 },
   {  "MQRC_RFH_STRING_ERROR"           ,       2335 },
   {  "MQRC_RMH_ERROR"                  ,       2220 },
   {  "MQRC_SCO_ERROR"                  ,       2380 },
   {  "MQRC_SD_ERROR"                   ,       2424 },
   {  "MQRC_SECOND_MARK_NOT_ALLOWED"    ,       2062 },
   {  "MQRC_SECURITY_ERROR"             ,       2063 },
   {  "MQRC_SEGMENTATION_NOT_ALLOWED"   ,       2443 },
   {  "MQRC_SEGMENTS_NOT_SUPPORTED"     ,       2365 },
   {  "MQRC_SEGMENT_LENGTH_ZERO"        ,       2253 },
   {  "MQRC_SELECTION_NOT_AVAILABLE"    ,       2551 },
   {  "MQRC_SELECTION_STRING_ERROR"     ,       2519 },
   {  "MQRC_SELECTOR_ALWAYS_FALSE"      ,       2504 },
   {  "MQRC_SELECTOR_COUNT_ERROR"       ,       2065 },
   {  "MQRC_SELECTOR_ERROR"             ,       2067 },
   {  "MQRC_SELECTOR_INVALID_FOR_TYPE"  ,       2516 },
   {  "MQRC_SELECTOR_LIMIT_EXCEEDED"    ,       2066 },
   {  "MQRC_SELECTOR_NOT_ALTERABLE"     ,       2524 },
   {  "MQRC_SELECTOR_NOT_FOR_TYPE"      ,       2068 },
   {  "MQRC_SELECTOR_NOT_PRESENT"       ,       2309 },
   {  "MQRC_SELECTOR_NOT_SUPPORTED"     ,       2318 },
   {  "MQRC_SELECTOR_NOT_UNIQUE"        ,       2305 },
   {  "MQRC_SELECTOR_OUT_OF_RANGE"      ,       2304 },
   {  "MQRC_SELECTOR_SYNTAX_ERROR"      ,       2459 },
   {  "MQRC_SELECTOR_TYPE_ERROR"        ,       2299 },
   {  "MQRC_SELECTOR_WRONG_TYPE"        ,       2312 },
   {  "MQRC_SERVICE_ERROR"              ,       2289 },
   {  "MQRC_SERVICE_NOT_AVAILABLE"      ,       2285 },
   {  "MQRC_SIGNAL1_ERROR"              ,       2099 },
   {  "MQRC_SIGNAL_OUTSTANDING"         ,       2069 },
   {  "MQRC_SIGNAL_REQUEST_ACCEPTED"    ,       2070 },
   {  "MQRC_SMPO_ERROR"                 ,       2463 },
   {  "MQRC_SOAP_AXIS_ERROR"            ,       2211 },
   {  "MQRC_SOAP_DOTNET_ERROR"          ,       2210 },
   {  "MQRC_SOAP_URL_ERROR"             ,       2212 },
   {  "MQRC_SOURCE_BUFFER_ERROR"        ,       2145 },
   {  "MQRC_SOURCE_CCSID_ERROR"         ,       2111 },
   {  "MQRC_SOURCE_DECIMAL_ENC_ERROR"   ,       2113 },
   {  "MQRC_SOURCE_FLOAT_ENC_ERROR"     ,       2114 },
   {  "MQRC_SOURCE_INTEGER_ENC_ERROR"   ,       2112 },
   {  "MQRC_SOURCE_LENGTH_ERROR"        ,       2143 },
   {  "MQRC_SRC_ENV_ERROR"              ,       2261 },
   {  "MQRC_SRC_NAME_ERROR"             ,       2262 },
   {  "MQRC_SRO_ERROR"                  ,       2438 },
   {  "MQRC_SSL_ALREADY_INITIALIZED"    ,       2391 },
   {  "MQRC_SSL_ALT_PROVIDER_REQUIRED"  ,       2570 },
   {  "MQRC_SSL_CERTIFICATE_REVOKED"    ,       2401 },
   {  "MQRC_SSL_CERT_STORE_ERROR"       ,       2402 },
   {  "MQRC_SSL_CONFIG_ERROR"           ,       2392 },
   {  "MQRC_SSL_INITIALIZATION_ERROR"   ,       2393 },
   {  "MQRC_SSL_KEY_RESET_ERROR"        ,       2409 },
   {  "MQRC_SSL_NOT_ALLOWED"            ,       2396 },
   {  "MQRC_SSL_PEER_NAME_ERROR"        ,       2399 },
   {  "MQRC_SSL_PEER_NAME_MISMATCH"     ,       2398 },
   {  "MQRC_STANDBY_Q_MGR"              ,       2543 },
   {  "MQRC_STAT_TYPE_ERROR"            ,       2430 },
   {  "MQRC_STOPPED_BY_CLUSTER_EXIT"    ,       2188 },
   {  "MQRC_STORAGE_CLASS_ERROR"        ,       2105 },
   {  "MQRC_STORAGE_MEDIUM_FULL"        ,       2192 },
   {  "MQRC_STORAGE_NOT_AVAILABLE"      ,       2071 },
   {  "MQRC_STRING_ERROR"               ,       2307 },
   {  "MQRC_STRING_LENGTH_ERROR"        ,       2323 },
   {  "MQRC_STRING_TRUNCATED"           ,       2311 },
   {  "MQRC_STRUC_ID_ERROR"             ,       6107 },
   {  "MQRC_STRUC_LENGTH_ERROR"         ,       6123 },
   {  "MQRC_STS_ERROR"                  ,       2426 },
   {  "MQRC_SUBLEVEL_NOT_ALTERABLE"     ,       2512 },
   {  "MQRC_SUBSCRIPTION_CHANGE"        ,       2581 },
   {  "MQRC_SUBSCRIPTION_CREATE"        ,       2579 },
   {  "MQRC_SUBSCRIPTION_DELETE"        ,       2580 },
   {  "MQRC_SUBSCRIPTION_IN_USE"        ,       2429 },
   {  "MQRC_SUBSCRIPTION_REFRESH"       ,       2582 },
   {  "MQRC_SUB_ALREADY_EXISTS"         ,       2432 },
   {  "MQRC_SUB_INHIBITED"              ,       2503 },
   {  "MQRC_SUB_JOIN_NOT_ALTERABLE"     ,      29440 },
   {  "MQRC_SUB_NAME_ERROR"             ,       2440 },
   {  "MQRC_SUB_USER_DATA_ERROR"        ,       2431 },
   {  "MQRC_SUITE_B_ERROR"              ,       2592 },
   {  "MQRC_SUPPRESSED_BY_EXIT"         ,       2109 },
   {  "MQRC_SYNCPOINT_LIMIT_REACHED"    ,       2024 },
   {  "MQRC_SYNCPOINT_NOT_ALLOWED"      ,       2569 },
   {  "MQRC_SYNCPOINT_NOT_AVAILABLE"    ,       2072 },
   {  "MQRC_SYSTEM_BAG_NOT_ALTERABLE"   ,       2315 },
   {  "MQRC_SYSTEM_BAG_NOT_DELETABLE"   ,       2328 },
   {  "MQRC_SYSTEM_ITEM_NOT_ALTERABLE"  ,       2302 },
   {  "MQRC_SYSTEM_ITEM_NOT_DELETABLE"  ,       2329 },
   {  "MQRC_TARGET_BUFFER_ERROR"        ,       2146 },
   {  "MQRC_TARGET_CCSID_ERROR"         ,       2115 },
   {  "MQRC_TARGET_DECIMAL_ENC_ERROR"   ,       2117 },
   {  "MQRC_TARGET_FLOAT_ENC_ERROR"     ,       2118 },
   {  "MQRC_TARGET_INTEGER_ENC_ERROR"   ,       2116 },
   {  "MQRC_TARGET_LENGTH_ERROR"        ,       2144 },
   {  "MQRC_TERMINATION_FAILED"         ,       2287 },
   {  "MQRC_TMC_ERROR"                  ,       2191 },
   {  "MQRC_TM_ERROR"                   ,       2265 },
   {  "MQRC_TOPIC_NOT_ALTERABLE"        ,       2510 },
   {  "MQRC_TOPIC_STRING_ERROR"         ,       2425 },
   {  "MQRC_TRIGGER_CONTROL_ERROR"      ,       2075 },
   {  "MQRC_TRIGGER_DEPTH_ERROR"        ,       2076 },
   {  "MQRC_TRIGGER_MSG_PRIORITY_ERR"   ,       2077 },
   {  "MQRC_TRIGGER_TYPE_ERROR"         ,       2078 },
   {  "MQRC_TRUNCATED_MSG_ACCEPTED"     ,       2079 },
   {  "MQRC_TRUNCATED_MSG_FAILED"       ,       2080 },
   {  "MQRC_UCS2_CONVERSION_ERROR"      ,       2341 },
   {  "MQRC_UNEXPECTED_ERROR"           ,       2195 },
   {  "MQRC_UNIT_OF_WORK_NOT_STARTED"   ,       2232 },
   {  "MQRC_UNKNOWN_ALIAS_BASE_Q"       ,       2082 },
   {  "MQRC_UNKNOWN_AUTH_ENTITY"        ,       2293 },
   {  "MQRC_UNKNOWN_CHANNEL_NAME"       ,       2540 },
   {  "MQRC_UNKNOWN_COMPONENT_NAME"     ,       2410 },
   {  "MQRC_UNKNOWN_DEF_XMIT_Q"         ,       2197 },
   {  "MQRC_UNKNOWN_ENTITY"             ,       2292 },
   {  "MQRC_UNKNOWN_OBJECT_NAME"        ,       2085 },
   {  "MQRC_UNKNOWN_OBJECT_Q_MGR"       ,       2086 },
   {  "MQRC_UNKNOWN_Q_NAME"             ,       2288 },
   {  "MQRC_UNKNOWN_REF_OBJECT"         ,       2294 },
   {  "MQRC_UNKNOWN_REMOTE_Q_MGR"       ,       2087 },
   {  "MQRC_UNKNOWN_REPORT_OPTION"      ,       2104 },
   {  "MQRC_UNKNOWN_XMIT_Q"             ,       2196 },
   {  "MQRC_UNSUPPORTED_CIPHER_SUITE"   ,       2400 },
   {  "MQRC_UNSUPPORTED_PROPERTY"       ,       2490 },
   {  "MQRC_UOW_CANCELED"               ,       2297 },
   {  "MQRC_UOW_COMMITTED"              ,       2408 },
   {  "MQRC_UOW_ENLISTMENT_ERROR"       ,       2354 },
   {  "MQRC_UOW_IN_PROGRESS"            ,       2128 },
   {  "MQRC_UOW_MIX_NOT_SUPPORTED"      ,       2355 },
   {  "MQRC_UOW_NOT_AVAILABLE"          ,       2255 },
   {  "MQRC_USER_ID_NOT_AVAILABLE"      ,       2291 },
   {  "MQRC_WAIT_INTERVAL_ERROR"        ,       2090 },
   {  "MQRC_WIH_ERROR"                  ,       2333 },
   {  "MQRC_WRONG_CF_LEVEL"             ,       2366 },
   {  "MQRC_WRONG_GMO_VERSION"          ,       2256 },
   {  "MQRC_WRONG_MD_VERSION"           ,       2257 },
   {  "MQRC_WRONG_VERSION"              ,       6128 },
   {  "MQRC_WXP_ERROR"                  ,       2356 },
   {  "MQRC_XEPO_ERROR"                 ,       2507 },
   {  "MQRC_XMIT_Q_TYPE_ERROR"          ,       2091 },
   {  "MQRC_XMIT_Q_USAGE_ERROR"         ,       2092 },
   {  "MQRC_XQH_ERROR"                  ,       2260 },
   {  "MQRC_XR_NOT_AVAILABLE"           ,       6130 },
   {  "MQRC_XWAIT_CANCELED"             ,       2107 },
   {  "MQRC_XWAIT_ERROR"                ,       2108 },
   {  "MQRC_ZERO_LENGTH"                ,       6116 },
   {  "MQRDNS_DISABLED"                 ,          1 },
   {  "MQRDNS_ENABLED"                  ,          0 },
   {  "MQRD_NO_DELAY"                   ,          0 },
   {  "MQRD_NO_RECONNECT"               ,         -1 },
   {  "MQREADA_BACKLOG"                 ,          4 },
   {  "MQREADA_DISABLED"                ,          2 },
   {  "MQREADA_INHIBITED"               ,          3 },
   {  "MQREADA_NO"                      ,          0 },
   {  "MQREADA_YES"                     ,          1 },
   {  "MQRECAUTO_NO"                    ,          0 },
   {  "MQRECAUTO_YES"                   ,          1 },
   {  "MQRECORDING_DISABLED"            ,          0 },
   {  "MQRECORDING_MSG"                 ,          2 },
   {  "MQRECORDING_Q"                   ,          1 },
   {  "MQREGO_ADD_NAME"                 ,      16384 },
   {  "MQREGO_ANONYMOUS"                ,          2 },
   {  "MQREGO_CORREL_ID_AS_IDENTITY"    ,          1 },
   {  "MQREGO_DEREGISTER_ALL"           ,         64 },
   {  "MQREGO_DIRECT_REQUESTS"          ,          8 },
   {  "MQREGO_DUPLICATES_OK"            ,        512 },
   {  "MQREGO_FULL_RESPONSE"            ,      65536 },
   {  "MQREGO_INCLUDE_STREAM_NAME"      ,        128 },
   {  "MQREGO_INFORM_IF_RETAINED"       ,        256 },
   {  "MQREGO_JOIN_EXCLUSIVE"           ,     262144 },
   {  "MQREGO_JOIN_SHARED"              ,     131072 },
   {  "MQREGO_LEAVE_ONLY"               ,     524288 },
   {  "MQREGO_LOCAL"                    ,          4 },
   {  "MQREGO_LOCKED"                   ,    2097152 },
   {  "MQREGO_NEW_PUBLICATIONS_ONLY"    ,         16 },
   {  "MQREGO_NONE"                     ,          0 },
   {  "MQREGO_NON_PERSISTENT"           ,       1024 },
   {  "MQREGO_NO_ALTERATION"            ,      32768 },
   {  "MQREGO_PERSISTENT"               ,       2048 },
   {  "MQREGO_PERSISTENT_AS_PUBLISH"    ,       4096 },
   {  "MQREGO_PERSISTENT_AS_Q"          ,       8192 },
   {  "MQREGO_PUBLISH_ON_REQUEST_ONLY"  ,         32 },
   {  "MQREGO_VARIABLE_USER_ID"         ,    1048576 },
   {  "MQREORG_DISABLED"                ,          0 },
   {  "MQREORG_ENABLED"                 ,          1 },
   {  "MQRFH2_CURRENT_LENGTH"           ,         36 },
   {  "MQRFH2_LENGTH_2"                 ,         36 },
   {  "MQRFH_CURRENT_LENGTH"            ,         32 },
   {  "MQRFH_FLAGS_RESTRICTED_MASK"     ,     -65536 },
   {  "MQRFH_LENGTH_1"                  ,         32 },
   {  "MQRFH_NONE"                      ,          0 },
   {  "MQRFH_NO_FLAGS"                  ,          0 },
   {  "MQRFH_STRUC_LENGTH_FIXED"        ,         32 },
   {  "MQRFH_STRUC_LENGTH_FIXED_2"      ,         36 },
   {  "MQRFH_VERSION_1"                 ,          1 },
   {  "MQRFH_VERSION_2"                 ,          2 },
   {  "MQRL_UNDEFINED"                  ,         -1 },
   {  "MQRMHF_LAST"                     ,          1 },
   {  "MQRMHF_NOT_LAST"                 ,          0 },
   {  "MQRMH_CURRENT_LENGTH"            ,        108 },
   {  "MQRMH_CURRENT_VERSION"           ,          1 },
   {  "MQRMH_LENGTH_1"                  ,        108 },
   {  "MQRMH_VERSION_1"                 ,          1 },
   {  "MQROUTE_ACCUMULATE_AND_REPLY"    ,      65541 },
   {  "MQROUTE_ACCUMULATE_IN_MSG"       ,      65540 },
   {  "MQROUTE_ACCUMULATE_NONE"         ,      65539 },
   {  "MQROUTE_DELIVER_NO"              ,       8192 },
   {  "MQROUTE_DELIVER_REJ_UNSUP_MASK"  ,     -65536 },
   {  "MQROUTE_DELIVER_YES"             ,       4096 },
   {  "MQROUTE_DETAIL_HIGH"             ,         32 },
   {  "MQROUTE_DETAIL_LOW"              ,          2 },
   {  "MQROUTE_DETAIL_MEDIUM"           ,          8 },
   {  "MQROUTE_FORWARD_ALL"             ,        256 },
   {  "MQROUTE_FORWARD_IF_SUPPORTED"    ,        512 },
   {  "MQROUTE_FORWARD_REJ_UNSUP_MASK"  ,     -65536 },
   {  "MQROUTE_UNLIMITED_ACTIVITIES"    ,          0 },
   {  "MQRO_ACCEPT_UNSUP_IF_XMIT_MASK"  ,     261888 },
   {  "MQRO_ACCEPT_UNSUP_MASK"          , -270532353 },
   {  "MQRO_ACTIVITY"                   ,          4 },
   {  "MQRO_COA"                        ,        256 },
   {  "MQRO_COA_WITH_DATA"              ,        768 },
   {  "MQRO_COA_WITH_FULL_DATA"         ,       1792 },
   {  "MQRO_COD"                        ,       2048 },
   {  "MQRO_COD_WITH_DATA"              ,       6144 },
   {  "MQRO_COD_WITH_FULL_DATA"         ,      14336 },
   {  "MQRO_COPY_MSG_ID_TO_CORREL_ID"   ,          0 },
   {  "MQRO_DEAD_LETTER_Q"              ,          0 },
   {  "MQRO_DISCARD_MSG"                ,  134217728 },
   {  "MQRO_EXCEPTION"                  ,   16777216 },
   {  "MQRO_EXCEPTION_WITH_DATA"        ,   50331648 },
   {  "MQRO_EXCEPTION_WITH_FULL_DATA"   ,  117440512 },
   {  "MQRO_EXPIRATION"                 ,    2097152 },
   {  "MQRO_EXPIRATION_WITH_DATA"       ,    6291456 },
   {  "MQRO_EXPIRATION_WITH_FULL_DATA"  ,   14680064 },
   {  "MQRO_NAN"                        ,          2 },
   {  "MQRO_NEW_MSG_ID"                 ,          0 },
   {  "MQRO_NONE"                       ,          0 },
   {  "MQRO_PAN"                        ,          1 },
   {  "MQRO_PASS_CORREL_ID"             ,         64 },
   {  "MQRO_PASS_DISCARD_AND_EXPIRY"    ,      16384 },
   {  "MQRO_PASS_MSG_ID"                ,        128 },
   {  "MQRO_REJECT_UNSUP_MASK"          ,  270270464 },
   {  "MQRP_NO"                         ,          0 },
   {  "MQRP_YES"                        ,          1 },
   {  "MQRQ_BRIDGE_STOPPED_ERROR"       ,         12 },
   {  "MQRQ_BRIDGE_STOPPED_OK"          ,         11 },
   {  "MQRQ_CAF_NOT_INSTALLED"          ,         28 },
   {  "MQRQ_CHANNEL_BLOCKED_ADDRESS"    ,         21 },
   {  "MQRQ_CHANNEL_BLOCKED_NOACCESS"   ,         23 },
   {  "MQRQ_CHANNEL_BLOCKED_USERID"     ,         22 },
   {  "MQRQ_CHANNEL_STOPPED_DISABLED"   ,         10 },
   {  "MQRQ_CHANNEL_STOPPED_ERROR"      ,          8 },
   {  "MQRQ_CHANNEL_STOPPED_OK"         ,          7 },
   {  "MQRQ_CHANNEL_STOPPED_RETRY"      ,          9 },
   {  "MQRQ_CLIENT_INST_LIMIT"          ,         27 },
   {  "MQRQ_CLOSE_NOT_AUTHORIZED"       ,          3 },
   {  "MQRQ_CMD_NOT_AUTHORIZED"         ,          4 },
   {  "MQRQ_CONN_NOT_AUTHORIZED"        ,          1 },
   {  "MQRQ_CSP_NOT_AUTHORIZED"         ,         29 },
   {  "MQRQ_FAILOVER_NOT_PERMITTED"     ,         31 },
   {  "MQRQ_FAILOVER_PERMITTED"         ,         30 },
   {  "MQRQ_MAX_ACTIVE_CHANNELS"        ,         24 },
   {  "MQRQ_MAX_CHANNELS"               ,         25 },
   {  "MQRQ_OPEN_NOT_AUTHORIZED"        ,          2 },
   {  "MQRQ_Q_MGR_QUIESCING"            ,          6 },
   {  "MQRQ_Q_MGR_STOPPING"             ,          5 },
   {  "MQRQ_SSL_CIPHER_SPEC_ERROR"      ,         14 },
   {  "MQRQ_SSL_CLIENT_AUTH_ERROR"      ,         15 },
   {  "MQRQ_SSL_HANDSHAKE_ERROR"        ,         13 },
   {  "MQRQ_SSL_PEER_NAME_ERROR"        ,         16 },
   {  "MQRQ_SSL_UNKNOWN_REVOCATION"     ,         19 },
   {  "MQRQ_STANDBY_ACTIVATED"          ,         32 },
   {  "MQRQ_SUB_DEST_NOT_AUTHORIZED"    ,         18 },
   {  "MQRQ_SUB_NOT_AUTHORIZED"         ,         17 },
   {  "MQRQ_SVRCONN_INST_LIMIT"         ,         26 },
   {  "MQRQ_SYS_CONN_NOT_AUTHORIZED"    ,         20 },
   {  "MQRT_CONFIGURATION"              ,          1 },
   {  "MQRT_EXPIRY"                     ,          2 },
   {  "MQRT_NSPROC"                     ,          3 },
   {  "MQRT_PROXYSUB"                   ,          4 },
   {  "MQRT_SUB_CONFIGURATION"          ,          5 },
   {  "MQRU_PUBLISH_ALL"                ,          2 },
   {  "MQRU_PUBLISH_ON_REQUEST"         ,          1 },
   {  "MQSBC_CURRENT_LENGTH (4 byte)"   ,        272 },
   {  "MQSBC_CURRENT_LENGTH (8 byte)"   ,        288 },
   {  "MQSBC_CURRENT_VERSION"           ,          1 },
   {  "MQSBC_LENGTH_1 (4 byte)"         ,        272 },
   {  "MQSBC_LENGTH_1 (8 byte)"         ,        288 },
   {  "MQSBC_VERSION_1"                 ,          1 },
   {  "MQSCA_NEVER_REQUIRED"            ,          2 },
   {  "MQSCA_OPTIONAL"                  ,          1 },
   {  "MQSCA_REQUIRED"                  ,          0 },
   {  "MQSCOPE_ALL"                     ,          0 },
   {  "MQSCOPE_AS_PARENT"               ,          1 },
   {  "MQSCOPE_QMGR"                    ,          4 },
   {  "MQSCO_CELL"                      ,          2 },
   {  "MQSCO_CURRENT_LENGTH (4 byte)"   ,        624 },
   {  "MQSCO_CURRENT_LENGTH (8 byte)"   ,        632 },
   {  "MQSCO_CURRENT_VERSION"           ,          5 },
   {  "MQSCO_LENGTH_1 (4 byte)"         ,        532 },
   {  "MQSCO_LENGTH_1 (8 byte)"         ,        536 },
   {  "MQSCO_LENGTH_2 (4 byte)"         ,        540 },
   {  "MQSCO_LENGTH_2 (8 byte)"         ,        544 },
   {  "MQSCO_LENGTH_3 (4 byte)"         ,        556 },
   {  "MQSCO_LENGTH_3 (8 byte)"         ,        560 },
   {  "MQSCO_LENGTH_4 (4 byte)"         ,        560 },
   {  "MQSCO_LENGTH_4 (8 byte)"         ,        568 },
   {  "MQSCO_LENGTH_5 (4 byte)"         ,        624 },
   {  "MQSCO_LENGTH_5 (8 byte)"         ,        632 },
   {  "MQSCO_Q_MGR"                     ,          1 },
   {  "MQSCO_RESET_COUNT_DEFAULT"       ,          0 },
   {  "MQSCO_VERSION_1"                 ,          1 },
   {  "MQSCO_VERSION_2"                 ,          2 },
   {  "MQSCO_VERSION_3"                 ,          3 },
   {  "MQSCO_VERSION_4"                 ,          4 },
   {  "MQSCO_VERSION_5"                 ,          5 },
   {  "MQSCYC_MIXED"                    ,          1 },
   {  "MQSCYC_UPPER"                    ,          0 },
   {  "MQSD_CURRENT_LENGTH (4 byte)"    ,        312 },
   {  "MQSD_CURRENT_LENGTH (8 byte)"    ,        344 },
   {  "MQSD_CURRENT_VERSION"            ,          1 },
   {  "MQSD_LENGTH_1 (4 byte)"          ,        312 },
   {  "MQSD_LENGTH_1 (8 byte)"          ,        344 },
   {  "MQSD_VERSION_1"                  ,          1 },
   {  "MQSECCOMM_ANON"                  ,          2 },
   {  "MQSECCOMM_NO"                    ,          0 },
   {  "MQSECCOMM_YES"                   ,          1 },
   {  "MQSECITEM_ALL"                   ,          0 },
   {  "MQSECITEM_MQADMIN"               ,          1 },
   {  "MQSECITEM_MQCMDS"                ,          6 },
   {  "MQSECITEM_MQCONN"                ,          5 },
   {  "MQSECITEM_MQNLIST"               ,          2 },
   {  "MQSECITEM_MQPROC"                ,          3 },
   {  "MQSECITEM_MQQUEUE"               ,          4 },
   {  "MQSECITEM_MXADMIN"               ,          7 },
   {  "MQSECITEM_MXNLIST"               ,          8 },
   {  "MQSECITEM_MXPROC"                ,          9 },
   {  "MQSECITEM_MXQUEUE"               ,         10 },
   {  "MQSECITEM_MXTOPIC"               ,         11 },
   {  "MQSECPROT_NONE"                  ,          0 },
   {  "MQSECPROT_SSLV30"                ,          1 },
   {  "MQSECPROT_TLSV10"                ,          2 },
   {  "MQSECPROT_TLSV12"                ,          4 },
   {  "MQSECSW_ALTERNATE_USER"          ,          7 },
   {  "MQSECSW_COMMAND"                 ,          8 },
   {  "MQSECSW_COMMAND_RESOURCES"       ,         11 },
   {  "MQSECSW_CONNECTION"              ,          9 },
   {  "MQSECSW_CONTEXT"                 ,          6 },
   {  "MQSECSW_NAMELIST"                ,          2 },
   {  "MQSECSW_OFF_ERROR"               ,         25 },
   {  "MQSECSW_OFF_FOUND"               ,         21 },
   {  "MQSECSW_OFF_NOT_FOUND"           ,         23 },
   {  "MQSECSW_ON_FOUND"                ,         22 },
   {  "MQSECSW_ON_NOT_FOUND"            ,         24 },
   {  "MQSECSW_ON_OVERRIDDEN"           ,         26 },
   {  "MQSECSW_PROCESS"                 ,          1 },
   {  "MQSECSW_Q"                       ,          3 },
   {  "MQSECSW_QSG"                     ,         16 },
   {  "MQSECSW_Q_MGR"                   ,         15 },
   {  "MQSECSW_SUBSYSTEM"               ,         10 },
   {  "MQSECSW_TOPIC"                   ,          4 },
   {  "MQSECTYPE_AUTHSERV"              ,          1 },
   {  "MQSECTYPE_CLASSES"               ,          3 },
   {  "MQSECTYPE_CONNAUTH"              ,          4 },
   {  "MQSECTYPE_SSL"                   ,          2 },
   {  "MQSELTYPE_EXTENDED"              ,          2 },
   {  "MQSELTYPE_NONE"                  ,          0 },
   {  "MQSELTYPE_STANDARD"              ,          1 },
   {  "MQSEL_ALL_SELECTORS"             ,     -30001 },
   {  "MQSEL_ALL_SYSTEM_SELECTORS"      ,     -30003 },
   {  "MQSEL_ALL_USER_SELECTORS"        ,     -30002 },
   {  "MQSEL_ANY_SELECTOR"              ,     -30001 },
   {  "MQSEL_ANY_SYSTEM_SELECTOR"       ,     -30003 },
   {  "MQSEL_ANY_USER_SELECTOR"         ,     -30002 },
   {  "MQSMPO_APPEND_PROPERTY"          ,          4 },
   {  "MQSMPO_CURRENT_LENGTH"           ,         20 },
   {  "MQSMPO_CURRENT_VERSION"          ,          1 },
   {  "MQSMPO_LENGTH_1"                 ,         20 },
   {  "MQSMPO_NONE"                     ,          0 },
   {  "MQSMPO_SET_FIRST"                ,          0 },
   {  "MQSMPO_SET_PROP_AFTER_CURSOR"    ,          2 },
   {  "MQSMPO_SET_PROP_BEFORE_CURSOR"   ,          8 },
   {  "MQSMPO_SET_PROP_UNDER_CURSOR"    ,          1 },
   {  "MQSMPO_VERSION_1"                ,          1 },
   {  "MQSO_ALTER"                      ,          1 },
   {  "MQSO_ALTERNATE_USER_AUTHORITY"   ,     262144 },
   {  "MQSO_ANY_USERID"                 ,        512 },
   {  "MQSO_CREATE"                     ,          2 },
   {  "MQSO_DURABLE"                    ,          8 },
   {  "MQSO_FAIL_IF_QUIESCING"          ,       8192 },
   {  "MQSO_FIXED_USERID"               ,        256 },
   {  "MQSO_GROUP_SUB"                  ,         16 },
   {  "MQSO_MANAGED"                    ,         32 },
   {  "MQSO_NEW_PUBLICATIONS_ONLY"      ,       4096 },
   {  "MQSO_NONE"                       ,          0 },
   {  "MQSO_NON_DURABLE"                ,          0 },
   {  "MQSO_NO_MULTICAST"               ,        128 },
   {  "MQSO_NO_READ_AHEAD"              ,  134217728 },
   {  "MQSO_PUBLICATIONS_ON_REQUEST"    ,       2048 },
   {  "MQSO_READ_AHEAD"                 ,  268435456 },
   {  "MQSO_READ_AHEAD_AS_Q_DEF"        ,          0 },
   {  "MQSO_RESUME"                     ,          4 },
   {  "MQSO_SCOPE_QMGR"                 ,   67108864 },
   {  "MQSO_SET_CORREL_ID"              ,    4194304 },
   {  "MQSO_SET_IDENTITY_CONTEXT"       ,         64 },
   {  "MQSO_WILDCARD_CHAR"              ,    1048576 },
   {  "MQSO_WILDCARD_TOPIC"             ,    2097152 },
   {  "MQSP_AVAILABLE"                  ,          1 },
   {  "MQSP_NOT_AVAILABLE"              ,          0 },
   {  "MQSQQM_IGNORE"                   ,          1 },
   {  "MQSQQM_USE"                      ,          0 },
   {  "MQSRO_CURRENT_LENGTH"            ,         16 },
   {  "MQSRO_CURRENT_VERSION"           ,          1 },
   {  "MQSRO_FAIL_IF_QUIESCING"         ,       8192 },
   {  "MQSRO_LENGTH_1"                  ,         16 },
   {  "MQSRO_NONE"                      ,          0 },
   {  "MQSRO_VERSION_1"                 ,          1 },
   {  "MQSR_ACTION_PUBLICATION"         ,          1 },
   {  "MQSSL_FIPS_NO"                   ,          0 },
   {  "MQSSL_FIPS_YES"                  ,          1 },
   {  "MQSTAT_TYPE_ASYNC_ERROR"         ,          0 },
   {  "MQSTAT_TYPE_RECONNECTION"        ,          1 },
   {  "MQSTAT_TYPE_RECONNECTION_ERROR"  ,          2 },
   {  "MQSTDBY_NOT_PERMITTED"           ,          0 },
   {  "MQSTDBY_PERMITTED"               ,          1 },
   {  "MQSTS_CURRENT_LENGTH (4 byte)"   ,        272 },
   {  "MQSTS_CURRENT_LENGTH (8 byte)"   ,        280 },
   {  "MQSTS_CURRENT_VERSION"           ,          2 },
   {  "MQSTS_LENGTH_1"                  ,        224 },
   {  "MQSTS_LENGTH_2 (4 byte)"         ,        272 },
   {  "MQSTS_LENGTH_2 (8 byte)"         ,        280 },
   {  "MQSTS_VERSION_1"                 ,          1 },
   {  "MQSTS_VERSION_2"                 ,          2 },
   {  "MQSUBTYPE_ADMIN"                 ,          2 },
   {  "MQSUBTYPE_ALL"                   ,         -1 },
   {  "MQSUBTYPE_API"                   ,          1 },
   {  "MQSUBTYPE_PROXY"                 ,          3 },
   {  "MQSUBTYPE_USER"                  ,         -2 },
   {  "MQSUB_DURABLE_ALL"               ,         -1 },
   {  "MQSUB_DURABLE_ALLOWED"           ,          1 },
   {  "MQSUB_DURABLE_AS_PARENT"         ,          0 },
   {  "MQSUB_DURABLE_INHIBITED"         ,          2 },
   {  "MQSUB_DURABLE_NO"                ,          2 },
   {  "MQSUB_DURABLE_YES"               ,          1 },
   {  "MQSUS_NO"                        ,          0 },
   {  "MQSUS_YES"                       ,          1 },
   {  "MQSVC_CONTROL_MANUAL"            ,          2 },
   {  "MQSVC_CONTROL_Q_MGR"             ,          0 },
   {  "MQSVC_CONTROL_Q_MGR_START"       ,          1 },
   {  "MQSVC_STATUS_RETRYING"           ,          4 },
   {  "MQSVC_STATUS_RUNNING"            ,          2 },
   {  "MQSVC_STATUS_STARTING"           ,          1 },
   {  "MQSVC_STATUS_STOPPED"            ,          0 },
   {  "MQSVC_STATUS_STOPPING"           ,          3 },
   {  "MQSVC_TYPE_COMMAND"              ,          0 },
   {  "MQSVC_TYPE_SERVER"               ,          1 },
   {  "MQSYNCPOINT_IFPER"               ,          1 },
   {  "MQSYNCPOINT_YES"                 ,          0 },
   {  "MQSYSOBJ_NO"                     ,          1 },
   {  "MQSYSOBJ_YES"                    ,          0 },
   {  "MQSYSP_ALLOC_BLK"                ,         20 },
   {  "MQSYSP_ALLOC_CYL"                ,         22 },
   {  "MQSYSP_ALLOC_TRK"                ,         21 },
   {  "MQSYSP_EXTENDED"                 ,          2 },
   {  "MQSYSP_NO"                       ,          0 },
   {  "MQSYSP_STATUS_ALLOC_ARCHIVE"     ,         34 },
   {  "MQSYSP_STATUS_AVAILABLE"         ,         32 },
   {  "MQSYSP_STATUS_BUSY"              ,         30 },
   {  "MQSYSP_STATUS_COPYING_BSDS"      ,         35 },
   {  "MQSYSP_STATUS_COPYING_LOG"       ,         36 },
   {  "MQSYSP_STATUS_PREMOUNT"          ,         31 },
   {  "MQSYSP_STATUS_UNKNOWN"           ,         33 },
   {  "MQSYSP_TYPE_ARCHIVE_TAPE"        ,         14 },
   {  "MQSYSP_TYPE_INITIAL"             ,         10 },
   {  "MQSYSP_TYPE_LOG_COPY"            ,         12 },
   {  "MQSYSP_TYPE_LOG_STATUS"          ,         13 },
   {  "MQSYSP_TYPE_SET"                 ,         11 },
   {  "MQSYSP_YES"                      ,          1 },
   {  "MQS_AVAIL_ERROR"                 ,          1 },
   {  "MQS_AVAIL_NORMAL"                ,          0 },
   {  "MQS_AVAIL_STOPPED"               ,          2 },
   {  "MQS_EXPANDST_FAILED"             ,          1 },
   {  "MQS_EXPANDST_MAXIMUM"            ,          2 },
   {  "MQS_EXPANDST_NORMAL"             ,          0 },
   {  "MQS_OPENMODE_NONE"               ,          0 },
   {  "MQS_OPENMODE_READONLY"           ,          1 },
   {  "MQS_OPENMODE_RECOVERY"           ,          3 },
   {  "MQS_OPENMODE_UPDATE"             ,          2 },
   {  "MQS_STATUS_ALLOCFAIL"            ,          5 },
   {  "MQS_STATUS_CLOSED"               ,          0 },
   {  "MQS_STATUS_CLOSING"              ,          1 },
   {  "MQS_STATUS_DATAFAIL"             ,          8 },
   {  "MQS_STATUS_NOTENABLED"           ,          4 },
   {  "MQS_STATUS_OPEN"                 ,          3 },
   {  "MQS_STATUS_OPENFAIL"             ,          6 },
   {  "MQS_STATUS_OPENING"              ,          2 },
   {  "MQS_STATUS_STGFAIL"              ,          7 },
   {  "MQTA_BLOCK"                      ,          1 },
   {  "MQTA_PASSTHRU"                   ,          2 },
   {  "MQTA_PROXY_SUB_FIRSTUSE"         ,          2 },
   {  "MQTA_PROXY_SUB_FORCE"            ,          1 },
   {  "MQTA_PUB_ALLOWED"                ,          2 },
   {  "MQTA_PUB_AS_PARENT"              ,          0 },
   {  "MQTA_PUB_INHIBITED"              ,          1 },
   {  "MQTA_SUB_ALLOWED"                ,          2 },
   {  "MQTA_SUB_AS_PARENT"              ,          0 },
   {  "MQTA_SUB_INHIBITED"              ,          1 },
   {  "MQTCPKEEP_NO"                    ,          0 },
   {  "MQTCPKEEP_YES"                   ,          1 },
   {  "MQTCPSTACK_MULTIPLE"             ,          1 },
   {  "MQTCPSTACK_SINGLE"               ,          0 },
   {  "MQTC_OFF"                        ,          0 },
   {  "MQTC_ON"                         ,          1 },
   {  "MQTIME_UNIT_MINS"                ,          0 },
   {  "MQTIME_UNIT_SECS"                ,          1 },
   {  "MQTMC2_CURRENT_LENGTH"           ,        732 },
   {  "MQTMC2_LENGTH_1"                 ,        684 },
   {  "MQTMC2_LENGTH_2"                 ,        732 },
   {  "MQTMC_CURRENT_LENGTH"            ,        684 },
   {  "MQTMC_LENGTH_1"                  ,        684 },
   {  "MQTM_CURRENT_LENGTH"             ,        684 },
   {  "MQTM_CURRENT_VERSION"            ,          1 },
   {  "MQTM_LENGTH_1"                   ,        684 },
   {  "MQTM_VERSION_1"                  ,          1 },
   {  "MQTOPT_ALL"                      ,          2 },
   {  "MQTOPT_CLUSTER"                  ,          1 },
   {  "MQTOPT_LOCAL"                    ,          0 },
   {  "MQTRAXSTR_NO"                    ,          0 },
   {  "MQTRAXSTR_YES"                   ,          1 },
   {  "MQTRIGGER_RESTART_NO"            ,          0 },
   {  "MQTRIGGER_RESTART_YES"           ,          1 },
   {  "MQTSCOPE_ALL"                    ,          2 },
   {  "MQTSCOPE_QMGR"                   ,          1 },
   {  "MQTT_DEPTH"                      ,          3 },
   {  "MQTT_EVERY"                      ,          2 },
   {  "MQTT_FIRST"                      ,          1 },
   {  "MQTT_NONE"                       ,          0 },
   {  "MQTYPE_AS_SET"                   ,          0 },
   {  "MQTYPE_BOOLEAN"                  ,          4 },
   {  "MQTYPE_BYTE_STRING"              ,          8 },
   {  "MQTYPE_FLOAT32"                  ,        256 },
   {  "MQTYPE_FLOAT64"                  ,        512 },
   {  "MQTYPE_INT16"                    ,         32 },
   {  "MQTYPE_INT32"                    ,         64 },
   {  "MQTYPE_INT64"                    ,        128 },
   {  "MQTYPE_INT8"                     ,         16 },
   {  "MQTYPE_LONG"                     ,         64 },
   {  "MQTYPE_NULL"                     ,          2 },
   {  "MQTYPE_STRING"                   ,       1024 },
   {  "MQUA_FIRST"                      ,      65536 },
   {  "MQUA_LAST"                       ,  999999999 },
   {  "MQUCI_NO"                        ,          0 },
   {  "MQUCI_YES"                       ,          1 },
   {  "MQUIDSUPP_NO"                    ,          0 },
   {  "MQUIDSUPP_YES"                   ,          1 },
   {  "MQUNDELIVERED_DISCARD"           ,          2 },
   {  "MQUNDELIVERED_KEEP"              ,          3 },
   {  "MQUNDELIVERED_NORMAL"            ,          0 },
   {  "MQUNDELIVERED_SAFE"              ,          1 },
   {  "MQUOWST_ACTIVE"                  ,          1 },
   {  "MQUOWST_NONE"                    ,          0 },
   {  "MQUOWST_PREPARED"                ,          2 },
   {  "MQUOWST_UNRESOLVED"              ,          3 },
   {  "MQUOWT_CICS"                     ,          1 },
   {  "MQUOWT_IMS"                      ,          3 },
   {  "MQUOWT_Q_MGR"                    ,          0 },
   {  "MQUOWT_RRS"                      ,          2 },
   {  "MQUOWT_XA"                       ,          4 },
   {  "MQUSAGE_DS_OLDEST_ACTIVE_UOW"    ,         10 },
   {  "MQUSAGE_DS_OLDEST_CF_RECOVERY"   ,         12 },
   {  "MQUSAGE_DS_OLDEST_PS_RECOVERY"   ,         11 },
   {  "MQUSAGE_EXPAND_NONE"             ,          3 },
   {  "MQUSAGE_EXPAND_SYSTEM"           ,          2 },
   {  "MQUSAGE_EXPAND_USER"             ,          1 },
   {  "MQUSAGE_PS_AVAILABLE"            ,          0 },
   {  "MQUSAGE_PS_DEFINED"              ,          1 },
   {  "MQUSAGE_PS_NOT_DEFINED"          ,          3 },
   {  "MQUSAGE_PS_OFFLINE"              ,          2 },
   {  "MQUSAGE_PS_SUSPENDED"            ,          4 },
   {  "MQUSAGE_SMDS_AVAILABLE"          ,          0 },
   {  "MQUSAGE_SMDS_NO_DATA"            ,          1 },
   {  "MQUSEDLQ_AS_PARENT"              ,          0 },
   {  "MQUSEDLQ_NO"                     ,          1 },
   {  "MQUSEDLQ_YES"                    ,          2 },
   {  "MQUSRC_CHANNEL"                  ,          2 },
   {  "MQUSRC_MAP"                      ,          0 },
   {  "MQUSRC_NOACCESS"                 ,          1 },
   {  "MQUS_NORMAL"                     ,          0 },
   {  "MQUS_TRANSMISSION"               ,          1 },
   {  "MQVL_EMPTY_STRING"               ,          0 },
   {  "MQVL_NULL_TERMINATED"            ,         -1 },
   {  "MQVS_NULL_TERMINATED"            ,         -1 },
   {  "MQVU_ANY_USER"                   ,          2 },
   {  "MQVU_FIXED_USER"                 ,          1 },
   {  "MQWARN_NO"                       ,          0 },
   {  "MQWARN_YES"                      ,          1 },
   {  "MQWDR1_CURRENT_LENGTH"           ,        124 },
   {  "MQWDR1_LENGTH_1"                 ,        124 },
   {  "MQWDR2_CURRENT_LENGTH"           ,        136 },
   {  "MQWDR2_LENGTH_1"                 ,        124 },
   {  "MQWDR2_LENGTH_2"                 ,        136 },
   {  "MQWDR_CURRENT_LENGTH"            ,        136 },
   {  "MQWDR_CURRENT_VERSION"           ,          2 },
   {  "MQWDR_LENGTH_1"                  ,        124 },
   {  "MQWDR_LENGTH_2"                  ,        136 },
   {  "MQWDR_VERSION_1"                 ,          1 },
   {  "MQWDR_VERSION_2"                 ,          2 },
   {  "MQWIH_CURRENT_LENGTH"            ,        120 },
   {  "MQWIH_CURRENT_VERSION"           ,          1 },
   {  "MQWIH_LENGTH_1"                  ,        120 },
   {  "MQWIH_NONE"                      ,          0 },
   {  "MQWIH_VERSION_1"                 ,          1 },
   {  "MQWI_UNLIMITED"                  ,         -1 },
   {  "MQWQR1_CURRENT_LENGTH"           ,        200 },
   {  "MQWQR1_LENGTH_1"                 ,        200 },
   {  "MQWQR2_CURRENT_LENGTH"           ,        208 },
   {  "MQWQR2_LENGTH_1"                 ,        200 },
   {  "MQWQR2_LENGTH_2"                 ,        208 },
   {  "MQWQR3_CURRENT_LENGTH"           ,        212 },
   {  "MQWQR3_LENGTH_1"                 ,        200 },
   {  "MQWQR3_LENGTH_2"                 ,        208 },
   {  "MQWQR3_LENGTH_3"                 ,        212 },
   {  "MQWQR_CURRENT_LENGTH"            ,        212 },
   {  "MQWQR_CURRENT_VERSION"           ,          3 },
   {  "MQWQR_LENGTH_1"                  ,        200 },
   {  "MQWQR_LENGTH_2"                  ,        208 },
   {  "MQWQR_LENGTH_3"                  ,        212 },
   {  "MQWQR_VERSION_1"                 ,          1 },
   {  "MQWQR_VERSION_2"                 ,          2 },
   {  "MQWQR_VERSION_3"                 ,          3 },
   {  "MQWS_CHAR"                       ,          1 },
   {  "MQWS_DEFAULT"                    ,          0 },
   {  "MQWS_TOPIC"                      ,          2 },
   {  "MQWXP1_CURRENT_LENGTH (4 byte)"  ,        208 },
   {  "MQWXP1_CURRENT_LENGTH (8 byte)"  ,        224 },
   {  "MQWXP1_LENGTH_1 (4 byte)"        ,        208 },
   {  "MQWXP1_LENGTH_1 (8 byte)"        ,        224 },
   {  "MQWXP2_CURRENT_LENGTH (4 byte)"  ,        216 },
   {  "MQWXP2_CURRENT_LENGTH (8 byte)"  ,        240 },
   {  "MQWXP2_LENGTH_1 (4 byte)"        ,        208 },
   {  "MQWXP2_LENGTH_1 (8 byte)"        ,        224 },
   {  "MQWXP2_LENGTH_2 (4 byte)"        ,        216 },
   {  "MQWXP2_LENGTH_2 (8 byte)"        ,        240 },
   {  "MQWXP3_CURRENT_LENGTH (4 byte)"  ,        220 },
   {  "MQWXP3_CURRENT_LENGTH (8 byte)"  ,        240 },
   {  "MQWXP3_LENGTH_1 (4 byte)"        ,        208 },
   {  "MQWXP3_LENGTH_1 (8 byte)"        ,        224 },
   {  "MQWXP3_LENGTH_2 (4 byte)"        ,        216 },
   {  "MQWXP3_LENGTH_2 (8 byte)"        ,        240 },
   {  "MQWXP3_LENGTH_3 (4 byte)"        ,        220 },
   {  "MQWXP3_LENGTH_3 (8 byte)"        ,        240 },
   {  "MQWXP4_CURRENT_LENGTH (4 byte)"  ,        224 },
   {  "MQWXP4_CURRENT_LENGTH (8 byte)"  ,        248 },
   {  "MQWXP4_LENGTH_1 (4 byte)"        ,        208 },
   {  "MQWXP4_LENGTH_1 (8 byte)"        ,        224 },
   {  "MQWXP4_LENGTH_2 (4 byte)"        ,        216 },
   {  "MQWXP4_LENGTH_2 (8 byte)"        ,        240 },
   {  "MQWXP4_LENGTH_3 (4 byte)"        ,        220 },
   {  "MQWXP4_LENGTH_3 (8 byte)"        ,        240 },
   {  "MQWXP4_LENGTH_4 (4 byte)"        ,        224 },
   {  "MQWXP4_LENGTH_4 (8 byte)"        ,        248 },
   {  "MQWXP_CURRENT_LENGTH (4 byte)"   ,        224 },
   {  "MQWXP_CURRENT_LENGTH (8 byte)"   ,        248 },
   {  "MQWXP_CURRENT_VERSION"           ,          4 },
   {  "MQWXP_LENGTH_1 (4 byte)"         ,        208 },
   {  "MQWXP_LENGTH_1 (8 byte)"         ,        224 },
   {  "MQWXP_LENGTH_2 (4 byte)"         ,        216 },
   {  "MQWXP_LENGTH_2 (8 byte)"         ,        240 },
   {  "MQWXP_LENGTH_3 (4 byte)"         ,        220 },
   {  "MQWXP_LENGTH_3 (8 byte)"         ,        240 },
   {  "MQWXP_LENGTH_4 (4 byte)"         ,        224 },
   {  "MQWXP_LENGTH_4 (8 byte)"         ,        248 },
   {  "MQWXP_PUT_BY_CLUSTER_CHL"        ,          2 },
   {  "MQWXP_VERSION_1"                 ,          1 },
   {  "MQWXP_VERSION_2"                 ,          2 },
   {  "MQWXP_VERSION_3"                 ,          3 },
   {  "MQWXP_VERSION_4"                 ,          4 },
   {  "MQXACT_EXTERNAL"                 ,          1 },
   {  "MQXACT_INTERNAL"                 ,          2 },
   {  "MQXCC_CLOSE_CHANNEL"             ,         -6 },
   {  "MQXCC_FAILED"                    ,         -8 },
   {  "MQXCC_OK"                        ,          0 },
   {  "MQXCC_REQUEST_ACK"               ,         -7 },
   {  "MQXCC_SEND_AND_REQUEST_SEC_MSG"  ,         -3 },
   {  "MQXCC_SEND_SEC_MSG"              ,         -4 },
   {  "MQXCC_SKIP_FUNCTION"             ,         -2 },
   {  "MQXCC_SUPPRESS_EXIT"             ,         -5 },
   {  "MQXCC_SUPPRESS_FUNCTION"         ,         -1 },
   {  "MQXC_CALLBACK"                   ,         48 },
   {  "MQXC_MQBACK"                     ,          9 },
   {  "MQXC_MQCB"                       ,         44 },
   {  "MQXC_MQCLOSE"                    ,          2 },
   {  "MQXC_MQCMIT"                     ,         10 },
   {  "MQXC_MQCTL"                      ,         45 },
   {  "MQXC_MQGET"                      ,          3 },
   {  "MQXC_MQINQ"                      ,          6 },
   {  "MQXC_MQOPEN"                     ,          1 },
   {  "MQXC_MQPUT"                      ,          4 },
   {  "MQXC_MQPUT1"                     ,          5 },
   {  "MQXC_MQSET"                      ,          8 },
   {  "MQXC_MQSTAT"                     ,         46 },
   {  "MQXC_MQSUB"                      ,         42 },
   {  "MQXC_MQSUBRQ"                    ,         43 },
   {  "MQXDR_CONVERSION_FAILED"         ,          1 },
   {  "MQXDR_OK"                        ,          0 },
   {  "MQXEPO_CURRENT_LENGTH (4 byte)"  ,         32 },
   {  "MQXEPO_CURRENT_LENGTH (8 byte)"  ,         40 },
   {  "MQXEPO_CURRENT_VERSION"          ,          1 },
   {  "MQXEPO_LENGTH_1 (4 byte)"        ,         32 },
   {  "MQXEPO_LENGTH_1 (8 byte)"        ,         40 },
   {  "MQXEPO_NONE"                     ,          0 },
   {  "MQXEPO_VERSION_1"                ,          1 },
   {  "MQXE_COMMAND_SERVER"             ,          3 },
   {  "MQXE_MCA"                        ,          1 },
   {  "MQXE_MCA_CLNTCONN"               ,          5 },
   {  "MQXE_MCA_SVRCONN"                ,          2 },
   {  "MQXE_MQSC"                       ,          4 },
   {  "MQXE_OTHER"                      ,          0 },
   {  "MQXF_AXREG"                      ,         34 },
   {  "MQXF_AXUNREG"                    ,         35 },
   {  "MQXF_BACK"                       ,         16 },
   {  "MQXF_BEGIN"                      ,         14 },
   {  "MQXF_CALLBACK"                   ,         21 },
   {  "MQXF_CB"                         ,         19 },
   {  "MQXF_CLOSE"                      ,          7 },
   {  "MQXF_CMIT"                       ,         15 },
   {  "MQXF_CONN"                       ,          3 },
   {  "MQXF_CONNX"                      ,          4 },
   {  "MQXF_CTL"                        ,         20 },
   {  "MQXF_DATA_CONV_ON_GET"           ,         11 },
   {  "MQXF_DISC"                       ,          5 },
   {  "MQXF_GET"                        ,         10 },
   {  "MQXF_INIT"                       ,          1 },
   {  "MQXF_INQ"                        ,         12 },
   {  "MQXF_OPEN"                       ,          6 },
   {  "MQXF_PUT"                        ,          9 },
   {  "MQXF_PUT1"                       ,          8 },
   {  "MQXF_SET"                        ,         13 },
   {  "MQXF_STAT"                       ,         18 },
   {  "MQXF_SUB"                        ,         22 },
   {  "MQXF_SUBRQ"                      ,         23 },
   {  "MQXF_TERM"                       ,          2 },
   {  "MQXF_XACLOSE"                    ,         24 },
   {  "MQXF_XACOMMIT"                   ,         25 },
   {  "MQXF_XACOMPLETE"                 ,         26 },
   {  "MQXF_XAEND"                      ,         27 },
   {  "MQXF_XAFORGET"                   ,         28 },
   {  "MQXF_XAOPEN"                     ,         29 },
   {  "MQXF_XAPREPARE"                  ,         30 },
   {  "MQXF_XARECOVER"                  ,         31 },
   {  "MQXF_XAROLLBACK"                 ,         32 },
   {  "MQXF_XASTART"                    ,         33 },
   {  "MQXPT_ALL"                       ,         -1 },
   {  "MQXPT_DECNET"                    ,          5 },
   {  "MQXPT_LOCAL"                     ,          0 },
   {  "MQXPT_LU62"                      ,          1 },
   {  "MQXPT_NETBIOS"                   ,          3 },
   {  "MQXPT_SPX"                       ,          4 },
   {  "MQXPT_TCP"                       ,          2 },
   {  "MQXPT_UDP"                       ,          6 },
   {  "MQXP_CURRENT_LENGTH"             ,         44 },
   {  "MQXP_LENGTH_1"                   ,         44 },
   {  "MQXP_VERSION_1"                  ,          1 },
   {  "MQXQH_CURRENT_LENGTH"            ,        428 },
   {  "MQXQH_CURRENT_VERSION"           ,          1 },
   {  "MQXQH_LENGTH_1"                  ,        428 },
   {  "MQXQH_VERSION_1"                 ,          1 },
   {  "MQXR2_CONTINUE_CHAIN"            ,          8 },
   {  "MQXR2_DEFAULT_CONTINUATION"      ,          0 },
   {  "MQXR2_DYNAMIC_CACHE"             ,         32 },
   {  "MQXR2_PUT_WITH_DEF_ACTION"       ,          0 },
   {  "MQXR2_PUT_WITH_DEF_USERID"       ,          1 },
   {  "MQXR2_PUT_WITH_MSG_USERID"       ,          2 },
   {  "MQXR2_STATIC_CACHE"              ,          0 },
   {  "MQXR2_SUPPRESS_CHAIN"            ,         16 },
   {  "MQXR2_USE_AGENT_BUFFER"          ,          0 },
   {  "MQXR2_USE_EXIT_BUFFER"           ,          4 },
   {  "MQXR_ACK_RECEIVED"               ,         26 },
   {  "MQXR_AFTER"                      ,          2 },
   {  "MQXR_AUTO_CLUSRCVR"              ,         28 },
   {  "MQXR_AUTO_CLUSSDR"               ,         18 },
   {  "MQXR_AUTO_RECEIVER"              ,         19 },
   {  "MQXR_AUTO_SVRCONN"               ,         27 },
   {  "MQXR_BEFORE"                     ,          1 },
   {  "MQXR_BEFORE_CONVERT"             ,          4 },
   {  "MQXR_CLWL_MOVE"                  ,         22 },
   {  "MQXR_CLWL_OPEN"                  ,         20 },
   {  "MQXR_CLWL_PUT"                   ,         21 },
   {  "MQXR_CLWL_REPOS"                 ,         23 },
   {  "MQXR_CLWL_REPOS_MOVE"            ,         24 },
   {  "MQXR_CONNECTION"                 ,          3 },
   {  "MQXR_END_BATCH"                  ,         25 },
   {  "MQXR_INIT"                       ,         11 },
   {  "MQXR_INIT_SEC"                   ,         16 },
   {  "MQXR_MSG"                        ,         13 },
   {  "MQXR_PRECONNECT"                 ,         31 },
   {  "MQXR_PUBLICATION"                ,         30 },
   {  "MQXR_RETRY"                      ,         17 },
   {  "MQXR_SEC_MSG"                    ,         15 },
   {  "MQXR_SEC_PARMS"                  ,         29 },
   {  "MQXR_TERM"                       ,         12 },
   {  "MQXR_XMIT"                       ,         14 },
   {  "MQXT_API_CROSSING_EXIT"          ,          1 },
   {  "MQXT_API_EXIT"                   ,          2 },
   {  "MQXT_CHANNEL_AUTO_DEF_EXIT"      ,         16 },
   {  "MQXT_CHANNEL_MSG_EXIT"           ,         12 },
   {  "MQXT_CHANNEL_MSG_RETRY_EXIT"     ,         15 },
   {  "MQXT_CHANNEL_RCV_EXIT"           ,         14 },
   {  "MQXT_CHANNEL_SEC_EXIT"           ,         11 },
   {  "MQXT_CHANNEL_SEND_EXIT"          ,         13 },
   {  "MQXT_CLUSTER_WORKLOAD_EXIT"      ,         20 },
   {  "MQXT_PRECONNECT_EXIT"            ,         23 },
   {  "MQXT_PUBLISH_EXIT"               ,         22 },
   {  "MQXT_PUBSUB_ROUTING_EXIT"        ,         21 },
   {  "MQXWD_CURRENT_LENGTH"            ,         24 },
   {  "MQXWD_LENGTH_1"                  ,         24 },
   {  "MQXWD_VERSION_1"                 ,          1 },
   {  "MQZAC_CURRENT_LENGTH"            ,         84 },
   {  "MQZAC_CURRENT_VERSION"           ,          1 },
   {  "MQZAC_LENGTH_1"                  ,         84 },
   {  "MQZAC_VERSION_1"                 ,          1 },
   {  "MQZAD_CURRENT_LENGTH (4 byte)"   ,         76 },
   {  "MQZAD_CURRENT_LENGTH (8 byte)"   ,         80 },
   {  "MQZAD_CURRENT_VERSION"           ,          2 },
   {  "MQZAD_LENGTH_1 (4 byte)"         ,         72 },
   {  "MQZAD_LENGTH_1 (8 byte)"         ,         80 },
   {  "MQZAD_LENGTH_2 (4 byte)"         ,         76 },
   {  "MQZAD_LENGTH_2 (8 byte)"         ,         80 },
   {  "MQZAD_VERSION_1"                 ,          1 },
   {  "MQZAD_VERSION_2"                 ,          2 },
   {  "MQZAET_GROUP"                    ,          2 },
   {  "MQZAET_NONE"                     ,          0 },
   {  "MQZAET_PRINCIPAL"                ,          1 },
   {  "MQZAET_UNKNOWN"                  ,          3 },
   {  "MQZAO_ALL"                       ,   50216959 },
   {  "MQZAO_ALL_ADMIN"                 ,   16646144 },
   {  "MQZAO_ALL_MQI"                   ,      16383 },
   {  "MQZAO_ALTERNATE_USER_AUTHORITY"  ,       1024 },
   {  "MQZAO_AUTHORIZE"                 ,    8388608 },
   {  "MQZAO_BROWSE"                    ,          2 },
   {  "MQZAO_CHANGE"                    ,     524288 },
   {  "MQZAO_CLEAR"                     ,    1048576 },
   {  "MQZAO_CONNECT"                   ,          1 },
   {  "MQZAO_CONTROL"                   ,    2097152 },
   {  "MQZAO_CONTROL_EXTENDED"          ,    4194304 },
   {  "MQZAO_CREATE"                    ,      65536 },
   {  "MQZAO_CREATE_ONLY"               ,   67108864 },
   {  "MQZAO_DELETE"                    ,     131072 },
   {  "MQZAO_DISPLAY"                   ,     262144 },
   {  "MQZAO_INPUT"                     ,          4 },
   {  "MQZAO_INQUIRE"                   ,         16 },
   {  "MQZAO_NONE"                      ,          0 },
   {  "MQZAO_OUTPUT"                    ,          8 },
   {  "MQZAO_PASS_ALL_CONTEXT"          ,        128 },
   {  "MQZAO_PASS_IDENTITY_CONTEXT"     ,         64 },
   {  "MQZAO_PUBLISH"                   ,       2048 },
   {  "MQZAO_REMOVE"                    ,   16777216 },
   {  "MQZAO_RESUME"                    ,       8192 },
   {  "MQZAO_SET"                       ,         32 },
   {  "MQZAO_SET_ALL_CONTEXT"           ,        512 },
   {  "MQZAO_SET_IDENTITY_CONTEXT"      ,        256 },
   {  "MQZAO_SUBSCRIBE"                 ,       4096 },
   {  "MQZAO_SYSTEM"                    ,   33554432 },
   {  "MQZAS_VERSION_1"                 ,          1 },
   {  "MQZAS_VERSION_2"                 ,          2 },
   {  "MQZAS_VERSION_3"                 ,          3 },
   {  "MQZAS_VERSION_4"                 ,          4 },
   {  "MQZAS_VERSION_5"                 ,          5 },
   {  "MQZAS_VERSION_6"                 ,          6 },
   {  "MQZAT_CHANGE_CONTEXT"            ,          1 },
   {  "MQZAT_INITIAL_CONTEXT"           ,          0 },
   {  "MQZCI_CONTINUE"                  ,          0 },
   {  "MQZCI_DEFAULT"                   ,          0 },
   {  "MQZCI_STOP"                      ,          1 },
   {  "MQZED_CURRENT_LENGTH (4 byte)"   ,         60 },
   {  "MQZED_CURRENT_LENGTH (8 byte)"   ,         72 },
   {  "MQZED_CURRENT_VERSION"           ,          2 },
   {  "MQZED_LENGTH_1 (4 byte)"         ,         56 },
   {  "MQZED_LENGTH_1 (8 byte)"         ,         64 },
   {  "MQZED_LENGTH_2 (4 byte)"         ,         60 },
   {  "MQZED_LENGTH_2 (8 byte)"         ,         72 },
   {  "MQZED_VERSION_1"                 ,          1 },
   {  "MQZED_VERSION_2"                 ,          2 },
   {  "MQZFP_CURRENT_LENGTH (4 byte)"   ,         20 },
   {  "MQZFP_CURRENT_LENGTH (8 byte)"   ,         24 },
   {  "MQZFP_CURRENT_VERSION"           ,          1 },
   {  "MQZFP_LENGTH_1 (4 byte)"         ,         20 },
   {  "MQZFP_LENGTH_1 (8 byte)"         ,         24 },
   {  "MQZFP_VERSION_1"                 ,          1 },
   {  "MQZIC_CURRENT_LENGTH"            ,         84 },
   {  "MQZIC_CURRENT_VERSION"           ,          1 },
   {  "MQZIC_LENGTH_1"                  ,         84 },
   {  "MQZIC_VERSION_1"                 ,          1 },
   {  "MQZID_AUTHENTICATE_USER"         ,         10 },
   {  "MQZID_CHECK_AUTHORITY"           ,          2 },
   {  "MQZID_CHECK_PRIVILEGED"          ,         13 },
   {  "MQZID_COPY_ALL_AUTHORITY"        ,          3 },
   {  "MQZID_DELETE_AUTHORITY"          ,          4 },
   {  "MQZID_DELETE_NAME"               ,          4 },
   {  "MQZID_ENUMERATE_AUTHORITY_DATA"  ,          9 },
   {  "MQZID_FIND_USERID"               ,          2 },
   {  "MQZID_FREE_USER"                 ,         11 },
   {  "MQZID_GET_AUTHORITY"             ,          6 },
   {  "MQZID_GET_EXPLICIT_AUTHORITY"    ,          7 },
   {  "MQZID_INIT"                      ,          0 },
   {  "MQZID_INIT_AUTHORITY"            ,          0 },
   {  "MQZID_INIT_NAME"                 ,          0 },
   {  "MQZID_INIT_USERID"               ,          0 },
   {  "MQZID_INQUIRE"                   ,         12 },
   {  "MQZID_INSERT_NAME"               ,          3 },
   {  "MQZID_LOOKUP_NAME"               ,          2 },
   {  "MQZID_REFRESH_CACHE"             ,          8 },
   {  "MQZID_SET_AUTHORITY"             ,          5 },
   {  "MQZID_TERM"                      ,          1 },
   {  "MQZID_TERM_AUTHORITY"            ,          1 },
   {  "MQZID_TERM_NAME"                 ,          1 },
   {  "MQZID_TERM_USERID"               ,          1 },
   {  "MQZIO_PRIMARY"                   ,          0 },
   {  "MQZIO_SECONDARY"                 ,          1 },
   {  "MQZNS_VERSION_1"                 ,          1 },
   {  "MQZSE_CONTINUE"                  ,          0 },
   {  "MQZSE_START"                     ,          1 },
   {  "MQZSL_NOT_RETURNED"              ,          0 },
   {  "MQZSL_RETURNED"                  ,          1 },
   {  "MQZTO_PRIMARY"                   ,          0 },
   {  "MQZTO_SECONDARY"                 ,          1 },
   {  "MQZUS_VERSION_1"                 ,          1 },
   {  "MQ_ABEND_CODE_LENGTH"            ,          4 },
   {  "MQ_ACCOUNTING_TOKEN_LENGTH"      ,         32 },
   {  "MQ_AMQP_CLIENT_ID_LENGTH"        ,        256 },
   {  "MQ_APPL_DESC_LENGTH"             ,         64 },
   {  "MQ_APPL_FUNCTION_NAME_LENGTH"    ,         10 },
   {  "MQ_APPL_IDENTITY_DATA_LENGTH"    ,         32 },
   {  "MQ_APPL_NAME_LENGTH"             ,         28 },
   {  "MQ_APPL_ORIGIN_DATA_LENGTH"      ,          4 },
   {  "MQ_APPL_TAG_LENGTH"              ,         28 },
   {  "MQ_ARCHIVE_PFX_LENGTH"           ,         36 },
   {  "MQ_ARCHIVE_UNIT_LENGTH"          ,          8 },
   {  "MQ_ARM_SUFFIX_LENGTH"            ,          2 },
   {  "MQ_ASID_LENGTH"                  ,          4 },
   {  "MQ_ATTENTION_ID_LENGTH"          ,          4 },
   {  "MQ_AUTHENTICATOR_LENGTH"         ,          8 },
   {  "MQ_AUTH_INFO_CONN_NAME_LENGTH"   ,        264 },
   {  "MQ_AUTH_INFO_DESC_LENGTH"        ,         64 },
   {  "MQ_AUTH_INFO_NAME_LENGTH"        ,         48 },
   {  "MQ_AUTH_INFO_OCSP_URL_LENGTH"    ,        256 },
   {  "MQ_AUTH_PROFILE_NAME_LENGTH"     ,         48 },
   {  "MQ_AUTO_REORG_CATALOG_LENGTH"    ,         44 },
   {  "MQ_AUTO_REORG_TIME_LENGTH"       ,          4 },
   {  "MQ_BATCH_INTERFACE_ID_LENGTH"    ,          8 },
   {  "MQ_BRIDGE_NAME_LENGTH"           ,         24 },
   {  "MQ_CANCEL_CODE_LENGTH"           ,          4 },
   {  "MQ_CERT_LABEL_LENGTH"            ,         64 },
   {  "MQ_CERT_VAL_POLICY_ANY"          ,          0 },
   {  "MQ_CERT_VAL_POLICY_DEFAULT"      ,          0 },
   {  "MQ_CERT_VAL_POLICY_RFC5280"      ,          1 },
   {  "MQ_CF_LEID_LENGTH"               ,         12 },
   {  "MQ_CF_STRUC_DESC_LENGTH"         ,         64 },
   {  "MQ_CF_STRUC_NAME_LENGTH"         ,         12 },
   {  "MQ_CHANNEL_DATE_LENGTH"          ,         12 },
   {  "MQ_CHANNEL_DESC_LENGTH"          ,         64 },
   {  "MQ_CHANNEL_NAME_LENGTH"          ,         20 },
   {  "MQ_CHANNEL_TIME_LENGTH"          ,          8 },
   {  "MQ_CHINIT_SERVICE_PARM_LENGTH"   ,         32 },
   {  "MQ_CHLAUTH_DESC_LENGTH"          ,         64 },
   {  "MQ_CICS_FILE_NAME_LENGTH"        ,          8 },
   {  "MQ_CLIENT_ID_LENGTH"             ,         23 },
   {  "MQ_CLIENT_USER_ID_LENGTH"        ,       1024 },
   {  "MQ_CLUSTER_NAME_LENGTH"          ,         48 },
   {  "MQ_COMMAND_MQSC_LENGTH"          ,      32768 },
   {  "MQ_COMM_INFO_DESC_LENGTH"        ,         64 },
   {  "MQ_COMM_INFO_NAME_LENGTH"        ,         48 },
   {  "MQ_CONNECTION_ID_LENGTH"         ,         24 },
   {  "MQ_CONN_NAME_LENGTH"             ,        264 },
   {  "MQ_CONN_TAG_LENGTH"              ,        128 },
   {  "MQ_CORREL_ID_LENGTH"             ,         24 },
   {  "MQ_CREATION_DATE_LENGTH"         ,         12 },
   {  "MQ_CREATION_TIME_LENGTH"         ,          8 },
   {  "MQ_CSP_PASSWORD_LENGTH"          ,        256 },
   {  "MQ_CUSTOM_LENGTH"                ,        128 },
   {  "MQ_DATA_SET_NAME_LENGTH"         ,         44 },
   {  "MQ_DATE_LENGTH"                  ,         12 },
   {  "MQ_DB2_NAME_LENGTH"              ,          4 },
   {  "MQ_DISTINGUISHED_NAME_LENGTH"    ,       1024 },
   {  "MQ_DNS_GROUP_NAME_LENGTH"        ,         18 },
   {  "MQ_DSG_NAME_LENGTH"              ,          8 },
   {  "MQ_ENTITY_NAME_LENGTH"           ,       1024 },
   {  "MQ_ENV_INFO_LENGTH"              ,         96 },
   {  "MQ_EXIT_DATA_LENGTH"             ,         32 },
   {  "MQ_EXIT_INFO_NAME_LENGTH"        ,         48 },
   {  "MQ_EXIT_NAME_LENGTH"             ,        128 },
   {  "MQ_EXIT_PD_AREA_LENGTH"          ,         48 },
   {  "MQ_EXIT_USER_AREA_LENGTH"        ,         16 },
   {  "MQ_FACILITY_LENGTH"              ,          8 },
   {  "MQ_FACILITY_LIKE_LENGTH"         ,          4 },
   {  "MQ_FORMAT_LENGTH"                ,          8 },
   {  "MQ_FUNCTION_LENGTH"              ,          4 },
   {  "MQ_GROUP_ADDRESS_LENGTH"         ,        264 },
   {  "MQ_GROUP_ID_LENGTH"              ,         24 },
   {  "MQ_INSTALLATION_DESC_LENGTH"     ,         64 },
   {  "MQ_INSTALLATION_NAME_LENGTH"     ,         16 },
   {  "MQ_INSTALLATION_PATH_LENGTH"     ,        256 },
   {  "MQ_IP_ADDRESS_LENGTH"            ,         48 },
   {  "MQ_JAAS_CONFIG_LENGTH"           ,       1024 },
   {  "MQ_LDAP_BASE_DN_LENGTH"          ,       1024 },
   {  "MQ_LDAP_CLASS_LENGTH"            ,        128 },
   {  "MQ_LDAP_FIELD_LENGTH"            ,        128 },
   {  "MQ_LDAP_MCA_USER_ID_LENGTH"      ,       1024 },
   {  "MQ_LDAP_PASSWORD_LENGTH"         ,         32 },
   {  "MQ_LISTENER_DESC_LENGTH"         ,         64 },
   {  "MQ_LISTENER_NAME_LENGTH"         ,         48 },
   {  "MQ_LOCAL_ADDRESS_LENGTH"         ,         48 },
   {  "MQ_LOG_CORREL_ID_LENGTH"         ,          8 },
   {  "MQ_LOG_EXTENT_NAME_LENGTH"       ,         24 },
   {  "MQ_LOG_PATH_LENGTH"              ,       1024 },
   {  "MQ_LRSN_LENGTH"                  ,         12 },
   {  "MQ_LTERM_OVERRIDE_LENGTH"        ,          8 },
   {  "MQ_LUWID_LENGTH"                 ,         16 },
   {  "MQ_LU_NAME_LENGTH"               ,          8 },
   {  "MQ_MAX_EXIT_NAME_LENGTH"         ,        128 },
   {  "MQ_MAX_LDAP_MCA_USER_ID_LENGTH"  ,       1024 },
   {  "MQ_MAX_MCA_USER_ID_LENGTH"       ,         64 },
   {  "MQ_MAX_PROPERTY_NAME_LENGTH"     ,       4095 },
   {  "MQ_MAX_USER_ID_LENGTH"           ,         64 },
   {  "MQ_MCA_JOB_NAME_LENGTH"          ,         28 },
   {  "MQ_MCA_NAME_LENGTH"              ,         20 },
   {  "MQ_MCA_USER_DATA_LENGTH"         ,         32 },
   {  "MQ_MCA_USER_ID_LENGTH"           ,         12 },
   {  "MQ_MFS_MAP_NAME_LENGTH"          ,          8 },
   {  "MQ_MODE_NAME_LENGTH"             ,          8 },
   {  "MQ_MQTT_MAX_KEEP_ALIVE"          ,      65536 },
   {  "MQ_MSG_HEADER_LENGTH"            ,       4000 },
   {  "MQ_MSG_ID_LENGTH"                ,         24 },
   {  "MQ_MSG_TOKEN_LENGTH"             ,         16 },
   {  "MQ_NAMELIST_DESC_LENGTH"         ,         64 },
   {  "MQ_NAMELIST_NAME_LENGTH"         ,         48 },
   {  "MQ_OBJECT_INSTANCE_ID_LENGTH"    ,         24 },
   {  "MQ_OBJECT_NAME_LENGTH"           ,         48 },
   {  "MQ_OPERATOR_MESSAGE_LENGTH"      ,          4 },
   {  "MQ_ORIGIN_NAME_LENGTH"           ,          8 },
   {  "MQ_PASSWORD_LENGTH"              ,         12 },
   {  "MQ_PASS_TICKET_APPL_LENGTH"      ,          8 },
   {  "MQ_PROCESS_APPL_ID_LENGTH"       ,        256 },
   {  "MQ_PROCESS_DESC_LENGTH"          ,         64 },
   {  "MQ_PROCESS_ENV_DATA_LENGTH"      ,        128 },
   {  "MQ_PROCESS_NAME_LENGTH"          ,         48 },
   {  "MQ_PROCESS_USER_DATA_LENGTH"     ,        128 },
   {  "MQ_PROGRAM_NAME_LENGTH"          ,         20 },
   {  "MQ_PSB_NAME_LENGTH"              ,          8 },
   {  "MQ_PST_ID_LENGTH"                ,          8 },
   {  "MQ_PUT_APPL_NAME_LENGTH"         ,         28 },
   {  "MQ_PUT_DATE_LENGTH"              ,          8 },
   {  "MQ_PUT_TIME_LENGTH"              ,          8 },
   {  "MQ_QSG_NAME_LENGTH"              ,          4 },
   {  "MQ_Q_DESC_LENGTH"                ,         64 },
   {  "MQ_Q_MGR_CPF_LENGTH"             ,          4 },
   {  "MQ_Q_MGR_DESC_LENGTH"            ,         64 },
   {  "MQ_Q_MGR_IDENTIFIER_LENGTH"      ,         48 },
   {  "MQ_Q_MGR_NAME_LENGTH"            ,         48 },
   {  "MQ_Q_NAME_LENGTH"                ,         48 },
   {  "MQ_RBA_LENGTH"                   ,         16 },
   {  "MQ_REMOTE_PRODUCT_LENGTH"        ,          4 },
   {  "MQ_REMOTE_SYS_ID_LENGTH"         ,          4 },
   {  "MQ_REMOTE_VERSION_LENGTH"        ,          8 },
   {  "MQ_RESPONSE_ID_LENGTH"           ,         24 },
   {  "MQ_SECURITY_ID_LENGTH"           ,         40 },
   {  "MQ_SECURITY_PROFILE_LENGTH"      ,         40 },
   {  "MQ_SELECTOR_LENGTH"              ,      10240 },
   {  "MQ_SERVICE_ARGS_LENGTH"          ,        255 },
   {  "MQ_SERVICE_COMMAND_LENGTH"       ,        255 },
   {  "MQ_SERVICE_COMPONENT_LENGTH"     ,         48 },
   {  "MQ_SERVICE_DESC_LENGTH"          ,         64 },
   {  "MQ_SERVICE_NAME_LENGTH"          ,         32 },
   {  "MQ_SERVICE_PATH_LENGTH"          ,        255 },
   {  "MQ_SERVICE_STEP_LENGTH"          ,          8 },
   {  "MQ_SHORT_CONN_NAME_LENGTH"       ,         20 },
   {  "MQ_SHORT_DNAME_LENGTH"           ,        256 },
   {  "MQ_SMDS_NAME_LENGTH"             ,          4 },
   {  "MQ_SSL_CIPHER_SPEC_LENGTH"       ,         32 },
   {  "MQ_SSL_CIPHER_SUITE_LENGTH"      ,         32 },
   {  "MQ_SSL_CRYPTO_HARDWARE_LENGTH"   ,        256 },
   {  "MQ_SSL_HANDSHAKE_STAGE_LENGTH"   ,         32 },
   {  "MQ_SSL_KEY_LIBRARY_LENGTH"       ,         44 },
   {  "MQ_SSL_KEY_MEMBER_LENGTH"        ,          8 },
   {  "MQ_SSL_KEY_PASSPHRASE_LENGTH"    ,       1024 },
   {  "MQ_SSL_KEY_REPOSITORY_LENGTH"    ,        256 },
   {  "MQ_SSL_PEER_NAME_LENGTH"         ,       1024 },
   {  "MQ_SSL_SHORT_PEER_NAME_LENGTH"   ,        256 },
   {  "MQ_START_CODE_LENGTH"            ,          4 },
   {  "MQ_STORAGE_CLASS_DESC_LENGTH"    ,         64 },
   {  "MQ_STORAGE_CLASS_LENGTH"         ,          8 },
   {  "MQ_SUB_IDENTITY_LENGTH"          ,        128 },
   {  "MQ_SUB_NAME_LENGTH"              ,      10240 },
   {  "MQ_SUB_POINT_LENGTH"             ,        128 },
   {  "MQ_SUITE_B_128_BIT"              ,          2 },
   {  "MQ_SUITE_B_192_BIT"              ,          4 },
   {  "MQ_SUITE_B_NONE"                 ,          1 },
   {  "MQ_SUITE_B_NOT_AVAILABLE"        ,          0 },
   {  "MQ_SUITE_B_SIZE"                 ,          4 },
   {  "MQ_SYSP_SERVICE_LENGTH"          ,         32 },
   {  "MQ_SYSTEM_NAME_LENGTH"           ,          8 },
   {  "MQ_TASK_NUMBER_LENGTH"           ,          8 },
   {  "MQ_TCP_NAME_LENGTH"              ,          8 },
   {  "MQ_TIME_LENGTH"                  ,          8 },
   {  "MQ_TOPIC_DESC_LENGTH"            ,         64 },
   {  "MQ_TOPIC_NAME_LENGTH"            ,         48 },
   {  "MQ_TOPIC_STR_LENGTH"             ,      10240 },
   {  "MQ_TOTAL_EXIT_DATA_LENGTH"       ,        999 },
   {  "MQ_TOTAL_EXIT_NAME_LENGTH"       ,        999 },
   {  "MQ_TPIPE_NAME_LENGTH"            ,          8 },
   {  "MQ_TPIPE_PFX_LENGTH"             ,          4 },
   {  "MQ_TP_NAME_LENGTH"               ,         64 },
   {  "MQ_TRANSACTION_ID_LENGTH"        ,          4 },
   {  "MQ_TRAN_INSTANCE_ID_LENGTH"      ,         16 },
   {  "MQ_TRIGGER_DATA_LENGTH"          ,         64 },
   {  "MQ_TRIGGER_PROGRAM_NAME_LENGTH"  ,          8 },
   {  "MQ_TRIGGER_TERM_ID_LENGTH"       ,          4 },
   {  "MQ_TRIGGER_TRANS_ID_LENGTH"      ,          4 },
   {  "MQ_UOW_ID_LENGTH"                ,        256 },
   {  "MQ_USER_DATA_LENGTH"             ,      10240 },
   {  "MQ_USER_ID_LENGTH"               ,         12 },
   {  "MQ_VERSION_LENGTH"               ,          8 },
   {  "MQ_VOLSER_LENGTH"                ,          6 },
   {  "MQ_XCF_GROUP_NAME_LENGTH"        ,          8 },
   {  "MQ_XCF_MEMBER_NAME_LENGTH"       ,         16 },
   {  "", 0 }
 };


 /****************************************************************/
 /*  End of CMQSTRC                                              */
 /****************************************************************/
 #endif /* End of header file */
