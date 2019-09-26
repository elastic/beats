 #if !defined(MQC_INCLUDED)            /* File not yet included? */
   #define MQC_INCLUDED                /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQC                                        */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for Main MQI                   */
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
 /*                  structures and named constants for the      */
 /*                  main MQI.                                   */
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
 /* @(#) famfiles/xml/approved/cmqc.xml, mqmake, p000 1.125      */
 /* 12/02/10 10:34:59                                            */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 #if defined(__LP64__)
  #define MQ_64_BIT
 #endif

 /****************************************************************/
 /* Values Related to MQAIR Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQAIR_STRUC_ID                 "AIR "

 /* Structure Identifier (array form) */
 #define MQAIR_STRUC_ID_ARRAY           'A','I','R',' '

 /* Structure Version Number */
 #define MQAIR_VERSION_1                1
 #define MQAIR_VERSION_2                2
 #define MQAIR_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQAIR_LENGTH_1                 328
#else
 #define MQAIR_LENGTH_1                 320
#endif
#if defined(MQ_64_BIT)
 #define MQAIR_LENGTH_2                 584
#else
 #define MQAIR_LENGTH_2                 576
#endif
#if defined(MQ_64_BIT)
 #define MQAIR_CURRENT_LENGTH           584
#else
 #define MQAIR_CURRENT_LENGTH           576
#endif

 /* Authentication Information Type */
 #define MQAIT_ALL                      0
 #define MQAIT_CRL_LDAP                 1
 #define MQAIT_OCSP                     2
 #define MQAIT_IDPW_OS                  3
 #define MQAIT_IDPW_LDAP                4

 /****************************************************************/
 /* Values Related to MQBMHO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQBMHO_STRUC_ID                "BMHO"

 /* Structure Identifier (array form) */
 #define MQBMHO_STRUC_ID_ARRAY          'B','M','H','O'

 /* Structure Version Number */
 #define MQBMHO_VERSION_1               1
 #define MQBMHO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQBMHO_LENGTH_1                12
 #define MQBMHO_CURRENT_LENGTH          12

 /* Buffer To Message Handle Options */
 #define MQBMHO_NONE                    0x00000000
 #define MQBMHO_DELETE_PROPERTIES       0x00000001

 /****************************************************************/
 /* Values Related to MQBO Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQBO_STRUC_ID                  "BO  "

 /* Structure Identifier (array form) */
 #define MQBO_STRUC_ID_ARRAY            'B','O',' ',' '

 /* Structure Version Number */
 #define MQBO_VERSION_1                 1
 #define MQBO_CURRENT_VERSION           1

 /* Structure Length */
 #define MQBO_LENGTH_1                  12
 #define MQBO_CURRENT_LENGTH            12

 /* Begin Options */
 #define MQBO_NONE                      0x00000000

 /****************************************************************/
 /* Values Related to MQCBC Structure - Callback Context         */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCBC_STRUC_ID                 "CBC "

 /* Structure Identifier (array form) */
 #define MQCBC_STRUC_ID_ARRAY           'C','B','C',' '

 /* Structure Version Number */
 #define MQCBC_VERSION_1                1
 #define MQCBC_VERSION_2                2
 #define MQCBC_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQCBC_LENGTH_1                 56
#else
 #define MQCBC_LENGTH_1                 48
#endif
#if defined(MQ_64_BIT)
 #define MQCBC_LENGTH_2                 64
#else
 #define MQCBC_LENGTH_2                 52
#endif
#if defined(MQ_64_BIT)
 #define MQCBC_CURRENT_LENGTH           64
#else
 #define MQCBC_CURRENT_LENGTH           52
#endif

 /* Flags */
 #define MQCBCF_NONE                    0x00000000
 #define MQCBCF_READA_BUFFER_EMPTY      0x00000001

 /* Callback type */
 #define MQCBCT_START_CALL              1
 #define MQCBCT_STOP_CALL               2
 #define MQCBCT_REGISTER_CALL           3
 #define MQCBCT_DEREGISTER_CALL         4
 #define MQCBCT_EVENT_CALL              5
 #define MQCBCT_MSG_REMOVED             6
 #define MQCBCT_MSG_NOT_REMOVED         7
 #define MQCBCT_MC_EVENT_CALL           8

 /* Consumer state */
 #define MQCS_NONE                      0
 #define MQCS_SUSPENDED_TEMPORARY       1
 #define MQCS_SUSPENDED_USER_ACTION     2
 #define MQCS_SUSPENDED                 3
 #define MQCS_STOPPED                   4

 /* Reconnect delay */
 #define MQRD_NO_RECONNECT              (-1)
 #define MQRD_NO_DELAY                  0

 /****************************************************************/
 /* Values Related to MQCBD Structure - Callback Descriptor      */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCBD_STRUC_ID                 "CBD "

 /* Structure Identifier (array form) */
 #define MQCBD_STRUC_ID_ARRAY           'C','B','D',' '

 /* Structure Version Number */
 #define MQCBD_VERSION_1                1
 #define MQCBD_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQCBD_LENGTH_1                 168
#else
 #define MQCBD_LENGTH_1                 156
#endif
#if defined(MQ_64_BIT)
 #define MQCBD_CURRENT_LENGTH           168
#else
 #define MQCBD_CURRENT_LENGTH           156
#endif

 /* Callback Options */
 #define MQCBDO_NONE                    0x00000000
 #define MQCBDO_START_CALL              0x00000001
 #define MQCBDO_STOP_CALL               0x00000004
 #define MQCBDO_REGISTER_CALL           0x00000100
 #define MQCBDO_DEREGISTER_CALL         0x00000200
 #define MQCBDO_FAIL_IF_QUIESCING       0x00002000
 #define MQCBDO_EVENT_CALL              0x00004000
 #define MQCBDO_MC_EVENT_CALL           0x00008000

 /* This is the type of the Callback Function */
 #define MQCBT_MESSAGE_CONSUMER         0x00000001
 #define MQCBT_EVENT_HANDLER            0x00000002

 /* Buffer size values */
 #define MQCBD_FULL_MSG_LENGTH          (-1)

 /****************************************************************/
 /* Values Related to MQCHARV Structure                          */
 /****************************************************************/

 /* Variable String Length */
 #define MQVS_NULL_TERMINATED           (-1)

 /****************************************************************/
 /* Values Related to MQCIH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCIH_STRUC_ID                 "CIH "

 /* Structure Identifier (array form) */
 #define MQCIH_STRUC_ID_ARRAY           'C','I','H',' '

 /* Structure Version Number */
 #define MQCIH_VERSION_1                1
 #define MQCIH_VERSION_2                2
 #define MQCIH_CURRENT_VERSION          2

 /* Structure Length */
 #define MQCIH_LENGTH_1                 164
 #define MQCIH_LENGTH_2                 180
 #define MQCIH_CURRENT_LENGTH           180

 /* Flags */
 #define MQCIH_NONE                     0x00000000
 #define MQCIH_PASS_EXPIRATION          0x00000001
 #define MQCIH_UNLIMITED_EXPIRATION     0x00000000
 #define MQCIH_REPLY_WITHOUT_NULLS      0x00000002
 #define MQCIH_REPLY_WITH_NULLS         0x00000000
 #define MQCIH_SYNC_ON_RETURN           0x00000004
 #define MQCIH_NO_SYNC_ON_RETURN        0x00000000

 /* Return Codes */
 #define MQCRC_OK                       0
 #define MQCRC_CICS_EXEC_ERROR          1
 #define MQCRC_MQ_API_ERROR             2
 #define MQCRC_BRIDGE_ERROR             3
 #define MQCRC_BRIDGE_ABEND             4
 #define MQCRC_APPLICATION_ABEND        5
 #define MQCRC_SECURITY_ERROR           6
 #define MQCRC_PROGRAM_NOT_AVAILABLE    7
 #define MQCRC_BRIDGE_TIMEOUT           8
 #define MQCRC_TRANSID_NOT_AVAILABLE    9

 /* Unit-of-Work Controls */
 #define MQCUOWC_ONLY                   0x00000111
 #define MQCUOWC_CONTINUE               0x00010000
 #define MQCUOWC_FIRST                  0x00000011
 #define MQCUOWC_MIDDLE                 0x00000010
 #define MQCUOWC_LAST                   0x00000110
 #define MQCUOWC_COMMIT                 0x00000100
 #define MQCUOWC_BACKOUT                0x00001100

 /* Get Wait Interval */
 #define MQCGWI_DEFAULT                 (-2)

 /* Link Types */
 #define MQCLT_PROGRAM                  1
 #define MQCLT_TRANSACTION              2

 /* Output Data Length */
 #define MQCODL_AS_INPUT                (-1)

 /* ADS Descriptors */
 #define MQCADSD_NONE                   0x00000000
 #define MQCADSD_SEND                   0x00000001
 #define MQCADSD_RECV                   0x00000010
 #define MQCADSD_MSGFORMAT              0x00000100

 /* Conversational Task Options */
 #define MQCCT_YES                      0x00000001
 #define MQCCT_NO                       0x00000000

 /* Task End Status */
 #define MQCTES_NOSYNC                  0x00000000
 #define MQCTES_COMMIT                  0x00000100
 #define MQCTES_BACKOUT                 0x00001100
 #define MQCTES_ENDTASK                 0x00010000

 /* Facility */
 #define MQCFAC_NONE                    "\0\0\0\0\0\0\0\0"

 /* Facility (array form) */
 #define MQCFAC_NONE_ARRAY              '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Functions */
 #define MQCFUNC_MQCONN                 "CONN"
 #define MQCFUNC_MQGET                  "GET "
 #define MQCFUNC_MQINQ                  "INQ "
 #define MQCFUNC_MQOPEN                 "OPEN"
 #define MQCFUNC_MQPUT                  "PUT "
 #define MQCFUNC_MQPUT1                 "PUT1"
 #define MQCFUNC_NONE                   "    "

 /* Functions (array form) */
 #define MQCFUNC_MQCONN_ARRAY           'C','O','N','N'
 #define MQCFUNC_MQGET_ARRAY            'G','E','T',' '
 #define MQCFUNC_MQINQ_ARRAY            'I','N','Q',' '
 #define MQCFUNC_MQOPEN_ARRAY           'O','P','E','N'
 #define MQCFUNC_MQPUT_ARRAY            'P','U','T',' '
 #define MQCFUNC_MQPUT1_ARRAY           'P','U','T','1'
 #define MQCFUNC_NONE_ARRAY             ' ',' ',' ',' '

 /* Start Codes */
 #define MQCSC_START                    "S   "
 #define MQCSC_STARTDATA                "SD  "
 #define MQCSC_TERMINPUT                "TD  "
 #define MQCSC_NONE                     "    "

 /* Start Codes (array form) */
 #define MQCSC_START_ARRAY              'S',' ',' ',' '
 #define MQCSC_STARTDATA_ARRAY          'S','D',' ',' '
 #define MQCSC_TERMINPUT_ARRAY          'T','D',' ',' '
 #define MQCSC_NONE_ARRAY               ' ',' ',' ',' '

 /****************************************************************/
 /* Values Related to MQCMHO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCMHO_STRUC_ID                "CMHO"

 /* Structure Identifier (array form) */
 #define MQCMHO_STRUC_ID_ARRAY          'C','M','H','O'

 /* Structure Version Number */
 #define MQCMHO_VERSION_1               1
 #define MQCMHO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQCMHO_LENGTH_1                12
 #define MQCMHO_CURRENT_LENGTH          12

 /* Create Message Handle Options */
 #define MQCMHO_DEFAULT_VALIDATION      0x00000000
 #define MQCMHO_NO_VALIDATION           0x00000001
 #define MQCMHO_VALIDATE                0x00000002
 #define MQCMHO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQCTLO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCTLO_STRUC_ID                "CTLO"

 /* Structure Identifier (array form) */
 #define MQCTLO_STRUC_ID_ARRAY          'C','T','L','O'

 /* Structure Version Number */
 #define MQCTLO_VERSION_1               1
 #define MQCTLO_CURRENT_VERSION         1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQCTLO_LENGTH_1                24
#else
 #define MQCTLO_LENGTH_1                20
#endif
#if defined(MQ_64_BIT)
 #define MQCTLO_CURRENT_LENGTH          24
#else
 #define MQCTLO_CURRENT_LENGTH          20
#endif

 /* Consumer Control Options */
 #define MQCTLO_NONE                    0x00000000
 #define MQCTLO_THREAD_AFFINITY         0x00000001
 #define MQCTLO_FAIL_IF_QUIESCING       0x00002000

 /****************************************************************/
 /* Values Related to MQSCO Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSCO_STRUC_ID                 "SCO "

 /* Structure Identifier (array form) */
 #define MQSCO_STRUC_ID_ARRAY           'S','C','O',' '

 /* Structure Version Number */
 #define MQSCO_VERSION_1                1
 #define MQSCO_VERSION_2                2
 #define MQSCO_VERSION_3                3
 #define MQSCO_VERSION_4                4
 #define MQSCO_VERSION_5                5
 #define MQSCO_CURRENT_VERSION          5

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQSCO_LENGTH_1                 536
#else
 #define MQSCO_LENGTH_1                 532
#endif
#if defined(MQ_64_BIT)
 #define MQSCO_LENGTH_2                 544
#else
 #define MQSCO_LENGTH_2                 540
#endif
#if defined(MQ_64_BIT)
 #define MQSCO_LENGTH_3                 560
#else
 #define MQSCO_LENGTH_3                 556
#endif
#if defined(MQ_64_BIT)
 #define MQSCO_LENGTH_4                 568
#else
 #define MQSCO_LENGTH_4                 560
#endif
#if defined(MQ_64_BIT)
 #define MQSCO_LENGTH_5                 632
#else
 #define MQSCO_LENGTH_5                 624
#endif
#if defined(MQ_64_BIT)
 #define MQSCO_CURRENT_LENGTH           632
#else
 #define MQSCO_CURRENT_LENGTH           624
#endif

 /* SuiteB Type */
 #define MQ_SUITE_B_NOT_AVAILABLE       0
 #define MQ_SUITE_B_NONE                1
 #define MQ_SUITE_B_128_BIT             2
 #define MQ_SUITE_B_192_BIT             4

 /* Key Reset Count */
 #define MQSCO_RESET_COUNT_DEFAULT      0

 /* Certificate Validation Policy Type */
 #define MQ_CERT_VAL_POLICY_DEFAULT     0
 #define MQ_CERT_VAL_POLICY_ANY         0
 #define MQ_CERT_VAL_POLICY_RFC5280     1

 /****************************************************************/
 /* Values Related to MQCSP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCSP_STRUC_ID                 "CSP "

 /* Structure Identifier (array form) */
 #define MQCSP_STRUC_ID_ARRAY           'C','S','P',' '

 /* Structure Version Number */
 #define MQCSP_VERSION_1                1
 #define MQCSP_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQCSP_LENGTH_1                 56
#else
 #define MQCSP_LENGTH_1                 48
#endif
#if defined(MQ_64_BIT)
 #define MQCSP_CURRENT_LENGTH           56
#else
 #define MQCSP_CURRENT_LENGTH           48
#endif

 /* Authentication Types */
 #define MQCSP_AUTH_NONE                0
 #define MQCSP_AUTH_USER_ID_AND_PWD     1

 /****************************************************************/
 /* Values Related to MQCNO Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCNO_STRUC_ID                 "CNO "

 /* Structure Identifier (array form) */
 #define MQCNO_STRUC_ID_ARRAY           'C','N','O',' '

 /* Structure Version Number */
 #define MQCNO_VERSION_1                1
 #define MQCNO_VERSION_2                2
 #define MQCNO_VERSION_3                3
 #define MQCNO_VERSION_4                4
 #define MQCNO_VERSION_5                5
 #define MQCNO_VERSION_6                6
 #define MQCNO_CURRENT_VERSION          6

 /* Structure Length */
 #define MQCNO_LENGTH_1                 12
#if defined(MQ_64_BIT)
 #define MQCNO_LENGTH_2                 24
#else
 #define MQCNO_LENGTH_2                 20
#endif
#if defined(MQ_64_BIT)
 #define MQCNO_LENGTH_3                 152
#else
 #define MQCNO_LENGTH_3                 148
#endif
#if defined(MQ_64_BIT)
 #define MQCNO_LENGTH_4                 168
#else
 #define MQCNO_LENGTH_4                 156
#endif
#if defined(MQ_64_BIT)
 #define MQCNO_LENGTH_5                 200
#else
 #define MQCNO_LENGTH_5                 188
#endif
#if defined(MQ_64_BIT)
 #define MQCNO_LENGTH_6                 224
#else
 #define MQCNO_LENGTH_6                 208
#endif
#if defined(MQ_64_BIT)
 #define MQCNO_CURRENT_LENGTH           224
#else
 #define MQCNO_CURRENT_LENGTH           208
#endif

 /* Connect Options */
 #define MQCNO_STANDARD_BINDING         0x00000000
 #define MQCNO_FASTPATH_BINDING         0x00000001
 #define MQCNO_SERIALIZE_CONN_TAG_Q_MGR 0x00000002
 #define MQCNO_SERIALIZE_CONN_TAG_QSG   0x00000004
 #define MQCNO_RESTRICT_CONN_TAG_Q_MGR  0x00000008
 #define MQCNO_RESTRICT_CONN_TAG_QSG    0x00000010
 #define MQCNO_HANDLE_SHARE_NONE        0x00000020
 #define MQCNO_HANDLE_SHARE_BLOCK       0x00000040
 #define MQCNO_HANDLE_SHARE_NO_BLOCK    0x00000080
 #define MQCNO_SHARED_BINDING           0x00000100
 #define MQCNO_ISOLATED_BINDING         0x00000200
 #define MQCNO_LOCAL_BINDING            0x00000400
 #define MQCNO_CLIENT_BINDING           0x00000800
 #define MQCNO_ACCOUNTING_MQI_ENABLED   0x00001000
 #define MQCNO_ACCOUNTING_MQI_DISABLED  0x00002000
 #define MQCNO_ACCOUNTING_Q_ENABLED     0x00004000
 #define MQCNO_ACCOUNTING_Q_DISABLED    0x00008000
 #define MQCNO_NO_CONV_SHARING          0x00010000
 #define MQCNO_ALL_CONVS_SHARE          0x00040000
 #define MQCNO_CD_FOR_OUTPUT_ONLY       0x00080000
 #define MQCNO_USE_CD_SELECTION         0x00100000
 #define MQCNO_RECONNECT_AS_DEF         0x00000000
 #define MQCNO_RECONNECT                0x01000000
 #define MQCNO_RECONNECT_DISABLED       0x02000000
 #define MQCNO_RECONNECT_Q_MGR          0x04000000
 #define MQCNO_ACTIVITY_TRACE_ENABLED   0x08000000
 #define MQCNO_ACTIVITY_TRACE_DISABLED  0x10000000
 #define MQCNO_NONE                     0x00000000

 /* Queue Manager Connection Tag */
 #define MQCT_NONE                      "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Queue Manager Connection Tag (array form) */
 #define MQCT_NONE_ARRAY                '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Connection Identifier */
 #define MQCONNID_NONE                  "\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Connection Identifier (array form) */
 #define MQCONNID_NONE_ARRAY            '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /****************************************************************/
 /* Values Related to MQDH Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQDH_STRUC_ID                  "DH  "

 /* Structure Identifier (array form) */
 #define MQDH_STRUC_ID_ARRAY            'D','H',' ',' '

 /* Structure Version Number */
 #define MQDH_VERSION_1                 1
 #define MQDH_CURRENT_VERSION           1

 /* Structure Length */
 #define MQDH_LENGTH_1                  48
 #define MQDH_CURRENT_LENGTH            48

 /* Flags */
 #define MQDHF_NEW_MSG_IDS              0x00000001
 #define MQDHF_NONE                     0x00000000

 /****************************************************************/
 /* Values Related to MQDLH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQDLH_STRUC_ID                 "DLH "

 /* Structure Identifier (array form) */
 #define MQDLH_STRUC_ID_ARRAY           'D','L','H',' '

 /* Structure Version Number */
 #define MQDLH_VERSION_1                1
 #define MQDLH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQDLH_LENGTH_1                 172
 #define MQDLH_CURRENT_LENGTH           172

 /****************************************************************/
 /* Values Related to MQDMHO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQDMHO_STRUC_ID                "DMHO"

 /* Structure Identifier (array form) */
 #define MQDMHO_STRUC_ID_ARRAY          'D','M','H','O'

 /* Structure Version Number */
 #define MQDMHO_VERSION_1               1
 #define MQDMHO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQDMHO_LENGTH_1                12
 #define MQDMHO_CURRENT_LENGTH          12

 /* Delete Message Handle Options */
 #define MQDMHO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQDMPO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQDMPO_STRUC_ID                "DMPO"

 /* Structure Identifier (array form) */
 #define MQDMPO_STRUC_ID_ARRAY          'D','M','P','O'

 /* Structure Version Number */
 #define MQDMPO_VERSION_1               1
 #define MQDMPO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQDMPO_LENGTH_1                12
 #define MQDMPO_CURRENT_LENGTH          12

 /* Delete Message Property Options */
 #define MQDMPO_DEL_FIRST               0x00000000
 #define MQDMPO_DEL_PROP_UNDER_CURSOR   0x00000001
 #define MQDMPO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQGMO Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQGMO_STRUC_ID                 "GMO "

 /* Structure Identifier (array form) */
 #define MQGMO_STRUC_ID_ARRAY           'G','M','O',' '

 /* Structure Version Number */
 #define MQGMO_VERSION_1                1
 #define MQGMO_VERSION_2                2
 #define MQGMO_VERSION_3                3
 #define MQGMO_VERSION_4                4
 #define MQGMO_CURRENT_VERSION          4

 /* Structure Length */
 #define MQGMO_LENGTH_1                 72
 #define MQGMO_LENGTH_2                 80
 #define MQGMO_LENGTH_3                 100
 #define MQGMO_LENGTH_4                 112
 #define MQGMO_CURRENT_LENGTH           112

 /* Get Message Options */
 #define MQGMO_WAIT                     0x00000001
 #define MQGMO_NO_WAIT                  0x00000000
 #define MQGMO_SET_SIGNAL               0x00000008
 #define MQGMO_FAIL_IF_QUIESCING        0x00002000
 #define MQGMO_SYNCPOINT                0x00000002
 #define MQGMO_SYNCPOINT_IF_PERSISTENT  0x00001000
 #define MQGMO_NO_SYNCPOINT             0x00000004
 #define MQGMO_MARK_SKIP_BACKOUT        0x00000080
 #define MQGMO_BROWSE_FIRST             0x00000010
 #define MQGMO_BROWSE_NEXT              0x00000020
 #define MQGMO_BROWSE_MSG_UNDER_CURSOR  0x00000800
 #define MQGMO_MSG_UNDER_CURSOR         0x00000100
 #define MQGMO_LOCK                     0x00000200
 #define MQGMO_UNLOCK                   0x00000400
 #define MQGMO_ACCEPT_TRUNCATED_MSG     0x00000040
 #define MQGMO_CONVERT                  0x00004000
 #define MQGMO_LOGICAL_ORDER            0x00008000
 #define MQGMO_COMPLETE_MSG             0x00010000
 #define MQGMO_ALL_MSGS_AVAILABLE       0x00020000
 #define MQGMO_ALL_SEGMENTS_AVAILABLE   0x00040000
 #define MQGMO_MARK_BROWSE_HANDLE       0x00100000
 #define MQGMO_MARK_BROWSE_CO_OP        0x00200000
 #define MQGMO_UNMARK_BROWSE_CO_OP      0x00400000
 #define MQGMO_UNMARK_BROWSE_HANDLE     0x00800000
 #define MQGMO_UNMARKED_BROWSE_MSG      0x01000000
 #define MQGMO_PROPERTIES_FORCE_MQRFH2  0x02000000
 #define MQGMO_NO_PROPERTIES            0x04000000
 #define MQGMO_PROPERTIES_IN_HANDLE     0x08000000
 #define MQGMO_PROPERTIES_COMPATIBILITY 0x10000000
 #define MQGMO_PROPERTIES_AS_Q_DEF      0x00000000
 #define MQGMO_NONE                     0x00000000
 #define MQGMO_BROWSE_HANDLE            ( MQGMO_BROWSE_FIRST \
                                        | MQGMO_UNMARKED_BROWSE_MSG \
                                        | MQGMO_MARK_BROWSE_HANDLE )
 #define MQGMO_BROWSE_CO_OP             ( MQGMO_BROWSE_FIRST \
                                        | MQGMO_UNMARKED_BROWSE_MSG \
                                        | MQGMO_MARK_BROWSE_CO_OP )

 /* Wait Interval */
 #define MQWI_UNLIMITED                 (-1)

 /* Signal Values */
 #define MQEC_MSG_ARRIVED               2
 #define MQEC_WAIT_INTERVAL_EXPIRED     3
 #define MQEC_WAIT_CANCELED             4
 #define MQEC_Q_MGR_QUIESCING           5
 #define MQEC_CONNECTION_QUIESCING      6

 /* Match Options */
 #define MQMO_MATCH_MSG_ID              0x00000001
 #define MQMO_MATCH_CORREL_ID           0x00000002
 #define MQMO_MATCH_GROUP_ID            0x00000004
 #define MQMO_MATCH_MSG_SEQ_NUMBER      0x00000008
 #define MQMO_MATCH_OFFSET              0x00000010
 #define MQMO_MATCH_MSG_TOKEN           0x00000020
 #define MQMO_NONE                      0x00000000

 /* Group Status */
 #define MQGS_NOT_IN_GROUP              ' '
 #define MQGS_MSG_IN_GROUP              'G'
 #define MQGS_LAST_MSG_IN_GROUP         'L'

 /* Segment Status */
 #define MQSS_NOT_A_SEGMENT             ' '
 #define MQSS_SEGMENT                   'S'
 #define MQSS_LAST_SEGMENT              'L'

 /* Segmentation */
 #define MQSEG_INHIBITED                ' '
 #define MQSEG_ALLOWED                  'A'

 /* Message Token */
 #define MQMTOK_NONE                    "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Message Token (array form) */
 #define MQMTOK_NONE_ARRAY              '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Returned Length */
 #define MQRL_UNDEFINED                 (-1)

 /****************************************************************/
 /* Values Related to MQIIH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQIIH_STRUC_ID                 "IIH "

 /* Structure Identifier (array form) */
 #define MQIIH_STRUC_ID_ARRAY           'I','I','H',' '

 /* Structure Version Number */
 #define MQIIH_VERSION_1                1
 #define MQIIH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQIIH_LENGTH_1                 84
 #define MQIIH_CURRENT_LENGTH           84

 /* Flags */
 #define MQIIH_NONE                     0x00000000
 #define MQIIH_PASS_EXPIRATION          0x00000001
 #define MQIIH_UNLIMITED_EXPIRATION     0x00000000
 #define MQIIH_REPLY_FORMAT_NONE        0x00000008
 #define MQIIH_IGNORE_PURG              0x00000010
 #define MQIIH_CM0_REQUEST_RESPONSE     0x00000020

 /* Authenticator */
 #define MQIAUT_NONE                    "        "

 /* Authenticator (array form) */
 #define MQIAUT_NONE_ARRAY              ' ',' ',' ',' ',' ',' ',' ',' '

 /* Transaction Instance Identifier */
 #define MQITII_NONE                    "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Transaction Instance Identifier (array form) */
 #define MQITII_NONE_ARRAY              '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Transaction States */
 #define MQITS_IN_CONVERSATION          'C'
 #define MQITS_NOT_IN_CONVERSATION      ' '
 #define MQITS_ARCHITECTED              'A'

 /* Commit Modes */
 #define MQICM_COMMIT_THEN_SEND         '0'
 #define MQICM_SEND_THEN_COMMIT         '1'

 /* Security Scopes */
 #define MQISS_CHECK                    'C'
 #define MQISS_FULL                     'F'

 /****************************************************************/
 /* Values Related to MQIMPO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQIMPO_STRUC_ID                "IMPO"

 /* Structure Identifier (array form) */
 #define MQIMPO_STRUC_ID_ARRAY          'I','M','P','O'

 /* Structure Version Number */
 #define MQIMPO_VERSION_1               1
 #define MQIMPO_CURRENT_VERSION         1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQIMPO_LENGTH_1                64
#else
 #define MQIMPO_LENGTH_1                60
#endif
#if defined(MQ_64_BIT)
 #define MQIMPO_CURRENT_LENGTH          64
#else
 #define MQIMPO_CURRENT_LENGTH          60
