// Created by cgo -godefs - DO NOT EDIT
// cgo.exe -godefs defs_pdh_windows.go

package perfmon

type PdhErrno uintptr

const (
	PDH_CSTATUS_VALID_DATA                     PdhErrno = 0x0
	PDH_CSTATUS_NEW_DATA                       PdhErrno = 0x1
	PDH_CSTATUS_NO_MACHINE                     PdhErrno = 0x800007d0
	PDH_CSTATUS_NO_INSTANCE                    PdhErrno = 0x800007d1
	PDH_MORE_DATA                              PdhErrno = 0x800007d2
	PDH_CSTATUS_ITEM_NOT_VALIDATED             PdhErrno = 0x800007d3
	PDH_RETRY                                  PdhErrno = 0x800007d4
	PDH_NO_DATA                                PdhErrno = 0x800007d5
	PDH_CALC_NEGATIVE_DENOMINATOR              PdhErrno = 0x800007d6
	PDH_CALC_NEGATIVE_TIMEBASE                 PdhErrno = 0x800007d7
	PDH_CALC_NEGATIVE_VALUE                    PdhErrno = 0x800007d8
	PDH_DIALOG_CANCELLED                       PdhErrno = 0x800007d9
	PDH_END_OF_LOG_FILE                        PdhErrno = 0x800007da
	PDH_ASYNC_QUERY_TIMEOUT                    PdhErrno = 0x800007db
	PDH_CANNOT_SET_DEFAULT_REALTIME_DATASOURCE PdhErrno = 0x800007dc
	PDH_CSTATUS_NO_OBJECT                      PdhErrno = 0xc0000bb8
	PDH_CSTATUS_NO_COUNTER                     PdhErrno = 0xc0000bb9
	PDH_CSTATUS_INVALID_DATA                   PdhErrno = 0xc0000bba
	PDH_MEMORY_ALLOCATION_FAILURE              PdhErrno = 0xc0000bbb
	PDH_INVALID_HANDLE                         PdhErrno = 0xc0000bbc
	PDH_INVALID_ARGUMENT                       PdhErrno = 0xc0000bbd
	PDH_FUNCTION_NOT_FOUND                     PdhErrno = 0xc0000bbe
	PDH_CSTATUS_NO_COUNTERNAME                 PdhErrno = 0xc0000bbf
	PDH_CSTATUS_BAD_COUNTERNAME                PdhErrno = 0xc0000bc0
	PDH_INVALID_BUFFER                         PdhErrno = 0xc0000bc1
	PDH_INSUFFICIENT_BUFFER                    PdhErrno = 0xc0000bc2
	PDH_CANNOT_CONNECT_MACHINE                 PdhErrno = 0xc0000bc3
	PDH_INVALID_PATH                           PdhErrno = 0xc0000bc4
	PDH_INVALID_INSTANCE                       PdhErrno = 0xc0000bc5
	PDH_INVALID_DATA                           PdhErrno = 0xc0000bc6
	PDH_NO_DIALOG_DATA                         PdhErrno = 0xc0000bc7
	PDH_CANNOT_READ_NAME_STRINGS               PdhErrno = 0xc0000bc8
	PDH_LOG_FILE_CREATE_ERROR                  PdhErrno = 0xc0000bc9
	PDH_LOG_FILE_OPEN_ERROR                    PdhErrno = 0xc0000bca
	PDH_LOG_TYPE_NOT_FOUND                     PdhErrno = 0xc0000bcb
	PDH_NO_MORE_DATA                           PdhErrno = 0xc0000bcc
	PDH_ENTRY_NOT_IN_LOG_FILE                  PdhErrno = 0xc0000bcd
	PDH_DATA_SOURCE_IS_LOG_FILE                PdhErrno = 0xc0000bce
	PDH_DATA_SOURCE_IS_REAL_TIME               PdhErrno = 0xc0000bcf
	PDH_UNABLE_READ_LOG_HEADER                 PdhErrno = 0xc0000bd0
	PDH_FILE_NOT_FOUND                         PdhErrno = 0xc0000bd1
	PDH_FILE_ALREADY_EXISTS                    PdhErrno = 0xc0000bd2
	PDH_NOT_IMPLEMENTED                        PdhErrno = 0xc0000bd3
	PDH_STRING_NOT_FOUND                       PdhErrno = 0xc0000bd4
	PDH_UNABLE_MAP_NAME_FILES                  PdhErrno = 0x80000bd5
	PDH_UNKNOWN_LOG_FORMAT                     PdhErrno = 0xc0000bd6
	PDH_UNKNOWN_LOGSVC_COMMAND                 PdhErrno = 0xc0000bd7
	PDH_LOGSVC_QUERY_NOT_FOUND                 PdhErrno = 0xc0000bd8
	PDH_LOGSVC_NOT_OPENED                      PdhErrno = 0xc0000bd9
	PDH_WBEM_ERROR                             PdhErrno = 0xc0000bda
	PDH_ACCESS_DENIED                          PdhErrno = 0xc0000bdb
	PDH_LOG_FILE_TOO_SMALL                     PdhErrno = 0xc0000bdc
	PDH_INVALID_DATASOURCE                     PdhErrno = 0xc0000bdd
	PDH_INVALID_SQLDB                          PdhErrno = 0xc0000bde
	PDH_NO_COUNTERS                            PdhErrno = 0xc0000bdf
	PDH_SQL_ALLOC_FAILED                       PdhErrno = 0xc0000be0
	PDH_SQL_ALLOCCON_FAILED                    PdhErrno = 0xc0000be1
	PDH_SQL_EXEC_DIRECT_FAILED                 PdhErrno = 0xc0000be2
	PDH_SQL_FETCH_FAILED                       PdhErrno = 0xc0000be3
	PDH_SQL_ROWCOUNT_FAILED                    PdhErrno = 0xc0000be4
	PDH_SQL_MORE_RESULTS_FAILED                PdhErrno = 0xc0000be5
	PDH_SQL_CONNECT_FAILED                     PdhErrno = 0xc0000be6
	PDH_SQL_BIND_FAILED                        PdhErrno = 0xc0000be7
	PDH_CANNOT_CONNECT_WMI_SERVER              PdhErrno = 0xc0000be8
	PDH_PLA_COLLECTION_ALREADY_RUNNING         PdhErrno = 0xc0000be9
	PDH_PLA_ERROR_SCHEDULE_OVERLAP             PdhErrno = 0xc0000bea
	PDH_PLA_COLLECTION_NOT_FOUND               PdhErrno = 0xc0000beb
	PDH_PLA_ERROR_SCHEDULE_ELAPSED             PdhErrno = 0xc0000bec
	PDH_PLA_ERROR_NOSTART                      PdhErrno = 0xc0000bed
	PDH_PLA_ERROR_ALREADY_EXISTS               PdhErrno = 0xc0000bee
	PDH_PLA_ERROR_TYPE_MISMATCH                PdhErrno = 0xc0000bef
	PDH_PLA_ERROR_FILEPATH                     PdhErrno = 0xc0000bf0
	PDH_PLA_SERVICE_ERROR                      PdhErrno = 0xc0000bf1
	PDH_PLA_VALIDATION_ERROR                   PdhErrno = 0xc0000bf2
	PDH_PLA_VALIDATION_WARNING                 PdhErrno = 0x80000bf3
	PDH_PLA_ERROR_NAME_TOO_LONG                PdhErrno = 0xc0000bf4
	PDH_INVALID_SQL_LOG_FORMAT                 PdhErrno = 0xc0000bf5
	PDH_COUNTER_ALREADY_IN_QUERY               PdhErrno = 0xc0000bf6
	PDH_BINARY_LOG_CORRUPT                     PdhErrno = 0xc0000bf7
	PDH_LOG_SAMPLE_TOO_SMALL                   PdhErrno = 0xc0000bf8
	PDH_OS_LATER_VERSION                       PdhErrno = 0xc0000bf9
	PDH_OS_EARLIER_VERSION                     PdhErrno = 0xc0000bfa
	PDH_INCORRECT_APPEND_TIME                  PdhErrno = 0xc0000bfb
	PDH_UNMATCHED_APPEND_COUNTER               PdhErrno = 0xc0000bfc
	PDH_SQL_ALTER_DETAIL_FAILED                PdhErrno = 0xc0000bfd
	PDH_QUERY_PERF_DATA_TIMEOUT                PdhErrno = 0xc0000bfe
)

