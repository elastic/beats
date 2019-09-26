 #if !defined(MQZC_INCLUDED)           /* File not yet included? */
   #define MQZC_INCLUDED               /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQZC                                       */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for Installable Services       */
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
 /*                  structures and named constants for          */
 /*                  Installable Services.                       */
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
 /* pn=com.ibm.mq.famfiles.data/xml/approved/cmqzc.xml           */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Values Related to MQZED Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQZED_STRUC_ID                 "ZED "

 /* Structure Identifier (array form) */
 #define MQZED_STRUC_ID_ARRAY           'Z','E','D',' '

 /* Structure Version Number */
 #define MQZED_VERSION_1                1
 #define MQZED_VERSION_2                2
 #define MQZED_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQZED_LENGTH_1                 64
#else
 #define MQZED_LENGTH_1                 56
#endif
#if defined(MQ_64_BIT)
 #define MQZED_LENGTH_2                 72
#else
 #define MQZED_LENGTH_2                 60
#endif
#if defined(MQ_64_BIT)
 #define MQZED_CURRENT_LENGTH           72
#else
 #define MQZED_CURRENT_LENGTH           60
#endif

 /****************************************************************/
 /* Values Related to MQZAC Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQZAC_STRUC_ID                 "ZAC "

 /* Structure Identifier (array form) */
 #define MQZAC_STRUC_ID_ARRAY           'Z','A','C',' '

 /* Structure Version Number */
 #define MQZAC_VERSION_1                1
 #define MQZAC_CURRENT_VERSION          1

 /* Structure Length */
 #define MQZAC_LENGTH_1                 84
 #define MQZAC_CURRENT_LENGTH           84

 /* Authentication Types */
 #define MQZAT_INITIAL_CONTEXT          0
 #define MQZAT_CHANGE_CONTEXT           1

 /****************************************************************/
 /* Values Related to MQZAD Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQZAD_STRUC_ID                 "ZAD "

 /* Structure Identifier (array form) */
 #define MQZAD_STRUC_ID_ARRAY           'Z','A','D',' '

 /* Structure Version Number */
 #define MQZAD_VERSION_1                1
 #define MQZAD_VERSION_2                2
 #define MQZAD_CURRENT_VERSION          2

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQZAD_LENGTH_1                 80
#else
 #define MQZAD_LENGTH_1                 72
#endif
#if defined(MQ_64_BIT)
 #define MQZAD_LENGTH_2                 80
#else
 #define MQZAD_LENGTH_2                 76
#endif
#if defined(MQ_64_BIT)
 #define MQZAD_CURRENT_LENGTH           80
#else
 #define MQZAD_CURRENT_LENGTH           76
#endif

 /****************************************************************/
 /* Values Related to MQZFP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQZFP_STRUC_ID                 "ZFP "

 /* Structure Identifier (array form) */
 #define MQZFP_STRUC_ID_ARRAY           'Z','F','P',' '

 /* Structure Version Number */
 #define MQZFP_VERSION_1                1
 #define MQZFP_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQZFP_LENGTH_1                 24
#else
 #define MQZFP_LENGTH_1                 20
#endif
#if defined(MQ_64_BIT)
 #define MQZFP_CURRENT_LENGTH           24
#else
 #define MQZFP_CURRENT_LENGTH           20
