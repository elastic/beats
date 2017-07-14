package file

type SecurityDescriptor struct{}

type ACL struct {
	AclRevision uint8
	Sbz1        uint8
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

// Types of Windows objects that support security.
type ObjectType uint32

const (
	UnknownObjectType ObjectType = iota
	FileObject
	Service
	Printer
	RegistryKey
	LMShare
	KernelObject
	WindowObject
	DSObject
	DSObjectAll
	ProviderDefinedObject
	WmiGuidObject
	RegistryWow64_32Key
)

type SecurityInformation uint32

const (
	OwnerSecurityInformation             SecurityInformation = 0x00000001
	GroupSecurityInformation             SecurityInformation = 0x00000002
	DaclSecurityInformation              SecurityInformation = 0x00000004
	SaclSecurityInformation              SecurityInformation = 0x00000008
	LabelSecurityInformation             SecurityInformation = 0x00000010
	AttributeSecurityInformation         SecurityInformation = 0x00000020
	ScopeSecurityInformation             SecurityInformation = 0x00000040
	ProcessTrustLabelSecurityInformation SecurityInformation = 0x00000080
	BackupSecurityInformation            SecurityInformation = 0x00010000
	ProtectedDaclSecurityInformation     SecurityInformation = 0x80000000
	ProtectedSaclSecurityInformation     SecurityInformation = 0x40000000
	UnprotectedDaclSecurityInformation   SecurityInformation = 0x20000000
	UnprotectedSaclSecurityInformation   SecurityInformation = 0x10000000
)

// Use "GOOS=windows go generate -v -x ." to generate the source.

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsecurity_windows.go security_windows.go

// Windows API calls
//sys GetSecurityInfo(handle syscall.Handle, objectType ObjectType, securityInformation SecurityInformation, ppsidOwner **syscall.SID, ppsidGroup **syscall.SID, ppDacl **ACL, ppSacl **ACL, ppSecurityDescriptor **SecurityDescriptor) (err error) [failretval!=0] = advapi32.GetSecurityInfo
