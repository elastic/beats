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
package ibmmqi

/*
#include <stdlib.h>
#include <stdio.h>
#include <cmqc.h>

extern void MQCALLBACK_Go(MQHCONN, MQMD *, MQGMO *, PMQVOID, MQCBC *);
void MQCALLBACK_C(MQHCONN hc,MQMD *md,MQGMO *gmo,PMQVOID buf,MQCBC *cbc) {
  MQCALLBACK_Go(hc,md,gmo,buf,cbc);
}
*/
import "C"

// This file exists purely to provide the linkage between a C callback function
// and Go for the MQCB/MQCTL asynchronous message consumer. The MQCALLBACK_C function
// has to be in a separate file to avoid "multiple definition" errors from the
// CGo compilation process. It looks like it is just a comment above, but
// that section of the file is processed by CGo.
