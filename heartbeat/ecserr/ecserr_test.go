package ecserr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const typ = "mytype"
const code = "mycode"
const message = "mymessage"

// A var since it's often used as a pointer
var stackTrace = "mystacktrace"

func TestEcsErrNewWithStack(t *testing.T) {
	e := NewECSErrWithStack(typ, code, message, &stackTrace)

	// Ensure that it implments the error interface
	var eErr error = e

	// check that wrapping it still includes the right message
	require.Equal(t, message, eErr.Error())
	require.Equal(t, message, e.Message)

	require.Equal(t, typ, e.Type)
	require.Equal(t, code, e.Code)
	require.Equal(t, stackTrace, *e.StackTrace)
}

func TestEcsErrNew(t *testing.T) {
	e := NewECSErr(typ, code, message)

	require.Equal(t, message, e.Message)
	require.Equal(t, typ, e.Type)
	require.Equal(t, code, e.Code)
}
