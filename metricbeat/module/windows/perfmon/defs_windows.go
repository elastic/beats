// +build ignore

package perfmon

/*
#include <windows.h>
#include <stdio.h>
#include <conio.h>
#include <pdh.h>
#include <pdhmsg.h>
#cgo LDFLAGS: -lpdh
*/
import "C"

const (
	ERROR_SUCCESS                 = C.ERROR_SUCCESS
	PDH_STATUS_VALID_DATA         = C.PDH_CSTATUS_VALID_DATA
	PDH_STATUS_NEW_DATA           = C.PDH_CSTATUS_NEW_DATA
	PDH_NO_DATA                   = C.PDH_NO_DATA
	PDH_STATUS_NO_OBJECT          = C.PDH_CSTATUS_NO_OBJECT
	PDH_STATUS_NO_COUNTER         = C.PDH_CSTATUS_NO_COUNTER
	PDH_STATUS_INVALID_DATA       = C.PDH_CSTATUS_INVALID_DATA
	PDH_INVALID_HANDLE            = C.PDH_INVALID_HANDLE
	PDH_INVALID_DATA              = C.PDH_INVALID_DATA
	PDH_NO_MORE_DATA              = C.PDH_NO_MORE_DATA
	PDH_CALC_NEGATIVE_DENOMINATOR = C.PDH_CALC_NEGATIVE_DENOMINATOR
	PdhFmtDouble                  = C.PDH_FMT_DOUBLE
	PdhFmtLarge                   = C.PDH_FMT_LARGE
	PdhFmtLong                    = C.PDH_FMT_LONG
)

type PdhCounterValue C.PDH_FMT_COUNTERVALUE
