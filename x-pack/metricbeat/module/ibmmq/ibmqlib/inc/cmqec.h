 #if !defined(MQEC_INCLUDED)           /* File not yet included? */
   #define MQEC_INCLUDED               /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQEC                                       */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for Interface Entry Points     */
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
 /*                  Interface Entry Points.                     */
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
 /* pn=com.ibm.mq.famfiles.data/xml/approved/cmqec.xml           */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Include files                                                */
 /****************************************************************/

 #include <cmqc.h>    /* Message Queueing Interface definitions  */
 #include <cmqxc.h>   /* MQI exit-related definitions            */
 #include <cmqzc.h>   /* MQI Installable services definitions    */

 /****************************************************************/
 /* Values Related to MQIEP Structure                            */
 /****************************************************************/

 /* Structure Identifier */
 #define MQIEP_STRUC_ID                 "IEP "

 /* Structure Identifier (array form) */
 #define MQIEP_STRUC_ID_ARRAY           'I','E','P',' '

 /* Structure Version Number */
 #define MQIEP_VERSION_1                1
 #define MQIEP_CURRENT_VERSION          1

 /* Structure Length */
#if defined(MQ_64_BIT)
 #define MQIEP_LENGTH_1                 264
#else
 #define MQIEP_LENGTH_1                 140
#endif
#if defined(MQ_64_BIT)
 #define MQIEP_CURRENT_LENGTH           264
#else
 #define MQIEP_CURRENT_LENGTH           140
#endif

 /* Flags */
 #define MQIEPF_NONE                    0x00000000
 #define MQIEPF_NON_THREADED_LIBRARY    0x00000000
 #define MQIEPF_THREADED_LIBRARY        0x00000001
 #define MQIEPF_CLIENT_LIBRARY          0x00000000
 #define MQIEPF_LOCAL_LIBRARY           0x00000002

 /****************************************************************/
 /* MQIEP Structure -- Interface Entry Points                    */
 /****************************************************************/


 struct tagMQIEP {
   MQCHAR4          StrucId;        /* Structure identifier */
   MQLONG           Version;        /* Structure version number */
   MQLONG           StrucLength;    /* Length of MQIEP structure */
   MQLONG           Flags;          /* Flags containing information */
                                    /* about the interface entry */
                                    /* points */
   MQPTR            Reserved;       /* Reserved */
   PMQ_BACK_CALL    MQBACK_Call;    /* MQBACK entry point */
   PMQ_BEGIN_CALL   MQBEGIN_Call;   /* MQBEGIN entry point */
   PMQ_BUFMH_CALL   MQBUFMH_Call;   /* MQBUFMH entry point */
   PMQ_CB_CALL      MQCB_Call;      /* MQCB entry point */
   PMQ_CLOSE_CALL   MQCLOSE_Call;   /* MQCLOSE entry point */
   PMQ_CMIT_CALL    MQCMIT_Call;    /* MQCMIT entry point */
   PMQ_CONN_CALL    MQCONN_Call;    /* MQCONN entry point */
   PMQ_CONNX_CALL   MQCONNX_Call;   /* MQCONNX entry point */
   PMQ_CRTMH_CALL   MQCRTMH_Call;   /* MQCRTMH entry point */
   PMQ_CTL_CALL     MQCTL_Call;     /* MQCTL entry point */
   PMQ_DISC_CALL    MQDISC_Call;    /* MQDISC entry point */
   PMQ_DLTMH_CALL   MQDLTMH_Call;   /* MQDLTMH entry point */
   PMQ_DLTMP_CALL   MQDLTMP_Call;   /* MQDLTMP entry point */
   PMQ_GET_CALL     MQGET_Call;     /* MQGET entry point */
   PMQ_INQ_CALL     MQINQ_Call;     /* MQINQ entry point */
   PMQ_INQMP_CALL   MQINQMP_Call;   /* MQINQMP entry point */
   PMQ_MHBUF_CALL   MQMHBUF_Call;   /* MQMHBUF entry point */
   PMQ_OPEN_CALL    MQOPEN_Call;    /* MQOPEN entry point */
   PMQ_PUT_CALL     MQPUT_Call;     /* MQPUT entry point */
   PMQ_PUT1_CALL    MQPUT1_Call;    /* MQPUT1 entry point */
   PMQ_SET_CALL     MQSET_Call;     /* MQSET entry point */
   PMQ_SETMP_CALL   MQSETMP_Call;   /* MQSETMP entry point */
   PMQ_STAT_CALL    MQSTAT_Call;    /* MQSTAT entry point */
   PMQ_SUB_CALL     MQSUB_Call;     /* MQSUB entry point */
   PMQ_SUBRQ_CALL   MQSUBRQ_Call;   /* MQSUBRQ entry point */
   PMQ_XCLWLN_CALL  MQXCLWLN_Call;  /* MQXCLWLN entry point */
   PMQ_XCNVC_CALL   MQXCNVC_Call;   /* MQXCNVC entry point */
   PMQ_XDX_CALL     MQXDX_Call;     /* MQXDX entry point */
   PMQ_XEP_CALL     MQXEP_Call;     /* MQXEP entry point */
   PMQ_ZEP_CALL     MQZEP_Call;     /* MQZEP entry point */
 };

 #define MQIEP_DEFAULT {MQIEP_STRUC_ID_ARRAY},\
                       MQIEP_VERSION_1,\
                       MQIEP_LENGTH_1,\
                       MQIEPF_NONE,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL,\
                       NULL


 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQEC                                                */
 /****************************************************************/
 #endif  /* End of header file */
