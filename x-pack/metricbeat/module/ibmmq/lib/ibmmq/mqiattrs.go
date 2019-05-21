package ibmmqi

/*
  Copyright (c) IBM Corporation 2018

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

*/
import "C"

/*
 * This file deals with the lengths of attributes that may be processed
 * by the MQSET/MQINQ calls. Only a small set of the object attributes
 * are supported by MQINQ (and even fewer for MQSET) so it's reasonable
 * to list them all here
 */
var mqInqLength = map[int32]int32{
	C.MQCA_ALTERATION_DATE:       C.MQ_DATE_LENGTH,
	C.MQCA_ALTERATION_TIME:       C.MQ_TIME_LENGTH,
	C.MQCA_APPL_ID:               C.MQ_PROCESS_APPL_ID_LENGTH,
	C.MQCA_BACKOUT_REQ_Q_NAME:    C.MQ_Q_NAME_LENGTH,
	C.MQCA_BASE_Q_NAME:           C.MQ_Q_NAME_LENGTH,
	C.MQCA_CF_STRUC_NAME:         C.MQ_CF_STRUC_NAME_LENGTH,
	C.MQCA_CHANNEL_AUTO_DEF_EXIT: C.MQ_EXIT_NAME_LENGTH,
	C.MQCA_CHINIT_SERVICE_PARM:   C.MQ_CHINIT_SERVICE_PARM_LENGTH,
	C.MQCA_CLUSTER_NAME:          C.MQ_CLUSTER_NAME_LENGTH,
	C.MQCA_CLUSTER_NAMELIST:      C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_CLUSTER_WORKLOAD_DATA: C.MQ_EXIT_DATA_LENGTH,
	C.MQCA_CLUSTER_WORKLOAD_EXIT: C.MQ_EXIT_NAME_LENGTH,
	C.MQCA_COMMAND_INPUT_Q_NAME:  C.MQ_Q_NAME_LENGTH,
	C.MQCA_CREATION_DATE:         C.MQ_DATE_LENGTH,
	C.MQCA_CREATION_TIME:         C.MQ_TIME_LENGTH,
	C.MQCA_DEAD_LETTER_Q_NAME:    C.MQ_Q_NAME_LENGTH,
	C.MQCA_DEF_XMIT_Q_NAME:       C.MQ_Q_NAME_LENGTH,
	C.MQCA_DNS_GROUP:             C.MQ_DNS_GROUP_NAME_LENGTH,
	C.MQCA_ENV_DATA:              C.MQ_PROCESS_ENV_DATA_LENGTH,
	C.MQCA_IGQ_USER_ID:           C.MQ_USER_ID_LENGTH,
	C.MQCA_INITIATION_Q_NAME:     C.MQ_Q_NAME_LENGTH,
	C.MQCA_INSTALLATION_DESC:     C.MQ_INSTALLATION_DESC_LENGTH,
	C.MQCA_INSTALLATION_NAME:     C.MQ_INSTALLATION_NAME_LENGTH,
	C.MQCA_INSTALLATION_PATH:     C.MQ_INSTALLATION_PATH_LENGTH,
	C.MQCA_LU62_ARM_SUFFIX:       C.MQ_ARM_SUFFIX_LENGTH,
	C.MQCA_LU_GROUP_NAME:         C.MQ_LU_NAME_LENGTH,
	C.MQCA_LU_NAME:               C.MQ_LU_NAME_LENGTH,
	C.MQCA_NAMELIST_DESC:         C.MQ_NAMELIST_DESC_LENGTH,
	C.MQCA_NAMELIST_NAME:         C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_NAMES:                 C.MQ_OBJECT_NAME_LENGTH * 256, // Maximum length to allocate
	C.MQCA_PARENT:                C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_PROCESS_DESC:          C.MQ_PROCESS_DESC_LENGTH,
	C.MQCA_PROCESS_NAME:          C.MQ_PROCESS_NAME_LENGTH,
	C.MQCA_Q_DESC:                C.MQ_Q_DESC_LENGTH,
	C.MQCA_Q_MGR_DESC:            C.MQ_Q_MGR_DESC_LENGTH,
	C.MQCA_Q_MGR_IDENTIFIER:      C.MQ_Q_MGR_IDENTIFIER_LENGTH,
	C.MQCA_Q_MGR_NAME:            C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_Q_NAME:                C.MQ_Q_NAME_LENGTH,
	C.MQCA_QSG_NAME:              C.MQ_QSG_NAME_LENGTH,
	C.MQCA_REMOTE_Q_MGR_NAME:     C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_REMOTE_Q_NAME:         C.MQ_Q_NAME_LENGTH,
	C.MQCA_REPOSITORY_NAME:       C.MQ_Q_MGR_NAME_LENGTH,
	C.MQCA_REPOSITORY_NAMELIST:   C.MQ_NAMELIST_NAME_LENGTH,
	C.MQCA_STORAGE_CLASS:         C.MQ_STORAGE_CLASS_LENGTH,
	C.MQCA_TCP_NAME:              C.MQ_TCP_NAME_LENGTH,
	C.MQCA_TRIGGER_DATA:          C.MQ_TRIGGER_DATA_LENGTH,
	C.MQCA_USER_DATA:             C.MQ_PROCESS_USER_DATA_LENGTH,
	C.MQCA_XMIT_Q_NAME:           C.MQ_Q_NAME_LENGTH,
}

/*
 * Return how many char & int attributes are in the list of selectors, and the
 * maximum length of the buffer needed to return them from the MQI
 */
func getAttrInfo(attrs []int32) (int, int, int) {
	var charAttrLength = 0
	var charAttrCount = 0
	var intAttrCount = 0

	for i := 0; i < len(attrs); i++ {
		if v, ok := mqInqLength[attrs[i]]; ok {
			charAttrCount++
			charAttrLength += int(v)
		} else if attrs[i] >= C.MQIA_FIRST && attrs[i] <= C.MQIA_LAST {
			intAttrCount++
		}
	}
	return intAttrCount, charAttrCount, charAttrLength
}

func getAttrLength(attr int32) int {
	if v, ok := mqInqLength[attr]; ok {
		return int(v)
	} else {
		return 0
	}

}
