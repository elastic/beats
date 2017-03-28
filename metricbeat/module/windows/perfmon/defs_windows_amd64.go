package perfmon

const (
	ERROR_SUCCESS                 = 0x0
	PDH_STATUS_VALID_DATA         = 0x0
	PDH_STATUS_NEW_DATA           = 0x1
	PDH_NO_DATA                   = 0x800007d5
	PDH_STATUS_NO_OBJECT          = 0xc0000bb8
	PDH_STATUS_NO_COUNTER         = 0xc0000bb9
	PDH_STATUS_INVALID_DATA       = 0xc0000bba
	PDH_INVALID_HANDLE            = 0xc0000bbc
	PDH_INVALID_DATA              = 0xc0000bc6
	PDH_NO_MORE_DATA              = 0xc0000bcc
	PDH_CALC_NEGATIVE_DENOMINATOR = 0x800007d6
	PdhFmtDouble                  = 0x00000200
	PdhFmtLarge                   = 0x00000400
	PdhFmtLong                    = 0x00000100
)

type PdhCounterValue struct {
	CStatus   uint32
	Pad_cgo_0 [4]byte
	LongValue int32
	Pad_cgo_1 [4]byte
}
