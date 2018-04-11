package rule

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/go-libaudit/auparse"
)

//go:generate sh -c "go tool cgo -godefs defs_kernel_types.go > zkernel_types.go && gofmt -w zkernel_types.go"

const (
	maxKeyLength = 256  // AUDIT_MAX_KEY_LEN
	pathMax      = 4096 // PATH_MAX
	keySeparator = 0x01 // AUDIT_KEY_SEPARATOR
)

// Build builds an audit rule.
func Build(rule Rule) (WireFormat, error) {
	data := &ruleData{allSyscalls: true}
	var err error

	switch v := rule.(type) {
	case *SyscallRule:
		if err = addList(data, v.List); err != nil {
			return nil, err
		}
		if err = addAction(data, v.Action); err != nil {
			return nil, err
		}

		for _, filter := range v.Filters {
			switch filter.Type {
			case ValueFilterType:
				if err = addFilter(data, filter.LHS, filter.Comparator, filter.RHS); err != nil {
					return nil, errors.Wrapf(err, "failed to add filter '%v'", filter)
				}
			case InterFieldFilterType:
				if err = addInterFieldComparator(data, filter.LHS, filter.Comparator, filter.RHS); err != nil {
					return nil, errors.Wrapf(err, "failed to add interfield comparison '%v'", filter)
				}
			}
		}

		for _, syscall := range v.Syscalls {
			if err = addSyscall(data, syscall); err != nil {
				return nil, errors.Wrapf(err, "failed to add syscall '%v'", syscall)
			}
		}

		if err = addKeys(data, v.Keys); err != nil {
			return nil, err
		}

	case *FileWatchRule:
		if err = addFileWatch(data, v); err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("unknown rule type: %T", v)
	}

	ard, err := data.toAuditRuleData()
	if err != nil {
		return nil, err
	}

	return ard.toWireFormat(), nil
}

func addFileWatch(data *ruleData, rule *FileWatchRule) error {
	path := filepath.Clean(rule.Path)

	if !filepath.IsAbs(path) {
		return errors.Errorf("path must be absolute: %v", path)
	}

	watchType := "path"
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		watchType = "dir"
	}

	var perms string
	if len(rule.Permissions) == 0 {
		perms = "rwxa"
	} else {
		perms = ""
		for _, p := range rule.Permissions {
			switch p {
			case ReadAccessType:
				perms += "r"
			case WriteAccessType:
				perms += "w"
			case ExecuteAccessType:
				perms += "x"
			case AttributeChangeAccessType:
				perms += "a"
			}
		}
	}

	// Build rule.
	data.flags = exitFilter
	data.action = alwaysAction
	data.allSyscalls = true
	if err := addFilter(data, watchType, "=", path); err != nil {
		return err
	}
	if err := addFilter(data, "perm", "=", perms); err != nil {
		return err
	}
	if err := addKeys(data, rule.Keys); err != nil {
		return err
	}
	return nil
}

func addKeys(data *ruleData, keys []string) error {
	if len(keys) > 0 {
		key := strings.Join(keys, string(keySeparator))
		if err := addFilter(data, "key", "=", key); err != nil {
			return errors.Wrapf(err, "failed to add keys [%v]", strings.Join(keys, ","))
		}
	}
	return nil
}

type ruleData struct {
	flags  filter
	action action

	allSyscalls bool
	syscalls    []uint32

	fields     []field
	values     []uint32
	fieldFlags []operator

	strings []string

	arch string
}

func (d ruleData) toAuditRuleData() (*auditRuleData, error) {
	rule := &auditRuleData{
		Flags:      d.flags,
		Action:     d.action,
		FieldCount: uint32(len(d.fields)),
	}

	if d.allSyscalls {
		for i := 0; i < len(rule.Mask)-1; i++ {
			rule.Mask[i] = 0xFFFFFFFF
		}
	} else {
		for _, syscallNum := range d.syscalls {
			word := syscallNum / 32
			bit := 1 << (syscallNum - (word * 32))
			if int(word) > len(rule.Mask) {
				return nil, errors.Errorf("invalid syscall number %v", syscallNum)
			}
			rule.Mask[word] |= uint32(bit)
		}
	}

	if len(d.fields) > len(rule.Fields) {
		return nil, errors.Errorf("too many filters and keys, only %v total are supported", len(rule.Fields))
	}
	for i := range d.fields {
		rule.Fields[i] = d.fields[i]
		rule.FieldFlags[i] = d.fieldFlags[i]
		rule.Values[i] = d.values[i]
	}

	for _, s := range d.strings {
		rule.Buf = append(rule.Buf, []byte(s)...)
	}
	rule.BufLen = uint32(len(rule.Buf))

	return rule, nil
}

