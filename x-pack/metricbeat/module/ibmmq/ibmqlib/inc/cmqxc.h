 #if !defined(MQXC_INCLUDED)           /* File not yet included? */
   #define MQXC_INCLUDED               /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQXC                                       */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for Exits and MQCD             */
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
 /*                  structures and named constants for exits    */
 /*                  and MQCD.                                   */
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
 /* pn=com.ibm.mq.famfiles.data/xml/approved/cmqxc.xml           */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Values Related to MQCD Structure                             */
 /****************************************************************/

 /* Structure Version Number */
 #define MQCD_VERSION_1                 1
 #define MQCD_VERSION_2                 2
 #define MQCD_VERSION_3                 3
 #define MQCD_VERSION_4                 4
 #define MQCD_VERSION_5                 5
 #define MQCD_VERSION_6                 6
 #define MQCD_VERSION_7                 7
 #define MQCD_VERSION_8                 8
 #define MQCD_VERSION_9                 9
 #define MQCD_VERSION_10                10
 #define MQCD_VERSION_11                11
 #define MQCD_CURRENT_VERSION           11

 /* Structure Length */
 #define MQCD_LENGTH_1                  984
 #define MQCD_LENGTH_2                  1312
 #define MQCD_LENGTH_3                  1480
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_4                  1568
#else
 #define MQCD_LENGTH_4                  1540
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_5                  1584
#else
 #define MQCD_LENGTH_5                  1552
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_6                  1688
#else
 #define MQCD_LENGTH_6                  1648
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_7                  1792
#else
 #define MQCD_LENGTH_7                  1748
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_8                  1888
#else
 #define MQCD_LENGTH_8                  1840
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_9                  1912
#else
 #define MQCD_LENGTH_9                  1864
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_10                 1920
#else
 #define MQCD_LENGTH_10                 1876
#endif
#if defined(MQ_64_BIT)
 #define MQCD_LENGTH_11                 1984
#else
 #define MQCD_LENGTH_11                 1940
#endif
#if defined(MQ_64_BIT)
 #define MQCD_CURRENT_LENGTH            1984
#else
 #define MQCD_CURRENT_LENGTH            1940
#endif

 /* Channel Types */
 #define MQCHT_SENDER                   1
 #define MQCHT_SERVER                   2
 #define MQCHT_RECEIVER                 3
 #define MQCHT_REQUESTER                4
 #define MQCHT_ALL                      5
 #define MQCHT_CLNTCONN                 6
 #define MQCHT_SVRCONN                  7
 #define MQCHT_CLUSRCVR                 8
 #define MQCHT_CLUSSDR                  9
 #define MQCHT_MQTT                     10
 #define MQCHT_AMQP                     11

 /* Channel Compression */
 #define MQCOMPRESS_NOT_AVAILABLE       (-1)
 #define MQCOMPRESS_NONE                0
 #define MQCOMPRESS_RLE                 1
 #define MQCOMPRESS_ZLIBFAST            2
 #define MQCOMPRESS_ZLIBHIGH            4
 #define MQCOMPRESS_SYSTEM              8
 #define MQCOMPRESS_ANY                 0x0FFFFFFF

 /* Transport Types */
 #define MQXPT_ALL                      (-1)
 #define MQXPT_LOCAL                    0
 #define MQXPT_LU62                     1
 #define MQXPT_TCP                      2
 #define MQXPT_NETBIOS                  3
 #define MQXPT_SPX                      4
 #define MQXPT_DECNET                   5
 #define MQXPT_UDP                      6

 /* Put Authority */
 #define MQPA_DEFAULT                   1
 #define MQPA_CONTEXT                   2
 #define MQPA_ONLY_MCA                  3
 #define MQPA_ALTERNATE_OR_MCA          4

 /* Channel Data Conversion */
 #define MQCDC_SENDER_CONVERSION        1
 #define MQCDC_NO_SENDER_CONVERSION     0

 /* MCA Types */
 #define MQMCAT_PROCESS                 1
 #define MQMCAT_THREAD                  2

 /* NonPersistent-Message Speeds */
 #define MQNPMS_NORMAL                  1
 #define MQNPMS_FAST                    2

 /* SSL Client Authentication */
 #define MQSCA_REQUIRED                 0
 #define MQSCA_OPTIONAL                 1
 #define MQSCA_NEVER_REQUIRED           2

 /* KeepAlive Interval */
 #define MQKAI_AUTO                     (-1)

 /* Connection Affinity Values */
 #define MQCAFTY_NONE                   0
 #define MQCAFTY_PREFERRED              1

 /* Client Reconnect */
 #define MQRCN_NO                       0
 #define MQRCN_YES                      1
 #define MQRCN_Q_MGR                    2
 #define MQRCN_DISABLED                 3

 /* Protocol */
 #define MQPROTO_MQTTV3                 1
 #define MQPROTO_HTTP                   2
 #define MQPROTO_AMQP                   3
 #define MQPROTO_MQTTV311               4

 /* Security Protocol */
 #define MQSECPROT_NONE                 0
 #define MQSECPROT_SSLV30               1
 #define MQSECPROT_TLSV10               2
 #define MQSECPROT_TLSV12               4

 /****************************************************************/
 /* Values Related to MQACH Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQACH_STRUC_ID                 "ACH "

 /* Structure Identifier (array form) */
 #define MQACH_STRUC_ID_ARRAY           'A','C','H',' '

 /* Structure Version Number */
 #define MQACH_VERSION_1                1
 #define MQACH_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQACH_LENGTH_1                 72
#else
 #define MQACH_LENGTH_1                 68
#endif
#if defined(MQ_64_BIT)
 #define MQACH_CURRENT_LENGTH           72
#else
 #define MQACH_CURRENT_LENGTH           68
#endif

 /****************************************************************/
 /* Values Related to MQAXC Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQAXC_STRUC_ID                 "AXC "

 /* Structure Identifier (array form) */
 #define MQAXC_STRUC_ID_ARRAY           'A','X','C',' '

 /* Structure Version Number */
 #define MQAXC_VERSION_1                1
 #define MQAXC_VERSION_2                2
 #define MQAXC_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQAXC_LENGTH_1                 392
#else
 #define MQAXC_LENGTH_1                 384
#endif
#if defined(MQ_64_BIT)
 #define MQAXC_LENGTH_2                 424
#else
 #define MQAXC_LENGTH_2                 412
#endif
#if defined(MQ_64_BIT)
 #define MQAXC_CURRENT_LENGTH           424
#else
 #define MQAXC_CURRENT_LENGTH           412
#endif

 /* Environments */
 #define MQXE_OTHER                     0
 #define MQXE_MCA                       1
 #define MQXE_MCA_SVRCONN               2
 #define MQXE_COMMAND_SERVER            3
 #define MQXE_MQSC                      4
 #define MQXE_MCA_CLNTCONN              5

 /****************************************************************/
 /* Values Related to MQAXP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQAXP_STRUC_ID                 "AXP "

 /* Structure Identifier (array form) */
 #define MQAXP_STRUC_ID_ARRAY           'A','X','P',' '

 /* Structure Version Number */
 #define MQAXP_VERSION_1                1
 #define MQAXP_VERSION_2                2
 #define MQAXP_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQAXP_LENGTH_1                 256
#else
 #define MQAXP_LENGTH_1                 244
#endif
#if defined(MQ_64_BIT)
 #define MQAXP_CURRENT_LENGTH           256
#else
 #define MQAXP_CURRENT_LENGTH           244
#endif

 /* API Caller Types */
 #define MQXACT_EXTERNAL                1
 #define MQXACT_INTERNAL                2

 /* Problem Determination Area */
 #define MQXPDA_NONE                    "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"\
                                        "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Problem Determination Area (array form) */
 #define MQXPDA_NONE_ARRAY              '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* API Function Identifiers */
 #define MQXF_INIT                      1
 #define MQXF_TERM                      2
 #define MQXF_CONN                      3
 #define MQXF_CONNX                     4
 #define MQXF_DISC                      5
 #define MQXF_OPEN                      6
 #define MQXF_CLOSE                     7
 #define MQXF_PUT1                      8
 #define MQXF_PUT                       9
 #define MQXF_GET                       10
 #define MQXF_DATA_CONV_ON_GET          11
 #define MQXF_INQ                       12
 #define MQXF_SET                       13
 #define MQXF_BEGIN                     14
 #define MQXF_CMIT                      15
 #define MQXF_BACK                      16
 #define MQXF_STAT                      18
 #define MQXF_CB                        19
 #define MQXF_CTL                       20
 #define MQXF_CALLBACK                  21
 #define MQXF_SUB                       22
 #define MQXF_SUBRQ                     23
 #define MQXF_XACLOSE                   24
 #define MQXF_XACOMMIT                  25
 #define MQXF_XACOMPLETE                26
 #define MQXF_XAEND                     27
 #define MQXF_XAFORGET                  28
 #define MQXF_XAOPEN                    29
 #define MQXF_XAPREPARE                 30
 #define MQXF_XARECOVER                 31
 #define MQXF_XAROLLBACK                32
 #define MQXF_XASTART                   33
 #define MQXF_AXREG                     34
 #define MQXF_AXUNREG                   35

 /****************************************************************/
 /* Values Related to MQCXP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQCXP_STRUC_ID                 "CXP "

 /* Structure Identifier (array form) */
 #define MQCXP_STRUC_ID_ARRAY           'C','X','P',' '

 /* Structure Version Number */
 #define MQCXP_VERSION_1                1
 #define MQCXP_VERSION_2                2
 #define MQCXP_VERSION_3                3
 #define MQCXP_VERSION_4                4
 #define MQCXP_VERSION_5                5
 #define MQCXP_VERSION_6                6
 #define MQCXP_VERSION_7                7
 #define MQCXP_VERSION_8                8
 #define MQCXP_VERSION_9                9
 #define MQCXP_CURRENT_VERSION          9

 /* Structure Length */
 #define MQCXP_LENGTH_3                 156
 #define MQCXP_LENGTH_4                 156
 #define MQCXP_LENGTH_5                 160
#if defined(MQ_64_BIT)
 #define MQCXP_LENGTH_6                 200
#else
 #define MQCXP_LENGTH_6                 192
#endif
#if defined(MQ_64_BIT)
 #define MQCXP_LENGTH_7                 208
#else
 #define MQCXP_LENGTH_7                 200
#endif
#if defined(MQ_64_BIT)
 #define MQCXP_LENGTH_8                 224
#else
 #define MQCXP_LENGTH_8                 208
#endif
#if defined(MQ_64_BIT)
 #define MQCXP_LENGTH_9                 240
#else
 #define MQCXP_LENGTH_9                 220
#endif
#if defined(MQ_64_BIT)
 #define MQCXP_CURRENT_LENGTH           240
#else
 #define MQCXP_CURRENT_LENGTH           220
#endif

 /* Exit Response 2 */
 #define MQXR2_PUT_WITH_DEF_ACTION      0
 #define MQXR2_PUT_WITH_DEF_USERID      1
 #define MQXR2_PUT_WITH_MSG_USERID      2
 #define MQXR2_USE_AGENT_BUFFER         0
 #define MQXR2_USE_EXIT_BUFFER          4
 #define MQXR2_DEFAULT_CONTINUATION     0
 #define MQXR2_CONTINUE_CHAIN           8
 #define MQXR2_SUPPRESS_CHAIN           16
 #define MQXR2_STATIC_CACHE             0
 #define MQXR2_DYNAMIC_CACHE            32

 /* Capability Flags */
 #define MQCF_NONE                      0x00000000
 #define MQCF_DIST_LISTS                0x00000001

 /****************************************************************/
 /* Values Related to MQDXP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQDXP_STRUC_ID                 "DXP "

 /* Structure Identifier (array form) */
 #define MQDXP_STRUC_ID_ARRAY           'D','X','P',' '

 /* Structure Version Number */
 #define MQDXP_VERSION_1                1
 #define MQDXP_VERSION_2                2
 #define MQDXP_CURRENT_VERSION          2

 /* Structure Length */
 #define MQDXP_LENGTH_1                 44
#if defined(MQ_64_BIT)
 #define MQDXP_LENGTH_2                 56
#else
 #define MQDXP_LENGTH_2                 48
#endif
#if defined(MQ_64_BIT)
 #define MQDXP_CURRENT_LENGTH           56
#else
 #define MQDXP_CURRENT_LENGTH           48
#endif

 /* Exit Response */
 #define MQXDR_OK                       0
 #define MQXDR_CONVERSION_FAILED        1

 /****************************************************************/
 /* Values Related to MQNXP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQNXP_STRUC_ID                 "NXP "

 /* Structure Identifier (array form) */
 #define MQNXP_STRUC_ID_ARRAY           'N','X','P',' '

 /* Structure Version Number */
 #define MQNXP_VERSION_1                1
 #define MQNXP_VERSION_2                2
 #define MQNXP_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQNXP_LENGTH_1                 64
#else
 #define MQNXP_LENGTH_1                 52
#endif
#if defined(MQ_64_BIT)
 #define MQNXP_LENGTH_2                 72
#else
 #define MQNXP_LENGTH_2                 56
#endif
#if defined(MQ_64_BIT)
 #define MQNXP_CURRENT_LENGTH           72
#else
 #define MQNXP_CURRENT_LENGTH           56
#endif

 /****************************************************************/
 /* Values Related to MQPBC Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQPBC_STRUC_ID                 "PBC "

 /* Structure Identifier (array form) */
 #define MQPBC_STRUC_ID_ARRAY           'P','B','C',' '

 /* Structure Version Number */
 #define MQPBC_VERSION_1                1
 #define MQPBC_VERSION_2                2
 #define MQPBC_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQPBC_LENGTH_1                 32