#endif

 /* Inquire Message Property Options */
 #define MQIMPO_CONVERT_TYPE            0x00000002
 #define MQIMPO_QUERY_LENGTH            0x00000004
 #define MQIMPO_INQ_FIRST               0x00000000
 #define MQIMPO_INQ_NEXT                0x00000008
 #define MQIMPO_INQ_PROP_UNDER_CURSOR   0x00000010
 #define MQIMPO_CONVERT_VALUE           0x00000020
 #define MQIMPO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQMD Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQMD_STRUC_ID                  "MD  "

 /* Structure Identifier (array form) */
 #define MQMD_STRUC_ID_ARRAY            'M','D',' ',' '

 /* Structure Version Number */
 #define MQMD_VERSION_1                 1
 #define MQMD_VERSION_2                 2
 #define MQMD_CURRENT_VERSION           2

 /* Structure Length */
 #define MQMD_LENGTH_1                  324
 #define MQMD_LENGTH_2                  364
 #define MQMD_CURRENT_LENGTH            364

 /* Report Options */
 #define MQRO_EXCEPTION                 0x01000000
 #define MQRO_EXCEPTION_WITH_DATA       0x03000000
 #define MQRO_EXCEPTION_WITH_FULL_DATA  0x07000000
 #define MQRO_EXPIRATION                0x00200000
 #define MQRO_EXPIRATION_WITH_DATA      0x00600000
 #define MQRO_EXPIRATION_WITH_FULL_DATA 0x00E00000
 #define MQRO_COA                       0x00000100
 #define MQRO_COA_WITH_DATA             0x00000300
 #define MQRO_COA_WITH_FULL_DATA        0x00000700
 #define MQRO_COD                       0x00000800
 #define MQRO_COD_WITH_DATA             0x00001800
 #define MQRO_COD_WITH_FULL_DATA        0x00003800
 #define MQRO_PAN                       0x00000001
 #define MQRO_NAN                       0x00000002
 #define MQRO_ACTIVITY                  0x00000004
 #define MQRO_NEW_MSG_ID                0x00000000
 #define MQRO_PASS_MSG_ID               0x00000080
 #define MQRO_COPY_MSG_ID_TO_CORREL_ID  0x00000000
 #define MQRO_PASS_CORREL_ID            0x00000040
 #define MQRO_DEAD_LETTER_Q             0x00000000
 #define MQRO_DISCARD_MSG               0x08000000
 #define MQRO_PASS_DISCARD_AND_EXPIRY   0x00004000
 #define MQRO_NONE                      0x00000000

 /* Report Options Masks */
 #define MQRO_REJECT_UNSUP_MASK         0x101C0000
 #define MQRO_ACCEPT_UNSUP_MASK         0xEFE000FF
 #define MQRO_ACCEPT_UNSUP_IF_XMIT_MASK 0x0003FF00

 /* Message Types */
 #define MQMT_SYSTEM_FIRST              1
 #define MQMT_REQUEST                   1
 #define MQMT_REPLY                     2
 #define MQMT_DATAGRAM                  8
 #define MQMT_REPORT                    4
 #define MQMT_MQE_FIELDS_FROM_MQE       112
 #define MQMT_MQE_FIELDS                113
 #define MQMT_SYSTEM_LAST               65535
 #define MQMT_APPL_FIRST                65536
 #define MQMT_APPL_LAST                 999999999

 /* Expiry */
 #define MQEI_UNLIMITED                 (-1)

 /* Feedback Values */
 #define MQFB_NONE                      0
 #define MQFB_SYSTEM_FIRST              1
 #define MQFB_QUIT                      256
 #define MQFB_EXPIRATION                258
 #define MQFB_COA                       259
 #define MQFB_COD                       260
 #define MQFB_CHANNEL_COMPLETED         262
 #define MQFB_CHANNEL_FAIL_RETRY        263
 #define MQFB_CHANNEL_FAIL              264
 #define MQFB_APPL_CANNOT_BE_STARTED    265
 #define MQFB_TM_ERROR                  266
 #define MQFB_APPL_TYPE_ERROR           267
 #define MQFB_STOPPED_BY_MSG_EXIT       268
 #define MQFB_ACTIVITY                  269
 #define MQFB_XMIT_Q_MSG_ERROR          271
 #define MQFB_PAN                       275
 #define MQFB_NAN                       276
 #define MQFB_STOPPED_BY_CHAD_EXIT      277
 #define MQFB_STOPPED_BY_PUBSUB_EXIT    279
 #define MQFB_NOT_A_REPOSITORY_MSG      280
 #define MQFB_BIND_OPEN_CLUSRCVR_DEL    281
 #define MQFB_MAX_ACTIVITIES            282
 #define MQFB_NOT_FORWARDED             283
 #define MQFB_NOT_DELIVERED             284
 #define MQFB_UNSUPPORTED_FORWARDING    285
 #define MQFB_UNSUPPORTED_DELIVERY      286
 #define MQFB_DATA_LENGTH_ZERO          291
 #define MQFB_DATA_LENGTH_NEGATIVE      292
 #define MQFB_DATA_LENGTH_TOO_BIG       293
 #define MQFB_BUFFER_OVERFLOW           294
 #define MQFB_LENGTH_OFF_BY_ONE         295
 #define MQFB_IIH_ERROR                 296
 #define MQFB_NOT_AUTHORIZED_FOR_IMS    298
 #define MQFB_IMS_ERROR                 300
 #define MQFB_IMS_FIRST                 301
 #define MQFB_IMS_LAST                  399
 #define MQFB_CICS_INTERNAL_ERROR       401
 #define MQFB_CICS_NOT_AUTHORIZED       402
 #define MQFB_CICS_BRIDGE_FAILURE       403
 #define MQFB_CICS_CORREL_ID_ERROR      404
 #define MQFB_CICS_CCSID_ERROR          405
 #define MQFB_CICS_ENCODING_ERROR       406
 #define MQFB_CICS_CIH_ERROR            407
 #define MQFB_CICS_UOW_ERROR            408
 #define MQFB_CICS_COMMAREA_ERROR       409
 #define MQFB_CICS_APPL_NOT_STARTED     410
 #define MQFB_CICS_APPL_ABENDED         411
 #define MQFB_CICS_DLQ_ERROR            412
 #define MQFB_CICS_UOW_BACKED_OUT       413
 #define MQFB_PUBLICATIONS_ON_REQUEST   501
 #define MQFB_SUBSCRIBER_IS_PUBLISHER   502
 #define MQFB_MSG_SCOPE_MISMATCH        503
 #define MQFB_SELECTOR_MISMATCH         504
 #define MQFB_NOT_A_GROUPUR_MSG         505
 #define MQFB_IMS_NACK_1A_REASON_FIRST  600
 #define MQFB_IMS_NACK_1A_REASON_LAST   855
 #define MQFB_SYSTEM_LAST               65535
 #define MQFB_APPL_FIRST                65536
 #define MQFB_APPL_LAST                 999999999

 /* Encoding */
 #define MQENC_NATIVE                   0x00000222

 /* Encoding Masks */
 #define MQENC_INTEGER_MASK             0x0000000F
 #define MQENC_DECIMAL_MASK             0x000000F0
 #define MQENC_FLOAT_MASK               0x00000F00
 #define MQENC_RESERVED_MASK            0xFFFFF000

 /* Encodings for Binary Integers */
 #define MQENC_INTEGER_UNDEFINED        0x00000000
 #define MQENC_INTEGER_NORMAL           0x00000001
 #define MQENC_INTEGER_REVERSED         0x00000002

 /* Encodings for Packed Decimal Integers */
 #define MQENC_DECIMAL_UNDEFINED        0x00000000
 #define MQENC_DECIMAL_NORMAL           0x00000010
 #define MQENC_DECIMAL_REVERSED         0x00000020

 /* Encodings for Floating Point Numbers */
 #define MQENC_FLOAT_UNDEFINED          0x00000000
 #define MQENC_FLOAT_IEEE_NORMAL        0x00000100
 #define MQENC_FLOAT_IEEE_REVERSED      0x00000200
 #define MQENC_FLOAT_S390               0x00000300
 #define MQENC_FLOAT_TNS                0x00000400

 /* Encodings for Multicast */
 #define MQENC_NORMAL                   ( MQENC_FLOAT_IEEE_NORMAL \
                                        | MQENC_DECIMAL_NORMAL \
                                        | MQENC_INTEGER_NORMAL )
 #define MQENC_REVERSED                 ( MQENC_FLOAT_IEEE_REVERSED \
                                        | MQENC_DECIMAL_REVERSED \
                                        | MQENC_INTEGER_REVERSED )
 #define MQENC_S390                     ( MQENC_FLOAT_S390 \
                                        | MQENC_DECIMAL_NORMAL \
                                        | MQENC_INTEGER_NORMAL )
 #define MQENC_TNS                      ( MQENC_FLOAT_TNS \
                                        | MQENC_DECIMAL_NORMAL \
                                        | MQENC_INTEGER_NORMAL )
 #define MQENC_AS_PUBLISHED             (-1)

 /* Coded Character Set Identifiers */
 #define MQCCSI_UNDEFINED               0
 #define MQCCSI_DEFAULT                 0
 #define MQCCSI_Q_MGR                   0
 #define MQCCSI_INHERIT                 (-2)
 #define MQCCSI_EMBEDDED                (-1)
 #if !defined(MQCCSI_APPL)
 #define MQCCSI_APPL                    (-3)
 #endif
 #define MQCCSI_AS_PUBLISHED            (-4)

 /* Formats */
 #define MQFMT_NONE                     "        "
 #define MQFMT_ADMIN                    "MQADMIN "
 #define MQFMT_AMQP                     "MQAMQP  "
 #define MQFMT_CHANNEL_COMPLETED        "MQCHCOM "
 #define MQFMT_CICS                     "MQCICS  "
 #define MQFMT_COMMAND_1                "MQCMD1  "
 #define MQFMT_COMMAND_2                "MQCMD2  "
 #define MQFMT_DEAD_LETTER_HEADER       "MQDEAD  "
 #define MQFMT_DIST_HEADER              "MQHDIST "
 #define MQFMT_EMBEDDED_PCF             "MQHEPCF "
 #define MQFMT_EVENT                    "MQEVENT "
 #define MQFMT_IMS                      "MQIMS   "
 #define MQFMT_IMS_VAR_STRING           "MQIMSVS "
 #define MQFMT_MD_EXTENSION             "MQHMDE  "
 #define MQFMT_PCF                      "MQPCF   "
 #define MQFMT_REF_MSG_HEADER           "MQHREF  "
 #define MQFMT_RF_HEADER                "MQHRF   "
 #define MQFMT_RF_HEADER_1              "MQHRF   "
 #define MQFMT_RF_HEADER_2              "MQHRF2  "
 #define MQFMT_STRING                   "MQSTR   "
 #define MQFMT_TRIGGER                  "MQTRIG  "
 #define MQFMT_WORK_INFO_HEADER         "MQHWIH  "
 #define MQFMT_XMIT_Q_HEADER            "MQXMIT  "

 /* Formats (array form) */
 #define MQFMT_NONE_ARRAY               ' ',' ',' ',' ',' ',' ',' ',' '
 #define MQFMT_ADMIN_ARRAY              'M','Q','A','D','M','I','N',' '
 #define MQFMT_AMQP_ARRAY               'M','Q','A','M','Q','P',' ',' '
 #define MQFMT_CHANNEL_COMPLETED_ARRAY  'M','Q','C','H','C','O','M',' '
 #define MQFMT_CICS_ARRAY               'M','Q','C','I','C','S',' ',' '
 #define MQFMT_COMMAND_1_ARRAY          'M','Q','C','M','D','1',' ',' '
 #define MQFMT_COMMAND_2_ARRAY          'M','Q','C','M','D','2',' ',' '
 #define MQFMT_DEAD_LETTER_HEADER_ARRAY 'M','Q','D','E','A','D',' ',' '
 #define MQFMT_DIST_HEADER_ARRAY        'M','Q','H','D','I','S','T',' '
 #define MQFMT_EMBEDDED_PCF_ARRAY       'M','Q','H','E','P','C','F',' '
 #define MQFMT_EVENT_ARRAY              'M','Q','E','V','E','N','T',' '
 #define MQFMT_IMS_ARRAY                'M','Q','I','M','S',' ',' ',' '
 #define MQFMT_IMS_VAR_STRING_ARRAY     'M','Q','I','M','S','V','S',' '
 #define MQFMT_MD_EXTENSION_ARRAY       'M','Q','H','M','D','E',' ',' '
 #define MQFMT_PCF_ARRAY                'M','Q','P','C','F',' ',' ',' '
 #define MQFMT_REF_MSG_HEADER_ARRAY     'M','Q','H','R','E','F',' ',' '
 #define MQFMT_RF_HEADER_ARRAY          'M','Q','H','R','F',' ',' ',' '
 #define MQFMT_RF_HEADER_1_ARRAY        'M','Q','H','R','F',' ',' ',' '
 #define MQFMT_RF_HEADER_2_ARRAY        'M','Q','H','R','F','2',' ',' '
 #define MQFMT_STRING_ARRAY             'M','Q','S','T','R',' ',' ',' '
 #define MQFMT_TRIGGER_ARRAY            'M','Q','T','R','I','G',' ',' '
 #define MQFMT_WORK_INFO_HEADER_ARRAY   'M','Q','H','W','I','H',' ',' '
 #define MQFMT_XMIT_Q_HEADER_ARRAY      'M','Q','X','M','I','T',' ',' '

 /* Priority */
 #define MQPRI_PRIORITY_AS_Q_DEF        (-1)
 #define MQPRI_PRIORITY_AS_PARENT       (-2)
 #define MQPRI_PRIORITY_AS_PUBLISHED    (-3)
 #define MQPRI_PRIORITY_AS_TOPIC_DEF    (-1)

 /* Persistence Values */
 #define MQPER_PERSISTENCE_AS_PARENT    (-1)
 #define MQPER_NOT_PERSISTENT           0
 #define MQPER_PERSISTENT               1
 #define MQPER_PERSISTENCE_AS_Q_DEF     2
 #define MQPER_PERSISTENCE_AS_TOPIC_DEF 2

 /* Put Response Values */
 #define MQPRT_RESPONSE_AS_PARENT       0
 #define MQPRT_SYNC_RESPONSE            1
 #define MQPRT_ASYNC_RESPONSE           2

 /* Message Identifier */
 #define MQMI_NONE                      "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Message Identifier (array form) */
 #define MQMI_NONE_ARRAY                '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Correlation Identifier */
 #define MQCI_NONE                      "\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0"
 #define MQCI_NEW_SESSION               "\x41\x4D\x51\x21\x4E\x45\x57\x5F"\
                                        "\x53\x45\x53\x53\x49\x4F\x4E\x5F"\
                                        "\x43\x4F\x52\x52\x45\x4C\x49\x44"

 /* Correlation Identifier (array form) */
 #define MQCI_NONE_ARRAY                '\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0'
 #define MQCI_NEW_SESSION_ARRAY         '\x41','\x4D','\x51','\x21',\
                                        '\x4E','\x45','\x57','\x5F',\
                                        '\x53','\x45','\x53','\x53',\
                                        '\x49','\x4F','\x4E','\x5F',\
                                        '\x43','\x4F','\x52','\x52',\
                                        '\x45','\x4C','\x49','\x44'

 /* Accounting Token */
 #define MQACT_NONE                     "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Accounting Token (array form) */
 #define MQACT_NONE_ARRAY               '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Accounting Token Types */
 #define MQACTT_UNKNOWN                 '\x00'
 #define MQACTT_CICS_LUOW_ID            '\x01'
 #define MQACTT_OS2_DEFAULT             '\x04'
 #define MQACTT_DOS_DEFAULT             '\x05'
 #define MQACTT_UNIX_NUMERIC_ID         '\x06'
 #define MQACTT_OS400_ACCOUNT_TOKEN     '\x08'
 #define MQACTT_WINDOWS_DEFAULT         '\x09'
 #define MQACTT_NT_SECURITY_ID          '\x0B'
 #define MQACTT_AZUREAD_SECURITY_ID     '\x0C'
 #define MQACTT_MS_ACC_AUTH_SECURITY_ID '\x0D'
 #define MQACTT_USER                    '\x19'

 /* Put Application Types */
 #define MQAT_UNKNOWN                   (-1)
 #define MQAT_NO_CONTEXT                0
 #define MQAT_CICS                      1
 #define MQAT_MVS                       2
 #define MQAT_OS390                     2
 #define MQAT_ZOS                       2
 #define MQAT_IMS                       3
 #define MQAT_OS2                       4
 #define MQAT_DOS                       5
 #define MQAT_AIX                       6
 #define MQAT_UNIX                      6
 #define MQAT_QMGR                      7
 #define MQAT_OS400                     8
 #define MQAT_WINDOWS                   9
 #define MQAT_CICS_VSE                  10
 #define MQAT_WINDOWS_NT                11
 #define MQAT_VMS                       12
 #define MQAT_GUARDIAN                  13
 #define MQAT_NSK                       13
 #define MQAT_VOS                       14
 #define MQAT_OPEN_TP1                  15
 #define MQAT_VM                        18
 #define MQAT_IMS_BRIDGE                19
 #define MQAT_XCF                       20
 #define MQAT_CICS_BRIDGE               21
 #define MQAT_NOTES_AGENT               22
 #define MQAT_TPF                       23
 #define MQAT_USER                      25
 #define MQAT_BROKER                    26
 #define MQAT_QMGR_PUBLISH              26
 #define MQAT_JAVA                      28
 #define MQAT_DQM                       29
 #define MQAT_CHANNEL_INITIATOR         30
 #define MQAT_WLM                       31
 #define MQAT_BATCH                     32
 #define MQAT_RRS_BATCH                 33
 #define MQAT_SIB                       34
 #define MQAT_SYSTEM_EXTENSION          35
 #define MQAT_MCAST_PUBLISH             36
 #define MQAT_AMQP                      37
 #define MQAT_DEFAULT                   6
 #define MQAT_USER_FIRST                65536
 #define MQAT_USER_LAST                 999999999

 /* Group Identifier */
 #define MQGI_NONE                      "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Group Identifier (array form) */
 #define MQGI_NONE_ARRAY                '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Message Flags */
 #define MQMF_SEGMENTATION_INHIBITED    0x00000000
 #define MQMF_SEGMENTATION_ALLOWED      0x00000001
 #define MQMF_MSG_IN_GROUP              0x00000008
 #define MQMF_LAST_MSG_IN_GROUP         0x00000010
 #define MQMF_SEGMENT                   0x00000002
 #define MQMF_LAST_SEGMENT              0x00000004
 #define MQMF_NONE                      0x00000000

 /* Message Flags Masks */
 #define MQMF_REJECT_UNSUP_MASK         0x00000FFF
 #define MQMF_ACCEPT_UNSUP_MASK         0xFFF00000
 #define MQMF_ACCEPT_UNSUP_IF_XMIT_MASK 0x000FF000

 /* Original Length */
 #define MQOL_UNDEFINED                 (-1)

 /****************************************************************/
 /* Values Related to MQMDE Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQMDE_STRUC_ID                 "MDE "

 /* Structure Identifier (array form) */
 #define MQMDE_STRUC_ID_ARRAY           'M','D','E',' '

 /* Structure Version Number */
 #define MQMDE_VERSION_2                2
 #define MQMDE_CURRENT_VERSION          2

 /* Structure Length */
 #define MQMDE_LENGTH_2                 72
 #define MQMDE_CURRENT_LENGTH           72

 /* Flags */
 #define MQMDEF_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQMD1 Structure                            */
 /****************************************************************/

 /* Structure Length */
 #define MQMD1_LENGTH_1                 324
 #define MQMD1_CURRENT_LENGTH           324

 /****************************************************************/
 /* Values Related to MQMD2 Structure                            */
 /****************************************************************/

 /* Structure Length */
 #define MQMD2_LENGTH_1                 324
 #define MQMD2_LENGTH_2                 364
 #define MQMD2_CURRENT_LENGTH           364

 /****************************************************************/
 /* Values Related to MQMHBO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQMHBO_STRUC_ID                "MHBO"

 /* Structure Identifier (array form) */
 #define MQMHBO_STRUC_ID_ARRAY          'M','H','B','O'

 /* Structure Version Number */
 #define MQMHBO_VERSION_1               1
 #define MQMHBO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQMHBO_LENGTH_1                12
 #define MQMHBO_CURRENT_LENGTH          12

 /* Message Handle To Buffer Options */
 #define MQMHBO_PROPERTIES_IN_MQRFH2    0x00000001
 #define MQMHBO_DELETE_PROPERTIES       0x00000002
 #define MQMHBO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQOD Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQOD_STRUC_ID                  "OD  "

 /* Structure Identifier (array form) */
 #define MQOD_STRUC_ID_ARRAY            'O','D',' ',' '

 /* Structure Version Number */
 #define MQOD_VERSION_1                 1
 #define MQOD_VERSION_2                 2
 #define MQOD_VERSION_3                 3
 #define MQOD_VERSION_4                 4
 #define MQOD_CURRENT_VERSION           4

 /* Structure Length */
 #define MQOD_LENGTH_1                  168
#if defined(MQ_64_BIT)
 #define MQOD_LENGTH_2                  208
#else
 #define MQOD_LENGTH_2                  200
#endif
#if defined(MQ_64_BIT)
 #define MQOD_LENGTH_3                  344
#else
 #define MQOD_LENGTH_3                  336
#endif
#if defined(MQ_64_BIT)
 #define MQOD_LENGTH_4                  424
#else
 #define MQOD_LENGTH_4                  400
#endif
#if defined(MQ_64_BIT)
 #define MQOD_CURRENT_LENGTH            424
#else
 #define MQOD_CURRENT_LENGTH            400
#endif

 /* Obsolete DB2 Messages options on Inquire Group */
 #define MQOM_NO                        0
 #define MQOM_YES                       1

 /* Object Types */
 #define MQOT_NONE                      0
 #define MQOT_Q                         1
 #define MQOT_NAMELIST                  2
 #define MQOT_PROCESS                   3
 #define MQOT_STORAGE_CLASS             4
 #define MQOT_Q_MGR                     5
 #define MQOT_CHANNEL                   6
 #define MQOT_AUTH_INFO                 7
 #define MQOT_TOPIC                     8
 #define MQOT_COMM_INFO                 9
 #define MQOT_CF_STRUC                  10
 #define MQOT_LISTENER                  11
 #define MQOT_SERVICE                   12
 #define MQOT_RESERVED_1                999

 /* Extended Object Types */
 #define MQOT_ALL                       1001
 #define MQOT_ALIAS_Q                   1002
 #define MQOT_MODEL_Q                   1003
 #define MQOT_LOCAL_Q                   1004
 #define MQOT_REMOTE_Q                  1005
 #define MQOT_SENDER_CHANNEL            1007
 #define MQOT_SERVER_CHANNEL            1008
 #define MQOT_REQUESTER_CHANNEL         1009
 #define MQOT_RECEIVER_CHANNEL          1010
 #define MQOT_CURRENT_CHANNEL           1011
 #define MQOT_SAVED_CHANNEL             1012
 #define MQOT_SVRCONN_CHANNEL           1013
 #define MQOT_CLNTCONN_CHANNEL          1014
 #define MQOT_SHORT_CHANNEL             1015
 #define MQOT_CHLAUTH                   1016
 #define MQOT_REMOTE_Q_MGR_NAME         1017
 #define MQOT_PROT_POLICY               1019
 #define MQOT_TT_CHANNEL                1020
 #define MQOT_AMQP_CHANNEL              1021
 #define MQOT_AUTH_REC                  1022

 /****************************************************************/
 /* Values Related to MQPD Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQPD_STRUC_ID                  "PD  "

 /* Structure Identifier (array form) */
 #define MQPD_STRUC_ID_ARRAY            'P','D',' ',' '

 /* Structure Version Number */
 #define MQPD_VERSION_1                 1
 #define MQPD_CURRENT_VERSION           1

 /* Structure Length */
 #define MQPD_LENGTH_1                  24
 #define MQPD_CURRENT_LENGTH            24

 /* Property Descriptor Options */
 #define MQPD_NONE                      0x00000000

 /* Property Support Options */
 #define MQPD_SUPPORT_OPTIONAL          0x00000001
 #define MQPD_SUPPORT_REQUIRED          0x00100000
 #define MQPD_SUPPORT_REQUIRED_IF_LOCAL 0x00000400
 #define MQPD_REJECT_UNSUP_MASK         0xFFF00000
 #define MQPD_ACCEPT_UNSUP_IF_XMIT_MASK 0x000FFC00
 #define MQPD_ACCEPT_UNSUP_MASK         0x000003FF

 /* Property Context */
 #define MQPD_NO_CONTEXT                0x00000000
 #define MQPD_USER_CONTEXT              0x00000001

 /* Property Copy Options */
 #define MQCOPY_NONE                    0x00000000
 #define MQCOPY_ALL                     0x00000001
 #define MQCOPY_FORWARD                 0x00000002
 #define MQCOPY_PUBLISH                 0x00000004
 #define MQCOPY_REPLY                   0x00000008
 #define MQCOPY_REPORT                  0x00000010
 #define MQCOPY_DEFAULT                 0x00000016

 /****************************************************************/
 /* Values Related to MQPMO Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQPMO_STRUC_ID                 "PMO "

 /* Structure Identifier (array form) */
 #define MQPMO_STRUC_ID_ARRAY           'P','M','O',' '

 /* Structure Version Number */
 #define MQPMO_VERSION_1                1
 #define MQPMO_VERSION_2                2
 #define MQPMO_VERSION_3                3
 #define MQPMO_CURRENT_VERSION          3

 /* Structure Length */
 #define MQPMO_LENGTH_1                 128
#if defined(MQ_64_BIT)
 #define MQPMO_LENGTH_2                 160
#else
 #define MQPMO_LENGTH_2                 152
#endif
#if defined(MQ_64_BIT)
 #define MQPMO_LENGTH_3                 184
#else
 #define MQPMO_LENGTH_3                 176
#endif
#if defined(MQ_64_BIT)
 #define MQPMO_CURRENT_LENGTH           184
#else
 #define MQPMO_CURRENT_LENGTH           176
#endif

 /* Put Message Options */
 #define MQPMO_SYNCPOINT                0x00000002
 #define MQPMO_NO_SYNCPOINT             0x00000004
 #define MQPMO_DEFAULT_CONTEXT          0x00000020
 #define MQPMO_NEW_MSG_ID               0x00000040
 #define MQPMO_NEW_CORREL_ID            0x00000080
 #define MQPMO_PASS_IDENTITY_CONTEXT    0x00000100
 #define MQPMO_PASS_ALL_CONTEXT         0x00000200
 #define MQPMO_SET_IDENTITY_CONTEXT     0x00000400
 #define MQPMO_SET_ALL_CONTEXT          0x00000800
 #define MQPMO_ALTERNATE_USER_AUTHORITY 0x00001000
 #define MQPMO_FAIL_IF_QUIESCING        0x00002000
 #define MQPMO_NO_CONTEXT               0x00004000
 #define MQPMO_LOGICAL_ORDER            0x00008000
 #define MQPMO_ASYNC_RESPONSE           0x00010000
 #define MQPMO_SYNC_RESPONSE            0x00020000
 #define MQPMO_RESOLVE_LOCAL_Q          0x00040000
 #define MQPMO_WARN_IF_NO_SUBS_MATCHED  0x00080000
 #define MQPMO_RETAIN                   0x00200000
 #define MQPMO_MD_FOR_OUTPUT_ONLY       0x00800000
 #define MQPMO_SCOPE_QMGR               0x04000000
 #define MQPMO_SUPPRESS_REPLYTO         0x08000000
 #define MQPMO_NOT_OWN_SUBS             0x10000000
 #define MQPMO_RESPONSE_AS_Q_DEF        0x00000000
 #define MQPMO_RESPONSE_AS_TOPIC_DEF    0x00000000
 #define MQPMO_NONE                     0x00000000

 /* Put Message Options for publish mask */
 #define MQPMO_PUB_OPTIONS_MASK         0x00200000

 /* Put Message Record Fields */
 #define MQPMRF_MSG_ID                  0x00000001
 #define MQPMRF_CORREL_ID               0x00000002
 #define MQPMRF_GROUP_ID                0x00000004
 #define MQPMRF_FEEDBACK                0x00000008
 #define MQPMRF_ACCOUNTING_TOKEN        0x00000010
 #define MQPMRF_NONE                    0x00000000

 /* Action */
 #define MQACTP_NEW                     0
 #define MQACTP_FORWARD                 1
 #define MQACTP_REPLY                   2
 #define MQACTP_REPORT                  3

 /****************************************************************/
 /* Values Related to MQRFH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQRFH_STRUC_ID                 "RFH "

 /* Structure Identifier (array form) */
 #define MQRFH_STRUC_ID_ARRAY           'R','F','H',' '

 /* Structure Version Number */
 #define MQRFH_VERSION_1                1
 #define MQRFH_VERSION_2                2

 /* Structure Length */
 #define MQRFH_STRUC_LENGTH_FIXED       32
 #define MQRFH_STRUC_LENGTH_FIXED_2     36
 #define MQRFH_LENGTH_1                 32
 #define MQRFH_CURRENT_LENGTH           32

 /* Flags */
 #define MQRFH_NONE                     0x00000000
 #define MQRFH_NO_FLAGS                 0
 #define MQRFH_FLAGS_RESTRICTED_MASK    0xFFFF0000
 /* MQRFH2 flags in the restricted mask are reserved for MQ use: */

 /* 0x80000000 - MQRFH_INTERNAL - This flag indicates the RFH2 header */
 /* was created by IBM MQ for internal use. */


 /* Names for Name/Value String */
 #define MQNVS_APPL_TYPE                "OPT_APP_GRP "
 #define MQNVS_MSG_TYPE                 "OPT_MSG_TYPE "

 /****************************************************************/
 /* Values Related to MQRFH Structure                            */
 /****************************************************************/

 /* Structure Length */
 #define MQRFH2_LENGTH_2                36
 #define MQRFH2_CURRENT_LENGTH          36

 /****************************************************************/
 /* Values Related to MQRMH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQRMH_STRUC_ID                 "RMH "

 /* Structure Identifier (array form) */
 #define MQRMH_STRUC_ID_ARRAY           'R','M','H',' '

 /* Structure Version Number */
 #define MQRMH_VERSION_1                1
 #define MQRMH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQRMH_LENGTH_1                 108
 #define MQRMH_CURRENT_LENGTH           108

 /* Flags */
 #define MQRMHF_LAST                    0x00000001
 #define MQRMHF_NOT_LAST                0x00000000

 /* Object Instance Identifier */
 #define MQOII_NONE                     "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Object Instance Identifier (array form) */
 #define MQOII_NONE_ARRAY               '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /****************************************************************/
 /* Values Related to MQSD Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSD_STRUC_ID                  "SD  "

 /* Structure Identifier (array form) */
 #define MQSD_STRUC_ID_ARRAY            'S','D',' ',' '

 /* Structure Version Number */
 #define MQSD_VERSION_1                 1
 #define MQSD_CURRENT_VERSION           1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQSD_LENGTH_1                  344
#else
 #define MQSD_LENGTH_1                  312
#endif
#if defined(MQ_64_BIT)
 #define MQSD_CURRENT_LENGTH            344
#else
 #define MQSD_CURRENT_LENGTH            312
