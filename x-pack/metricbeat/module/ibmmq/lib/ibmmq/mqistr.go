package ibmmqi

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqstrc.h>
*/
import "C"

import (
	"fmt"
)

/*
   Convert MQCC/MQRC values into readable text using
   the functions introduced in cmqstrc.h in MQ V8004
*/
func mqstrerror(verb string, mqcc C.MQLONG, mqrc C.MQLONG) string {
	return fmt.Sprintf("%s: MQCC = %s [%d] MQRC = %s [%d]", verb,
		C.GoString(C.MQCC_STR(mqcc)), mqcc,
		C.GoString(C.MQRC_STR(mqrc)), mqrc)
}

/*
MQItoString returns a string representation of the MQI #define. Only a few of the
sets of constants are decoded here; see cmqstrc.h for a full set. Some of the
sets are aggregated, so that "RC" will return something from either the MQRC
or MQRCCF sets. These sets are related and do not overlap values.
*/
func MQItoString(class string, value int) string {
	s := ""
	v := C.MQLONG(value)

	switch class {
	case "BACF":
		s = C.GoString(C.MQBACF_STR(v))

	case "CA":
		s = C.GoString(C.MQCA_STR(v))
		if s == "" {
			s = C.GoString(C.MQCACF_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQCACH_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQCAMO_STR(v))
		}

	case "CC":
		s = C.GoString(C.MQCC_STR(v))
	case "CMD":
		s = C.GoString(C.MQCMD_STR(v))

	case "IA":
		s = C.GoString(C.MQIA_STR(v))
		if s == "" {
			s = C.GoString(C.MQIACF_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIACH_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIAMO_STR(v))
		}
		if s == "" {
			s = C.GoString(C.MQIAMO64_STR(v))
		}

	case "OT":
		s = C.GoString(C.MQOT_STR(v))

	case "RC":
		s = C.GoString(C.MQRC_STR(v))
		if s == "" {
			s = C.GoString(C.MQRCCF_STR(v))
		}
	case "CFT":
		s = C.GoString(C.MQCFT_STR(v))
	case "CHS":
		s = C.GoString(C.MQCHS_STR(v))
	case "CHT":
		s = C.GoString(C.MQCHT_STR(v))
	case "QT":
		s = C.GoString(C.MQQT_STR(v))
	case "NPM":
		s = C.GoString(C.MQNPM_STR(v))
	}
	return s
}