#endif

 /****************************************************************/
 /* Values Related to MQZIC Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQZIC_STRUC_ID                 "ZIC "

 /* Structure Identifier (array form) */
 #define MQZIC_STRUC_ID_ARRAY           'Z','I','C',' '

 /* Structure Version Number */
 #define MQZIC_VERSION_1                1
 #define MQZIC_CURRENT_VERSION          1

 /* Structure Length */
 #define MQZIC_LENGTH_1                 84
 #define MQZIC_CURRENT_LENGTH           84

 /****************************************************************/
 /* Values Related to All Services                               */
 /****************************************************************/

 /* Initialization Options */
 #define MQZIO_PRIMARY                  0
 #define MQZIO_SECONDARY                1

 /* Termination Options */
 #define MQZTO_PRIMARY                  0
 #define MQZTO_SECONDARY                1

 /* Continuation Indicator */
 #define MQZCI_DEFAULT                  0
 #define MQZCI_CONTINUE                 0
 #define MQZCI_STOP                     1

 /****************************************************************/
 /* Values Related to Authority Service                          */
 /****************************************************************/

 /* Service Interface Version */
 #define MQZAS_VERSION_1                1
 #define MQZAS_VERSION_2                2
 #define MQZAS_VERSION_3                3
 #define MQZAS_VERSION_4                4
 #define MQZAS_VERSION_5                5
 #define MQZAS_VERSION_6                6

 /* Authorizations */
 #define MQZAO_CONNECT                  0x00000001
 #define MQZAO_BROWSE                   0x00000002
 #define MQZAO_INPUT                    0x00000004
 #define MQZAO_OUTPUT                   0x00000008
 #define MQZAO_INQUIRE                  0x00000010
 #define MQZAO_SET                      0x00000020
 #define MQZAO_PASS_IDENTITY_CONTEXT    0x00000040
 #define MQZAO_PASS_ALL_CONTEXT         0x00000080
 #define MQZAO_SET_IDENTITY_CONTEXT     0x00000100
 #define MQZAO_SET_ALL_CONTEXT          0x00000200
 #define MQZAO_ALTERNATE_USER_AUTHORITY 0x00000400
 #define MQZAO_PUBLISH                  0x00000800
 #define MQZAO_SUBSCRIBE                0x00001000
 #define MQZAO_RESUME                   0x00002000
 #define MQZAO_ALL_MQI                  0x00003FFF
 #define MQZAO_CREATE                   0x00010000
 #define MQZAO_DELETE                   0x00020000
 #define MQZAO_DISPLAY                  0x00040000
 #define MQZAO_CHANGE                   0x00080000
 #define MQZAO_CLEAR                    0x00100000
 #define MQZAO_CONTROL                  0x00200000
 #define MQZAO_CONTROL_EXTENDED         0x00400000
 #define MQZAO_AUTHORIZE                0x00800000
 #define MQZAO_ALL_ADMIN                0x00FE0000
 #define MQZAO_SYSTEM                   0x02000000
 #define MQZAO_ALL                      0x02FE3FFF
 #define MQZAO_REMOVE                   0x01000000
 #define MQZAO_NONE                     0x00000000
 #define MQZAO_CREATE_ONLY              0x04000000

 /* Entity Types */
 #define MQZAET_NONE                    0x00000000
 #define MQZAET_PRINCIPAL               0x00000001
 #define MQZAET_GROUP                   0x00000002
 #define MQZAET_UNKNOWN                 0x00000003

 /* Start-Enumeration Indicator */
 #define MQZSE_START                    1
 #define MQZSE_CONTINUE                 0

 /* Selector Indicator */
 #define MQZSL_NOT_RETURNED             0
 #define MQZSL_RETURNED                 1

 /****************************************************************/
 /* Values Related to Name Service                               */
 /****************************************************************/

 /* Service Interface Version */
 #define MQZNS_VERSION_1                1

 /****************************************************************/
 /* Values Related to Userid Service                             */
 /****************************************************************/

 /* Service Interface Version */
 #define MQZUS_VERSION_1                1

 /****************************************************************/
 /* Values Related to MQZEP Function                             */
 /****************************************************************/

 /* Function ids common to all services */
 #define MQZID_INIT                     0
 #define MQZID_TERM                     1

 /* Function ids for Authority service */
 #define MQZID_INIT_AUTHORITY           0
 #define MQZID_TERM_AUTHORITY           1
 #define MQZID_CHECK_AUTHORITY          2
 #define MQZID_COPY_ALL_AUTHORITY       3
 #define MQZID_DELETE_AUTHORITY         4
 #define MQZID_SET_AUTHORITY            5
 #define MQZID_GET_AUTHORITY            6
 #define MQZID_GET_EXPLICIT_AUTHORITY   7
 #define MQZID_REFRESH_CACHE            8
 #define MQZID_ENUMERATE_AUTHORITY_DATA 9
 #define MQZID_AUTHENTICATE_USER        10
 #define MQZID_FREE_USER                11
 #define MQZID_INQUIRE                  12
 #define MQZID_CHECK_PRIVILEGED         13

 /* Function ids for Name service */
 #define MQZID_INIT_NAME                0
 #define MQZID_TERM_NAME                1
 #define MQZID_LOOKUP_NAME              2
 #define MQZID_INSERT_NAME              3
 #define MQZID_DELETE_NAME              4

 /* Function ids for Userid service */
 #define MQZID_INIT_USERID              0
 #define MQZID_TERM_USERID              1
 #define MQZID_FIND_USERID              2

 /****************************************************************/
 /* MQZED Structure -- Entity Data                               */
 /****************************************************************/


 typedef struct tagMQZED MQZED;
 typedef MQZED MQPOINTER PMQZED;

 struct tagMQZED {
   MQCHAR4   StrucId;          /* Structure identifier */
   MQLONG    Version;          /* Structure version number */
   PMQCHAR   EntityNamePtr;    /* Address of entity name */
   PMQCHAR   EntityDomainPtr;  /* Address of entity domain name */
   MQBYTE40  SecurityId;       /* Security identifier */
   /* Ver:1 */
   MQPTR     CorrelationPtr;   /* Address of correlational data */
   /* Ver:2 */
 };

 #define MQZED_DEFAULT {MQZED_STRUC_ID_ARRAY},\
                       MQZED_VERSION_1,\
                       NULL,\
                       NULL,\
                       {MQSID_NONE_ARRAY},\
                       NULL

 /****************************************************************/
 /* MQZAC Structure -- Application Context                       */
 /****************************************************************/


 typedef struct tagMQZAC MQZAC;
 typedef MQZAC MQPOINTER PMQZAC;

 struct tagMQZAC {
   MQCHAR4   StrucId;             /* Structure identifier */
   MQLONG    Version;             /* Structure version number */
   MQPID     ProcessId;           /* Process identifier of */
                                  /* application */
   MQTID     ThreadId;            /* Thread identifier of application */
   MQCHAR28  ApplName;            /* Application name */
   MQCHAR12  UserID;              /* User ID of application */
   MQCHAR12  EffectiveUserID;     /* Effective user ID of application */
   MQLONG    Environment;         /* Environment of caller */
   MQLONG    CallerType;          /* Type of caller */
   MQLONG    AuthenticationType;  /* Type of authentication being */
                                  /* performed */
   MQLONG    BindType;            /* Type of bindings in use */
 };

 #define MQZAC_DEFAULT {MQZAC_STRUC_ID_ARRAY},\
                       MQZAC_VERSION_1,\
                       0,\
                       0,\
                       {""},\
                       {""},\
                       {""},\
                       0,\
                       0,\
                       0,\
                       0

 /****************************************************************/
 /* MQZAD Structure -- Authority Data                            */
 /****************************************************************/


 typedef struct tagMQZAD MQZAD;
 typedef MQZAD MQPOINTER PMQZAD;

 struct tagMQZAD {
   MQCHAR4   StrucId;        /* Structure identifier */
   MQLONG    Version;        /* Structure version number */
   MQCHAR48  ProfileName;    /* Profile name */
   MQLONG    ObjectType;     /* Object type */
   MQLONG    Authority;      /* Authority */
   PMQZED    EntityDataPtr;  /* Address of MQZED structure */
                             /* identifying an entity */
   MQLONG    EntityType;     /* Entity type */
   /* Ver:1 */
   MQLONG    Options;        /* Options */
   /* Ver:2 */
 };

 #define MQZAD_DEFAULT {MQZAD_STRUC_ID_ARRAY},\
                       MQZAD_VERSION_1,\
                       {""},\
                       MQOT_ALL,\
                       0,\
                       NULL,\
                       0,\
                       (MQAUTHOPT_NAME_ALL_MATCHING | MQAUTHOPT_ENTITY_EXPLICIT)

 /****************************************************************/
 /* MQZFP Structure -- Free Parameters                           */
 /****************************************************************/


 typedef struct tagMQZFP MQZFP;
 typedef MQZFP MQPOINTER PMQZFP;

 struct tagMQZFP {
   MQCHAR4  StrucId;         /* Structure identifier */
   MQLONG   Version;         /* Structure version number */
   MQBYTE8  Reserved;        /* Reserved */
   MQPTR    CorrelationPtr;  /* Address of correlational data */
 };

 #define MQZFP_DEFAULT {MQZFP_STRUC_ID_ARRAY},\
                       MQZFP_VERSION_1,\
                       {'\0','\0','\0','\0','\0','\0','\0','\0'},\
                       NULL

 /****************************************************************/
 /* MQZIC Structure -- Identity Context                          */
 /****************************************************************/


 typedef struct tagMQZIC MQZIC;
 typedef MQZIC MQPOINTER PMQZIC;

 struct tagMQZIC {
   MQCHAR4   StrucId;           /* Structure identifier */
   MQLONG    Version;           /* Structure version number */
   MQCHAR12  UserIdentifier;    /* User identifier */
   MQBYTE32  AccountingToken;   /* Accounting token */
   MQCHAR32  ApplIdentityData;  /* Application data relating to */
                                /* identity */
 };

 #define MQZIC_DEFAULT {MQZIC_STRUC_ID_ARRAY},\
                       MQZIC_VERSION_1,\
                       {""},\
                       {MQACT_NONE_ARRAY},\
                       {""}

 /****************************************************************/
 /* MQZEP -- Add Component Entry Point                           */
 /****************************************************************/

 void MQENTRY MQZEP (
   MQHCONFIG  Hconfig,      /* I: Configuration handle */
   MQLONG     Function,     /* I: Function identifier */
   PMQFUNC    pEntryPoint,  /* I: Function entry point */
   PMQLONG    pCompCode,    /* OC: Completion code */
   PMQLONG    pReason);     /* OR: Reason code qualifying CompCode */

 typedef void MQENTRY MQ_ZEP_CALL (
   MQHCONFIG  Hconfig,      /* I: Configuration handle */
   MQLONG     Function,     /* I: Function identifier */
   PMQFUNC    pEntryPoint,  /* I: Function entry point */
   PMQLONG    pCompCode,    /* OC: Completion code */
   PMQLONG    pReason);     /* OR: Reason code qualifying CompCode */
 typedef MQ_ZEP_CALL MQPOINTER PMQ_ZEP_CALL;


 /****************************************************************/
 /* MQZ_INIT_AUTHORITY - Initialize Authority-Services           */
 /****************************************************************/

 typedef void MQENTRY MQZ_INIT_AUTHORITY (
   MQHCONFIG  Hconfig,              /* I: Configuration handle */
   MQLONG     Options,              /* I: Initialization options */
   PMQCHAR    pQMgrName,            /* I: Queue manager name */
   MQLONG     ComponentDataLength,  /* I: Length of component data */
   PMQBYTE    pComponentData,       /* IO: Component data */
   PMQLONG    pVersion,             /* I: Version number */
   PMQLONG    pCompCode,            /* OC: Completion code */
   PMQLONG    pReason);             /* OR: Reason code qualifying */
                                    /* CompCode */
 typedef MQZ_INIT_AUTHORITY MQPOINTER PMQZ_INIT_AUTHORITY;


 /****************************************************************/
 /* MQZ_TERM_AUTHORITY - Terminate Authority-Services            */
 /****************************************************************/

 typedef void MQENTRY MQZ_TERM_AUTHORITY (
   MQHCONFIG  Hconfig,         /* I: Configuration handle */
   MQLONG     Options,         /* I: Termination options */
   PMQCHAR    pQMgrName,       /* I: Queue manager name */
   PMQBYTE    pComponentData,  /* I: Component data */
   PMQLONG    pCompCode,       /* OC: Completion code */
   PMQLONG    pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_TERM_AUTHORITY MQPOINTER PMQZ_TERM_AUTHORITY;


 /****************************************************************/
 /* MQZ_DELETE_AUTHORITY - Delete Authority                      */
 /****************************************************************/

 typedef void MQENTRY MQZ_DELETE_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_DELETE_AUTHORITY MQPOINTER PMQZ_DELETE_AUTHORITY;


 /****************************************************************/
 /* MQZ_GET_AUTHORITY - Get Authority                            */
 /****************************************************************/

 typedef void MQENTRY MQZ_GET_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pEntityName,     /* I: Entity name */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQLONG  pAuthority,      /* O: Authority of entity */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_GET_AUTHORITY MQPOINTER PMQZ_GET_AUTHORITY;


 /****************************************************************/
 /* MQZ_GET_AUTHORITY_2 - Get Authority Version 2                */
 /****************************************************************/

 typedef void MQENTRY MQZ_GET_AUTHORITY_2 (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZED   pEntityData,     /* I: Entity data */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQLONG  pAuthority,      /* O: Authority of entity */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_GET_AUTHORITY_2 MQPOINTER PMQZ_GET_AUTHORITY_2;


 /****************************************************************/
 /* MQZ_GET_EXPLICIT_AUTHORITY - Get Explicit Authority          */
 /****************************************************************/

 typedef void MQENTRY MQZ_GET_EXPLICIT_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pEntityName,     /* I: Entity name */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQLONG  pAuthority,      /* O: Authority of entity */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_GET_EXPLICIT_AUTHORITY MQPOINTER PMQZ_GET_EXPLICIT_AUTHORITY;


 /****************************************************************/
 /* MQZ_GET_EXPLICIT_AUTHORITY_2 - Get Explicit Authority        */
 /* Version 2                                                    */
 /****************************************************************/

 typedef void MQENTRY MQZ_GET_EXPLICIT_AUTHORITY_2 (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZED   pEntityData,     /* I: Entity data */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQLONG  pAuthority,      /* O: Authority of entity */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_GET_EXPLICIT_AUTHORITY_2 MQPOINTER PMQZ_GET_EXPLICIT_AUTHORITY_2;


 /****************************************************************/
 /* MQZ_ENUMERATE_AUTHORITY_DATA - Enumerate Authority Data      */
 /****************************************************************/

 typedef void MQENTRY MQZ_ENUMERATE_AUTHORITY_DATA (
   PMQCHAR  pQMgrName,              /* I: Queue manager name */
   MQLONG   StartEnumeration,       /* I: Flag indicating whether */
                                    /* call should start enumeration */
   PMQZAD   pFilter,                /* I: Filter */
   MQLONG   AuthorityBufferLength,  /* I: Length of AuthorityBuffer */
   PMQZAD   pAuthorityBuffer,       /* O: Authority data */
   PMQLONG  pAuthorityDataLength,   /* O: Length of data returned in */
                                    /* AuthorityBuffer */
   PMQBYTE  pComponentData,         /* IO: Component data */
   PMQLONG  pContinuation,          /* O: Continuation indicator set */
                                    /* by component */
   PMQLONG  pCompCode,              /* OC: Completion code */
   PMQLONG  pReason);               /* OR: Reason code qualifying */
                                    /* CompCode */
 typedef MQZ_ENUMERATE_AUTHORITY_DATA MQPOINTER PMQZ_ENUMERATE_AUTHORITY_DATA;


 /****************************************************************/
 /* MQZ_SET_AUTHORITY - Set Authority                            */
 /****************************************************************/

 typedef void MQENTRY MQZ_SET_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pEntityName,     /* I: Entity name */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   MQLONG   Authority,       /* I: Authority to be checked */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_SET_AUTHORITY MQPOINTER PMQZ_SET_AUTHORITY;


 /****************************************************************/
 /* MQZ_SET_AUTHORITY_2 - Set Authority Version 2                */
 /****************************************************************/

 typedef void MQENTRY MQZ_SET_AUTHORITY_2 (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZED   pEntityData,     /* I: Entity data */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   MQLONG   Authority,       /* I: Authority to be checked */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_SET_AUTHORITY_2 MQPOINTER PMQZ_SET_AUTHORITY_2;


 /****************************************************************/
 /* MQZ_COPY_ALL_AUTHORITY - Copy All Authority                  */
 /****************************************************************/

 typedef void MQENTRY MQZ_COPY_ALL_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pRefObjectName,  /* I: Reference object name */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_COPY_ALL_AUTHORITY MQPOINTER PMQZ_COPY_ALL_AUTHORITY;


 /****************************************************************/
 /* MQZ_CHECK_AUTHORITY - Check Authority                        */
 /****************************************************************/

 typedef void MQENTRY MQZ_CHECK_AUTHORITY (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pEntityName,     /* I: Entity name */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   MQLONG   Authority,       /* I: Authority to be checked */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_CHECK_AUTHORITY MQPOINTER PMQZ_CHECK_AUTHORITY;


 /****************************************************************/
 /* MQZ_CHECK_AUTHORITY_2 - Check Authority Version 2            */
 /****************************************************************/

 typedef void MQENTRY MQZ_CHECK_AUTHORITY_2 (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZED   pEntityData,     /* I: Entity data */
   MQLONG   EntityType,      /* I: Entity type */
   PMQCHAR  pObjectName,     /* I: Object name */
   MQLONG   ObjectType,      /* I: Object type */
   MQLONG   Authority,       /* I: Authority to be checked */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_CHECK_AUTHORITY_2 MQPOINTER PMQZ_CHECK_AUTHORITY_2;


 /****************************************************************/
 /* MQZ_AUTHENTICATE_USER - Authenticate User                    */
 /****************************************************************/

 typedef void MQENTRY MQZ_AUTHENTICATE_USER (
   PMQCHAR  pQMgrName,            /* I: Queue manager name */
   PMQCSP   pSecurityParms,       /* I: Security parameters */
   PMQZAC   pApplicationContext,  /* I: Application context */
   PMQZIC   pIdentityContext,     /* I: Identity context */
   PMQPTR   pCorrelationPtr,      /* I: Correlation data */
   PMQBYTE  pComponentData,       /* IO: Component data */
   PMQLONG  pContinuation,        /* O: Continuation indicator set by */
                                  /* component */
   PMQLONG  pCompCode,            /* OC: Completion code */
   PMQLONG  pReason);             /* OR: Reason code qualifying */
                                  /* CompCode */
 typedef MQZ_AUTHENTICATE_USER MQPOINTER PMQZ_AUTHENTICATE_USER;


 /****************************************************************/
 /* MQZ_FREE_USER - Free User                                    */
 /****************************************************************/

 typedef void MQENTRY MQZ_FREE_USER (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZFP   pFreeParms,      /* I: Free parameters */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_FREE_USER MQPOINTER PMQZ_FREE_USER;


 /****************************************************************/
 /* MQZ_INQUIRE - Inquire                                        */
 /****************************************************************/

 typedef void MQENTRY MQZ_INQUIRE (
   PMQCHAR  pQMgrName,          /* I: Queue manager name */
   MQLONG   SelectorCount,      /* I: Count of selectors */
   PMQLONG  pSelectors,         /* I: Array of attribute selectors */
   MQLONG   IntAttrCount,       /* I: Count of integer attributes */
   PMQLONG  pIntAttrs,          /* I: Array of integer attributes */
   MQLONG   CharAttrLength,     /* I: Length of character attributes */
                                /* buffer */
   PMQCHAR  pCharAttrs,         /* I: Character attributes */
   PMQLONG  pSelectorReturned,  /* O: Array of returned selector */
                                /* indicators */
   PMQBYTE  pComponentData,     /* IO: Component data */
   PMQLONG  pContinuation,      /* O: Continuation indicator set by */
                                /* component */
   PMQLONG  pCompCode,          /* OC: Completion code */
   PMQLONG  pReason);           /* OR: Reason code qualifying */
                                /* CompCode */
 typedef MQZ_INQUIRE MQPOINTER PMQZ_INQUIRE;


 /****************************************************************/
 /* MQZ_REFRESH_CACHE - Refresh Cache                            */
 /****************************************************************/

 typedef void MQENTRY MQZ_REFRESH_CACHE (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_REFRESH_CACHE MQPOINTER PMQZ_REFRESH_CACHE;


 /****************************************************************/
 /* MQZ_CHECK_PRIVILEGED - Check if User is Privileged           */
 /****************************************************************/

 typedef void MQENTRY MQZ_CHECK_PRIVILEGED (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQZED   pEntityData,     /* I: Entity data */
   MQLONG   EntityType,      /* I: Entity type */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_CHECK_PRIVILEGED MQPOINTER PMQZ_CHECK_PRIVILEGED;


 /****************************************************************/
 /* MQZ_INIT_NAME - Initialize Name-Services                     */
 /****************************************************************/

 typedef void MQENTRY MQZ_INIT_NAME (
   MQHCONFIG  Hconfig,              /* I: Configuration handle */
   MQLONG     Options,              /* I: Initialization options */
   PMQCHAR    pQMgrName,            /* I: Queue manager name */
   MQLONG     ComponentDataLength,  /* I: Length of component data */
   PMQBYTE    pComponentData,       /* IO: Component data */
   PMQLONG    pVersion,             /* I: Version number */
   PMQLONG    pCompCode,            /* OC: Completion code */
   PMQLONG    pReason);             /* OR: Reason code qualifying */
                                    /* CompCode */
 typedef MQZ_INIT_NAME MQPOINTER PMQZ_INIT_NAME;


 /****************************************************************/
 /* MQZ_TERM_NAME - Terminate Name-Services                      */
 /****************************************************************/

 typedef void MQENTRY MQZ_TERM_NAME (
   MQHCONFIG  Hconfig,         /* I: Configuration handle */
   MQLONG     Options,         /* I: Termination options */
   PMQCHAR    pQMgrName,       /* I: Queue manager name */
   PMQBYTE    pComponentData,  /* I: Component data */
   PMQLONG    pCompCode,       /* OC: Completion code */
   PMQLONG    pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_TERM_NAME MQPOINTER PMQZ_TERM_NAME;


 /****************************************************************/
 /* MQZ_LOOKUP_NAME - Look-Up Name                               */
 /****************************************************************/

 typedef void MQENTRY MQZ_LOOKUP_NAME (
   PMQCHAR  pQMgrName,          /* I: Queue manager name */
   PMQCHAR  pQName,             /* I: Queue name */
   PMQCHAR  pResolvedQMgrName,  /* I: Resolved queue manager name */
   PMQBYTE  pComponentData,     /* IO: Component data */
   PMQLONG  pContinuation,      /* O: Continuation indicator set by */
                                /* component */
   PMQLONG  pCompCode,          /* OC: Completion code */
   PMQLONG  pReason);           /* OR: Reason code qualifying */
                                /* CompCode */
 typedef MQZ_LOOKUP_NAME MQPOINTER PMQZ_LOOKUP_NAME;


 /****************************************************************/
 /* MQZ_INSERT_NAME - Insert Name                                */
 /****************************************************************/

 typedef void MQENTRY MQZ_INSERT_NAME (
   PMQCHAR  pQMgrName,          /* I: Queue manager name */
   PMQCHAR  pQName,             /* I: Queue name */
   PMQCHAR  pResolvedQMgrName,  /* I: Resolved queue manager name */
   PMQBYTE  pComponentData,     /* IO: Component data */
   PMQLONG  pContinuation,      /* O: Continuation indicator set by */
                                /* component */
   PMQLONG  pCompCode,          /* OC: Completion code */
   PMQLONG  pReason);           /* OR: Reason code qualifying */
                                /* CompCode */
 typedef MQZ_INSERT_NAME MQPOINTER PMQZ_INSERT_NAME;


 /****************************************************************/
 /* MQZ_DELETE_NAME - Delete Name                                */
 /****************************************************************/

 typedef void MQENTRY MQZ_DELETE_NAME (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pQName,          /* I: Queue name */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_DELETE_NAME MQPOINTER PMQZ_DELETE_NAME;


 /****************************************************************/
 /* MQZ_INIT_USERID - Initialize Userid-Services                 */
 /****************************************************************/

 typedef void MQENTRY MQZ_INIT_USERID (
   MQHCONFIG  Hconfig,              /* I: Configuration handle */
   MQLONG     Options,              /* I: Initialization options */
   PMQCHAR    pQMgrName,            /* I: Queue manager name */
   MQLONG     ComponentDataLength,  /* I: Length of component data */
   PMQBYTE    pComponentData,       /* IO: Component data */
   PMQLONG    pVersion,             /* I: Version number */
   PMQLONG    pCompCode,            /* OC: Completion code */
   PMQLONG    pReason);             /* OR: Reason code qualifying */
                                    /* CompCode */
 typedef MQZ_INIT_USERID MQPOINTER PMQZ_INIT_USERID;


 /****************************************************************/
 /* MQZ_TERM_USERID - Terminate Userid-Services                  */
 /****************************************************************/

 typedef void MQENTRY MQZ_TERM_USERID (
   MQHCONFIG  Hconfig,         /* I: Configuration handle */
   MQLONG     Options,         /* I: Termination options */
   PMQCHAR    pQMgrName,       /* I: Queue manager name */
   PMQBYTE    pComponentData,  /* I: Component data */
   PMQLONG    pCompCode,       /* OC: Completion code */
   PMQLONG    pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_TERM_USERID MQPOINTER PMQZ_TERM_USERID;


 /****************************************************************/
 /* MQZ_FIND_USERID - Find Userid                                */
 /****************************************************************/

 typedef void MQENTRY MQZ_FIND_USERID (
   PMQCHAR  pQMgrName,       /* I: Queue manager name */
   PMQCHAR  pUserId,         /* I: User identifier */
   PMQCHAR  pPassword,       /* I: Password */
   PMQBYTE  pComponentData,  /* IO: Component data */
   PMQLONG  pContinuation,   /* O: Continuation indicator set by */
                             /* component */
   PMQLONG  pCompCode,       /* OC: Completion code */
   PMQLONG  pReason);        /* OR: Reason code qualifying CompCode */
 typedef MQZ_FIND_USERID MQPOINTER PMQZ_FIND_USERID;



 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQZC                                                */
 /****************************************************************/
 #endif  /* End of header file */