#endif

 /* Security Identifier */
 #define MQSID_NONE                     "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Security Identifier (array form) */
 #define MQSID_NONE_ARRAY               '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Security Identifier Types */
 #define MQSIDT_NONE                    '\x00'
 #define MQSIDT_NT_SECURITY_ID          '\x01'
 #define MQSIDT_WAS_SECURITY_ID         '\x02'

 /****************************************************************/
 /* Values Related to MQSMPO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSMPO_STRUC_ID                "SMPO"

 /* Structure Identifier (array form) */
 #define MQSMPO_STRUC_ID_ARRAY          'S','M','P','O'

 /* Structure Version Number */
 #define MQSMPO_VERSION_1               1
 #define MQSMPO_CURRENT_VERSION         1

 /* Structure Length */
 #define MQSMPO_LENGTH_1                20
 #define MQSMPO_CURRENT_LENGTH          20

 /* Set Message Property Options */
 #define MQSMPO_SET_FIRST               0x00000000
 #define MQSMPO_SET_PROP_UNDER_CURSOR   0x00000001
 #define MQSMPO_SET_PROP_AFTER_CURSOR   0x00000002
 #define MQSMPO_APPEND_PROPERTY         0x00000004
 #define MQSMPO_SET_PROP_BEFORE_CURSOR  0x00000008
 #define MQSMPO_NONE                    0x00000000

 /****************************************************************/
 /* Values Related to MQSRO Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSRO_STRUC_ID                 "SRO "

 /* Structure Identifier (array form) */
 #define MQSRO_STRUC_ID_ARRAY           'S','R','O',' '

 /* Structure Version Number */
 #define MQSRO_VERSION_1                1
 #define MQSRO_CURRENT_VERSION          1

 /* Structure Length */
 #define MQSRO_LENGTH_1                 16
 #define MQSRO_CURRENT_LENGTH           16

 /* Subscription Request Options */
 #define MQSRO_NONE                     0x00000000
 #define MQSRO_FAIL_IF_QUIESCING        0x00002000

 /****************************************************************/
 /* Values Related to MQSTS Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSTS_STRUC_ID                 "STAT"

 /* Structure Identifier (array form) */
 #define MQSTS_STRUC_ID_ARRAY           'S','T','A','T'

 /* Structure Version Number */
 #define MQSTS_VERSION_1                1
 #define MQSTS_VERSION_2                2
 #define MQSTS_CURRENT_VERSION          2

 /* Structure Length */
 #define MQSTS_LENGTH_1                 224
#if defined(MQ_64_BIT)
 #define MQSTS_LENGTH_2                 280
#else
 #define MQSTS_LENGTH_2                 272
#endif
#if defined(MQ_64_BIT)
 #define MQSTS_CURRENT_LENGTH           280
#else
 #define MQSTS_CURRENT_LENGTH           272
