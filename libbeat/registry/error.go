package registry

import "errors"

var (
	errStoreClosed  = errors.New("store is closed")
	errTxUnknownKey = errors.New("unknown key")
	errTxClosed     = errors.New("transaction is already closed")
)