#else
 #define MQPBC_LENGTH_1                 28
#endif
#if defined(MQ_64_BIT)
 #define MQPBC_LENGTH_2                 40
#else
 #define MQPBC_LENGTH_2                 32
#endif
#if defined(MQ_64_BIT)
 #define MQPBC_CURRENT_LENGTH           40
#else
 #define MQPBC_CURRENT_LENGTH           32
#endif

 /****************************************************************/
 /* Values Related to MQPSXP Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQPSXP_STRUC_ID                "PSXP"

 /* Structure Identifier (array form) */
 #define MQPSXP_STRUC_ID_ARRAY          'P','S','X','P'

 /* Structure Version Number */
 #define MQPSXP_VERSION_1               1
 #define MQPSXP_VERSION_2               2
 #define MQPSXP_CURRENT_VERSION         2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQPSXP_LENGTH_1                176
#else
 #define MQPSXP_LENGTH_1                156
#endif
#if defined(MQ_64_BIT)
 #define MQPSXP_LENGTH_2                184
#else
 #define MQPSXP_LENGTH_2                160
#endif
#if defined(MQ_64_BIT)
 #define MQPSXP_CURRENT_LENGTH          184
#else
 #define MQPSXP_CURRENT_LENGTH          160
#endif

 /****************************************************************/
 /* Values Related to MQSBC Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQSBC_STRUC_ID                 "SBC "

 /* Structure Identifier (array form) */
 #define MQSBC_STRUC_ID_ARRAY           'S','B','C',' '

 /* Structure Version Number */
 #define MQSBC_VERSION_1                1
 #define MQSBC_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQSBC_LENGTH_1                 288
#else
 #define MQSBC_LENGTH_1                 272
#endif
#if defined(MQ_64_BIT)
 #define MQSBC_CURRENT_LENGTH           288
#else
 #define MQSBC_CURRENT_LENGTH           272
#endif

 /****************************************************************/
 /* Values Related to MQWDR Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQWDR_STRUC_ID                 "WDR "

 /* Structure Identifier (array form) */
 #define MQWDR_STRUC_ID_ARRAY           'W','D','R',' '

 /* Structure Version Number */
 #define MQWDR_VERSION_1                1
 #define MQWDR_VERSION_2                2
 #define MQWDR_CURRENT_VERSION          2

 /* Structure Length */
 #define MQWDR_LENGTH_1                 124
 #define MQWDR_LENGTH_2                 136
 #define MQWDR_CURRENT_LENGTH           136

 /* Queue Manager Flags */
 #define MQQMF_REPOSITORY_Q_MGR         0x00000002
 #define MQQMF_CLUSSDR_USER_DEFINED     0x00000008
 #define MQQMF_CLUSSDR_AUTO_DEFINED     0x00000010
 #define MQQMF_AVAILABLE                0x00000020

 /****************************************************************/
 /* Values Related to MQWDR Structure                            */
 /****************************************************************/

 /* Structure Length */
 #define MQWDR1_LENGTH_1                124
 #define MQWDR1_CURRENT_LENGTH          124

 /****************************************************************/
 /* Values Related to MQWDR2 Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQWDR2_LENGTH_1                124
 #define MQWDR2_LENGTH_2                136
 #define MQWDR2_CURRENT_LENGTH          136

 /****************************************************************/
 /* Values Related to MQWQR Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQWQR_STRUC_ID                 "WQR "

 /* Structure Identifier (array form) */
 #define MQWQR_STRUC_ID_ARRAY           'W','Q','R',' '

 /* Structure Version Number */
 #define MQWQR_VERSION_1                1
 #define MQWQR_VERSION_2                2
 #define MQWQR_VERSION_3                3
 #define MQWQR_CURRENT_VERSION          3

 /* Structure Length */
 #define MQWQR_LENGTH_1                 200
 #define MQWQR_LENGTH_2                 208
 #define MQWQR_LENGTH_3                 212
 #define MQWQR_CURRENT_LENGTH           212

 /* Queue Flags */
 #define MQQF_LOCAL_Q                   0x00000001
 #define MQQF_CLWL_USEQ_ANY             0x00000040
 #define MQQF_CLWL_USEQ_LOCAL           0x00000080

 /****************************************************************/
 /* Values Related to MQWQR1 Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQWQR1_LENGTH_1                200
 #define MQWQR1_CURRENT_LENGTH          200

 /****************************************************************/
 /* Values Related to MQWQR2 Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQWQR2_LENGTH_1                200
 #define MQWQR2_LENGTH_2                208
 #define MQWQR2_CURRENT_LENGTH          208

 /****************************************************************/
 /* Values Related to MQWQR3 Structure                           */
 /****************************************************************/

 /* Structure Length */
 #define MQWQR3_LENGTH_1                200
 #define MQWQR3_LENGTH_2                208
 #define MQWQR3_LENGTH_3                212
 #define MQWQR3_CURRENT_LENGTH          212

 /****************************************************************/
 /* Values Related to MQWXP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQWXP_STRUC_ID                 "WXP "

 /* Structure Identifier (array form) */
 #define MQWXP_STRUC_ID_ARRAY           'W','X','P',' '

 /* Structure Version Number */
 #define MQWXP_VERSION_1                1
 #define MQWXP_VERSION_2                2
 #define MQWXP_VERSION_3                3
 #define MQWXP_VERSION_4                4
 #define MQWXP_CURRENT_VERSION          4

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQWXP_LENGTH_1                 224
#else
 #define MQWXP_LENGTH_1                 208
#endif
#if defined(MQ_64_BIT)
 #define MQWXP_LENGTH_2                 240
#else
 #define MQWXP_LENGTH_2                 216
#endif
#if defined(MQ_64_BIT)
 #define MQWXP_LENGTH_3                 240
#else
 #define MQWXP_LENGTH_3                 220
#endif
#if defined(MQ_64_BIT)
 #define MQWXP_LENGTH_4                 248
#else
 #define MQWXP_LENGTH_4                 224
#endif
#if defined(MQ_64_BIT)
 #define MQWXP_CURRENT_LENGTH           248
#else
 #define MQWXP_CURRENT_LENGTH           224
#endif

 /* Cluster Workload Flags */
 #define MQWXP_PUT_BY_CLUSTER_CHL       0x00000002

 /****************************************************************/
 /* Values Related to MQWXP1 Structure                           */
 /****************************************************************/

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQWXP1_LENGTH_1                224
#else
 #define MQWXP1_LENGTH_1                208
#endif
#if defined(MQ_64_BIT)
 #define MQWXP1_CURRENT_LENGTH          224
#else
 #define MQWXP1_CURRENT_LENGTH          208
#endif

 /****************************************************************/
 /* Values Related to MQWXP2 Structure                           */
 /****************************************************************/

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQWXP2_LENGTH_1                224
#else
 #define MQWXP2_LENGTH_1                208
#endif
#if defined(MQ_64_BIT)
 #define MQWXP2_LENGTH_2                240
#else
 #define MQWXP2_LENGTH_2                216
#endif
#if defined(MQ_64_BIT)
 #define MQWXP2_CURRENT_LENGTH          240
#else
 #define MQWXP2_CURRENT_LENGTH          216
#endif

 /****************************************************************/
 /* Values Related to MQWXP3 Structure                           */
 /****************************************************************/

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQWXP3_LENGTH_1                224
#else
 #define MQWXP3_LENGTH_1                208
#endif
#if defined(MQ_64_BIT)
 #define MQWXP3_LENGTH_2                240
#else
 #define MQWXP3_LENGTH_2                216
#endif
#if defined(MQ_64_BIT)
 #define MQWXP3_LENGTH_3                240
#else
 #define MQWXP3_LENGTH_3                220
#endif
#if defined(MQ_64_BIT)
 #define MQWXP3_CURRENT_LENGTH          240
#else
 #define MQWXP3_CURRENT_LENGTH          220
#endif

 /****************************************************************/
 /* Values Related to MQWXP4 Structure                           */
 /****************************************************************/

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQWXP4_LENGTH_1                224
#else
 #define MQWXP4_LENGTH_1                208
#endif
#if defined(MQ_64_BIT)
 #define MQWXP4_LENGTH_2                240
#else
 #define MQWXP4_LENGTH_2                216
#endif
#if defined(MQ_64_BIT)
 #define MQWXP4_LENGTH_3                240
#else
 #define MQWXP4_LENGTH_3                220
#endif
#if defined(MQ_64_BIT)
 #define MQWXP4_LENGTH_4                248
#else
 #define MQWXP4_LENGTH_4                224
#endif
#if defined(MQ_64_BIT)
 #define MQWXP4_CURRENT_LENGTH          248
#else
 #define MQWXP4_CURRENT_LENGTH          224
#endif

 /****************************************************************/
 /* Values Related to MQXEPO Structure                           */
 /****************************************************************/

 /* Structure Identifier */
 #define MQXEPO_STRUC_ID                "XEPO"

 /* Structure Identifier (array form) */
 #define MQXEPO_STRUC_ID_ARRAY          'X','E','P','O'

 /* Structure Version Number */
 #define MQXEPO_VERSION_1               1
 #define MQXEPO_CURRENT_VERSION         1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQXEPO_LENGTH_1                40
#else
 #define MQXEPO_LENGTH_1                32
#endif
#if defined(MQ_64_BIT)
 #define MQXEPO_CURRENT_LENGTH          40
#else
 #define MQXEPO_CURRENT_LENGTH          32