#endif

 /****************************************************************/
 /* Values Related to MQTM Structure                             */
 /****************************************************************/

 /* Structure Identifier */
 #define MQTM_STRUC_ID                  "TM  "

 /* Structure Identifier (array form) */
 #define MQTM_STRUC_ID_ARRAY            'T','M',' ',' '

 /* Structure Version Number */
 #define MQTM_VERSION_1                 1
 #define MQTM_CURRENT_VERSION           1

 /* Structure Length */
 #define MQTM_LENGTH_1                  684
 #define MQTM_CURRENT_LENGTH            684

 /****************************************************************/
 /* Values Related to MQTMC2 Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQTMC_STRUC_ID                 "TMC "

 /* Structure Length */
 #define MQTMC2_LENGTH_1                684
 #define MQTMC2_LENGTH_2                732
 #define MQTMC2_CURRENT_LENGTH          732

 /* Structure Identifier (array form) */
 #define MQTMC_STRUC_ID_ARRAY           'T','M','C',' '

 /* Structure Version Number */
 #define MQTMC_VERSION_1                "   1"
 #define MQTMC_VERSION_2                "   2"
 #define MQTMC_CURRENT_VERSION          "   2"

 /* Structure Version Number (array form) */
 #define MQTMC_VERSION_1_ARRAY          ' ',' ',' ','1'
 #define MQTMC_VERSION_2_ARRAY          ' ',' ',' ','2'
 #define MQTMC_CURRENT_VERSION_ARRAY    ' ',' ',' ','2'

 /****************************************************************/
 /* Values Related to MQWIH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQWIH_STRUC_ID                 "WIH "

 /* Structure Identifier (array form) */
 #define MQWIH_STRUC_ID_ARRAY           'W','I','H',' '

 /* Structure Version Number */
 #define MQWIH_VERSION_1                1
 #define MQWIH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQWIH_LENGTH_1                 120
 #define MQWIH_CURRENT_LENGTH           120

 /* Flags */
 #define MQWIH_NONE                     0x00000000

 /****************************************************************/
 /* Values Related to MQXQH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQXQH_STRUC_ID                 "XQH "

 /* Structure Identifier (array form) */
 #define MQXQH_STRUC_ID_ARRAY           'X','Q','H',' '

 /* Structure Version Number */
 #define MQXQH_VERSION_1                1
 #define MQXQH_CURRENT_VERSION          1

 /* Structure Length */
 #define MQXQH_LENGTH_1                 428
 #define MQXQH_CURRENT_LENGTH           428

 /****************************************************************/
 /* Values Related to All Functions                              */
 /****************************************************************/

 /* Function Entry-Point and Pointer Attributes */
 #define MQENTRY
 #define MQPOINTER *

 /* Connection Handles */
 #define MQHC_DEF_HCONN                 0
 #define MQHC_UNUSABLE_HCONN            (-1)
 #define MQHC_UNASSOCIATED_HCONN        (-3)

 /* String Lengths */
 #define MQ_OPERATOR_MESSAGE_LENGTH     4
 #define MQ_ABEND_CODE_LENGTH           4
 #define MQ_ACCOUNTING_TOKEN_LENGTH     32
 #define MQ_APPL_DESC_LENGTH            64
 #define MQ_APPL_IDENTITY_DATA_LENGTH   32
 #define MQ_APPL_NAME_LENGTH            28
 #define MQ_APPL_ORIGIN_DATA_LENGTH     4
 #define MQ_APPL_TAG_LENGTH             28
 #define MQ_ARM_SUFFIX_LENGTH           2
 #define MQ_ATTENTION_ID_LENGTH         4
 #define MQ_AUTH_INFO_CONN_NAME_LENGTH  264
 #define MQ_AUTH_INFO_DESC_LENGTH       64
 #define MQ_AUTH_INFO_NAME_LENGTH       48
 #define MQ_AUTH_INFO_OCSP_URL_LENGTH   256
 #define MQ_AUTHENTICATOR_LENGTH        8
 #define MQ_AUTO_REORG_CATALOG_LENGTH   44
 #define MQ_AUTO_REORG_TIME_LENGTH      4
 #define MQ_BATCH_INTERFACE_ID_LENGTH   8
 #define MQ_BRIDGE_NAME_LENGTH          24
 #define MQ_CANCEL_CODE_LENGTH          4
 #define MQ_CF_STRUC_DESC_LENGTH        64
 #define MQ_CF_STRUC_NAME_LENGTH        12
 #define MQ_CHANNEL_DATE_LENGTH         12
 #define MQ_CHANNEL_DESC_LENGTH         64
 #define MQ_CHANNEL_NAME_LENGTH         20
 #define MQ_CHANNEL_TIME_LENGTH         8
 #define MQ_CHINIT_SERVICE_PARM_LENGTH  32
 #define MQ_CICS_FILE_NAME_LENGTH       8
 #define MQ_AMQP_CLIENT_ID_LENGTH       256
 #define MQ_CLIENT_ID_LENGTH            23
 #define MQ_CLIENT_USER_ID_LENGTH       1024
 #define MQ_CLUSTER_NAME_LENGTH         48
 #define MQ_COMM_INFO_DESC_LENGTH       64
 #define MQ_COMM_INFO_NAME_LENGTH       48
 #define MQ_CONN_NAME_LENGTH            264
 #define MQ_CONN_TAG_LENGTH             128
 #define MQ_CONNECTION_ID_LENGTH        24
 #define MQ_CORREL_ID_LENGTH            24
 #define MQ_CREATION_DATE_LENGTH        12
 #define MQ_CREATION_TIME_LENGTH        8
 #define MQ_CSP_PASSWORD_LENGTH         256
 #define MQ_DATE_LENGTH                 12
 #define MQ_DISTINGUISHED_NAME_LENGTH   1024
 #define MQ_DNS_GROUP_NAME_LENGTH       18
 #define MQ_EXIT_DATA_LENGTH            32
 #define MQ_EXIT_INFO_NAME_LENGTH       48
 #define MQ_EXIT_NAME_LENGTH            128
 #define MQ_EXIT_PD_AREA_LENGTH         48
 #define MQ_EXIT_USER_AREA_LENGTH       16
 #define MQ_FACILITY_LENGTH             8
 #define MQ_FACILITY_LIKE_LENGTH        4
 #define MQ_FORMAT_LENGTH               8
 #define MQ_FUNCTION_LENGTH             4
 #define MQ_GROUP_ID_LENGTH             24
 #define MQ_APPL_FUNCTION_NAME_LENGTH   10
 #define MQ_INSTALLATION_DESC_LENGTH    64
 #define MQ_INSTALLATION_NAME_LENGTH    16
 #define MQ_INSTALLATION_PATH_LENGTH    256
 #define MQ_JAAS_CONFIG_LENGTH          1024
 #define MQ_LDAP_PASSWORD_LENGTH        32
 #define MQ_LDAP_BASE_DN_LENGTH         1024
 #define MQ_LDAP_FIELD_LENGTH           128
 #define MQ_LDAP_CLASS_LENGTH           128
 #define MQ_LISTENER_NAME_LENGTH        48
 #define MQ_LISTENER_DESC_LENGTH        64
 #define MQ_LOCAL_ADDRESS_LENGTH        48
 #define MQ_LTERM_OVERRIDE_LENGTH       8
 #define MQ_LU_NAME_LENGTH              8
 #define MQ_LUWID_LENGTH                16
 #define MQ_MAX_EXIT_NAME_LENGTH        128
 #define MQ_MAX_MCA_USER_ID_LENGTH      64
 #define MQ_MAX_LDAP_MCA_USER_ID_LENGTH 1024
 #define MQ_MAX_PROPERTY_NAME_LENGTH    4095
 #define MQ_MAX_USER_ID_LENGTH          64
 #define MQ_MCA_JOB_NAME_LENGTH         28
 #define MQ_MCA_NAME_LENGTH             20
 #define MQ_MCA_USER_DATA_LENGTH        32
 #define MQ_MCA_USER_ID_LENGTH          12
 #define MQ_LDAP_MCA_USER_ID_LENGTH     1024
 #define MQ_MFS_MAP_NAME_LENGTH         8
 #define MQ_MODE_NAME_LENGTH            8
 #define MQ_MSG_HEADER_LENGTH           4000
 #define MQ_MSG_ID_LENGTH               24
 #define MQ_MSG_TOKEN_LENGTH            16
 #define MQ_NAMELIST_DESC_LENGTH        64
 #define MQ_NAMELIST_NAME_LENGTH        48
 #define MQ_OBJECT_INSTANCE_ID_LENGTH   24
 #define MQ_OBJECT_NAME_LENGTH          48
 #define MQ_PASS_TICKET_APPL_LENGTH     8
 #define MQ_PASSWORD_LENGTH             12
 #define MQ_PROCESS_APPL_ID_LENGTH      256
 #define MQ_PROCESS_DESC_LENGTH         64
 #define MQ_PROCESS_ENV_DATA_LENGTH     128
 #define MQ_PROCESS_NAME_LENGTH         48
 #define MQ_PROCESS_USER_DATA_LENGTH    128
 #define MQ_PROGRAM_NAME_LENGTH         20
 #define MQ_PUT_APPL_NAME_LENGTH        28
 #define MQ_PUT_DATE_LENGTH             8
 #define MQ_PUT_TIME_LENGTH             8
 #define MQ_Q_DESC_LENGTH               64
 #define MQ_Q_MGR_DESC_LENGTH           64
 #define MQ_Q_MGR_IDENTIFIER_LENGTH     48
 #define MQ_Q_MGR_NAME_LENGTH           48
 #define MQ_Q_NAME_LENGTH               48
 #define MQ_QSG_NAME_LENGTH             4
 #define MQ_REMOTE_SYS_ID_LENGTH        4
 #define MQ_SECURITY_ID_LENGTH          40
 #define MQ_SELECTOR_LENGTH             10240
 #define MQ_SERVICE_ARGS_LENGTH         255
 #define MQ_SERVICE_COMMAND_LENGTH      255
 #define MQ_SERVICE_DESC_LENGTH         64
 #define MQ_SERVICE_NAME_LENGTH         32
 #define MQ_SERVICE_PATH_LENGTH         255
 #define MQ_SERVICE_STEP_LENGTH         8
 #define MQ_SHORT_CONN_NAME_LENGTH      20
 #define MQ_SHORT_DNAME_LENGTH          256
 #define MQ_SSL_CIPHER_SPEC_LENGTH      32
 #define MQ_SSL_CIPHER_SUITE_LENGTH     32
 #define MQ_SSL_CRYPTO_HARDWARE_LENGTH  256
 #define MQ_SSL_HANDSHAKE_STAGE_LENGTH  32
 #define MQ_SSL_KEY_LIBRARY_LENGTH      44
 #define MQ_SSL_KEY_MEMBER_LENGTH       8
 #define MQ_SSL_KEY_REPOSITORY_LENGTH   256
 #define MQ_SSL_PEER_NAME_LENGTH        1024
 #define MQ_SSL_SHORT_PEER_NAME_LENGTH  256
 #define MQ_START_CODE_LENGTH           4
 #define MQ_STORAGE_CLASS_DESC_LENGTH   64
 #define MQ_STORAGE_CLASS_LENGTH        8
 #define MQ_SUB_IDENTITY_LENGTH         128
 #define MQ_SUB_POINT_LENGTH            128
 #define MQ_TCP_NAME_LENGTH             8
 #define MQ_TIME_LENGTH                 8
 #define MQ_TOPIC_DESC_LENGTH           64
 #define MQ_TOPIC_NAME_LENGTH           48
 #define MQ_TOPIC_STR_LENGTH            10240
 #define MQ_TOTAL_EXIT_DATA_LENGTH      999
 #define MQ_TOTAL_EXIT_NAME_LENGTH      999
 #define MQ_TP_NAME_LENGTH              64
 #define MQ_TPIPE_NAME_LENGTH           8
 #define MQ_TRAN_INSTANCE_ID_LENGTH     16
 #define MQ_TRANSACTION_ID_LENGTH       4
 #define MQ_TRIGGER_DATA_LENGTH         64
 #define MQ_TRIGGER_PROGRAM_NAME_LENGTH 8
 #define MQ_TRIGGER_TERM_ID_LENGTH      4
 #define MQ_TRIGGER_TRANS_ID_LENGTH     4
 #define MQ_USER_ID_LENGTH              12
 #define MQ_VERSION_LENGTH              8
 #define MQ_XCF_GROUP_NAME_LENGTH       8
 #define MQ_XCF_MEMBER_NAME_LENGTH      16
 #define MQ_SMDS_NAME_LENGTH            4
 #define MQ_CHLAUTH_DESC_LENGTH         64
 #define MQ_CUSTOM_LENGTH               128
 #define MQ_SUITE_B_SIZE                4
 #define MQ_CERT_LABEL_LENGTH           64

 /* Completion Codes */
 #define MQCC_OK                        0
 #define MQCC_WARNING                   1
 #define MQCC_FAILED                    2
 #define MQCC_UNKNOWN                   (-1)

 /* Reason Codes */
 #define MQRC_NONE                      0
 #define MQRC_APPL_FIRST                900
 #define MQRC_APPL_LAST                 999
 #define MQRC_ALIAS_BASE_Q_TYPE_ERROR   2001
 #define MQRC_ALREADY_CONNECTED         2002
 #define MQRC_BACKED_OUT                2003
 #define MQRC_BUFFER_ERROR              2004
 #define MQRC_BUFFER_LENGTH_ERROR       2005
 #define MQRC_CHAR_ATTR_LENGTH_ERROR    2006
 #define MQRC_CHAR_ATTRS_ERROR          2007
 #define MQRC_CHAR_ATTRS_TOO_SHORT      2008
 #define MQRC_CONNECTION_BROKEN         2009
 #define MQRC_DATA_LENGTH_ERROR         2010
 #define MQRC_DYNAMIC_Q_NAME_ERROR      2011
 #define MQRC_ENVIRONMENT_ERROR         2012
 #define MQRC_EXPIRY_ERROR              2013
 #define MQRC_FEEDBACK_ERROR            2014
 #define MQRC_GET_INHIBITED             2016
 #define MQRC_HANDLE_NOT_AVAILABLE      2017
 #define MQRC_HCONN_ERROR               2018
 #define MQRC_HOBJ_ERROR                2019
 #define MQRC_INHIBIT_VALUE_ERROR       2020
 #define MQRC_INT_ATTR_COUNT_ERROR      2021
 #define MQRC_INT_ATTR_COUNT_TOO_SMALL  2022
 #define MQRC_INT_ATTRS_ARRAY_ERROR     2023
 #define MQRC_SYNCPOINT_LIMIT_REACHED   2024
 #define MQRC_MAX_CONNS_LIMIT_REACHED   2025
 #define MQRC_MD_ERROR                  2026
 #define MQRC_MISSING_REPLY_TO_Q        2027
 #define MQRC_MSG_TYPE_ERROR            2029
 #define MQRC_MSG_TOO_BIG_FOR_Q         2030
 #define MQRC_MSG_TOO_BIG_FOR_Q_MGR     2031
 #define MQRC_NO_MSG_AVAILABLE          2033
 #define MQRC_NO_MSG_UNDER_CURSOR       2034
 #define MQRC_NOT_AUTHORIZED            2035
 #define MQRC_NOT_OPEN_FOR_BROWSE       2036
 #define MQRC_NOT_OPEN_FOR_INPUT        2037
 #define MQRC_NOT_OPEN_FOR_INQUIRE      2038
 #define MQRC_NOT_OPEN_FOR_OUTPUT       2039
 #define MQRC_NOT_OPEN_FOR_SET          2040
 #define MQRC_OBJECT_CHANGED            2041
 #define MQRC_OBJECT_IN_USE             2042
 #define MQRC_OBJECT_TYPE_ERROR         2043
 #define MQRC_OD_ERROR                  2044
 #define MQRC_OPTION_NOT_VALID_FOR_TYPE 2045
 #define MQRC_OPTIONS_ERROR             2046
 #define MQRC_PERSISTENCE_ERROR         2047
 #define MQRC_PERSISTENT_NOT_ALLOWED    2048
 #define MQRC_PRIORITY_EXCEEDS_MAXIMUM  2049
 #define MQRC_PRIORITY_ERROR            2050
 #define MQRC_PUT_INHIBITED             2051
 #define MQRC_Q_DELETED                 2052
 #define MQRC_Q_FULL                    2053
 #define MQRC_Q_NOT_EMPTY               2055
 #define MQRC_Q_SPACE_NOT_AVAILABLE     2056
 #define MQRC_Q_TYPE_ERROR              2057
 #define MQRC_Q_MGR_NAME_ERROR          2058
 #define MQRC_Q_MGR_NOT_AVAILABLE       2059
 #define MQRC_REPORT_OPTIONS_ERROR      2061
 #define MQRC_SECOND_MARK_NOT_ALLOWED   2062
 #define MQRC_SECURITY_ERROR            2063
 #define MQRC_SELECTOR_COUNT_ERROR      2065
 #define MQRC_SELECTOR_LIMIT_EXCEEDED   2066
 #define MQRC_SELECTOR_ERROR            2067
 #define MQRC_SELECTOR_NOT_FOR_TYPE     2068
 #define MQRC_SIGNAL_OUTSTANDING        2069
 #define MQRC_SIGNAL_REQUEST_ACCEPTED   2070
 #define MQRC_STORAGE_NOT_AVAILABLE     2071
 #define MQRC_SYNCPOINT_NOT_AVAILABLE   2072
 #define MQRC_TRIGGER_CONTROL_ERROR     2075
 #define MQRC_TRIGGER_DEPTH_ERROR       2076
 #define MQRC_TRIGGER_MSG_PRIORITY_ERR  2077
 #define MQRC_TRIGGER_TYPE_ERROR        2078
 #define MQRC_TRUNCATED_MSG_ACCEPTED    2079
 #define MQRC_TRUNCATED_MSG_FAILED      2080
 #define MQRC_UNKNOWN_ALIAS_BASE_Q      2082
 #define MQRC_UNKNOWN_OBJECT_NAME       2085
 #define MQRC_UNKNOWN_OBJECT_Q_MGR      2086
 #define MQRC_UNKNOWN_REMOTE_Q_MGR      2087
 #define MQRC_WAIT_INTERVAL_ERROR       2090
 #define MQRC_XMIT_Q_TYPE_ERROR         2091
 #define MQRC_XMIT_Q_USAGE_ERROR        2092
 #define MQRC_NOT_OPEN_FOR_PASS_ALL     2093
 #define MQRC_NOT_OPEN_FOR_PASS_IDENT   2094
 #define MQRC_NOT_OPEN_FOR_SET_ALL      2095
 #define MQRC_NOT_OPEN_FOR_SET_IDENT    2096
 #define MQRC_CONTEXT_HANDLE_ERROR      2097
 #define MQRC_CONTEXT_NOT_AVAILABLE     2098
 #define MQRC_SIGNAL1_ERROR             2099
 #define MQRC_OBJECT_ALREADY_EXISTS     2100
 #define MQRC_OBJECT_DAMAGED            2101
 #define MQRC_RESOURCE_PROBLEM          2102
 #define MQRC_ANOTHER_Q_MGR_CONNECTED   2103
 #define MQRC_UNKNOWN_REPORT_OPTION     2104
 #define MQRC_STORAGE_CLASS_ERROR       2105
 #define MQRC_COD_NOT_VALID_FOR_XCF_Q   2106
 #define MQRC_XWAIT_CANCELED            2107
 #define MQRC_XWAIT_ERROR               2108
 #define MQRC_SUPPRESSED_BY_EXIT        2109
 #define MQRC_FORMAT_ERROR              2110
 #define MQRC_SOURCE_CCSID_ERROR        2111
 #define MQRC_SOURCE_INTEGER_ENC_ERROR  2112
 #define MQRC_SOURCE_DECIMAL_ENC_ERROR  2113
 #define MQRC_SOURCE_FLOAT_ENC_ERROR    2114
 #define MQRC_TARGET_CCSID_ERROR        2115
 #define MQRC_TARGET_INTEGER_ENC_ERROR  2116
 #define MQRC_TARGET_DECIMAL_ENC_ERROR  2117
 #define MQRC_TARGET_FLOAT_ENC_ERROR    2118
 #define MQRC_NOT_CONVERTED             2119
 #define MQRC_CONVERTED_MSG_TOO_BIG     2120
 #define MQRC_TRUNCATED                 2120
 #define MQRC_NO_EXTERNAL_PARTICIPANTS  2121
 #define MQRC_PARTICIPANT_NOT_AVAILABLE 2122
 #define MQRC_OUTCOME_MIXED             2123
 #define MQRC_OUTCOME_PENDING           2124
 #define MQRC_BRIDGE_STARTED            2125
 #define MQRC_BRIDGE_STOPPED            2126
 #define MQRC_ADAPTER_STORAGE_SHORTAGE  2127
 #define MQRC_UOW_IN_PROGRESS           2128
 #define MQRC_ADAPTER_CONN_LOAD_ERROR   2129
 #define MQRC_ADAPTER_SERV_LOAD_ERROR   2130
 #define MQRC_ADAPTER_DEFS_ERROR        2131
 #define MQRC_ADAPTER_DEFS_LOAD_ERROR   2132
 #define MQRC_ADAPTER_CONV_LOAD_ERROR   2133
 #define MQRC_BO_ERROR                  2134
 #define MQRC_DH_ERROR                  2135
 #define MQRC_MULTIPLE_REASONS          2136
 #define MQRC_OPEN_FAILED               2137
 #define MQRC_ADAPTER_DISC_LOAD_ERROR   2138
 #define MQRC_CNO_ERROR                 2139
 #define MQRC_CICS_WAIT_FAILED          2140
 #define MQRC_DLH_ERROR                 2141
 #define MQRC_HEADER_ERROR              2142
 #define MQRC_SOURCE_LENGTH_ERROR       2143
 #define MQRC_TARGET_LENGTH_ERROR       2144
 #define MQRC_SOURCE_BUFFER_ERROR       2145
 #define MQRC_TARGET_BUFFER_ERROR       2146
 #define MQRC_IIH_ERROR                 2148
 #define MQRC_PCF_ERROR                 2149
 #define MQRC_DBCS_ERROR                2150
 #define MQRC_OBJECT_NAME_ERROR         2152
 #define MQRC_OBJECT_Q_MGR_NAME_ERROR   2153
 #define MQRC_RECS_PRESENT_ERROR        2154
 #define MQRC_OBJECT_RECORDS_ERROR      2155
 #define MQRC_RESPONSE_RECORDS_ERROR    2156
 #define MQRC_ASID_MISMATCH             2157
 #define MQRC_PMO_RECORD_FLAGS_ERROR    2158
 #define MQRC_PUT_MSG_RECORDS_ERROR     2159
 #define MQRC_CONN_ID_IN_USE            2160
 #define MQRC_Q_MGR_QUIESCING           2161
 #define MQRC_Q_MGR_STOPPING            2162
 #define MQRC_DUPLICATE_RECOV_COORD     2163
 #define MQRC_PMO_ERROR                 2173
 #define MQRC_API_EXIT_NOT_FOUND        2182
 #define MQRC_API_EXIT_LOAD_ERROR       2183
 #define MQRC_REMOTE_Q_NAME_ERROR       2184
 #define MQRC_INCONSISTENT_PERSISTENCE  2185
 #define MQRC_GMO_ERROR                 2186
 #define MQRC_CICS_BRIDGE_RESTRICTION   2187
 #define MQRC_STOPPED_BY_CLUSTER_EXIT   2188
 #define MQRC_CLUSTER_RESOLUTION_ERROR  2189
 #define MQRC_CONVERTED_STRING_TOO_BIG  2190
 #define MQRC_TMC_ERROR                 2191
 #define MQRC_STORAGE_MEDIUM_FULL       2192
 #define MQRC_PAGESET_FULL              2192
 #define MQRC_PAGESET_ERROR             2193
 #define MQRC_NAME_NOT_VALID_FOR_TYPE   2194
 #define MQRC_UNEXPECTED_ERROR          2195
 #define MQRC_UNKNOWN_XMIT_Q            2196
 #define MQRC_UNKNOWN_DEF_XMIT_Q        2197
 #define MQRC_DEF_XMIT_Q_TYPE_ERROR     2198
 #define MQRC_DEF_XMIT_Q_USAGE_ERROR    2199
 #define MQRC_MSG_MARKED_BROWSE_CO_OP   2200
 #define MQRC_NAME_IN_USE               2201
 #define MQRC_CONNECTION_QUIESCING      2202
 #define MQRC_CONNECTION_STOPPING       2203
 #define MQRC_ADAPTER_NOT_AVAILABLE     2204
 #define MQRC_MSG_ID_ERROR              2206
 #define MQRC_CORREL_ID_ERROR           2207
 #define MQRC_FILE_SYSTEM_ERROR         2208
 #define MQRC_NO_MSG_LOCKED             2209
 #define MQRC_SOAP_DOTNET_ERROR         2210
 #define MQRC_SOAP_AXIS_ERROR           2211
 #define MQRC_SOAP_URL_ERROR            2212
 #define MQRC_FILE_NOT_AUDITED          2216
 #define MQRC_CONNECTION_NOT_AUTHORIZED 2217
 #define MQRC_MSG_TOO_BIG_FOR_CHANNEL   2218
 #define MQRC_CALL_IN_PROGRESS          2219
 #define MQRC_RMH_ERROR                 2220
 #define MQRC_Q_MGR_ACTIVE              2222
 #define MQRC_Q_MGR_NOT_ACTIVE          2223
 #define MQRC_Q_DEPTH_HIGH              2224
 #define MQRC_Q_DEPTH_LOW               2225
 #define MQRC_Q_SERVICE_INTERVAL_HIGH   2226
 #define MQRC_Q_SERVICE_INTERVAL_OK     2227
 #define MQRC_RFH_HEADER_FIELD_ERROR    2228
 #define MQRC_RAS_PROPERTY_ERROR        2229
 #define MQRC_UNIT_OF_WORK_NOT_STARTED  2232
 #define MQRC_CHANNEL_AUTO_DEF_OK       2233
 #define MQRC_CHANNEL_AUTO_DEF_ERROR    2234
 #define MQRC_CFH_ERROR                 2235
 #define MQRC_CFIL_ERROR                2236
 #define MQRC_CFIN_ERROR                2237
 #define MQRC_CFSL_ERROR                2238
 #define MQRC_CFST_ERROR                2239
 #define MQRC_INCOMPLETE_GROUP          2241
 #define MQRC_INCOMPLETE_MSG            2242
 #define MQRC_INCONSISTENT_CCSIDS       2243
 #define MQRC_INCONSISTENT_ENCODINGS    2244
 #define MQRC_INCONSISTENT_UOW          2245
 #define MQRC_INVALID_MSG_UNDER_CURSOR  2246
 #define MQRC_MATCH_OPTIONS_ERROR       2247
 #define MQRC_MDE_ERROR                 2248
 #define MQRC_MSG_FLAGS_ERROR           2249
 #define MQRC_MSG_SEQ_NUMBER_ERROR      2250
 #define MQRC_OFFSET_ERROR              2251
 #define MQRC_ORIGINAL_LENGTH_ERROR     2252
 #define MQRC_SEGMENT_LENGTH_ZERO       2253
 #define MQRC_UOW_NOT_AVAILABLE         2255
 #define MQRC_WRONG_GMO_VERSION         2256
 #define MQRC_WRONG_MD_VERSION          2257
 #define MQRC_GROUP_ID_ERROR            2258
 #define MQRC_INCONSISTENT_BROWSE       2259
 #define MQRC_XQH_ERROR                 2260
 #define MQRC_SRC_ENV_ERROR             2261
 #define MQRC_SRC_NAME_ERROR            2262
 #define MQRC_DEST_ENV_ERROR            2263
 #define MQRC_DEST_NAME_ERROR           2264
 #define MQRC_TM_ERROR                  2265
 #define MQRC_CLUSTER_EXIT_ERROR        2266
 #define MQRC_CLUSTER_EXIT_LOAD_ERROR   2267
 #define MQRC_CLUSTER_PUT_INHIBITED     2268
 #define MQRC_CLUSTER_RESOURCE_ERROR    2269
 #define MQRC_NO_DESTINATIONS_AVAILABLE 2270
 #define MQRC_CONN_TAG_IN_USE           2271
 #define MQRC_PARTIALLY_CONVERTED       2272
 #define MQRC_CONNECTION_ERROR          2273
 #define MQRC_OPTION_ENVIRONMENT_ERROR  2274
 #define MQRC_CD_ERROR                  2277
 #define MQRC_CLIENT_CONN_ERROR         2278
 #define MQRC_CHANNEL_STOPPED_BY_USER   2279
 #define MQRC_HCONFIG_ERROR             2280
 #define MQRC_FUNCTION_ERROR            2281
 #define MQRC_CHANNEL_STARTED           2282
 #define MQRC_CHANNEL_STOPPED           2283
 #define MQRC_CHANNEL_CONV_ERROR        2284
 #define MQRC_SERVICE_NOT_AVAILABLE     2285
 #define MQRC_INITIALIZATION_FAILED     2286
 #define MQRC_TERMINATION_FAILED        2287
 #define MQRC_UNKNOWN_Q_NAME            2288
 #define MQRC_SERVICE_ERROR             2289
 #define MQRC_Q_ALREADY_EXISTS          2290
 #define MQRC_USER_ID_NOT_AVAILABLE     2291
 #define MQRC_UNKNOWN_ENTITY            2292
 #define MQRC_UNKNOWN_AUTH_ENTITY       2293
 #define MQRC_UNKNOWN_REF_OBJECT        2294
 #define MQRC_CHANNEL_ACTIVATED         2295
 #define MQRC_CHANNEL_NOT_ACTIVATED     2296
 #define MQRC_UOW_CANCELED              2297
 #define MQRC_FUNCTION_NOT_SUPPORTED    2298
 #define MQRC_SELECTOR_TYPE_ERROR       2299
 #define MQRC_COMMAND_TYPE_ERROR        2300
 #define MQRC_MULTIPLE_INSTANCE_ERROR   2301
 #define MQRC_SYSTEM_ITEM_NOT_ALTERABLE 2302
 #define MQRC_BAG_CONVERSION_ERROR      2303
 #define MQRC_SELECTOR_OUT_OF_RANGE     2304
 #define MQRC_SELECTOR_NOT_UNIQUE       2305
 #define MQRC_INDEX_NOT_PRESENT         2306
 #define MQRC_STRING_ERROR              2307
 #define MQRC_ENCODING_NOT_SUPPORTED    2308
 #define MQRC_SELECTOR_NOT_PRESENT      2309
 #define MQRC_OUT_SELECTOR_ERROR        2310
 #define MQRC_STRING_TRUNCATED          2311
 #define MQRC_SELECTOR_WRONG_TYPE       2312
 #define MQRC_INCONSISTENT_ITEM_TYPE    2313
 #define MQRC_INDEX_ERROR               2314
 #define MQRC_SYSTEM_BAG_NOT_ALTERABLE  2315
 #define MQRC_ITEM_COUNT_ERROR          2316
 #define MQRC_FORMAT_NOT_SUPPORTED      2317
 #define MQRC_SELECTOR_NOT_SUPPORTED    2318
 #define MQRC_ITEM_VALUE_ERROR          2319
 #define MQRC_HBAG_ERROR                2320
 #define MQRC_PARAMETER_MISSING         2321
 #define MQRC_CMD_SERVER_NOT_AVAILABLE  2322
 #define MQRC_STRING_LENGTH_ERROR       2323
 #define MQRC_INQUIRY_COMMAND_ERROR     2324
 #define MQRC_NESTED_BAG_NOT_SUPPORTED  2325
 #define MQRC_BAG_WRONG_TYPE            2326
 #define MQRC_ITEM_TYPE_ERROR           2327
 #define MQRC_SYSTEM_BAG_NOT_DELETABLE  2328
 #define MQRC_SYSTEM_ITEM_NOT_DELETABLE 2329
 #define MQRC_CODED_CHAR_SET_ID_ERROR   2330
 #define MQRC_MSG_TOKEN_ERROR           2331
 #define MQRC_MISSING_WIH               2332
 #define MQRC_WIH_ERROR                 2333
 #define MQRC_RFH_ERROR                 2334
 #define MQRC_RFH_STRING_ERROR          2335
 #define MQRC_RFH_COMMAND_ERROR         2336
 #define MQRC_RFH_PARM_ERROR            2337
 #define MQRC_RFH_DUPLICATE_PARM        2338
 #define MQRC_RFH_PARM_MISSING          2339
 #define MQRC_CHAR_CONVERSION_ERROR     2340
 #define MQRC_UCS2_CONVERSION_ERROR     2341
 #define MQRC_DB2_NOT_AVAILABLE         2342
 #define MQRC_OBJECT_NOT_UNIQUE         2343
 #define MQRC_CONN_TAG_NOT_RELEASED     2344
 #define MQRC_CF_NOT_AVAILABLE          2345
 #define MQRC_CF_STRUC_IN_USE           2346
 #define MQRC_CF_STRUC_LIST_HDR_IN_USE  2347
 #define MQRC_CF_STRUC_AUTH_FAILED      2348
 #define MQRC_CF_STRUC_ERROR            2349
 #define MQRC_CONN_TAG_NOT_USABLE       2350
 #define MQRC_GLOBAL_UOW_CONFLICT       2351
 #define MQRC_LOCAL_UOW_CONFLICT        2352
 #define MQRC_HANDLE_IN_USE_FOR_UOW     2353
 #define MQRC_UOW_ENLISTMENT_ERROR      2354
 #define MQRC_UOW_MIX_NOT_SUPPORTED     2355
 #define MQRC_WXP_ERROR                 2356
 #define MQRC_CURRENT_RECORD_ERROR      2357
 #define MQRC_NEXT_OFFSET_ERROR         2358
 #define MQRC_NO_RECORD_AVAILABLE       2359
 #define MQRC_OBJECT_LEVEL_INCOMPATIBLE 2360
 #define MQRC_NEXT_RECORD_ERROR         2361
 #define MQRC_BACKOUT_THRESHOLD_REACHED 2362
 #define MQRC_MSG_NOT_MATCHED           2363
 #define MQRC_JMS_FORMAT_ERROR          2364
 #define MQRC_SEGMENTS_NOT_SUPPORTED    2365
 #define MQRC_WRONG_CF_LEVEL            2366
 #define MQRC_CONFIG_CREATE_OBJECT      2367
 #define MQRC_CONFIG_CHANGE_OBJECT      2368
 #define MQRC_CONFIG_DELETE_OBJECT      2369
 #define MQRC_CONFIG_REFRESH_OBJECT     2370
 #define MQRC_CHANNEL_SSL_ERROR         2371
 #define MQRC_PARTICIPANT_NOT_DEFINED   2372
 #define MQRC_CF_STRUC_FAILED           2373
 #define MQRC_API_EXIT_ERROR            2374
 #define MQRC_API_EXIT_INIT_ERROR       2375
 #define MQRC_API_EXIT_TERM_ERROR       2376
 #define MQRC_EXIT_REASON_ERROR         2377
 #define MQRC_RESERVED_VALUE_ERROR      2378
 #define MQRC_NO_DATA_AVAILABLE         2379
 #define MQRC_SCO_ERROR                 2380
 #define MQRC_KEY_REPOSITORY_ERROR      2381
 #define MQRC_CRYPTO_HARDWARE_ERROR     2382
 #define MQRC_AUTH_INFO_REC_COUNT_ERROR 2383
 #define MQRC_AUTH_INFO_REC_ERROR       2384
 #define MQRC_AIR_ERROR                 2385
 #define MQRC_AUTH_INFO_TYPE_ERROR      2386
 #define MQRC_AUTH_INFO_CONN_NAME_ERROR 2387
 #define MQRC_LDAP_USER_NAME_ERROR      2388
 #define MQRC_LDAP_USER_NAME_LENGTH_ERR 2389
 #define MQRC_LDAP_PASSWORD_ERROR       2390
 #define MQRC_SSL_ALREADY_INITIALIZED   2391
 #define MQRC_SSL_CONFIG_ERROR          2392
 #define MQRC_SSL_INITIALIZATION_ERROR  2393
 #define MQRC_Q_INDEX_TYPE_ERROR        2394
 #define MQRC_CFBS_ERROR                2395
 #define MQRC_SSL_NOT_ALLOWED           2396
 #define MQRC_JSSE_ERROR                2397
 #define MQRC_SSL_PEER_NAME_MISMATCH    2398
 #define MQRC_SSL_PEER_NAME_ERROR       2399
 #define MQRC_UNSUPPORTED_CIPHER_SUITE  2400
 #define MQRC_SSL_CERTIFICATE_REVOKED   2401
 #define MQRC_SSL_CERT_STORE_ERROR      2402
 #define MQRC_CLIENT_EXIT_LOAD_ERROR    2406
 #define MQRC_CLIENT_EXIT_ERROR         2407
 #define MQRC_UOW_COMMITTED             2408
 #define MQRC_SSL_KEY_RESET_ERROR       2409
 #define MQRC_UNKNOWN_COMPONENT_NAME    2410
 #define MQRC_LOGGER_STATUS             2411
 #define MQRC_COMMAND_MQSC              2412
 #define MQRC_COMMAND_PCF               2413
 #define MQRC_CFIF_ERROR                2414
 #define MQRC_CFSF_ERROR                2415
 #define MQRC_CFGR_ERROR                2416
 #define MQRC_MSG_NOT_ALLOWED_IN_GROUP  2417
 #define MQRC_FILTER_OPERATOR_ERROR     2418
 #define MQRC_NESTED_SELECTOR_ERROR     2419
 #define MQRC_EPH_ERROR                 2420
 #define MQRC_RFH_FORMAT_ERROR          2421
 #define MQRC_CFBF_ERROR                2422
 #define MQRC_CLIENT_CHANNEL_CONFLICT   2423
 #define MQRC_SD_ERROR                  2424
 #define MQRC_TOPIC_STRING_ERROR        2425
 #define MQRC_STS_ERROR                 2426
 #define MQRC_NO_SUBSCRIPTION           2428
 #define MQRC_SUBSCRIPTION_IN_USE       2429
 #define MQRC_STAT_TYPE_ERROR           2430
 #define MQRC_SUB_USER_DATA_ERROR       2431
 #define MQRC_SUB_ALREADY_EXISTS        2432
 #define MQRC_IDENTITY_MISMATCH         2434
 #define MQRC_ALTER_SUB_ERROR           2435
 #define MQRC_DURABILITY_NOT_ALLOWED    2436
 #define MQRC_NO_RETAINED_MSG           2437
 #define MQRC_SRO_ERROR                 2438
 #define MQRC_SUB_NAME_ERROR            2440
 #define MQRC_OBJECT_STRING_ERROR       2441
 #define MQRC_PROPERTY_NAME_ERROR       2442
 #define MQRC_SEGMENTATION_NOT_ALLOWED  2443
 #define MQRC_CBD_ERROR                 2444
 #define MQRC_CTLO_ERROR                2445
 #define MQRC_NO_CALLBACKS_ACTIVE       2446
 #define MQRC_CALLBACK_NOT_REGISTERED   2448
 #define MQRC_OPTIONS_CHANGED           2457
 #define MQRC_READ_AHEAD_MSGS           2458
 #define MQRC_SELECTOR_SYNTAX_ERROR     2459
 #define MQRC_HMSG_ERROR                2460
 #define MQRC_CMHO_ERROR                2461
 #define MQRC_DMHO_ERROR                2462
 #define MQRC_SMPO_ERROR                2463
 #define MQRC_IMPO_ERROR                2464
 #define MQRC_PROPERTY_NAME_TOO_BIG     2465
 #define MQRC_PROP_VALUE_NOT_CONVERTED  2466
 #define MQRC_PROP_TYPE_NOT_SUPPORTED   2467
 #define MQRC_PROPERTY_VALUE_TOO_BIG    2469
 #define MQRC_PROP_CONV_NOT_SUPPORTED   2470
 #define MQRC_PROPERTY_NOT_AVAILABLE    2471
 #define MQRC_PROP_NUMBER_FORMAT_ERROR  2472
 #define MQRC_PROPERTY_TYPE_ERROR       2473
 #define MQRC_PROPERTIES_TOO_BIG        2478
 #define MQRC_PUT_NOT_RETAINED          2479
 #define MQRC_ALIAS_TARGTYPE_CHANGED    2480
 #define MQRC_DMPO_ERROR                2481
 #define MQRC_PD_ERROR                  2482
 #define MQRC_CALLBACK_TYPE_ERROR       2483
 #define MQRC_CBD_OPTIONS_ERROR         2484
 #define MQRC_MAX_MSG_LENGTH_ERROR      2485
 #define MQRC_CALLBACK_ROUTINE_ERROR    2486
 #define MQRC_CALLBACK_LINK_ERROR       2487
 #define MQRC_OPERATION_ERROR           2488
 #define MQRC_BMHO_ERROR                2489
 #define MQRC_UNSUPPORTED_PROPERTY      2490
 #define MQRC_PROP_NAME_NOT_CONVERTED   2492
 #define MQRC_GET_ENABLED               2494
 #define MQRC_MODULE_NOT_FOUND          2495
 #define MQRC_MODULE_INVALID            2496
 #define MQRC_MODULE_ENTRY_NOT_FOUND    2497
 #define MQRC_MIXED_CONTENT_NOT_ALLOWED 2498
 #define MQRC_MSG_HANDLE_IN_USE         2499
 #define MQRC_HCONN_ASYNC_ACTIVE        2500
 #define MQRC_MHBO_ERROR                2501
 #define MQRC_PUBLICATION_FAILURE       2502
 #define MQRC_SUB_INHIBITED             2503
 #define MQRC_SELECTOR_ALWAYS_FALSE     2504
 #define MQRC_XEPO_ERROR                2507
 #define MQRC_DURABILITY_NOT_ALTERABLE  2509
 #define MQRC_TOPIC_NOT_ALTERABLE       2510
 #define MQRC_SUBLEVEL_NOT_ALTERABLE    2512
 #define MQRC_PROPERTY_NAME_LENGTH_ERR  2513
 #define MQRC_DUPLICATE_GROUP_SUB       2514
 #define MQRC_GROUPING_NOT_ALTERABLE    2515
 #define MQRC_SELECTOR_INVALID_FOR_TYPE 2516
 #define MQRC_HOBJ_QUIESCED             2517
 #define MQRC_HOBJ_QUIESCED_NO_MSGS     2518
 #define MQRC_SELECTION_STRING_ERROR    2519
 #define MQRC_RES_OBJECT_STRING_ERROR   2520
 #define MQRC_CONNECTION_SUSPENDED      2521
 #define MQRC_INVALID_DESTINATION       2522
 #define MQRC_INVALID_SUBSCRIPTION      2523
 #define MQRC_SELECTOR_NOT_ALTERABLE    2524
 #define MQRC_RETAINED_MSG_Q_ERROR      2525
 #define MQRC_RETAINED_NOT_DELIVERED    2526
 #define MQRC_RFH_RESTRICTED_FORMAT_ERR 2527
 #define MQRC_CONNECTION_STOPPED        2528
 #define MQRC_ASYNC_UOW_CONFLICT        2529
 #define MQRC_ASYNC_XA_CONFLICT         2530
 #define MQRC_PUBSUB_INHIBITED          2531
 #define MQRC_MSG_HANDLE_COPY_FAILURE   2532
 #define MQRC_DEST_CLASS_NOT_ALTERABLE  2533
 #define MQRC_OPERATION_NOT_ALLOWED     2534
 #define MQRC_ACTION_ERROR              2535
 #define MQRC_CHANNEL_NOT_AVAILABLE     2537
 #define MQRC_HOST_NOT_AVAILABLE        2538
 #define MQRC_CHANNEL_CONFIG_ERROR      2539
 #define MQRC_UNKNOWN_CHANNEL_NAME      2540
 #define MQRC_LOOPING_PUBLICATION       2541
 #define MQRC_ALREADY_JOINED            2542
 #define MQRC_STANDBY_Q_MGR             2543
 #define MQRC_RECONNECTING              2544
 #define MQRC_RECONNECTED               2545
 #define MQRC_RECONNECT_QMID_MISMATCH   2546
 #define MQRC_RECONNECT_INCOMPATIBLE    2547
 #define MQRC_RECONNECT_FAILED          2548
 #define MQRC_CALL_INTERRUPTED          2549
 #define MQRC_NO_SUBS_MATCHED           2550
 #define MQRC_SELECTION_NOT_AVAILABLE   2551
 #define MQRC_CHANNEL_SSL_WARNING       2552
 #define MQRC_OCSP_URL_ERROR            2553
 #define MQRC_CONTENT_ERROR             2554
 #define MQRC_RECONNECT_Q_MGR_REQD      2555
 #define MQRC_RECONNECT_TIMED_OUT       2556
 #define MQRC_PUBLISH_EXIT_ERROR        2557
 #define MQRC_COMMINFO_ERROR            2558
 #define MQRC_DEF_SYNCPOINT_INHIBITED   2559
 #define MQRC_MULTICAST_ONLY            2560
 #define MQRC_DATA_SET_NOT_AVAILABLE    2561
 #define MQRC_GROUPING_NOT_ALLOWED      2562
 #define MQRC_GROUP_ADDRESS_ERROR       2563
 #define MQRC_MULTICAST_CONFIG_ERROR    2564
 #define MQRC_MULTICAST_INTERFACE_ERROR 2565
 #define MQRC_MULTICAST_SEND_ERROR      2566
 #define MQRC_MULTICAST_INTERNAL_ERROR  2567
 #define MQRC_CONNECTION_NOT_AVAILABLE  2568
 #define MQRC_SYNCPOINT_NOT_ALLOWED     2569
 #define MQRC_SSL_ALT_PROVIDER_REQUIRED 2570
 #define MQRC_MCAST_PUB_STATUS          2571
 #define MQRC_MCAST_SUB_STATUS          2572
 #define MQRC_PRECONN_EXIT_LOAD_ERROR   2573
 #define MQRC_PRECONN_EXIT_NOT_FOUND    2574
 #define MQRC_PRECONN_EXIT_ERROR        2575
 #define MQRC_CD_ARRAY_ERROR            2576
 #define MQRC_CHANNEL_BLOCKED           2577
 #define MQRC_CHANNEL_BLOCKED_WARNING   2578
 #define MQRC_SUBSCRIPTION_CREATE       2579
 #define MQRC_SUBSCRIPTION_DELETE       2580
 #define MQRC_SUBSCRIPTION_CHANGE       2581
 #define MQRC_SUBSCRIPTION_REFRESH      2582
 #define MQRC_INSTALLATION_MISMATCH     2583
 #define MQRC_NOT_PRIVILEGED            2584
 #define MQRC_PROPERTIES_DISABLED       2586
 #define MQRC_HMSG_NOT_AVAILABLE        2587
 #define MQRC_EXIT_PROPS_NOT_SUPPORTED  2588
 #define MQRC_INSTALLATION_MISSING      2589
 #define MQRC_FASTPATH_NOT_AVAILABLE    2590
 #define MQRC_CIPHER_SPEC_NOT_SUITE_B   2591
 #define MQRC_SUITE_B_ERROR             2592
 #define MQRC_CERT_VAL_POLICY_ERROR     2593
 #define MQRC_PASSWORD_PROTECTION_ERROR 2594
 #define MQRC_CSP_ERROR                 2595
 #define MQRC_CERT_LABEL_NOT_ALLOWED    2596
 #define MQRC_ADMIN_TOPIC_STRING_ERROR  2598
 #define MQRC_AMQP_NOT_AVAILABLE        2599
 #define MQRC_CCDT_URL_ERROR            2600
 #define MQRC_REOPEN_EXCL_INPUT_ERROR   6100
 #define MQRC_REOPEN_INQUIRE_ERROR      6101
 #define MQRC_REOPEN_SAVED_CONTEXT_ERR  6102
 #define MQRC_REOPEN_TEMPORARY_Q_ERROR  6103
 #define MQRC_ATTRIBUTE_LOCKED          6104
 #define MQRC_CURSOR_NOT_VALID          6105
 #define MQRC_ENCODING_ERROR            6106
 #define MQRC_STRUC_ID_ERROR            6107
 #define MQRC_NULL_POINTER              6108
 #define MQRC_NO_CONNECTION_REFERENCE   6109
 #define MQRC_NO_BUFFER                 6110
 #define MQRC_BINARY_DATA_LENGTH_ERROR  6111
 #define MQRC_BUFFER_NOT_AUTOMATIC      6112
 #define MQRC_INSUFFICIENT_BUFFER       6113
 #define MQRC_INSUFFICIENT_DATA         6114
 #define MQRC_DATA_TRUNCATED            6115
 #define MQRC_ZERO_LENGTH               6116
 #define MQRC_NEGATIVE_LENGTH           6117
 #define MQRC_NEGATIVE_OFFSET           6118
 #define MQRC_INCONSISTENT_FORMAT       6119
 #define MQRC_INCONSISTENT_OBJECT_STATE 6120
 #define MQRC_CONTEXT_OBJECT_NOT_VALID  6121
 #define MQRC_CONTEXT_OPEN_ERROR        6122
 #define MQRC_STRUC_LENGTH_ERROR        6123
 #define MQRC_NOT_CONNECTED             6124
 #define MQRC_NOT_OPEN                  6125
 #define MQRC_DISTRIBUTION_LIST_EMPTY   6126
 #define MQRC_INCONSISTENT_OPEN_OPTIONS 6127
 #define MQRC_WRONG_VERSION             6128
 #define MQRC_REFERENCE_ERROR           6129
 #define MQRC_XR_NOT_AVAILABLE          6130
 #define MQRC_SUB_JOIN_NOT_ALTERABLE    29440

 /****************************************************************/
 /* Values Related to Queue Attributes                           */
 /****************************************************************/

 /* Queue Types */
 #define MQQT_LOCAL                     1
 #define MQQT_MODEL                     2
 #define MQQT_ALIAS                     3
 #define MQQT_REMOTE                    6
 #define MQQT_CLUSTER                   7

 /* Cluster Queue Types */
 #define MQCQT_LOCAL_Q                  1
 #define MQCQT_ALIAS_Q                  2
 #define MQCQT_REMOTE_Q                 3
 #define MQCQT_Q_MGR_ALIAS              4

 /* Extended Queue Types */
 #define MQQT_ALL                       1001

 /* Queue Definition Types */
 #define MQQDT_PREDEFINED               1
 #define MQQDT_PERMANENT_DYNAMIC        2
 #define MQQDT_TEMPORARY_DYNAMIC        3
 #define MQQDT_SHARED_DYNAMIC           4

 /* Inhibit Get Values */
 #define MQQA_GET_INHIBITED             1
 #define MQQA_GET_ALLOWED               0

 /* Inhibit Put Values */
 #define MQQA_PUT_INHIBITED             1
 #define MQQA_PUT_ALLOWED               0

 /* Queue Shareability */
 #define MQQA_SHAREABLE                 1
 #define MQQA_NOT_SHAREABLE             0

 /* Back-Out Hardening */
 #define MQQA_BACKOUT_HARDENED          1
 #define MQQA_BACKOUT_NOT_HARDENED      0

 /* Message Delivery Sequence */
 #define MQMDS_PRIORITY                 0
 #define MQMDS_FIFO                     1

 /* Nonpersistent Message Class */
 #define MQNPM_CLASS_NORMAL             0
 #define MQNPM_CLASS_HIGH               10

 /* Trigger Controls */
 #define MQTC_OFF                       0
 #define MQTC_ON                        1

 /* Trigger Types */
 #define MQTT_NONE                      0
 #define MQTT_FIRST                     1
 #define MQTT_EVERY                     2
 #define MQTT_DEPTH                     3

 /* Trigger Restart */
 #define MQTRIGGER_RESTART_NO           0
 #define MQTRIGGER_RESTART_YES          1

 /* Queue Usages */
 #define MQUS_NORMAL                    0
 #define MQUS_TRANSMISSION              1

 /* Distribution Lists */
 #define MQDL_SUPPORTED                 1
 #define MQDL_NOT_SUPPORTED             0

 /* Index Types */
 #define MQIT_NONE                      0
 #define MQIT_MSG_ID                    1
 #define MQIT_CORREL_ID                 2
 #define MQIT_MSG_TOKEN                 4
 #define MQIT_GROUP_ID                  5

 /* Default Bindings */
 #define MQBND_BIND_ON_OPEN             0
 #define MQBND_BIND_NOT_FIXED           1
 #define MQBND_BIND_ON_GROUP            2

 /* Queue Sharing Group Dispositions */
 #define MQQSGD_ALL                     (-1)
 #define MQQSGD_Q_MGR                   0
 #define MQQSGD_COPY                    1
 #define MQQSGD_SHARED                  2
 #define MQQSGD_GROUP                   3
 #define MQQSGD_PRIVATE                 4
 #define MQQSGD_LIVE                    6

 /* Reorganization Controls */
 #define MQREORG_DISABLED               0
 #define MQREORG_ENABLED                1

 /* Read Ahead Values */
 #define MQREADA_NO                     0
 #define MQREADA_YES                    1
 #define MQREADA_DISABLED               2
 #define MQREADA_INHIBITED              3
 #define MQREADA_BACKLOG                4

 /* Queue and Channel Property Control Values */
 #define MQPROP_COMPATIBILITY           0
 #define MQPROP_NONE                    1
 #define MQPROP_ALL                     2
 #define MQPROP_FORCE_MQRFH2            3
 #define MQPROP_V6COMPAT                4

 /****************************************************************/
 /* Values Related to Namelist Attributes                        */
 /****************************************************************/

 /* Name Count */
 #define MQNC_MAX_NAMELIST_NAME_COUNT   256

 /* Namelist Types */
 #define MQNT_NONE                      0
 #define MQNT_Q                         1
 #define MQNT_CLUSTER                   2
 #define MQNT_AUTH_INFO                 4
 #define MQNT_ALL                       1001

 /****************************************************************/
 /* Values Related to CF-Structure Attributes                    */
 /****************************************************************/

 /* CF Recoverability */
 #define MQCFR_YES                      1
 #define MQCFR_NO                       0

 /* CF Automatic Recovery */
 #define MQRECAUTO_NO                   0
 #define MQRECAUTO_YES                  1

 /* CF Loss of Connectivity Action */
 #define MQCFCONLOS_TERMINATE           0
 #define MQCFCONLOS_TOLERATE            1
 #define MQCFCONLOS_ASQMGR              2

 /****************************************************************/
 /* Values Related to Service Attributes                         */
 /****************************************************************/

 /* Service Types */
 #define MQSVC_TYPE_COMMAND             0
 #define MQSVC_TYPE_SERVER              1

 /****************************************************************/
 /* Values Related to QueueManager Attributes                    */
 /****************************************************************/

 /* Adopt New MCA Checks */
 #define MQADOPT_CHECK_NONE             0
 #define MQADOPT_CHECK_ALL              1
 #define MQADOPT_CHECK_Q_MGR_NAME       2
 #define MQADOPT_CHECK_NET_ADDR         4
 #define MQADOPT_CHECK_CHANNEL_NAME     8

 /* Adopt New MCA Types */
 #define MQADOPT_TYPE_NO                0
 #define MQADOPT_TYPE_ALL               1
 #define MQADOPT_TYPE_SVR               2
 #define MQADOPT_TYPE_SDR               4
 #define MQADOPT_TYPE_RCVR              8
 #define MQADOPT_TYPE_CLUSRCVR          16

 /* Autostart */
 #define MQAUTO_START_NO                0
 #define MQAUTO_START_YES               1

 /* Channel Auto Definition */
 #define MQCHAD_DISABLED                0
 #define MQCHAD_ENABLED                 1

 /* Cluster Workload */
 #define MQCLWL_USEQ_LOCAL              0
 #define MQCLWL_USEQ_ANY                1
 #define MQCLWL_USEQ_AS_Q_MGR           (-3)

 /* Command Levels */
 #define MQCMDL_LEVEL_1                 100
 #define MQCMDL_LEVEL_101               101
 #define MQCMDL_LEVEL_110               110
 #define MQCMDL_LEVEL_114               114
 #define MQCMDL_LEVEL_120               120
 #define MQCMDL_LEVEL_200               200
 #define MQCMDL_LEVEL_201               201
 #define MQCMDL_LEVEL_210               210
 #define MQCMDL_LEVEL_211               211
 #define MQCMDL_LEVEL_220               220
 #define MQCMDL_LEVEL_221               221
 #define MQCMDL_LEVEL_230               230
 #define MQCMDL_LEVEL_320               320
 #define MQCMDL_LEVEL_420               420
 #define MQCMDL_LEVEL_500               500
 #define MQCMDL_LEVEL_510               510
 #define MQCMDL_LEVEL_520               520
 #define MQCMDL_LEVEL_530               530
 #define MQCMDL_LEVEL_531               531
 #define MQCMDL_LEVEL_600               600
 #define MQCMDL_LEVEL_700               700
 #define MQCMDL_LEVEL_701               701
 #define MQCMDL_LEVEL_710               710
 #define MQCMDL_LEVEL_711               711
 #define MQCMDL_LEVEL_750               750
 #define MQCMDL_LEVEL_800               800
 #define MQCMDL_LEVEL_801               801
 #define MQCMDL_LEVEL_802               802
 #define MQCMDL_LEVEL_900               900
 #define MQCMDL_LEVEL_901               901
 #define MQCMDL_LEVEL_902               902
 #define MQCMDL_LEVEL_903               903
 #define MQCMDL_LEVEL_904               904
 #define MQCMDL_LEVEL_905               905
 #define MQCMDL_LEVEL_910               910
 #define MQCMDL_LEVEL_911               911
 #define MQCMDL_CURRENT_LEVEL           911

 /* Command Server Options */
 #define MQCSRV_CONVERT_NO              0
 #define MQCSRV_CONVERT_YES             1
 #define MQCSRV_DLQ_NO                  0
 #define MQCSRV_DLQ_YES                 1

 /* DNS WLM */
 #define MQDNSWLM_NO                    0
 #define MQDNSWLM_YES                   1

 /* Expiration Scan Interval */
 #define MQEXPI_OFF                     0

 /* Intra-Group Queuing */
 #define MQIGQ_DISABLED                 0
 #define MQIGQ_ENABLED                  1

 /* Intra-Group Queuing Put Authority */
 #define MQIGQPA_DEFAULT                1
 #define MQIGQPA_CONTEXT                2
 #define MQIGQPA_ONLY_IGQ               3
 #define MQIGQPA_ALTERNATE_OR_IGQ       4

 /* IP Address Versions */
 #define MQIPADDR_IPV4                  0
 #define MQIPADDR_IPV6                  1

 /* Message Mark-Browse Interval */
 #define MQMMBI_UNLIMITED               (-1)

 /* Monitoring Values */
 #define MQMON_NOT_AVAILABLE            (-1)
 #define MQMON_NONE                     (-1)
 #define MQMON_Q_MGR                    (-3)
 #define MQMON_OFF                      0
 #define MQMON_ON                       1
 #define MQMON_DISABLED                 0
 #define MQMON_ENABLED                  1
 #define MQMON_LOW                      17
 #define MQMON_MEDIUM                   33
 #define MQMON_HIGH                     65

 /* Application Function Types */
 #define MQFUN_TYPE_UNKNOWN             0
 #define MQFUN_TYPE_JVM                 1
 #define MQFUN_TYPE_PROGRAM             2
 #define MQFUN_TYPE_PROCEDURE           3
 #define MQFUN_TYPE_USERDEF             4
 #define MQFUN_TYPE_COMMAND             5

 /* Application Activity Trace Detail */
 #define MQACTV_DETAIL_LOW              1
 #define MQACTV_DETAIL_MEDIUM           2
 #define MQACTV_DETAIL_HIGH             3

 /* Platforms */
 #define MQPL_MVS                       1
 #define MQPL_OS390                     1
 #define MQPL_ZOS                       1
 #define MQPL_OS2                       2
 #define MQPL_AIX                       3
 #define MQPL_UNIX                      3
 #define MQPL_OS400                     4
 #define MQPL_WINDOWS                   5
 #define MQPL_WINDOWS_NT                11
 #define MQPL_VMS                       12
 #define MQPL_NSK                       13
 #define MQPL_NSS                       13
 #define MQPL_OPEN_TP1                  15
 #define MQPL_VM                        18
 #define MQPL_TPF                       23
 #define MQPL_VSE                       27
 #define MQPL_APPLIANCE                 28
 #define MQPL_NATIVE                    3

 /* Maximum Properties Length */
 #define MQPROP_UNRESTRICTED_LENGTH     (-1)

 /* Pub/Sub Mode */
 #define MQPSM_DISABLED                 0
 #define MQPSM_COMPAT                   1
 #define MQPSM_ENABLED                  2

 /* Pub/Sub clusters */
 #define MQPSCLUS_DISABLED              0
 #define MQPSCLUS_ENABLED               1

 /* Control Options */
 #define MQQMOPT_DISABLED               0
 #define MQQMOPT_ENABLED                1
 #define MQQMOPT_REPLY                  2

 /* Receive Timeout Types */
 #define MQRCVTIME_MULTIPLY             0
 #define MQRCVTIME_ADD                  1
 #define MQRCVTIME_EQUAL                2

 /* Recording Options */
 #define MQRECORDING_DISABLED           0
 #define MQRECORDING_Q                  1
 #define MQRECORDING_MSG                2

 /* Security Case */
 #define MQSCYC_UPPER                   0
 #define MQSCYC_MIXED                   1

 /* Shared Queue Queue Manager Name */
 #define MQSQQM_USE                     0
 #define MQSQQM_IGNORE                  1

 /* SSL FIPS Requirements */
 #define MQSSL_FIPS_NO                  0
 #define MQSSL_FIPS_YES                 1

 /* Syncpoint Availability */
 #define MQSP_AVAILABLE                 1
 #define MQSP_NOT_AVAILABLE             0

 /* Service Controls */
 #define MQSVC_CONTROL_Q_MGR            0
 #define MQSVC_CONTROL_Q_MGR_START      1
 #define MQSVC_CONTROL_MANUAL           2

 /* Service Status */
 #define MQSVC_STATUS_STOPPED           0
 #define MQSVC_STATUS_STARTING          1
 #define MQSVC_STATUS_RUNNING           2
 #define MQSVC_STATUS_STOPPING          3
 #define MQSVC_STATUS_RETRYING          4

 /* TCP Keepalive */
 #define MQTCPKEEP_NO                   0
 #define MQTCPKEEP_YES                  1

 /* TCP Stack Types */
 #define MQTCPSTACK_SINGLE              0
 #define MQTCPSTACK_MULTIPLE            1

 /* Channel Initiator Trace Autostart */
 #define MQTRAXSTR_NO                   0
 #define MQTRAXSTR_YES                  1

 /* Capability */
 #define MQCAP_NOT_SUPPORTED            0
 #define MQCAP_SUPPORTED                1
 #define MQCAP_EXPIRED                  2

 /* Media Image Scheduling */
 #define MQMEDIMGSCHED_MANUAL           0
 #define MQMEDIMGSCHED_AUTO             1

 /* Automatic Media Image Interval */
 #define MQMEDIMGINTVL_OFF              0

 /* Automatic Media Image Log Length */
 #define MQMEDIMGLOGLN_OFF              0

 /* Media Image Recoverability */
 #define MQIMGRCOV_NO                   0
 #define MQIMGRCOV_YES                  1
 #define MQIMGRCOV_AS_Q_MGR             2

 /****************************************************************/
 /* Values Related to Topic Attributes                           */
 /****************************************************************/

 /* Persistent/Non-persistent Message Delivery */
 #define MQDLV_AS_PARENT                0
 #define MQDLV_ALL                      1
 #define MQDLV_ALL_DUR                  2
 #define MQDLV_ALL_AVAIL                3

 /* Master administration */
 #define MQMASTER_NO                    0
 #define MQMASTER_YES                   1

 /* Publish scope */
 #define MQSCOPE_ALL                    0
 #define MQSCOPE_AS_PARENT              1
 #define MQSCOPE_QMGR                   4

 /* Durable subscriptions */
 #define MQSUB_DURABLE_AS_PARENT        0
 #define MQSUB_DURABLE_ALLOWED          1
 #define MQSUB_DURABLE_INHIBITED        2

 /* Wildcards */
 #define MQTA_BLOCK                     1
 #define MQTA_PASSTHRU                  2

 /* Subscriptions Allowed */
 #define MQTA_SUB_AS_PARENT             0
 #define MQTA_SUB_INHIBITED             1
 #define MQTA_SUB_ALLOWED               2

 /* Proxy Sub Propagation */
 #define MQTA_PROXY_SUB_FORCE           1
 #define MQTA_PROXY_SUB_FIRSTUSE        2

 /* Publications Allowed */
 #define MQTA_PUB_AS_PARENT             0
 #define MQTA_PUB_INHIBITED             1
 #define MQTA_PUB_ALLOWED               2

 /* Topic Type */
 #define MQTOPT_LOCAL                   0
 #define MQTOPT_CLUSTER                 1
 #define MQTOPT_ALL                     2

 /* Multicast */
 #define MQMC_AS_PARENT                 0
 #define MQMC_ENABLED                   1
 #define MQMC_DISABLED                  2
 #define MQMC_ONLY                      3

 /* CommInfo Type */
 #define MQCIT_MULTICAST                1

 /****************************************************************/
 /* Values Related to Subscription Attributes                    */
 /****************************************************************/

 /* Destination Class */
 #define MQDC_MANAGED                   1
 #define MQDC_PROVIDED                  2

 /* Pub/Sub Message Properties */
 #define MQPSPROP_NONE                  0
 #define MQPSPROP_COMPAT                1
 #define MQPSPROP_RFH2                  2
 #define MQPSPROP_MSGPROP               3

 /* Request Only */
 #define MQRU_PUBLISH_ON_REQUEST        1
 #define MQRU_PUBLISH_ALL               2

 /* Durable Subscriptions */
 #define MQSUB_DURABLE_ALL              (-1)
 #define MQSUB_DURABLE_YES              1
 #define MQSUB_DURABLE_NO               2

 /* Subscription Scope */
 #define MQTSCOPE_QMGR                  1
 #define MQTSCOPE_ALL                   2

 /* Variable User ID */
 #define MQVU_FIXED_USER                1
 #define MQVU_ANY_USER                  2

 /* Wildcard Schema */
 #define MQWS_DEFAULT                   0
 #define MQWS_CHAR                      1
 #define MQWS_TOPIC                     2

 /****************************************************************/
 /* Values Related to Channel Authentication Configuration       */
 /* Attributes                                                   */
 /****************************************************************/

 /* User Source Options */
 #define MQUSRC_MAP                     0
 #define MQUSRC_NOACCESS                1
 #define MQUSRC_CHANNEL                 2

 /* Warn Options */
 #define MQWARN_YES                     1
 #define MQWARN_NO                      0

 /* DSBlock Options */
 #define MQDSB_DEFAULT                  0
 #define MQDSB_8K                       1
 #define MQDSB_16K                      2
 #define MQDSB_32K                      3
 #define MQDSB_64K                      4
 #define MQDSB_128K                     5
 #define MQDSB_256K                     6
 #define MQDSB_512K                     7
 #define MQDSB_1024K                    8
 #define MQDSB_1M                       8

 /* DSExpand Options */
 #define MQDSE_DEFAULT                  0
 #define MQDSE_YES                      1
 #define MQDSE_NO                       2

 /* OffldUse Options */
 #define MQCFOFFLD_NONE                 0
 #define MQCFOFFLD_SMDS                 1
 #define MQCFOFFLD_DB2                  2
 #define MQCFOFFLD_BOTH                 3

 /* Use Dead Letter Queue Options */
 #define MQUSEDLQ_AS_PARENT             0
 #define MQUSEDLQ_NO                    1
 #define MQUSEDLQ_YES                   2

 /****************************************************************/
 /* Constants for MQ Extended Reach                              */
 /****************************************************************/

 /* General Constants */
 #define MQ_MQTT_MAX_KEEP_ALIVE         65536
 #define MQ_SSL_KEY_PASSPHRASE_LENGTH   1024

 /****************************************************************/
 /* Values Related to MQCLOSE Function                           */
 /****************************************************************/

 /* Object Handle */
 #define MQHO_UNUSABLE_HOBJ             (-1)
 #define MQHO_NONE                      0

 /* Close Options */
 #define MQCO_IMMEDIATE                 0x00000000
 #define MQCO_NONE                      0x00000000
 #define MQCO_DELETE                    0x00000001
 #define MQCO_DELETE_PURGE              0x00000002
 #define MQCO_KEEP_SUB                  0x00000004
 #define MQCO_REMOVE_SUB                0x00000008
 #define MQCO_QUIESCE                   0x00000020

 /****************************************************************/
 /* Values Related to MQCTL and MQCB Functions                   */
 /****************************************************************/

 /* Operation codes for MQCTL */
 #define MQOP_START                     0x00000001
 #define MQOP_START_WAIT                0x00000002
 #define MQOP_STOP                      0x00000004

 /* Operation codes for MQCB */
 #define MQOP_REGISTER                  0x00000100
 #define MQOP_DEREGISTER                0x00000200

 /* Operation codes for MQCTL and MQCB */
 #define MQOP_SUSPEND                   0x00010000
 #define MQOP_RESUME                    0x00020000

 /****************************************************************/
 /* Values Related to MQDLTMH Function                           */
 /****************************************************************/

 /* Message handle */
 #define MQHM_UNUSABLE_HMSG             (-1)
 #define MQHM_NONE                      0

 /****************************************************************/
 /* Values Related to MQINQ Function                             */
 /****************************************************************/

 /* Byte Attribute Selectors */
 #define MQBA_FIRST                     6001
 #define MQBA_LAST                      8000

 /* Character Attribute Selectors */
 #define MQCA_ADMIN_TOPIC_NAME          2105
 #define MQCA_ALTERATION_DATE           2027
 #define MQCA_ALTERATION_TIME           2028
 #define MQCA_AMQP_SSL_CIPHER_SUITES    2137
 #define MQCA_AMQP_VERSION              2136
 #define MQCA_APPL_ID                   2001
 #define MQCA_AUTH_INFO_CONN_NAME       2053
 #define MQCA_AUTH_INFO_DESC            2046
 #define MQCA_AUTH_INFO_NAME            2045
 #define MQCA_AUTH_INFO_OCSP_URL        2109
 #define MQCA_AUTO_REORG_CATALOG        2091
 #define MQCA_AUTO_REORG_START_TIME     2090
 #define MQCA_BACKOUT_REQ_Q_NAME        2019
 #define MQCA_BASE_OBJECT_NAME          2002
 #define MQCA_BASE_Q_NAME               2002
 #define MQCA_BATCH_INTERFACE_ID        2068
 #define MQCA_CERT_LABEL                2121
 #define MQCA_CF_STRUC_DESC             2052
 #define MQCA_CF_STRUC_NAME             2039
 #define MQCA_CHANNEL_AUTO_DEF_EXIT     2026
 #define MQCA_CHILD                     2101
 #define MQCA_CHINIT_SERVICE_PARM       2076
 #define MQCA_CHLAUTH_DESC              2118
 #define MQCA_CICS_FILE_NAME            2060
 #define MQCA_CLUSTER_DATE              2037
 #define MQCA_CLUSTER_NAME              2029
 #define MQCA_CLUSTER_NAMELIST          2030
 #define MQCA_CLUSTER_Q_MGR_NAME        2031
 #define MQCA_CLUSTER_TIME              2038
 #define MQCA_CLUSTER_WORKLOAD_DATA     2034
 #define MQCA_CLUSTER_WORKLOAD_EXIT     2033
 #define MQCA_CLUS_CHL_NAME             2124
 #define MQCA_COMMAND_INPUT_Q_NAME      2003
 #define MQCA_COMMAND_REPLY_Q_NAME      2067
 #define MQCA_COMM_INFO_DESC            2111
 #define MQCA_COMM_INFO_NAME            2110
 #define MQCA_CONN_AUTH                 2125
 #define MQCA_CREATION_DATE             2004
 #define MQCA_CREATION_TIME             2005
 #define MQCA_CUSTOM                    2119
 #define MQCA_DEAD_LETTER_Q_NAME        2006
 #define MQCA_DEF_XMIT_Q_NAME           2025
 #define MQCA_DNS_GROUP                 2071
 #define MQCA_ENV_DATA                  2007
 #define MQCA_FIRST                     2001
 #define MQCA_IGQ_USER_ID               2041
 #define MQCA_INITIATION_Q_NAME         2008
 #define MQCA_INSTALLATION_DESC         2115
 #define MQCA_INSTALLATION_NAME         2116
 #define MQCA_INSTALLATION_PATH         2117
 #define MQCA_LAST                      4000
 #define MQCA_LAST_USED                 2137
 #define MQCA_LDAP_BASE_DN_GROUPS       2132
 #define MQCA_LDAP_BASE_DN_USERS        2126
 #define MQCA_LDAP_FIND_GROUP_FIELD     2135
 #define MQCA_LDAP_GROUP_ATTR_FIELD     2134
 #define MQCA_LDAP_GROUP_OBJECT_CLASS   2133
 #define MQCA_LDAP_PASSWORD             2048
 #define MQCA_LDAP_SHORT_USER_FIELD     2127
 #define MQCA_LDAP_USER_ATTR_FIELD      2129
 #define MQCA_LDAP_USER_NAME            2047
 #define MQCA_LDAP_USER_OBJECT_CLASS    2128
 #define MQCA_LU62_ARM_SUFFIX           2074
 #define MQCA_LU_GROUP_NAME             2072
 #define MQCA_LU_NAME                   2073
 #define MQCA_MODEL_DURABLE_Q           2096
 #define MQCA_MODEL_NON_DURABLE_Q       2097
 #define MQCA_MONITOR_Q_NAME            2066
 #define MQCA_NAMELIST_DESC             2009
 #define MQCA_NAMELIST_NAME             2010
 #define MQCA_NAMES                     2020
 #define MQCA_PARENT                    2102
 #define MQCA_PASS_TICKET_APPL          2086
 #define MQCA_POLICY_NAME               2112
 #define MQCA_PROCESS_DESC              2011
 #define MQCA_PROCESS_NAME              2012
 #define MQCA_QSG_CERT_LABEL            2131
 #define MQCA_QSG_NAME                  2040
 #define MQCA_Q_DESC                    2013
 #define MQCA_Q_MGR_DESC                2014
 #define MQCA_Q_MGR_IDENTIFIER          2032
 #define MQCA_Q_MGR_NAME                2015
 #define MQCA_Q_NAME                    2016
 #define MQCA_RECIPIENT_DN              2114
 #define MQCA_REMOTE_Q_MGR_NAME         2017
 #define MQCA_REMOTE_Q_NAME             2018
 #define MQCA_REPOSITORY_NAME           2035
 #define MQCA_REPOSITORY_NAMELIST       2036
 #define MQCA_RESUME_DATE               2098
 #define MQCA_RESUME_TIME               2099
 #define MQCA_SERVICE_DESC              2078
 #define MQCA_SERVICE_NAME              2077
 #define MQCA_SERVICE_START_ARGS        2080
 #define MQCA_SERVICE_START_COMMAND     2079
 #define MQCA_SERVICE_STOP_ARGS         2082
 #define MQCA_SERVICE_STOP_COMMAND      2081
 #define MQCA_SIGNER_DN                 2113
 #define MQCA_SSL_CERT_ISSUER_NAME      2130
 #define MQCA_SSL_CRL_NAMELIST          2050
 #define MQCA_SSL_CRYPTO_HARDWARE       2051
 #define MQCA_SSL_KEY_LIBRARY           2069
 #define MQCA_SSL_KEY_MEMBER            2070
 #define MQCA_SSL_KEY_REPOSITORY        2049
 #define MQCA_STDERR_DESTINATION        2084
 #define MQCA_STDOUT_DESTINATION        2083
 #define MQCA_STORAGE_CLASS             2022
 #define MQCA_STORAGE_CLASS_DESC        2042
 #define MQCA_SYSTEM_LOG_Q_NAME         2065
 #define MQCA_TCP_NAME                  2075
 #define MQCA_TOPIC_DESC                2093
 #define MQCA_TOPIC_NAME                2092
 #define MQCA_TOPIC_STRING              2094
 #define MQCA_TOPIC_STRING_FILTER       2108
 #define MQCA_TPIPE_NAME                2085
 #define MQCA_TRIGGER_CHANNEL_NAME      2064
 #define MQCA_TRIGGER_DATA              2023
 #define MQCA_TRIGGER_PROGRAM_NAME      2062
 #define MQCA_TRIGGER_TERM_ID           2063
 #define MQCA_TRIGGER_TRANS_ID          2061
 #define MQCA_USER_DATA                 2021
 #define MQCA_USER_LIST                 4000
 #define MQCA_VERSION                   2120
 #define MQCA_XCF_GROUP_NAME            2043
 #define MQCA_XCF_MEMBER_NAME           2044
 #define MQCA_XMIT_Q_NAME               2024
 #define MQCA_XR_SSL_CIPHER_SUITES      2123
 #define MQCA_XR_VERSION                2122

 /* Integer Attribute Selectors */
 #define MQIA_ACCOUNTING_CONN_OVERRIDE  136
 #define MQIA_ACCOUNTING_INTERVAL       135
 #define MQIA_ACCOUNTING_MQI            133
 #define MQIA_ACCOUNTING_Q              134
 #define MQIA_ACTIVE_CHANNELS           100
 #define MQIA_ACTIVITY_CONN_OVERRIDE    239
 #define MQIA_ACTIVITY_RECORDING        138
 #define MQIA_ACTIVITY_TRACE            240
 #define MQIA_ADOPTNEWMCA_CHECK         102
 #define MQIA_ADOPTNEWMCA_INTERVAL      104
 #define MQIA_ADOPTNEWMCA_TYPE          103
 #define MQIA_ADOPT_CONTEXT             260
 #define MQIA_ADVANCED_CAPABILITY       273
 #define MQIA_AMQP_CAPABILITY           265
 #define MQIA_APPL_TYPE                 1
 #define MQIA_ARCHIVE                   60
 #define MQIA_AUTHENTICATION_FAIL_DELAY 259
 #define MQIA_AUTHENTICATION_METHOD     266
 #define MQIA_AUTHORITY_EVENT           47
 #define MQIA_AUTH_INFO_TYPE            66
 #define MQIA_AUTO_REORGANIZATION       173
 #define MQIA_AUTO_REORG_INTERVAL       174
 #define MQIA_BACKOUT_THRESHOLD         22
 #define MQIA_BASE_TYPE                 193
 #define MQIA_BATCH_INTERFACE_AUTO      86
 #define MQIA_BRIDGE_EVENT              74
 #define MQIA_CERT_VAL_POLICY           252
 #define MQIA_CF_CFCONLOS               246
 #define MQIA_CF_LEVEL                  70
 #define MQIA_CF_OFFLDUSE               229
 #define MQIA_CF_OFFLOAD                224
 #define MQIA_CF_OFFLOAD_THRESHOLD1     225
 #define MQIA_CF_OFFLOAD_THRESHOLD2     226
 #define MQIA_CF_OFFLOAD_THRESHOLD3     227
 #define MQIA_CF_RECAUTO                244
 #define MQIA_CF_RECOVER                71
 #define MQIA_CF_SMDS_BUFFERS           228
 #define MQIA_CHANNEL_AUTO_DEF          55
 #define MQIA_CHANNEL_AUTO_DEF_EVENT    56
 #define MQIA_CHANNEL_EVENT             73
 #define MQIA_CHECK_CLIENT_BINDING      258
 #define MQIA_CHECK_LOCAL_BINDING       257
 #define MQIA_CHINIT_ADAPTERS           101
 #define MQIA_CHINIT_CONTROL            119
 #define MQIA_CHINIT_DISPATCHERS        105
 #define MQIA_CHINIT_TRACE_AUTO_START   117
 #define MQIA_CHINIT_TRACE_TABLE_SIZE   118
 #define MQIA_CHLAUTH_RECORDS           248
 #define MQIA_CLUSTER_OBJECT_STATE      256
 #define MQIA_CLUSTER_PUB_ROUTE         255
 #define MQIA_CLUSTER_Q_TYPE            59
 #define MQIA_CLUSTER_WORKLOAD_LENGTH   58
 #define MQIA_CLWL_MRU_CHANNELS         97
 #define MQIA_CLWL_Q_PRIORITY           96
 #define MQIA_CLWL_Q_RANK               95
 #define MQIA_CLWL_USEQ                 98
 #define MQIA_CMD_SERVER_AUTO           87
 #define MQIA_CMD_SERVER_CONTROL        120
 #define MQIA_CMD_SERVER_CONVERT_MSG    88
 #define MQIA_CMD_SERVER_DLQ_MSG        89
 #define MQIA_CODED_CHAR_SET_ID         2
 #define MQIA_COMMAND_EVENT             99
 #define MQIA_COMMAND_LEVEL             31
 #define MQIA_COMM_EVENT                232
 #define MQIA_COMM_INFO_TYPE            223
 #define MQIA_CONFIGURATION_EVENT       51
 #define MQIA_CPI_LEVEL                 27
 #define MQIA_CURRENT_Q_DEPTH           3
 #define MQIA_DEFINITION_TYPE           7
 #define MQIA_DEF_BIND                  61
 #define MQIA_DEF_CLUSTER_XMIT_Q_TYPE   250
 #define MQIA_DEF_INPUT_OPEN_OPTION     4
 #define MQIA_DEF_PERSISTENCE           5
 #define MQIA_DEF_PRIORITY              6
 #define MQIA_DEF_PUT_RESPONSE_TYPE     184
 #define MQIA_DEF_READ_AHEAD            188
 #define MQIA_DISPLAY_TYPE              262
 #define MQIA_DIST_LISTS                34
 #define MQIA_DNS_WLM                   106
 #define MQIA_DURABLE_SUB               175
 #define MQIA_ENCRYPTION_ALGORITHM      237
 #define MQIA_EXPIRY_INTERVAL           39
 #define MQIA_FIRST                     1
 #define MQIA_GROUP_UR                  221
 #define MQIA_HARDEN_GET_BACKOUT        8
 #define MQIA_HIGH_Q_DEPTH              36
 #define MQIA_IGQ_PUT_AUTHORITY         65
 #define MQIA_INDEX_TYPE                57
 #define MQIA_INHIBIT_EVENT             48
 #define MQIA_INHIBIT_GET               9
 #define MQIA_INHIBIT_PUB               181
 #define MQIA_INHIBIT_PUT               10
 #define MQIA_INHIBIT_SUB               182
 #define MQIA_INTRA_GROUP_QUEUING       64
 #define MQIA_IP_ADDRESS_VERSION        93
 #define MQIA_KEY_REUSE_COUNT           267
 #define MQIA_LAST                      2000
 #define MQIA_LAST_USED                 273
 #define MQIA_LDAP_AUTHORMD             263
 #define MQIA_LDAP_NESTGRP              264
 #define MQIA_LDAP_SECURE_COMM          261
 #define MQIA_LISTENER_PORT_NUMBER      85
 #define MQIA_LISTENER_TIMER            107
 #define MQIA_LOCAL_EVENT               49
 #define MQIA_LOGGER_EVENT              94
 #define MQIA_LU62_CHANNELS             108
 #define MQIA_MASTER_ADMIN              186
 #define MQIA_MAX_CHANNELS              109
 #define MQIA_MAX_CLIENTS               172
 #define MQIA_MAX_GLOBAL_LOCKS          83
 #define MQIA_MAX_HANDLES               11
 #define MQIA_MAX_LOCAL_LOCKS           84
 #define MQIA_MAX_MSG_LENGTH            13
 #define MQIA_MAX_OPEN_Q                80
 #define MQIA_MAX_PRIORITY              14
 #define MQIA_MAX_PROPERTIES_LENGTH     192
 #define MQIA_MAX_Q_DEPTH               15
 #define MQIA_MAX_Q_TRIGGERS            90
 #define MQIA_MAX_RECOVERY_TASKS        171
 #define MQIA_MAX_RESPONSES             230
 #define MQIA_MAX_UNCOMMITTED_MSGS      33
 #define MQIA_MCAST_BRIDGE              233
 #define MQIA_MEDIA_IMAGE_INTERVAL      269
 #define MQIA_MEDIA_IMAGE_LOG_LENGTH    270
 #define MQIA_MEDIA_IMAGE_RECOVER_OBJ   271
 #define MQIA_MEDIA_IMAGE_RECOVER_Q     272
 #define MQIA_MEDIA_IMAGE_SCHEDULING    268
 #define MQIA_MONITORING_AUTO_CLUSSDR   124
 #define MQIA_MONITORING_CHANNEL        122
 #define MQIA_MONITORING_Q              123
 #define MQIA_MONITOR_INTERVAL          81
 #define MQIA_MSG_DELIVERY_SEQUENCE     16
 #define MQIA_MSG_DEQ_COUNT             38
 #define MQIA_MSG_ENQ_COUNT             37
 #define MQIA_MSG_MARK_BROWSE_INTERVAL  68
 #define MQIA_MULTICAST                 176
 #define MQIA_NAMELIST_TYPE             72
 #define MQIA_NAME_COUNT                19
 #define MQIA_NPM_CLASS                 78
 #define MQIA_NPM_DELIVERY              196
 #define MQIA_OPEN_INPUT_COUNT          17
 #define MQIA_OPEN_OUTPUT_COUNT         18
 #define MQIA_OUTBOUND_PORT_MAX         140
 #define MQIA_OUTBOUND_PORT_MIN         110
 #define MQIA_PAGESET_ID                62
 #define MQIA_PERFORMANCE_EVENT         53
 #define MQIA_PLATFORM                  32
 #define MQIA_PM_DELIVERY               195
 #define MQIA_POLICY_VERSION            238
 #define MQIA_PROPERTY_CONTROL          190
 #define MQIA_PROT_POLICY_CAPABILITY    251
 #define MQIA_PROXY_SUB                 199
 #define MQIA_PUBSUB_CLUSTER            249
 #define MQIA_PUBSUB_MAXMSG_RETRY_COUNT 206
 #define MQIA_PUBSUB_MODE               187
 #define MQIA_PUBSUB_NP_MSG             203
 #define MQIA_PUBSUB_NP_RESP            205
 #define MQIA_PUBSUB_SYNC_PT            207
 #define MQIA_PUB_COUNT                 215
 #define MQIA_PUB_SCOPE                 219
 #define MQIA_QMGR_CFCONLOS             245
 #define MQIA_QMOPT_CONS_COMMS_MSGS     155
 #define MQIA_QMOPT_CONS_CRITICAL_MSGS  154
 #define MQIA_QMOPT_CONS_ERROR_MSGS     153
 #define MQIA_QMOPT_CONS_INFO_MSGS      151
 #define MQIA_QMOPT_CONS_REORG_MSGS     156
 #define MQIA_QMOPT_CONS_SYSTEM_MSGS    157
 #define MQIA_QMOPT_CONS_WARNING_MSGS   152
 #define MQIA_QMOPT_CSMT_ON_ERROR       150
 #define MQIA_QMOPT_INTERNAL_DUMP       170
 #define MQIA_QMOPT_LOG_COMMS_MSGS      162
 #define MQIA_QMOPT_LOG_CRITICAL_MSGS   161
 #define MQIA_QMOPT_LOG_ERROR_MSGS      160
 #define MQIA_QMOPT_LOG_INFO_MSGS       158
 #define MQIA_QMOPT_LOG_REORG_MSGS      163
 #define MQIA_QMOPT_LOG_SYSTEM_MSGS     164
 #define MQIA_QMOPT_LOG_WARNING_MSGS    159
 #define MQIA_QMOPT_TRACE_COMMS         166
 #define MQIA_QMOPT_TRACE_CONVERSION    168
 #define MQIA_QMOPT_TRACE_MQI_CALLS     165
 #define MQIA_QMOPT_TRACE_REORG         167
 #define MQIA_QMOPT_TRACE_SYSTEM        169
 #define MQIA_QSG_DISP                  63
 #define MQIA_Q_DEPTH_HIGH_EVENT        43
 #define MQIA_Q_DEPTH_HIGH_LIMIT        40
 #define MQIA_Q_DEPTH_LOW_EVENT         44
 #define MQIA_Q_DEPTH_LOW_LIMIT         41
 #define MQIA_Q_DEPTH_MAX_EVENT         42
 #define MQIA_Q_SERVICE_INTERVAL        54
 #define MQIA_Q_SERVICE_INTERVAL_EVENT  46
 #define MQIA_Q_TYPE                    20
 #define MQIA_Q_USERS                   82
 #define MQIA_READ_AHEAD                189
 #define MQIA_RECEIVE_TIMEOUT           111
 #define MQIA_RECEIVE_TIMEOUT_MIN       113
 #define MQIA_RECEIVE_TIMEOUT_TYPE      112
 #define MQIA_REMOTE_EVENT              50
 #define MQIA_RESPONSE_RESTART_POINT    231
 #define MQIA_RETENTION_INTERVAL        21
 #define MQIA_REVERSE_DNS_LOOKUP        254
 #define MQIA_SCOPE                     45
 #define MQIA_SECURITY_CASE             141
 #define MQIA_SERVICE_CONTROL           139
 #define MQIA_SERVICE_TYPE              121
 #define MQIA_SHAREABILITY              23
 #define MQIA_SHARED_Q_Q_MGR_NAME       77
 #define MQIA_SIGNATURE_ALGORITHM       236
 #define MQIA_SSL_EVENT                 75
 #define MQIA_SSL_FIPS_REQUIRED         92
 #define MQIA_SSL_RESET_COUNT           76
 #define MQIA_SSL_TASKS                 69
 #define MQIA_START_STOP_EVENT          52
 #define MQIA_STATISTICS_AUTO_CLUSSDR   130
 #define MQIA_STATISTICS_CHANNEL        129
 #define MQIA_STATISTICS_INTERVAL       131
 #define MQIA_STATISTICS_MQI            127
 #define MQIA_STATISTICS_Q              128
 #define MQIA_SUB_CONFIGURATION_EVENT   242
 #define MQIA_SUB_COUNT                 204
 #define MQIA_SUB_SCOPE                 218
 #define MQIA_SUITE_B_STRENGTH          247
 #define MQIA_SYNCPOINT                 30
 #define MQIA_TCP_CHANNELS              114
 #define MQIA_TCP_KEEP_ALIVE            115
 #define MQIA_TCP_STACK_TYPE            116
 #define MQIA_TIME_SINCE_RESET          35
 #define MQIA_TOLERATE_UNPROTECTED      235
 #define MQIA_TOPIC_DEF_PERSISTENCE     185
 #define MQIA_TOPIC_NODE_COUNT          253
 #define MQIA_TOPIC_TYPE                208
 #define MQIA_TRACE_ROUTE_RECORDING     137
 #define MQIA_TREE_LIFE_TIME            183
 #define MQIA_TRIGGER_CONTROL           24
 #define MQIA_TRIGGER_DEPTH             29
 #define MQIA_TRIGGER_INTERVAL          25
 #define MQIA_TRIGGER_MSG_PRIORITY      26
 #define MQIA_TRIGGER_RESTART           91
 #define MQIA_TRIGGER_TYPE              28
 #define MQIA_UR_DISP                   222
 #define MQIA_USAGE                     12
 #define MQIA_USER_LIST                 2000
 #define MQIA_USE_DEAD_LETTER_Q         234
 #define MQIA_WILDCARD_OPERATION        216
 #define MQIA_XR_CAPABILITY             243

 /* Integer Attribute Values */
 #define MQIAV_NOT_APPLICABLE           (-1)
 #define MQIAV_UNDEFINED                (-2)

 /* CommInfo Bridge */
 #define MQMCB_DISABLED                 0
 #define MQMCB_ENABLED                  1

 /* Key reuse count */
 #define MQKEY_REUSE_DISABLED           0
 #define MQKEY_REUSE_UNLIMITED          (-1)

 /* Group Attribute Selectors */
 #define MQGA_FIRST                     8001
 #define MQGA_LAST                      9000

 /****************************************************************/
 /* Values Related to MQINQMP Function                           */
 /****************************************************************/

 /* Inquire on all properties -  "%" */
 #define MQPROP_INQUIRE_ALL     (MQPTR)(char*)"%",\
                                 0,\
                                 0,\
                                 1,\
                                 MQCCSI_APPL

 /* Inquire on all 'usr' properties - "usr.%" */
 #define MQPROP_INQUIRE_ALL_USR (MQPTR)(char*)"usr.%",\
                                 0,\
                                 0,\
                                 5,\
                                 MQCCSI_APPL

 /****************************************************************/
 /* Values Related to MQOPEN Function                            */
 /****************************************************************/

 /* Open Options */
 #define MQOO_BIND_AS_Q_DEF             0x00000000
 #define MQOO_READ_AHEAD_AS_Q_DEF       0x00000000
 #define MQOO_INPUT_AS_Q_DEF            0x00000001
 #define MQOO_INPUT_SHARED              0x00000002
 #define MQOO_INPUT_EXCLUSIVE           0x00000004
 #define MQOO_BROWSE                    0x00000008
 #define MQOO_OUTPUT                    0x00000010
 #define MQOO_INQUIRE                   0x00000020
 #define MQOO_SET                       0x00000040
 #define MQOO_SAVE_ALL_CONTEXT          0x00000080
 #define MQOO_PASS_IDENTITY_CONTEXT     0x00000100
 #define MQOO_PASS_ALL_CONTEXT          0x00000200
 #define MQOO_SET_IDENTITY_CONTEXT      0x00000400
 #define MQOO_SET_ALL_CONTEXT           0x00000800
 #define MQOO_ALTERNATE_USER_AUTHORITY  0x00001000
 #define MQOO_FAIL_IF_QUIESCING         0x00002000
 #define MQOO_BIND_ON_OPEN              0x00004000
 #define MQOO_BIND_ON_GROUP             0x00400000
 #define MQOO_BIND_NOT_FIXED            0x00008000
 #define MQOO_CO_OP                     0x00020000
 #define MQOO_NO_READ_AHEAD             0x00080000
 #define MQOO_READ_AHEAD                0x00100000
 #define MQOO_NO_MULTICAST              0x00200000
 #define MQOO_RESOLVE_LOCAL_Q           0x00040000
 #define MQOO_RESOLVE_LOCAL_TOPIC       0x00040000

 /* Following used in C++ only */
 #define MQOO_RESOLVE_NAMES             0x00010000

 /****************************************************************/
 /* Values Related to MQSETMP Function                           */
 /****************************************************************/

 /* Property data types */
 #define MQTYPE_AS_SET                  0x00000000
 #define MQTYPE_NULL                    0x00000002
 #define MQTYPE_BOOLEAN                 0x00000004
 #define MQTYPE_BYTE_STRING             0x00000008
 #define MQTYPE_INT8                    0x00000010
 #define MQTYPE_INT16                   0x00000020
 #define MQTYPE_INT32                   0x00000040
 #define MQTYPE_LONG                    0x00000040
 #define MQTYPE_INT64                   0x00000080
 #define MQTYPE_FLOAT32                 0x00000100
 #define MQTYPE_FLOAT64                 0x00000200
 #define MQTYPE_STRING                  0x00000400

 /* Property value lengths */
 #define MQVL_NULL_TERMINATED           (-1)
 #define MQVL_EMPTY_STRING              0

 /****************************************************************/
 /* Values Related to MQSTAT Function                            */
 /****************************************************************/

 /* Stat Options */
 #define MQSTAT_TYPE_ASYNC_ERROR        0
 #define MQSTAT_TYPE_RECONNECTION       1
 #define MQSTAT_TYPE_RECONNECTION_ERROR 2

 /****************************************************************/
 /* Values Related to MQSUB Function                             */
 /****************************************************************/

 /* Subscribe Options */
 #define MQSO_NONE                      0x00000000
 #define MQSO_NON_DURABLE               0x00000000
 #define MQSO_READ_AHEAD_AS_Q_DEF       0x00000000
 #define MQSO_ALTER                     0x00000001
 #define MQSO_CREATE                    0x00000002
 #define MQSO_RESUME                    0x00000004
 #define MQSO_DURABLE                   0x00000008
 #define MQSO_GROUP_SUB                 0x00000010
 #define MQSO_MANAGED                   0x00000020
 #define MQSO_SET_IDENTITY_CONTEXT      0x00000040
 #define MQSO_NO_MULTICAST              0x00000080
 #define MQSO_FIXED_USERID              0x00000100
 #define MQSO_ANY_USERID                0x00000200
 #define MQSO_PUBLICATIONS_ON_REQUEST   0x00000800
 #define MQSO_NEW_PUBLICATIONS_ONLY     0x00001000
 #define MQSO_FAIL_IF_QUIESCING         0x00002000
 #define MQSO_ALTERNATE_USER_AUTHORITY  0x00040000
 #define MQSO_WILDCARD_CHAR             0x00100000
 #define MQSO_WILDCARD_TOPIC            0x00200000
 #define MQSO_SET_CORREL_ID             0x00400000
 #define MQSO_SCOPE_QMGR                0x04000000
 #define MQSO_NO_READ_AHEAD             0x08000000
 #define MQSO_READ_AHEAD                0x10000000

 /****************************************************************/
 /* Values Related to MQSUBRQ Function                           */
 /****************************************************************/

 /* Action */
 #define MQSR_ACTION_PUBLICATION        1

 /****************************************************************/
 /*   Simple Data Types                                          */
 /****************************************************************/

 /* Byte Datatypes */
 typedef unsigned char MQBYTE;
 typedef MQBYTE MQPOINTER PMQBYTE;
 typedef PMQBYTE MQPOINTER PPMQBYTE;
 typedef MQBYTE MQBYTE4[4];
 typedef MQBYTE4 MQPOINTER PMQBYTE4;
 typedef MQBYTE MQBYTE8[8];
 typedef MQBYTE8 MQPOINTER PMQBYTE8;
 typedef MQBYTE MQBYTE16[16];
 typedef MQBYTE16 MQPOINTER PMQBYTE16;
 typedef MQBYTE MQBYTE24[24];
 typedef MQBYTE24 MQPOINTER PMQBYTE24;
 typedef MQBYTE MQBYTE32[32];
 typedef MQBYTE32 MQPOINTER PMQBYTE32;
 typedef MQBYTE MQBYTE40[40];
 typedef MQBYTE40 MQPOINTER PMQBYTE40;
 typedef MQBYTE MQBYTE48[48];
 typedef MQBYTE48 MQPOINTER PMQBYTE48;
 typedef MQBYTE MQBYTE128[128];
 typedef MQBYTE128 MQPOINTER PMQBYTE128;

 /* Character Datatypes */
 typedef char MQCHAR;
 typedef MQCHAR MQPOINTER PMQCHAR;
 typedef PMQCHAR MQPOINTER PPMQCHAR;
 typedef MQCHAR MQCHAR4[4];
 typedef MQCHAR4 MQPOINTER PMQCHAR4;
 typedef MQCHAR MQCHAR8[8];
 typedef MQCHAR8 MQPOINTER PMQCHAR8;
 typedef MQCHAR MQCHAR12[12];
 typedef MQCHAR12 MQPOINTER PMQCHAR12;
 typedef MQCHAR MQCHAR16[16];
 typedef MQCHAR16 MQPOINTER PMQCHAR16;
 typedef MQCHAR MQCHAR20[20];
 typedef MQCHAR20 MQPOINTER PMQCHAR20;
 typedef MQCHAR MQCHAR28[28];
 typedef MQCHAR28 MQPOINTER PMQCHAR28;
 typedef MQCHAR MQCHAR32[32];
 typedef MQCHAR32 MQPOINTER PMQCHAR32;
 typedef MQCHAR MQCHAR48[48];
 typedef MQCHAR48 MQPOINTER PMQCHAR48;
 typedef MQCHAR MQCHAR64[64];
 typedef MQCHAR64 MQPOINTER PMQCHAR64;
 typedef MQCHAR MQCHAR128[128];
 typedef MQCHAR128 MQPOINTER PMQCHAR128;
 typedef MQCHAR MQCHAR256[256];
 typedef MQCHAR256 MQPOINTER PMQCHAR256;
 typedef MQCHAR MQCHAR264[264];
 typedef MQCHAR264 MQPOINTER PMQCHAR264;

 /* Other Datatypes */
