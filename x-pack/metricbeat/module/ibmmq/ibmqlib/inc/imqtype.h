/*  @(#) MQMBID sn=p911-L190205 su=_ttBuwCkmEemoQ845UTHgVQ pn=include/imqtype.pre_h */

/*********************************************************************/
/*                                                                   */
/* Library:       IBM MQ                                       */
/* Component:     IMQB (IBM MQ C++ support)                    */
/* Part:          IMQTYPE.H                                          */
/*                                                                   */
/* Description:   This part identifies the compilation environment   */
/*                and sets manifest constants accordingly. Data      */
/*                types for IBM MQ C++ are defined.            */
/* <copyright                                                        */
/* notice="lm-source-program"                                        */
/* pids=""                                                           */
/* years="1994,2018"                                                 */
/* crc="2270496699" >                                                */
/* Licensed Materials - Property of IBM                              */
/*                                                                   */
/*                                                                   */
/*                                                                   */
/* (C) Copyright IBM Corp. 1994, 2018 All Rights Reserved.           */
/*                                                                   */
/* US Government Users Restricted Rights - Use, duplication or       */
/* disclosure restricted by GSA ADP Schedule Contract with           */
/* IBM Corp.                                                         */
/* </copyright>                                                      */
/*********************************************************************/

#if !defined(_IMQTYPE_H_)
#define _IMQTYPE_H_

#if defined( _MSC_VER ) || defined( __WINDOWS__ )

/*
 * Windows family
 */

#define __WIN__ 1

/*
 * Visual C++ 2 or better, VisualAge C++ 3.5
 */

#define __32__ 1
#define _CPP_VER 300

#elif defined( _AIX )

/*
 * IBM AIX
 */

#define __AIX__ 1
#define __UNIX__ 1
#define __32__ 1
#define _CPP_VER 300

#elif defined( __MVS__ )

/*
 * IBM MVS/ESA
 */

#define __32__ 1
#define _CPP_VER 300

#elif defined( __OS400__ )

/*
 * IBM AS/400
 */

#define __64__ 1
#define _CPP_VER 300

#elif defined( sun )

/*
 * SUN Solaris
 */

#define __UNIX__ 1
#define __32__ 1
#define _CPP_VER 210

#elif defined( hpux )

#define __UNIX__ 1
#define __32__ 1
#define _CPP_VER 210

#else

/*
 * Other OS. Defaults assumed
 */

#define __UNIX__ 1
#define __32__ 1
#define _CPP_VER 210

#endif

#if !defined( TRUE )
#define TRUE 1
#endif

#if !defined( FALSE )
#define FALSE 0
#endif

#if !defined( _IMQ_VER )
#define _IMQ_VER 230
#endif

#if !defined( MQCNO_VERSION_3 )
#define MQCNO_VERSION_3 3
#endif

/* --------------------------------------------------------------- */
/* IMQ_EXPORTCLASS used to be defined as __declspec(dllexport) for */
/* 64-bit Windows (this is from way a long time back, when a port  */
/* to Itanium was being considered).                               */
/*                                                                 */
/* This definition makes all of the C++ classes get exported, thus */
/* doing away with the need for a def file (and the horrible       */
/* maintenance it entails). However, not exporting methods with a  */
/* def file means that ordinals are not fixed from release to      */
/* release, so customer apps would need to be continually rebuilt. */
/*                                                                 */
/* Rather than remove the macro definition entirely, instead it is */
/* left, in place in all of the class definitions, meaning that if */
/* we ever did want to ship the C++ classes that way and move away */
/* from def files, the required task is simply to redefine the     */
/* macro (potentially on 32-bit Windows too).                      */
/* --------------------------------------------------------------- */
#ifndef _IMQ_EXPORTCLASS

/*
 * NB: Do NOT define _IMQ_EXPORTCLASS here
 */

#define IMQ_EXPORTCLASS
#endif

typedef unsigned char ImqBoolean ;

#ifndef MQRC_REOPEN_EXCL_INPUT_ERROR
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
#endif

#endif