#endif

 /* Exit Options */
 #define MQXEPO_NONE                    0x00000000

 /****************************************************************/
 /* General Values Related to Exits                              */
 /****************************************************************/

 /* Exit Identifiers */
 #define MQXT_API_CROSSING_EXIT         1
 #define MQXT_API_EXIT                  2
 #define MQXT_CHANNEL_SEC_EXIT          11
 #define MQXT_CHANNEL_MSG_EXIT          12
 #define MQXT_CHANNEL_SEND_EXIT         13
 #define MQXT_CHANNEL_RCV_EXIT          14
 #define MQXT_CHANNEL_MSG_RETRY_EXIT    15
 #define MQXT_CHANNEL_AUTO_DEF_EXIT     16
 #define MQXT_CLUSTER_WORKLOAD_EXIT     20
 #define MQXT_PUBSUB_ROUTING_EXIT       21
 #define MQXT_PUBLISH_EXIT              22
 #define MQXT_PRECONNECT_EXIT           23

 /* Exit Reasons */
 #define MQXR_BEFORE                    1
 #define MQXR_AFTER                     2
 #define MQXR_CONNECTION                3
 #define MQXR_BEFORE_CONVERT            4
 #define MQXR_INIT                      11
 #define MQXR_TERM                      12
 #define MQXR_MSG                       13
 #define MQXR_XMIT                      14
 #define MQXR_SEC_MSG                   15
 #define MQXR_INIT_SEC                  16
 #define MQXR_RETRY                     17
 #define MQXR_AUTO_CLUSSDR              18
 #define MQXR_AUTO_RECEIVER             19
 #define MQXR_CLWL_OPEN                 20
 #define MQXR_CLWL_PUT                  21
 #define MQXR_CLWL_MOVE                 22
 #define MQXR_CLWL_REPOS                23
 #define MQXR_CLWL_REPOS_MOVE           24
 #define MQXR_END_BATCH                 25
 #define MQXR_ACK_RECEIVED              26
 #define MQXR_AUTO_SVRCONN              27
 #define MQXR_AUTO_CLUSRCVR             28
 #define MQXR_SEC_PARMS                 29
 #define MQXR_PUBLICATION               30
 #define MQXR_PRECONNECT                31

 /* Exit Responses */
 #define MQXCC_OK                       0
 #define MQXCC_SUPPRESS_FUNCTION        (-1)
 #define MQXCC_SKIP_FUNCTION            (-2)
 #define MQXCC_SEND_AND_REQUEST_SEC_MSG (-3)
 #define MQXCC_SEND_SEC_MSG             (-4)
 #define MQXCC_SUPPRESS_EXIT            (-5)
 #define MQXCC_CLOSE_CHANNEL            (-6)
 #define MQXCC_REQUEST_ACK              (-7)
 #define MQXCC_FAILED                   (-8)

 /* Exit User Area Value */
 #define MQXUA_NONE                     "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"

 /* Exit User Area Value (array form) */
 #define MQXUA_NONE_ARRAY               '\0','\0','\0','\0','\0','\0','\0','\0',\
                                        '\0','\0','\0','\0','\0','\0','\0','\0'

 /* Cluster Cache Types */
 #define MQCLCT_STATIC                  0
 #define MQCLCT_DYNAMIC                 1

 /* Multicast Events */
 #define MQMCEV_PACKET_LOSS             1
 #define MQMCEV_HEARTBEAT_TIMEOUT       2
 #define MQMCEV_VERSION_CONFLICT        3
 #define MQMCEV_RELIABILITY             4
 #define MQMCEV_CLOSED_TRANS            5
 #define MQMCEV_STREAM_ERROR            6
 #define MQMCEV_NEW_SOURCE              10
 #define MQMCEV_RECEIVE_QUEUE_TRIMMED   11
 #define MQMCEV_PACKET_LOSS_NACK_EXPIRE 12
 #define MQMCEV_ACK_RETRIES_EXCEEDED    13
 #define MQMCEV_STREAM_SUSPEND_NACK     14
 #define MQMCEV_STREAM_RESUME_NACK      15
 #define MQMCEV_STREAM_EXPELLED         16
 #define MQMCEV_FIRST_MESSAGE           20
 #define MQMCEV_LATE_JOIN_FAILURE       21
 #define MQMCEV_MESSAGE_LOSS            22
 #define MQMCEV_SEND_PACKET_FAILURE     23
 #define MQMCEV_REPAIR_DELAY            24
 #define MQMCEV_MEMORY_ALERT_ON         25
 #define MQMCEV_MEMORY_ALERT_OFF        26
 #define MQMCEV_NACK_ALERT_ON           27
 #define MQMCEV_NACK_ALERT_OFF          28
 #define MQMCEV_REPAIR_ALERT_ON         29
 #define MQMCEV_REPAIR_ALERT_OFF        30
 #define MQMCEV_RELIABILITY_CHANGED     31
 #define MQMCEV_SHM_DEST_UNUSABLE       80
 #define MQMCEV_SHM_PORT_UNUSABLE       81
 #define MQMCEV_CCT_GETTIME_FAILED      110
 #define MQMCEV_DEST_INTERFACE_FAILURE  120
 #define MQMCEV_DEST_INTERFACE_FAILOVER 121
 #define MQMCEV_PORT_INTERFACE_FAILURE  122
 #define MQMCEV_PORT_INTERFACE_FAILOVER 123

 /****************************************************************/
 /* Values Related to MQ_CHANNEL_EXIT Function                   */
 /****************************************************************/

 /* Channel exit */
 #define MQCHANNELEXIT                  MQ_CHANNEL_EXIT

 /****************************************************************/
 /* Values Related to MQ_CHANNEL_AUTO_DEF_EXIT Function          */
 /****************************************************************/

 /* Channel auto-definition exit */
 #define MQCHANNELAUTODEFEXIT           MQ_CHANNEL_AUTO_DEF_EXIT

 /****************************************************************/
 /* Values Related to MQ_CLUSTER_WORKLOAD_EXIT Function          */
 /****************************************************************/

 /* Channel exit */

 /****************************************************************/
 /* Values Related to MQ_DATA_CONV_EXIT Function                 */
 /****************************************************************/

 /* Data Conversion Exit */
 #define MQDATACONVEXIT                 MQ_DATA_CONV_EXIT

 /****************************************************************/
 /* Values Related to MQ_TRANSPORT_EXIT Function                 */
 /****************************************************************/

 /* Channel exit */
 #define MQTRANSPORTEXIT                MQ_TRANSPORT_EXIT

 /****************************************************************/
 /* Values Related to MQXCNVC Function                           */
 /****************************************************************/

 /* Conversion Options */
 #define MQDCC_DEFAULT_CONVERSION       0x00000001
 #define MQDCC_FILL_TARGET_BUFFER       0x00000002
 #define MQDCC_INT_DEFAULT_CONVERSION   0x00000004
 #define MQDCC_SOURCE_ENC_NATIVE        0x00000020
 #define MQDCC_SOURCE_ENC_NORMAL        0x00000010
 #define MQDCC_SOURCE_ENC_REVERSED      0x00000020
 #define MQDCC_SOURCE_ENC_UNDEFINED     0x00000000
 #define MQDCC_TARGET_ENC_NATIVE        0x00000200
 #define MQDCC_TARGET_ENC_NORMAL        0x00000100
 #define MQDCC_TARGET_ENC_REVERSED      0x00000200
 #define MQDCC_TARGET_ENC_UNDEFINED     0x00000000
 #define MQDCC_NONE                     0x00000000

 /* Conversion Options Masks and Factors */
 #define MQDCC_SOURCE_ENC_MASK          0x000000F0
 #define MQDCC_TARGET_ENC_MASK          0x00000F00
 #define MQDCC_SOURCE_ENC_FACTOR        16
 #define MQDCC_TARGET_ENC_FACTOR        256

 /****************************************************************/
 /* MQCD Structure -- Channel Definition                         */
 /****************************************************************/


 typedef struct tagMQCD MQCD;
 typedef MQCD  MQPOINTER PMQCD;
 typedef PMQCD MQPOINTER PPMQCD;

 struct tagMQCD {
   MQCHAR    ChannelName[20];           /* Channel definition name */
   MQLONG    Version;                   /* Structure version number */
   MQLONG    ChannelType;               /* Channel type */
   MQLONG    TransportType;             /* Transport type */
   MQCHAR    Desc[64];                  /* Channel description */
   MQCHAR    QMgrName[48];              /* Queue-manager name */
   MQCHAR    XmitQName[48];             /* Transmission queue name */
   MQCHAR    ShortConnectionName[20];   /* First 20 bytes of */
                                        /* connection name */
   MQCHAR    MCAName[20];               /* Reserved */
   MQCHAR    ModeName[8];               /* LU 6.2 Mode name */
   MQCHAR    TpName[64];                /* LU 6.2 transaction program */
                                        /* name */
   MQLONG    BatchSize;                 /* Batch size */
   MQLONG    DiscInterval;              /* Disconnect interval */
   MQLONG    ShortRetryCount;           /* Short retry count */
   MQLONG    ShortRetryInterval;        /* Short retry wait interval */
   MQLONG    LongRetryCount;            /* Long retry count */
   MQLONG    LongRetryInterval;         /* Long retry wait interval */
   MQCHAR    SecurityExit[128];         /* Channel security exit name */
   MQCHAR    MsgExit[128];              /* Channel message exit name */
   MQCHAR    SendExit[128];             /* Channel send exit name */
   MQCHAR    ReceiveExit[128];          /* Channel receive exit name */
   MQLONG    SeqNumberWrap;             /* Highest allowable message */
                                        /* sequence number */
   MQLONG    MaxMsgLength;              /* Maximum message length */
   MQLONG    PutAuthority;              /* Put authority */
   MQLONG    DataConversion;            /* Data conversion */
   MQCHAR    SecurityUserData[32];      /* Channel security exit user */
                                        /* data */
   MQCHAR    MsgUserData[32];           /* Channel message exit user */
                                        /* data */
   MQCHAR    SendUserData[32];          /* Channel send exit user */
                                        /* data */
   MQCHAR    ReceiveUserData[32];       /* Channel receive exit user */
                                        /* data */
   /* Ver:1 */
   MQCHAR    UserIdentifier[12];        /* User identifier */
   MQCHAR    Password[12];              /* Password */
   MQCHAR    MCAUserIdentifier[12];     /* First 12 bytes of MCA user */
                                        /* identifier */
   MQLONG    MCAType;                   /* Message channel agent type */
   MQCHAR    ConnectionName[264];       /* Connection name */
   MQCHAR    RemoteUserIdentifier[12];  /* First 12 bytes of user */
                                        /* identifier from partner */
   MQCHAR    RemotePassword[12];        /* Password from partner */
   /* Ver:2 */
   MQCHAR    MsgRetryExit[128];         /* Channel message retry exit */
                                        /* name */
   MQCHAR    MsgRetryUserData[32];      /* Channel message retry exit */
                                        /* user data */
   MQLONG    MsgRetryCount;             /* Number of times MCA will */
                                        /* try to put the message, */
                                        /* after first attempt has */
                                        /* failed */
   MQLONG    MsgRetryInterval;          /* Minimum interval in */
                                        /* milliseconds after which */
                                        /* the open or put operation */
                                        /* will be retried */
   /* Ver:3 */
   MQLONG    HeartbeatInterval;         /* Time in seconds between */
                                        /* heartbeat flows */
   MQLONG    BatchInterval;             /* Batch duration */
   MQLONG    NonPersistentMsgSpeed;     /* Speed at which */
                                        /* nonpersistent messages are */
                                        /* sent */
   MQLONG    StrucLength;               /* Length of MQCD structure */
   MQLONG    ExitNameLength;            /* Length of exit name */
   MQLONG    ExitDataLength;            /* Length of exit user data */
   MQLONG    MsgExitsDefined;           /* Number of message exits */
                                        /* defined */
   MQLONG    SendExitsDefined;          /* Number of send exits */
                                        /* defined */
   MQLONG    ReceiveExitsDefined;       /* Number of receive exits */
                                        /* defined */
   MQPTR     MsgExitPtr;                /* Address of first MsgExit */
                                        /* field */
   MQPTR     MsgUserDataPtr;            /* Address of first */
                                        /* MsgUserData field */
   MQPTR     SendExitPtr;               /* Address of first SendExit */
                                        /* field */
   MQPTR     SendUserDataPtr;           /* Address of first */
                                        /* SendUserData field */
   MQPTR     ReceiveExitPtr;            /* Address of first */
                                        /* ReceiveExit field */
   MQPTR     ReceiveUserDataPtr;        /* Address of first */
                                        /* ReceiveUserData field */
   /* Ver:4 */
   MQPTR     ClusterPtr;                /* Address of a list of */
                                        /* cluster names */
   MQLONG    ClustersDefined;           /* Number of clusters to */
                                        /* which the channel belongs */
   MQLONG    NetworkPriority;           /* Network priority */
   /* Ver:5 */
   MQLONG    LongMCAUserIdLength;       /* Length of long MCA user */
                                        /* identifier */
   MQLONG    LongRemoteUserIdLength;    /* Length of long remote user */
                                        /* identifier */
   MQPTR     LongMCAUserIdPtr;          /* Address of long MCA user */
                                        /* identifier */
   MQPTR     LongRemoteUserIdPtr;       /* Address of long remote */
                                        /* user identifier */
   MQBYTE40  MCASecurityId;             /* MCA security identifier */
   MQBYTE40  RemoteSecurityId;          /* Remote security identifier */
   /* Ver:6 */
   MQCHAR    SSLCipherSpec[32];         /* SSL CipherSpec */
   MQPTR     SSLPeerNamePtr;            /* Address of SSL peer name */
   MQLONG    SSLPeerNameLength;         /* Length of SSL peer name */
   MQLONG    SSLClientAuth;             /* Whether SSL client */
                                        /* authentication is required */
   MQLONG    KeepAliveInterval;         /* Keepalive interval */
   MQCHAR    LocalAddress[48];          /* Local communications */
                                        /* address */
   MQLONG    BatchHeartbeat;            /* Batch heartbeat interval */
   /* Ver:7 */
   MQLONG    HdrCompList[2];            /* Header data compression */
                                        /* list */
   MQLONG    MsgCompList[16];           /* Message data compression */
                                        /* list */
   MQLONG    CLWLChannelRank;           /* Channel rank */
   MQLONG    CLWLChannelPriority;       /* Channel priority */
   MQLONG    CLWLChannelWeight;         /* Channel weight */
   MQLONG    ChannelMonitoring;         /* Channel monitoring */
   MQLONG    ChannelStatistics;         /* Channel statistics */
   /* Ver:8 */
   MQLONG    SharingConversations;      /* Limit on sharing */
                                        /* conversations */
   MQLONG    PropertyControl;           /* Message property control */
   MQLONG    MaxInstances;              /* Limit on SVRCONN channel */
                                        /* instances */
   MQLONG    MaxInstancesPerClient;     /* Limit on SVRCONN channel */
                                        /* instances per client */
   MQLONG    ClientChannelWeight;       /* Client channel weight */
   MQLONG    ConnectionAffinity;        /* Connection affinity */
   /* Ver:9 */
   MQLONG    BatchDataLimit;            /* Batch data limit */
   MQLONG    UseDLQ;                    /* Use Dead Letter Queue */
   MQLONG    DefReconnect;              /* Default client reconnect */
                                        /* option */
   /* Ver:10 */
   MQCHAR    CertificateLabel[64];      /* Certificate label */
   /* Ver:11 */
 };

 #define MQCD_DEFAULT {""},\
                      MQCD_VERSION_6,\
                      MQCHT_SENDER,\
                      MQXPT_LU62,\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      50,\
                      6000,\
                      10,\
                      60,\
                      999999999,\
                      1200,\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      999999999,\
                      4194304,\
                      MQPA_DEFAULT,\
                      MQCDC_NO_SENDER_CONVERSION,\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      MQMCAT_PROCESS,\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      {""},\
                      10,\
                      1000,\
                      300,\
                      0,\
                      MQNPMS_FAST,\
                      MQCD_CURRENT_LENGTH,\
                      MQ_EXIT_NAME_LENGTH,\
                      MQ_EXIT_DATA_LENGTH,\
                      0,\
                      0,\
                      0,\
                      NULL,\
                      NULL,\
                      NULL,\
                      NULL,\
                      NULL,\
                      NULL,\
                      NULL,\
                      0,\
                      0,\
                      0,\
                      0,\
                      NULL,\
                      NULL,\
                      {MQSID_NONE_ARRAY},\
                      {MQSID_NONE_ARRAY},\
                      {""},\
                      NULL,\
                      0,\
                      MQSCA_REQUIRED,\
                      MQKAI_AUTO,\
                      {""},\
                      0,\
                      {MQCOMPRESS_NONE,\
                       MQCOMPRESS_NOT_AVAILABLE},\
                      {MQCOMPRESS_NONE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE,\
                       MQCOMPRESS_NOT_AVAILABLE},\
                      0,\
                      0,\
                      50,\
                      MQMON_OFF,\
                      MQMON_OFF,\
                      10,\
                      MQPROP_COMPATIBILITY,\
                      999999999,\
                      999999999,\
                      0,\
                      MQCAFTY_PREFERRED,\
                      5000,\
                      MQUSEDLQ_YES,\
                      MQRCN_NO,\
                      {""}