var pdhErrors = map[PdhErrno]struct{}{
	PDH_CSTATUS_VALID_DATA:                     struct{}{},
	PDH_CSTATUS_NEW_DATA:                       struct{}{},
	PDH_CSTATUS_NO_MACHINE:                     struct{}{},
	PDH_CSTATUS_NO_INSTANCE:                    struct{}{},
	PDH_MORE_DATA:                              struct{}{},
	PDH_CSTATUS_ITEM_NOT_VALIDATED:             struct{}{},
	PDH_RETRY:                                  struct{}{},
	PDH_NO_DATA:                                struct{}{},
	PDH_CALC_NEGATIVE_DENOMINATOR:              struct{}{},
	PDH_CALC_NEGATIVE_TIMEBASE:                 struct{}{},
	PDH_CALC_NEGATIVE_VALUE:                    struct{}{},
	PDH_DIALOG_CANCELLED:                       struct{}{},
	PDH_END_OF_LOG_FILE:                        struct{}{},
	PDH_ASYNC_QUERY_TIMEOUT:                    struct{}{},
	PDH_CANNOT_SET_DEFAULT_REALTIME_DATASOURCE: struct{}{},
	PDH_CSTATUS_NO_OBJECT:                      struct{}{},
	PDH_CSTATUS_NO_COUNTER:                     struct{}{},
	PDH_CSTATUS_INVALID_DATA:                   struct{}{},
	PDH_MEMORY_ALLOCATION_FAILURE:              struct{}{},
	PDH_INVALID_HANDLE:                         struct{}{},
	PDH_INVALID_ARGUMENT:                       struct{}{},
	PDH_FUNCTION_NOT_FOUND:                     struct{}{},
	PDH_CSTATUS_NO_COUNTERNAME:                 struct{}{},
	PDH_CSTATUS_BAD_COUNTERNAME:                struct{}{},
	PDH_INVALID_BUFFER:                         struct{}{},
	PDH_INSUFFICIENT_BUFFER:                    struct{}{},
	PDH_CANNOT_CONNECT_MACHINE:                 struct{}{},
	PDH_INVALID_PATH:                           struct{}{},
	PDH_INVALID_INSTANCE:                       struct{}{},
	PDH_INVALID_DATA:                           struct{}{},
	PDH_NO_DIALOG_DATA:                         struct{}{},
	PDH_CANNOT_READ_NAME_STRINGS:               struct{}{},
	PDH_LOG_FILE_CREATE_ERROR:                  struct{}{},
	PDH_LOG_FILE_OPEN_ERROR:                    struct{}{},
	PDH_LOG_TYPE_NOT_FOUND:                     struct{}{},
	PDH_NO_MORE_DATA:                           struct{}{},
	PDH_ENTRY_NOT_IN_LOG_FILE:                  struct{}{},
	PDH_DATA_SOURCE_IS_LOG_FILE:                struct{}{},
	PDH_DATA_SOURCE_IS_REAL_TIME:               struct{}{},
	PDH_UNABLE_READ_LOG_HEADER:                 struct{}{},
	PDH_FILE_NOT_FOUND:                         struct{}{},
	PDH_FILE_ALREADY_EXISTS:                    struct{}{},
	PDH_NOT_IMPLEMENTED:                        struct{}{},
	PDH_STRING_NOT_FOUND:                       struct{}{},
	PDH_UNABLE_MAP_NAME_FILES:                  struct{}{},
	PDH_UNKNOWN_LOG_FORMAT:                     struct{}{},
	PDH_UNKNOWN_LOGSVC_COMMAND:                 struct{}{},
	PDH_LOGSVC_QUERY_NOT_FOUND:                 struct{}{},
	PDH_LOGSVC_NOT_OPENED:                      struct{}{},
	PDH_WBEM_ERROR:                             struct{}{},
	PDH_ACCESS_DENIED:                          struct{}{},
	PDH_LOG_FILE_TOO_SMALL:                     struct{}{},
	PDH_INVALID_DATASOURCE:                     struct{}{},
	PDH_INVALID_SQLDB:                          struct{}{},
	PDH_NO_COUNTERS:                            struct{}{},
	PDH_SQL_ALLOC_FAILED:                       struct{}{},
	PDH_SQL_ALLOCCON_FAILED:                    struct{}{},
	PDH_SQL_EXEC_DIRECT_FAILED:                 struct{}{},
	PDH_SQL_FETCH_FAILED:                       struct{}{},
	PDH_SQL_ROWCOUNT_FAILED:                    struct{}{},
	PDH_SQL_MORE_RESULTS_FAILED:                struct{}{},
	PDH_SQL_CONNECT_FAILED:                     struct{}{},
	PDH_SQL_BIND_FAILED:                        struct{}{},
	PDH_CANNOT_CONNECT_WMI_SERVER:              struct{}{},
	PDH_PLA_COLLECTION_ALREADY_RUNNING:         struct{}{},
	PDH_PLA_ERROR_SCHEDULE_OVERLAP:             struct{}{},
	PDH_PLA_COLLECTION_NOT_FOUND:               struct{}{},
	PDH_PLA_ERROR_SCHEDULE_ELAPSED:             struct{}{},
	PDH_PLA_ERROR_NOSTART:                      struct{}{},
	PDH_PLA_ERROR_ALREADY_EXISTS:               struct{}{},
	PDH_PLA_ERROR_TYPE_MISMATCH:                struct{}{},
	PDH_PLA_ERROR_FILEPATH:                     struct{}{},
	PDH_PLA_SERVICE_ERROR:                      struct{}{},
	PDH_PLA_VALIDATION_ERROR:                   struct{}{},
	PDH_PLA_VALIDATION_WARNING:                 struct{}{},
	PDH_PLA_ERROR_NAME_TOO_LONG:                struct{}{},
	PDH_INVALID_SQL_LOG_FORMAT:                 struct{}{},
	PDH_COUNTER_ALREADY_IN_QUERY:               struct{}{},
	PDH_BINARY_LOG_CORRUPT:                     struct{}{},
	PDH_LOG_SAMPLE_TOO_SMALL:                   struct{}{},
	PDH_OS_LATER_VERSION:                       struct{}{},
	PDH_OS_EARLIER_VERSION:                     struct{}{},
	PDH_INCORRECT_APPEND_TIME:                  struct{}{},
	PDH_UNMATCHED_APPEND_COUNTER:               struct{}{},
	PDH_SQL_ALTER_DETAIL_FAILED:                struct{}{},
	PDH_QUERY_PERF_DATA_TIMEOUT:                struct{}{},
}

type PdhCounterFormat uint32

const (
	PdhFmtDouble PdhCounterFormat = 0x200

	PdhFmtLarge PdhCounterFormat = 0x400

	PdhFmtLong PdhCounterFormat = 0x100

	PdhFmtNoScale PdhCounterFormat = 0x1000

	PdhFmtNoCap100 PdhCounterFormat = 0x8000

	PdhFmtMultiply1000 PdhCounterFormat = 0x2000
)

type PdhCounterValue struct {
	CStatus   uint32
	Pad_cgo_0 [4]byte
	LongValue int32
	Pad_cgo_1 [4]byte
}

type PdhRawCounter struct {
	CStatus     uint32
	TimeStamp   PdhFileTime
	Pad_cgo_0   [4]byte
	FirstValue  int64
	SecondValue int64
	MultiCount  uint32
	Pad_cgo_1   [4]byte
}

type PdhFileTime struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}
