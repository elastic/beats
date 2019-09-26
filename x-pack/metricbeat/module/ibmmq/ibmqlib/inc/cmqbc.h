 #if !defined(MQBC_INCLUDED)           /* File not yet included? */
   #define MQBC_INCLUDED               /* Show file now included */
 /****************************************************************/
 /*                                                              */
 /*                        IBM MQ for Mac                        */
 /*                                                              */
 /*  FILE NAME:      CMQBC                                       */
 /*                                                              */
 /*  DESCRIPTION:    Declarations for MQ Administration          */
 /*                  Interface                                   */
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
 /*                  structures and named constants for the MQ   */
 /*                  administration interface (MQAI).            */
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
 /* @(#) famfiles/xml/approved/cmqbc.xml, famfiles, p000,        */
 /* p000-L100302 1.11 10/02/27 07:31:45                          */
 /* <END_BUILDINFO>                                              */
 /****************************************************************/

 #if defined(__cplusplus)
   extern "C" {
 #endif



 /****************************************************************/
 /* Values Related to Specific Functions                         */
 /****************************************************************/

 /* Create-Bag Options for mqCreateBag */
 #define MQCBO_NONE                     0x00000000
 #define MQCBO_USER_BAG                 0x00000000
 #define MQCBO_ADMIN_BAG                0x00000001
 #define MQCBO_COMMAND_BAG              0x00000010
 #define MQCBO_SYSTEM_BAG               0x00000020
 #define MQCBO_GROUP_BAG                0x00000040
 #define MQCBO_LIST_FORM_ALLOWED        0x00000002
 #define MQCBO_LIST_FORM_INHIBITED      0x00000000
 #define MQCBO_REORDER_AS_REQUIRED      0x00000004
 #define MQCBO_DO_NOT_REORDER           0x00000000
 #define MQCBO_CHECK_SELECTORS          0x00000008
 #define MQCBO_DO_NOT_CHECK_SELECTORS   0x00000000

 /* Buffer Length for mqAddString and mqSetString */
 #define MQBL_NULL_TERMINATED           (-1)

 /* Item Types for mqInquireItemInfo */
 #define MQITEM_INTEGER                 1
 #define MQITEM_STRING                  2
 #define MQITEM_BAG                     3
 #define MQITEM_BYTE_STRING             4
 #define MQITEM_INTEGER_FILTER          5
 #define MQITEM_STRING_FILTER           6
 #define MQITEM_INTEGER64               7
 #define MQITEM_BYTE_STRING_FILTER      8

 /*  */
 #define MQIT_INTEGER                   1
 #define MQIT_STRING                    2
 #define MQIT_BAG                       3

 /****************************************************************/
 /* Values Related to Most Functions                             */
 /****************************************************************/

 /* Handle Selectors */
 #define MQHA_FIRST                     4001
 #define MQHA_BAG_HANDLE                4001
 #define MQHA_LAST_USED                 4001
 #define MQHA_LAST                      6000

 /* Limits for Selectors for Object Attributes */
 #define MQOA_FIRST                     1
 #define MQOA_LAST                      9000

 /* Integer System Selectors */
 #define MQIASY_FIRST                   (-1)
 #define MQIASY_CODED_CHAR_SET_ID       (-1)
 #define MQIASY_TYPE                    (-2)
 #define MQIASY_COMMAND                 (-3)
 #define MQIASY_MSG_SEQ_NUMBER          (-4)
 #define MQIASY_CONTROL                 (-5)
 #define MQIASY_COMP_CODE               (-6)
 #define MQIASY_REASON                  (-7)
 #define MQIASY_BAG_OPTIONS             (-8)
 #define MQIASY_VERSION                 (-9)
 #define MQIASY_LAST_USED               (-9)
 #define MQIASY_LAST                    (-2000)

 /* Special Selector Values */
 #define MQSEL_ANY_SELECTOR             (-30001)
 #define MQSEL_ANY_USER_SELECTOR        (-30002)
 #define MQSEL_ANY_SYSTEM_SELECTOR      (-30003)
 #define MQSEL_ALL_SELECTORS            (-30001)
 #define MQSEL_ALL_USER_SELECTORS       (-30002)
 #define MQSEL_ALL_SYSTEM_SELECTORS     (-30003)

 /* Special Index Values */
 #define MQIND_NONE                     (-1)
 #define MQIND_ALL                      (-2)

 /* Bag Handles */
 #define MQHB_UNUSABLE_HBAG             (-1)
 #define MQHB_NONE                      (-2)

 /****************************************************************/
 /* Simple Data Types                                            */
 /****************************************************************/

 typedef MQLONG MQHBAG;
 typedef MQHBAG MQPOINTER PMQHBAG;

 /****************************************************************/
 /* Short Names for Functions                                    */
 /****************************************************************/

 #define MQADDBF  mqAddByteStringFilter
 #define MQADDBG  mqAddBag
 #define MQADDBS  mqAddByteString
 #define MQADDIQ  mqAddInquiry
 #define MQADDIN  mqAddInteger
 #define MQADD64  mqAddInteger64
 #define MQADDIF  mqAddIntegerFilter
 #define MQADDST  mqAddString
 #define MQADDSF  mqAddStringFilter
 #define MQBG2BF  mqBagToBuffer
 #define MQBF2BG  mqBufferToBag
 #define MQCLRBG  mqClearBag
 #define MQCNTIT  mqCountItems
 #define MQCRTBG  mqCreateBag
 #define MQDELBG  mqDeleteBag
 #define MQDELIT  mqDeleteItem
 #define MQEXEC   mqExecute
 #define MQGETBG  mqGetBag
 #define MQINQBF  mqInquireByteStringFilter
 #define MQINQBG  mqInquireBag
 #define MQINQBS  mqInquireByteString
 #define MQINQIN  mqInquireInteger
 #define MQINQ64  mqInquireInteger64
 #define MQINQIF  mqInquireIntegerFilter
 #define MQINQIT  mqInquireItemInfo
 #define MQINQST  mqInquireString
 #define MQINQSF  mqInquireStringFilter
 #define MQPAD    mqPad
 #define MQPUTBG  mqPutBag
 #define MQSETBF  mqSetByteStringFilter
 #define MQSETBS  mqSetByteString
 #define MQSETIN  mqSetInteger
 #define MQSET64  mqSetInteger64
 #define MQSETIF  mqSetIntegerFilter
 #define MQSETST  mqSetString
 #define MQSETSF  mqSetStringFilter
 #define MQTRIM   mqTrim
 #define MQTRNBG  mqTruncateBag

 /****************************************************************/
 /* mqAddBag Function -- Add Nested Bag to Bag                   */
 /****************************************************************/

 void MQENTRY mqAddBag (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQHBAG   ItemValue,  /* I: Item value */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddByteString Function -- Add Byte String to Bag           */
 /****************************************************************/

 void MQENTRY mqAddByteString (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQBYTE  pBuffer,       /* IB: Buffer containing item value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddByteStringFilter Function -- Add Byte String Filter to  */
 /* Bag                                                          */
 /****************************************************************/

 void MQENTRY mqAddByteStringFilter (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQBYTE  pBuffer,       /* IB: Buffer containing item value */
   MQLONG   Operator,      /* I: Item operator */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddInquiry Function -- Add an Inquiry Item to Bag          */
 /****************************************************************/

 void MQENTRY mqAddInquiry (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Attribute selector */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddInteger Function -- Add Integer to Bag                  */
 /****************************************************************/

 void MQENTRY mqAddInteger (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemValue,  /* I: Item value */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddInteger64 Function -- Add 64-bit Integer to Bag         */
 /****************************************************************/

 void MQENTRY mqAddInteger64 (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQINT64  ItemValue,  /* I: Item value */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddIntegerFilter Function -- Add Integer Filter to Bag     */
 /****************************************************************/

 void MQENTRY mqAddIntegerFilter (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemValue,  /* I: Item value */
   MQLONG   Operator,   /* I: Item operator */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddString Function -- Add String to Bag                    */
 /****************************************************************/

 void MQENTRY mqAddString (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* IB: Buffer containing item value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqAddStringFilter Function -- Add String Filter to Bag       */
 /****************************************************************/

 void MQENTRY mqAddStringFilter (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* IB: Buffer containing item value */
   MQLONG   Operator,      /* I: Item operator */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqBagToBuffer Function -- Convert Bag to PCF                 */
 /****************************************************************/

 void MQENTRY mqBagToBuffer (
   MQHBAG   OptionsBag,    /* I: Handle of options bag */
   MQHBAG   DataBag,       /* I: Handle of data bag */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQVOID  pBuffer,       /* OB: Buffer to contain PCF */
   PMQLONG  pDataLength,   /* OL: Length of PCF returned in buffer */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqBufferToBag Function -- Convert PCF to Bag                 */
 /****************************************************************/

 void MQENTRY mqBufferToBag (
   MQHBAG   OptionsBag,    /* I: Handle of options bag */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQVOID  pBuffer,       /* IB: Buffer containing PCF */
   MQHBAG   DataBag,       /* IO: Handle of bag to contain data */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqClearBag Function -- Delete All Items in Bag               */
 /****************************************************************/

 void MQENTRY mqClearBag (
   MQHBAG   Bag,        /* I: Bag handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqCountItems Function -- Count Items in Bag                  */
 /****************************************************************/

 void MQENTRY mqCountItems (
   MQHBAG   Bag,         /* I: Bag handle */
   MQLONG   Selector,    /* I: Item selector */
   PMQLONG  pItemCount,  /* O: Number of items */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqCreateBag Function -- Create Bag                           */
 /****************************************************************/

 void MQENTRY mqCreateBag (
   MQLONG   Options,    /* I: Bag options */
   PMQHBAG  pBag,       /* O: Handle of bag created */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqDeleteBag Function -- Delete Bag                           */
 /****************************************************************/

 void MQENTRY mqDeleteBag (
   PMQHBAG  pBag,       /* IO: Bag handle */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqDeleteItem Function -- Delete Item in Bag                  */
 /****************************************************************/

 void MQENTRY mqDeleteItem (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemIndex,  /* I: Item index */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqExecute Function -- Send Admin Command and Receive Reponse */
 /****************************************************************/

 void MQENTRY mqExecute (
   MQHCONN  Hconn,        /* I: Connection handle */
   MQLONG   Command,      /* I: Command identifier */
   MQHBAG   OptionsBag,   /* I: Handle of options bag */
   MQHBAG   AdminBag,     /* I: Handle of admin bag */
   MQHBAG   ResponseBag,  /* I: Handle of response bag */
   MQHOBJ   AdminQ,       /* I: Handle of admin queue */
   MQHOBJ   ResponseQ,    /* I: Handle of response queue */
   PMQLONG  pCompCode,    /* OC: Completion code */
   PMQLONG  pReason);     /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqGetBag Function -- Receive PCF Message into Bag            */
 /****************************************************************/

 void MQENTRY mqGetBag (
   MQHCONN  Hconn,        /* I: Connection handle */
   MQHOBJ   Hobj,         /* I: Queue handle */
   PMQVOID  pMsgDesc,     /* IO: Message descriptor */
   PMQVOID  pGetMsgOpts,  /* IO: Get-message options */
   MQHBAG   Bag,          /* IO: Handle of bag to contain message */
   PMQLONG  pCompCode,    /* OC: Completion code */
   PMQLONG  pReason);     /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireBag Function -- Inquire Handle in Bag               */
 /****************************************************************/

 void MQENTRY mqInquireBag (
   MQHBAG   Bag,         /* I: Bag handle */
   MQLONG   Selector,    /* I: Item selector */
   MQLONG   ItemIndex,   /* I: Item index */
   PMQHBAG  pItemValue,  /* O: Item value */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireByteString Function -- Inquire Byte String in Bag   */
 /****************************************************************/

 void MQENTRY mqInquireByteString (
   MQHBAG   Bag,                /* I: Bag handle */
   MQLONG   Selector,           /* I: Item selector */
   MQLONG   ItemIndex,          /* I: Item index */
   MQLONG   BufferLength,       /* IL: Length of buffer */
   PMQBYTE  pBuffer,            /* OB: Buffer to contain string */
   PMQLONG  pByteStringLength,  /* O: Length of byte string returned */
   PMQLONG  pCompCode,          /* OC: Completion code */
   PMQLONG  pReason);           /* OR: Reason code qualifying */
                                /* CompCode */


 /****************************************************************/
 /* mqInquireByteStringFilter Function -- Inquire Byte String    */
 /* Filter in Bag                                                */
 /****************************************************************/

 void MQENTRY mqInquireByteStringFilter (
   MQHBAG   Bag,                /* I: Bag handle */
   MQLONG   Selector,           /* I: Item selector */
   MQLONG   ItemIndex,          /* I: Item index */
   MQLONG   BufferLength,       /* IL: Length of buffer */
   PMQBYTE  pBuffer,            /* OB: Buffer to contain string */
   PMQLONG  pByteStringLength,  /* O: Length of byte string returned */
   PMQLONG  pOperator,          /* O: Item operator */
   PMQLONG  pCompCode,          /* OC: Completion code */
   PMQLONG  pReason);           /* OR: Reason code qualifying */
                                /* CompCode */


 /****************************************************************/
 /* mqInquireInteger Function -- Inquire Integer in Bag          */
 /****************************************************************/

 void MQENTRY mqInquireInteger (
   MQHBAG   Bag,         /* I: Bag handle */
   MQLONG   Selector,    /* I: Item selector */
   MQLONG   ItemIndex,   /* I: Item index */
   PMQLONG  pItemValue,  /* O: Item value */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireInteger64 Function -- Inquire 64-bit Integer in Bag */
 /****************************************************************/

 void MQENTRY mqInquireInteger64 (
   MQHBAG    Bag,         /* I: Bag handle */
   MQLONG    Selector,    /* I: Item selector */
   MQLONG    ItemIndex,   /* I: Item index */
   PMQINT64  pItemValue,  /* O: Item value */
   PMQLONG   pCompCode,   /* OC: Completion code */
   PMQLONG   pReason);    /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireIntegerFilter Function -- Inquire Integer Filter in */
 /* Bag                                                          */
 /****************************************************************/

 void MQENTRY mqInquireIntegerFilter (
   MQHBAG   Bag,         /* I: Bag handle */
   MQLONG   Selector,    /* I: Item selector */
   MQLONG   ItemIndex,   /* I: Item index */
   PMQLONG  pItemValue,  /* O: Item value */
   PMQLONG  pOperator,   /* O: Item operator */
   PMQLONG  pCompCode,   /* OC: Completion code */
   PMQLONG  pReason);    /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireItemInfo Function -- Inquire Attributes of Item in  */
 /* Bag                                                          */
 /****************************************************************/

 void MQENTRY mqInquireItemInfo (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   ItemIndex,     /* I: Item index */
   PMQLONG  pOutSelector,  /* O: Selector of item */
   PMQLONG  pItemType,     /* O: Data type of item */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireString Function -- Inquire String in Bag            */
 /****************************************************************/

 void MQENTRY mqInquireString (
   MQHBAG   Bag,              /* I: Bag handle */
   MQLONG   Selector,         /* I: Item selector */
   MQLONG   ItemIndex,        /* I: Item index */
   MQLONG   BufferLength,     /* IL: Length of buffer */
   PMQCHAR  pBuffer,          /* OB: Buffer to contain string */
   PMQLONG  pStringLength,    /* O: Length of string returned */
   PMQLONG  pCodedCharSetId,  /* O: Character-set identifier of */
                              /* string */
   PMQLONG  pCompCode,        /* OC: Completion code */
   PMQLONG  pReason);         /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqInquireStringFilter Function -- Inquire String Filter in   */
 /* Bag                                                          */
 /****************************************************************/

 void MQENTRY mqInquireStringFilter (
   MQHBAG   Bag,              /* I: Bag handle */
   MQLONG   Selector,         /* I: Item selector */
   MQLONG   ItemIndex,        /* I: Item index */
   MQLONG   BufferLength,     /* IL: Length of buffer */
   PMQCHAR  pBuffer,          /* OB: Buffer to contain string */
   PMQLONG  pStringLength,    /* O: Length of string returned */
   PMQLONG  pCodedCharSetId,  /* O: Character-set identifier of */
                              /* string */
   PMQLONG  pOperator,        /* O: Item operator */
   PMQLONG  pCompCode,        /* OC: Completion code */
   PMQLONG  pReason);         /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqPad Function -- Pad Null-terminated String with Blanks     */
 /****************************************************************/

 void MQENTRY mqPad (
   PMQCHAR  pString,       /* I: Null-terminated string to be padded */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* OB: Buffer to contain padded string */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqPutBag Function -- Send Bag as PCF Message                 */
 /****************************************************************/

 void MQENTRY mqPutBag (
   MQHCONN  Hconn,        /* I: Connection handle */
   MQHOBJ   Hobj,         /* I: Queue handle */
   PMQVOID  pMsgDesc,     /* IO: Message descriptor */
   PMQVOID  pPutMsgOpts,  /* IO: Put-message options */
   MQHBAG   Bag,          /* I: Handle of bag containing message data */
   PMQLONG  pCompCode,    /* OC: Completion code */
   PMQLONG  pReason);     /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetByteString Function -- Modify Byte String in Bag        */
 /****************************************************************/

 void MQENTRY mqSetByteString (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   ItemIndex,     /* I: Item index */
   MQLONG   BufferLength,  /* I: Length of buffer */
   PMQBYTE  pBuffer,       /* I: Buffer containing item value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetByteStringFilter Function -- Modify Byte String Filter  */
 /* in Bag                                                       */
 /****************************************************************/

 void MQENTRY mqSetByteStringFilter (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   ItemIndex,     /* I: Item index */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQBYTE  pBuffer,       /* IB: Buffer containing item value */
   MQLONG   Operator,      /* I: Item operator */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetInteger Function -- Modify Integer in Bag               */
 /****************************************************************/

 void MQENTRY mqSetInteger (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemIndex,  /* I: Item index */
   MQLONG   ItemValue,  /* I: Item value */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetInteger64 Function -- Modify 64-bit Integer in Bag      */
 /****************************************************************/

 void MQENTRY mqSetInteger64 (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemIndex,  /* I: Item index */
   MQINT64  ItemValue,  /* I: Item value */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetIntegerFilter Function -- Modify Integer Filter in Bag  */
 /****************************************************************/

 void MQENTRY mqSetIntegerFilter (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   Selector,   /* I: Item selector */
   MQLONG   ItemIndex,  /* I: Item index */
   MQLONG   ItemValue,  /* I: Item value */
   MQLONG   Operator,   /* I: Item operator */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetString Function -- Modify String in Bag                 */
 /****************************************************************/

 void MQENTRY mqSetString (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   ItemIndex,     /* I: Item index */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* IB: Buffer containing item value */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqSetStringFilter Function -- Modify String Filter in Bag    */
 /****************************************************************/

 void MQENTRY mqSetStringFilter (
   MQHBAG   Bag,           /* I: Bag handle */
   MQLONG   Selector,      /* I: Item selector */
   MQLONG   ItemIndex,     /* I: Item index */
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* IB: Buffer containing item value */
   MQLONG   Operator,      /* I: Item operator */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqTrim Function -- Replace Trailing Blanks with Null         */
 /* Character                                                    */
 /****************************************************************/

 void MQENTRY mqTrim (
   MQLONG   BufferLength,  /* IL: Length of buffer */
   PMQCHAR  pBuffer,       /* IB: Buffer containing blank-padded */
                           /* string */
   PMQCHAR  pString,       /* O: String with blanks discarded */
   PMQLONG  pCompCode,     /* OC: Completion code */
   PMQLONG  pReason);      /* OR: Reason code qualifying CompCode */


 /****************************************************************/
 /* mqTruncateBag Function -- Delete Trailing Items in Bag       */
 /****************************************************************/

 void MQENTRY mqTruncateBag (
   MQHBAG   Bag,        /* I: Bag handle */
   MQLONG   ItemCount,  /* I: Number of items to remain in bag */
   PMQLONG  pCompCode,  /* OC: Completion code */
   PMQLONG  pReason);   /* OR: Reason code qualifying CompCode */



 #if defined(__cplusplus)
   }
 #endif

 /****************************************************************/
 /*  End of CMQBC                                                */
 /****************************************************************/
 #endif  /* End of header file */