/* Initial values for MQCD when passed on MQCONNX function */
 #define MQCD_CLIENT_CONN_DEFAULT {""},\
                                  MQCD_VERSION_6,\
                                  MQCHT_CLNTCONN,\
                                  MQXPT_TCP,\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  50,\
                                  6000,\
                                  10,\
                                  60,\
                                  999999999,\
                                  1200,\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  999999999,\
                                  4194304,\
                                  MQPA_DEFAULT,\
                                  MQCDC_NO_SENDER_CONVERSION,\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  MQMCAT_PROCESS,\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  {""},\
                                  10,\
                                  1000,\
                                  1,\
                                  0,\
                                  MQNPMS_FAST,\
                                  MQCD_CURRENT_LENGTH,\
                                  MQ_EXIT_NAME_LENGTH,\
                                  MQ_EXIT_DATA_LENGTH,\
                                  0,\
                                  0,\
                                  0,\
                                  NULL,\
                                  NULL,\
                                  NULL,\
                                  NULL,\
                                  NULL,\
                                  NULL,\
                                  NULL,\
                                  0,\
                                  0,\
                                  0,\
                                  0,\
                                  NULL,\
                                  NULL,\
                                  {MQSID_NONE_ARRAY},\
                                  {MQSID_NONE_ARRAY},\
                                  {""},\
                                  NULL,\
                                  0,\
                                  MQSCA_REQUIRED,\
                                  (-1),\
                                  {""},\
                                  0,\
                                  {MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE},\
                                  {MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE,\
                                   MQCOMPRESS_NOT_AVAILABLE},\
                                  0,\
                                  0,\
                                  50,\
                                  MQMON_OFF,\
                                  MQMON_OFF,\
                                  10,\
                                  MQPROP_COMPATIBILITY,\
                                  999999999,\
                                  999999999,\
                                  0,\
                                  MQCAFTY_PREFERRED,\
                                  5000,\
                                  MQUSEDLQ_YES,\
                                  MQRCN_NO,\
                                  {""}

 /****************************************************************/
 /* API Exit Chain Area Header                                   */
 /****************************************************************/


 typedef struct tagMQACH MQACH;
 typedef MQACH MQPOINTER PMQACH;

 struct tagMQACH {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    StrucLength;       /* Length of MQACH structure */
   MQLONG    ChainAreaLength;   /* Total length of chain area */
   MQCHAR48  ExitInfoName;      /* Exit information name */
   PMQACH    NextChainAreaPtr;  /* Address of next MQACH structure in */
                                /* chain */
 };

 #define MQACH_DEFAULT {MQACH_STRUC_ID_ARRAY},\
                       MQACH_VERSION_1,\
                       MQACH_CURRENT_LENGTH,\
                       0,\
                       {""},\
                       NULL

 /****************************************************************/
 /* MQAXC Structure -- API Exit Context                          */
 /****************************************************************/


 typedef struct tagMQAXC MQAXC;
 typedef MQAXC MQPOINTER PMQAXC;

 struct tagMQAXC {
   MQCHAR4   StrucId;                 /* Structure identifier */
   MQLONG    Version;                 /* Structure version number */
   MQLONG    Environment;             /* Environment */
   MQCHAR12  UserId;                  /* User identifier */
   MQBYTE40  SecurityId;              /* Security identifier */
   MQCHAR    ConnectionName[264];     /* Connection name */
   MQLONG    LongMCAUserIdLength;     /* Length of long MCA user */
                                      /* identifier */
   MQLONG    LongRemoteUserIdLength;  /* Length of long remote user */
                                      /* identifier */
   MQPTR     LongMCAUserIdPtr;        /* Address of long MCA user */
                                      /* identifier */
   MQPTR     LongRemoteUserIdPtr;     /* Address of long remote user */
                                      /* identifier */
   MQCHAR28  ApplName;                /* Application name */
   MQLONG    ApplType;                /* Application type */
   MQPID     ProcessId;               /* Process identifier */
   MQTID     ThreadId;                /* Thread identifier */
   /* Ver:1 */
   MQCHAR    ChannelName[20];         /* Channel Name */
   MQBYTE4   Reserved1;               /* Reserved */
   PMQCD     pChannelDefinition;      /* Pointer to Channel */
                                      /* Definition */
   /* Ver:2 */
 };

 #define MQAXC_DEFAULT {MQAXC_STRUC_ID_ARRAY},\
                       MQAXC_VERSION_1,\
                       MQXE_OTHER,\
                       {""},\
                       {MQSID_NONE_ARRAY},\
                       {""},\
                       0,\
                       0,\
                       NULL,\
                       NULL,\
                       {""},\
                       MQAT_DEFAULT,\
                       0,\
                       0,\
                       {""},\
                       {'\0','\0','\0','\0'},\
                       NULL

 /****************************************************************/
 /* MQAXP Structure -- API Exit Parameter                        */
 /****************************************************************/


 typedef struct tagMQAXP MQAXP;
 typedef MQAXP MQPOINTER PMQAXP;

 struct tagMQAXP {
   MQCHAR4    StrucId;           /* Structure identifier */
   MQLONG     Version;           /* Structure version number */
   MQLONG     ExitId;            /* Type of exit */
   MQLONG     ExitReason;        /* Reason for invoking exit */
   MQLONG     ExitResponse;      /* Response from exit */
   MQLONG     ExitResponse2;     /* Secondary response from exit */
   MQLONG     Feedback;          /* Feedback */
   MQLONG     APICallerType;     /* API caller type */
   MQBYTE16   ExitUserArea;      /* Exit user area */
   MQCHAR32   ExitData;          /* Exit data */
   MQCHAR48   ExitInfoName;      /* Exit information name */
   MQBYTE48   ExitPDArea;        /* Problem determination area */
   MQCHAR48   QMgrName;          /* Name of local queue manager */
   PMQACH     ExitChainAreaPtr;  /* Address of first MQACH structure */
                                 /* in chain */
   MQHCONFIG  Hconfig;           /* Configuration handle */
   MQLONG     Function;          /* API function identifier */
   MQHMSG     ExitMsgHandle;     /* Exit message handle */
 };

 #define MQAXP_DEFAULT {MQAXP_STRUC_ID_ARRAY},\
                       MQAXP_VERSION_1,\
                       MQXT_API_EXIT,\
                       0,\
                       MQXCC_OK,\
                       MQXR2_DEFAULT_CONTINUATION,\
                       MQFB_NONE,\
                       MQXACT_EXTERNAL,\
                       {MQXUA_NONE_ARRAY},\
                       {""},\
                       {""},\
                       {MQXPDA_NONE_ARRAY},\
                       {""},\
                       NULL,\
                       NULL,\
                       0,\
                       MQHM_NONE

 /****************************************************************/
 /* MQCXP Structure -- Channel Exit Parameter                    */
 /****************************************************************/


 typedef struct tagMQCXP MQCXP;
 typedef MQCXP MQPOINTER PMQCXP;

 struct tagMQCXP {
   MQCHAR4   StrucId;                  /* Structure identifier */
   MQLONG    Version;                  /* Structure version number */
   MQLONG    ExitId;                   /* Type of exit */
   MQLONG    ExitReason;               /* Reason for invoking exit */
   MQLONG    ExitResponse;             /* Response from exit */
   MQLONG    ExitResponse2;            /* Secondary response from */
                                       /* exit */
   MQLONG    Feedback;                 /* Feedback code */
   MQLONG    MaxSegmentLength;         /* Maximum segment length */
   MQBYTE16  ExitUserArea;             /* Exit user area */
   MQCHAR32  ExitData;                 /* Exit data */
   MQLONG    MsgRetryCount;            /* Number of times the message */
                                       /* has been retried */
   MQLONG    MsgRetryInterval;         /* Minimum interval in */
                                       /* milliseconds after which */
                                       /* the put operation should be */
                                       /* retried */
   MQLONG    MsgRetryReason;           /* Reason code from previous */
                                       /* attempt to put the message */
   MQLONG    HeaderLength;             /* Length of header */
                                       /* information */
   MQCHAR48  PartnerName;              /* Partner Name */
   MQLONG    FAPLevel;                 /* Negotiated Formats and */
                                       /* Protocols level */
   MQLONG    CapabilityFlags;          /* Capability flags */
   MQLONG    ExitNumber;               /* Exit number */
   /* Ver:3 */
   /* Ver:4 */
   MQLONG    ExitSpace;                /* Number of bytes in */
                                       /* transmission buffer */
                                       /* reserved for exit to use */
   /* Ver:5 */
   MQCHAR12  SSLCertUserid;            /* User identifier associated */
                                       /* with remote SSL certificate */
   MQLONG    SSLRemCertIssNameLength;  /* Length of distinguished */
                                       /* name of issuer of remote */
                                       /* SSL certificate */
   MQPTR     SSLRemCertIssNamePtr;     /* Address of distinguished */
                                       /* name of issuer of remote */
                                       /* SSL certificate */
   PMQCSP    SecurityParms;            /* Address of security */
                                       /* parameters */
   MQLONG    CurHdrCompression;        /* Header data compression */
                                       /* used for current message */
   MQLONG    CurMsgCompression;        /* Message data compression */
                                       /* used for current message */
   /* Ver:6 */
   MQHCONN   Hconn;                    /* Connection handle */
   MQBOOL    SharingConversations;     /* Multiple conversations */
                                       /* allowed */
   /* Ver:7 */
   MQLONG    MCAUserSource;            /* The source of the run-time */
                                       /* user ID */
   PMQIEP    pEntryPoints;             /* Interface entry points */
   /* Ver:8 */
   MQCHAR4   RemoteProduct;            /* The identifier for the */
                                       /* remote product */
   MQCHAR8   RemoteVersion;            /* The version of the remote */
                                       /* product */
   /* Ver:9 */
 };

 #define MQCXP_DEFAULT {MQCXP_STRUC_ID_ARRAY},\
                       MQCXP_VERSION_5,\
                       0,\
                       0,\
                       MQXCC_OK,\
                       MQXR2_PUT_WITH_DEF_ACTION,\
                       0,\
                       0,\
                       {MQXUA_NONE_ARRAY},\
                       {""},\
                       0,\
                       0,\
                       0,\
                       0,\
                       {""},\
                       0,\
                       MQCF_NONE,\
                       1,\
                       0,\
                       {""},\
                       0,\
                       NULL,\
                       NULL,\
                       MQCOMPRESS_NONE,\
                       MQCOMPRESS_NONE,\
                       MQHC_UNUSABLE_HCONN,\
                       0,\
                       MQUSRC_MAP,\
                       NULL,\
                       {""},\
                       {""}

 /****************************************************************/
 /* MQDXP Structure -- Data Conversion Exit Parameter            */
 /****************************************************************/


 typedef struct tagMQDXP MQDXP;
 typedef MQDXP MQPOINTER PMQDXP;

 struct tagMQDXP {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   ExitOptions;     /* Reserved */
   MQLONG   AppOptions;      /* Application options */
   MQLONG   Encoding;        /* Numeric encoding required by */
                             /* application */
   MQLONG   CodedCharSetId;  /* Character set required by application */
   MQLONG   DataLength;      /* Length in bytes of message data */
   MQLONG   CompCode;        /* Completion code */
   MQLONG   Reason;          /* Reason code qualifying CompCode */
   MQLONG   ExitResponse;    /* Response from exit */
   MQHCONN  Hconn;           /* Connection handle */
   /* Ver:1 */
   PMQIEP   pEntryPoints;    /* Interface entry points */
   /* Ver:2 */
 };

 #define MQDXP_DEFAULT {MQDXP_STRUC_ID_ARRAY},\
                       MQDXP_VERSION_1,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       MQCC_OK,\
                       MQRC_NONE,\
                       MQXDR_CONVERSION_FAILED,\
                       0,\
                       NULL

 /****************************************************************/
 /* MQNXP Structure -- PreConnect Exit options                   */
 /****************************************************************/


 typedef struct tagMQNXP MQNXP;
 typedef MQNXP MQPOINTER PMQNXP;

 struct tagMQNXP {
   MQCHAR4  StrucId;           /* Structure identifier */
   MQLONG   Version;           /* Structure version number */
   MQLONG   ExitId;            /* Type of exit */
   MQLONG   ExitReason;        /* Reason for invoking exit */
   MQLONG   ExitResponse;      /* Response from exit */
   MQLONG   ExitResponse2;     /* Secondary response from exit */
   MQLONG   Feedback;          /* Feedback */
   MQLONG   ExitDataLength;    /* Length of exit data */
   PMQCHAR  pExitDataPtr;      /* Address of exit data */
   MQPTR    pExitUserAreaPtr;  /* Address of exit user area */
   PPMQCD   ppMQCDArrayPtr;    /* Address of pointers referencing */
                               /* MQCDs */
   MQLONG   MQCDArrayCount;    /* Count of MQCDs referenced */
   MQLONG   MaxMQCDVersion;    /* Maximum MQCD version requested */
   /* Ver:1 */
   PMQIEP   pEntryPoints;      /* Interface entry points */
   /* Ver:2 */
 };

 #define MQNXP_DEFAULT {MQNXP_STRUC_ID_ARRAY},\
                       MQNXP_VERSION_1,\
                       MQXT_PRECONNECT_EXIT,\
                       0,\
                       MQXCC_OK,\
                       MQXR2_DEFAULT_CONTINUATION,\
                       MQFB_NONE,\
                       0,\
                       NULL,\
                       NULL,\
                       NULL,\
                       0,\
                       MQCD_CURRENT_VERSION,\
                       NULL

 /****************************************************************/
 /* MQPBC Structure -- Publish Exit Publication Context          */
 /****************************************************************/


 typedef struct tagMQPBC MQPBC;
 typedef MQPBC MQPOINTER PMQPBC;

 struct tagMQPBC {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQCHARV  PubTopicString;  /* Publish topic string */
   /* Ver:1 */
   PMQMD    MsgDescPtr;      /* Address of publisher message */
                             /* descriptor */
   /* Ver:2 */
 };

 #define MQPBC_DEFAULT {MQPBC_STRUC_ID_ARRAY},\
                       MQPBC_VERSION_1,\
                       {MQCHARV_DEFAULT},\
                       NULL

 /****************************************************************/
 /* MQPSXP Structure -- Publish Exit Parameter                   */
 /****************************************************************/


 typedef struct tagMQPSXP MQPSXP;
 typedef MQPSXP MQPOINTER PMQPSXP;

 struct tagMQPSXP {
   MQCHAR4   StrucId;        /* Structure identifier */
   MQLONG    Version;        /* Structure version number */
   MQLONG    ExitId;         /* Type of exit */
   MQLONG    ExitReason;     /* Reason for invoking exit */
   MQLONG    ExitResponse;   /* Response from exit */
   MQLONG    ExitResponse2;  /* Reserved */
   MQLONG    Feedback;       /* Feedback code */
   MQHCONN   Hconn;          /* Connection handle */
   MQBYTE16  ExitUserArea;   /* Exit user area */
   MQCHAR32  ExitData;       /* Exit data */
   MQCHAR48  QMgrName;       /* Name of local queue manager */
   MQHMSG    MsgHandle;      /* Handle to message properties */
   PMQMD     MsgDescPtr;     /* Address of message descriptor */
   PMQVOID   MsgInPtr;       /* Address of input message data */
   MQLONG    MsgInLength;    /* Length of input message data */
   PMQVOID   MsgOutPtr;      /* Address of output message data */
   MQLONG    MsgOutLength;   /* Length of output message data */
   /* Ver:1 */
   PMQIEP    pEntryPoints;   /* Interface entry points */
   /* Ver:2 */
 };

 #define MQPSXP_DEFAULT {MQPSXP_STRUC_ID_ARRAY},\
                        MQPSXP_VERSION_1,\
                        MQXT_PUBLISH_EXIT,\
                        0,\
                        MQXCC_OK,\
                        0,\
                        MQFB_NONE,\
                        MQHC_UNUSABLE_HCONN,\
                        {MQXUA_NONE_ARRAY},\
                        {""},\
                        {""},\
                        MQHM_NONE,\
                        NULL,\
                        NULL,\
                        0,\
                        NULL,\
                        0,\
                        NULL

 /****************************************************************/
 /* MQSBC Structure -- Publish Exit Subscription Context         */
 /****************************************************************/


 typedef struct tagMQSBC MQSBC;
 typedef MQSBC MQPOINTER PMQSBC;

 struct tagMQSBC {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQCHAR48  DestinationQMgrName;  /* Destination queue manager */
   MQCHAR48  DestinationQName;     /* Destination queue name */
   MQLONG    SubType;              /* Type of subscription */
   MQLONG    SubOptions;           /* Subscription options */
   MQCHAR48  ObjectName;           /* Object name */
   MQCHARV   ObjectString;         /* Object string */
   MQCHARV   SubTopicString;       /* Subscription topic string */
   MQCHARV   SubName;              /* Subscription name */
   MQBYTE24  SubId;                /* Subscription identifier */
   MQCHARV   SelectionString;      /* Subscription selection string */
   MQLONG    SubLevel;             /* Subscription level */
   MQLONG    PSProperties;         /* Publish/subscribe properties */
 };

 #define MQSBC_DEFAULT {MQSBC_STRUC_ID_ARRAY},\
                       MQSBC_VERSION_1,\
                       "",\
                       "",\
                       0,\
                       0,\
                       "",\
                       {MQCHARV_DEFAULT},\
                       {MQCHARV_DEFAULT},\
                       {MQCHARV_DEFAULT},\
                       {MQCI_NONE_ARRAY},\
                       {MQCHARV_DEFAULT},\
                       1,\
                       0

 /****************************************************************/
 /* MQWCR Structure -- Cluster Workload Exit Cluster Record      */
 /****************************************************************/


 typedef struct tagMQWCR MQWCR;
 typedef MQWCR MQPOINTER PMQWCR;

 struct tagMQWCR {
   MQCHAR48  ClusterName;       /* Cluster name */
   MQLONG    ClusterRecOffset;  /* Offset of next cluster record */
   MQLONG    ClusterFlags;      /* Cluster flags */
 };

 #define MQWCR_DEFAULT {""},\
                       0,\
                       0

 /****************************************************************/
 /* MQWDR Structure -- Cluster Workload Exit Destination Record  */
 /****************************************************************/


 typedef struct tagMQWDR MQWDR;
 typedef MQWDR  MQPOINTER PMQWDR;
 typedef PMQWDR MQPOINTER PPMQWDR;

 struct tagMQWDR {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    StrucLength;       /* Length of MQWDR structure */
   MQLONG    QMgrFlags;         /* Queue-manager flags */
   MQCHAR48  QMgrIdentifier;    /* Queue-manager identifier */
   MQCHAR48  QMgrName;          /* Queue-manager name */
   MQLONG    ClusterRecOffset;  /* Offset of first cluster record */
   MQLONG    ChannelState;      /* Channel state */
   MQLONG    ChannelDefOffset;  /* Offset of channel definition */
                                /* structure */
   /* Ver:1 */
   MQLONG    DestSeqNumber;     /* Cluster channel destination */
                                /* sequence number */
   MQINT64   DestSeqFactor;     /* Cluster channel destination */
                                /* sequence factor */
   /* Ver:2 */
 };

 #define MQWDR_DEFAULT {MQWDR_STRUC_ID_ARRAY},\
                       MQWDR_VERSION_1,\
                       MQWDR_CURRENT_LENGTH,\
                       0,\
                       {""},\
                       {""},\
                       0,\
                       0,\
                       0,\
                       0,\
                       0

 /****************************************************************/
 /* MQWDR1 Structure -- Version-1 CLWL Exit Destination Record   */
 /****************************************************************/


 typedef struct tagMQWDR1 MQWDR1;
 typedef MQWDR1  MQPOINTER PMQWDR1;
 typedef PMQWDR1 MQPOINTER PPMQWDR1;

 struct tagMQWDR1 {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    StrucLength;       /* Length of MQWDR structure */
   MQLONG    QMgrFlags;         /* Queue-manager flags */
   MQCHAR48  QMgrIdentifier;    /* Queue-manager identifier */
   MQCHAR48  QMgrName;          /* Queue-manager name */
   MQLONG    ClusterRecOffset;  /* Offset of first cluster record */
   MQLONG    ChannelState;      /* Channel state */
   MQLONG    ChannelDefOffset;  /* Offset of channel definition */
                                /* structure */
 };

 #define MQWDR1_DEFAULT {MQWDR_STRUC_ID_ARRAY},\
                        MQWDR_VERSION_1,\
                        MQWDR_LENGTH_1,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        0

 /****************************************************************/
 /* MQWDR2 Structure -- Version-2 CLWL Exit Destination Record   */
 /****************************************************************/


 typedef struct tagMQWDR2 MQWDR2;
 typedef MQWDR2  MQPOINTER PMQWDR2;
 typedef PMQWDR2 MQPOINTER PPMQWDR2;

 struct tagMQWDR2 {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    StrucLength;       /* Length of MQWDR structure */
   MQLONG    QMgrFlags;         /* Queue-manager flags */
   MQCHAR48  QMgrIdentifier;    /* Queue-manager identifier */
   MQCHAR48  QMgrName;          /* Queue-manager name */
   MQLONG    ClusterRecOffset;  /* Offset of first cluster record */
   MQLONG    ChannelState;      /* Channel state */
   MQLONG    ChannelDefOffset;  /* Offset of channel definition */
                                /* structure */
   /* Ver:1 */
   MQLONG    DestSeqNumber;     /* Cluster channel destination */
                                /* sequence number */
   MQINT64   DestSeqFactor;     /* Cluster channel destination */
                                /* sequence factor */
   /* Ver:2 */
 };

 #define MQWDR2_DEFAULT {MQWDR_STRUC_ID_ARRAY},\
                        MQWDR_VERSION_2,\
                        MQWDR_LENGTH_2,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        0,\
                        0,\
                        0

 /****************************************************************/
 /* MQWQR Structure -- Cluster Workload Exit Queue Record        */
 /****************************************************************/


 typedef struct tagMQWQR MQWQR;
 typedef MQWQR  MQPOINTER PMQWQR;
 typedef PMQWQR MQPOINTER PPMQWQR;

 struct tagMQWQR {
   MQCHAR4   StrucId;            /* Structure identifier */
   MQLONG    Version;            /* Structure version number */
   MQLONG    StrucLength;        /* Length of MQWQR structure */
   MQLONG    QFlags;             /* Queue flags */
   MQCHAR48  QName;              /* Queue name */
   MQCHAR48  QMgrIdentifier;     /* Queue-manager identifier */
   MQLONG    ClusterRecOffset;   /* Offset of first cluster record */
   MQLONG    QType;              /* Queue type */
   MQCHAR64  QDesc;              /* Queue description */
   MQLONG    DefBind;            /* Default binding */
   MQLONG    DefPersistence;     /* Default message persistence */
   MQLONG    DefPriority;        /* Default message priority */
   MQLONG    InhibitPut;         /* Whether put operations on the */
                                 /* queue are allowed */
   /* Ver:1 */
   MQLONG    CLWLQueuePriority;  /* Queue priority */
   MQLONG    CLWLQueueRank;      /* Queue rank */
   /* Ver:2 */
   MQLONG    DefPutResponse;     /* Default put response */
   /* Ver:3 */
 };

 #define MQWQR_DEFAULT {MQWQR_STRUC_ID_ARRAY},\
                       MQWQR_VERSION_1,\
                       MQWQR_CURRENT_LENGTH,\
                       0,\
                       {""},\
                       {""},\
                       0,\
                       0,\
                       {""},\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       0,\
                       1

 /****************************************************************/
 /* MQWQR1 Structure -- Version-1 CLWL Exit Queue Record         */
 /****************************************************************/


 typedef struct tagMQWQR1 MQWQR1;
 typedef MQWQR1  MQPOINTER PMQWQR1;
 typedef PMQWQR1 MQPOINTER PPMQWQR1;

 struct tagMQWQR1 {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQLONG    StrucLength;       /* Length of MQWQR structure */
   MQLONG    QFlags;            /* Queue flags */
   MQCHAR48  QName;             /* Queue name */
   MQCHAR48  QMgrIdentifier;    /* Queue-manager identifier */
   MQLONG    ClusterRecOffset;  /* Offset of first cluster record */
   MQLONG    QType;             /* Queue type */
   MQCHAR64  QDesc;             /* Queue description */
   MQLONG    DefBind;           /* Default binding */
   MQLONG    DefPersistence;    /* Default message persistence */
   MQLONG    DefPriority;       /* Default message priority */
   MQLONG    InhibitPut;        /* Whether put operations on the */
                                /* queue are allowed */
 };

 #define MQWQR1_DEFAULT {MQWQR_STRUC_ID_ARRAY},\
                        MQWQR_VERSION_1,\
                        MQWQR_LENGTH_1,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        {""},\
                        0,\
                        0,\
                        0,\
                        0

 /****************************************************************/
 /* MQWQR2 Structure -- Version-2 CLWL Exit Queue Record         */
 /****************************************************************/


 typedef struct tagMQWQR2 MQWQR2;
 typedef MQWQR2  MQPOINTER PMQWQR2;
 typedef PMQWQR2 MQPOINTER PPMQWQR2;

 struct tagMQWQR2 {
   MQCHAR4   StrucId;            /* Structure identifier */
   MQLONG    Version;            /* Structure version number */
   MQLONG    StrucLength;        /* Length of MQWQR structure */
   MQLONG    QFlags;             /* Queue flags */
   MQCHAR48  QName;              /* Queue name */
   MQCHAR48  QMgrIdentifier;     /* Queue-manager identifier */
   MQLONG    ClusterRecOffset;   /* Offset of first cluster record */
   MQLONG    QType;              /* Queue type */
   MQCHAR64  QDesc;              /* Queue description */
   MQLONG    DefBind;            /* Default binding */
   MQLONG    DefPersistence;     /* Default message persistence */
   MQLONG    DefPriority;        /* Default message priority */
   MQLONG    InhibitPut;         /* Whether put operations on the */
                                 /* queue are allowed */
   /* Ver:1 */
   MQLONG    CLWLQueuePriority;  /* Queue priority */
   MQLONG    CLWLQueueRank;      /* Queue rank */
   /* Ver:2 */
 };

 #define MQWQR2_DEFAULT {MQWQR_STRUC_ID_ARRAY},\
                        MQWQR_VERSION_2,\
                        MQWQR_LENGTH_2,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        {""},\
                        0,\
                        0,\
                        0,\
                        0,\
                        0,\
                        0

 /****************************************************************/
 /* MQWQR3 Structure -- Version-3 CLWL Exit Queue         Record */
 /****************************************************************/


 typedef struct tagMQWQR3 MQWQR3;
 typedef MQWQR3  MQPOINTER PMQWQR3;
 typedef PMQWQR3 MQPOINTER PPMQWQR3;

 struct tagMQWQR3 {
   MQCHAR4   StrucId;            /* Structure identifier */
   MQLONG    Version;            /* Structure version number */
   MQLONG    StrucLength;        /* Length of MQWQR structure */
   MQLONG    QFlags;             /* Queue flags */
   MQCHAR48  QName;              /* Queue name */
   MQCHAR48  QMgrIdentifier;     /* Queue-manager identifier */
   MQLONG    ClusterRecOffset;   /* Offset of first cluster record */
   MQLONG    QType;              /* Queue type */
   MQCHAR64  QDesc;              /* Queue description */
   MQLONG    DefBind;            /* Default binding */
   MQLONG    DefPersistence;     /* Default message persistence */
   MQLONG    DefPriority;        /* Default message priority */
   MQLONG    InhibitPut;         /* Whether put operations on the */
                                 /* queue are allowed */
   /* Ver:1 */
   MQLONG    CLWLQueuePriority;  /* Queue priority */
   MQLONG    CLWLQueueRank;      /* Queue rank */
   /* Ver:2 */
   MQLONG    DefPutResponse;     /* Default put response */
   /* Ver:3 */
 };

 #define MQWQR3_DEFAULT {MQWQR_STRUC_ID_ARRAY},\
                        MQWQR_VERSION_3,\
                        MQWQR_LENGTH_3,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        {""},\
                        0,\
                        0,\
                        0,\
                        0,\
                        0,\
                        0,\
                        1

 /****************************************************************/
 /* MQWXP Structure -- Cluster Workload Exit Parameter           */
 /****************************************************************/


 typedef struct tagMQWXP MQWXP;
 typedef MQWXP MQPOINTER PMQWXP;

 struct tagMQWXP {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ExitId;               /* Type of exit */
   MQLONG    ExitReason;           /* Reason for invoking exit */
   MQLONG    ExitResponse;         /* Response from exit */
   MQLONG    ExitResponse2;        /* Secondary response from exit */
   MQLONG    Feedback;             /* Reserved */
   MQLONG    Flags;                /* Flags */
   MQBYTE16  ExitUserArea;         /* Exit user area */
   MQCHAR32  ExitData;             /* Exit data */
   PMQMD     MsgDescPtr;           /* Address of message descriptor */
   PMQVOID   MsgBufferPtr;         /* Address of buffer containing */
                                   /* some or all of the message data */
   MQLONG    MsgBufferLength;      /* Length of buffer containing */
                                   /* message data */
   MQLONG    MsgLength;            /* Length of complete message */
   MQCHAR48  QName;                /* Queue name */
   MQCHAR48  QMgrName;             /* Name of local queue manager */
   MQLONG    DestinationCount;     /* Number of possible destinations */
   MQLONG    DestinationChosen;    /* Destination chosen */
   PPMQWDR   DestinationArrayPtr;  /* Address of an array of pointers */
                                   /* to destination records */
   PPMQWQR   QArrayPtr;            /* Address of an array of pointers */
                                   /* to queue records */
   /* Ver:1 */
   MQPTR     CacheContext;         /* Context information */
   MQLONG    CacheType;            /* Type of cluster cache */
   /* Ver:2 */
   MQLONG    CLWLMRUChannels;      /* Number of allowed active */
                                   /* outbound channels */
   /* Ver:3 */
   PMQIEP    pEntryPoints;         /* Interface entry points */
   /* Ver:4 */
 };

 #define MQWXP_DEFAULT {MQWXP_STRUC_ID_ARRAY},\
                       MQWXP_VERSION_2,\
                       0,\
                       0,\
                       MQXCC_OK,\
                       0,\
                       0,\
                       0,\
                       {MQXUA_NONE_ARRAY},\
                       {""},\
                       NULL,\
                       NULL,\
                       0,\
                       0,\
                       {""},\
                       {""},\
                       0,\
                       0,\
                       NULL,\
                       NULL,\
                       NULL,\
                       MQCLCT_DYNAMIC,\
                       0,\
                       NULL

 /****************************************************************/
 /* MQWXP1 Structure -- Version-1 CLWL Exit Parameter            */
 /****************************************************************/


 typedef struct tagMQWXP1 MQWXP1;
 typedef MQWXP1 MQPOINTER PMQWXP1;

 struct tagMQWXP1 {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ExitId;               /* Type of exit */
   MQLONG    ExitReason;           /* Reason for invoking exit */
   MQLONG    ExitResponse;         /* Response from exit */
   MQLONG    ExitResponse2;        /* Secondary response from exit */
   MQLONG    Feedback;             /* Reserved */
   MQLONG    Flags;                /* Flags */
   MQBYTE16  ExitUserArea;         /* Exit user area */
   MQCHAR32  ExitData;             /* Exit data */
   PMQMD     MsgDescPtr;           /* Address of message descriptor */
   PMQVOID   MsgBufferPtr;         /* Address of buffer containing */
                                   /* some or all of the message data */
   MQLONG    MsgBufferLength;      /* Length of buffer containing */
                                   /* message data */
   MQLONG    MsgLength;            /* Length of complete message */
   MQCHAR48  QName;                /* Queue name */
   MQCHAR48  QMgrName;             /* Name of local queue manager */
   MQLONG    DestinationCount;     /* Number of possible destinations */
   MQLONG    DestinationChosen;    /* Destination chosen */
   PPMQWDR   DestinationArrayPtr;  /* Address of an array of pointers */
                                   /* to destination records */
   PPMQWQR   QArrayPtr;            /* Address of an array of pointers */
                                   /* to queue records */
 };

 #define MQWXP1_DEFAULT {MQWXP_STRUC_ID_ARRAY},\
                        MQWXP_VERSION_1,\
                        0,\
                        0,\
                        MQXCC_OK,\
                        0,\
                        0,\
                        0,\
                        {MQXUA_NONE_ARRAY},\
                        {""},\
                        NULL,\
                        NULL,\
                        0,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        NULL,\
                        NULL

 /****************************************************************/
 /* MQWXP2 Structure -- Version-2 CLWL Exit Parameter            */
 /****************************************************************/


 typedef struct tagMQWXP2 MQWXP2;
 typedef MQWXP2 MQPOINTER PMQWXP2;

 struct tagMQWXP2 {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ExitId;               /* Type of exit */
   MQLONG    ExitReason;           /* Reason for invoking exit */
   MQLONG    ExitResponse;         /* Response from exit */
   MQLONG    ExitResponse2;        /* Secondary response from exit */
   MQLONG    Feedback;             /* Reserved */
   MQLONG    Flags;                /* Flags */
   MQBYTE16  ExitUserArea;         /* Exit user area */
   MQCHAR32  ExitData;             /* Exit data */
   PMQMD     MsgDescPtr;           /* Address of message descriptor */
   PMQVOID   MsgBufferPtr;         /* Address of buffer containing */
                                   /* some or all of the message data */
   MQLONG    MsgBufferLength;      /* Length of buffer containing */
                                   /* message data */
   MQLONG    MsgLength;            /* Length of complete message */
   MQCHAR48  QName;                /* Queue name */
   MQCHAR48  QMgrName;             /* Name of local queue manager */
   MQLONG    DestinationCount;     /* Number of possible destinations */
   MQLONG    DestinationChosen;    /* Destination chosen */
   PPMQWDR   DestinationArrayPtr;  /* Address of an array of pointers */
                                   /* to destination records */
   PPMQWQR   QArrayPtr;            /* Address of an array of pointers */
                                   /* to queue records */
   /* Ver:1 */
   MQPTR     CacheContext;         /* Context information */
   MQLONG    CacheType;            /* Type of cluster cache */
   /* Ver:2 */
 };

 #define MQWXP2_DEFAULT {MQWXP_STRUC_ID_ARRAY},\
                        MQWXP_VERSION_2,\
                        0,\
                        0,\
                        MQXCC_OK,\
                        0,\
                        0,\
                        0,\
                        {MQXUA_NONE_ARRAY},\
                        {""},\
                        NULL,\
                        NULL,\
                        0,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        NULL,\
                        NULL,\
                        NULL,\
                        MQCLCT_DYNAMIC

 /****************************************************************/
 /* MQWXP3 Structure -- Version-3 CLWL Exit Parameter            */
 /****************************************************************/


 typedef struct tagMQWXP3 MQWXP3;
 typedef MQWXP3 MQPOINTER PMQWXP3;

 struct tagMQWXP3 {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ExitId;               /* Type of exit */
   MQLONG    ExitReason;           /* Reason for invoking exit */
   MQLONG    ExitResponse;         /* Response from exit */
   MQLONG    ExitResponse2;        /* Secondary response from exit */
   MQLONG    Feedback;             /* Reserved */
   MQLONG    Flags;                /* Flags */
   MQBYTE16  ExitUserArea;         /* Exit user area */
   MQCHAR32  ExitData;             /* Exit data */
   PMQMD     MsgDescPtr;           /* Address of message descriptor */
   PMQVOID   MsgBufferPtr;         /* Address of buffer containing */
                                   /* some or all of the message data */
   MQLONG    MsgBufferLength;      /* Length of buffer containing */
                                   /* message data */
   MQLONG    MsgLength;            /* Length of complete message */
   MQCHAR48  QName;                /* Queue name */
   MQCHAR48  QMgrName;             /* Name of local queue manager */
   MQLONG    DestinationCount;     /* Number of possible destinations */
   MQLONG    DestinationChosen;    /* Destination chosen */
   PPMQWDR   DestinationArrayPtr;  /* Address of an array of pointers */
                                   /* to destination records */
   PPMQWQR   QArrayPtr;            /* Address of an array of pointers */
                                   /* to queue records */
   /* Ver:1 */
   MQPTR     CacheContext;         /* Context information */
   MQLONG    CacheType;            /* Type of cluster cache */
   /* Ver:2 */
   MQLONG    CLWLMRUChannels;      /* Number of allowed active */
                                   /* outbound channels */
   /* Ver:3 */
 };

 #define MQWXP3_DEFAULT {MQWXP_STRUC_ID_ARRAY},\
                        MQWXP_VERSION_3,\
                        0,\
                        0,\
                        MQXCC_OK,\
                        0,\
                        0,\
                        0,\
                        {MQXUA_NONE_ARRAY},\
                        {""},\
                        NULL,\
                        NULL,\
                        0,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        NULL,\
                        NULL,\
                        NULL,\
                        MQCLCT_DYNAMIC,\
                        0

 /****************************************************************/
 /* MQWXP4 Structure -- Version-4 CLWL Exit Parameter            */
 /****************************************************************/


 typedef struct tagMQWXP4 MQWXP4;
 typedef MQWXP4 MQPOINTER PMQWXP4;

 struct tagMQWXP4 {
   MQCHAR4   StrucId;              /* Structure identifier */
   MQLONG    Version;              /* Structure version number */
   MQLONG    ExitId;               /* Type of exit */
   MQLONG    ExitReason;           /* Reason for invoking exit */
   MQLONG    ExitResponse;         /* Response from exit */
   MQLONG    ExitResponse2;        /* Secondary response from exit */
   MQLONG    Feedback;             /* Reserved */
   MQLONG    Flags;                /* Flags */
   MQBYTE16  ExitUserArea;         /* Exit user area */
   MQCHAR32  ExitData;             /* Exit data */
   PMQMD     MsgDescPtr;           /* Address of message descriptor */
   PMQVOID   MsgBufferPtr;         /* Address of buffer containing */
                                   /* some or all of the message data */
   MQLONG    MsgBufferLength;      /* Length of buffer containing */
                                   /* message data */
   MQLONG    MsgLength;            /* Length of complete message */
   MQCHAR48  QName;                /* Queue name */
   MQCHAR48  QMgrName;             /* Name of local queue manager */
   MQLONG    DestinationCount;     /* Number of possible destinations */
   MQLONG    DestinationChosen;    /* Destination chosen */
   PPMQWDR   DestinationArrayPtr;  /* Address of an array of pointers */
                                   /* to destination records */
   PPMQWQR   QArrayPtr;            /* Address of an array of pointers */
                                   /* to queue records */
   /* Ver:1 */
   MQPTR     CacheContext;         /* Context information */
   MQLONG    CacheType;            /* Type of cluster cache */
   /* Ver:2 */
   MQLONG    CLWLMRUChannels;      /* Number of allowed active */
                                   /* outbound channels */
   /* Ver:3 */
   PMQIEP    pEntryPoints;         /* Interface entry points */
   /* Ver:4 */
 };

 #define MQWXP4_DEFAULT {MQWXP_STRUC_ID_ARRAY},\
                        MQWXP_VERSION_4,\
                        0,\
                        0,\
                        MQXCC_OK,\
                        0,\
                        0,\
                        0,\
                        {MQXUA_NONE_ARRAY},\
                        {""},\
                        NULL,\
                        NULL,\
                        0,\
                        0,\
                        {""},\
                        {""},\
                        0,\
                        0,\
                        NULL,\
                        NULL,\
                        NULL,\
                        MQCLCT_DYNAMIC,\
                        0,\
                        NULL

 /****************************************************************/
 /* MQXEPO Structure -- Register entry point options             */
 /****************************************************************/


 typedef struct tagMQXEPO MQXEPO;
 typedef MQXEPO MQPOINTER PMQXEPO;

 struct tagMQXEPO {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQLONG   Options;         /* Options that control the action of */
                             /* MQXEP */
   MQCHARV  ExitProperties;  /* Exit properties */
 };

 #define MQXEPO_DEFAULT {MQXEPO_STRUC_ID_ARRAY},\
                        MQXEPO_VERSION_1,\
                        MQXEPO_NONE,\
                        { NULL,\
                          0,\
                          0,\
                          0,\
                          MQCCSI_APPL }

 /****************************************************************/
 /* API Exit Functions                                           */
 /****************************************************************/

 /****************************************************************/
 /* MQXEP Function -- Register Entry Point                       */
 /****************************************************************/

 void MQENTRY MQXEP (
   MQHCONFIG  Hconfig,      /* I: Configuration handle */
   MQLONG     ExitReason,   /* I: Exit reason */
   MQLONG     Function,     /* I: Function identifier */
   PMQFUNC    pEntryPoint,  /* I: Exit function entry point */
   PMQXEPO    pExitOpts,    /* I: Options that control the action of */
                            /* MQXEP */
   PMQLONG    pCompCode,    /* O: Completion code */
   PMQLONG    pReason);     /* O: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_XEP_CALL (
   MQHCONFIG  Hconfig,      /* I: Configuration handle */
   MQLONG     ExitReason,   /* I: Exit reason */
   MQLONG     Function,     /* I: Function identifier */
   PMQFUNC    pEntryPoint,  /* I: Exit function entry point */
   PMQXEPO    pExitOpts,    /* I: Options that control the action of */
                            /* MQXEP */
   PMQLONG    pCompCode,    /* O: Completion code */
   PMQLONG    pReason);     /* O: Reason code qualifying CompCode */
 typedef MQ_XEP_CALL MQPOINTER PMQ_XEP_CALL;


 /****************************************************************/
 /* MQ_BACK_EXIT -- Back Out Changes Exit                        */
 /****************************************************************/

 typedef void MQENTRY MQ_BACK_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_BACK_EXIT MQPOINTER PMQ_BACK_EXIT;


 /****************************************************************/
 /* MQ_BEGIN_EXIT -- Begin Unit of Work Exit                     */
 /****************************************************************/

 typedef void MQENTRY MQ_BEGIN_EXIT (
   PMQAXP    pExitParms,      /* IO: Exit parameter structure */
   PMQAXC    pExitContext,    /* IO: Exit context structure */
   PMQHCONN  pHconn,          /* IO: Connection handle */
   PPMQBO    ppBeginOptions,  /* IO: Options that control the action */
                              /* of MQBEGIN */
   PMQLONG   pCompCode,       /* OC: Completion code */
   PMQLONG   pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQ_BEGIN_EXIT MQPOINTER PMQ_BEGIN_EXIT;


 /****************************************************************/
 /* MQ_CALLBACK_EXIT -- Callback Function Exit                   */
 /****************************************************************/

 typedef void MQENTRY MQ_CALLBACK_EXIT (
   PMQAXP    pExitParms,     /* IO: Exit parameter structure */
   PMQAXC    pExitContext,   /* IO: Exit context structure */
   PMQHCONN  pHconn,         /* IO: Connection handle */
   PPMQMD    ppMsgDesc,      /* IO: Message descriptor */
   PPMQGMO   ppGetMsgOpts,   /* IO: Options that define the operation */
                             /* of the consumer */
   PPMQVOID  ppBuffer,       /* IO: Area to contain the message data */
   PPMQCBC   ppMQCBContext); /* IO: Context data for the callback */
 typedef MQ_CALLBACK_EXIT MQPOINTER PMQ_CALLBACK_EXIT;


 /****************************************************************/
 /* MQ_CB_EXIT -- Register Callback Exit                         */
 /****************************************************************/

 typedef void MQENTRY MQ_CB_EXIT (
   PMQAXP    pExitParms,      /* IO: Exit parameter structure */
   PMQAXC    pExitContext,    /* IO: Exit context structure */
   PMQHCONN  pHconn,          /* IO: Connection handle */
   PMQLONG   pOperation,      /* IO: Operation */
   PPMQCBD   ppCallbackDesc,  /* IO: Callback descriptor */
   PMQHOBJ   pHobj,           /* IO: Object handle */
   PPMQMD    ppMsgDesc,       /* IO: Message descriptor */
   PPMQGMO   ppGetMsgOpts,    /* IO: Get message options */
   PMQLONG   pCompCode,       /* OC: Completion code */
   PMQLONG   pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQ_CB_EXIT MQPOINTER PMQ_CB_EXIT;


 /****************************************************************/
 /* MQ_CLOSE_EXIT -- Close Object Exit                           */
 /****************************************************************/

 typedef void MQENTRY MQ_CLOSE_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQHOBJ  ppHobj,        /* IO: Object handle */
   PMQLONG   pOptions,      /* IO: Options that control the action of */
                            /* MQCLOSE */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CLOSE_EXIT MQPOINTER PMQ_CLOSE_EXIT;


 /****************************************************************/
 /* MQ_CMIT_EXIT -- Commit Changes Exit                          */
 /****************************************************************/

 typedef void MQENTRY MQ_CMIT_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CMIT_EXIT MQPOINTER PMQ_CMIT_EXIT;


 /****************************************************************/
 /* MQ_CONNX_EXIT -- Connect Queue Manager Exit                  */
 /****************************************************************/

 typedef void MQENTRY MQ_CONNX_EXIT (
   PMQAXP     pExitParms,     /* IO: Exit parameter structure */
   PMQAXC     pExitContext,   /* IO: Exit context structure */
   PMQCHAR    pQMgrName,      /* IO: Name of queue manager */
   PPMQCNO    ppConnectOpts,  /* IO: Options that control the action */
                              /* of MQCONNX */
   PPMQHCONN  ppHconn,        /* IO: Connection handle */
   PMQLONG    pCompCode,      /* OC: Completion code */
   PMQLONG    pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_CONNX_EXIT MQPOINTER PMQ_CONNX_EXIT;


 /****************************************************************/
 /* MQ_CTL_EXIT -- Control Asynchronous Operations Exit          */
 /****************************************************************/

 typedef void MQENTRY MQ_CTL_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQLONG   pOperation,    /* IO: Operation */
   PPMQCTLO  ppCtlOpts,     /* IO: Control options */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_CTL_EXIT MQPOINTER PMQ_CTL_EXIT;


 /****************************************************************/
 /* MQ_DISC_EXIT -- Disconnect Queue Manager Exit                */
 /****************************************************************/

 typedef void MQENTRY MQ_DISC_EXIT (
   PMQAXP     pExitParms,    /* IO: Exit parameter structure */
   PMQAXC     pExitContext,  /* IO: Exit context structure */
   PPMQHCONN  ppHconn,       /* IO: Connection handle */
   PMQLONG    pCompCode,     /* OC: Completion code */
   PMQLONG    pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_DISC_EXIT MQPOINTER PMQ_DISC_EXIT;


 /****************************************************************/
 /* MQ_GET_EXIT -- Get Message Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_GET_EXIT (
   PMQAXP    pExitParms,     /* IO: Exit parameter structure */
   PMQAXC    pExitContext,   /* IO: Exit context structure */
   PMQHCONN  pHconn,         /* IO: Connection handle */
   PMQHOBJ   pHobj,          /* IO: Object handle */
   PPMQMD    ppMsgDesc,      /* IO: Message descriptor */
   PPMQGMO   ppGetMsgOpts,   /* IO: Options that control the action */
                             /* of MQGET */
   PMQLONG   pBufferLength,  /* IO: Length in bytes of pBuffer area */
   PPMQVOID  ppBuffer,       /* IO: Area to contain the message data */
   PPMQLONG  ppDataLength,   /* OC: Length of the message */
   PMQLONG   pCompCode,      /* OC: Completion code */
   PMQLONG   pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_GET_EXIT MQPOINTER PMQ_GET_EXIT;


 /****************************************************************/
 /* MQ_INIT_EXIT -- Initialization Exit                          */
 /****************************************************************/

 typedef void MQENTRY MQ_INIT_EXIT (
   PMQAXP   pExitParms,    /* IO: Exit parameter structure */
   PMQAXC   pExitContext,  /* IO: Exit context structure */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_INIT_EXIT MQPOINTER PMQ_INIT_EXIT;


 /****************************************************************/
 /* MQ_INQ_EXIT -- Inquire Object Attributes Exit                */
 /****************************************************************/

 typedef void MQENTRY MQ_INQ_EXIT (
   PMQAXP    pExitParms,       /* IO: Exit parameter structure */
   PMQAXC    pExitContext,     /* IO: Exit context structure */
   PMQHCONN  pHconn,           /* IO: Connection handle */
   PMQHOBJ   pHobj,            /* IO: Object handle */
   PMQLONG   pSelectorCount,   /* IO: Count of selectors */
   PPMQLONG  ppSelectors,      /* IO: Array of attribute selectors */
   PMQLONG   pIntAttrCount,    /* IO: Count of integer attributes */
   PPMQLONG  ppIntAttrs,       /* IO: Array of integer attributes */
   PMQLONG   pCharAttrLength,  /* OC: Length of character attributes */
   PPMQCHAR  ppCharAttrs,      /* OC: Character attributes */
   PMQLONG   pCompCode,        /* OC: Completion code */
   PMQLONG   pReason);         /* OR: Reason code qualifying CompCode */
 typedef MQ_INQ_EXIT MQPOINTER PMQ_INQ_EXIT;


 /****************************************************************/
 /* MQ_OPEN_EXIT -- Open Object Exit                             */
 /****************************************************************/

 typedef void MQENTRY MQ_OPEN_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQOD    ppObjDesc,     /* IO: Object descriptor */
   PMQLONG   pOptions,      /* IO: Options that control the action of */
                            /* MQOPEN */
   PPMQHOBJ  ppHobj,        /* IO: Object handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_OPEN_EXIT MQPOINTER PMQ_OPEN_EXIT;


 /****************************************************************/
 /* MQ_PUT_EXIT -- Put Message Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_PUT_EXIT (
   PMQAXP    pExitParms,     /* IO: Exit parameter structure */
   PMQAXC    pExitContext,   /* IO: Exit context structure */
   PMQHCONN  pHconn,         /* IO: Connection handle */
   PMQHOBJ   pHobj,          /* IO: Object handle */
   PPMQMD    ppMsgDesc,      /* IO: Message descriptor */
   PPMQPMO   ppPutMsgOpts,   /* IO: Options that control the action */
                             /* of MQPUT */
   PMQLONG   pBufferLength,  /* IO: Length of the message in pBuffer */
   PPMQVOID  ppBuffer,       /* IO: Message data */
   PMQLONG   pCompCode,      /* OC: Completion code */
   PMQLONG   pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_PUT_EXIT MQPOINTER PMQ_PUT_EXIT;


 /****************************************************************/
 /* MQ_PUT1_EXIT -- Put One Message Exit                         */
 /****************************************************************/

 typedef void MQENTRY MQ_PUT1_EXIT (
   PMQAXP    pExitParms,     /* IO: Exit parameter structure */
   PMQAXC    pExitContext,   /* IO: Exit context structure */
   PMQHCONN  pHconn,         /* IO: Connection handle */
   PPMQOD    ppObjDesc,      /* IO: Object descriptor */
   PPMQMD    ppMsgDesc,      /* IO: Message descriptor */
   PPMQPMO   ppPutMsgOpts,   /* IO: Options that control the action */
                             /* of MQPUT1 */
   PMQLONG   pBufferLength,  /* IO: Length of the message in pBuffer */
   PPMQVOID  ppBuffer,       /* IO: Message data */
   PMQLONG   pCompCode,      /* OC: Completion code */
   PMQLONG   pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_PUT1_EXIT MQPOINTER PMQ_PUT1_EXIT;


 /****************************************************************/
 /* MQ_SET_EXIT -- Set Object Attributes Exit                    */
 /****************************************************************/

 typedef void MQENTRY MQ_SET_EXIT (
   PMQAXP    pExitParms,       /* IO: Exit parameter structure */
   PMQAXC    pExitContext,     /* IO: Exit context structure */
   PMQHCONN  pHconn,           /* IO: Connection handle */
   PMQHOBJ   pHobj,            /* IO: Object handle */
   PMQLONG   pSelectorCount,   /* IO: Count of selectors */
   PPMQLONG  ppSelectors,      /* IO: Array of attribute selectors */
   PMQLONG   pIntAttrCount,    /* IO: Count of integer attributes */
   PPMQLONG  ppIntAttrs,       /* IO: Array of integer attributes */
   PMQLONG   pCharAttrLength,  /* OC: Length of character attributes */
                               /* buffer */
   PPMQCHAR  ppCharAttrs,      /* OC: Character attributes */
   PMQLONG   pCompCode,        /* OC: Completion code */
   PMQLONG   pReason);         /* OR: Reason code qualifying CompCode */
 typedef MQ_SET_EXIT MQPOINTER PMQ_SET_EXIT;


 /****************************************************************/
 /* MQ_STAT_EXIT -- Get Status Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_STAT_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQLONG   pType,         /* IO: Status Type */
   PPMQSTS   ppStatus,      /* IO: Status Buffer */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_STAT_EXIT MQPOINTER PMQ_STAT_EXIT;


 /****************************************************************/
 /* MQ_SUBRQ_EXIT -- Subscribe Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_SUBRQ_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQHOBJ   pHsub,         /* IO: Subscription handle */
   PMQLONG   pAction,       /* IO: Request action */
   PPMQSRO   ppSubRqOpts,   /* IO: Subscription Request options */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_SUBRQ_EXIT MQPOINTER PMQ_SUBRQ_EXIT;


 /****************************************************************/
 /* MQ_SUB_EXIT -- Subscribe Exit                                */
 /****************************************************************/

 typedef void MQENTRY MQ_SUB_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQSD    ppSubDesc,     /* IO: Subscription descriptor */
   PPMQHOBJ  ppHobj,        /* IO: Queue object handle */
   PPMQHOBJ  ppHsub,        /* IO: Subscription object handle */
   PMQLONG   pCompCode,     /* OC: Completion code */
   PMQLONG   pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_SUB_EXIT MQPOINTER PMQ_SUB_EXIT;


 /****************************************************************/
 /* MQ_TERM_EXIT -- Termination Exit                             */
 /****************************************************************/

 typedef void MQENTRY MQ_TERM_EXIT (
   PMQAXP   pExitParms,    /* IO: Exit parameter structure */
   PMQAXC   pExitContext,  /* IO: Exit context structure */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */
 typedef MQ_TERM_EXIT MQPOINTER PMQ_TERM_EXIT;


 /****************************************************************/
 /* XA_CLOSE_EXIT -- xa_close Exit                               */
 /****************************************************************/

 typedef void MQENTRY XA_CLOSE_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQCHAR  ppXa_info,     /* IO: Instance-specific RM info */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_CLOSE_EXIT MQPOINTER PXA_CLOSE_EXIT;


 /****************************************************************/
 /* XA_COMMIT_EXIT -- xa_commit Exit                             */
 /****************************************************************/

 typedef void MQENTRY XA_COMMIT_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_COMMIT_EXIT MQPOINTER PXA_COMMIT_EXIT;


 /****************************************************************/
 /* XA_COMPLETE_EXIT -- xa_complete Exit                         */
 /****************************************************************/

 typedef void MQENTRY XA_COMPLETE_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQLONG  ppHandle,      /* IO: Ptr to asynchronous op */
   PPMQLONG  ppRetval,      /* IO: Return value of async op */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_COMPLETE_EXIT MQPOINTER PXA_COMPLETE_EXIT;


 /****************************************************************/
 /* XA_END_EXIT -- xa_end Exit                                   */
 /****************************************************************/

 typedef void MQENTRY XA_END_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_END_EXIT MQPOINTER PXA_END_EXIT;


 /****************************************************************/
 /* XA_FORGET_EXIT -- xa_forget Exit                             */
 /****************************************************************/

 typedef void MQENTRY XA_FORGET_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_FORGET_EXIT MQPOINTER PXA_FORGET_EXIT;


 /****************************************************************/
 /* XA_OPEN_EXIT -- xa_open Exit                                 */
 /****************************************************************/

 typedef void MQENTRY XA_OPEN_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PPMQCHAR  ppXa_info,     /* IO: Instance-specific RM info */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_OPEN_EXIT MQPOINTER PXA_OPEN_EXIT;


 /****************************************************************/
 /* XA_PREPARE_EXIT -- xa_prepare Exit                           */
 /****************************************************************/

 typedef void MQENTRY XA_PREPARE_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_PREPARE_EXIT MQPOINTER PXA_PREPARE_EXIT;


 /****************************************************************/
 /* XA_RECOVER_EXIT -- xa_recover Exit                           */
 /****************************************************************/

 typedef void MQENTRY XA_RECOVER_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pCount,        /* IO: Max XIDs in XID array */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_RECOVER_EXIT MQPOINTER PXA_RECOVER_EXIT;


 /****************************************************************/
 /* XA_ROLLBACK_EXIT -- xa_rollback Exit                         */
 /****************************************************************/

 typedef void MQENTRY XA_ROLLBACK_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_ROLLBACK_EXIT MQPOINTER PXA_ROLLBACK_EXIT;


 /****************************************************************/
 /* XA_START_EXIT -- xa_start Exit                               */
 /****************************************************************/

 typedef void MQENTRY XA_START_EXIT (
   PMQAXP    pExitParms,    /* IO: Exit parameter structure */
   PMQAXC    pExitContext,  /* IO: Exit context structure */
   PMQHCONN  pHconn,        /* IO: Connection handle */
   PMQPTR    ppXID,         /* IO: Transaction branch ID */
   PMQLONG   pRmid,         /* IO: Resource manager identifier */
   PMQLONG   pFlags,        /* IO: Resource manager options */
   PMQLONG   pXARetCode);   /* OR: Response from XA call */
 typedef XA_START_EXIT MQPOINTER PXA_START_EXIT;


 /****************************************************************/
 /* AX_REG_EXIT -- ax_reg Exit                                   */
 /****************************************************************/

 typedef void MQENTRY AX_REG_EXIT (
   PMQAXP   pExitParms,    /* IO: Exit parameter structure */
   PMQAXC   pExitContext,  /* IO: Exit context structure */
   PMQPTR   ppXID,         /* IO: Transaction branch ID */
   PMQLONG  pRmid,         /* IO: Resource manager identifier */
   PMQLONG  pFlags,        /* IO: Resource manager options */
   PMQLONG  pXARetCode);   /* OR: Response from XA call */
 typedef AX_REG_EXIT MQPOINTER PAX_REG_EXIT;


 /****************************************************************/
 /* AX_UNREG_EXIT -- ax_unreg Exit                               */
 /****************************************************************/

 typedef void MQENTRY AX_UNREG_EXIT (
   PMQAXP   pExitParms,    /* IO: Exit parameter structure */
   PMQAXC   pExitContext,  /* IO: Exit context structure */
   PMQLONG  pRmid,         /* IO: Resource manager identifier */
   PMQLONG  pFlags,        /* IO: Resource manager options */
   PMQLONG  pXARetCode);   /* OR: Response from XA call */
 typedef AX_UNREG_EXIT MQPOINTER PAX_UNREG_EXIT;


 /****************************************************************/
 /* Other Exit Functions                                         */
 /****************************************************************/

 /****************************************************************/
 /* MQ_CHANNEL_EXIT -- Channel Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_CHANNEL_EXIT (
   PMQVOID  pChannelExitParms,   /* IO: Channel exit parameter block */
   PMQVOID  pChannelDefinition,  /* IO: Channel definition */
   PMQLONG  pDataLength,         /* IO: Length of data */
   PMQLONG  pAgentBufferLength,  /* IL: Length of agent buffer */
   PMQVOID  pAgentBuffer,        /* IOB: Agent buffer */
   PMQLONG  pExitBufferLength,   /* IOL: Length of exit buffer */
   PMQPTR   pExitBufferAddr);    /* IOB: Address of exit buffer */
 typedef MQ_CHANNEL_EXIT MQPOINTER PMQ_CHANNEL_EXIT;


 /****************************************************************/
 /* MQ_CHANNEL_AUTO_DEF_EXIT -- Channel Auto Definition Exit     */
 /****************************************************************/

 typedef void MQENTRY MQ_CHANNEL_AUTO_DEF_EXIT (
   PMQVOID  pChannelExitParms,   /* IO: Channel exit parameter block */
   PMQVOID  pChannelDefinition); /* IO: Channel definition */
 typedef MQ_CHANNEL_AUTO_DEF_EXIT MQPOINTER PMQ_CHANNEL_AUTO_DEF_EXIT;


 /****************************************************************/
 /* MQ_CLUSTER_WORKLOAD_EXIT -- Cluster Workload Exit            */
 /****************************************************************/

 typedef void MQENTRY MQ_CLUSTER_WORKLOAD_EXIT (
   PMQWXP  pExitParms); /* IO: Exit parameter block */
 typedef MQ_CLUSTER_WORKLOAD_EXIT MQPOINTER PMQ_CLUSTER_WORKLOAD_EXIT;


 /****************************************************************/
 /* MQ_DATA_CONV_EXIT -- Data Conversion Exit                    */
 /****************************************************************/

 typedef void MQENTRY MQ_DATA_CONV_EXIT (
   PMQDXP   pDataConvExitParms,  /* IO: Data-conversion exit */
                                 /* parameter block */
   PMQMD    pMsgDesc,            /* IO: Message descriptor */
   MQLONG   InBufferLength,      /* IL: Length in bytes of InBuffer */
   PMQVOID  pInBuffer,           /* IB: Buffer containing the */
                                 /* unconverted message */
   MQLONG   OutBufferLength,     /* IL: Length in bytes of OutBuffer */
   PMQVOID  pOutBuffer);         /* OB: Buffer containing the */
                                 /* converted message */
 typedef MQ_DATA_CONV_EXIT MQPOINTER PMQ_DATA_CONV_EXIT;


 /****************************************************************/
 /* MQ_PUBLISH_EXIT -- Publish Exit                              */
 /****************************************************************/

 typedef void MQENTRY MQ_PUBLISH_EXIT (
   PMQPSXP  pExitParms,   /* IO: Exit parameter block */
   PMQPBC   pPubContext,  /* I: Publication context structure */
   PMQSBC   pSubContext); /* I: Subscription context structure */
 typedef MQ_PUBLISH_EXIT MQPOINTER PMQ_PUBLISH_EXIT;


 /****************************************************************/
 /* MQ_TRANSPORT_EXIT -- Transport Retry Exit                    */
 /****************************************************************/

 typedef void MQENTRY MQ_TRANSPORT_EXIT (
   PMQVOID  pExitParms,         /* IO: Exit parameter block */
   MQLONG   DestAddressLength,  /* IL: Length in bytes of destination */
                                /* IP address */
   PMQCHAR  pDestAddress);      /* IB: Destination IP address */
 typedef MQ_TRANSPORT_EXIT MQPOINTER PMQ_TRANSPORT_EXIT;


 /****************************************************************/
 /* MQ_PRECONNECT_EXIT Function -- Preconnect Exit               */
 /****************************************************************/

 typedef void MQENTRY MQ_PRECONNECT_EXIT (
   PMQNXP   pExitParms,     /* IO: Exit parameter structure */
   PMQCHAR  pQMgrName,      /* IO: Name of queue manager */
   PPMQCNO  ppConnectOpts,  /* IO: Options that control the action of */
                            /* MQCONNX */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_PRECONNECT_EXIT MQPOINTER PMQ_PRECONNECT_EXIT;


 /****************************************************************/
 /* MQXCLWLN Function -- Cluster Workload Navigate Records       */
 /****************************************************************/

 void MQENTRY MQXCLWLN (
   PMQWXP   pExitParms,     /* IO: Exit parameter structure */
   MQPTR    CurrentRecord,  /* I: Address of current record */
   MQLONG   NextOffset,     /* I: Offset of next record */
   PMQPTR   pNextRecord,    /* O: Address of next record or structure */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_XCLWLN_CALL (
   PMQWXP   pExitParms,     /* IO: Exit parameter structure */
   MQPTR    CurrentRecord,  /* I: Address of current record */
   MQLONG   NextOffset,     /* I: Offset of next record */
   PMQPTR   pNextRecord,    /* O: Address of next record or structure */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_XCLWLN_CALL MQPOINTER PMQ_XCLWLN_CALL;


 /****************************************************************/
 /* MQXCNVC Function -- Convert Characters                       */
 /****************************************************************/

 void MQENTRY MQXCNVC (
   MQHCONN  Hconn,          /* I: Connection handle */
   MQLONG   Options,        /* I: Options that control the action of */
                            /* MQXCNVC */
   MQLONG   SourceCCSID,    /* I: Coded character set identifier of */
                            /* string before conversion */
   MQLONG   SourceLength,   /* IL: Length of string before conversion */
   PMQCHAR  pSourceBuffer,  /* IB: String to be converted */
   MQLONG   TargetCCSID,    /* I: Coded character set identifier of */
                            /* string after conversion */
   MQLONG   TargetLength,   /* IL: Length of output buffer */
   PMQCHAR  pTargetBuffer,  /* OB: String after conversion */
   PMQLONG  pDataLength,    /* O: Length of output string */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_XCNVC_CALL (
   MQHCONN  Hconn,          /* I: Connection handle */
   MQLONG   Options,        /* I: Options that control the action of */
                            /* MQXCNVC */
   MQLONG   SourceCCSID,    /* I: Coded character set identifier of */
                            /* string before conversion */
   MQLONG   SourceLength,   /* IL: Length of string before conversion */
   PMQCHAR  pSourceBuffer,  /* IB: String to be converted */
   MQLONG   TargetCCSID,    /* I: Coded character set identifier of */
                            /* string after conversion */
   MQLONG   TargetLength,   /* IL: Length of output buffer */
   PMQCHAR  pTargetBuffer,  /* OB: String after conversion */
   PMQLONG  pDataLength,    /* O: Length of output string */
   PMQLONG  pCompCode,      /* OC: Completion code */
   PMQLONG  pReason);       /* OR: Reason code qualifying CompCode */
 typedef MQ_XCNVC_CALL MQPOINTER PMQ_XCNVC_CALL;


 /****************************************************************/
 /* MQXDX Function -- Convert Message Data                       */
 /****************************************************************/

 void MQENTRY MQXDX (
   PMQDXP   pDataConvExitParms,  /* IO: Data-conversion exit */
                                 /* parameter block */
   PMQMD    pMsgDesc,            /* IO: Message descriptor */
   MQLONG   InBufferLength,      /* IL: Length in bytes of InBuffer */
   PMQVOID  pInBuffer,           /* IB: Buffer containing the */
                                 /* unconverted message */
   MQLONG   OutBufferLength,     /* IL: Length in bytes of OutBuffer */
   PMQVOID  pOutBuffer);         /* OB: Buffer containing the */
                                 /* converted message */

 typedef void MQENTRY MQ_XDX_CALL (
   PMQDXP   pDataConvExitParms,  /* IO: Data-conversion exit */
                                 /* parameter block */
   PMQMD    pMsgDesc,            /* IO: Message descriptor */
   MQLONG   InBufferLength,      /* IL: Length in bytes of InBuffer */
   PMQVOID  pInBuffer,           /* IB: Buffer containing the */
                                 /* unconverted message */
   MQLONG   OutBufferLength,     /* IL: Length in bytes of OutBuffer */
   PMQVOID  pOutBuffer);         /* OB: Buffer containing the */
                                 /* converted message */
 typedef MQ_XDX_CALL MQPOINTER PMQ_XDX_CALL;



 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQXC                                                */
 /****************************************************************/
 #endif  /* End of header file */
