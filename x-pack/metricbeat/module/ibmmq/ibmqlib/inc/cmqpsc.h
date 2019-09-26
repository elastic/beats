 #if !defined(MQPSC_INCLUDED)          /* File not yet included? */
   #define MQPSC_INCLUDED              /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQPSC                                      */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for Publish/Subscribe          */
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
 /*  FUNCTION:       This file declares the named constants      */
 /*                  for publish/subscribe.                      */
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
 /* pn=com.ibm.mq.famfiles.data/xml/approved/cmqpsc.xml          */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Definitions used by MQRFH(1) - Rules and Formatting Header   */
 /****************************************************************/

 /****************************************************************/
 /* Publish/Subscribe Tags                                       */
 /****************************************************************/

 /* Tags as strings */
 #define MQPS_COMMAND                   "MQPSCommand"
 #define MQPS_COMP_CODE                 "MQPSCompCode"
 #define MQPS_CORREL_ID                 "MQPSCorrelId"
 #define MQPS_DELETE_OPTIONS            "MQPSDelOpts"
 #define MQPS_ERROR_ID                  "MQPSErrorId"
 #define MQPS_ERROR_POS                 "MQPSErrorPos"
 #define MQPS_INTEGER_DATA              "MQPSIntData"
 #define MQPS_PARAMETER_ID              "MQPSParmId"
 #define MQPS_PUBLICATION_OPTIONS       "MQPSPubOpts"
 #define MQPS_PUBLISH_TIMESTAMP         "MQPSPubTime"
 #define MQPS_Q_MGR_NAME                "MQPSQMgrName"
 #define MQPS_Q_NAME                    "MQPSQName"
 #define MQPS_REASON                    "MQPSReason"
 #define MQPS_REASON_TEXT               "MQPSReasonText"
 #define MQPS_REGISTRATION_OPTIONS      "MQPSRegOpts"
 #define MQPS_SEQUENCE_NUMBER           "MQPSSeqNum"
 #define MQPS_STREAM_NAME               "MQPSStreamName"
 #define MQPS_STRING_DATA               "MQPSStringData"
 #define MQPS_SUBSCRIPTION_IDENTITY     "MQPSSubIdentity"
 #define MQPS_SUBSCRIPTION_NAME         "MQPSSubName"
 #define MQPS_SUBSCRIPTION_USER_DATA    "MQPSSubUserData"
 #define MQPS_TOPIC                     "MQPSTopic"
 #define MQPS_USER_ID                   "MQPSUserId"

 /* Tags as blank-enclosed strings */
 #define MQPS_COMMAND_B                 " MQPSCommand "
 #define MQPS_COMP_CODE_B               " MQPSCompCode "
 #define MQPS_CORREL_ID_B               " MQPSCorrelId "
 #define MQPS_DELETE_OPTIONS_B          " MQPSDelOpts "
 #define MQPS_ERROR_ID_B                " MQPSErrorId "
 #define MQPS_ERROR_POS_B               " MQPSErrorPos "
 #define MQPS_INTEGER_DATA_B            " MQPSIntData "
 #define MQPS_PARAMETER_ID_B            " MQPSParmId "
 #define MQPS_PUBLICATION_OPTIONS_B     " MQPSPubOpts "
 #define MQPS_PUBLISH_TIMESTAMP_B       " MQPSPubTime "
 #define MQPS_Q_MGR_NAME_B              " MQPSQMgrName "
 #define MQPS_Q_NAME_B                  " MQPSQName "
 #define MQPS_REASON_B                  " MQPSReason "
 #define MQPS_REASON_TEXT_B             " MQPSReasonText "
 #define MQPS_REGISTRATION_OPTIONS_B    " MQPSRegOpts "
 #define MQPS_SEQUENCE_NUMBER_B         " MQPSSeqNum "
 #define MQPS_STREAM_NAME_B             " MQPSStreamName "
 #define MQPS_STRING_DATA_B             " MQPSStringData "
 #define MQPS_SUBSCRIPTION_IDENTITY_B   " MQPSSubIdentity "
 #define MQPS_SUBSCRIPTION_NAME_B       " MQPSSubName "
 #define MQPS_SUBSCRIPTION_USER_DATA_B  " MQPSSubUserData "
 #define MQPS_TOPIC_B                   " MQPSTopic "
 #define MQPS_USER_ID_B                 " MQPSUserId "

 /* Tags as blank-enclosed arrays */
 #define MQPS_COMMAND_A                 ' ','M','Q','P','S','C','o','m',\
                                        'm','a','n','d',' '
 #define MQPS_COMP_CODE_A               ' ','M','Q','P','S','C','o','m',\
                                        'p','C','o','d','e',' '
 #define MQPS_CORREL_ID_A               ' ','M','Q','P','S','C','o','r',\
                                        'r','e','l','I','d',' '
 #define MQPS_DELETE_OPTIONS_A          ' ','M','Q','P','S','D','e','l',\
                                        'O','p','t','s',' '
 #define MQPS_ERROR_ID_A                ' ','M','Q','P','S','E','r','r',\
                                        'o','r','I','d',' '
 #define MQPS_ERROR_POS_A               ' ','M','Q','P','S','E','r','r',\
                                        'o','r','P','o','s',' '
 #define MQPS_INTEGER_DATA_A            ' ','M','Q','P','S','I','n','t',\
                                        'D','a','t','a',' '
 #define MQPS_PARAMETER_ID_A            ' ','M','Q','P','S','P','a','r',\
                                        'm','I','d',' '
 #define MQPS_PUBLICATION_OPTIONS_A     ' ','M','Q','P','S','P','u','b',\
                                        'O','p','t','s',' '
 #define MQPS_PUBLISH_TIMESTAMP_A       ' ','M','Q','P','S','P','u','b',\
                                        'T','i','m','e',' '
 #define MQPS_Q_MGR_NAME_A              ' ','M','Q','P','S','Q','M','g',\
                                        'r','N','a','m','e',' '
 #define MQPS_Q_NAME_A                  ' ','M','Q','P','S','Q','N','a',\
                                        'm','e',' '
 #define MQPS_REASON_A                  ' ','M','Q','P','S','R','e','a',\
                                        's','o','n',' '
 #define MQPS_REASON_TEXT_A             ' ','M','Q','P','S','R','e','a',\
                                        's','o','n','T','e','x','t',' '
 #define MQPS_REGISTRATION_OPTIONS_A    ' ','M','Q','P','S','R','e','g',\
                                        'O','p','t','s',' '
 #define MQPS_SEQUENCE_NUMBER_A         ' ','M','Q','P','S','S','e','q',\
                                        'N','u','m',' '
 #define MQPS_STREAM_NAME_A             ' ','M','Q','P','S','S','t','r',\
                                        'e','a','m','N','a','m','e',' '
 #define MQPS_STRING_DATA_A             ' ','M','Q','P','S','S','t','r',\
                                        'i','n','g','D','a','t','a',' '
 #define MQPS_SUBSCRIPTION_IDENTITY_A   ' ','M','Q','P','S','S','u','b',\
                                        'I','d','e','n','t','i','t','y',\
                                        ' '
 #define MQPS_SUBSCRIPTION_NAME_A       ' ','M','Q','P','S','S','u','b',\
                                        'N','a','m','e',' '
 #define MQPS_SUBSCRIPTION_USER_DATA_A  ' ','M','Q','P','S','S','u','b',\
                                        'U','s','e','r','D','a','t','a',\
                                        ' '
 #define MQPS_TOPIC_A                   ' ','M','Q','P','S','T','o','p',\
                                        'i','c',' '
 #define MQPS_USER_ID_A                 ' ','M','Q','P','S','U','s','e',\
                                        'r','I','d',' '

 /****************************************************************/
 /* Values for MQPS_COMMAND Tag                                  */
 /****************************************************************/

 /* Values as strings */
 #define MQPS_DELETE_PUBLICATION        "DeletePub"
 #define MQPS_DEREGISTER_PUBLISHER      "DeregPub"
 #define MQPS_DEREGISTER_SUBSCRIBER     "DeregSub"
 #define MQPS_PUBLISH                   "Publish"
 #define MQPS_REGISTER_PUBLISHER        "RegPub"
 #define MQPS_REGISTER_SUBSCRIBER       "RegSub"
 #define MQPS_REQUEST_UPDATE            "ReqUpdate"

 /* Values as blank-enclosed strings */
 #define MQPS_DELETE_PUBLICATION_B      " DeletePub "
 #define MQPS_DEREGISTER_PUBLISHER_B    " DeregPub "
 #define MQPS_DEREGISTER_SUBSCRIBER_B   " DeregSub "
 #define MQPS_PUBLISH_B                 " Publish "
 #define MQPS_REGISTER_PUBLISHER_B      " RegPub "
 #define MQPS_REGISTER_SUBSCRIBER_B     " RegSub "
 #define MQPS_REQUEST_UPDATE_B          " ReqUpdate "

 /* Values as blank-enclosed arrays */
 #define MQPS_DELETE_PUBLICATION_A      ' ','D','e','l','e','t','e','P',\
                                        'u','b',' '
 #define MQPS_DEREGISTER_PUBLISHER_A    ' ','D','e','r','e','g','P','u',\
                                        'b',' '
 #define MQPS_DEREGISTER_SUBSCRIBER_A   ' ','D','e','r','e','g','S','u',\
                                        'b',' '
 #define MQPS_PUBLISH_A                 ' ','P','u','b','l','i','s','h',\
                                        ' '
 #define MQPS_REGISTER_PUBLISHER_A      ' ','R','e','g','P','u','b',' '
 #define MQPS_REGISTER_SUBSCRIBER_A     ' ','R','e','g','S','u','b',' '
 #define MQPS_REQUEST_UPDATE_A          ' ','R','e','q','U','p','d','a',\
                                        't','e',' '

 /****************************************************************/
 /* Values for following tags:                                   */
 /*   MQPS_DELETE_OPTIONS                                        */
 /*   MQPS_PUBLICATION_OPTIONS                                   */
 /*   MQPS_REGISTRATION_OPTIONS                                  */
 /****************************************************************/

 /* Values as strings */
 #define MQPS_ADD_NAME                  "AddName"
 #define MQPS_ANONYMOUS                 "Anon"
 #define MQPS_CORREL_ID_AS_IDENTITY     "CorrelAsId"
 #define MQPS_DEREGISTER_ALL            "DeregAll"
 #define MQPS_DIRECT_REQUESTS           "DirectReq"
 #define MQPS_DUPLICATES_OK             "DupsOK"
 #define MQPS_FULL_RESPONSE             "FullResp"
 #define MQPS_INCLUDE_STREAM_NAME       "InclStreamName"
 #define MQPS_INFORM_IF_RETAINED        "InformIfRet"
 #define MQPS_IS_RETAINED_PUBLICATION   "IsRetainedPub"
 #define MQPS_JOIN_EXCLUSIVE            "JoinExcl"
 #define MQPS_JOIN_SHARED               "JoinShared"
 #define MQPS_LEAVE_ONLY                "LeaveOnly"
 #define MQPS_LOCAL                     "Local"
 #define MQPS_LOCKED                    "Locked"
 #define MQPS_NEW_PUBLICATIONS_ONLY     "NewPubsOnly"
 #define MQPS_NO_ALTERATION             "NoAlter"
 #define MQPS_NO_REGISTRATION           "NoReg"
 #define MQPS_NON_PERSISTENT            "NonPers"
 #define MQPS_NONE                      "None"
 #define MQPS_OTHER_SUBSCRIBERS_ONLY    "OtherSubsOnly"
 #define MQPS_PERSISTENT                "Pers"
 #define MQPS_PERSISTENT_AS_PUBLISH     "PersAsPub"
 #define MQPS_PERSISTENT_AS_Q           "PersAsQueue"
 #define MQPS_PUBLISH_ON_REQUEST_ONLY   "PubOnReqOnly"
 #define MQPS_RETAIN_PUBLICATION        "RetainPub"
 #define MQPS_VARIABLE_USER_ID          "VariableUserId"

 /* Values as blank-enclosed strings */
 #define MQPS_ADD_NAME_B                " AddName "
 #define MQPS_ANONYMOUS_B               " Anon "
 #define MQPS_CORREL_ID_AS_IDENTITY_B   " CorrelAsId "
 #define MQPS_DEREGISTER_ALL_B          " DeregAll "
 #define MQPS_DIRECT_REQUESTS_B         " DirectReq "
 #define MQPS_DUPLICATES_OK_B           " DupsOK "
 #define MQPS_FULL_RESPONSE_B           " FullResp "
 #define MQPS_INCLUDE_STREAM_NAME_B     " InclStreamName "
 #define MQPS_INFORM_IF_RETAINED_B      " InformIfRet "
 #define MQPS_IS_RETAINED_PUBLICATION_B " IsRetainedPub "
 #define MQPS_JOIN_EXCLUSIVE_B          " JoinExcl "
 #define MQPS_JOIN_SHARED_B             " JoinShared "
 #define MQPS_LEAVE_ONLY_B              " LeaveOnly "
 #define MQPS_LOCAL_B                   " Local "
 #define MQPS_LOCKED_B                  " Locked "
 #define MQPS_NEW_PUBLICATIONS_ONLY_B   " NewPubsOnly "
 #define MQPS_NO_ALTERATION_B           " NoAlter "
 #define MQPS_NO_REGISTRATION_B         " NoReg "
 #define MQPS_NON_PERSISTENT_B          " NonPers "
 #define MQPS_NONE_B                    " None "
 #define MQPS_OTHER_SUBSCRIBERS_ONLY_B  " OtherSubsOnly "
 #define MQPS_PERSISTENT_B              " Pers "
 #define MQPS_PERSISTENT_AS_PUBLISH_B   " PersAsPub "
 #define MQPS_PERSISTENT_AS_Q_B         " PersAsQueue "
 #define MQPS_PUBLISH_ON_REQUEST_ONLY_B " PubOnReqOnly "
 #define MQPS_RETAIN_PUBLICATION_B      " RetainPub "
 #define MQPS_VARIABLE_USER_ID_B        " VariableUserId "

 /* Values as blank-enclosed arrays */
 #define MQPS_ADD_NAME_A                ' ','A','d','d','N','a','m','e',\
                                        ' '
 #define MQPS_ANONYMOUS_A               ' ','A','n','o','n',' '
 #define MQPS_CORREL_ID_AS_IDENTITY_A   ' ','C','o','r','r','e','l','A',\
                                        's','I','d',' '
 #define MQPS_DEREGISTER_ALL_A          ' ','D','e','r','e','g','A','l',\
                                        'l',' '
 #define MQPS_DIRECT_REQUESTS_A         ' ','D','i','r','e','c','t','R',\
                                        'e','q',' '
 #define MQPS_DUPLICATES_OK_A           ' ','D','u','p','s','O','K',' '
 #define MQPS_FULL_RESPONSE_A           ' ','F','u','l','l','R','e','s',\
                                        'p',' '
 #define MQPS_INCLUDE_STREAM_NAME_A     ' ','I','n','c','l','S','t','r',\
                                        'e','a','m','N','a','m','e',' '
 #define MQPS_INFORM_IF_RETAINED_A      ' ','I','n','f','o','r','m','I',\
                                        'f','R','e','t',' '
 #define MQPS_IS_RETAINED_PUBLICATION_A ' ','I','s','R','e','t','a','i',\
                                        'n','e','d','P','u','b',' '
 #define MQPS_JOIN_EXCLUSIVE_A          ' ','J','o','i','n','E','x','c',\
                                        'l',' '
 #define MQPS_JOIN_SHARED_A             ' ','J','o','i','n','S','h','a',\
                                        'r','e','d',' '
 #define MQPS_LEAVE_ONLY_A              ' ','L','e','a','v','e','O','n',\
                                        'l','y',' '
 #define MQPS_LOCAL_A                   ' ','L','o','c','a','l',' '
 #define MQPS_LOCKED_A                  ' ','L','o','c','k','e','d',' '
 #define MQPS_NEW_PUBLICATIONS_ONLY_A   ' ','N','e','w','P','u','b','s',\
                                        'O','n','l','y',' '
 #define MQPS_NO_ALTERATION_A           ' ','N','o','A','l','t','e','r',\
                                        ' '
 #define MQPS_NO_REGISTRATION_A         ' ','N','o','R','e','g',' '
 #define MQPS_NONE_A                    ' ','N','o','n','e',' '
 #define MQPS_NON_PERSISTENT_A          ' ','N','o','n','P','e','r','s',\
                                        ' '
 #define MQPS_OTHER_SUBSCRIBERS_ONLY_A  ' ','O','t','h','e','r','S','u',\
                                        'b','s','O','n','l','y',' '
 #define MQPS_PERSISTENT_A              ' ','P','e','r','s',' '
 #define MQPS_PERSISTENT_AS_PUBLISH_A   ' ','P','e','r','s','A','s','P',\
                                        'u','b',' '
 #define MQPS_PERSISTENT_AS_Q_A         ' ','P','e','r','s','A','s','Q',\
                                        'u','e','u','e',' '
 #define MQPS_PUBLISH_ON_REQUEST_ONLY_A ' ','P','u','b','O','n','R','e',\
                                        'q','O','n','l','y',' '
 #define MQPS_RETAIN_PUBLICATION_A      ' ','R','e','t','a','i','n','P',\
                                        'u','b',' '
 #define MQPS_VARIABLE_USER_ID_A        ' ','V','a','r','i','a','b','l',\
                                        'e','U','s','e','r','I','d',' '

 /****************************************************************/
 /* Definitions used by MQRFH2 - Rules and Formatting Header 2   */
 /****************************************************************/

 /****************************************************************/
 /* RFH2 Top-level folder Tags                                   */
 /****************************************************************/

 #if !defined(MQRFH2_NAME_VALUE_VERSION)
 #define MQRFH2_NAME_VALUE_VERSION 1

 /* Tag names */
 #define MQRFH2_PUBSUB_CMD_FOLDER       "psc"
 #define MQRFH2_PUBSUB_RESP_FOLDER      "pscr"
 #define MQRFH2_MSG_CONTENT_FOLDER      "mcd"
 #define MQRFH2_USER_FOLDER             "usr"

 /* Tag names as character arrays */
 #define MQRFH2_PUBSUB_CMD_FOLDER_A     'p','s','c'
 #define MQRFH2_PUBSUB_RESP_FOLDER_A    'p','s','c','r'
 #define MQRFH2_MSG_CONTENT_FOLDER_A    'm','c','d'
 #define MQRFH2_USER_FOLDER_A           'u','s','r'

 /* XML tag names */
 #define MQRFH2_PUBSUB_CMD_FOLDER_B     "<psc>"
 #define MQRFH2_PUBSUB_CMD_FOLDER_E     "</psc>"
 #define MQRFH2_PUBSUB_RESP_FOLDER_B    "<pscr>"
 #define MQRFH2_PUBSUB_RESP_FOLDER_E    "</pscr>"
 #define MQRFH2_MSG_CONTENT_FOLDER_B    "<mcd>"
 #define MQRFH2_MSG_CONTENT_FOLDER_E    "</mcd>"
 #define MQRFH2_USER_FOLDER_B           "<usr>"
 #define MQRFH2_USER_FOLDER_E           "</usr>"

 /* XML tag names as character arrays */
 #define MQRFH2_PUBSUB_CMD_FOLDER_BA    '<','p','s','c','>'
 #define MQRFH2_PUBSUB_CMD_FOLDER_EA    '<','/','p','s','c','>'
 #define MQRFH2_PUBSUB_RESP_FOLDER_BA   '<','p','s','c','r','>'
 #define MQRFH2_PUBSUB_RESP_FOLDER_EA   '<','/','p','s','c','r','>'
 #define MQRFH2_MSG_CONTENT_FOLDER_BA   '<','m','c','d','>'
 #define MQRFH2_MSG_CONTENT_FOLDER_EA   '<','/','m','c','d','>'
 #define MQRFH2_USER_FOLDER_BA          '<','u','s','r','>'
 #define MQRFH2_USER_FOLDER_EA          '<','/','u','s','r','>'

 #endif /* MQRFH2_NAME_VALUE_VERSION */

 /****************************************************************/
 /* Message Content Descriptor (mcd) Tags                        */
 /****************************************************************/

 #if !defined(MQMCD_FOLDER_VERSION)
 #define MQMCD_FOLDER_VERSION 1

 /* Tag names */
 #define MQMCD_MSG_DOMAIN               "Msd"
 #define MQMCD_MSG_SET                  "Set"
 #define MQMCD_MSG_TYPE                 "Type"
 #define MQMCD_MSG_FORMAT               "Fmt"

 /* Tag names as character arrays */
 #define MQMCD_MSG_DOMAIN_A             'M','s','d'
 #define MQMCD_MSG_SET_A                'S','e','t'
 #define MQMCD_MSG_TYPE_A               'T','y','p','e'
 #define MQMCD_MSG_FORMAT_A             'F','m','t'

 /* XML tag names */
 #define MQMCD_MSG_DOMAIN_B             "<Msd>"
 #define MQMCD_MSG_DOMAIN_E             "</Msd>"
 #define MQMCD_MSG_SET_B                "<Set>"
 #define MQMCD_MSG_SET_E                "</Set>"
 #define MQMCD_MSG_TYPE_B               "<Type>"
 #define MQMCD_MSG_TYPE_E               "</Type>"
 #define MQMCD_MSG_FORMAT_B             "<Fmt>"
 #define MQMCD_MSG_FORMAT_E             "</Fmt>"

 /* XML tag names as character arrays */
 #define MQMCD_MSG_DOMAIN_BA            '<','M','s','d','>'
 #define MQMCD_MSG_DOMAIN_EA            '<','/','M','s','d','>'
 #define MQMCD_MSG_SET_BA               '<','S','e','t','>'
 #define MQMCD_MSG_SET_EA               '<','/','S','e','t','>'
 #define MQMCD_MSG_TYPE_BA              '<','T','y','p','e','>'
 #define MQMCD_MSG_TYPE_EA              '<','/','T','y','p','e','>'
 #define MQMCD_MSG_FORMAT_BA            '<','F','m','t','>'
 #define MQMCD_MSG_FORMAT_EA            '<','/','F','m','t','>'

 /* Tag values */
 #define MQMCD_DOMAIN_NONE              "none"
 #define MQMCD_DOMAIN_NEON              "neon"
 #define MQMCD_DOMAIN_MRM               "mrm"
 #define MQMCD_DOMAIN_JMS_NONE          "jms_none"
 #define MQMCD_DOMAIN_JMS_TEXT          "jms_text"
 #define MQMCD_DOMAIN_JMS_OBJECT        "jms_object"
 #define MQMCD_DOMAIN_JMS_MAP           "jms_map"
 #define MQMCD_DOMAIN_JMS_STREAM        "jms_stream"
 #define MQMCD_DOMAIN_JMS_BYTES         "jms_bytes"

 /* Tag values as character arrays */
 #define MQMCD_DOMAIN_NONE_A            'n','o','n','e'
 #define MQMCD_DOMAIN_NEON_A            'n','e','o','n'
 #define MQMCD_DOMAIN_MRM_A             'm','r','m'
 #define MQMCD_DOMAIN_JMS_NONE_A        'j','m','s','_','n','o','n','e'
 #define MQMCD_DOMAIN_JMS_TEXT_A        'j','m','s','_','t','e','x','t'
 #define MQMCD_DOMAIN_JMS_OBJECT_A      'j','m','s','_','o','b','j','e',\
                                        'c','t'
 #define MQMCD_DOMAIN_JMS_MAP_A         'j','m','s','_','m','a','p'
 #define MQMCD_DOMAIN_JMS_STREAM_A      'j','m','s','_','s','t','r','e',\
                                        'a','m'
 #define MQMCD_DOMAIN_JMS_BYTES_A       'j','m','s','_','b','y','t','e',\
                                        's'

 #endif /* MQMCD_FOLDER_VERSION */

 /****************************************************************/
 /* Publish/Subscribe Command Folder (psc) Tags                  */
 /****************************************************************/

 #if !defined(MQPSC_FOLDER_VERSION)
 #define MQPSC_FOLDER_VERSION 1

 /* Tag names */
 #define MQPSC_COMMAND                  "Command"
 #define MQPSC_REGISTRATION_OPTION      "RegOpt"
 #define MQPSC_PUBLICATION_OPTION       "PubOpt"
 #define MQPSC_DELETE_OPTION            "DelOpt"
 #define MQPSC_TOPIC                    "Topic"
 #define MQPSC_SUBSCRIPTION_POINT       "SubPoint"
 #define MQPSC_FILTER                   "Filter"
 #define MQPSC_Q_MGR_NAME               "QMgrName"
 #define MQPSC_Q_NAME                   "QName"
 #define MQPSC_PUBLISH_TIMESTAMP        "PubTime"
 #define MQPSC_SEQUENCE_NUMBER          "SeqNum"
 #define MQPSC_SUBSCRIPTION_NAME        "SubName"
 #define MQPSC_SUBSCRIPTION_IDENTITY    "SubIdentity"
 #define MQPSC_SUBSCRIPTION_USER_DATA   "SubUserData"
 #define MQPSC_CORREL_ID                "CorrelId"

 /* Tag names as character arrays */
 #define MQPSC_COMMAND_A                'C','o','m','m','a','n','d'
 #define MQPSC_REGISTRATION_OPTION_A    'R','e','g','O','p','t'
 #define MQPSC_PUBLICATION_OPTION_A     'P','u','b','O','p','t'
 #define MQPSC_DELETE_OPTION_A          'D','e','l','O','p','t'
 #define MQPSC_TOPIC_A                  'T','o','p','i','c'
 #define MQPSC_SUBSCRIPTION_POINT_A     'S','u','b','P','o','i','n','t'
 #define MQPSC_FILTER_A                 'F','i','l','t','e','r'
 #define MQPSC_Q_MGR_NAME_A             'Q','M','g','r','N','a','m','e'
 #define MQPSC_Q_NAME_A                 'Q','N','a','m','e'
 #define MQPSC_PUBLISH_TIMESTAMP_A      'P','u','b','T','i','m','e'
 #define MQPSC_SEQUENCE_NUMBER_A        'S','e','q','N','u','m'
 #define MQPSC_SUBSCRIPTION_NAME_A      'S','u','b','N','a','m','e'
 #define MQPSC_SUBSCRIPTION_IDENTITY_A  'S','u','b','I','d','e','n','t',\
                                        'i','t','y'
 #define MQPSC_SUBSCRIPTION_USER_DATA_A 'S','u','b','U','s','e','r','D',\
                                        'a','t','a'
 #define MQPSC_CORREL_ID_A              'C','o','r','r','e','l','I','d'

 /* XML tag names */
 #define MQPSC_COMMAND_B                "<Command>"
 #define MQPSC_COMMAND_E                "</Command>"
 #define MQPSC_REGISTRATION_OPTION_B    "<RegOpt>"
 #define MQPSC_REGISTRATION_OPTION_E    "</RegOpt>"
 #define MQPSC_PUBLICATION_OPTION_B     "<PubOpt>"
 #define MQPSC_PUBLICATION_OPTION_E     "</PubOpt>"
 #define MQPSC_DELETE_OPTION_B          "<DelOpt>"
 #define MQPSC_DELETE_OPTION_E          "</DelOpt>"
 #define MQPSC_TOPIC_B                  "<Topic>"
 #define MQPSC_TOPIC_E                  "</Topic>"
 #define MQPSC_SUBSCRIPTION_POINT_B     "<SubPoint>"
 #define MQPSC_SUBSCRIPTION_POINT_E     "</SubPoint>"
 #define MQPSC_FILTER_B                 "<Filter>"
 #define MQPSC_FILTER_E                 "</Filter>"
 #define MQPSC_Q_MGR_NAME_B             "<QMgrName>"
 #define MQPSC_Q_MGR_NAME_E             "</QMgrName>"
 #define MQPSC_Q_NAME_B                 "<QName>"
 #define MQPSC_Q_NAME_E                 "</QName>"
 #define MQPSC_PUBLISH_TIMESTAMP_B      "<PubTime>"
 #define MQPSC_PUBLISH_TIMESTAMP_E      "</PubTime>"
 #define MQPSC_SEQUENCE_NUMBER_B        "<SeqNum>"
 #define MQPSC_SEQUENCE_NUMBER_E        "</SeqNum>"
 #define MQPSC_SUBSCRIPTION_NAME_B      "<SubName>"
 #define MQPSC_SUBSCRIPTION_NAME_E      "</SubName>"
 #define MQPSC_SUBSCRIPTION_IDENTITY_B  "<SubIdentity>"
 #define MQPSC_SUBSCRIPTION_IDENTITY_E  "</SubIdentity>"
 #define MQPSC_SUBSCRIPTION_USER_DATA_B "<SubUserData>"
 #define MQPSC_SUBSCRIPTION_USER_DATA_E "</SubUserData>"
 #define MQPSC_CORREL_ID_B              "<CorrelId>"
 #define MQPSC_CORREL_ID_E              "</CorrelId>"

 /* XML tag names as character arrays */
 #define MQPSC_COMMAND_BA                '<','C','o','m','m','a','n','d',\
                                         '>'
 #define MQPSC_COMMAND_EA                '<','/','C','o','m','m','a','n',\
                                         'd','>'
 #define MQPSC_REGISTRATION_OPTION_BA    '<','R','e','g','O','p','t','>'
 #define MQPSC_REGISTRATION_OPTION_EA    '<','/','R','e','g','O','p','t',\
                                         '>'
 #define MQPSC_PUBLICATION_OPTION_BA     '<','P','u','b','O','p','t','>'
 #define MQPSC_PUBLICATION_OPTION_EA     '<','/','P','u','b','O','p','t',\
                                         '>'
 #define MQPSC_DELETE_OPTION_BA          '<','D','e','l','O','p','t','>'
 #define MQPSC_DELETE_OPTION_EA          '<','/','D','e','l','O','p','t',\
                                         '>'
 #define MQPSC_TOPIC_BA                  '<','T','o','p','i','c','>'
 #define MQPSC_TOPIC_EA                  '<','/','T','o','p','i','c','>'
 #define MQPSC_SUBSCRIPTION_POINT_BA     '<','S','u','b','P','o','i','n',\
                                         't','>'
 #define MQPSC_SUBSCRIPTION_POINT_EA     '<','/','S','u','b','P','o','i',\
                                         'n','t','>'
 #define MQPSC_FILTER_BA                 '<','F','i','l','t','e','r','>'
 #define MQPSC_FILTER_EA                 '<','/','F','i','l','t','e','r',\
                                         '>'
 #define MQPSC_Q_MGR_NAME_BA             '<','Q','M','g','r','N','a','m',\
                                         'e','>'
 #define MQPSC_Q_MGR_NAME_EA             '<','/','Q','M','g','r','N','a',\
                                         'm','e','>'
 #define MQPSC_Q_NAME_BA                 '<','Q','N','a','m','e','>'
 #define MQPSC_Q_NAME_EA                 '<','/','Q','N','a','m','e','>'
 #define MQPSC_PUBLISH_TIMESTAMP_BA      '<','P','u','b','T','i','m','e',\
                                         '>'
 #define MQPSC_PUBLISH_TIMESTAMP_EA      '<','/','P','u','b','T','i','m',\
                                         'e','>'
 #define MQPSC_SEQUENCE_NUMBER_BA        '<','S','e','q','N','u','m','>'
 #define MQPSC_SEQUENCE_NUMBER_EA        '<','/','S','e','q','N','u','m',\
                                         '>'
 #define MQPSC_SUBSCRIPTION_NAME_BA      '<','S','u','b','N','a','m','e',\
                                         '>'
 #define MQPSC_SUBSCRIPTION_NAME_EA      '<','/','S','u','b','N','a','m',\
                                         'e','>'
 #define MQPSC_SUBSCRIPTION_IDENTITY_BA  '<','S','u','b','I','d','e','n',\
                                         't','i','t','y','>'
 #define MQPSC_SUBSCRIPTION_IDENTITY_EA  '<','/','S','u','b','I','d','e',\
                                         'n','t','i','t','y','>'
 #define MQPSC_SUBSCRIPTION_USER_DATA_BA '<','S','u','b','U','s','e','r',\
                                         'D','a','t','a','>'
 #define MQPSC_SUBSCRIPTION_USER_DATA_EA '<','/','S','u','b','U','s','e',\
                                         'r','D','a','t','a','>'
 #define MQPSC_CORREL_ID_BA              '<','C','o','r','r','e','l','I',\
                                         'd','>'
 #define MQPSC_CORREL_ID_EA              '<','/','C','o','r','r','e','l',\
                                         'I','d','>'

 /****************************************************************/
 /* Values for MQPSC_COMMAND Tag                                 */
 /****************************************************************/

 /* Values as strings */
 #define MQPSC_DELETE_PUBLICATION       "DeletePub"
 #define MQPSC_DEREGISTER_SUBSCRIBER    "DeregSub"
 #define MQPSC_PUBLISH                  "Publish"
 #define MQPSC_REGISTER_SUBSCRIBER      "RegSub"
 #define MQPSC_REQUEST_UPDATE           "ReqUpdate"

 /* Values as character arrays */
 #define MQPSC_DELETE_PUBLICATION_A     'D','e','l','e','t','e','P','u',\
                                        'b'
 #define MQPSC_DEREGISTER_SUBSCRIBER_A  'D','e','r','e','g','S','u','b'
 #define MQPSC_PUBLISH_A                'P','u','b','l','i','s','h'
 #define MQPSC_REGISTER_SUBSCRIBER_A    'R','e','g','S','u','b'
 #define MQPSC_REQUEST_UPDATE_A         'R','e','q','U','p','d','a','t',\
                                        'e'

 /****************************************************************/
 /* Values for following tags:                                   */
 /*   MQPSC_DELETE_OPTION                                        */
 /*   MQPSC_PUBLICATION_OPTION                                   */
 /*   MQPSC_REGISTRATION_OPTION                                  */
 /****************************************************************/

 /* Values as strings */
 #define MQPSC_ADD_NAME                 "AddName"
 #define MQPSC_CORREL_ID_AS_IDENTITY    "CorrelAsId"
 #define MQPSC_DEREGISTER_ALL           "DeregAll"
 #define MQPSC_DUPLICATES_OK            "DupsOK"
 #define MQPSC_FULL_RESPONSE            "FullResp"
 #define MQPSC_INFORM_IF_RETAINED       "InformIfRet"
 #define MQPSC_IS_RETAINED_PUB          "IsRetainedPub"
 #define MQPSC_JOIN_SHARED              "JoinShared"
 #define MQPSC_JOIN_EXCLUSIVE           "JoinExcl"
 #define MQPSC_LEAVE_ONLY               "LeaveOnly"
 #define MQPSC_LOCAL                    "Local"
 #define MQPSC_LOCKED                   "Locked"
 #define MQPSC_NEW_PUBS_ONLY            "NewPubsOnly"
 #define MQPSC_NO_ALTERATION            "NoAlter"
 #define MQPSC_NON_PERSISTENT           "NonPers"
 #define MQPSC_OTHER_SUBS_ONLY          "OtherSubsOnly"
 #define MQPSC_PERSISTENT               "Pers"
 #define MQPSC_PERSISTENT_AS_PUBLISH    "PersAsPub"
 #define MQPSC_PERSISTENT_AS_Q          "PersAsQueue"
 #define MQPSC_NONE                     "None"
 #define MQPSC_PUB_ON_REQUEST_ONLY      "PubOnReqOnly"
 #define MQPSC_RETAIN_PUB               "RetainPub"
 #define MQPSC_VARIABLE_USER_ID         "VariableUserId"

 /* Values as character arrays */
 #define MQPSC_ADD_NAME_A               'A','d','d','N','a','m','e'
 #define MQPSC_CORREL_ID_AS_IDENTITY_A  'C','o','r','r','e','l','A','s',\
                                        'I','d'
 #define MQPSC_DEREGISTER_ALL_A         'D','e','r','e','g','A','l','l'
 #define MQPSC_DUPLICATES_OK_A          'D','u','p','s','O','K'
 #define MQPSC_FULL_RESPONSE_A          'F','u','l','l','R','e','s','p'
 #define MQPSC_INFORM_IF_RETAINED_A     'I','n','f','o','r','m','I','f',\
                                        'R','e','t'
 #define MQPSC_IS_RETAINED_PUB_A        'I','s','R','e','t','a','i','n',\
                                        'e','d','P','u','b'
 #define MQPSC_JOIN_SHARED_A            'J','o','i','n','S','h','a','r',\
                                        'e','d'
 #define MQPSC_JOIN_EXCLUSIVE_A         'J','o','i','n','E','x','c','l'
 #define MQPSC_LEAVE_ONLY_A             'L','e','a','v','e','O','n','l',\
                                        'y'
 #define MQPSC_LOCAL_A                  'L','o','c','a','l'
 #define MQPSC_LOCKED_A                 'L','o','c','k','e','d'
 #define MQPSC_NEW_PUBS_ONLY_A          'N','e','w','P','u','b','s','O',\
                                        'n','l','y'
 #define MQPSC_NO_ALTERATION_A          'N','o','A','l','t','e','r'
 #define MQPSC_NON_PERSISTENT_A         'N','o','n','P','e','r','s'
 #define MQPSC_OTHER_SUBS_ONLY_A        'O','t','h','e','r','S','u','b',\
                                        's','O','n','l','y'
 #define MQPSC_PERSISTENT_A             'P','e','r','s'
 #define MQPSC_PERSISTENT_AS_PUBLISH_A  'P','e','r','s','A','s','P','u',\
                                        'b'
 #define MQPSC_PERSISTENT_AS_Q_A        'P','e','r','s','A','s','Q','u',\
                                        'e','u','e'
 #define MQPSC_NONE_A                   'N','o','n','e'
 #define MQPSC_PUB_ON_REQUEST_ONLY_A    'P','u','b','O','n','R','e','q',\
                                        'O','n','l','y'
 #define MQPSC_RETAIN_PUB_A             'R','e','t','a','i','n','P','u',\
                                        'b'
 #define MQPSC_VARIABLE_USER_ID_A       'V','a','r','i','a','b','l','e',\
                                        'U','s','e','r','I','d'

 #endif /* MQPSC_FOLDER_VERSION */

 /****************************************************************/
 /* Publish/Subscribe Response Folder (pscr) Tags                */
 /****************************************************************/

 #if !defined(MQPSCR_FOLDER_VERSION)
 #define MQPSCR_FOLDER_VERSION 1

 /* Tag names */
 #define MQPSCR_COMPLETION              "Completion"
 #define MQPSCR_RESPONSE                "Response"
 #define MQPSCR_REASON                  "Reason"

 /* Tag names as character arrays */
 #define MQPSCR_COMPLETION_A            'C','o','m','p','l','e','t','i',\
                                        'o','n'
 #define MQPSCR_RESPONSE_A              'R','e','s','p','o','n','s','e'
 #define MQPSCR_REASON_A                'R','e','a','s','o','n'

 /* XML tag names */
 #define MQPSCR_COMPLETION_B            "<Completion>"
 #define MQPSCR_COMPLETION_E            "</Completion>"
 #define MQPSCR_RESPONSE_B              "<Response>"
 #define MQPSCR_RESPONSE_E              "</Response>"
 #define MQPSCR_REASON_B                "<Reason>"
 #define MQPSCR_REASON_E                "</Reason>"

 /* XML tag names as character arrays */
 #define MQPSCR_COMPLETION_BA           '<','C','o','m','p','l','e','t',\
                                        'i','o','n','>'
 #define MQPSCR_COMPLETION_EA           '<','/','C','o','m','p','l','e',\
                                        't','i','o','n','>'
 #define MQPSCR_RESPONSE_BA             '<','R','e','s','p','o','n','s',\
                                        'e','>'
 #define MQPSCR_RESPONSE_EA             '<','/','R','e','s','p','o','n',\
                                        's','e','>'
 #define MQPSCR_REASON_BA               '<','R','e','a','s','o','n','>'
 #define MQPSCR_REASON_EA               '<','/','R','e','a','s','o','n',\
                                        '>'

 /* Tag values */
 #define MQPSCR_OK                      "ok"
 #define MQPSCR_WARNING                 "warning"
 #define MQPSCR_ERROR                   "error"

 /* Tag values as character arrays */
 #define MQPSCR_OK_A                    'o','k'
 #define MQPSCR_WARNING_A               'w','a','r','n','i','n','g'
 #define MQPSCR_ERROR_A                 'e','r','r','o','r'

 #endif /* MQPSCR_FOLDER_VERSION */


 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQPSC                                               */
 /****************************************************************/
 #endif  /* End of header file */