func addList(rule *ruleData, list string) error {
	switch list {
	case "exit":
		rule.flags = exitFilter
	case "task":
		rule.flags = taskFilter
	case "user":
		rule.flags = userFilter
	case "exclude":
		rule.flags = excludeFilter
	default:
		return errors.Errorf("invalid list '%v'", list)
	}

	return nil
}

func addAction(rule *ruleData, action string) error {
	switch action {
	case "always":
		rule.action = alwaysAction
	case "never":
		rule.action = neverAction
	default:
		return errors.Errorf("invalid action '%v'", action)
	}

	return nil
}

// Convert name to number.
// Look for conditions when arch needs to be specified.
// Add syscall bit to mask.
func addSyscall(rule *ruleData, syscall string) error {
	if syscall == "all" {
		rule.allSyscalls = true
		return nil
	}
	rule.allSyscalls = false

	syscallNum, err := strconv.Atoi(syscall)
	if nerr, ok := err.(*strconv.NumError); ok {
		if nerr.Err != strconv.ErrSyntax {
			return errors.Wrapf(err, "failed to parse syscall number '%v'", syscall)
		}

		arch := rule.arch
		if arch == "" {
			arch, err = getRuntimeArch()
			if err != nil {
				return errors.Wrap(err, "failed to add syscall")
			}
		}

		// Convert name to number.
		table, found := reverseSyscall[arch]
		if !found {
			return errors.Errorf("syscall table not found for arch %v", arch)
		}

		syscallNum, found = table[syscall]
		if !found {
			return errors.Errorf("unknown syscall '%v' for arch %v", syscall, arch)
		}
	}

	rule.syscalls = append(rule.syscalls, uint32(syscallNum))
	return nil
}

func addInterFieldComparator(rule *ruleData, lhs, comparator, rhs string) error {
	op, found := operatorsTable[comparator]
	if !found {
		return errors.Errorf("invalid operator '%v'", comparator)
	}

	switch op {
	case equalOperator, notEqualOperator:
	default:
		return errors.Errorf("invalid operator '%v', only '=' or '!=' can be used", comparator)
	}

	leftField, found := fieldsTable[lhs]
	if !found {
		return errors.Errorf("invalid field '%v' on left", lhs)
	}

	rightField, found := fieldsTable[rhs]
	if !found {
		return errors.Errorf("invalid field '%v' on right", lhs)
	}

	table, found := comparisonsTable[leftField]
	if !found {
		return errors.Errorf("field '%v' cannot be used in an interfield comparison", lhs)
	}

	comparison, found := table[rightField]
	if !found {
		return errors.Errorf("field '%v' cannot be used in an interfield comparison", rhs)
	}

	rule.fields = append(rule.fields, fieldCompare)
	rule.fieldFlags = append(rule.fieldFlags, op)
	rule.values = append(rule.values, uint32(comparison))

	return nil
}