#if defined(MQ_64_BIT)
 typedef int MQLONG;
 typedef unsigned int MQULONG;
 typedef long MQINT64;
 typedef unsigned long MQUINT64;
#else
 typedef long MQLONG;
 typedef unsigned long MQULONG;
 typedef long long MQINT64;
 typedef unsigned long long MQUINT64;
#endif
 typedef MQLONG MQPOINTER PMQLONG;
 typedef PMQLONG MQPOINTER PPMQLONG;
 typedef signed char MQINT8;
 typedef MQINT8 MQPOINTER PMQINT8;
 typedef PMQINT8 MQPOINTER PPMQINT8;
 typedef unsigned char MQUINT8;
 typedef MQUINT8 MQPOINTER PMQUINT8;
 typedef PMQUINT8 MQPOINTER PPMQUINT8;
 typedef short MQINT16;
 typedef MQINT16 MQPOINTER PMQINT16;
 typedef PMQINT16 MQPOINTER PPMQINT16;
 typedef unsigned short MQUINT16;
 typedef MQUINT16 MQPOINTER PMQUINT16;
 typedef PMQUINT16 MQPOINTER PPMQUINT16;
 typedef MQLONG MQINT32;
 typedef PMQLONG PMQINT32;
 typedef PPMQLONG PPMQINT32;
 typedef MQINT64 MQPOINTER PMQINT64;
 typedef PMQINT64 MQPOINTER PPMQINT64;
 typedef MQULONG MQPOINTER PMQULONG;
 typedef PMQULONG MQPOINTER PPMQULONG;
 typedef MQULONG MQUINT32;
 typedef PMQULONG PMQUINT32;
 typedef PPMQULONG PPMQUINT32;
 typedef MQUINT64 MQPOINTER PMQUINT64;
 typedef PMQUINT64 MQPOINTER PPMQUINT64;
 typedef float MQFLOAT32;
 typedef MQFLOAT32 MQPOINTER PMQFLOAT32;
 typedef PMQFLOAT32 MQPOINTER PPMQFLOAT32;
 typedef double MQFLOAT64;
 typedef MQFLOAT64 MQPOINTER PMQFLOAT64;
 typedef PMQFLOAT64 MQPOINTER PPMQFLOAT64;
 typedef struct tagMQIEP MQIEP;
 typedef MQIEP  MQPOINTER PMQIEP;
 typedef PMQIEP MQPOINTER PPMQIEP;
 typedef PMQIEP MQHCONFIG;
 typedef MQHCONFIG MQPOINTER PMQHCONFIG;
 typedef MQLONG MQHCONN;
 typedef MQHCONN MQPOINTER PMQHCONN;
 typedef PMQHCONN MQPOINTER PPMQHCONN;
 typedef MQLONG MQHOBJ;
 typedef MQHOBJ MQPOINTER PMQHOBJ;
 typedef PMQHOBJ MQPOINTER PPMQHOBJ;
 typedef void MQPOINTER MQPTR;
 typedef MQPTR MQPOINTER PMQPTR;
 typedef void MQPOINTER PMQFUNC;
 typedef void MQPOINTER PMQVOID;
 typedef PMQVOID MQPOINTER PPMQVOID;
 typedef MQLONG MQBOOL;
 typedef MQBOOL MQPOINTER PMQBOOL;
 typedef PMQBOOL MQPOINTER PPMQBOOL;
 typedef MQINT64 MQHMSG;
 typedef MQHMSG MQPOINTER PMQHMSG;
 typedef PMQHMSG MQPOINTER PPMQHMSG;
 typedef MQLONG MQPID;
 typedef MQPID MQPOINTER PMQPID;
 typedef MQLONG MQTID;
 typedef MQTID MQPOINTER PMQTID;


 /****************************************************************/
 /* MQAIR Structure -- Authentication Information Record         */
 /****************************************************************/


 typedef struct tagMQAIR MQAIR;
 typedef MQAIR MQPOINTER PMQAIR;

 struct tagMQAIR {
   MQCHAR4    StrucId;                /* Structure identifier */
   MQLONG     Version;                /* Structure version number */
   MQLONG     AuthInfoType;           /* Type of authentication */
                                      /* information */
   MQCHAR     AuthInfoConnName[264];  /* Connection name of CRL LDAP */
                                      /* server */
   PMQCHAR    LDAPUserNamePtr;        /* Address of LDAP user name */
   MQLONG     LDAPUserNameOffset;     /* Offset of LDAP user name */
                                      /* from start of MQAIR */
                                      /* structure */
   MQLONG     LDAPUserNameLength;     /* Length of LDAP user name */
   MQCHAR32   LDAPPassword;           /* Password to access LDAP */
                                      /* server */
   /* Ver:1 */
   MQCHAR256  OCSPResponderURL;       /* URL of the OCSP responder */
   /* Ver:2 */
 };

 #define MQAIR_DEFAULT {MQAIR_STRUC_ID_ARRAY},\
                       MQAIR_VERSION_1,\
                       MQAIT_CRL_LDAP,\
                       {""},\
                       NULL,\
                       0,\
                       0,\
                       {""},\
                       {""}

 /****************************************************************/
 /* MQBMHO Structure -- Buffer To Message Handle Options         */
 /****************************************************************/


 typedef struct tagMQBMHO MQBMHO;
 typedef MQBMHO  MQPOINTER PMQBMHO;
 typedef PMQBMHO MQPOINTER PPMQBMHO;

 struct tagMQBMHO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQBUFMH */
 };

 #define MQBMHO_DEFAULT {MQBMHO_STRUC_ID_ARRAY},\
                        MQBMHO_VERSION_1,\
                        MQBMHO_DELETE_PROPERTIES

 /****************************************************************/
 /* MQBO Structure -- Begin Options                              */
 /****************************************************************/


 typedef struct tagMQBO MQBO;
 typedef MQBO  MQPOINTER PMQBO;
 typedef PMQBO MQPOINTER PPMQBO;

 struct tagMQBO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQBEGIN */
 };

 #define MQBO_DEFAULT {MQBO_STRUC_ID_ARRAY},\
                      MQBO_VERSION_1,\
                      MQBO_NONE

 /****************************************************************/
 /* MQCBC Structure -- Callback Context                          */
 /****************************************************************/


 typedef struct tagMQCBC MQCBC;
 typedef MQCBC  MQPOINTER PMQCBC;
 typedef PMQCBC MQPOINTER PPMQCBC;

 struct tagMQCBC {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   CallType;        /* Why Function was called */
   MQHOBJ   Hobj;            /* Object Handle */
   MQPTR    CallbackArea;    /* Callback data passed to the function */
   MQPTR    ConnectionArea;  /* MQCTL Data area passed to the */
                             /* function */
   MQLONG   CompCode;        /* Completion Code */
   MQLONG   Reason;          /* Reason Code */
   MQLONG   State;           /* Consumer State */
   MQLONG   DataLength;      /* Message Data Length */
   MQLONG   BufferLength;    /* Buffer Length */
   MQLONG   Flags;           /* Flags containing information about */
                             /* this consumer */
   /* Ver:1 */
   MQLONG   ReconnectDelay;  /* Number of milliseconds before */
                             /* reconnect attempt */
   /* Ver:2 */
 };

 /****************************************************************/
 /* MQCBD Structure -- Callback Data Descriptor                  */
 /****************************************************************/


 typedef struct tagMQCBD MQCBD;
 typedef MQCBD  MQPOINTER PMQCBD;
 typedef PMQCBD MQPOINTER PPMQCBD;

 struct tagMQCBD {
   MQCHAR4    StrucId;           /* Structure identifier */
   MQLONG     Version;           /* Structure version number */
   MQLONG     CallbackType;      /* Callback function type */
   MQLONG     Options;           /* Options controlling message */
                                 /* consumption */
   MQPTR      CallbackArea;      /* User data passed to the function */
   MQPTR      CallbackFunction;  /* Callback function pointer */
   MQCHAR128  CallbackName;      /* Callback name */
   MQLONG     MaxMsgLength;      /* Maximum message length */
 };

 #define MQCBD_DEFAULT {MQCBD_STRUC_ID_ARRAY},\
                       MQCBD_VERSION_1,\
                       MQCBT_MESSAGE_CONSUMER,\
                       MQCBDO_NONE,\
                       NULL,\
                       NULL,\
                       {"\0"},\
                       MQCBD_FULL_MSG_LENGTH

 /****************************************************************/
 /* MQCHARV Structure -- Variable-length string                  */
 /****************************************************************/


 typedef struct tagMQCHARV MQCHARV;
 typedef MQCHARV MQPOINTER PMQCHARV;

 struct tagMQCHARV {
   MQPTR   VSPtr;      /* Address of variable length string */
   MQLONG  VSOffset;   /* Offset of variable length string */
   MQLONG  VSBufSize;  /* Size of buffer */
   MQLONG  VSLength;   /* Length of variable length string */
   MQLONG  VSCCSID;    /* CCSID of variable length string */
 };

 #define MQCHARV_DEFAULT NULL,\
                         0,\
                         0,\
                         0,\
                         MQCCSI_APPL

 /****************************************************************/
 /* MQCIH Structure -- CICS Information Header                   */
 /****************************************************************/


 typedef struct tagMQCIH MQCIH;
 typedef MQCIH MQPOINTER PMQCIH;

 struct tagMQCIH {
   MQCHAR4  StrucId;             /* Structure identifier */
   MQLONG   Version;             /* Structure version number */
   MQLONG   StrucLength;         /* Length of MQCIH structure */
   MQLONG   Encoding;            /* Reserved */
   MQLONG   CodedCharSetId;      /* Reserved */
   MQCHAR8  Format;              /* MQ format name of data that */
                                 /* follows MQCIH */
   MQLONG   Flags;               /* Flags */
   MQLONG   ReturnCode;          /* Return code from bridge */
   MQLONG   CompCode;            /* MQ completion code or CICS */
                                 /* EIBRESP */
   MQLONG   Reason;              /* MQ reason or feedback code, or */
                                 /* CICS EIBRESP2 */
   MQLONG   UOWControl;          /* Unit-of-work control */
   MQLONG   GetWaitInterval;     /* Wait interval for MQGET call */
                                 /* issued by bridge task */
   MQLONG   LinkType;            /* Link type */
   MQLONG   OutputDataLength;    /* Output COMMAREA data length */
   MQLONG   FacilityKeepTime;    /* Bridge facility release time */
   MQLONG   ADSDescriptor;       /* Send/receive ADS descriptor */
   MQLONG   ConversationalTask;  /* Whether task can be */
                                 /* conversational */
   MQLONG   TaskEndStatus;       /* Status at end of task */
   MQBYTE8  Facility;            /* Bridge facility token */
   MQCHAR4  Function;            /* MQ call name or CICS EIBFN */
                                 /* function */
   MQCHAR4  AbendCode;           /* Abend code */
   MQCHAR8  Authenticator;       /* Password or passticket */
   MQCHAR8  Reserved1;           /* Reserved */
   MQCHAR8  ReplyToFormat;       /* MQ format name of reply message */
   MQCHAR4  RemoteSysId;         /* Remote CICS system id to use */
   MQCHAR4  RemoteTransId;       /* CICS RTRANSID to use */
   MQCHAR4  TransactionId;       /* Transaction to attach */
   MQCHAR4  FacilityLike;        /* Terminal emulated attributes */
   MQCHAR4  AttentionId;         /* AID key */
   MQCHAR4  StartCode;           /* Transaction start code */
   MQCHAR4  CancelCode;          /* Abend transaction code */
   MQCHAR4  NextTransactionId;   /* Next transaction to attach */
   MQCHAR8  Reserved2;           /* Reserved */
   MQCHAR8  Reserved3;           /* Reserved */
   /* Ver:1 */
   MQLONG   CursorPosition;      /* Cursor position */
   MQLONG   ErrorOffset;         /* Offset of error in message */
   MQLONG   InputItem;           /* Reserved */
   MQLONG   Reserved4;           /* Reserved */
   /* Ver:2 */
 };

 #define MQCIH_DEFAULT {MQCIH_STRUC_ID_ARRAY},\
                       MQCIH_VERSION_2,\
                       MQCIH_LENGTH_2,\
                       0,\
                       0,\
                       {MQFMT_NONE_ARRAY},\
                       MQCIH_NONE,\
                       MQCRC_OK,\
                       MQCC_OK,\
                       MQRC_NONE,\
                       MQCUOWC_ONLY,\
                       MQCGWI_DEFAULT,\
                       MQCLT_PROGRAM,\
                       MQCODL_AS_INPUT,\
                       0,\
                       MQCADSD_NONE,\
                       MQCCT_NO,\
                       MQCTES_NOSYNC,\
                       {MQCFAC_NONE_ARRAY},\
                       {MQCFUNC_NONE_ARRAY},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {MQFMT_NONE_ARRAY},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' '},\
                       {MQCSC_NONE_ARRAY},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       0,\
                       0,\
                       0,\
                       0

 /****************************************************************/
 /* MQCMHO Structure -- Create Message Handle Options            */
 /****************************************************************/


 typedef struct tagMQCMHO MQCMHO;
 typedef MQCMHO  MQPOINTER PMQCMHO;
 typedef PMQCMHO MQPOINTER PPMQCMHO;

 struct tagMQCMHO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQCRTMH */
 };

 #define MQCMHO_DEFAULT {MQCMHO_STRUC_ID_ARRAY},\
                        MQCMHO_VERSION_1,\
                        MQCMHO_DEFAULT_VALIDATION

 /****************************************************************/
 /* MQCTLO Structure -- MQCTL function options                   */
 /****************************************************************/


 typedef struct tagMQCTLO MQCTLO;
 typedef MQCTLO  MQPOINTER PMQCTLO;
 typedef PMQCTLO MQPOINTER PPMQCTLO;

 struct tagMQCTLO {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   Options;         /* Options that control the action of */
                             /* MQCTL */
   MQLONG   Reserved;        /* Reserved */
   MQPTR    ConnectionArea;  /* MQCTL Data area passed to the */
                             /* function */
 };

 #define MQCTLO_DEFAULT {MQCTLO_STRUC_ID_ARRAY},\
                        MQCTLO_VERSION_1,\
                        MQCTLO_NONE,\
                        MQWI_UNLIMITED,\
                        NULL

 /****************************************************************/
 /* MQSCO Structure -- SSL Configuration Options                 */
 /****************************************************************/


 typedef struct tagMQSCO MQSCO;
 typedef MQSCO MQPOINTER PMQSCO;

 struct tagMQSCO {
   MQCHAR4    StrucId;                    /* Structure identifier */
   MQLONG     Version;                    /* Structure version number */
   MQCHAR256  KeyRepository;              /* Location of SSL key */
                                          /* repository */
   MQCHAR256  CryptoHardware;             /* Cryptographic hardware */
                                          /* configuration string */
   MQLONG     AuthInfoRecCount;           /* Number of MQAIR records */
                                          /* present */
   MQLONG     AuthInfoRecOffset;          /* Offset of first MQAIR */
                                          /* record from start of */
                                          /* MQSCO structure */
   PMQAIR     AuthInfoRecPtr;             /* Address of first MQAIR */
                                          /* record */
   /* Ver:1 */
   MQLONG     KeyResetCount;              /* Number of unencrypted */
                                          /* bytes sent/received */
                                          /* before secret key is */
                                          /* reset */
   MQLONG     FipsRequired;               /* Using FIPS-certified */
                                          /* algorithms */
   /* Ver:2 */
   MQLONG     EncryptionPolicySuiteB[4];  /* Use only Suite B */
                                          /* cryptographic algorithms */
   /* Ver:3 */
   MQLONG     CertificateValPolicy;       /* Certificate validation */
                                          /* policy */
   /* Ver:4 */
   MQCHAR64   CertificateLabel;           /* SSL/TLS certificate */
                                          /* label */
   /* Ver:5 */
 };

 #define MQSCO_DEFAULT {MQSCO_STRUC_ID_ARRAY},\
                       MQSCO_VERSION_1,\
                       {""},\
                       {""},\
                       0,\
                       0,\
                       NULL,\
                       MQSCO_RESET_COUNT_DEFAULT,\
                       MQSSL_FIPS_NO,\
                       {MQ_SUITE_B_NONE,\
                        MQ_SUITE_B_NOT_AVAILABLE,\
                        MQ_SUITE_B_NOT_AVAILABLE,\
                        MQ_SUITE_B_NOT_AVAILABLE},\
                       MQ_CERT_VAL_POLICY_DEFAULT,\
                       {""}

 /****************************************************************/
 /* MQCSP Structure -- Security Parameters                       */
 /****************************************************************/


 typedef struct tagMQCSP MQCSP;
 typedef MQCSP MQPOINTER PMQCSP;

 struct tagMQCSP {
   MQCHAR4  StrucId;             /* Structure identifier */
   MQLONG   Version;             /* Structure version number */
   MQLONG   AuthenticationType;  /* Type of authentication */
   MQBYTE4  Reserved1;           /* Reserved */
   MQPTR    CSPUserIdPtr;        /* Address of user ID */
   MQLONG   CSPUserIdOffset;     /* Offset of user ID */
   MQLONG   CSPUserIdLength;     /* Length of user ID */
   MQBYTE8  Reserved2;           /* Reserved */
   MQPTR    CSPPasswordPtr;      /* Address of password */
   MQLONG   CSPPasswordOffset;   /* Offset of password */
   MQLONG   CSPPasswordLength;   /* Length of password */
 };

 #define MQCSP_DEFAULT {MQCSP_STRUC_ID_ARRAY},\
                       MQCSP_VERSION_1,\
                       MQCSP_AUTH_NONE,\
                       {'\0','\0','\0','\0'},\
                       NULL,\
                       0,\
                       0,\
                       {'\0','\0','\0','\0','\0','\0','\0','\0'},\
                       NULL,\
                       0,\
                       0

 /****************************************************************/
 /* MQCNO Structure -- Connect Options                           */
 /****************************************************************/


 typedef struct tagMQCNO MQCNO;
 typedef MQCNO  MQPOINTER PMQCNO;
 typedef PMQCNO MQPOINTER PPMQCNO;

 struct tagMQCNO {
   MQCHAR4    StrucId;              /* Structure identifier */
   MQLONG     Version;              /* Structure version number */
   MQLONG     Options;              /* Options that control the */
                                    /* action of MQCONNX */
   /* Ver:1 */
   MQLONG     ClientConnOffset;     /* Offset of MQCD structure for */
                                    /* client connection */
   MQPTR      ClientConnPtr;        /* Address of MQCD structure for */
                                    /* client connection */
   /* Ver:2 */
   MQBYTE128  ConnTag;              /* Queue-manager connection tag */
   /* Ver:3 */
   PMQSCO     SSLConfigPtr;         /* Address of MQSCO structure for */
                                    /* client connection */
   MQLONG     SSLConfigOffset;      /* Offset of MQSCO structure for */
                                    /* client connection */
   /* Ver:4 */
   MQBYTE24   ConnectionId;         /* Unique Connection Identifier */
   MQLONG     SecurityParmsOffset;  /* Offset of MQCSP structure */
   PMQCSP     SecurityParmsPtr;     /* Address of MQCSP structure */
   /* Ver:5 */
   PMQCHAR    CCDTUrlPtr;           /* Address of CCDT URL string */
   MQLONG     CCDTUrlOffset;        /* Offset of CCDT URL string */
   MQLONG     CCDTUrlLength;        /* Length of CCDT URL */
   MQBYTE8    Reserved;             /* Reserved */
   /* Ver:6 */
 };

 #define MQCNO_DEFAULT {MQCNO_STRUC_ID_ARRAY},\
                       MQCNO_VERSION_1,\
                       MQCNO_NONE,\
                       0,\
                       NULL,\
                       {MQCT_NONE_ARRAY},\
                       NULL,\
                       0,\
                       {MQCONNID_NONE_ARRAY},\
                       0,\
                       NULL,\
                       NULL,\
                       0,\
                       0,\
                       {'\0','\0','\0','\0','\0','\0','\0','\0'}

 /****************************************************************/
 /* MQDH Structure -- Distribution Header                        */
 /****************************************************************/


 typedef struct tagMQDH MQDH;
 typedef MQDH MQPOINTER PMQDH;

 struct tagMQDH {
   MQCHAR4  StrucId;          /* Structure identifier */
   MQLONG   Version;          /* Structure version number */
   MQLONG   StrucLength;      /* Length of MQDH structure plus */
                              /* following MQOR and MQPMR records */
   MQLONG   Encoding;         /* Numeric encoding of data that */
                              /* follows the MQOR and MQPMR records */
   MQLONG   CodedCharSetId;   /* Character set identifier of data */
                              /* that follows the MQOR and MQPMR */
                              /* records */
   MQCHAR8  Format;           /* Format name of data that follows the */
                              /* MQOR and MQPMR records */
   MQLONG   Flags;            /* General flags */
   MQLONG   PutMsgRecFields;  /* Flags indicating which MQPMR fields */
                              /* are present */
   MQLONG   RecsPresent;      /* Number of MQOR records present */
   MQLONG   ObjectRecOffset;  /* Offset of first MQOR record from */
                              /* start of MQDH */
   MQLONG   PutMsgRecOffset;  /* Offset of first MQPMR record from */
                              /* start of MQDH */
 };

 #define MQDH_DEFAULT {MQDH_STRUC_ID_ARRAY},\
                      MQDH_VERSION_1,\
                      0,\
                      0,\
                      MQCCSI_UNDEFINED,\
                      {MQFMT_NONE_ARRAY},\
                      MQDHF_NONE,\
                      MQPMRF_NONE,\
                      0,\
                      0,\
                      0

 /****************************************************************/
 /* MQDLH Structure -- Dead Letter Header                        */
 /****************************************************************/


 typedef struct tagMQDLH MQDLH;
 typedef MQDLH MQPOINTER PMQDLH;

 struct tagMQDLH {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQLONG    Reason;          /* Reason message arrived on */
                              /* dead-letter (undelivered-message) */
                              /* queue */
   MQCHAR48  DestQName;       /* Name of original destination queue */
   MQCHAR48  DestQMgrName;    /* Name of original destination queue */
                              /* manager */
   MQLONG    Encoding;        /* Numeric encoding of data that */
                              /* follows MQDLH */
   MQLONG    CodedCharSetId;  /* Character set identifier of data */
                              /* that follows MQDLH */
   MQCHAR8   Format;          /* Format name of data that follows */
                              /* MQDLH */
   MQLONG    PutApplType;     /* Type of application that put message */
                              /* on dead-letter (undelivered-message) */
                              /* queue */
   MQCHAR28  PutApplName;     /* Name of application that put message */
                              /* on dead-letter (undelivered-message) */
                              /* queue */
   MQCHAR8   PutDate;         /* Date when message was put on */
                              /* dead-letter (undelivered-message) */
                              /* queue */
   MQCHAR8   PutTime;         /* Time when message was put on */
                              /* dead-letter (undelivered-message) */
                              /* queue */
 };

 #define MQDLH_DEFAULT {MQDLH_STRUC_ID_ARRAY},\
                       MQDLH_VERSION_1,\
                       MQRC_NONE,\
                       {""},\
                       {""},\
                       0,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       0,\
                       {""},\
                       {""},\
                       {""}

 /****************************************************************/
 /* MQDMHO Structure -- Delete Message Handle Options            */
 /****************************************************************/


 typedef struct tagMQDMHO MQDMHO;
 typedef MQDMHO  MQPOINTER PMQDMHO;
 typedef PMQDMHO MQPOINTER PPMQDMHO;

 struct tagMQDMHO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQDLTMH */
 };

 #define MQDMHO_DEFAULT {MQDMHO_STRUC_ID_ARRAY},\
                        MQDMHO_VERSION_1,\
                        MQDMHO_NONE

 /****************************************************************/
 /* MQDMPO Structure -- Delete Message Property Options          */
 /****************************************************************/


 typedef struct tagMQDMPO MQDMPO;
 typedef MQDMPO  MQPOINTER PMQDMPO;
 typedef PMQDMPO MQPOINTER PPMQDMPO;

 struct tagMQDMPO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQDLTMP */
 };

 #define MQDMPO_DEFAULT {MQDMPO_STRUC_ID_ARRAY},\
                        MQDMPO_VERSION_1,\
                        MQDMPO_DEL_FIRST

 /****************************************************************/
 /* MQGMO Structure -- Get Message Options                       */
 /****************************************************************/


 typedef struct tagMQGMO MQGMO;
 typedef MQGMO  MQPOINTER PMQGMO;
 typedef PMQGMO MQPOINTER PPMQGMO;

 struct tagMQGMO {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQLONG    Options;         /* Options that control the action of */
                              /* MQGET */
   MQLONG    WaitInterval;    /* Wait interval */
   MQLONG    Signal1;         /* Signal */
   MQLONG    Signal2;         /* Signal identifier */
   MQCHAR48  ResolvedQName;   /* Resolved name of destination queue */
   /* Ver:1 */
   MQLONG    MatchOptions;    /* Options controlling selection */
                              /* criteria used for MQGET */
   MQCHAR    GroupStatus;     /* Flag indicating whether message */
                              /* retrieved is in a group */
   MQCHAR    SegmentStatus;   /* Flag indicating whether message */
                              /* retrieved is a segment of a logical */
                              /* message */
   MQCHAR    Segmentation;    /* Flag indicating whether further */
                              /* segmentation is allowed for the */
                              /* message retrieved */
   MQCHAR    Reserved1;       /* Reserved */
   /* Ver:2 */
   MQBYTE16  MsgToken;        /* Message token */
   MQLONG    ReturnedLength;  /* Length of message data returned */
                              /* (bytes) */
   /* Ver:3 */
   MQLONG    Reserved2;       /* Reserved */
   MQHMSG    MsgHandle;       /* Message handle */
   /* Ver:4 */
 };

 #define MQGMO_DEFAULT {MQGMO_STRUC_ID_ARRAY},\
                       MQGMO_VERSION_1,\
                       (MQGMO_NO_WAIT+MQGMO_PROPERTIES_AS_Q_DEF),\
                       0,\
                       0,\
                       0,\
                       {""},\
                       (MQMO_MATCH_MSG_ID+MQMO_MATCH_CORREL_ID),\
                       MQGS_NOT_IN_GROUP,\
                       MQSS_NOT_A_SEGMENT,\
                       MQSEG_INHIBITED,\
                       ' ',\
                       {MQMTOK_NONE_ARRAY},\
                       MQRL_UNDEFINED,\
                       0,\
                       MQHM_NONE

 /****************************************************************/
 /* MQIIH Structure -- IMS Information Header                    */
 /****************************************************************/


 typedef struct tagMQIIH MQIIH;
 typedef MQIIH MQPOINTER PMQIIH;

 struct tagMQIIH {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQLONG    StrucLength;     /* Length of MQIIH structure */
   MQLONG    Encoding;        /* Reserved */
   MQLONG    CodedCharSetId;  /* Reserved */
   MQCHAR8   Format;          /* MQ format name of data that follows */
                              /* MQIIH */
   MQLONG    Flags;           /* Flags */
   MQCHAR8   LTermOverride;   /* Logical terminal override */
   MQCHAR8   MFSMapName;      /* Message format services map name */
   MQCHAR8   ReplyToFormat;   /* MQ format name of reply message */
   MQCHAR8   Authenticator;   /* RACF password or passticket */
   MQBYTE16  TranInstanceId;  /* Transaction instance identifier */
   MQCHAR    TranState;       /* Transaction state */
   MQCHAR    CommitMode;      /* Commit mode */
   MQCHAR    SecurityScope;   /* Security scope */
   MQCHAR    Reserved;        /* Reserved */
 };

 #define MQIIH_DEFAULT {MQIIH_STRUC_ID_ARRAY},\
                       MQIIH_VERSION_1,\
                       MQIIH_LENGTH_1,\
                       0,\
                       0,\
                       {MQFMT_NONE_ARRAY},\
                       MQIIH_NONE,\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {MQFMT_NONE_ARRAY},\
                       {MQIAUT_NONE_ARRAY},\
                       {MQITII_NONE_ARRAY},\
                       MQITS_NOT_IN_CONVERSATION,\
                       MQICM_COMMIT_THEN_SEND,\
                       MQISS_CHECK,\
                       ' '

 /****************************************************************/
 /* MQIMPO Structure -- Inquire Message Property Options         */
 /****************************************************************/


 typedef struct tagMQIMPO MQIMPO;
 typedef MQIMPO  MQPOINTER PMQIMPO;
 typedef PMQIMPO MQPOINTER PPMQIMPO;

 struct tagMQIMPO {
   MQCHAR4  StrucId;            /* Structure identifier */
   MQLONG   Version;            /* Structure version number */
   MQLONG   Options;            /* Options that control the action of */
                                /* MQINQMP */
   MQLONG   RequestedEncoding;  /* Requested encoding of Value */
   MQLONG   RequestedCCSID;     /* Requested character set identifier */
                                /* of Value */
   MQLONG   ReturnedEncoding;   /* Returned encoding of Value */
   MQLONG   ReturnedCCSID;      /* Returned character set identifier */
                                /* of Value */
   MQLONG   Reserved1;          /* Reserved */
   MQCHARV  ReturnedName;       /* Returned property name */
   MQCHAR8  TypeString;         /* Property data type as a string */
 };

 #define MQIMPO_DEFAULT {MQIMPO_STRUC_ID_ARRAY},\
                        MQIMPO_VERSION_1,\
                        MQIMPO_INQ_FIRST,\
                        MQENC_NATIVE,\
                        MQCCSI_APPL,\
                        MQENC_NATIVE,\
                        0,\
                        0,\
                        {MQCHARV_DEFAULT},\
                        {""}

 /****************************************************************/
 /* MQMD Structure -- Message Descriptor                         */
 /****************************************************************/


 typedef struct tagMQMD MQMD;
 typedef MQMD  MQPOINTER PMQMD;
 typedef PMQMD MQPOINTER PPMQMD;

 struct tagMQMD {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    Report;            /* Options for report messages */
   MQLONG    MsgType;           /* Message type */
   MQLONG    Expiry;            /* Message lifetime */
   MQLONG    Feedback;          /* Feedback or reason code */
   MQLONG    Encoding;          /* Numeric encoding of message data */
   MQLONG    CodedCharSetId;    /* Character set identifier of */
                                /* message data */
   MQCHAR8   Format;            /* Format name of message data */
   MQLONG    Priority;          /* Message priority */
   MQLONG    Persistence;       /* Message persistence */
   MQBYTE24  MsgId;             /* Message identifier */
   MQBYTE24  CorrelId;          /* Correlation identifier */
   MQLONG    BackoutCount;      /* Backout counter */
   MQCHAR48  ReplyToQ;          /* Name of reply queue */
   MQCHAR48  ReplyToQMgr;       /* Name of reply queue manager */
   MQCHAR12  UserIdentifier;    /* User identifier */
   MQBYTE32  AccountingToken;   /* Accounting token */
   MQCHAR32  ApplIdentityData;  /* Application data relating to */
                                /* identity */
   MQLONG    PutApplType;       /* Type of application that put the */
                                /* message */
   MQCHAR28  PutApplName;       /* Name of application that put the */
                                /* message */
   MQCHAR8   PutDate;           /* Date when message was put */
   MQCHAR8   PutTime;           /* Time when message was put */
   MQCHAR4   ApplOriginData;    /* Application data relating to */
                                /* origin */
   /* Ver:1 */
   MQBYTE24  GroupId;           /* Group identifier */
   MQLONG    MsgSeqNumber;      /* Sequence number of logical message */
                                /* within group */
   MQLONG    Offset;            /* Offset of data in physical message */
                                /* from start of logical message */
   MQLONG    MsgFlags;          /* Message flags */
   MQLONG    OriginalLength;    /* Length of original message */
   /* Ver:2 */
 };

 #define MQMD_DEFAULT {MQMD_STRUC_ID_ARRAY},\
                      MQMD_VERSION_1,\
                      MQRO_NONE,\
                      MQMT_DATAGRAM,\
                      MQEI_UNLIMITED,\
                      MQFB_NONE,\
                      MQENC_NATIVE,\
                      MQCCSI_Q_MGR,\
                      {MQFMT_NONE_ARRAY},\
                      MQPRI_PRIORITY_AS_Q_DEF,\
                      MQPER_PERSISTENCE_AS_Q_DEF,\
                      {MQMI_NONE_ARRAY},\
                      {MQCI_NONE_ARRAY},\
                      0,\
                      {""},\
                      {""},\
                      {""},\
                      {MQACT_NONE_ARRAY},\
                      {""},\
                      MQAT_NO_CONTEXT,\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {MQGI_NONE_ARRAY},\
                      1,\
                      0,\
                      MQMF_NONE,\
                      MQOL_UNDEFINED

 /****************************************************************/
 /* MQMDE Structure -- Message Descriptor Extension              */
 /****************************************************************/


 typedef struct tagMQMDE MQMDE;
 typedef MQMDE MQPOINTER PMQMDE;

 struct tagMQMDE {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQLONG    StrucLength;     /* Length of MQMDE structure */
   MQLONG    Encoding;        /* Numeric encoding of data that */
                              /* follows MQMDE */
   MQLONG    CodedCharSetId;  /* Character-set identifier of data */
                              /* that follows MQMDE */
   MQCHAR8   Format;          /* Format name of data that follows */
                              /* MQMDE */
   MQLONG    Flags;           /* General flags */
   MQBYTE24  GroupId;         /* Group identifier */
   MQLONG    MsgSeqNumber;    /* Sequence number of logical message */
                              /* within group */
   MQLONG    Offset;          /* Offset of data in physical message */
                              /* from start of logical message */
   MQLONG    MsgFlags;        /* Message flags */
   MQLONG    OriginalLength;  /* Length of original message */
 };

 #define MQMDE_DEFAULT {MQMDE_STRUC_ID_ARRAY},\
                       MQMDE_VERSION_2,\
                       MQMDE_LENGTH_2,\
                       MQENC_NATIVE,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       MQMDEF_NONE,\
                       {MQGI_NONE_ARRAY},\
                       1,\
                       0,\
                       MQMF_NONE,\
                       MQOL_UNDEFINED

 /****************************************************************/
 /* MQMD1 Structure -- Version-1 Message Descriptor              */
 /****************************************************************/


 typedef struct tagMQMD1 MQMD1;
 typedef MQMD1 MQPOINTER PMQMD1;

 struct tagMQMD1 {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    Report;            /* Options for report messages */
   MQLONG    MsgType;           /* Message type */
   MQLONG    Expiry;            /* Message lifetime */
   MQLONG    Feedback;          /* Feedback or reason code */
   MQLONG    Encoding;          /* Numeric encoding of message data */
   MQLONG    CodedCharSetId;    /* Character set identifier of */
                                /* message data */
   MQCHAR8   Format;            /* Format name of message data */
   MQLONG    Priority;          /* Message priority */
   MQLONG    Persistence;       /* Message persistence */
   MQBYTE24  MsgId;             /* Message identifier */
   MQBYTE24  CorrelId;          /* Correlation identifier */
   MQLONG    BackoutCount;      /* Backout counter */
   MQCHAR48  ReplyToQ;          /* Name of reply queue */
   MQCHAR48  ReplyToQMgr;       /* Name of reply queue manager */
   MQCHAR12  UserIdentifier;    /* User identifier */
   MQBYTE32  AccountingToken;   /* Accounting token */
   MQCHAR32  ApplIdentityData;  /* Application data relating to */
                                /* identity */
   MQLONG    PutApplType;       /* Type of application that put the */
                                /* message */
   MQCHAR28  PutApplName;       /* Name of application that put the */
                                /* message */
   MQCHAR8   PutDate;           /* Date when message was put */
   MQCHAR8   PutTime;           /* Time when message was put */
   MQCHAR4   ApplOriginData;    /* Application data relating to */
                                /* origin */
 };

 #define MQMD1_DEFAULT {MQMD_STRUC_ID_ARRAY},\
                       MQMD_VERSION_1,\
                       MQRO_NONE,\
                       MQMT_DATAGRAM,\
                       MQEI_UNLIMITED,\
                       MQFB_NONE,\
                       MQENC_NATIVE,\
                       MQCCSI_Q_MGR,\
                       {MQFMT_NONE_ARRAY},\
                       MQPRI_PRIORITY_AS_Q_DEF,\
                       MQPER_PERSISTENCE_AS_Q_DEF,\
                       {MQMI_NONE_ARRAY},\
                       {MQCI_NONE_ARRAY},\
                       0,\
                       {""},\
                       {""},\
                       {""},\
                       {MQACT_NONE_ARRAY},\
                       {""},\
                       MQAT_NO_CONTEXT,\
                       {""},\
                       {""},\
                       {""},\
                       {""}

 /****************************************************************/
 /* MQMD2 Structure -- Version-2 Message Descriptor              */
 /****************************************************************/


 typedef struct tagMQMD2 MQMD2;
 typedef MQMD2 MQPOINTER PMQMD2;

 struct tagMQMD2 {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    Report;            /* Options for report messages */
   MQLONG    MsgType;           /* Message type */
   MQLONG    Expiry;            /* Message lifetime */
   MQLONG    Feedback;          /* Feedback or reason code */
   MQLONG    Encoding;          /* Numeric encoding of message data */
   MQLONG    CodedCharSetId;    /* Character set identifier of */
                                /* message data */
   MQCHAR8   Format;            /* Format name of message data */
   MQLONG    Priority;          /* Message priority */
   MQLONG    Persistence;       /* Message persistence */
   MQBYTE24  MsgId;             /* Message identifier */
   MQBYTE24  CorrelId;          /* Correlation identifier */
   MQLONG    BackoutCount;      /* Backout counter */
   MQCHAR48  ReplyToQ;          /* Name of reply queue */
   MQCHAR48  ReplyToQMgr;       /* Name of reply queue manager */
   MQCHAR12  UserIdentifier;    /* User identifier */
   MQBYTE32  AccountingToken;   /* Accounting token */
   MQCHAR32  ApplIdentityData;  /* Application data relating to */
                                /* identity */
   MQLONG    PutApplType;       /* Type of application that put the */
                                /* message */
   MQCHAR28  PutApplName;       /* Name of application that put the */
                                /* message */
   MQCHAR8   PutDate;           /* Date when message was put */
   MQCHAR8   PutTime;           /* Time when message was put */
   MQCHAR4   ApplOriginData;    /* Application data relating to */
                                /* origin */
   /* Ver:1 */
   MQBYTE24  GroupId;           /* Group identifier */
   MQLONG    MsgSeqNumber;      /* Sequence number of logical message */
                                /* within group */
   MQLONG    Offset;            /* Offset of data in physical message */
                                /* from start of logical message */
   MQLONG    MsgFlags;          /* Message flags */
   MQLONG    OriginalLength;    /* Length of original message */
   /* Ver:2 */
 };

 #define MQMD2_DEFAULT {MQMD_STRUC_ID_ARRAY},\
                       MQMD_VERSION_2,\
                       MQRO_NONE,\
                       MQMT_DATAGRAM,\
                       MQEI_UNLIMITED,\
                       MQFB_NONE,\
                       MQENC_NATIVE,\
                       MQCCSI_Q_MGR,\
                       {MQFMT_NONE_ARRAY},\
                       MQPRI_PRIORITY_AS_Q_DEF,\
                       MQPER_PERSISTENCE_AS_Q_DEF,\
                       {MQMI_NONE_ARRAY},\
                       {MQCI_NONE_ARRAY},\
                       0,\
                       {""},\
                       {""},\
                       {""},\
                       {MQACT_NONE_ARRAY},\
                       {""},\
                       MQAT_NO_CONTEXT,\
                       {""},\
                       {""},\
                       {""},\
                       {""},\
                       {MQGI_NONE_ARRAY},\
                       1,\
                       0,\
                       MQMF_NONE,\
                       MQOL_UNDEFINED

 /****************************************************************/
 /* MQMHBO Structure -- Message Handle To Buffer Options         */
 /****************************************************************/


 typedef struct tagMQMHBO MQMHBO;
 typedef MQMHBO  MQPOINTER PMQMHBO;
 typedef PMQMHBO MQPOINTER PPMQMHBO;

 struct tagMQMHBO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQMHBUF */
 };

 #define MQMHBO_DEFAULT {MQMHBO_STRUC_ID_ARRAY},\
                        MQMHBO_VERSION_1,\
                        MQMHBO_PROPERTIES_IN_MQRFH2

 /****************************************************************/
 /* MQOD Structure -- Object descriptor                          */
 /****************************************************************/


 typedef struct tagMQOD MQOD;
 typedef MQOD  MQPOINTER PMQOD;
 typedef PMQOD MQPOINTER PPMQOD;

 struct tagMQOD {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ObjectType;           /* Object type */
   MQCHAR48  ObjectName;           /* Object name */
   MQCHAR48  ObjectQMgrName;       /* Object queue manager name */
   MQCHAR48  DynamicQName;         /* Dynamic queue name */
   MQCHAR12  AlternateUserId;      /* Alternate user identifier */
   /* Ver:1 */
   MQLONG    RecsPresent;          /* Number of object records */
                                   /* present */
   MQLONG    KnownDestCount;       /* Number of local queues opened */
                                   /* successfully */
   MQLONG    UnknownDestCount;     /* Number of remote queues opened */
   MQLONG    InvalidDestCount;     /* Number of queues that failed to */
                                   /* open */
   MQLONG    ObjectRecOffset;      /* Offset of first object record */
                                   /* from start of MQOD */
   MQLONG    ResponseRecOffset;    /* Offset of first response record */
                                   /* from start of MQOD */
   MQPTR     ObjectRecPtr;         /* Address of first object record */
   MQPTR     ResponseRecPtr;       /* Address of first response */
                                   /* record */
   /* Ver:2 */
   MQBYTE40  AlternateSecurityId;  /* Alternate security identifier */
   MQCHAR48  ResolvedQName;        /* Resolved queue name */
   MQCHAR48  ResolvedQMgrName;     /* Resolved queue manager name */
   /* Ver:3 */
   MQCHARV   ObjectString;         /* Object long name */
   MQCHARV   SelectionString;      /* Message Selector */
   MQCHARV   ResObjectString;      /* Resolved long object name */
   MQLONG    ResolvedType;         /* Alias queue resolved object */
                                   /* type */
   /* Ver:4 */
 };

 #define MQOD_DEFAULT {MQOD_STRUC_ID_ARRAY},\
                      MQOD_VERSION_1,\
                      MQOT_Q,\
                      {""},\
                      {""},\
                      {"AMQ.*"},\
                      {""},\
                      0,\
                      0,\
                      0,\
                      0,\
                      0,\
                      0,\
                      NULL,\
                      NULL,\
                      {MQSID_NONE_ARRAY},\
                      {""},\
                      {""},\
                      {MQCHARV_DEFAULT},\
                      {MQCHARV_DEFAULT},\
                      {MQCHARV_DEFAULT},\
                      MQOT_NONE

 /****************************************************************/
 /* MQOR Structure -- Object Record                              */
 /****************************************************************/


 typedef struct tagMQOR MQOR;
 typedef MQOR MQPOINTER PMQOR;

 struct tagMQOR {
   MQCHAR48  ObjectName;      /* Object name */
   MQCHAR48  ObjectQMgrName;  /* Object queue manager name */
 };

 #define MQOR_DEFAULT {""},\
                      {""}

 /****************************************************************/
 /* MQPD Structure -- Property descriptor                        */
 /****************************************************************/


 typedef struct tagMQPD MQPD;
 typedef MQPD MQPOINTER PMQPD;

 struct tagMQPD {
   MQCHAR4  StrucId;      /* Structure identifier */
   MQLONG   Version;      /* Structure version number */
   MQLONG   Options;      /* Options that control the action of */
                          /* MQSETMP and MQINQMP */
   MQLONG   Support;      /* Property support option */
   MQLONG   Context;      /* Property context */
   MQLONG   CopyOptions;  /* Property copy options */
 };

 #define MQPD_DEFAULT {MQPD_STRUC_ID_ARRAY},\
                      MQPD_VERSION_1,\
                      MQPD_NONE,\
                      MQPD_SUPPORT_OPTIONAL,\
                      MQPD_NO_CONTEXT,\
                      MQCOPY_DEFAULT

 /****************************************************************/
 /* MQPMO Structure -- Put Message Options                       */
 /****************************************************************/


 typedef struct tagMQPMO MQPMO;
 typedef MQPMO  MQPOINTER PMQPMO;
 typedef PMQPMO MQPOINTER PPMQPMO;

 struct tagMQPMO {
   MQCHAR4   StrucId;            /* Structure identifier */
   MQLONG    Version;            /* Structure version number */
   MQLONG    Options;            /* Options that control the action */
                                 /* of MQPUT and MQPUT1 */
   MQLONG    Timeout;            /* Reserved */
   MQHOBJ    Context;            /* Object handle of input queue */
   MQLONG    KnownDestCount;     /* Number of messages sent */
                                 /* successfully to local queues */
   MQLONG    UnknownDestCount;   /* Number of messages sent */
                                 /* successfully to remote queues */
   MQLONG    InvalidDestCount;   /* Number of messages that could not */
                                 /* be sent */
   MQCHAR48  ResolvedQName;      /* Resolved name of destination */
                                 /* queue */
   MQCHAR48  ResolvedQMgrName;   /* Resolved name of destination */
                                 /* queue manager */
   /* Ver:1 */
   MQLONG    RecsPresent;        /* Number of put message records or */
                                 /* response records present */
   MQLONG    PutMsgRecFields;    /* Flags indicating which MQPMR */
                                 /* fields are present */
   MQLONG    PutMsgRecOffset;    /* Offset of first put message */
                                 /* record from start of MQPMO */
   MQLONG    ResponseRecOffset;  /* Offset of first response record */
                                 /* from start of MQPMO */
   MQPTR     PutMsgRecPtr;       /* Address of first put message */
                                 /* record */
   MQPTR     ResponseRecPtr;     /* Address of first response record */
   /* Ver:2 */
   MQHMSG    OriginalMsgHandle;  /* Original message handle */
   MQHMSG    NewMsgHandle;       /* New message handle */
   MQLONG    Action;             /* The action being performed */
   MQLONG    PubLevel;           /* Publication level */
   /* Ver:3 */
 };

 #define MQPMO_DEFAULT {MQPMO_STRUC_ID_ARRAY},\
                       MQPMO_VERSION_1,\
                       MQPMO_NONE,\
                       (-1),\
                       0,\
                       0,\
                       0,\
                       0,\
                       {""},\
                       {""},\
                       0,\
                       MQPMRF_NONE,\
                       0,\
                       0,\
                       NULL,\
                       NULL,\
                       MQHM_NONE,\
                       MQHM_NONE,\
                       MQACTP_NEW,\
                       9

 /****************************************************************/
 /* MQRFH Structure -- Rules and Formatting Header               */
 /****************************************************************/


 typedef struct tagMQRFH MQRFH;
 typedef MQRFH MQPOINTER PMQRFH;

 struct tagMQRFH {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   StrucLength;     /* Total length of MQRFH including */
                             /* NameValueString */
   MQLONG   Encoding;        /* Numeric encoding of data that follows */
                             /* NameValueString */
   MQLONG   CodedCharSetId;  /* Character set identifier of data that */
                             /* follows NameValueString */
   MQCHAR8  Format;          /* Format name of data that follows */
                             /* NameValueString */
   MQLONG   Flags;           /* Flags */
 };

 #define MQRFH_DEFAULT {MQRFH_STRUC_ID_ARRAY},\
                       MQRFH_VERSION_1,\
                       MQRFH_STRUC_LENGTH_FIXED,\
                       MQENC_NATIVE,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       MQRFH_NONE

 /****************************************************************/
 /* MQRFH2 Structure -- Rules and Formatting Header 2            */
 /****************************************************************/


 typedef struct tagMQRFH2 MQRFH2;
 typedef MQRFH2 MQPOINTER PMQRFH2;

 struct tagMQRFH2 {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   StrucLength;     /* Total length of MQRFH2 including all */
                             /* NameValueLength and NameValueData */
                             /* fields */
   MQLONG   Encoding;        /* Numeric encoding of data that follows */
                             /* last NameValueData field */
   MQLONG   CodedCharSetId;  /* Character set identifier of data that */
                             /* follows last NameValueData field */
   MQCHAR8  Format;          /* Format name of data that follows last */
                             /* NameValueData field */
   MQLONG   Flags;           /* Flags */
   MQLONG   NameValueCCSID;  /* Character set identifier of */
                             /* NameValueData */
 };

 #define MQRFH2_DEFAULT {MQRFH_STRUC_ID_ARRAY},\
                        MQRFH_VERSION_2,\
                        MQRFH_STRUC_LENGTH_FIXED_2,\
                        MQENC_NATIVE,\
                        MQCCSI_INHERIT,\
                        {MQFMT_NONE_ARRAY},\
                        MQRFH_NONE,\
                        1208

 /****************************************************************/
 /* MQRMH Structure -- Reference Message Header                  */
 /****************************************************************/


 typedef struct tagMQRMH MQRMH;
 typedef MQRMH MQPOINTER PMQRMH;

 struct tagMQRMH {
   MQCHAR4   StrucId;             /* Structure identifier */
   MQLONG    Version;             /* Structure version number */
   MQLONG    StrucLength;         /* Total length of MQRMH, including */
                                  /* strings at end of fixed fields, */
                                  /* but not the bulk data */
   MQLONG    Encoding;            /* Numeric encoding of bulk data */
   MQLONG    CodedCharSetId;      /* Character set identifier of bulk */
                                  /* data */
   MQCHAR8   Format;              /* Format name of bulk data */
   MQLONG    Flags;               /* Reference message flags */
   MQCHAR8   ObjectType;          /* Object type */
   MQBYTE24  ObjectInstanceId;    /* Object instance identifier */
   MQLONG    SrcEnvLength;        /* Length of source environment */
                                  /* data */
   MQLONG    SrcEnvOffset;        /* Offset of source environment */
                                  /* data */
   MQLONG    SrcNameLength;       /* Length of source object name */
   MQLONG    SrcNameOffset;       /* Offset of source object name */
   MQLONG    DestEnvLength;       /* Length of destination */
                                  /* environment data */
   MQLONG    DestEnvOffset;       /* Offset of destination */
                                  /* environment */
   MQLONG    DestNameLength;      /* Length of destination object */
                                  /* name */
   MQLONG    DestNameOffset;      /* Offset of destination object */
                                  /* name */
   MQLONG    DataLogicalLength;   /* Length of bulk data */
   MQLONG    DataLogicalOffset;   /* Low offset of bulk data */
   MQLONG    DataLogicalOffset2;  /* High offset of bulk data */
 };

 #define MQRMH_DEFAULT {MQRMH_STRUC_ID_ARRAY},\
                       MQRMH_VERSION_1,\
                       0,\
                       MQENC_NATIVE,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       MQRMHF_NOT_LAST,\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {MQOII_NONE_ARRAY},\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0

 /****************************************************************/
 /* MQRR Structure -- Response Record                            */
 /****************************************************************/


 typedef struct tagMQRR MQRR;
 typedef MQRR MQPOINTER PMQRR;

 struct tagMQRR {
   MQLONG  CompCode;  /* Completion code for queue */
   MQLONG  Reason;    /* Reason code for queue */
 };

 #define MQRR_DEFAULT MQCC_OK,\
                      MQRC_NONE

 /****************************************************************/
 /* MQSD Structure -- Subscription Descriptor                    */
 /****************************************************************/


 typedef struct tagMQSD MQSD;
 typedef MQSD  MQPOINTER PMQSD;
 typedef PMQSD MQPOINTER PPMQSD;

 struct tagMQSD {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    Options;              /* Options associated with */
                                   /* subscribing */
   MQCHAR48  ObjectName;           /* Object name */
   MQCHAR12  AlternateUserId;      /* Alternate user identifier */
   MQBYTE40  AlternateSecurityId;  /* Alternate security identifier */
   MQLONG    SubExpiry;            /* Expiry of Subscription */
   MQCHARV   ObjectString;         /* Object long name */
   MQCHARV   SubName;              /* Subscription name */
   MQCHARV   SubUserData;          /* Subscription user data */
   MQBYTE24  SubCorrelId;          /* Correlation Id related to this */
                                   /* subscription */
   MQLONG    PubPriority;          /* Priority set in publications */
   MQBYTE32  PubAccountingToken;   /* Accounting Token set in */
                                   /* publications */
   MQCHAR32  PubApplIdentityData;  /* Appl Identity Data set in */
                                   /* publications */
   MQCHARV   SelectionString;      /* Message selector structure */
   MQLONG    SubLevel;             /* Subscription level */
   MQCHARV   ResObjectString;      /* Resolved long object name */
 };

 #define MQSD_DEFAULT {MQSD_STRUC_ID_ARRAY},\
                      MQSD_VERSION_1,\
                      0,\
                      {""},\
                      {""},\
                      {MQSID_NONE_ARRAY},\
                      MQEI_UNLIMITED,\
                      {MQCHARV_DEFAULT},\
                      {MQCHARV_DEFAULT},\
                      {MQCHARV_DEFAULT},\
                      {MQCI_NONE_ARRAY},\
                      MQPRI_PRIORITY_AS_PUBLISHED,\
                      {MQACT_NONE_ARRAY},\
                      {""},\
                      {MQCHARV_DEFAULT},\
                      1,\
                      {MQCHARV_DEFAULT}

 /****************************************************************/
 /* MQSMPO Structure -- Set Message Property Options             */
 /****************************************************************/


 typedef struct tagMQSMPO MQSMPO;
 typedef MQSMPO  MQPOINTER PMQSMPO;
 typedef PMQSMPO MQPOINTER PPMQSMPO;

 struct tagMQSMPO {
   MQCHAR4  StrucId;        /* Structure identifier */
   MQLONG   Version;        /* Structure version number */
   MQLONG   Options;        /* Options that control the action of */
                            /* MQSETMP */
   MQLONG   ValueEncoding;  /* Encoding of Value */
   MQLONG   ValueCCSID;     /* Character set identifier of Value */
 };

 #define MQSMPO_DEFAULT {MQSMPO_STRUC_ID_ARRAY},\
                        MQSMPO_VERSION_1,\
                        MQSMPO_SET_FIRST,\
                        MQENC_NATIVE,\
                        MQCCSI_APPL

 /****************************************************************/
 /* MQSRO Structure -- Subscription Request Options              */
 /****************************************************************/


 typedef struct tagMQSRO MQSRO;
 typedef MQSRO  MQPOINTER PMQSRO;
 typedef PMQSRO MQPOINTER PPMQSRO;

 struct tagMQSRO {
   MQCHAR4  StrucId;  /* Structure identifier */
   MQLONG   Version;  /* Structure version number */
   MQLONG   Options;  /* Options that control the action of MQSUBRQ */
   MQLONG   NumPubs;  /* Number of publications sent */
 };

 #define MQSRO_DEFAULT {MQSRO_STRUC_ID_ARRAY},\
                       MQSRO_VERSION_1,\
                       0,\
                       0

 /****************************************************************/
 /* MQSTS Structure -- Status Information Record                 */
 /****************************************************************/


 typedef struct tagMQSTS MQSTS;
 typedef MQSTS  MQPOINTER PMQSTS;
 typedef PMQSTS MQPOINTER PPMQSTS;

 struct tagMQSTS {
   MQCHAR4   StrucId;             /* Structure identifier */
   MQLONG    Version;             /* Structure version number */
   MQLONG    CompCode;            /* Completion Code of first error */
   MQLONG    Reason;              /* Reason Code of first error */
   MQLONG    PutSuccessCount;     /* Number of Async put calls */
                                  /* succeeded */
   MQLONG    PutWarningCount;     /* Number of Async put calls had */
                                  /* warnings */
   MQLONG    PutFailureCount;     /* Number of Async put calls had */
                                  /* failures */
   MQLONG    ObjectType;          /* Failing object type */
   MQCHAR48  ObjectName;          /* Failing object name */
   MQCHAR48  ObjectQMgrName;      /* Failing object queue manager */
   MQCHAR48  ResolvedObjectName;  /* Resolved name of destination */
                                  /* queue */
   MQCHAR48  ResolvedQMgrName;    /* Resolved name of destination */
                                  /* qmgr */
   /* Ver:1 */
   MQCHARV   ObjectString;        /* Failing object long name */
   MQCHARV   SubName;             /* Failing subscription name */
   MQLONG    OpenOptions;         /* Failing open options */
   MQLONG    SubOptions;          /* Failing subscription options */
   /* Ver:2 */
 };

 #define MQSTS_DEFAULT {MQSTS_STRUC_ID_ARRAY},\
                       MQSTS_VERSION_1,\
                       0,\
                       0,\
                       0,\
                       MQCC_OK,\
                       MQRC_NONE,\
                       MQOT_Q,\
                       {""},\
                       {""},\
                       {""},\
                       {""},\
                       {MQCHARV_DEFAULT},\
                       {MQCHARV_DEFAULT},\
                       0,\
                       0

 /****************************************************************/
 /* MQTM Structure -- Trigger Message                            */
 /****************************************************************/


 typedef struct tagMQTM MQTM;
 typedef MQTM MQPOINTER PMQTM;

 struct tagMQTM {
   MQCHAR4    StrucId;      /* Structure identifier */
   MQLONG     Version;      /* Structure version number */
   MQCHAR48   QName;        /* Name of triggered queue */
   MQCHAR48   ProcessName;  /* Name of process object */
   MQCHAR64   TriggerData;  /* Trigger data */
   MQLONG     ApplType;     /* Application type */
   MQCHAR256  ApplId;       /* Application identifier */
   MQCHAR128  EnvData;      /* Environment data */
   MQCHAR128  UserData;     /* User data */
 };

 #define MQTM_DEFAULT {MQTM_STRUC_ID_ARRAY},\
                      MQTM_VERSION_1,\
                      {""},\
                      {""},\
                      {""},\
                      0,\
                      {""},\
                      {""},\
                      {""}

 /****************************************************************/
 /* MQTMC2 Structure -- Trigger Message 2 (Character)            */
 /****************************************************************/


 typedef struct tagMQTMC2 MQTMC2;
 typedef MQTMC2 MQPOINTER PMQTMC2;

 struct tagMQTMC2 {
   MQCHAR4    StrucId;      /* Structure identifier */
   MQCHAR4    Version;      /* Structure version number */
   MQCHAR48   QName;        /* Name of triggered queue */
   MQCHAR48   ProcessName;  /* Name of process object */
   MQCHAR64   TriggerData;  /* Trigger data */
   MQCHAR4    ApplType;     /* Application type */
   MQCHAR256  ApplId;       /* Application identifier */
   MQCHAR128  EnvData;      /* Environment data */
   MQCHAR128  UserData;     /* User data */
   /* Ver:1 */
   MQCHAR48   QMgrName;     /* Queue manager name */
   /* Ver:2 */
 };

 #define MQTMC2_DEFAULT {MQTMC_STRUC_ID_ARRAY},\
                        {MQTMC_VERSION_2_ARRAY},\
                        {""},\
                        {""},\
                        {""},\
                        {""},\
                        {""},\
                        {""},\
                        {""},\
                        {""}

 /****************************************************************/
 /* MQWIH Structure -- Work Information Header                   */
 /****************************************************************/


 typedef struct tagMQWIH MQWIH;
 typedef MQWIH MQPOINTER PMQWIH;

 struct tagMQWIH {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQLONG    StrucLength;     /* Length of MQWIH structure */
   MQLONG    Encoding;        /* Numeric encoding of data that */
                              /* follows MQWIH */
   MQLONG    CodedCharSetId;  /* Character-set identifier of data */
                              /* that follows MQWIH */
   MQCHAR8   Format;          /* Format name of data that follows */
                              /* MQWIH */
   MQLONG    Flags;           /* Flags */
   MQCHAR32  ServiceName;     /* Service name */
   MQCHAR8   ServiceStep;     /* Service step name */
   MQBYTE16  MsgToken;        /* Message token */
   MQCHAR32  Reserved;        /* Reserved */
 };

 #define MQWIH_DEFAULT {MQWIH_STRUC_ID_ARRAY},\
                       MQWIH_VERSION_1,\
                       MQWIH_LENGTH_1,\
                       0,\
                       MQCCSI_UNDEFINED,\
                       {MQFMT_NONE_ARRAY},\
                       MQWIH_NONE,\
                       {' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' '},\
                       {' ',' ',' ',' ',' ',' ',' ',' '},\
                       {MQMTOK_NONE_ARRAY},\
                       {' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' ',\
                       ' ',' ',' ',' ',' ',' ',' ',' '}

 /****************************************************************/
 /* MQXQH Structure -- Transmission Queue Header                 */
 /****************************************************************/


 typedef struct tagMQXQH MQXQH;
 typedef MQXQH MQPOINTER PMQXQH;

 struct tagMQXQH {
   MQCHAR4   StrucId;         /* Structure identifier */
   MQLONG    Version;         /* Structure version number */
   MQCHAR48  RemoteQName;     /* Name of destination queue */
   MQCHAR48  RemoteQMgrName;  /* Name of destination queue manager */
   MQMD1     MsgDesc;         /* Original message descriptor */
 };

 #define MQXQH_DEFAULT {MQXQH_STRUC_ID_ARRAY},\
                       MQXQH_VERSION_1,\
                       {""},\
                       {""},\
                       {MQMD1_DEFAULT}

 /****************************************************************/
 /*  Parameter usage in functions and structures                 */
 /*    I:    input                                               */
 /*    IB:   input, data buffer                                  */
 /*    IL:   input, length of data buffer                        */
 /*    IO:   input and output                                    */
 /*    IOB:  input and output, data buffer                       */
 /*    IOL:  input and output, length of data buffer             */
 /*    O:    output                                              */
 /*    OB:   output, data buffer                                 */
 /*    OC:   output, completion code                             */
 /*    OR:   output, reason code                                 */
 /*    FP:   function pointer                                    */
 /****************************************************************/

 /****************************************************************/
 /* MQBACK Function -- Back Out Changes                          */
 /****************************************************************/

 void MQENTRY MQBACK (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_BACK_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_BACK_CALL MQPOINTER PMQ_BACK_CALL;


 /****************************************************************/
 /* MQBEGIN Function -- Begin Unit of Work                       */
 /****************************************************************/

 void MQENTRY MQBEGIN (
   MQHCONN  Hconn,          /* I: Connection handle */
   PMQVOID  pBeginOptions,  /* IO: Options that control the action of */
                            /* MQBEGIN */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_BEGIN_CALL (
   MQHCONN  Hconn,          /* I: Connection handle */
   PMQVOID  pBeginOptions,  /* IO: Options that control the action of */
                            /* MQBEGIN */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_BEGIN_CALL MQPOINTER PMQ_BEGIN_CALL;


 /****************************************************************/
 /* MQBUFMH Function -- Buffer To Message Handle                 */
 /****************************************************************/

 void MQENTRY MQBUFMH (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pBufMsgHOpts,  /* I: Options that control the action of */
                           /* MQBUFMH */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* IOB: Area to contain the message buffer */
   PMQLONG  pDataLength,   /* O: Length of the output buffer */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_BUFMH_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pBufMsgHOpts,  /* I: Options that control the action of */
                           /* MQBUFMH */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* IOB: Area to contain the message buffer */
   PMQLONG  pDataLength,   /* O: Length of the output buffer */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_BUFMH_CALL MQPOINTER PMQ_BUFMH_CALL;


 /****************************************************************/
 /* MQCB Function -- Register Message consumer                   */
 /****************************************************************/

 void MQENTRY MQCB (
   MQHCONN  Hconn,          /* I: Connection handle */
   MQLONG   Operation,      /* I: Operation */
   PMQVOID  pCallbackDesc,  /* I: Callback descriptor */
   MQHOBJ   Hobj,           /* I: Object handle */
   PMQVOID  pMsgDesc,       /* I: Message Descriptor */
   PMQVOID  pGetMsgOpts,    /* I: Get options */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CB_CALL (
   MQHCONN  Hconn,          /* I: Connection handle */
   MQLONG   Operation,      /* I: Operation */
   PMQVOID  pCallbackDesc,  /* I: Callback descriptor */
   MQHOBJ   Hobj,           /* I: Object handle */
   PMQVOID  pMsgDesc,       /* I: Message Descriptor */
   PMQVOID  pGetMsgOpts,    /* I: Get options */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_CB_CALL MQPOINTER PMQ_CB_CALL;


 /****************************************************************/
 /* MQCLOSE Function -- Close Object                             */
 /****************************************************************/

 void MQENTRY MQCLOSE (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQHOBJ  pHobj,      /* IO: Object handle */
   MQLONG   Options,    /* I: Options that control the action of */
                        /* MQCLOSE */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CLOSE_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQHOBJ  pHobj,      /* IO: Object handle */
   MQLONG   Options,    /* I: Options that control the action of */
                        /* MQCLOSE */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_CLOSE_CALL MQPOINTER PMQ_CLOSE_CALL;


 /****************************************************************/
 /* MQCMIT Function -- Commit Changes                            */
 /****************************************************************/

 void MQENTRY MQCMIT (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CMIT_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_CMIT_CALL MQPOINTER PMQ_CMIT_CALL;


 /****************************************************************/
 /* MQCONN Function -- Connect Queue Manager                     */
 /****************************************************************/

 void MQENTRY MQCONN (
   PMQCHAR   pQMgrName,  /* I: Name of queue manager */
   PMQHCONN  pHconn,     /* O: Connection handle */
   PMQLONG   pCompCode,  /* OC: Completion code */
   PMQLONG   pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CONN_CALL (
   PMQCHAR   pQMgrName,  /* I: Name of queue manager */
   PMQHCONN  pHconn,     /* O: Connection handle */
   PMQLONG   pCompCode,  /* OC: Completion code */
   PMQLONG   pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_CONN_CALL MQPOINTER PMQ_CONN_CALL;


 /****************************************************************/
 /* MQCONNX Function -- Connect Queue Manager (Extended)         */
 /****************************************************************/

 void MQENTRY MQCONNX (
   PMQCHAR   pQMgrName,     /* I: Name of queue manager */
   PMQCNO    pConnectOpts,  /* IO: Options that control the action of */
                            /* MQCONNX */
   PMQHCONN  pHconn,        /* O: Connection handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CONNX_CALL (
   PMQCHAR   pQMgrName,     /* I: Name of queue manager */
   PMQCNO    pConnectOpts,  /* IO: Options that control the action of */
                            /* MQCONNX */
   PMQHCONN  pHconn,        /* O: Connection handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CONNX_CALL MQPOINTER PMQ_CONNX_CALL;


 /****************************************************************/
 /* MQCRTMH Function -- Create Message Handle                    */
 /****************************************************************/

 void MQENTRY MQCRTMH (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQVOID  pCrtMsgHOpts,  /* I: Options that control the action of */
                           /* MQCRTMH */
   PMQHMSG  pHmsg,         /* O: Message handle */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CRTMH_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQVOID  pCrtMsgHOpts,  /* I: Options that control the action of */
                           /* MQCRTMH */
   PMQHMSG  pHmsg,         /* O: Message handle */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CRTMH_CALL MQPOINTER PMQ_CRTMH_CALL;


 /****************************************************************/
 /* MQCTL Function -- Control Consumer                           */
 /****************************************************************/

 void MQENTRY MQCTL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQLONG   Operation,     /* I: Operation */
   PMQVOID  pControlOpts,  /* I: Control options */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_CTL_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQLONG   Operation,     /* I: Operation */
   PMQVOID  pControlOpts,  /* I: Control options */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CTL_CALL MQPOINTER PMQ_CTL_CALL;


 /****************************************************************/
 /* MQDISC Function -- Disconnect Queue Manager                  */
 /****************************************************************/

 void MQENTRY MQDISC (
   PMQHCONN  pHconn,     /* IO: Connection handle */
   PMQLONG   pCompCode,  /* OC: Completion code */
   PMQLONG   pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_DISC_CALL (
   PMQHCONN  pHconn,     /* IO: Connection handle */
   PMQLONG   pCompCode,  /* OC: Completion code */
   PMQLONG   pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_DISC_CALL MQPOINTER PMQ_DISC_CALL;


 /****************************************************************/
 /* MQDLTMH Function -- Delete Message Handle                    */
 /****************************************************************/

 void MQENTRY MQDLTMH (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQHMSG  pHmsg,         /* IO: Message handle */
   PMQVOID  pDltMsgHOpts,  /* I: Options that control the action of */
                           /* MQDLTMH */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_DLTMH_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQHMSG  pHmsg,         /* IO: Message handle */
   PMQVOID  pDltMsgHOpts,  /* I: Options that control the action of */
                           /* MQDLTMH */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_DLTMH_CALL MQPOINTER PMQ_DLTMH_CALL;


 /****************************************************************/
 /* MQDLTMP Function -- Delete Message Property                  */
 /****************************************************************/

 void MQENTRY MQDLTMP (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pDltPropOpts,  /* I: Options that control the action of */
                           /* MQDLTMP */
   PMQVOID  pName,         /* I: Property name */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_DLTMP_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pDltPropOpts,  /* I: Options that control the action of */
                           /* MQDLTMP */
   PMQVOID  pName,         /* I: Property name */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_DLTMP_CALL MQPOINTER PMQ_DLTMP_CALL;


 /****************************************************************/
 /* MQGET Function -- Get Message                                */
 /****************************************************************/

 void MQENTRY MQGET (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHOBJ   Hobj,          /* I: Object handle */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pGetMsgOpts,   /* IO: Options that control the action of */
                           /* MQGET */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* OB: Area to contain the message data */
   PMQLONG  pDataLength,   /* O: Length of the message */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_GET_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHOBJ   Hobj,          /* I: Object handle */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pGetMsgOpts,   /* IO: Options that control the action of */
                           /* MQGET */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* OB: Area to contain the message data */
   PMQLONG  pDataLength,   /* O: Length of the message */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_GET_CALL MQPOINTER PMQ_GET_CALL;


 /****************************************************************/
 /* MQINQ Function -- Inquire Object Attributes                  */
 /****************************************************************/

 void MQENTRY MQINQ (
   MQHCONN  Hconn,           /* I: Connection handle */
   MQHOBJ   Hobj,            /* I: Object handle */
   MQLONG   SelectorCount,   /* I: Count of selectors */
   PMQLONG  pSelectors,      /* I: Array of attribute selectors */
   MQLONG   IntAttrCount,    /* I: Count of integer attributes */
   PMQLONG  pIntAttrs,       /* O: Array of integer attributes */
   MQLONG   CharAttrLength,  /* IL: Length of character attributes */
                             /* buffer */
   PMQCHAR  pCharAttrs,      /* OB: Character attributes */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_INQ_CALL (
   MQHCONN  Hconn,           /* I: Connection handle */
   MQHOBJ   Hobj,            /* I: Object handle */
   MQLONG   SelectorCount,   /* I: Count of selectors */
   PMQLONG  pSelectors,      /* I: Array of attribute selectors */
   MQLONG   IntAttrCount,    /* I: Count of integer attributes */
   PMQLONG  pIntAttrs,       /* O: Array of integer attributes */
   MQLONG   CharAttrLength,  /* IL: Length of character attributes */
                             /* buffer */
   PMQCHAR  pCharAttrs,      /* OB: Character attributes */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQ_INQ_CALL MQPOINTER PMQ_INQ_CALL;


 /****************************************************************/
 /* MQINQMP Function -- Inquire Message Property                 */
 /****************************************************************/

 void MQENTRY MQINQMP (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pInqPropOpts,  /* I: Options that control the action of */
                           /* MQINQMP */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pPropDesc,     /* O: Property descriptor */
   PMQLONG  pType,         /* IO: Property data type */
   MQLONG   ValueLength,   /* IL: Length in bytes of the Value area */
   PMQVOID  pValue,        /* OB: Property value */
   PMQLONG  pDataLength,   /* O: Length of the property value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_INQMP_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pInqPropOpts,  /* I: Options that control the action of */
                           /* MQINQMP */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pPropDesc,     /* O: Property descriptor */
   PMQLONG  pType,         /* IO: Property data type */
   MQLONG   ValueLength,   /* IL: Length in bytes of the Value area */
   PMQVOID  pValue,        /* OB: Property value */
   PMQLONG  pDataLength,   /* O: Length of the property value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_INQMP_CALL MQPOINTER PMQ_INQMP_CALL;


 /****************************************************************/
 /* MQMHBUF Function -- Message Handle To Buffer                 */
 /****************************************************************/

 void MQENTRY MQMHBUF (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pMsgHBufOpts,  /* I: Options that control the action of */
                           /* MQMHBUF */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* OB: Area to contain the properties */
   PMQLONG  pDataLength,   /* O: Length of the properties */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_MHBUF_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pMsgHBufOpts,  /* I: Options that control the action of */
                           /* MQMHBUF */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   MQLONG   BufferLength,  /* IL: Length in bytes of the Buffer area */
   PMQVOID  pBuffer,       /* OB: Area to contain the properties */
   PMQLONG  pDataLength,   /* O: Length of the properties */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_MHBUF_CALL MQPOINTER PMQ_MHBUF_CALL;


 /****************************************************************/
 /* MQOPEN Function -- Open Object                               */
 /****************************************************************/

 void MQENTRY MQOPEN (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQVOID  pObjDesc,   /* IO: Object descriptor */
   MQLONG   Options,    /* I: Options that control the action of */
                        /* MQOPEN */
   PMQHOBJ  pHobj,      /* O: Object handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_OPEN_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQVOID  pObjDesc,   /* IO: Object descriptor */
   MQLONG   Options,    /* I: Options that control the action of */
                        /* MQOPEN */
   PMQHOBJ  pHobj,      /* O: Object handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_OPEN_CALL MQPOINTER PMQ_OPEN_CALL;


 /****************************************************************/
 /* MQPUT Function -- Put Message                                */
 /****************************************************************/

 void MQENTRY MQPUT (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHOBJ   Hobj,          /* I: Object handle */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pPutMsgOpts,   /* IO: Options that control the action of */
                           /* MQPUT */
   MQLONG   BufferLength,  /* IL: Length of the message in Buffer */
   PMQVOID  pBuffer,       /* IB: Message data */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_PUT_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHOBJ   Hobj,          /* I: Object handle */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pPutMsgOpts,   /* IO: Options that control the action of */
                           /* MQPUT */
   MQLONG   BufferLength,  /* IL: Length of the message in Buffer */
   PMQVOID  pBuffer,       /* IB: Message data */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_PUT_CALL MQPOINTER PMQ_PUT_CALL;


 /****************************************************************/
 /* MQPUT1 Function -- Put One Message                           */
 /****************************************************************/

 void MQENTRY MQPUT1 (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQVOID  pObjDesc,      /* IO: Object descriptor */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pPutMsgOpts,   /* IO: Options that control the action of */
                           /* MQPUT1 */
   MQLONG   BufferLength,  /* IL: Length of the message in Buffer */
   PMQVOID  pBuffer,       /* IB: Message data */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_PUT1_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   PMQVOID  pObjDesc,      /* IO: Object descriptor */
   PMQVOID  pMsgDesc,      /* IO: Message descriptor */
   PMQVOID  pPutMsgOpts,   /* IO: Options that control the action of */
                           /* MQPUT1 */
   MQLONG   BufferLength,  /* IL: Length of the message in Buffer */
   PMQVOID  pBuffer,       /* IB: Message data */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_PUT1_CALL MQPOINTER PMQ_PUT1_CALL;


 /****************************************************************/
 /* MQSET Function -- Set Object Attributes                      */
 /****************************************************************/

 void MQENTRY MQSET (
   MQHCONN  Hconn,           /* I: Connection handle */
   MQHOBJ   Hobj,            /* I: Object handle */
   MQLONG   SelectorCount,   /* I: Count of selectors */
   PMQLONG  pSelectors,      /* I: Array of attribute selectors */
   MQLONG   IntAttrCount,    /* I: Count of integer attributes */
   PMQLONG  pIntAttrs,       /* I: Array of integer attributes */
   MQLONG   CharAttrLength,  /* IL: Length of character attributes */
                             /* buffer */
   PMQCHAR  pCharAttrs,      /* IB: Character attributes */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_SET_CALL (
   MQHCONN  Hconn,           /* I: Connection handle */
   MQHOBJ   Hobj,            /* I: Object handle */
   MQLONG   SelectorCount,   /* I: Count of selectors */
   PMQLONG  pSelectors,      /* I: Array of attribute selectors */
   MQLONG   IntAttrCount,    /* I: Count of integer attributes */
   PMQLONG  pIntAttrs,       /* I: Array of integer attributes */
   MQLONG   CharAttrLength,  /* IL: Length of character attributes */
                             /* buffer */
   PMQCHAR  pCharAttrs,      /* IB: Character attributes */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQ_SET_CALL MQPOINTER PMQ_SET_CALL;


 /****************************************************************/
 /* MQSETMP Function -- Set Message Property                     */
 /****************************************************************/

 void MQENTRY MQSETMP (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pSetPropOpts,  /* I: Options that control the action of */
                           /* MQSETMP */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pPropDesc,     /* IO: Property descriptor */
   MQLONG   Type,          /* I: Property data type */
   MQLONG   ValueLength,   /* IL: Length of the Value area */
   PMQVOID  pValue,        /* IB: Property value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_SETMP_CALL (
   MQHCONN  Hconn,         /* I: Connection handle */
   MQHMSG   Hmsg,          /* I: Message handle */
   PMQVOID  pSetPropOpts,  /* I: Options that control the action of */
                           /* MQSETMP */
   PMQVOID  pName,         /* I: Property name */
   PMQVOID  pPropDesc,     /* IO: Property descriptor */
   MQLONG   Type,          /* I: Property data type */
   MQLONG   ValueLength,   /* IL: Length of the Value area */
   PMQVOID  pValue,        /* IB: Property value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_SETMP_CALL MQPOINTER PMQ_SETMP_CALL;


 /****************************************************************/
 /* MQSTAT Function -- Get Status Information                    */
 /****************************************************************/

 void MQENTRY MQSTAT (
   MQHCONN  Hconn,      /* I: Connection handle */
   MQLONG   Type,       /* I: Status information type */
   PMQVOID  pStatus,    /* IO: Status information */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_STAT_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   MQLONG   Type,       /* I: Status information type */
   PMQVOID  pStatus,    /* IO: Status information */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_STAT_CALL MQPOINTER PMQ_STAT_CALL;


 /****************************************************************/
 /* MQSUB Function -- Subscribe to topic                         */
 /****************************************************************/

 void MQENTRY MQSUB (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQVOID  pSubDesc,   /* IO: Subscription descriptor */
   PMQHOBJ  pHobj,      /* IO: Object handle for queue */
   PMQHOBJ  pHsub,      /* O: Subscription object handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_SUB_CALL (
   MQHCONN  Hconn,      /* I: Connection handle */
   PMQVOID  pSubDesc,   /* IO: Subscription descriptor */
   PMQHOBJ  pHobj,      /* IO: Object handle for queue */
   PMQHOBJ  pHsub,      /* O: Subscription object handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */
 typedef MQ_SUB_CALL MQPOINTER PMQ_SUB_CALL;


 /****************************************************************/
 /* MQSUBRQ Function -- Subscription Request                     */
 /****************************************************************/

 void MQENTRY MQSUBRQ (
   MQHCONN  Hconn,       /* I: Connection handle */
   MQHOBJ   Hsub,        /* I: Subscription handle */
   MQLONG   Action,      /* I: Action requested on the subscription */
   PMQVOID  pSubRqOpts,  /* IO: Subscription Request Options */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_SUBRQ_CALL (
   MQHCONN  Hconn,       /* I: Connection handle */
   MQHOBJ   Hsub,        /* I: Subscription handle */
   MQLONG   Action,      /* I: Action requested on the subscription */
   PMQVOID  pSubRqOpts,  /* IO: Subscription Request Options */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */
 typedef MQ_SUBRQ_CALL MQPOINTER PMQ_SUBRQ_CALL;


 /****************************************************************/
 /* MQCB_FUNCTION - Message Consumer routine (Called by MQ)      */
 /****************************************************************/

 typedef void MQENTRY MQCB_FUNCTION (
   MQHCONN  Hconn,        /* I: Connection handle */
   PMQVOID  pMsgDesc,     /* I: Message descriptor */
   PMQVOID  pGetMsgOpts,  /* I: Area containing the MQGMO */
   PMQVOID  pBuffer,      /* I: Area containing the message data */
   PMQCBC   pContext);    /* I: Area containing the Consumer context */
 typedef MQCB_FUNCTION MQPOINTER PMQCB_FUNCTION;



 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQC                                                 */
 /****************************************************************/
 #endif  /* End of header file */
