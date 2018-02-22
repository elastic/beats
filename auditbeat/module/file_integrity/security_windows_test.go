package file_integrity

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestGetSecurityInfo(t *testing.T) {
	// Create a temp file that we will use in checking the owner.
	file, err := ioutil.TempFile("", "go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	// Get the file owner.
	var securityID *syscall.SID
	var securityDescriptor *SecurityDescriptor
	if err = GetSecurityInfo(syscall.Handle(file.Fd()), FileObject,
		OwnerSecurityInformation, &securityID, nil, nil, nil, &securityDescriptor); err != nil {
		t.Fatal(err)
	}

	_, err = securityID.String()
	assert.NoError(t, err)
	_, _, _, err = securityID.LookupAccount("")
	assert.NoError(t, err)

	// Freeing the security descriptor releases the memory used by the SID.
	_, err = syscall.LocalFree((syscall.Handle)(unsafe.Pointer(securityDescriptor)))
	if err != nil {
		t.Fatal(err)
	}
}