func addFilter(rule *ruleData, lhs, comparator, rhs string) error {
	op, found := operatorsTable[comparator]
	if !found {
		return errors.Errorf("invalid operator '%v'", comparator)
	}

	field, found := fieldsTable[lhs]
	if !found {
		return errors.Errorf("invalid field '%v' on left", lhs)
	}

	// Only newer kernel versions support exclude for credential types. Older
	// kernels only support exclude on the msgtype field.
	if rule.flags == excludeFilter {
		switch field {
		case pidField, uidField, gidField, auidField, msgTypeField,
			subjectUserField, subjectRoleField, subjectTypeField,
			subjectSensitivityField, subjectClearanceField:
		default:
			return errors.Errorf("field '%v' cannot be used the exclude flag", lhs)
		}
	}

	switch field {
	case uidField, euidField, suidField, fsuidField, auidField, objectUIDField:
		// Convert RHS to number.
		// Or attempt to lookup the name to get the number.
		uid, err := getUID(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, uid)
	case gidField, egidField, sgidField, fsgidField, objectGIDField:
		gid, err := getGID(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, gid)
	case exitField:
		// Flag must be FilterExit.
		if rule.flags != exitFilter {
			return errors.New("exit filter can only be applied to syscall exit")
		}
		exitCode, err := getExitCode(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, uint32(exitCode))
	case msgTypeField:
		// Flag must be exclude or user.
		if rule.flags != userFilter && rule.flags != excludeFilter {
			return errors.New("msgtype filter can only be applied to the user or exclude lists")
		}
		msgType, err := getAuditMsgType(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, msgType)
	case objectUserField, objectRoleField, objectTypeField, objectLevelLowField,
		objectLevelHighField, pathField, dirField:
		// Flag must be FilterExit.
		if rule.flags != exitFilter {
			return errors.Errorf("%v filter can only be applied to the syscall exit", lhs)
		}
		fallthrough
	case subjectUserField, subjectRoleField, subjectTypeField,
		subjectSensitivityField, subjectClearanceField, keyField, exeField:
		// Add string to strings.
		if field == keyField && len(rhs) > maxKeyLength {
			return errors.Errorf("%v cannot be longer than %v", lhs, maxKeyLength)
		} else if len(rhs) > pathMax {
			return errors.Errorf("%v cannot be longer than %v", lhs, pathMax)
		}
		rule.values = append(rule.values, uint32(len(rhs)))
		rule.strings = append(rule.strings, rhs)
	case archField:
		// Arch should come before syscall.
		// Arch only supports = and !=.
		if op != equalOperator && op != notEqualOperator {
			return errors.Errorf("arch only supports the = and != operators")
		}
		// Or convert name to arch or validate given arch.
		archName, arch, err := getArch(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, arch)
		rule.arch = archName
	case permField:
		// Perm is only valid for exit.
		if rule.flags != exitFilter {
			return errors.Errorf("perm filter can only be applied to the syscall exit")
		}
		// Perm is only valid for =.
		if op != equalOperator {
			return errors.Errorf("perm only support the = operator")
		}
		perm, err := getPerm(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, perm)
	case filetypeField:
		// Filetype is only valid for exit.
		if rule.flags != exitFilter {
			return errors.Errorf("filetype filter can only be applied to the syscall exit")
		}
		filetype, err := getFiletype(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, uint32(filetype))
	case arg0Field, arg1Field, arg2Field, arg3Field:
		// Convert RHS to a number.
		arg, err := parseNum(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, arg)
	//case SessionIDField:
	case inodeField:
		// Flag must be FilterExit.
		if rule.flags != exitFilter {
			return errors.Errorf("inode filter can only be applied to the syscall exit")
		}
		// Comparator must be = or !=.
		if op != equalOperator && op != notEqualOperator {
			return errors.Errorf("inode only supports the = and != operators")
		}
		// Convert RHS to number.
		inode, err := parseNum(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, inode)
	case devMajorField, devMinorField, successField, ppidField:
		// Flag must be FilterExit.
		if rule.flags != exitFilter {
			return errors.Errorf("%v filter can only be applied to the syscall exit", lhs)
		}
		fallthrough
	default:
		// Convert RHS to number.
		num, err := parseNum(rhs)
		if err != nil {
			return err
		}
		rule.values = append(rule.values, num)
	}

	rule.fields = append(rule.fields, field)
	rule.fieldFlags = append(rule.fieldFlags, op)
	return nil
}

func getUID(uid string) (uint32, error) {
	if uid == "unset" || uid == "-1" {
		return 4294967295, nil
	}

	v, err := strconv.ParseUint(uid, 10, 32)
	if nerr, ok := err.(*strconv.NumError); ok {
		if nerr.Err != strconv.ErrSyntax {
			return 0, errors.Wrapf(err, "failed to parse uid '%v'", uid)
		}

		u, err := user.Lookup(uid)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to convert user '%v' to a numeric ID", uid)
		}

		v, err = strconv.ParseUint(u.Uid, 10, 32)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to parse uid '%v' belonging to user '%v'", u.Uid, u.Username)
		}
	}

	return uint32(v), nil
}

func getGID(gid string) (uint32, error) {
	v, err := strconv.ParseUint(gid, 10, 32)
	if nerr, ok := err.(*strconv.NumError); ok {
		if nerr.Err != strconv.ErrSyntax {
			return 0, errors.Wrapf(err, "failed to parse gid '%v'", gid)
		}

		g, err := user.LookupGroup(gid)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to convert group '%v' to a numeric ID", gid)
		}

		v, err = strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to parse gid '%v' belonging to group '%v'", g.Gid, g.Name)
		}
	}

	return uint32(v), nil
}

