 #if !defined(MQCFC_INCLUDED)          /* File not yet included? */
   #define MQCFC_INCLUDED              /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQCFC                                      */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for PCF and Events             */
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
 /*  FUNCTION:       This file declares the functions,           */
 /*                  structures and named constants for PCF      */
 /*                  and event messages.                         */
 /*                                                              */
 /*  PROCESSOR:      C                                           */
 /*                                                              */
 /****************************************************************/

 /****************************************************************/
 /* <BEGIN_BUILDINFO>                                            */
 /* Generated on:  05/02/19 11:08                                */
 /* Build Level:   p911-L190205                                  */
 /* Build Type:    Production                                    */
 /* Pointer Size:  32 Bit, 64 Bit                                */
 /* Source File:                                                 */
 /* @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ       */
 /* pn=com.ibm.mq.famfiles.data/xml/approved/cmqcfc.xml          */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Values Related to MQCFH Structure                            */
 /****************************************************************/

 /* Structure Length */
 #define MQCFH_STRUC_LENGTH             36

 /* Structure Version Number */
 #define MQCFH_VERSION_1                1
 #define MQCFH_VERSION_2                2
 #define MQCFH_VERSION_3                3
 #define MQCFH_CURRENT_VERSION          3

 /* Command Codes */
 #define MQCMD_NONE                     0
 #define MQCMD_CHANGE_Q_MGR             1
 #define MQCMD_INQUIRE_Q_MGR            2
 #define MQCMD_CHANGE_PROCESS           3
 #define MQCMD_COPY_PROCESS             4
 #define MQCMD_CREATE_PROCESS           5
 #define MQCMD_DELETE_PROCESS           6
 #define MQCMD_INQUIRE_PROCESS          7
 #define MQCMD_CHANGE_Q                 8
 #define MQCMD_CLEAR_Q                  9
 #define MQCMD_COPY_Q                   10
 #define MQCMD_CREATE_Q                 11
 #define MQCMD_DELETE_Q                 12
 #define MQCMD_INQUIRE_Q                13
 #define MQCMD_REFRESH_Q_MGR            16
 #define MQCMD_RESET_Q_STATS            17
 #define MQCMD_INQUIRE_Q_NAMES          18
 #define MQCMD_INQUIRE_PROCESS_NAMES    19
 #define MQCMD_INQUIRE_CHANNEL_NAMES    20
 #define MQCMD_CHANGE_CHANNEL           21
 #define MQCMD_COPY_CHANNEL             22
 #define MQCMD_CREATE_CHANNEL           23
 #define MQCMD_DELETE_CHANNEL           24
 #define MQCMD_INQUIRE_CHANNEL          25
 #define MQCMD_PING_CHANNEL             26
 #define MQCMD_RESET_CHANNEL            27
 #define MQCMD_START_CHANNEL            28
 #define MQCMD_STOP_CHANNEL             29
 #define MQCMD_START_CHANNEL_INIT       30
 #define MQCMD_START_CHANNEL_LISTENER   31
 #define MQCMD_CHANGE_NAMELIST          32
 #define MQCMD_COPY_NAMELIST            33
 #define MQCMD_CREATE_NAMELIST          34
 #define MQCMD_DELETE_NAMELIST          35
 #define MQCMD_INQUIRE_NAMELIST         36
 #define MQCMD_INQUIRE_NAMELIST_NAMES   37
 #define MQCMD_ESCAPE                   38
 #define MQCMD_RESOLVE_CHANNEL          39
 #define MQCMD_PING_Q_MGR               40
 #define MQCMD_INQUIRE_Q_STATUS         41
 #define MQCMD_INQUIRE_CHANNEL_STATUS   42
 #define MQCMD_CONFIG_EVENT             43
 #define MQCMD_Q_MGR_EVENT              44
 #define MQCMD_PERFM_EVENT              45
 #define MQCMD_CHANNEL_EVENT            46
 #define MQCMD_DELETE_PUBLICATION       60
 #define MQCMD_DEREGISTER_PUBLISHER     61
 #define MQCMD_DEREGISTER_SUBSCRIBER    62
 #define MQCMD_PUBLISH                  63
 #define MQCMD_REGISTER_PUBLISHER       64
 #define MQCMD_REGISTER_SUBSCRIBER      65
 #define MQCMD_REQUEST_UPDATE           66
 #define MQCMD_BROKER_INTERNAL          67
 #define MQCMD_ACTIVITY_MSG             69
 #define MQCMD_INQUIRE_CLUSTER_Q_MGR    70
 #define MQCMD_RESUME_Q_MGR_CLUSTER     71
 #define MQCMD_SUSPEND_Q_MGR_CLUSTER    72
 #define MQCMD_REFRESH_CLUSTER          73
 #define MQCMD_RESET_CLUSTER            74
 #define MQCMD_TRACE_ROUTE              75
 #define MQCMD_REFRESH_SECURITY         78
 #define MQCMD_CHANGE_AUTH_INFO         79
 #define MQCMD_COPY_AUTH_INFO           80
 #define MQCMD_CREATE_AUTH_INFO         81
 #define MQCMD_DELETE_AUTH_INFO         82
 #define MQCMD_INQUIRE_AUTH_INFO        83
 #define MQCMD_INQUIRE_AUTH_INFO_NAMES  84
 #define MQCMD_INQUIRE_CONNECTION       85
 #define MQCMD_STOP_CONNECTION          86
 #define MQCMD_INQUIRE_AUTH_RECS        87
 #define MQCMD_INQUIRE_ENTITY_AUTH      88
 #define MQCMD_DELETE_AUTH_REC          89
 #define MQCMD_SET_AUTH_REC             90
 #define MQCMD_LOGGER_EVENT             91
 #define MQCMD_RESET_Q_MGR              92
 #define MQCMD_CHANGE_LISTENER          93
 #define MQCMD_COPY_LISTENER            94
 #define MQCMD_CREATE_LISTENER          95
 #define MQCMD_DELETE_LISTENER          96
 #define MQCMD_INQUIRE_LISTENER         97
 #define MQCMD_INQUIRE_LISTENER_STATUS  98
 #define MQCMD_COMMAND_EVENT            99
 #define MQCMD_CHANGE_SECURITY          100
 #define MQCMD_CHANGE_CF_STRUC          101
 #define MQCMD_CHANGE_STG_CLASS         102
 #define MQCMD_CHANGE_TRACE             103
 #define MQCMD_ARCHIVE_LOG              104
 #define MQCMD_BACKUP_CF_STRUC          105
 #define MQCMD_CREATE_BUFFER_POOL       106
 #define MQCMD_CREATE_PAGE_SET          107
 #define MQCMD_CREATE_CF_STRUC          108
 #define MQCMD_CREATE_STG_CLASS         109
 #define MQCMD_COPY_CF_STRUC            110
 #define MQCMD_COPY_STG_CLASS           111
 #define MQCMD_DELETE_CF_STRUC          112
 #define MQCMD_DELETE_STG_CLASS         113
 #define MQCMD_INQUIRE_ARCHIVE          114
 #define MQCMD_INQUIRE_CF_STRUC         115
 #define MQCMD_INQUIRE_CF_STRUC_STATUS  116
 #define MQCMD_INQUIRE_CMD_SERVER       117
 #define MQCMD_INQUIRE_CHANNEL_INIT     118
 #define MQCMD_INQUIRE_QSG              119
 #define MQCMD_INQUIRE_LOG              120
 #define MQCMD_INQUIRE_SECURITY         121
 #define MQCMD_INQUIRE_STG_CLASS        122
 #define MQCMD_INQUIRE_SYSTEM           123
 #define MQCMD_INQUIRE_THREAD           124
 #define MQCMD_INQUIRE_TRACE            125
 #define MQCMD_INQUIRE_USAGE            126
 #define MQCMD_MOVE_Q                   127
 #define MQCMD_RECOVER_BSDS             128
 #define MQCMD_RECOVER_CF_STRUC         129
 #define MQCMD_RESET_TPIPE              130
 #define MQCMD_RESOLVE_INDOUBT          131
 #define MQCMD_RESUME_Q_MGR             132
 #define MQCMD_REVERIFY_SECURITY        133
 #define MQCMD_SET_ARCHIVE              134
 #define MQCMD_SET_LOG                  136
 #define MQCMD_SET_SYSTEM               137
 #define MQCMD_START_CMD_SERVER         138
 #define MQCMD_START_Q_MGR              139
 #define MQCMD_START_TRACE              140
 #define MQCMD_STOP_CHANNEL_INIT        141
 #define MQCMD_STOP_CHANNEL_LISTENER    142
 #define MQCMD_STOP_CMD_SERVER          143
 #define MQCMD_STOP_Q_MGR               144
 #define MQCMD_STOP_TRACE               145
 #define MQCMD_SUSPEND_Q_MGR            146
 #define MQCMD_INQUIRE_CF_STRUC_NAMES   147
 #define MQCMD_INQUIRE_STG_CLASS_NAMES  148
 #define MQCMD_CHANGE_SERVICE           149
 #define MQCMD_COPY_SERVICE             150
 #define MQCMD_CREATE_SERVICE           151
 #define MQCMD_DELETE_SERVICE           152
 #define MQCMD_INQUIRE_SERVICE          153
 #define MQCMD_INQUIRE_SERVICE_STATUS   154
 #define MQCMD_START_SERVICE            155
 #define MQCMD_STOP_SERVICE             156
 #define MQCMD_DELETE_BUFFER_POOL       157
 #define MQCMD_DELETE_PAGE_SET          158
 #define MQCMD_CHANGE_BUFFER_POOL       159
 #define MQCMD_CHANGE_PAGE_SET          160
 #define MQCMD_INQUIRE_Q_MGR_STATUS     161
 #define MQCMD_CREATE_LOG               162
 #define MQCMD_STATISTICS_MQI           164
 #define MQCMD_STATISTICS_Q             165
 #define MQCMD_STATISTICS_CHANNEL       166
 #define MQCMD_ACCOUNTING_MQI           167
 #define MQCMD_ACCOUNTING_Q             168
 #define MQCMD_INQUIRE_AUTH_SERVICE     169
 #define MQCMD_CHANGE_TOPIC             170
 #define MQCMD_COPY_TOPIC               171
 #define MQCMD_CREATE_TOPIC             172
 #define MQCMD_DELETE_TOPIC             173
 #define MQCMD_INQUIRE_TOPIC            174
 #define MQCMD_INQUIRE_TOPIC_NAMES      175
 #define MQCMD_INQUIRE_SUBSCRIPTION     176
 #define MQCMD_CREATE_SUBSCRIPTION      177
 #define MQCMD_CHANGE_SUBSCRIPTION      178
 #define MQCMD_DELETE_SUBSCRIPTION      179
 #define MQCMD_COPY_SUBSCRIPTION        181
 #define MQCMD_INQUIRE_SUB_STATUS       182
 #define MQCMD_INQUIRE_TOPIC_STATUS     183
 #define MQCMD_CLEAR_TOPIC_STRING       184
 #define MQCMD_INQUIRE_PUBSUB_STATUS    185
 #define MQCMD_INQUIRE_SMDS             186
 #define MQCMD_CHANGE_SMDS              187
 #define MQCMD_RESET_SMDS               188
 #define MQCMD_CREATE_COMM_INFO         190
 #define MQCMD_INQUIRE_COMM_INFO        191
 #define MQCMD_CHANGE_COMM_INFO         192
 #define MQCMD_COPY_COMM_INFO           193
 #define MQCMD_DELETE_COMM_INFO         194
 #define MQCMD_PURGE_CHANNEL            195
 #define MQCMD_MQXR_DIAGNOSTICS         196
 #define MQCMD_START_SMDSCONN           197
 #define MQCMD_STOP_SMDSCONN            198
 #define MQCMD_INQUIRE_SMDSCONN         199
 #define MQCMD_INQUIRE_MQXR_STATUS      200
 #define MQCMD_START_CLIENT_TRACE       201
 #define MQCMD_STOP_CLIENT_TRACE        202
 #define MQCMD_SET_CHLAUTH_REC          203
 #define MQCMD_INQUIRE_CHLAUTH_RECS     204
 #define MQCMD_INQUIRE_PROT_POLICY      205
 #define MQCMD_CREATE_PROT_POLICY       206
 #define MQCMD_DELETE_PROT_POLICY       207
 #define MQCMD_CHANGE_PROT_POLICY       208
 #define MQCMD_SET_PROT_POLICY          208
 #define MQCMD_ACTIVITY_TRACE           209
 #define MQCMD_RESET_CF_STRUC           213
 #define MQCMD_INQUIRE_XR_CAPABILITY    214
 #define MQCMD_INQUIRE_AMQP_CAPABILITY  216
 #define MQCMD_AMQP_DIAGNOSTICS         217

 /* Control Options */
 #define MQCFC_LAST                     1
 #define MQCFC_NOT_LAST                 0

 /* Reason Codes */
 #define MQRCCF_CFH_TYPE_ERROR          3001
 #define MQRCCF_CFH_LENGTH_ERROR        3002
 #define MQRCCF_CFH_VERSION_ERROR       3003
 #define MQRCCF_CFH_MSG_SEQ_NUMBER_ERR  3004
 #define MQRCCF_CFH_CONTROL_ERROR       3005
 #define MQRCCF_CFH_PARM_COUNT_ERROR    3006
 #define MQRCCF_CFH_COMMAND_ERROR       3007
 #define MQRCCF_COMMAND_FAILED          3008
 #define MQRCCF_CFIN_LENGTH_ERROR       3009
 #define MQRCCF_CFST_LENGTH_ERROR       3010
 #define MQRCCF_CFST_STRING_LENGTH_ERR  3011
 #define MQRCCF_FORCE_VALUE_ERROR       3012
 #define MQRCCF_STRUCTURE_TYPE_ERROR    3013
 #define MQRCCF_CFIN_PARM_ID_ERROR      3014
 #define MQRCCF_CFST_PARM_ID_ERROR      3015
 #define MQRCCF_MSG_LENGTH_ERROR        3016
 #define MQRCCF_CFIN_DUPLICATE_PARM     3017
 #define MQRCCF_CFST_DUPLICATE_PARM     3018
 #define MQRCCF_PARM_COUNT_TOO_SMALL    3019
 #define MQRCCF_PARM_COUNT_TOO_BIG      3020
 #define MQRCCF_Q_ALREADY_IN_CELL       3021
 #define MQRCCF_Q_TYPE_ERROR            3022
 #define MQRCCF_MD_FORMAT_ERROR         3023
 #define MQRCCF_CFSL_LENGTH_ERROR       3024
 #define MQRCCF_REPLACE_VALUE_ERROR     3025
 #define MQRCCF_CFIL_DUPLICATE_VALUE    3026
 #define MQRCCF_CFIL_COUNT_ERROR        3027
 #define MQRCCF_CFIL_LENGTH_ERROR       3028
 #define MQRCCF_QUIESCE_VALUE_ERROR     3029
 #define MQRCCF_MODE_VALUE_ERROR        3029
 #define MQRCCF_MSG_SEQ_NUMBER_ERROR    3030
 #define MQRCCF_PING_DATA_COUNT_ERROR   3031
 #define MQRCCF_PING_DATA_COMPARE_ERROR 3032
 #define MQRCCF_CFSL_PARM_ID_ERROR      3033
 #define MQRCCF_CHANNEL_TYPE_ERROR      3034
 #define MQRCCF_PARM_SEQUENCE_ERROR     3035
 #define MQRCCF_XMIT_PROTOCOL_TYPE_ERR  3036
 #define MQRCCF_BATCH_SIZE_ERROR        3037
 #define MQRCCF_DISC_INT_ERROR          3038
 #define MQRCCF_SHORT_RETRY_ERROR       3039
 #define MQRCCF_SHORT_TIMER_ERROR       3040
 #define MQRCCF_LONG_RETRY_ERROR        3041
 #define MQRCCF_LONG_TIMER_ERROR        3042
 #define MQRCCF_SEQ_NUMBER_WRAP_ERROR   3043
 #define MQRCCF_MAX_MSG_LENGTH_ERROR    3044
 #define MQRCCF_PUT_AUTH_ERROR          3045
 #define MQRCCF_PURGE_VALUE_ERROR       3046
 #define MQRCCF_CFIL_PARM_ID_ERROR      3047
 #define MQRCCF_MSG_TRUNCATED           3048
 #define MQRCCF_CCSID_ERROR             3049
 #define MQRCCF_ENCODING_ERROR          3050
 #define MQRCCF_QUEUES_VALUE_ERROR      3051
 #define MQRCCF_DATA_CONV_VALUE_ERROR   3052
 #define MQRCCF_INDOUBT_VALUE_ERROR     3053
 #define MQRCCF_ESCAPE_TYPE_ERROR       3054
 #define MQRCCF_REPOS_VALUE_ERROR       3055
 #define MQRCCF_CHANNEL_TABLE_ERROR     3062
 #define MQRCCF_MCA_TYPE_ERROR          3063
 #define MQRCCF_CHL_INST_TYPE_ERROR     3064
 #define MQRCCF_CHL_STATUS_NOT_FOUND    3065
 #define MQRCCF_CFSL_DUPLICATE_PARM     3066
 #define MQRCCF_CFSL_TOTAL_LENGTH_ERROR 3067
 #define MQRCCF_CFSL_COUNT_ERROR        3068
 #define MQRCCF_CFSL_STRING_LENGTH_ERR  3069
 #define MQRCCF_BROKER_DELETED          3070
 #define MQRCCF_STREAM_ERROR            3071
 #define MQRCCF_TOPIC_ERROR             3072
 #define MQRCCF_NOT_REGISTERED          3073
 #define MQRCCF_Q_MGR_NAME_ERROR        3074
 #define MQRCCF_INCORRECT_STREAM        3075
 #define MQRCCF_Q_NAME_ERROR            3076
 #define MQRCCF_NO_RETAINED_MSG         3077
 #define MQRCCF_DUPLICATE_IDENTITY      3078
 #define MQRCCF_INCORRECT_Q             3079
 #define MQRCCF_CORREL_ID_ERROR         3080
 #define MQRCCF_NOT_AUTHORIZED          3081
 #define MQRCCF_UNKNOWN_STREAM          3082
 #define MQRCCF_REG_OPTIONS_ERROR       3083
 #define MQRCCF_PUB_OPTIONS_ERROR       3084
 #define MQRCCF_UNKNOWN_BROKER          3085
 #define MQRCCF_Q_MGR_CCSID_ERROR       3086
 #define MQRCCF_DEL_OPTIONS_ERROR       3087
 #define MQRCCF_CLUSTER_NAME_CONFLICT   3088
 #define MQRCCF_REPOS_NAME_CONFLICT     3089
 #define MQRCCF_CLUSTER_Q_USAGE_ERROR   3090
 #define MQRCCF_ACTION_VALUE_ERROR      3091
 #define MQRCCF_COMMS_LIBRARY_ERROR     3092
 #define MQRCCF_NETBIOS_NAME_ERROR      3093
 #define MQRCCF_BROKER_COMMAND_FAILED   3094
 #define MQRCCF_CFST_CONFLICTING_PARM   3095
 #define MQRCCF_PATH_NOT_VALID          3096
 #define MQRCCF_PARM_SYNTAX_ERROR       3097
 #define MQRCCF_PWD_LENGTH_ERROR        3098
 #define MQRCCF_FILTER_ERROR            3150
 #define MQRCCF_WRONG_USER              3151
 #define MQRCCF_DUPLICATE_SUBSCRIPTION  3152
 #define MQRCCF_SUB_NAME_ERROR          3153
 #define MQRCCF_SUB_IDENTITY_ERROR      3154
 #define MQRCCF_SUBSCRIPTION_IN_USE     3155
 #define MQRCCF_SUBSCRIPTION_LOCKED     3156
 #define MQRCCF_ALREADY_JOINED          3157
 #define MQRCCF_OBJECT_IN_USE           3160
 #define MQRCCF_UNKNOWN_FILE_NAME       3161
 #define MQRCCF_FILE_NOT_AVAILABLE      3162
 #define MQRCCF_DISC_RETRY_ERROR        3163
 #define MQRCCF_ALLOC_RETRY_ERROR       3164
 #define MQRCCF_ALLOC_SLOW_TIMER_ERROR  3165
 #define MQRCCF_ALLOC_FAST_TIMER_ERROR  3166
 #define MQRCCF_PORT_NUMBER_ERROR       3167
 #define MQRCCF_CHL_SYSTEM_NOT_ACTIVE   3168
 #define MQRCCF_ENTITY_NAME_MISSING     3169
 #define MQRCCF_PROFILE_NAME_ERROR      3170
 #define MQRCCF_AUTH_VALUE_ERROR        3171
 #define MQRCCF_AUTH_VALUE_MISSING      3172
 #define MQRCCF_OBJECT_TYPE_MISSING     3173
 #define MQRCCF_CONNECTION_ID_ERROR     3174
 #define MQRCCF_LOG_TYPE_ERROR          3175
 #define MQRCCF_PROGRAM_NOT_AVAILABLE   3176
 #define MQRCCF_PROGRAM_AUTH_FAILED     3177
 #define MQRCCF_NONE_FOUND              3200
 #define MQRCCF_SECURITY_SWITCH_OFF     3201
 #define MQRCCF_SECURITY_REFRESH_FAILED 3202
 #define MQRCCF_PARM_CONFLICT           3203
 #define MQRCCF_COMMAND_INHIBITED       3204
 #define MQRCCF_OBJECT_BEING_DELETED    3205
 #define MQRCCF_STORAGE_CLASS_IN_USE    3207
 #define MQRCCF_OBJECT_NAME_RESTRICTED  3208
 #define MQRCCF_OBJECT_LIMIT_EXCEEDED   3209
 #define MQRCCF_OBJECT_OPEN_FORCE       3210
 #define MQRCCF_DISPOSITION_CONFLICT    3211
 #define MQRCCF_Q_MGR_NOT_IN_QSG        3212
 #define MQRCCF_ATTR_VALUE_FIXED        3213
 #define MQRCCF_NAMELIST_ERROR          3215
 #define MQRCCF_NO_CHANNEL_INITIATOR    3217
 #define MQRCCF_CHANNEL_INITIATOR_ERROR 3218
 #define MQRCCF_COMMAND_LEVEL_CONFLICT  3222
 #define MQRCCF_Q_ATTR_CONFLICT         3223
 #define MQRCCF_EVENTS_DISABLED         3224
 #define MQRCCF_COMMAND_SCOPE_ERROR     3225
 #define MQRCCF_COMMAND_REPLY_ERROR     3226
 #define MQRCCF_FUNCTION_RESTRICTED     3227
 #define MQRCCF_PARM_MISSING            3228
 #define MQRCCF_PARM_VALUE_ERROR        3229
 #define MQRCCF_COMMAND_LENGTH_ERROR    3230
 #define MQRCCF_COMMAND_ORIGIN_ERROR    3231
 #define MQRCCF_LISTENER_CONFLICT       3232
 #define MQRCCF_LISTENER_STARTED        3233
 #define MQRCCF_LISTENER_STOPPED        3234
 #define MQRCCF_CHANNEL_ERROR           3235
 #define MQRCCF_CF_STRUC_ERROR          3236
 #define MQRCCF_UNKNOWN_USER_ID         3237
 #define MQRCCF_UNEXPECTED_ERROR        3238
 #define MQRCCF_NO_XCF_PARTNER          3239
 #define MQRCCF_CFGR_PARM_ID_ERROR      3240
 #define MQRCCF_CFIF_LENGTH_ERROR       3241
 #define MQRCCF_CFIF_OPERATOR_ERROR     3242
 #define MQRCCF_CFIF_PARM_ID_ERROR      3243
 #define MQRCCF_CFSF_FILTER_VAL_LEN_ERR 3244
 #define MQRCCF_CFSF_LENGTH_ERROR       3245
 #define MQRCCF_CFSF_OPERATOR_ERROR     3246
 #define MQRCCF_CFSF_PARM_ID_ERROR      3247
 #define MQRCCF_TOO_MANY_FILTERS        3248
 #define MQRCCF_LISTENER_RUNNING        3249
 #define MQRCCF_LSTR_STATUS_NOT_FOUND   3250
 #define MQRCCF_SERVICE_RUNNING         3251
 #define MQRCCF_SERV_STATUS_NOT_FOUND   3252
 #define MQRCCF_SERVICE_STOPPED         3253
 #define MQRCCF_CFBS_DUPLICATE_PARM     3254
 #define MQRCCF_CFBS_LENGTH_ERROR       3255
 #define MQRCCF_CFBS_PARM_ID_ERROR      3256
 #define MQRCCF_CFBS_STRING_LENGTH_ERR  3257
 #define MQRCCF_CFGR_LENGTH_ERROR       3258
 #define MQRCCF_CFGR_PARM_COUNT_ERROR   3259
 #define MQRCCF_CONN_NOT_STOPPED        3260
 #define MQRCCF_SERVICE_REQUEST_PENDING 3261
 #define MQRCCF_NO_START_CMD            3262
 #define MQRCCF_NO_STOP_CMD             3263
 #define MQRCCF_CFBF_LENGTH_ERROR       3264
 #define MQRCCF_CFBF_PARM_ID_ERROR      3265
 #define MQRCCF_CFBF_OPERATOR_ERROR     3266
 #define MQRCCF_CFBF_FILTER_VAL_LEN_ERR 3267
 #define MQRCCF_LISTENER_STILL_ACTIVE   3268
 #define MQRCCF_DEF_XMIT_Q_CLUS_ERROR   3269
 #define MQRCCF_TOPICSTR_ALREADY_EXISTS 3300
 #define MQRCCF_SHARING_CONVS_ERROR     3301
 #define MQRCCF_SHARING_CONVS_TYPE      3302
 #define MQRCCF_SECURITY_CASE_CONFLICT  3303
 #define MQRCCF_TOPIC_TYPE_ERROR        3305
 #define MQRCCF_MAX_INSTANCES_ERROR     3306
 #define MQRCCF_MAX_INSTS_PER_CLNT_ERR  3307
 #define MQRCCF_TOPIC_STRING_NOT_FOUND  3308
 #define MQRCCF_SUBSCRIPTION_POINT_ERR  3309
 #define MQRCCF_SUB_ALREADY_EXISTS      3311
 #define MQRCCF_UNKNOWN_OBJECT_NAME     3312
 #define MQRCCF_REMOTE_Q_NAME_ERROR     3313
 #define MQRCCF_DURABILITY_NOT_ALLOWED  3314
 #define MQRCCF_HOBJ_ERROR              3315
 #define MQRCCF_DEST_NAME_ERROR         3316
 #define MQRCCF_INVALID_DESTINATION     3317
 #define MQRCCF_PUBSUB_INHIBITED        3318
 #define MQRCCF_GROUPUR_CHECKS_FAILED   3319
 #define MQRCCF_COMM_INFO_TYPE_ERROR    3320
 #define MQRCCF_USE_CLIENT_ID_ERROR     3321
 #define MQRCCF_CLIENT_ID_NOT_FOUND     3322
 #define MQRCCF_CLIENT_ID_ERROR         3323
 #define MQRCCF_PORT_IN_USE             3324
 #define MQRCCF_SSL_ALT_PROVIDER_REQD   3325
 #define MQRCCF_CHLAUTH_TYPE_ERROR      3326
 #define MQRCCF_CHLAUTH_ACTION_ERROR    3327
 #define MQRCCF_POLICY_NOT_FOUND        3328
 #define MQRCCF_ENCRYPTION_ALG_ERROR    3329
 #define MQRCCF_SIGNATURE_ALG_ERROR     3330
 #define MQRCCF_TOLERATION_POL_ERROR    3331
 #define MQRCCF_POLICY_VERSION_ERROR    3332
 #define MQRCCF_RECIPIENT_DN_MISSING    3333
 #define MQRCCF_POLICY_NAME_MISSING     3334
 #define MQRCCF_CHLAUTH_USERSRC_ERROR   3335
 #define MQRCCF_WRONG_CHLAUTH_TYPE      3336
 #define MQRCCF_CHLAUTH_ALREADY_EXISTS  3337
 #define MQRCCF_CHLAUTH_NOT_FOUND       3338
 #define MQRCCF_WRONG_CHLAUTH_ACTION    3339
 #define MQRCCF_WRONG_CHLAUTH_USERSRC   3340
 #define MQRCCF_CHLAUTH_WARN_ERROR      3341
 #define MQRCCF_WRONG_CHLAUTH_MATCH     3342
 #define MQRCCF_IPADDR_RANGE_CONFLICT   3343
 #define MQRCCF_CHLAUTH_MAX_EXCEEDED    3344
 #define MQRCCF_IPADDR_ERROR            3345
 #define MQRCCF_ADDRESS_ERROR           3345
 #define MQRCCF_IPADDR_RANGE_ERROR      3346
 #define MQRCCF_PROFILE_NAME_MISSING    3347
 #define MQRCCF_CHLAUTH_CLNTUSER_ERROR  3348
 #define MQRCCF_CHLAUTH_NAME_ERROR      3349
 #define MQRCCF_CHLAUTH_RUNCHECK_ERROR  3350
 #define MQRCCF_CF_STRUC_ALREADY_FAILED 3351
 #define MQRCCF_CFCONLOS_CHECKS_FAILED  3352
 #define MQRCCF_SUITE_B_ERROR           3353
 #define MQRCCF_CHANNEL_NOT_STARTED     3354
 #define MQRCCF_CUSTOM_ERROR            3355
 #define MQRCCF_BACKLOG_OUT_OF_RANGE    3356
 #define MQRCCF_CHLAUTH_DISABLED        3357
 #define MQRCCF_SMDS_REQUIRES_DSGROUP   3358
 #define MQRCCF_PSCLUS_DISABLED_TOPDEF  3359
 #define MQRCCF_PSCLUS_TOPIC_EXISTS     3360
 #define MQRCCF_SSL_CIPHER_SUITE_ERROR  3361
 #define MQRCCF_SOCKET_ERROR            3362
 #define MQRCCF_CLUS_XMIT_Q_USAGE_ERROR 3363
 #define MQRCCF_CERT_VAL_POLICY_ERROR   3364
 #define MQRCCF_INVALID_PROTOCOL        3365
 #define MQRCCF_REVDNS_DISABLED         3366
 #define MQRCCF_CLROUTE_NOT_ALTERABLE   3367
 #define MQRCCF_CLUSTER_TOPIC_CONFLICT  3368
 #define MQRCCF_DEFCLXQ_MODEL_Q_ERROR   3369
 #define MQRCCF_CHLAUTH_CHKCLI_ERROR    3370
 #define MQRCCF_CERT_LABEL_NOT_ALLOWED  3371
 #define MQRCCF_Q_MGR_ATTR_CONFLICT     3372
 #define MQRCCF_ENTITY_TYPE_MISSING     3373
 #define MQRCCF_CLWL_EXIT_NAME_ERROR    3374
 #define MQRCCF_SERVICE_NAME_ERROR      3375
 #define MQRCCF_REMOTE_CHL_TYPE_ERROR   3376
 #define MQRCCF_TOPIC_RESTRICTED        3377
 #define MQRCCF_CURRENT_LOG_EXTENT      3378
 #define MQRCCF_LOG_EXTENT_NOT_FOUND    3379
 #define MQRCCF_LOG_NOT_REDUCED         3380
 #define MQRCCF_LOG_EXTENT_ERROR        3381
 #define MQRCCF_ACCESS_BLOCKED          3382
 #define MQRCCF_OBJECT_ALREADY_EXISTS   4001
 #define MQRCCF_OBJECT_WRONG_TYPE       4002
 #define MQRCCF_LIKE_OBJECT_WRONG_TYPE  4003
 #define MQRCCF_OBJECT_OPEN             4004
 #define MQRCCF_ATTR_VALUE_ERROR        4005
 #define MQRCCF_UNKNOWN_Q_MGR           4006
 #define MQRCCF_Q_WRONG_TYPE            4007
 #define MQRCCF_OBJECT_NAME_ERROR       4008
 #define MQRCCF_ALLOCATE_FAILED         4009
 #define MQRCCF_HOST_NOT_AVAILABLE      4010
 #define MQRCCF_CONFIGURATION_ERROR     4011
 #define MQRCCF_CONNECTION_REFUSED      4012
 #define MQRCCF_ENTRY_ERROR             4013
 #define MQRCCF_SEND_FAILED             4014
 #define MQRCCF_RECEIVED_DATA_ERROR     4015
 #define MQRCCF_RECEIVE_FAILED          4016
 #define MQRCCF_CONNECTION_CLOSED       4017
 #define MQRCCF_NO_STORAGE              4018
 #define MQRCCF_NO_COMMS_MANAGER        4019
 #define MQRCCF_LISTENER_NOT_STARTED    4020
 #define MQRCCF_BIND_FAILED             4024
 #define MQRCCF_CHANNEL_INDOUBT         4025
 #define MQRCCF_MQCONN_FAILED           4026
 #define MQRCCF_MQOPEN_FAILED           4027
 #define MQRCCF_MQGET_FAILED            4028
 #define MQRCCF_MQPUT_FAILED            4029
 #define MQRCCF_PING_ERROR              4030
 #define MQRCCF_CHANNEL_IN_USE          4031
 #define MQRCCF_CHANNEL_NOT_FOUND       4032
 #define MQRCCF_UNKNOWN_REMOTE_CHANNEL  4033
 #define MQRCCF_REMOTE_QM_UNAVAILABLE   4034
 #define MQRCCF_REMOTE_QM_TERMINATING   4035
 #define MQRCCF_MQINQ_FAILED            4036
 #define MQRCCF_NOT_XMIT_Q              4037
 #define MQRCCF_CHANNEL_DISABLED        4038
 #define MQRCCF_USER_EXIT_NOT_AVAILABLE 4039
 #define MQRCCF_COMMIT_FAILED           4040
 #define MQRCCF_WRONG_CHANNEL_TYPE      4041
 #define MQRCCF_CHANNEL_ALREADY_EXISTS  4042
 #define MQRCCF_DATA_TOO_LARGE          4043
 #define MQRCCF_CHANNEL_NAME_ERROR      4044
 #define MQRCCF_XMIT_Q_NAME_ERROR       4045
 #define MQRCCF_MCA_NAME_ERROR          4047
 #define MQRCCF_SEND_EXIT_NAME_ERROR    4048
 #define MQRCCF_SEC_EXIT_NAME_ERROR     4049
 #define MQRCCF_MSG_EXIT_NAME_ERROR     4050
 #define MQRCCF_RCV_EXIT_NAME_ERROR     4051
 #define MQRCCF_XMIT_Q_NAME_WRONG_TYPE  4052
 #define MQRCCF_MCA_NAME_WRONG_TYPE     4053
 #define MQRCCF_DISC_INT_WRONG_TYPE     4054
 #define MQRCCF_SHORT_RETRY_WRONG_TYPE  4055
 #define MQRCCF_SHORT_TIMER_WRONG_TYPE  4056
 #define MQRCCF_LONG_RETRY_WRONG_TYPE   4057
 #define MQRCCF_LONG_TIMER_WRONG_TYPE   4058
 #define MQRCCF_PUT_AUTH_WRONG_TYPE     4059
 #define MQRCCF_KEEP_ALIVE_INT_ERROR    4060
 #define MQRCCF_MISSING_CONN_NAME       4061
 #define MQRCCF_CONN_NAME_ERROR         4062
 #define MQRCCF_MQSET_FAILED            4063
 #define MQRCCF_CHANNEL_NOT_ACTIVE      4064
 #define MQRCCF_TERMINATED_BY_SEC_EXIT  4065
 #define MQRCCF_DYNAMIC_Q_SCOPE_ERROR   4067
 #define MQRCCF_CELL_DIR_NOT_AVAILABLE  4068
 #define MQRCCF_MR_COUNT_ERROR          4069
 #define MQRCCF_MR_COUNT_WRONG_TYPE     4070
 #define MQRCCF_MR_EXIT_NAME_ERROR      4071
 #define MQRCCF_MR_EXIT_NAME_WRONG_TYPE 4072
 #define MQRCCF_MR_INTERVAL_ERROR       4073
 #define MQRCCF_MR_INTERVAL_WRONG_TYPE  4074
 #define MQRCCF_NPM_SPEED_ERROR         4075
 #define MQRCCF_NPM_SPEED_WRONG_TYPE    4076
 #define MQRCCF_HB_INTERVAL_ERROR       4077
 #define MQRCCF_HB_INTERVAL_WRONG_TYPE  4078
 #define MQRCCF_CHAD_ERROR              4079
 #define MQRCCF_CHAD_WRONG_TYPE         4080
 #define MQRCCF_CHAD_EVENT_ERROR        4081
 #define MQRCCF_CHAD_EVENT_WRONG_TYPE   4082
 #define MQRCCF_CHAD_EXIT_ERROR         4083
 #define MQRCCF_CHAD_EXIT_WRONG_TYPE    4084
 #define MQRCCF_SUPPRESSED_BY_EXIT      4085
 #define MQRCCF_BATCH_INT_ERROR         4086
 #define MQRCCF_BATCH_INT_WRONG_TYPE    4087
 #define MQRCCF_NET_PRIORITY_ERROR      4088
 #define MQRCCF_NET_PRIORITY_WRONG_TYPE 4089
 #define MQRCCF_CHANNEL_CLOSED          4090
 #define MQRCCF_Q_STATUS_NOT_FOUND      4091
 #define MQRCCF_SSL_CIPHER_SPEC_ERROR   4092
 #define MQRCCF_SSL_PEER_NAME_ERROR     4093
 #define MQRCCF_SSL_CLIENT_AUTH_ERROR   4094
 #define MQRCCF_RETAINED_NOT_SUPPORTED  4095

 /****************************************************************/
 /* Values Related to MQCFBF Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFBF_STRUC_LENGTH_FIXED      20

 /****************************************************************/
 /* Values Related to MQCFBS Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFBS_STRUC_LENGTH_FIXED      16

 /****************************************************************/
 /* Values Related to MQCFGR Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQCFGR_STRUC_LENGTH            16

 /****************************************************************/
 /* Values Related to MQCFIF Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQCFIF_STRUC_LENGTH            20

 /****************************************************************/
 /* Values Related to MQCFIL Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFIL_STRUC_LENGTH_FIXED      16

 /****************************************************************/
 /* Values Related to MQCFIL64 Structure                         */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFIL64_STRUC_LENGTH_FIXED    16

 /****************************************************************/
 /* Values Related to MQCFIN Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQCFIN_STRUC_LENGTH            16

 /****************************************************************/
 /* Values Related to MQCFIN64 Structure                         */
 /****************************************************************/

 /* Structure Length */
 #define MQCFIN64_STRUC_LENGTH          24

 /****************************************************************/
 /* Values Related to MQCFSF Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFSF_STRUC_LENGTH_FIXED      24

 /****************************************************************/
 /* Values Related to MQCFSL Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFSL_STRUC_LENGTH_FIXED      24

 /****************************************************************/
 /* Values Related to MQCFST Structure                           */
 /****************************************************************/

 /* Structure Length (Fixed Part) */
 #define MQCFST_STRUC_LENGTH_FIXED      20

 /****************************************************************/
 /* Values Related to MQEPH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQEPH_STRUC_ID                 "EPH "

 /* Structure Identifier (array form) */
 #define MQEPH_STRUC_ID_ARRAY           'E','P','H',' '

 /* Structure Length (Fixed Part) */
 #define MQEPH_STRUC_LENGTH_FIXED       68

 /* Structure Version Number */
 #define MQEPH_VERSION_1                1
 #define MQEPH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQEPH_LENGTH_1                 68
 #define MQEPH_CURRENT_LENGTH           68

 /* Flags */
 #define MQEPH_NONE                     0x00000000
 #define MQEPH_CCSID_EMBEDDED           0x00000001

 /****************************************************************/
 /* Values Related to All Structures                             */
 /****************************************************************/

 /* String Lengths */
 #define MQ_ARCHIVE_PFX_LENGTH          36
 #define MQ_ARCHIVE_UNIT_LENGTH         8
 #define MQ_ASID_LENGTH                 4
 #define MQ_AUTH_PROFILE_NAME_LENGTH    48
 #define MQ_CF_LEID_LENGTH              12
 #define MQ_COMMAND_MQSC_LENGTH         32768
 #define MQ_DATA_SET_NAME_LENGTH        44
 #define MQ_DB2_NAME_LENGTH             4
 #define MQ_DSG_NAME_LENGTH             8
 #define MQ_ENTITY_NAME_LENGTH          1024
 #define MQ_ENV_INFO_LENGTH             96
 #define MQ_GROUP_ADDRESS_LENGTH        264
 #define MQ_IP_ADDRESS_LENGTH           48
 #define MQ_LOG_CORREL_ID_LENGTH        8
 #define MQ_LOG_EXTENT_NAME_LENGTH      24
 #define MQ_LOG_PATH_LENGTH             1024
 #define MQ_LRSN_LENGTH                 12
 #define MQ_ORIGIN_NAME_LENGTH          8
 #define MQ_PSB_NAME_LENGTH             8
 #define MQ_PST_ID_LENGTH               8
 #define MQ_Q_MGR_CPF_LENGTH            4
 #define MQ_RESPONSE_ID_LENGTH          24
 #define MQ_RBA_LENGTH                  16
 #define MQ_SECURITY_PROFILE_LENGTH     40
 #define MQ_SERVICE_COMPONENT_LENGTH    48
 #define MQ_SUB_NAME_LENGTH             10240
 #define MQ_SYSP_SERVICE_LENGTH         32
 #define MQ_SYSTEM_NAME_LENGTH          8
 #define MQ_TASK_NUMBER_LENGTH          8
 #define MQ_TPIPE_PFX_LENGTH            4
 #define MQ_UOW_ID_LENGTH               256
 #define MQ_USER_DATA_LENGTH            10240
 #define MQ_VOLSER_LENGTH               6
 #define MQ_REMOTE_PRODUCT_LENGTH       4
 #define MQ_REMOTE_VERSION_LENGTH       8

 /* Filter Operators */
 #define MQCFOP_LESS                    1
 #define MQCFOP_EQUAL                   2
 #define MQCFOP_GREATER                 4
 #define MQCFOP_NOT_LESS                6
 #define MQCFOP_NOT_EQUAL               5
 #define MQCFOP_NOT_GREATER             3
 #define MQCFOP_LIKE                    18
 #define MQCFOP_NOT_LIKE                21
 #define MQCFOP_CONTAINS                10
 #define MQCFOP_EXCLUDES                13
 #define MQCFOP_CONTAINS_GEN            26
 #define MQCFOP_EXCLUDES_GEN            29

 /* Types of Structure */
 #define MQCFT_NONE                     0
 #define MQCFT_COMMAND                  1
 #define MQCFT_RESPONSE                 2
 #define MQCFT_INTEGER                  3
 #define MQCFT_STRING                   4
 #define MQCFT_INTEGER_LIST             5
 #define MQCFT_STRING_LIST              6
 #define MQCFT_EVENT                    7
 #define MQCFT_USER                     8
 #define MQCFT_BYTE_STRING              9
 #define MQCFT_TRACE_ROUTE              10
 #define MQCFT_REPORT                   12
 #define MQCFT_INTEGER_FILTER           13
 #define MQCFT_STRING_FILTER            14
 #define MQCFT_BYTE_STRING_FILTER       15
 #define MQCFT_COMMAND_XR               16
 #define MQCFT_XR_MSG                   17
 #define MQCFT_XR_ITEM                  18
 #define MQCFT_XR_SUMMARY               19
 #define MQCFT_GROUP                    20
 #define MQCFT_STATISTICS               21
 #define MQCFT_ACCOUNTING               22
 #define MQCFT_INTEGER64                23
 #define MQCFT_INTEGER64_LIST           25
 #define MQCFT_APP_ACTIVITY             26

 /* Major Release Function */
 #define MQOPMODE_COMPAT                0
 #define MQOPMODE_NEW_FUNCTION          1

 /****************************************************************/
 /* Values Related to Byte Parameter Structures                  */
 /****************************************************************/

 /* Byte Parameter Types */
 #define MQBACF_FIRST                   7001
 #define MQBACF_EVENT_ACCOUNTING_TOKEN  7001
 #define MQBACF_EVENT_SECURITY_ID       7002
 #define MQBACF_RESPONSE_SET            7003
 #define MQBACF_RESPONSE_ID             7004
 #define MQBACF_EXTERNAL_UOW_ID         7005
 #define MQBACF_CONNECTION_ID           7006
 #define MQBACF_GENERIC_CONNECTION_ID   7007
 #define MQBACF_ORIGIN_UOW_ID           7008
 #define MQBACF_Q_MGR_UOW_ID            7009
 #define MQBACF_ACCOUNTING_TOKEN        7010
 #define MQBACF_CORREL_ID               7011
 #define MQBACF_GROUP_ID                7012
 #define MQBACF_MSG_ID                  7013
 #define MQBACF_CF_LEID                 7014
 #define MQBACF_DESTINATION_CORREL_ID   7015
 #define MQBACF_SUB_ID                  7016
 #define MQBACF_ALTERNATE_SECURITYID    7019
 #define MQBACF_MESSAGE_DATA            7020
 #define MQBACF_MQBO_STRUCT             7021
 #define MQBACF_MQCB_FUNCTION           7022
 #define MQBACF_MQCBC_STRUCT            7023
 #define MQBACF_MQCBD_STRUCT            7024
 #define MQBACF_MQCD_STRUCT             7025
 #define MQBACF_MQCNO_STRUCT            7026
 #define MQBACF_MQGMO_STRUCT            7027
 #define MQBACF_MQMD_STRUCT             7028
 #define MQBACF_MQPMO_STRUCT            7029
 #define MQBACF_MQSD_STRUCT             7030
 #define MQBACF_MQSTS_STRUCT            7031
 #define MQBACF_SUB_CORREL_ID           7032
 #define MQBACF_XA_XID                  7033
 #define MQBACF_XQH_CORREL_ID           7034
 #define MQBACF_XQH_MSG_ID              7035
 #define MQBACF_LAST_USED               7035

 /****************************************************************/
 /* Values Related to Integer Parameter Structures               */
 /****************************************************************/

 /* Integer Monitoring Parameter Types */
 #define MQIAMO_FIRST                   701
 #define MQIAMO_AVG_BATCH_SIZE          702
 #define MQIAMO_AVG_Q_TIME              703
 #define MQIAMO64_AVG_Q_TIME            703
 #define MQIAMO_BACKOUTS                704
 #define MQIAMO_BROWSES                 705
 #define MQIAMO_BROWSE_MAX_BYTES        706
 #define MQIAMO_BROWSE_MIN_BYTES        707
 #define MQIAMO_BROWSES_FAILED          708
 #define MQIAMO_CLOSES                  709
 #define MQIAMO_COMMITS                 710
 #define MQIAMO_COMMITS_FAILED          711
 #define MQIAMO_CONNS                   712
 #define MQIAMO_CONNS_MAX               713
 #define MQIAMO_DISCS                   714
 #define MQIAMO_DISCS_IMPLICIT          715
 #define MQIAMO_DISC_TYPE               716
 #define MQIAMO_EXIT_TIME_AVG           717
 #define MQIAMO_EXIT_TIME_MAX           718
 #define MQIAMO_EXIT_TIME_MIN           719
 #define MQIAMO_FULL_BATCHES            720
 #define MQIAMO_GENERATED_MSGS          721
 #define MQIAMO_GETS                    722
 #define MQIAMO_GET_MAX_BYTES           723
 #define MQIAMO_GET_MIN_BYTES           724
 #define MQIAMO_GETS_FAILED             725
 #define MQIAMO_INCOMPLETE_BATCHES      726
 #define MQIAMO_INQS                    727
 #define MQIAMO_MSGS                    728
 #define MQIAMO_NET_TIME_AVG            729
 #define MQIAMO_NET_TIME_MAX            730
 #define MQIAMO_NET_TIME_MIN            731
 #define MQIAMO_OBJECT_COUNT            732
 #define MQIAMO_OPENS                   733
 #define MQIAMO_PUT1S                   734
 #define MQIAMO_PUTS                    735
 #define MQIAMO_PUT_MAX_BYTES           736
 #define MQIAMO_PUT_MIN_BYTES           737
 #define MQIAMO_PUT_RETRIES             738
 #define MQIAMO_Q_MAX_DEPTH             739
 #define MQIAMO_Q_MIN_DEPTH             740
 #define MQIAMO_Q_TIME_AVG              741
 #define MQIAMO64_Q_TIME_AVG            741
 #define MQIAMO_Q_TIME_MAX              742
 #define MQIAMO64_Q_TIME_MAX            742
 #define MQIAMO_Q_TIME_MIN              743
 #define MQIAMO64_Q_TIME_MIN            743
 #define MQIAMO_SETS                    744
 #define MQIAMO64_BROWSE_BYTES          745
 #define MQIAMO64_BYTES                 746
 #define MQIAMO64_GET_BYTES             747
 #define MQIAMO64_PUT_BYTES             748
 #define MQIAMO_CONNS_FAILED            749
 #define MQIAMO_OPENS_FAILED            751
 #define MQIAMO_INQS_FAILED             752
 #define MQIAMO_SETS_FAILED             753
 #define MQIAMO_PUTS_FAILED             754
 #define MQIAMO_PUT1S_FAILED            755
 #define MQIAMO_CLOSES_FAILED           757
 #define MQIAMO_MSGS_EXPIRED            758
 #define MQIAMO_MSGS_NOT_QUEUED         759
 #define MQIAMO_MSGS_PURGED             760
 #define MQIAMO_SUBS_DUR                764
 #define MQIAMO_SUBS_NDUR               765
 #define MQIAMO_SUBS_FAILED             766
 #define MQIAMO_SUBRQS                  767
 #define MQIAMO_SUBRQS_FAILED           768
 #define MQIAMO_CBS                     769
 #define MQIAMO_CBS_FAILED              770
 #define MQIAMO_CTLS                    771
 #define MQIAMO_CTLS_FAILED             772
 #define MQIAMO_STATS                   773
 #define MQIAMO_STATS_FAILED            774
 #define MQIAMO_SUB_DUR_HIGHWATER       775
 #define MQIAMO_SUB_DUR_LOWWATER        776
 #define MQIAMO_SUB_NDUR_HIGHWATER      777
 #define MQIAMO_SUB_NDUR_LOWWATER       778
 #define MQIAMO_TOPIC_PUTS              779
 #define MQIAMO_TOPIC_PUTS_FAILED       780
 #define MQIAMO_TOPIC_PUT1S             781
 #define MQIAMO_TOPIC_PUT1S_FAILED      782
 #define MQIAMO64_TOPIC_PUT_BYTES       783
 #define MQIAMO_PUBLISH_MSG_COUNT       784
 #define MQIAMO64_PUBLISH_MSG_BYTES     785
 #define MQIAMO_UNSUBS_DUR              786
 #define MQIAMO_UNSUBS_NDUR             787
 #define MQIAMO_UNSUBS_FAILED           788
 #define MQIAMO_INTERVAL                789
 #define MQIAMO_MSGS_SENT               790
 #define MQIAMO_BYTES_SENT              791
 #define MQIAMO_REPAIR_BYTES            792
 #define MQIAMO_FEEDBACK_MODE           793
 #define MQIAMO_RELIABILITY_TYPE        794
 #define MQIAMO_LATE_JOIN_MARK          795
 #define MQIAMO_NACKS_RCVD              796
 #define MQIAMO_REPAIR_PKTS             797
 #define MQIAMO_HISTORY_PKTS            798
 #define MQIAMO_PENDING_PKTS            799
 #define MQIAMO_PKT_RATE                800
 #define MQIAMO_MCAST_XMIT_RATE         801
 #define MQIAMO_MCAST_BATCH_TIME        802
 #define MQIAMO_MCAST_HEARTBEAT         803
 #define MQIAMO_DEST_DATA_PORT          804
 #define MQIAMO_DEST_REPAIR_PORT        805
 #define MQIAMO_ACKS_RCVD               806
 #define MQIAMO_ACTIVE_ACKERS           807
 #define MQIAMO_PKTS_SENT               808
 #define MQIAMO_TOTAL_REPAIR_PKTS       809
 #define MQIAMO_TOTAL_PKTS_SENT         810
 #define MQIAMO_TOTAL_MSGS_SENT         811
 #define MQIAMO_TOTAL_BYTES_SENT        812
 #define MQIAMO_NUM_STREAMS             813
 #define MQIAMO_ACK_FEEDBACK            814
 #define MQIAMO_NACK_FEEDBACK           815
 #define MQIAMO_PKTS_LOST               816
 #define MQIAMO_MSGS_RCVD               817
 #define MQIAMO_MSG_BYTES_RCVD          818
 #define MQIAMO_MSGS_DELIVERED          819
 #define MQIAMO_PKTS_PROCESSED          820
 #define MQIAMO_PKTS_DELIVERED          821
 #define MQIAMO_PKTS_DROPPED            822
 #define MQIAMO_PKTS_DUPLICATED         823
 #define MQIAMO_NACKS_CREATED           824
 #define MQIAMO_NACK_PKTS_SENT          825
 #define MQIAMO_REPAIR_PKTS_RQSTD       826
 #define MQIAMO_REPAIR_PKTS_RCVD        827
 #define MQIAMO_PKTS_REPAIRED           828
 #define MQIAMO_TOTAL_MSGS_RCVD         829
 #define MQIAMO_TOTAL_MSG_BYTES_RCVD    830
 #define MQIAMO_TOTAL_REPAIR_PKTS_RCVD  831
 #define MQIAMO_TOTAL_REPAIR_PKTS_RQSTD 832
 #define MQIAMO_TOTAL_MSGS_PROCESSED    833
 #define MQIAMO_TOTAL_MSGS_SELECTED     834
 #define MQIAMO_TOTAL_MSGS_EXPIRED      835
 #define MQIAMO_TOTAL_MSGS_DELIVERED    836
 #define MQIAMO_TOTAL_MSGS_RETURNED     837
 #define MQIAMO64_HIGHRES_TIME          838
 #define MQIAMO_MONITOR_CLASS           839
 #define MQIAMO_MONITOR_TYPE            840
 #define MQIAMO_MONITOR_ELEMENT         841
 #define MQIAMO_MONITOR_DATATYPE        842
 #define MQIAMO_MONITOR_FLAGS           843
 #define MQIAMO64_QMGR_OP_DURATION      844
 #define MQIAMO64_MONITOR_INTERVAL      845
 #define MQIAMO_LAST_USED               845

 /* Defined values for MQIAMO_MONITOR_FLAGS */
 #define MQIAMO_MONITOR_FLAGS_NONE      0
 #define MQIAMO_MONITOR_FLAGS_OBJNAME   1

 /* Defined values for MQIAMO_MONITOR_DATATYPE */
 #define MQIAMO_MONITOR_UNIT            1
 #define MQIAMO_MONITOR_DELTA           2
 #define MQIAMO_MONITOR_HUNDREDTHS      100
 #define MQIAMO_MONITOR_KB              1024
 #define MQIAMO_MONITOR_PERCENT         10000
 #define MQIAMO_MONITOR_MICROSEC        1000000
 #define MQIAMO_MONITOR_MB              1048576
 #define MQIAMO_MONITOR_GB              100000000

 /* Integer Parameter Types */
 #define MQIACF_FIRST                   1001
 #define MQIACF_Q_MGR_ATTRS             1001
 #define MQIACF_Q_ATTRS                 1002
 #define MQIACF_PROCESS_ATTRS           1003
 #define MQIACF_NAMELIST_ATTRS          1004
 #define MQIACF_FORCE                   1005
 #define MQIACF_REPLACE                 1006
 #define MQIACF_PURGE                   1007
 #define MQIACF_QUIESCE                 1008
 #define MQIACF_MODE                    1008
 #define MQIACF_ALL                     1009
 #define MQIACF_EVENT_APPL_TYPE         1010
 #define MQIACF_EVENT_ORIGIN            1011
 #define MQIACF_PARAMETER_ID            1012
 #define MQIACF_ERROR_ID                1013
 #define MQIACF_ERROR_IDENTIFIER        1013
 #define MQIACF_SELECTOR                1014
 #define MQIACF_CHANNEL_ATTRS           1015
 #define MQIACF_OBJECT_TYPE             1016
 #define MQIACF_ESCAPE_TYPE             1017
 #define MQIACF_ERROR_OFFSET            1018
 #define MQIACF_AUTH_INFO_ATTRS         1019
 #define MQIACF_REASON_QUALIFIER        1020
 #define MQIACF_COMMAND                 1021
 #define MQIACF_OPEN_OPTIONS            1022
 #define MQIACF_OPEN_TYPE               1023
 #define MQIACF_PROCESS_ID              1024
 #define MQIACF_THREAD_ID               1025
 #define MQIACF_Q_STATUS_ATTRS          1026
 #define MQIACF_UNCOMMITTED_MSGS        1027
 #define MQIACF_HANDLE_STATE            1028
 #define MQIACF_AUX_ERROR_DATA_INT_1    1070
 #define MQIACF_AUX_ERROR_DATA_INT_2    1071
 #define MQIACF_CONV_REASON_CODE        1072
 #define MQIACF_BRIDGE_TYPE             1073
 #define MQIACF_INQUIRY                 1074
 #define MQIACF_WAIT_INTERVAL           1075
 #define MQIACF_OPTIONS                 1076
 #define MQIACF_BROKER_OPTIONS          1077
 #define MQIACF_REFRESH_TYPE            1078
 #define MQIACF_SEQUENCE_NUMBER         1079
 #define MQIACF_INTEGER_DATA            1080
 #define MQIACF_REGISTRATION_OPTIONS    1081
 #define MQIACF_PUBLICATION_OPTIONS     1082
 #define MQIACF_CLUSTER_INFO            1083
 #define MQIACF_Q_MGR_DEFINITION_TYPE   1084
 #define MQIACF_Q_MGR_TYPE              1085
 #define MQIACF_ACTION                  1086
 #define MQIACF_SUSPEND                 1087
 #define MQIACF_BROKER_COUNT            1088
 #define MQIACF_APPL_COUNT              1089
 #define MQIACF_ANONYMOUS_COUNT         1090
 #define MQIACF_REG_REG_OPTIONS         1091
 #define MQIACF_DELETE_OPTIONS          1092
 #define MQIACF_CLUSTER_Q_MGR_ATTRS     1093
 #define MQIACF_REFRESH_INTERVAL        1094
 #define MQIACF_REFRESH_REPOSITORY      1095
 #define MQIACF_REMOVE_QUEUES           1096
 #define MQIACF_OPEN_INPUT_TYPE         1098
 #define MQIACF_OPEN_OUTPUT             1099
 #define MQIACF_OPEN_SET                1100
 #define MQIACF_OPEN_INQUIRE            1101
 #define MQIACF_OPEN_BROWSE             1102
 #define MQIACF_Q_STATUS_TYPE           1103
 #define MQIACF_Q_HANDLE                1104
 #define MQIACF_Q_STATUS                1105
 #define MQIACF_SECURITY_TYPE           1106
 #define MQIACF_CONNECTION_ATTRS        1107
 #define MQIACF_CONNECT_OPTIONS         1108
 #define MQIACF_CONN_INFO_TYPE          1110
 #define MQIACF_CONN_INFO_CONN          1111
 #define MQIACF_CONN_INFO_HANDLE        1112
 #define MQIACF_CONN_INFO_ALL           1113
 #define MQIACF_AUTH_PROFILE_ATTRS      1114
 #define MQIACF_AUTHORIZATION_LIST      1115
 #define MQIACF_AUTH_ADD_AUTHS          1116
 #define MQIACF_AUTH_REMOVE_AUTHS       1117
 #define MQIACF_ENTITY_TYPE             1118
 #define MQIACF_COMMAND_INFO            1120
 #define MQIACF_CMDSCOPE_Q_MGR_COUNT    1121
 #define MQIACF_Q_MGR_SYSTEM            1122
 #define MQIACF_Q_MGR_EVENT             1123
 #define MQIACF_Q_MGR_DQM               1124
 #define MQIACF_Q_MGR_CLUSTER           1125
 #define MQIACF_QSG_DISPS               1126
 #define MQIACF_UOW_STATE               1128
 #define MQIACF_SECURITY_ITEM           1129
 #define MQIACF_CF_STRUC_STATUS         1130
 #define MQIACF_UOW_TYPE                1132
 #define MQIACF_CF_STRUC_ATTRS          1133
 #define MQIACF_EXCLUDE_INTERVAL        1134
 #define MQIACF_CF_STATUS_TYPE          1135
 #define MQIACF_CF_STATUS_SUMMARY       1136
 #define MQIACF_CF_STATUS_CONNECT       1137
 #define MQIACF_CF_STATUS_BACKUP        1138
 #define MQIACF_CF_STRUC_TYPE           1139
 #define MQIACF_CF_STRUC_SIZE_MAX       1140
 #define MQIACF_CF_STRUC_SIZE_USED      1141
 #define MQIACF_CF_STRUC_ENTRIES_MAX    1142
 #define MQIACF_CF_STRUC_ENTRIES_USED   1143
 #define MQIACF_CF_STRUC_BACKUP_SIZE    1144
 #define MQIACF_MOVE_TYPE               1145
 #define MQIACF_MOVE_TYPE_MOVE          1146
 #define MQIACF_MOVE_TYPE_ADD           1147
 #define MQIACF_Q_MGR_NUMBER            1148
 #define MQIACF_Q_MGR_STATUS            1149
 #define MQIACF_DB2_CONN_STATUS         1150
 #define MQIACF_SECURITY_ATTRS          1151
 #define MQIACF_SECURITY_TIMEOUT        1152
 #define MQIACF_SECURITY_INTERVAL       1153
 #define MQIACF_SECURITY_SWITCH         1154
 #define MQIACF_SECURITY_SETTING        1155
 #define MQIACF_STORAGE_CLASS_ATTRS     1156
 #define MQIACF_USAGE_TYPE              1157
 #define MQIACF_BUFFER_POOL_ID          1158
 #define MQIACF_USAGE_TOTAL_PAGES       1159
 #define MQIACF_USAGE_UNUSED_PAGES      1160
 #define MQIACF_USAGE_PERSIST_PAGES     1161
 #define MQIACF_USAGE_NONPERSIST_PAGES  1162
 #define MQIACF_USAGE_RESTART_EXTENTS   1163
 #define MQIACF_USAGE_EXPAND_COUNT      1164
 #define MQIACF_PAGESET_STATUS          1165
 #define MQIACF_USAGE_TOTAL_BUFFERS     1166
 #define MQIACF_USAGE_DATA_SET_TYPE     1167
 #define MQIACF_USAGE_PAGESET           1168
 #define MQIACF_USAGE_DATA_SET          1169
 #define MQIACF_USAGE_BUFFER_POOL       1170
 #define MQIACF_MOVE_COUNT              1171
 #define MQIACF_EXPIRY_Q_COUNT          1172
 #define MQIACF_CONFIGURATION_OBJECTS   1173
 #define MQIACF_CONFIGURATION_EVENTS    1174
 #define MQIACF_SYSP_TYPE               1175
 #define MQIACF_SYSP_DEALLOC_INTERVAL   1176
 #define MQIACF_SYSP_MAX_ARCHIVE        1177
 #define MQIACF_SYSP_MAX_READ_TAPES     1178
 #define MQIACF_SYSP_IN_BUFFER_SIZE     1179
 #define MQIACF_SYSP_OUT_BUFFER_SIZE    1180
 #define MQIACF_SYSP_OUT_BUFFER_COUNT   1181
 #define MQIACF_SYSP_ARCHIVE            1182
 #define MQIACF_SYSP_DUAL_ACTIVE        1183
 #define MQIACF_SYSP_DUAL_ARCHIVE       1184
 #define MQIACF_SYSP_DUAL_BSDS          1185
 #define MQIACF_SYSP_MAX_CONNS          1186
 #define MQIACF_SYSP_MAX_CONNS_FORE     1187
 #define MQIACF_SYSP_MAX_CONNS_BACK     1188
 #define MQIACF_SYSP_EXIT_INTERVAL      1189
 #define MQIACF_SYSP_EXIT_TASKS         1190
 #define MQIACF_SYSP_CHKPOINT_COUNT     1191
 #define MQIACF_SYSP_OTMA_INTERVAL      1192
 #define MQIACF_SYSP_Q_INDEX_DEFER      1193
 #define MQIACF_SYSP_DB2_TASKS          1194
 #define MQIACF_SYSP_RESLEVEL_AUDIT     1195
 #define MQIACF_SYSP_ROUTING_CODE       1196
 #define MQIACF_SYSP_SMF_ACCOUNTING     1197
 #define MQIACF_SYSP_SMF_STATS          1198
 #define MQIACF_SYSP_SMF_INTERVAL       1199
 #define MQIACF_SYSP_TRACE_CLASS        1200
 #define MQIACF_SYSP_TRACE_SIZE         1201
 #define MQIACF_SYSP_WLM_INTERVAL       1202
 #define MQIACF_SYSP_ALLOC_UNIT         1203
 #define MQIACF_SYSP_ARCHIVE_RETAIN     1204
 #define MQIACF_SYSP_ARCHIVE_WTOR       1205
 #define MQIACF_SYSP_BLOCK_SIZE         1206
 #define MQIACF_SYSP_CATALOG            1207
 #define MQIACF_SYSP_COMPACT            1208
 #define MQIACF_SYSP_ALLOC_PRIMARY      1209
 #define MQIACF_SYSP_ALLOC_SECONDARY    1210
 #define MQIACF_SYSP_PROTECT            1211
 #define MQIACF_SYSP_QUIESCE_INTERVAL   1212
 #define MQIACF_SYSP_TIMESTAMP          1213
 #define MQIACF_SYSP_UNIT_ADDRESS       1214
 #define MQIACF_SYSP_UNIT_STATUS        1215
 #define MQIACF_SYSP_LOG_COPY           1216
 #define MQIACF_SYSP_LOG_USED           1217
 #define MQIACF_SYSP_LOG_SUSPEND        1218
 #define MQIACF_SYSP_OFFLOAD_STATUS     1219
 #define MQIACF_SYSP_TOTAL_LOGS         1220
 #define MQIACF_SYSP_FULL_LOGS          1221
 #define MQIACF_LISTENER_ATTRS          1222
 #define MQIACF_LISTENER_STATUS_ATTRS   1223
 #define MQIACF_SERVICE_ATTRS           1224
 #define MQIACF_SERVICE_STATUS_ATTRS    1225
 #define MQIACF_Q_TIME_INDICATOR        1226
 #define MQIACF_OLDEST_MSG_AGE          1227
 #define MQIACF_AUTH_OPTIONS            1228
 #define MQIACF_Q_MGR_STATUS_ATTRS      1229
 #define MQIACF_CONNECTION_COUNT        1230
 #define MQIACF_Q_MGR_FACILITY          1231
 #define MQIACF_CHINIT_STATUS           1232
 #define MQIACF_CMD_SERVER_STATUS       1233
 #define MQIACF_ROUTE_DETAIL            1234
 #define MQIACF_RECORDED_ACTIVITIES     1235
 #define MQIACF_MAX_ACTIVITIES          1236
 #define MQIACF_DISCONTINUITY_COUNT     1237
 #define MQIACF_ROUTE_ACCUMULATION      1238
 #define MQIACF_ROUTE_DELIVERY          1239
 #define MQIACF_OPERATION_TYPE          1240
 #define MQIACF_BACKOUT_COUNT           1241
 #define MQIACF_COMP_CODE               1242
 #define MQIACF_ENCODING                1243
 #define MQIACF_EXPIRY                  1244
 #define MQIACF_FEEDBACK                1245
 #define MQIACF_MSG_FLAGS               1247
 #define MQIACF_MSG_LENGTH              1248
 #define MQIACF_MSG_TYPE                1249
 #define MQIACF_OFFSET                  1250
 #define MQIACF_ORIGINAL_LENGTH         1251
 #define MQIACF_PERSISTENCE             1252
 #define MQIACF_PRIORITY                1253
 #define MQIACF_REASON_CODE             1254
 #define MQIACF_REPORT                  1255
 #define MQIACF_VERSION                 1256
 #define MQIACF_UNRECORDED_ACTIVITIES   1257
 #define MQIACF_MONITORING              1258
 #define MQIACF_ROUTE_FORWARDING        1259
 #define MQIACF_SERVICE_STATUS          1260
 #define MQIACF_Q_TYPES                 1261
 #define MQIACF_USER_ID_SUPPORT         1262
 #define MQIACF_INTERFACE_VERSION       1263
 #define MQIACF_AUTH_SERVICE_ATTRS      1264
 #define MQIACF_USAGE_EXPAND_TYPE       1265
 #define MQIACF_SYSP_CLUSTER_CACHE      1266
 #define MQIACF_SYSP_DB2_BLOB_TASKS     1267
 #define MQIACF_SYSP_WLM_INT_UNITS      1268
 #define MQIACF_TOPIC_ATTRS             1269
 #define MQIACF_PUBSUB_PROPERTIES       1271
 #define MQIACF_DESTINATION_CLASS       1273
 #define MQIACF_DURABLE_SUBSCRIPTION    1274
 #define MQIACF_SUBSCRIPTION_SCOPE      1275
 #define MQIACF_VARIABLE_USER_ID        1277
 #define MQIACF_REQUEST_ONLY            1280
 #define MQIACF_PUB_PRIORITY            1283
 #define MQIACF_SUB_ATTRS               1287
 #define MQIACF_WILDCARD_SCHEMA         1288
 #define MQIACF_SUB_TYPE                1289
 #define MQIACF_MESSAGE_COUNT           1290
 #define MQIACF_Q_MGR_PUBSUB            1291
 #define MQIACF_Q_MGR_VERSION           1292
 #define MQIACF_SUB_STATUS_ATTRS        1294
 #define MQIACF_TOPIC_STATUS            1295
 #define MQIACF_TOPIC_SUB               1296
 #define MQIACF_TOPIC_PUB               1297
 #define MQIACF_RETAINED_PUBLICATION    1300
 #define MQIACF_TOPIC_STATUS_ATTRS      1301
 #define MQIACF_TOPIC_STATUS_TYPE       1302
 #define MQIACF_SUB_OPTIONS             1303
 #define MQIACF_PUBLISH_COUNT           1304
 #define MQIACF_CLEAR_TYPE              1305
 #define MQIACF_CLEAR_SCOPE             1306
 #define MQIACF_SUB_LEVEL               1307
 #define MQIACF_ASYNC_STATE             1308
 #define MQIACF_SUB_SUMMARY             1309
 #define MQIACF_OBSOLETE_MSGS           1310
 #define MQIACF_PUBSUB_STATUS           1311
 #define MQIACF_PS_STATUS_TYPE          1314
 #define MQIACF_PUBSUB_STATUS_ATTRS     1318
 #define MQIACF_SELECTOR_TYPE           1321
 #define MQIACF_LOG_COMPRESSION         1322
 #define MQIACF_GROUPUR_CHECK_ID        1323
 #define MQIACF_MULC_CAPTURE            1324
 #define MQIACF_PERMIT_STANDBY          1325
 #define MQIACF_OPERATION_MODE          1326
 #define MQIACF_COMM_INFO_ATTRS         1327
 #define MQIACF_CF_SMDS_BLOCK_SIZE      1328
 #define MQIACF_CF_SMDS_EXPAND          1329
 #define MQIACF_USAGE_FREE_BUFF         1330
 #define MQIACF_USAGE_FREE_BUFF_PERC    1331
 #define MQIACF_CF_STRUC_ACCESS         1332
 #define MQIACF_CF_STATUS_SMDS          1333
 #define MQIACF_SMDS_ATTRS              1334
 #define MQIACF_USAGE_SMDS              1335
 #define MQIACF_USAGE_BLOCK_SIZE        1336
 #define MQIACF_USAGE_DATA_BLOCKS       1337
 #define MQIACF_USAGE_EMPTY_BUFFERS     1338
 #define MQIACF_USAGE_INUSE_BUFFERS     1339
 #define MQIACF_USAGE_LOWEST_FREE       1340
 #define MQIACF_USAGE_OFFLOAD_MSGS      1341
 #define MQIACF_USAGE_READS_SAVED       1342
 #define MQIACF_USAGE_SAVED_BUFFERS     1343
 #define MQIACF_USAGE_TOTAL_BLOCKS      1344
 #define MQIACF_USAGE_USED_BLOCKS       1345
 #define MQIACF_USAGE_USED_RATE         1346
 #define MQIACF_USAGE_WAIT_RATE         1347
 #define MQIACF_SMDS_OPENMODE           1348
 #define MQIACF_SMDS_STATUS             1349
 #define MQIACF_SMDS_AVAIL              1350
 #define MQIACF_MCAST_REL_INDICATOR     1351
 #define MQIACF_CHLAUTH_TYPE            1352
 #define MQIACF_MQXR_DIAGNOSTICS_TYPE   1354
 #define MQIACF_CHLAUTH_ATTRS           1355
 #define MQIACF_OPERATION_ID            1356
 #define MQIACF_API_CALLER_TYPE         1357
 #define MQIACF_API_ENVIRONMENT         1358
 #define MQIACF_TRACE_DETAIL            1359
 #define MQIACF_HOBJ                    1360
 #define MQIACF_CALL_TYPE               1361
 #define MQIACF_MQCB_OPERATION          1362
 #define MQIACF_MQCB_TYPE               1363
 #define MQIACF_MQCB_OPTIONS            1364
 #define MQIACF_CLOSE_OPTIONS           1365
 #define MQIACF_CTL_OPERATION           1366
 #define MQIACF_GET_OPTIONS             1367
 #define MQIACF_RECS_PRESENT            1368
 #define MQIACF_KNOWN_DEST_COUNT        1369
 #define MQIACF_UNKNOWN_DEST_COUNT      1370
 #define MQIACF_INVALID_DEST_COUNT      1371
 #define MQIACF_RESOLVED_TYPE           1372
 #define MQIACF_PUT_OPTIONS             1373
 #define MQIACF_BUFFER_LENGTH           1374
 #define MQIACF_TRACE_DATA_LENGTH       1375
 #define MQIACF_SMDS_EXPANDST           1376
 #define MQIACF_STRUC_LENGTH            1377
 #define MQIACF_ITEM_COUNT              1378
 #define MQIACF_EXPIRY_TIME             1379
 #define MQIACF_CONNECT_TIME            1380
 #define MQIACF_DISCONNECT_TIME         1381
 #define MQIACF_HSUB                    1382
 #define MQIACF_SUBRQ_OPTIONS           1383
 #define MQIACF_XA_RMID                 1384
 #define MQIACF_XA_FLAGS                1385
 #define MQIACF_XA_RETCODE              1386
 #define MQIACF_XA_HANDLE               1387
 #define MQIACF_XA_RETVAL               1388
 #define MQIACF_STATUS_TYPE             1389
 #define MQIACF_XA_COUNT                1390
 #define MQIACF_SELECTOR_COUNT          1391
 #define MQIACF_SELECTORS               1392
 #define MQIACF_INTATTR_COUNT           1393
 #define MQIACF_INT_ATTRS               1394
 #define MQIACF_SUBRQ_ACTION            1395
 #define MQIACF_NUM_PUBS                1396
 #define MQIACF_POINTER_SIZE            1397
 #define MQIACF_REMOVE_AUTHREC          1398
 #define MQIACF_XR_ATTRS                1399
 #define MQIACF_APPL_FUNCTION_TYPE      1400
 #define MQIACF_AMQP_ATTRS              1401
 #define MQIACF_EXPORT_TYPE             1402
 #define MQIACF_EXPORT_ATTRS            1403
 #define MQIACF_SYSTEM_OBJECTS          1404
 #define MQIACF_CONNECTION_SWAP         1405
 #define MQIACF_AMQP_DIAGNOSTICS_TYPE   1406
 #define MQIACF_BUFFER_POOL_LOCATION    1408
 #define MQIACF_LDAP_CONNECTION_STATUS  1409
 #define MQIACF_SYSP_MAX_ACE_POOL       1410
 #define MQIACF_PAGECLAS                1411
 #define MQIACF_AUTH_REC_TYPE           1412
 #define MQIACF_SYSP_MAX_CONC_OFFLOADS  1413
 #define MQIACF_SYSP_ZHYPERWRITE        1414
 #define MQIACF_Q_MGR_STATUS_LOG        1415
 #define MQIACF_ARCHIVE_LOG_SIZE        1416
 #define MQIACF_MEDIA_LOG_SIZE          1417
 #define MQIACF_RESTART_LOG_SIZE        1418
 #define MQIACF_REUSABLE_LOG_SIZE       1419
 #define MQIACF_LOG_IN_USE              1420
 #define MQIACF_LOG_UTILIZATION         1421
 #define MQIACF_LOG_REDUCTION           1422
 #define MQIACF_IGNORE_STATE            1423
 #define MQIACF_LAST_USED               1423

 /* Access Options */
 #define MQCFACCESS_ENABLED             0
 #define MQCFACCESS_SUSPENDED           1
 #define MQCFACCESS_DISABLED            2

 /* Open Mode Options */
 #define MQS_OPENMODE_NONE              0
 #define MQS_OPENMODE_READONLY          1
 #define MQS_OPENMODE_UPDATE            2
 #define MQS_OPENMODE_RECOVERY          3

 /* SMDS Status Options */
 #define MQS_STATUS_CLOSED              0
 #define MQS_STATUS_CLOSING             1
 #define MQS_STATUS_OPENING             2
 #define MQS_STATUS_OPEN                3
 #define MQS_STATUS_NOTENABLED          4
 #define MQS_STATUS_ALLOCFAIL           5
 #define MQS_STATUS_OPENFAIL            6
 #define MQS_STATUS_STGFAIL             7
 #define MQS_STATUS_DATAFAIL            8

 /* SMDS Availability Options */
 #define MQS_AVAIL_NORMAL               0
 #define MQS_AVAIL_ERROR                1
 #define MQS_AVAIL_STOPPED              2

 /* Values for MQIACF_BUFFER_POOL_LOCATION. */
 #define MQBPLOCATION_BELOW             0
 #define MQBPLOCATION_ABOVE             1
 #define MQBPLOCATION_SWITCHING_ABOVE   2
 #define MQBPLOCATION_SWITCHING_BELOW   3

 /* Values for MQIACF_PAGECLAS. */
 #define MQPAGECLAS_4KB                 0
 #define MQPAGECLAS_FIXED4KB            1

 /* Expandst Options */
 #define MQS_EXPANDST_NORMAL            0
 #define MQS_EXPANDST_FAILED            1
 #define MQS_EXPANDST_MAXIMUM           2

 /* Usage SMDS Options */
 #define MQUSAGE_SMDS_AVAILABLE         0
 #define MQUSAGE_SMDS_NO_DATA           1

 /* Integer Channel Types */
 #define MQIACH_FIRST                   1501
 #define MQIACH_XMIT_PROTOCOL_TYPE      1501
 #define MQIACH_BATCH_SIZE              1502
 #define MQIACH_DISC_INTERVAL           1503
 #define MQIACH_SHORT_TIMER             1504
 #define MQIACH_SHORT_RETRY             1505
 #define MQIACH_LONG_TIMER              1506
 #define MQIACH_LONG_RETRY              1507
 #define MQIACH_PUT_AUTHORITY           1508
 #define MQIACH_SEQUENCE_NUMBER_WRAP    1509
 #define MQIACH_MAX_MSG_LENGTH          1510
 #define MQIACH_CHANNEL_TYPE            1511
 #define MQIACH_DATA_COUNT              1512
 #define MQIACH_NAME_COUNT              1513
 #define MQIACH_MSG_SEQUENCE_NUMBER     1514
 #define MQIACH_DATA_CONVERSION         1515
 #define MQIACH_IN_DOUBT                1516
 #define MQIACH_MCA_TYPE                1517
 #define MQIACH_SESSION_COUNT           1518
 #define MQIACH_ADAPTER                 1519
 #define MQIACH_COMMAND_COUNT           1520
 #define MQIACH_SOCKET                  1521
 #define MQIACH_PORT                    1522
 #define MQIACH_CHANNEL_INSTANCE_TYPE   1523
 #define MQIACH_CHANNEL_INSTANCE_ATTRS  1524
 #define MQIACH_CHANNEL_ERROR_DATA      1525
 #define MQIACH_CHANNEL_TABLE           1526
 #define MQIACH_CHANNEL_STATUS          1527
 #define MQIACH_INDOUBT_STATUS          1528
 #define MQIACH_LAST_SEQ_NUMBER         1529
 #define MQIACH_LAST_SEQUENCE_NUMBER    1529
 #define MQIACH_CURRENT_MSGS            1531
 #define MQIACH_CURRENT_SEQ_NUMBER      1532
 #define MQIACH_CURRENT_SEQUENCE_NUMBER 1532
 #define MQIACH_SSL_RETURN_CODE         1533
 #define MQIACH_MSGS                    1534
 #define MQIACH_BYTES_SENT              1535
 #define MQIACH_BYTES_RCVD              1536
 #define MQIACH_BYTES_RECEIVED          1536
 #define MQIACH_BATCHES                 1537
 #define MQIACH_BUFFERS_SENT            1538
 #define MQIACH_BUFFERS_RCVD            1539
 #define MQIACH_BUFFERS_RECEIVED        1539
 #define MQIACH_LONG_RETRIES_LEFT       1540
 #define MQIACH_SHORT_RETRIES_LEFT      1541
 #define MQIACH_MCA_STATUS              1542
 #define MQIACH_STOP_REQUESTED          1543
 #define MQIACH_MR_COUNT                1544
 #define MQIACH_MR_INTERVAL             1545
 #define MQIACH_NPM_SPEED               1562
 #define MQIACH_HB_INTERVAL             1563
 #define MQIACH_BATCH_INTERVAL          1564
 #define MQIACH_NETWORK_PRIORITY        1565
 #define MQIACH_KEEP_ALIVE_INTERVAL     1566
 #define MQIACH_BATCH_HB                1567
 #define MQIACH_SSL_CLIENT_AUTH         1568
 #define MQIACH_ALLOC_RETRY             1570
 #define MQIACH_ALLOC_FAST_TIMER        1571
 #define MQIACH_ALLOC_SLOW_TIMER        1572
 #define MQIACH_DISC_RETRY              1573
 #define MQIACH_PORT_NUMBER             1574
 #define MQIACH_HDR_COMPRESSION         1575
 #define MQIACH_MSG_COMPRESSION         1576
 #define MQIACH_CLWL_CHANNEL_RANK       1577
 #define MQIACH_CLWL_CHANNEL_PRIORITY   1578
 #define MQIACH_CLWL_CHANNEL_WEIGHT     1579
 #define MQIACH_CHANNEL_DISP            1580
 #define MQIACH_INBOUND_DISP            1581
 #define MQIACH_CHANNEL_TYPES           1582
 #define MQIACH_ADAPS_STARTED           1583
 #define MQIACH_ADAPS_MAX               1584
 #define MQIACH_DISPS_STARTED           1585
 #define MQIACH_DISPS_MAX               1586
 #define MQIACH_SSLTASKS_STARTED        1587
 #define MQIACH_SSLTASKS_MAX            1588
 #define MQIACH_CURRENT_CHL             1589
 #define MQIACH_CURRENT_CHL_MAX         1590
 #define MQIACH_CURRENT_CHL_TCP         1591
 #define MQIACH_CURRENT_CHL_LU62        1592
 #define MQIACH_ACTIVE_CHL              1593
 #define MQIACH_ACTIVE_CHL_MAX          1594
 #define MQIACH_ACTIVE_CHL_PAUSED       1595
 #define MQIACH_ACTIVE_CHL_STARTED      1596
 #define MQIACH_ACTIVE_CHL_STOPPED      1597
 #define MQIACH_ACTIVE_CHL_RETRY        1598
 #define MQIACH_LISTENER_STATUS         1599
 #define MQIACH_SHARED_CHL_RESTART      1600
 #define MQIACH_LISTENER_CONTROL        1601
 #define MQIACH_BACKLOG                 1602
 #define MQIACH_XMITQ_TIME_INDICATOR    1604
 #define MQIACH_NETWORK_TIME_INDICATOR  1605
 #define MQIACH_EXIT_TIME_INDICATOR     1606
 #define MQIACH_BATCH_SIZE_INDICATOR    1607
 #define MQIACH_XMITQ_MSGS_AVAILABLE    1608
 #define MQIACH_CHANNEL_SUBSTATE        1609
 #define MQIACH_SSL_KEY_RESETS          1610
 #define MQIACH_COMPRESSION_RATE        1611
 #define MQIACH_COMPRESSION_TIME        1612
 #define MQIACH_MAX_XMIT_SIZE           1613
 #define MQIACH_DEF_CHANNEL_DISP        1614
 #define MQIACH_SHARING_CONVERSATIONS   1615
 #define MQIACH_MAX_SHARING_CONVS       1616
 #define MQIACH_CURRENT_SHARING_CONVS   1617
 #define MQIACH_MAX_INSTANCES           1618
 #define MQIACH_MAX_INSTS_PER_CLIENT    1619
 #define MQIACH_CLIENT_CHANNEL_WEIGHT   1620
 #define MQIACH_CONNECTION_AFFINITY     1621
 #define MQIACH_RESET_REQUESTED         1623
 #define MQIACH_BATCH_DATA_LIMIT        1624
 #define MQIACH_MSG_HISTORY             1625
 #define MQIACH_MULTICAST_PROPERTIES    1626
 #define MQIACH_NEW_SUBSCRIBER_HISTORY  1627
 #define MQIACH_MC_HB_INTERVAL          1628
 #define MQIACH_USE_CLIENT_ID           1629
 #define MQIACH_MQTT_KEEP_ALIVE         1630
 #define MQIACH_IN_DOUBT_IN             1631
 #define MQIACH_IN_DOUBT_OUT            1632
 #define MQIACH_MSGS_SENT               1633
 #define MQIACH_MSGS_RECEIVED           1634
 #define MQIACH_MSGS_RCVD               1634
 #define MQIACH_PENDING_OUT             1635
 #define MQIACH_AVAILABLE_CIPHERSPECS   1636
 #define MQIACH_MATCH                   1637
 #define MQIACH_USER_SOURCE             1638
 #define MQIACH_WARNING                 1639
 #define MQIACH_DEF_RECONNECT           1640
 #define MQIACH_CHANNEL_SUMMARY_ATTRS   1642
 #define MQIACH_PROTOCOL                1643
 #define MQIACH_AMQP_KEEP_ALIVE         1644
 #define MQIACH_SECURITY_PROTOCOL       1645
 #define MQIACH_LAST_USED               1645

 /****************************************************************/
 /* Values Related to Character Parameter Structures             */
 /****************************************************************/

 /* Character Monitoring Parameter Types */
 #define MQCAMO_FIRST                   2701
 #define MQCAMO_CLOSE_DATE              2701
 #define MQCAMO_CLOSE_TIME              2702
 #define MQCAMO_CONN_DATE               2703
 #define MQCAMO_CONN_TIME               2704
 #define MQCAMO_DISC_DATE               2705
 #define MQCAMO_DISC_TIME               2706
 #define MQCAMO_END_DATE                2707
 #define MQCAMO_END_TIME                2708
 #define MQCAMO_OPEN_DATE               2709
 #define MQCAMO_OPEN_TIME               2710
 #define MQCAMO_START_DATE              2711
 #define MQCAMO_START_TIME              2712
 #define MQCAMO_MONITOR_CLASS           2713
 #define MQCAMO_MONITOR_TYPE            2714
 #define MQCAMO_MONITOR_DESC            2715
 #define MQCAMO_LAST_USED               2715

 /* Character Parameter Types */
 #define MQCACF_FIRST                   3001
 #define MQCACF_FROM_Q_NAME             3001
 #define MQCACF_TO_Q_NAME               3002
 #define MQCACF_FROM_PROCESS_NAME       3003
 #define MQCACF_TO_PROCESS_NAME         3004
 #define MQCACF_FROM_NAMELIST_NAME      3005
 #define MQCACF_TO_NAMELIST_NAME        3006
 #define MQCACF_FROM_CHANNEL_NAME       3007
 #define MQCACF_TO_CHANNEL_NAME         3008
 #define MQCACF_FROM_AUTH_INFO_NAME     3009
 #define MQCACF_TO_AUTH_INFO_NAME       3010
 #define MQCACF_Q_NAMES                 3011
 #define MQCACF_PROCESS_NAMES           3012
 #define MQCACF_NAMELIST_NAMES          3013
 #define MQCACF_ESCAPE_TEXT             3014
 #define MQCACF_LOCAL_Q_NAMES           3015
 #define MQCACF_MODEL_Q_NAMES           3016
 #define MQCACF_ALIAS_Q_NAMES           3017
 #define MQCACF_REMOTE_Q_NAMES          3018
 #define MQCACF_SENDER_CHANNEL_NAMES    3019
 #define MQCACF_SERVER_CHANNEL_NAMES    3020
 #define MQCACF_REQUESTER_CHANNEL_NAMES 3021
 #define MQCACF_RECEIVER_CHANNEL_NAMES  3022
 #define MQCACF_OBJECT_Q_MGR_NAME       3023
 #define MQCACF_APPL_NAME               3024
 #define MQCACF_USER_IDENTIFIER         3025
 #define MQCACF_AUX_ERROR_DATA_STR_1    3026
 #define MQCACF_AUX_ERROR_DATA_STR_2    3027
 #define MQCACF_AUX_ERROR_DATA_STR_3    3028
 #define MQCACF_BRIDGE_NAME             3029
 #define MQCACF_STREAM_NAME             3030
 #define MQCACF_TOPIC                   3031
 #define MQCACF_PARENT_Q_MGR_NAME       3032
 #define MQCACF_CORREL_ID               3033
 #define MQCACF_PUBLISH_TIMESTAMP       3034
 #define MQCACF_STRING_DATA             3035
 #define MQCACF_SUPPORTED_STREAM_NAME   3036
 #define MQCACF_REG_TOPIC               3037
 #define MQCACF_REG_TIME                3038
 #define MQCACF_REG_USER_ID             3039
 #define MQCACF_CHILD_Q_MGR_NAME        3040
 #define MQCACF_REG_STREAM_NAME         3041
 #define MQCACF_REG_Q_MGR_NAME          3042
 #define MQCACF_REG_Q_NAME              3043
 #define MQCACF_REG_CORREL_ID           3044
 #define MQCACF_EVENT_USER_ID           3045
 #define MQCACF_OBJECT_NAME             3046
 #define MQCACF_EVENT_Q_MGR             3047
 #define MQCACF_AUTH_INFO_NAMES         3048
 #define MQCACF_EVENT_APPL_IDENTITY     3049
 #define MQCACF_EVENT_APPL_NAME         3050
 #define MQCACF_EVENT_APPL_ORIGIN       3051
 #define MQCACF_SUBSCRIPTION_NAME       3052
 #define MQCACF_REG_SUB_NAME            3053
 #define MQCACF_SUBSCRIPTION_IDENTITY   3054
 #define MQCACF_REG_SUB_IDENTITY        3055
 #define MQCACF_SUBSCRIPTION_USER_DATA  3056
 #define MQCACF_REG_SUB_USER_DATA       3057
 #define MQCACF_APPL_TAG                3058
 #define MQCACF_DATA_SET_NAME           3059
 #define MQCACF_UOW_START_DATE          3060
 #define MQCACF_UOW_START_TIME          3061
 #define MQCACF_UOW_LOG_START_DATE      3062
 #define MQCACF_UOW_LOG_START_TIME      3063
 #define MQCACF_UOW_LOG_EXTENT_NAME     3064
 #define MQCACF_PRINCIPAL_ENTITY_NAMES  3065
 #define MQCACF_GROUP_ENTITY_NAMES      3066
 #define MQCACF_AUTH_PROFILE_NAME       3067
 #define MQCACF_ENTITY_NAME             3068
 #define MQCACF_SERVICE_COMPONENT       3069
 #define MQCACF_RESPONSE_Q_MGR_NAME     3070
 #define MQCACF_CURRENT_LOG_EXTENT_NAME 3071
 #define MQCACF_RESTART_LOG_EXTENT_NAME 3072
 #define MQCACF_MEDIA_LOG_EXTENT_NAME   3073
 #define MQCACF_LOG_PATH                3074
 #define MQCACF_COMMAND_MQSC            3075
 #define MQCACF_Q_MGR_CPF               3076
 #define MQCACF_USAGE_LOG_RBA           3078
 #define MQCACF_USAGE_LOG_LRSN          3079
 #define MQCACF_COMMAND_SCOPE           3080
 #define MQCACF_ASID                    3081
 #define MQCACF_PSB_NAME                3082
 #define MQCACF_PST_ID                  3083
 #define MQCACF_TASK_NUMBER             3084
 #define MQCACF_TRANSACTION_ID          3085
 #define MQCACF_Q_MGR_UOW_ID            3086
 #define MQCACF_ORIGIN_NAME             3088
 #define MQCACF_ENV_INFO                3089
 #define MQCACF_SECURITY_PROFILE        3090
 #define MQCACF_CONFIGURATION_DATE      3091
 #define MQCACF_CONFIGURATION_TIME      3092
 #define MQCACF_FROM_CF_STRUC_NAME      3093
 #define MQCACF_TO_CF_STRUC_NAME        3094
 #define MQCACF_CF_STRUC_NAMES          3095
 #define MQCACF_FAIL_DATE               3096
 #define MQCACF_FAIL_TIME               3097
 #define MQCACF_BACKUP_DATE             3098
 #define MQCACF_BACKUP_TIME             3099
 #define MQCACF_SYSTEM_NAME             3100
 #define MQCACF_CF_STRUC_BACKUP_START   3101
 #define MQCACF_CF_STRUC_BACKUP_END     3102
 #define MQCACF_CF_STRUC_LOG_Q_MGRS     3103
 #define MQCACF_FROM_STORAGE_CLASS      3104
 #define MQCACF_TO_STORAGE_CLASS        3105
 #define MQCACF_STORAGE_CLASS_NAMES     3106
 #define MQCACF_DSG_NAME                3108
 #define MQCACF_DB2_NAME                3109
 #define MQCACF_SYSP_CMD_USER_ID        3110
 #define MQCACF_SYSP_OTMA_GROUP         3111
 #define MQCACF_SYSP_OTMA_MEMBER        3112
 #define MQCACF_SYSP_OTMA_DRU_EXIT      3113
 #define MQCACF_SYSP_OTMA_TPIPE_PFX     3114
 #define MQCACF_SYSP_ARCHIVE_PFX1       3115
 #define MQCACF_SYSP_ARCHIVE_UNIT1      3116
 #define MQCACF_SYSP_LOG_CORREL_ID      3117
 #define MQCACF_SYSP_UNIT_VOLSER        3118
 #define MQCACF_SYSP_Q_MGR_TIME         3119
 #define MQCACF_SYSP_Q_MGR_DATE         3120
 #define MQCACF_SYSP_Q_MGR_RBA          3121
 #define MQCACF_SYSP_LOG_RBA            3122
 #define MQCACF_SYSP_SERVICE            3123
 #define MQCACF_FROM_LISTENER_NAME      3124
 #define MQCACF_TO_LISTENER_NAME        3125
 #define MQCACF_FROM_SERVICE_NAME       3126
 #define MQCACF_TO_SERVICE_NAME         3127
 #define MQCACF_LAST_PUT_DATE           3128
 #define MQCACF_LAST_PUT_TIME           3129
 #define MQCACF_LAST_GET_DATE           3130
 #define MQCACF_LAST_GET_TIME           3131
 #define MQCACF_OPERATION_DATE          3132
 #define MQCACF_OPERATION_TIME          3133
 #define MQCACF_ACTIVITY_DESC           3134
 #define MQCACF_APPL_IDENTITY_DATA      3135
 #define MQCACF_APPL_ORIGIN_DATA        3136
 #define MQCACF_PUT_DATE                3137
 #define MQCACF_PUT_TIME                3138
 #define MQCACF_REPLY_TO_Q              3139
 #define MQCACF_REPLY_TO_Q_MGR          3140
 #define MQCACF_RESOLVED_Q_NAME         3141
 #define MQCACF_STRUC_ID                3142
 #define MQCACF_VALUE_NAME              3143
 #define MQCACF_SERVICE_START_DATE      3144
 #define MQCACF_SERVICE_START_TIME      3145
 #define MQCACF_SYSP_OFFLINE_RBA        3146
 #define MQCACF_SYSP_ARCHIVE_PFX2       3147
 #define MQCACF_SYSP_ARCHIVE_UNIT2      3148
 #define MQCACF_TO_TOPIC_NAME           3149
 #define MQCACF_FROM_TOPIC_NAME         3150
 #define MQCACF_TOPIC_NAMES             3151
 #define MQCACF_SUB_NAME                3152
 #define MQCACF_DESTINATION_Q_MGR       3153
 #define MQCACF_DESTINATION             3154
 #define MQCACF_SUB_USER_ID             3156
 #define MQCACF_SUB_USER_DATA           3159
 #define MQCACF_SUB_SELECTOR            3160
 #define MQCACF_LAST_PUB_DATE           3161
 #define MQCACF_LAST_PUB_TIME           3162
 #define MQCACF_FROM_SUB_NAME           3163
 #define MQCACF_TO_SUB_NAME             3164
 #define MQCACF_LAST_MSG_TIME           3167
 #define MQCACF_LAST_MSG_DATE           3168
 #define MQCACF_SUBSCRIPTION_POINT      3169
 #define MQCACF_FILTER                  3170
 #define MQCACF_NONE                    3171
 #define MQCACF_ADMIN_TOPIC_NAMES       3172
 #define MQCACF_ROUTING_FINGER_PRINT    3173
 #define MQCACF_APPL_DESC               3174
 #define MQCACF_Q_MGR_START_DATE        3175
 #define MQCACF_Q_MGR_START_TIME        3176
 #define MQCACF_FROM_COMM_INFO_NAME     3177
 #define MQCACF_TO_COMM_INFO_NAME       3178
 #define MQCACF_CF_OFFLOAD_SIZE1        3179
 #define MQCACF_CF_OFFLOAD_SIZE2        3180
 #define MQCACF_CF_OFFLOAD_SIZE3        3181
 #define MQCACF_CF_SMDS_GENERIC_NAME    3182
 #define MQCACF_CF_SMDS                 3183
 #define MQCACF_RECOVERY_DATE           3184
 #define MQCACF_RECOVERY_TIME           3185
 #define MQCACF_CF_SMDSCONN             3186
 #define MQCACF_CF_STRUC_NAME           3187
 #define MQCACF_ALTERNATE_USERID        3188
 #define MQCACF_CHAR_ATTRS              3189
 #define MQCACF_DYNAMIC_Q_NAME          3190
 #define MQCACF_HOST_NAME               3191
 #define MQCACF_MQCB_NAME               3192
 #define MQCACF_OBJECT_STRING           3193
 #define MQCACF_RESOLVED_LOCAL_Q_MGR    3194
 #define MQCACF_RESOLVED_LOCAL_Q_NAME   3195
 #define MQCACF_RESOLVED_OBJECT_STRING  3196
 #define MQCACF_RESOLVED_Q_MGR          3197
 #define MQCACF_SELECTION_STRING        3198
 #define MQCACF_XA_INFO                 3199
 #define MQCACF_APPL_FUNCTION           3200
 #define MQCACF_XQH_REMOTE_Q_NAME       3201
 #define MQCACF_XQH_REMOTE_Q_MGR        3202
 #define MQCACF_XQH_PUT_TIME            3203
 #define MQCACF_XQH_PUT_DATE            3204
 #define MQCACF_EXCL_OPERATOR_MESSAGES  3205
 #define MQCACF_CSP_USER_IDENTIFIER     3206
 #define MQCACF_AMQP_CLIENT_ID          3207
 #define MQCACF_ARCHIVE_LOG_EXTENT_NAME 3208
 #define MQCACF_LAST_USED               3208

 /* Character Channel Parameter Types */
 #define MQCACH_FIRST                   3501
 #define MQCACH_CHANNEL_NAME            3501
 #define MQCACH_DESC                    3502
 #define MQCACH_MODE_NAME               3503
 #define MQCACH_TP_NAME                 3504
 #define MQCACH_XMIT_Q_NAME             3505
 #define MQCACH_CONNECTION_NAME         3506
 #define MQCACH_MCA_NAME                3507
 #define MQCACH_SEC_EXIT_NAME           3508
 #define MQCACH_MSG_EXIT_NAME           3509
 #define MQCACH_SEND_EXIT_NAME          3510
 #define MQCACH_RCV_EXIT_NAME           3511
 #define MQCACH_CHANNEL_NAMES           3512
 #define MQCACH_SEC_EXIT_USER_DATA      3513
 #define MQCACH_MSG_EXIT_USER_DATA      3514
 #define MQCACH_SEND_EXIT_USER_DATA     3515
 #define MQCACH_RCV_EXIT_USER_DATA      3516
 #define MQCACH_USER_ID                 3517
 #define MQCACH_PASSWORD                3518
 #define MQCACH_LOCAL_ADDRESS           3520
 #define MQCACH_LOCAL_NAME              3521
 #define MQCACH_LAST_MSG_TIME           3524
 #define MQCACH_LAST_MSG_DATE           3525
 #define MQCACH_MCA_USER_ID             3527
 #define MQCACH_CHANNEL_START_TIME      3528
 #define MQCACH_CHANNEL_START_DATE      3529
 #define MQCACH_MCA_JOB_NAME            3530
 #define MQCACH_LAST_LUWID              3531
 #define MQCACH_CURRENT_LUWID           3532
 #define MQCACH_FORMAT_NAME             3533
 #define MQCACH_MR_EXIT_NAME            3534
 #define MQCACH_MR_EXIT_USER_DATA       3535
 #define MQCACH_SSL_CIPHER_SPEC         3544
 #define MQCACH_SSL_PEER_NAME           3545
 #define MQCACH_SSL_HANDSHAKE_STAGE     3546
 #define MQCACH_SSL_SHORT_PEER_NAME     3547
 #define MQCACH_REMOTE_APPL_TAG         3548
 #define MQCACH_SSL_CERT_USER_ID        3549
 #define MQCACH_SSL_CERT_ISSUER_NAME    3550
 #define MQCACH_LU_NAME                 3551
 #define MQCACH_IP_ADDRESS              3552
 #define MQCACH_TCP_NAME                3553
 #define MQCACH_LISTENER_NAME           3554
 #define MQCACH_LISTENER_DESC           3555
 #define MQCACH_LISTENER_START_DATE     3556
 #define MQCACH_LISTENER_START_TIME     3557
 #define MQCACH_SSL_KEY_RESET_DATE      3558
 #define MQCACH_SSL_KEY_RESET_TIME      3559
 #define MQCACH_REMOTE_VERSION          3560
 #define MQCACH_REMOTE_PRODUCT          3561
 #define MQCACH_GROUP_ADDRESS           3562
 #define MQCACH_JAAS_CONFIG             3563
 #define MQCACH_CLIENT_ID               3564
 #define MQCACH_SSL_KEY_PASSPHRASE      3565
 #define MQCACH_CONNECTION_NAME_LIST    3566
 #define MQCACH_CLIENT_USER_ID          3567
 #define MQCACH_MCA_USER_ID_LIST        3568
 #define MQCACH_SSL_CIPHER_SUITE        3569
 #define MQCACH_WEBCONTENT_PATH         3570
 #define MQCACH_TOPIC_ROOT              3571
 #define MQCACH_LAST_USED               3571

 /****************************************************************/
 /* Values Related to Group Parameter Structures                 */
 /****************************************************************/

 /* Group Parameter Types */
 #define MQGACF_FIRST                   8001
 #define MQGACF_COMMAND_CONTEXT         8001
 #define MQGACF_COMMAND_DATA            8002
 #define MQGACF_TRACE_ROUTE             8003
 #define MQGACF_OPERATION               8004
 #define MQGACF_ACTIVITY                8005
 #define MQGACF_EMBEDDED_MQMD           8006
 #define MQGACF_MESSAGE                 8007
 #define MQGACF_MQMD                    8008
 #define MQGACF_VALUE_NAMING            8009
 #define MQGACF_Q_ACCOUNTING_DATA       8010
 #define MQGACF_Q_STATISTICS_DATA       8011
 #define MQGACF_CHL_STATISTICS_DATA     8012
 #define MQGACF_ACTIVITY_TRACE          8013
 #define MQGACF_APP_DIST_LIST           8014
 #define MQGACF_MONITOR_CLASS           8015
 #define MQGACF_MONITOR_TYPE            8016
 #define MQGACF_MONITOR_ELEMENT         8017
 #define MQGACF_LAST_USED               8017

 /****************************************************************/
 /* Parameter Values                                             */
 /****************************************************************/

 /* Action Options */
 #define MQACT_FORCE_REMOVE             1
 #define MQACT_ADVANCE_LOG              2
 #define MQACT_COLLECT_STATISTICS       3
 #define MQACT_PUBSUB                   4
 #define MQACT_ADD                      5
 #define MQACT_REPLACE                  6
 #define MQACT_REMOVE                   7
 #define MQACT_REMOVEALL                8
 #define MQACT_FAIL                     9
 #define MQACT_REDUCE_LOG               10
 #define MQACT_ARCHIVE_LOG              11

 /* State Options */
 #define MQIS_NO                        0
 #define MQIS_YES                       1

 /* Asynchronous State Values */
 #define MQAS_NONE                      0
 #define MQAS_STARTED                   1
 #define MQAS_START_WAIT                2
 #define MQAS_STOPPED                   3
 #define MQAS_SUSPENDED                 4
 #define MQAS_SUSPENDED_TEMPORARY       5
 #define MQAS_ACTIVE                    6
 #define MQAS_INACTIVE                  7

 /* Authority Values */
 #define MQAUTH_NONE                    0
 #define MQAUTH_ALT_USER_AUTHORITY      1
 #define MQAUTH_BROWSE                  2
 #define MQAUTH_CHANGE                  3
 #define MQAUTH_CLEAR                   4
 #define MQAUTH_CONNECT                 5
 #define MQAUTH_CREATE                  6
 #define MQAUTH_DELETE                  7
 #define MQAUTH_DISPLAY                 8
 #define MQAUTH_INPUT                   9
 #define MQAUTH_INQUIRE                 10
 #define MQAUTH_OUTPUT                  11
 #define MQAUTH_PASS_ALL_CONTEXT        12
 #define MQAUTH_PASS_IDENTITY_CONTEXT   13
 #define MQAUTH_SET                     14
 #define MQAUTH_SET_ALL_CONTEXT         15
 #define MQAUTH_SET_IDENTITY_CONTEXT    16
 #define MQAUTH_CONTROL                 17
 #define MQAUTH_CONTROL_EXTENDED        18
 #define MQAUTH_PUBLISH                 19
 #define MQAUTH_SUBSCRIBE               20
 #define MQAUTH_RESUME                  21
 #define MQAUTH_SYSTEM                  22
 #define MQAUTH_ALL                     (-1)
 #define MQAUTH_ALL_ADMIN               (-2)
 #define MQAUTH_ALL_MQI                 (-3)

 /* Authority Options */
 #define MQAUTHOPT_ENTITY_EXPLICIT      0x00000001
 #define MQAUTHOPT_ENTITY_SET           0x00000002
 #define MQAUTHOPT_NAME_EXPLICIT        0x00000010
 #define MQAUTHOPT_NAME_ALL_MATCHING    0x00000020
 #define MQAUTHOPT_NAME_AS_WILDCARD     0x00000040
 #define MQAUTHOPT_CUMULATIVE           0x00000100
 #define MQAUTHOPT_EXCLUDE_TEMP         0x00000200

 /* Bridge Types */
 #define MQBT_OTMA                      1

 /* Refresh Repository Options */
 #define MQCFO_REFRESH_REPOSITORY_YES   1
 #define MQCFO_REFRESH_REPOSITORY_NO    0

 /* Remove Queues Options */
 #define MQCFO_REMOVE_QUEUES_YES        1
 #define MQCFO_REMOVE_QUEUES_NO         0

 /* CHLAUTH Type */
 #define MQCAUT_ALL                     0
 #define MQCAUT_BLOCKUSER               1
 #define MQCAUT_BLOCKADDR               2
 #define MQCAUT_SSLPEERMAP              3
 #define MQCAUT_ADDRESSMAP              4
 #define MQCAUT_USERMAP                 5
 #define MQCAUT_QMGRMAP                 6

 /* CF Status */
 #define MQCFSTATUS_NOT_FOUND           0
 #define MQCFSTATUS_ACTIVE              1
 #define MQCFSTATUS_IN_RECOVER          2
 #define MQCFSTATUS_IN_BACKUP           3
 #define MQCFSTATUS_FAILED              4
 #define MQCFSTATUS_NONE                5
 #define MQCFSTATUS_UNKNOWN             6
 #define MQCFSTATUS_RECOVERED           7
 #define MQCFSTATUS_EMPTY               8
 #define MQCFSTATUS_NEW                 9
 #define MQCFSTATUS_ADMIN_INCOMPLETE    20
 #define MQCFSTATUS_NEVER_USED          21
 #define MQCFSTATUS_NO_BACKUP           22
 #define MQCFSTATUS_NOT_FAILED          23
 #define MQCFSTATUS_NOT_RECOVERABLE     24
 #define MQCFSTATUS_XES_ERROR           25

 /* CF Types */
 #define MQCFTYPE_APPL                  0
 #define MQCFTYPE_ADMIN                 1

 /* Indoubt Status */
 #define MQCHIDS_NOT_INDOUBT            0
 #define MQCHIDS_INDOUBT                1

 /* Channel Dispositions */
 #define MQCHLD_ALL                     (-1)
 #define MQCHLD_DEFAULT                 1
 #define MQCHLD_SHARED                  2
 #define MQCHLD_PRIVATE                 4
 #define MQCHLD_FIXSHARED               5

 /* Use ClientID */
 #define MQUCI_YES                      1
 #define MQUCI_NO                       0

 /* Channel Status */
 #define MQCHS_INACTIVE                 0
 #define MQCHS_BINDING                  1
 #define MQCHS_STARTING                 2
 #define MQCHS_RUNNING                  3
 #define MQCHS_STOPPING                 4
 #define MQCHS_RETRYING                 5
 #define MQCHS_STOPPED                  6
 #define MQCHS_REQUESTING               7
 #define MQCHS_PAUSED                   8
 #define MQCHS_DISCONNECTED             9
 #define MQCHS_INITIALIZING             13
 #define MQCHS_SWITCHING                14

 /* Channel Substates */
 #define MQCHSSTATE_OTHER               0
 #define MQCHSSTATE_END_OF_BATCH        100
 #define MQCHSSTATE_SENDING             200
 #define MQCHSSTATE_RECEIVING           300
 #define MQCHSSTATE_SERIALIZING         400
 #define MQCHSSTATE_RESYNCHING          500
 #define MQCHSSTATE_HEARTBEATING        600
 #define MQCHSSTATE_IN_SCYEXIT          700
 #define MQCHSSTATE_IN_RCVEXIT          800
 #define MQCHSSTATE_IN_SENDEXIT         900
 #define MQCHSSTATE_IN_MSGEXIT          1000
 #define MQCHSSTATE_IN_MREXIT           1100
 #define MQCHSSTATE_IN_CHADEXIT         1200
 #define MQCHSSTATE_NET_CONNECTING      1250
 #define MQCHSSTATE_SSL_HANDSHAKING     1300
 #define MQCHSSTATE_NAME_SERVER         1400
 #define MQCHSSTATE_IN_MQPUT            1500
 #define MQCHSSTATE_IN_MQGET            1600
 #define MQCHSSTATE_IN_MQI_CALL         1700
 #define MQCHSSTATE_COMPRESSING         1800

 /* Channel Shared Restart Options */
 #define MQCHSH_RESTART_NO              0
 #define MQCHSH_RESTART_YES             1

 /* Channel Stop Options */
 #define MQCHSR_STOP_NOT_REQUESTED      0
 #define MQCHSR_STOP_REQUESTED          1

 /* Channel reset requested */
 #define MQCHRR_RESET_NOT_REQUESTED     0

 /* Channel Table Types */
 #define MQCHTAB_Q_MGR                  1
 #define MQCHTAB_CLNTCONN               2

 /* Clear Topic String Scope */
 #define MQCLRS_LOCAL                   1
 #define MQCLRS_GLOBAL                  2

 /* Clear Topic String Type */
 #define MQCLRT_RETAINED                1

 /* Command Information Values */
 #define MQCMDI_CMDSCOPE_ACCEPTED       1
 #define MQCMDI_CMDSCOPE_GENERATED      2
 #define MQCMDI_CMDSCOPE_COMPLETED      3
 #define MQCMDI_QSG_DISP_COMPLETED      4
 #define MQCMDI_COMMAND_ACCEPTED        5
 #define MQCMDI_CLUSTER_REQUEST_QUEUED  6
 #define MQCMDI_CHANNEL_INIT_STARTED    7
 #define MQCMDI_RECOVER_STARTED         11
 #define MQCMDI_BACKUP_STARTED          12
 #define MQCMDI_RECOVER_COMPLETED       13
 #define MQCMDI_SEC_TIMER_ZERO          14
 #define MQCMDI_REFRESH_CONFIGURATION   16
 #define MQCMDI_SEC_SIGNOFF_ERROR       17
 #define MQCMDI_IMS_BRIDGE_SUSPENDED    18
 #define MQCMDI_DB2_SUSPENDED           19
 #define MQCMDI_DB2_OBSOLETE_MSGS       20
 #define MQCMDI_SEC_UPPERCASE           21
 #define MQCMDI_SEC_MIXEDCASE           22

 /* Disconnect Types */
 #define MQDISCONNECT_NORMAL            0
 #define MQDISCONNECT_IMPLICIT          1
 #define MQDISCONNECT_Q_MGR             2

 /* Escape Types */
 #define MQET_MQSC                      1

 /* Event Origins */
 #define MQEVO_OTHER                    0
 #define MQEVO_CONSOLE                  1
 #define MQEVO_INIT                     2
 #define MQEVO_MSG                      3
 #define MQEVO_MQSET                    4
 #define MQEVO_INTERNAL                 5
 #define MQEVO_MQSUB                    6
 #define MQEVO_CTLMSG                   7
 #define MQEVO_REST                     8

 /* Event Recording */
 #define MQEVR_DISABLED                 0
 #define MQEVR_ENABLED                  1
 #define MQEVR_EXCEPTION                2
 #define MQEVR_NO_DISPLAY               3
 #define MQEVR_API_ONLY                 4
 #define MQEVR_ADMIN_ONLY               5
 #define MQEVR_USER_ONLY                6

 /* Force Options */
 #define MQFC_YES                       1
 #define MQFC_NO                        0

 /* Handle States */
 #define MQHSTATE_INACTIVE              0
 #define MQHSTATE_ACTIVE                1

 /* Inbound Dispositions */
 #define MQINBD_Q_MGR                   0
 #define MQINBD_GROUP                   3

 /* Indoubt Options */
 #define MQIDO_COMMIT                   1
 #define MQIDO_BACKOUT                  2

 /* Match Types */
 #define MQMATCH_GENERIC                0
 #define MQMATCH_RUNCHECK               1
 #define MQMATCH_EXACT                  2
 #define MQMATCH_ALL                    3

 /* Message Channel Agent Status */
 #define MQMCAS_STOPPED                 0
 #define MQMCAS_RUNNING                 3

 /* Mode Options */
 #define MQMODE_FORCE                   0
 #define MQMODE_QUIESCE                 1
 #define MQMODE_TERMINATE               2

 /* Message Level Protection */
 #define MQMLP_TOLERATE_UNPROTECTED_NO  0
 #define MQMLP_TOLERATE_UNPROTECTED_YES 1
 #define MQMLP_ENCRYPTION_ALG_NONE      0
 #define MQMLP_ENCRYPTION_ALG_RC2       1
 #define MQMLP_ENCRYPTION_ALG_DES       2
 #define MQMLP_ENCRYPTION_ALG_3DES      3
 #define MQMLP_ENCRYPTION_ALG_AES128    4
 #define MQMLP_ENCRYPTION_ALG_AES256    5
 #define MQMLP_SIGN_ALG_NONE            0
 #define MQMLP_SIGN_ALG_MD5             1
 #define MQMLP_SIGN_ALG_SHA1            2
 #define MQMLP_SIGN_ALG_SHA224          3
 #define MQMLP_SIGN_ALG_SHA256          4
 #define MQMLP_SIGN_ALG_SHA384          5
 #define MQMLP_SIGN_ALG_SHA512          6

 /* Purge Options */
 #define MQPO_YES                       1
 #define MQPO_NO                        0

 /* Pub/Sub Status Counts */
 #define MQPSCT_NONE                    (-1)

 /* Pub/Sub Status Type */
 #define MQPSST_ALL                     0
 #define MQPSST_LOCAL                   1
 #define MQPSST_PARENT                  2
 #define MQPSST_CHILD                   3

 /* Pub/Sub Status */
 #define MQPS_STATUS_INACTIVE           0
 #define MQPS_STATUS_STARTING           1
 #define MQPS_STATUS_STOPPING           2
 #define MQPS_STATUS_ACTIVE             3
 #define MQPS_STATUS_COMPAT             4
 #define MQPS_STATUS_ERROR              5
 #define MQPS_STATUS_REFUSED            6

 /* Queue Manager Definition Types */
 #define MQQMDT_EXPLICIT_CLUSTER_SENDER 1
 #define MQQMDT_AUTO_CLUSTER_SENDER     2
 #define MQQMDT_AUTO_EXP_CLUSTER_SENDER 4
 #define MQQMDT_CLUSTER_RECEIVER        3

 /* Queue Manager Facility */
 #define MQQMFAC_IMS_BRIDGE             1
 #define MQQMFAC_DB2                    2

 /* Queue Manager Status */
 #define MQQMSTA_STARTING               1
 #define MQQMSTA_RUNNING                2
 #define MQQMSTA_QUIESCING              3
 #define MQQMSTA_STANDBY                4

 /* Queue Manager Types */
 #define MQQMT_NORMAL                   0
 #define MQQMT_REPOSITORY               1

 /* Quiesce Options */
 #define MQQO_YES                       1
 #define MQQO_NO                        0

 /* Queue Service-Interval Events */
 #define MQQSIE_NONE                    0
 #define MQQSIE_HIGH                    1
 #define MQQSIE_OK                      2

 /* Queue Status Open Types */
 #define MQQSOT_ALL                     1
 #define MQQSOT_INPUT                   2
 #define MQQSOT_OUTPUT                  3

 /* QSG Status */
 #define MQQSGS_UNKNOWN                 0
 #define MQQSGS_CREATED                 1
 #define MQQSGS_ACTIVE                  2
 #define MQQSGS_INACTIVE                3
 #define MQQSGS_FAILED                  4
 #define MQQSGS_PENDING                 5

 /* Queue Status Open Options for SET, BROWSE, INPUT */
 #define MQQSO_NO                       0
 #define MQQSO_YES                      1
 #define MQQSO_SHARED                   1
 #define MQQSO_EXCLUSIVE                2

 /* Queue Status Uncommitted Messages */
 #define MQQSUM_YES                     1
 #define MQQSUM_NO                      0

 /* Remove Authority Record Options */
 #define MQRAR_YES                      1
 #define MQRAR_NO                       0

 /* Replace Options */
 #define MQRP_YES                       1
 #define MQRP_NO                        0

 /* Reason Qualifiers */
 #define MQRQ_CONN_NOT_AUTHORIZED       1
 #define MQRQ_OPEN_NOT_AUTHORIZED       2
 #define MQRQ_CLOSE_NOT_AUTHORIZED      3
 #define MQRQ_CMD_NOT_AUTHORIZED        4
 #define MQRQ_Q_MGR_STOPPING            5
 #define MQRQ_Q_MGR_QUIESCING           6
 #define MQRQ_CHANNEL_STOPPED_OK        7
 #define MQRQ_CHANNEL_STOPPED_ERROR     8
 #define MQRQ_CHANNEL_STOPPED_RETRY     9
 #define MQRQ_CHANNEL_STOPPED_DISABLED  10
 #define MQRQ_BRIDGE_STOPPED_OK         11
 #define MQRQ_BRIDGE_STOPPED_ERROR      12
 #define MQRQ_SSL_HANDSHAKE_ERROR       13
 #define MQRQ_SSL_CIPHER_SPEC_ERROR     14
 #define MQRQ_SSL_CLIENT_AUTH_ERROR     15
 #define MQRQ_SSL_PEER_NAME_ERROR       16
 #define MQRQ_SUB_NOT_AUTHORIZED        17
 #define MQRQ_SUB_DEST_NOT_AUTHORIZED   18
 #define MQRQ_SSL_UNKNOWN_REVOCATION    19
 #define MQRQ_SYS_CONN_NOT_AUTHORIZED   20
 #define MQRQ_CHANNEL_BLOCKED_ADDRESS   21
 #define MQRQ_CHANNEL_BLOCKED_USERID    22
 #define MQRQ_CHANNEL_BLOCKED_NOACCESS  23
 #define MQRQ_MAX_ACTIVE_CHANNELS       24
 #define MQRQ_MAX_CHANNELS              25
 #define MQRQ_SVRCONN_INST_LIMIT        26
 #define MQRQ_CLIENT_INST_LIMIT         27
 #define MQRQ_CAF_NOT_INSTALLED         28
 #define MQRQ_CSP_NOT_AUTHORIZED        29
 #define MQRQ_FAILOVER_PERMITTED        30
 #define MQRQ_FAILOVER_NOT_PERMITTED    31
 #define MQRQ_STANDBY_ACTIVATED         32

 /* Refresh Types */
 #define MQRT_CONFIGURATION             1
 #define MQRT_EXPIRY                    2
 #define MQRT_NSPROC                    3
 #define MQRT_PROXYSUB                  4
 #define MQRT_SUB_CONFIGURATION         5

 /* Queue Definition Scope */
 #define MQSCO_Q_MGR                    1
 #define MQSCO_CELL                     2

 /* Security Items */
 #define MQSECITEM_ALL                  0
 #define MQSECITEM_MQADMIN              1
 #define MQSECITEM_MQNLIST              2
 #define MQSECITEM_MQPROC               3
 #define MQSECITEM_MQQUEUE              4
 #define MQSECITEM_MQCONN               5
 #define MQSECITEM_MQCMDS               6
 #define MQSECITEM_MXADMIN              7
 #define MQSECITEM_MXNLIST              8
 #define MQSECITEM_MXPROC               9
 #define MQSECITEM_MXQUEUE              10
 #define MQSECITEM_MXTOPIC              11

 /* Security Switches */
 #define MQSECSW_PROCESS                1
 #define MQSECSW_NAMELIST               2
 #define MQSECSW_Q                      3
 #define MQSECSW_TOPIC                  4
 #define MQSECSW_CONTEXT                6
 #define MQSECSW_ALTERNATE_USER         7
 #define MQSECSW_COMMAND                8
 #define MQSECSW_CONNECTION             9
 #define MQSECSW_SUBSYSTEM              10
 #define MQSECSW_COMMAND_RESOURCES      11
 #define MQSECSW_Q_MGR                  15
 #define MQSECSW_QSG                    16

 /* Security Switch States */
 #define MQSECSW_OFF_FOUND              21
 #define MQSECSW_ON_FOUND               22
 #define MQSECSW_OFF_NOT_FOUND          23
 #define MQSECSW_ON_NOT_FOUND           24
 #define MQSECSW_OFF_ERROR              25
 #define MQSECSW_ON_OVERRIDDEN          26

 /* Security Types */
 #define MQSECTYPE_AUTHSERV             1
 #define MQSECTYPE_SSL                  2
 #define MQSECTYPE_CLASSES              3
 #define MQSECTYPE_CONNAUTH             4

 /* Authentication Validation Types */
 #define MQCHK_OPTIONAL                 0
 #define MQCHK_NONE                     1
 #define MQCHK_REQUIRED_ADMIN           2
 #define MQCHK_REQUIRED                 3
 #define MQCHK_AS_Q_MGR                 4

 /* Authentication Adoption Context */
 #define MQADPCTX_NO                    0
 #define MQADPCTX_YES                   1

 /* LDAP SSL/TLS Connection State */
 #define MQSECCOMM_NO                   0
 #define MQSECCOMM_YES                  1
 #define MQSECCOMM_ANON                 2

 /* LDAP Authorisation Method */
 #define MQLDAP_AUTHORMD_OS             0
 #define MQLDAP_AUTHORMD_SEARCHGRP      1
 #define MQLDAP_AUTHORMD_SEARCHUSR      2
 #define MQLDAP_AUTHORMD_SRCHGRPSN      3

 /* LDAP Nested Group Policy */
 #define MQLDAP_NESTGRP_NO              0
 #define MQLDAP_NESTGRP_YES             1

 /* Authentication Method */
 #define MQAUTHENTICATE_OS              0
 #define MQAUTHENTICATE_PAM             1

 /* QMgr LDAP Connection Status */
 #define MQLDAPC_INACTIVE               0
 #define MQLDAPC_CONNECTED              1
 #define MQLDAPC_ERROR                  2

 /* Selector types */
 #define MQSELTYPE_NONE                 0
 #define MQSELTYPE_STANDARD             1
 #define MQSELTYPE_EXTENDED             2

 /* CHLAUTH QMGR State */
 #define MQCHLA_DISABLED                0
 #define MQCHLA_ENABLED                 1

 /* REVDNS QMGR State */
 #define MQRDNS_ENABLED                 0
 #define MQRDNS_DISABLED                1

 /* CLROUTE Topic State */
 #define MQCLROUTE_DIRECT               0
 #define MQCLROUTE_TOPIC_HOST           1
 #define MQCLROUTE_NONE                 2

 /* CLSTATE Clustered Topic Definition State */
 #define MQCLST_ACTIVE                  0
 #define MQCLST_PENDING                 1
 #define MQCLST_INVALID                 2
 #define MQCLST_ERROR                   3

 /* Transmission queue types */
 #define MQCLXQ_SCTQ                    0
 #define MQCLXQ_CHANNEL                 1

 /* Suspend Status */
 #define MQSUS_YES                      1
 #define MQSUS_NO                       0

 /* Syncpoint values for Pub/Sub migration */
 #define MQSYNCPOINT_YES                0
 #define MQSYNCPOINT_IFPER              1

 /* System Parameter Values */
 #define MQSYSP_NO                      0
 #define MQSYSP_YES                     1
 #define MQSYSP_EXTENDED                2
 #define MQSYSP_TYPE_INITIAL            10
 #define MQSYSP_TYPE_SET                11
 #define MQSYSP_TYPE_LOG_COPY           12
 #define MQSYSP_TYPE_LOG_STATUS         13
 #define MQSYSP_TYPE_ARCHIVE_TAPE       14
 #define MQSYSP_ALLOC_BLK               20
 #define MQSYSP_ALLOC_TRK               21
 #define MQSYSP_ALLOC_CYL               22
 #define MQSYSP_STATUS_BUSY             30
 #define MQSYSP_STATUS_PREMOUNT         31
 #define MQSYSP_STATUS_AVAILABLE        32
 #define MQSYSP_STATUS_UNKNOWN          33
 #define MQSYSP_STATUS_ALLOC_ARCHIVE    34
 #define MQSYSP_STATUS_COPYING_BSDS     35
 #define MQSYSP_STATUS_COPYING_LOG      36

 /* Export Type */
 #define MQEXT_ALL                      0
 #define MQEXT_OBJECT                   1
 #define MQEXT_AUTHORITY                2

 /* Export Attrs */
 #define MQEXTATTRS_ALL                 0
 #define MQEXTATTRS_NONDEF              1

 /* System Objects */
 #define MQSYSOBJ_YES                   0
 #define MQSYSOBJ_NO                    1

 /* Subscription Types */
 #define MQSUBTYPE_API                  1
 #define MQSUBTYPE_ADMIN                2
 #define MQSUBTYPE_PROXY                3
 #define MQSUBTYPE_ALL                  (-1)
 #define MQSUBTYPE_USER                 (-2)

 /* Display Subscription Types */
 #define MQDOPT_RESOLVED                0
 #define MQDOPT_DEFINED                 1

 /* Time units */
 #define MQTIME_UNIT_MINS               0
 #define MQTIME_UNIT_SECS               1

 /* User ID Support */
 #define MQUIDSUPP_NO                   0
 #define MQUIDSUPP_YES                  1

 /* Undelivered values for Pub/Sub migration */
 #define MQUNDELIVERED_NORMAL           0
 #define MQUNDELIVERED_SAFE             1
 #define MQUNDELIVERED_DISCARD          2
 #define MQUNDELIVERED_KEEP             3

 /* UOW States */
 #define MQUOWST_NONE                   0
 #define MQUOWST_ACTIVE                 1
 #define MQUOWST_PREPARED               2
 #define MQUOWST_UNRESOLVED             3

 /* UOW Types */
 #define MQUOWT_Q_MGR                   0
 #define MQUOWT_CICS                    1
 #define MQUOWT_RRS                     2
 #define MQUOWT_IMS                     3
 #define MQUOWT_XA                      4

 /* Page Set Usage Values */
 #define MQUSAGE_PS_AVAILABLE           0
 #define MQUSAGE_PS_DEFINED             1
 #define MQUSAGE_PS_OFFLINE             2
 #define MQUSAGE_PS_NOT_DEFINED         3
 #define MQUSAGE_PS_SUSPENDED           4

 /* Expand Usage Values */
 #define MQUSAGE_EXPAND_USER            1
 #define MQUSAGE_EXPAND_SYSTEM          2
 #define MQUSAGE_EXPAND_NONE            3

 /* Data Set Usage Values */
 #define MQUSAGE_DS_OLDEST_ACTIVE_UOW   10
 #define MQUSAGE_DS_OLDEST_PS_RECOVERY  11
 #define MQUSAGE_DS_OLDEST_CF_RECOVERY  12

 /* Multicast Properties Options */
 #define MQMCP_REPLY                    2
 #define MQMCP_USER                     1
 #define MQMCP_NONE                     0
 #define MQMCP_ALL                      (-1)
 #define MQMCP_COMPAT                   (-2)

 /* Multicast New Subscriber History Options */
 #define MQNSH_NONE                     0
 #define MQNSH_ALL                      (-1)

 /* Reduce Log Options */
 #define MQLR_ONE                       1
 #define MQLR_AUTO                      (-1)
 #define MQLR_MAX                       (-2)

 /****************************************************************/
 /* Values Related to Trace-route and Activity Operations        */
 /****************************************************************/

 /* Activity Operations */
 #define MQOPER_SYSTEM_FIRST            0
 #define MQOPER_UNKNOWN                 0
 #define MQOPER_BROWSE                  1
 #define MQOPER_DISCARD                 2
 #define MQOPER_GET                     3
 #define MQOPER_PUT                     4
 #define MQOPER_PUT_REPLY               5
 #define MQOPER_PUT_REPORT              6
 #define MQOPER_RECEIVE                 7
 #define MQOPER_SEND                    8
 #define MQOPER_TRANSFORM               9
 #define MQOPER_PUBLISH                 10
 #define MQOPER_EXCLUDED_PUBLISH        11
 #define MQOPER_DISCARDED_PUBLISH       12
 #define MQOPER_SYSTEM_LAST             65535
 #define MQOPER_APPL_FIRST              65536
 #define MQOPER_APPL_LAST               999999999

 /* Trace-route Max Activities (MQIACF_MAX_ACTIVITIES) */
 #define MQROUTE_UNLIMITED_ACTIVITIES   0

 /* Trace-route Detail (MQIACF_ROUTE_DETAIL) */
 #define MQROUTE_DETAIL_LOW             0x00000002
 #define MQROUTE_DETAIL_MEDIUM          0x00000008
 #define MQROUTE_DETAIL_HIGH            0x00000020

 /* Trace-route Forwarding (MQIACF_ROUTE_FORWARDING) */
 #define MQROUTE_FORWARD_ALL            0x00000100
 #define MQROUTE_FORWARD_IF_SUPPORTED   0x00000200
 #define MQROUTE_FORWARD_REJ_UNSUP_MASK 0xFFFF0000

 /* Trace-route Delivery (MQIACF_ROUTE_DELIVERY) */
 #define MQROUTE_DELIVER_YES            0x00001000
 #define MQROUTE_DELIVER_NO             0x00002000
 #define MQROUTE_DELIVER_REJ_UNSUP_MASK 0xFFFF0000

 /* Trace-route Accumulation (MQIACF_ROUTE_ACCUMULATION) */
 #define MQROUTE_ACCUMULATE_NONE        0x00010003
 #define MQROUTE_ACCUMULATE_IN_MSG      0x00010004
 #define MQROUTE_ACCUMULATE_AND_REPLY   0x00010005

 /****************************************************************/
 /* Values Related to Publish/Subscribe                          */
 /****************************************************************/

 /* Delete Options */
 #define MQDELO_NONE                    0x00000000
 #define MQDELO_LOCAL                   0x00000004

 /* Publication Options */
 #define MQPUBO_NONE                    0x00000000
 #define MQPUBO_CORREL_ID_AS_IDENTITY   0x00000001
 #define MQPUBO_RETAIN_PUBLICATION      0x00000002
 #define MQPUBO_OTHER_SUBSCRIBERS_ONLY  0x00000004
 #define MQPUBO_NO_REGISTRATION         0x00000008
 #define MQPUBO_IS_RETAINED_PUBLICATION 0x00000010

 /* Registration Options */
 #define MQREGO_NONE                    0x00000000
 #define MQREGO_CORREL_ID_AS_IDENTITY   0x00000001
 #define MQREGO_ANONYMOUS               0x00000002
 #define MQREGO_LOCAL                   0x00000004
 #define MQREGO_DIRECT_REQUESTS         0x00000008
 #define MQREGO_NEW_PUBLICATIONS_ONLY   0x00000010
 #define MQREGO_PUBLISH_ON_REQUEST_ONLY 0x00000020
 #define MQREGO_DEREGISTER_ALL          0x00000040
 #define MQREGO_INCLUDE_STREAM_NAME     0x00000080
 #define MQREGO_INFORM_IF_RETAINED      0x00000100
 #define MQREGO_DUPLICATES_OK           0x00000200
 #define MQREGO_NON_PERSISTENT          0x00000400
 #define MQREGO_PERSISTENT              0x00000800
 #define MQREGO_PERSISTENT_AS_PUBLISH   0x00001000
 #define MQREGO_PERSISTENT_AS_Q         0x00002000
 #define MQREGO_ADD_NAME                0x00004000
 #define MQREGO_NO_ALTERATION           0x00008000
 #define MQREGO_FULL_RESPONSE           0x00010000
 #define MQREGO_JOIN_SHARED             0x00020000
 #define MQREGO_JOIN_EXCLUSIVE          0x00040000
 #define MQREGO_LEAVE_ONLY              0x00080000
 #define MQREGO_VARIABLE_USER_ID        0x00100000
 #define MQREGO_LOCKED                  0x00200000

 /* User Attribute Selectors */
 #define MQUA_FIRST                     65536
 #define MQUA_LAST                      999999999

 /* Grouped Units of Recovery */
 #define MQGUR_DISABLED                 0
 #define MQGUR_ENABLED                  1

 /* Measured usage by API */
 #define MQMULC_STANDARD                0
 #define MQMULC_REFINED                 1

 /* Multi-instance Queue Managers */
 #define MQSTDBY_NOT_PERMITTED          0
 #define MQSTDBY_PERMITTED              1

 /****************************************************************/
 /* MQCFH Structure -- PCF Header                                */
 /****************************************************************/


 typedef struct tagMQCFH MQCFH;
 typedef MQCFH MQPOINTER PMQCFH;

 struct tagMQCFH {
   MQLONG  Type;            /* Structure type */
   MQLONG  StrucLength;     /* Structure length */
   MQLONG  Version;         /* Structure version number */
   MQLONG  Command;         /* Command identifier */
   MQLONG  MsgSeqNumber;    /* Message sequence number */
   MQLONG  Control;         /* Control options */
   MQLONG  CompCode;        /* Completion code */
   MQLONG  Reason;          /* Reason code qualifying completion code */
   MQLONG  ParameterCount;  /* Count of parameter structures */
 };

 #define MQCFH_DEFAULT MQCFT_COMMAND,\
                       MQCFH_STRUC_LENGTH,\
                       MQCFH_VERSION_1,\
                       MQCMD_NONE,\
                       1,\
                       MQCFC_LAST,\
                       MQCC_OK,\
                       MQRC_NONE,\
                       0

 /****************************************************************/
 /* MQCFBF Structure -- PCF Byte String Filter Parameter         */
 /****************************************************************/


 typedef struct tagMQCFBF MQCFBF;
 typedef MQCFBF MQPOINTER PMQCFBF;

 struct tagMQCFBF {
   MQLONG  Type;               /* Structure type */
   MQLONG  StrucLength;        /* Structure length */
   MQLONG  Parameter;          /* Parameter identifier */
   MQLONG  Operator;           /* Operator identifier */
   MQLONG  FilterValueLength;  /* Filter value length */
   MQBYTE  FilterValue[1];     /* Filter value -- first byte */
 };

 #define MQCFBF_DEFAULT MQCFT_BYTE_STRING_FILTER,\
                        MQCFBF_STRUC_LENGTH_FIXED,\
                        0,\
                        0,\
                        0,\
                        {""}

 /****************************************************************/
 /* MQCFBS Structure -- PCF Byte String Parameter                */
 /****************************************************************/


 typedef struct tagMQCFBS MQCFBS;
 typedef MQCFBS MQPOINTER PMQCFBS;

 struct tagMQCFBS {
   MQLONG  Type;          /* Structure type */
   MQLONG  StrucLength;   /* Structure length */
   MQLONG  Parameter;     /* Parameter identifier */
   MQLONG  StringLength;  /* Length of string */
   MQBYTE  String[1];     /* String value -- first byte */
 };

 #define MQCFBS_DEFAULT MQCFT_BYTE_STRING,\
                        MQCFBS_STRUC_LENGTH_FIXED,\
                        0,\
                        0,\
                        {""}

 /****************************************************************/
 /* MQCFGR Structure -- PCF Group Parameter                      */
 /****************************************************************/


 typedef struct tagMQCFGR MQCFGR;
 typedef MQCFGR MQPOINTER PMQCFGR;

 struct tagMQCFGR {
   MQLONG  Type;            /* Structure type */
   MQLONG  StrucLength;     /* Structure length */
   MQLONG  Parameter;       /* Parameter identifier */
   MQLONG  ParameterCount;  /* Count of group parameter structures */
 };

 #define MQCFGR_DEFAULT MQCFT_GROUP,\
                        MQCFGR_STRUC_LENGTH,\
                        0,\
                        0

 /****************************************************************/
 /* MQCFIF Structure -- PCF Integer Filter Parameter             */
 /****************************************************************/


 typedef struct tagMQCFIF MQCFIF;
 typedef MQCFIF MQPOINTER PMQCFIF;

 struct tagMQCFIF {
   MQLONG  Type;         /* Structure type */
   MQLONG  StrucLength;  /* Structure length */
   MQLONG  Parameter;    /* Parameter identifier */
   MQLONG  Operator;     /* Operator identifier */
   MQLONG  FilterValue;  /* Filter value */
 };

 #define MQCFIF_DEFAULT MQCFT_INTEGER_FILTER,\
                        MQCFIF_STRUC_LENGTH,\
                        0,\
                        0,\
                        0

 /****************************************************************/
 /* MQCFIL Structure -- PCF Integer-List Parameter               */
 /****************************************************************/


 typedef struct tagMQCFIL MQCFIL;
 typedef MQCFIL MQPOINTER PMQCFIL;

 struct tagMQCFIL {
   MQLONG  Type;         /* Structure type */
   MQLONG  StrucLength;  /* Structure length */
   MQLONG  Parameter;    /* Parameter identifier */
   MQLONG  Count;        /* Count of parameter values */
   MQLONG  Values[1];    /* Parameter values -- first element */
 };

 #define MQCFIL_DEFAULT MQCFT_INTEGER_LIST,\
                        MQCFIL_STRUC_LENGTH_FIXED,\
                        0,\
                        0,\
                        {0}

 /****************************************************************/
 /* MQCFIL64 Structure -- PCF 64-bit Integer-List Parameter      */
 /****************************************************************/


 typedef struct tagMQCFIL64 MQCFIL64;
 typedef MQCFIL64 MQPOINTER PMQCFIL64;

 struct tagMQCFIL64 {
   MQLONG   Type;         /* Structure type */
   MQLONG   StrucLength;  /* Structure length */
   MQLONG   Parameter;    /* Parameter identifier */
   MQLONG   Count;        /* Count of parameter values */
   MQINT64  Values[1];    /* Parameter values -- first element */
 };

 #define MQCFIL64_DEFAULT MQCFT_INTEGER64_LIST,\
                          MQCFIL64_STRUC_LENGTH_FIXED,\
                          0,\
                          0,\
                          {0}

 /****************************************************************/
 /* MQCFIN Structure -- PCF Integer Parameter                    */
 /****************************************************************/


 typedef struct tagMQCFIN MQCFIN;
 typedef MQCFIN MQPOINTER PMQCFIN;

 struct tagMQCFIN {
   MQLONG  Type;         /* Structure type */
   MQLONG  StrucLength;  /* Structure length */
   MQLONG  Parameter;    /* Parameter identifier */
   MQLONG  Value;        /* Parameter value */
 };

 #define MQCFIN_DEFAULT MQCFT_INTEGER,\
                        MQCFIN_STRUC_LENGTH,\
                        0,\
                        0

 /****************************************************************/
 /* MQCFIN64 Structure -- PCF 64-bit Integer Parameter           */
 /****************************************************************/


 typedef struct tagMQCFIN64 MQCFIN64;
 typedef MQCFIN64 MQPOINTER PMQCFIN64;

 struct tagMQCFIN64 {
   MQLONG   Type;         /* Structure type */
   MQLONG   StrucLength;  /* Structure length */
   MQLONG   Parameter;    /* Parameter identifier */
   MQLONG   Reserved;     /* Reserved */
   MQINT64  Value;        /* Parameter value */
 };

 #define MQCFIN64_DEFAULT MQCFT_INTEGER64,\
                          MQCFIN64_STRUC_LENGTH,\
                          0,\
                          0,\
                          0

 /****************************************************************/
 /* MQCFSF Structure -- PCF String Filter Parameter              */
 /****************************************************************/


 typedef struct tagMQCFSF MQCFSF;
 typedef MQCFSF MQPOINTER PMQCFSF;

 struct tagMQCFSF {
   MQLONG  Type;               /* Structure type */
   MQLONG  StrucLength;        /* Structure length */
   MQLONG  Parameter;          /* Parameter identifier */
   MQLONG  Operator;           /* Operator identifier */
   MQLONG  CodedCharSetId;     /* Coded character set identifier */
   MQLONG  FilterValueLength;  /* Filter value length */
   MQCHAR  FilterValue[1];     /* Filter value -- first character */
 };

 #define MQCFSF_DEFAULT MQCFT_STRING_FILTER,\
                        MQCFSF_STRUC_LENGTH_FIXED,\
                        0,\
                        0,\
                        MQCCSI_DEFAULT,\
                        0,\
                        {""}

 /****************************************************************/
 /* MQCFSL Structure -- PCF String-List Parameter                */
 /****************************************************************/


 typedef struct tagMQCFSL MQCFSL;
 typedef MQCFSL MQPOINTER PMQCFSL;

 struct tagMQCFSL {
   MQLONG  Type;            /* Structure type */
   MQLONG  StrucLength;     /* Structure length */
   MQLONG  Parameter;       /* Parameter identifier */
   MQLONG  CodedCharSetId;  /* Coded character set identifier */
   MQLONG  Count;           /* Count of parameter values */
   MQLONG  StringLength;    /* Length of one string */
   MQCHAR  Strings[1];      /* String values -- first character */
 };

 #define MQCFSL_DEFAULT MQCFT_STRING_LIST,\
                        MQCFSL_STRUC_LENGTH_FIXED,\
                        0,\
                        MQCCSI_DEFAULT,\
                        0,\
                        0,\
                        {""}

 /****************************************************************/
 /* MQCFST Structure -- PCF String Parameter                     */
 /****************************************************************/


 typedef struct tagMQCFST MQCFST;
 typedef MQCFST MQPOINTER PMQCFST;

 struct tagMQCFST {
   MQLONG  Type;            /* Structure type */
   MQLONG  StrucLength;     /* Structure length */
   MQLONG  Parameter;       /* Parameter identifier */
   MQLONG  CodedCharSetId;  /* Coded character set identifier */
   MQLONG  StringLength;    /* Length of string */
   MQCHAR  String[1];       /* String value -- first character */
 };

 #define MQCFST_DEFAULT MQCFT_STRING,\
                        MQCFST_STRUC_LENGTH_FIXED,\
                        0,\
                        MQCCSI_DEFAULT,\
                        0,\
                        {""}

 /****************************************************************/
 /* MQEPH Structure -- Embedded PCF header                       */
 /****************************************************************/


 typedef struct tagMQEPH MQEPH;
 typedef MQEPH MQPOINTER PMQEPH;

 struct tagMQEPH {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   StrucLength;     /* Total length of MQEPH including MQCFH */
                             /* and parameter structures that follow */
   MQLONG   Encoding;        /* Numeric encoding of data that follows */
                             /* last PCF parameter structure */
   MQLONG   CodedCharSetId;  /* Character set identifier of data that */
                             /* follows last PCF parameter structure */
   MQCHAR8  Format;          /* Format name of data that follows last */
                             /* PCF parameter structure */
   MQLONG   Flags;           /* Flags */
   MQCFH    PCFHeader;       /* Programmable Command Format Header */
 };

 #define MQEPH_DEFAULT {MQEPH_STRUC_ID_ARRAY},\
                       MQEPH_VERSION_1,\
                       MQEPH_STRUC_LENGTH_FIXED,\
                       0,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       MQEPH_NONE,\
                       { MQCFT_COMMAND,\
                         MQCFH_STRUC_LENGTH,\
                         MQCFH_VERSION_3,\
                         MQCMD_NONE,\
                         1,\
                         MQCFC_LAST,\
                         MQCC_OK,\
                         MQRC_NONE,\
                         0 }


 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQCFC                                               */
 /****************************************************************/
 #endif  /* End of header file */
