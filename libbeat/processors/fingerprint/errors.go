package fingerprint

import (
	"errors"
	"fmt"
)

var errNoFields = errors.New("must specify at least one field")

type (
	errUnknownEncoding    struct{ encoding string }
	errUnknownMethod      struct{ method string }
	errConfigUnpack       struct{ cause error }
	errComputeFingerprint struct{ cause error }
)

func makeErrUnknownEncoding(encoding string) errUnknownEncoding {
	return errUnknownEncoding{encoding}
}
func (e errUnknownEncoding) Error() string {
	return fmt.Sprintf("invalid encoding [%s]", e.encoding)
}

func makeErrUnknownMethod(method string) errUnknownMethod {
	return errUnknownMethod{method}
}
func (e errUnknownMethod) Error() string {
	return fmt.Sprintf("invalid fingerprinting method [%s]", e.method)
}

func makeErrConfigUnpack(cause error) errConfigUnpack {
	return errConfigUnpack{cause}
}
func (e errConfigUnpack) Error() string {
	return fmt.Sprintf("failed to unpack %v processor configuration: %v", processorName, e.cause)
}

func makeErrComputeFingerprint(cause error) errComputeFingerprint {
	return errComputeFingerprint{cause}
}
func (e errComputeFingerprint) Error() string {
	return fmt.Sprintf("failed to compute fingerprint: %v", e.cause)
}