func getExitCode(exit string) (int32, error) {
	v, err := strconv.ParseInt(exit, 0, 32)
	if nerr, ok := err.(*strconv.NumError); ok {
		if nerr.Err != strconv.ErrSyntax {
			return 0, errors.Wrapf(err, "failed to parse exit code '%v'", exit)
		}

		sign := 1
		code := exit
		if strings.HasPrefix(exit, "-") {
			sign = -1
			code = exit[1:]
		}

		num, found := auparse.AuditErrnoToNum[code]
		if !found {
			return 0, errors.Errorf("failed to convert error to exit code '%v'", exit)
		}
		v = int64(sign * num)
	}

	return int32(v), nil
}

func getArch(arch string) (string, uint32, error) {
	var realArch = arch
	switch strings.ToLower(arch) {
	case "b64":
		runtimeArch, err := getRuntimeArch()
		if err != nil {
			return "", 0, err
		}

		switch runtimeArch {
		case "aarch64", "x86_64", "ppc":
			realArch = runtimeArch
		default:
			return "", 0, errors.Errorf("cannot use b64 on %v", runtimeArch)
		}
	case "b32":
		runtimeArch, err := getRuntimeArch()
		if err != nil {
			return "", 0, err
		}

		switch runtimeArch {
		case "arm", "i386":
			realArch = runtimeArch
		case "aarch64":
			realArch = "arm"
		case "x86_64":
			realArch = "i386"
		default:
			return "", 0, errors.Errorf("cannot use b32 on %v", runtimeArch)
		}
	}

	archValue, found := reverseArch[realArch]
	if !found {
		return "", 0, errors.Errorf("unknown arch '%v'", arch)
	}
	return realArch, archValue, nil
}

// getRuntimeArch returns the program's arch (not the machine's arch).
func getRuntimeArch() (string, error) {
	var arch string
	switch runtime.GOARCH {
	case "arm":
		arch = "arm"
	case "arm64":
		arch = "aarch64"
	case "386":
		arch = "i386"
	case "amd64":
		arch = "x86_64"
	case "ppc64", "ppc64le":
		arch = "ppc"
	case "s390":
		arch = "s390"
	case "s390x":
		arch = "s390x"
	case "mips", "mipsle", "mips64", "mips64le":
		fallthrough
	default:
		return "", errors.Errorf("unsupported arch: %v", runtime.GOARCH)
	}

	return arch, nil
}

func getAuditMsgType(msgType string) (uint32, error) {
	v, err := strconv.ParseUint(msgType, 0, 32)
	if nerr, ok := err.(*strconv.NumError); ok {
		if nerr.Err != strconv.ErrSyntax {
			return 0, errors.Wrapf(err, "failed to parse msgtype '%v'", msgType)
		}

		typ, err := auparse.GetAuditMessageType(msgType)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to convert msgtype '%v' to numeric value", msgType)
		}
		v = uint64(typ)
	}

	return uint32(v), nil
}

func getPerm(perm string) (uint32, error) {
	var permBits permission
	for _, p := range perm {
		switch p {
		case 'r':
			permBits |= readPerm
		case 'w':
			permBits |= writePerm
		case 'x':
			permBits |= execPerm
		case 'a':
			permBits |= attrPerm
		default:
			return 0, errors.Errorf("invalid permission access type '%v'", p)
		}
	}

	return uint32(permBits), nil
}

func getFiletype(filetype string) (filetype, error) {
	switch strings.ToLower(filetype) {
	case "file":
		return fileFiletype, nil
	case "dir":
		return dirFiletype, nil
	case "socket":
		return socketFiletype, nil
	case "symlink":
		return linkFiletype, nil
	case "char":
		return characterFiletype, nil
	case "block":
		return blockFiletype, nil
	case "fifo":
		return fifoFiletype, nil
	default:
		return 0, errors.Errorf("invalid filetype '%v'", filetype)
	}
}

func parseNum(num string) (uint32, error) {
	if strings.HasPrefix(num, "-") {
		v, err := strconv.ParseInt(num, 0, 32)
		return uint32(v), err
	}
	v, err := strconv.ParseUint(num, 0, 32)
	return uint32(v), err
}
