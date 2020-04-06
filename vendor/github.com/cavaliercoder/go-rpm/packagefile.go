package rpm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// A PackageFile is an RPM package definition loaded directly from the package
// file itself.
type PackageFile struct {
	Lead    Lead
	Headers []Header

	path     string
	fileSize uint64
	fileTime time.Time

	files []FileInfo // memoize .Files()
}

const (
	r_headerCount = 2
)

// ReadPackageFile reads a rpm package file from a stream and returns a pointer
// to it.
func ReadPackageFile(r io.Reader) (*PackageFile, error) {
	// See: http://ftp.rpm.org/max-rpm/s1-rpm-file-format-rpm-file-format.html
	p := &PackageFile{}

	// read the deprecated "lead"
	lead, err := ReadPackageLead(r)
	if err != nil {
		return nil, err
	}
	p.Lead = *lead

	// read signature and header headers
	p.Headers = make([]Header, r_headerCount)
	for i := 0; i < r_headerCount; i++ {
		h, err := ReadPackageHeader(r)
		if err != nil {
			return nil, err
		}

		// pad to next header except on last header
		if i < r_headerCount-1 {
			if _, err := io.CopyN(ioutil.Discard, r, int64(8-(h.Length%8))%8); err != nil {
				return nil, err
			}
		}

		p.Headers[i] = *h
	}

	return p, nil
}

// OpenPackageFile reads a rpm package from the file system or a URL and returns
// a pointer to it.
func OpenPackageFile(path string) (*PackageFile, error) {
	lc := strings.ToLower(path)
	if strings.HasPrefix(lc, "http://") || strings.HasPrefix(lc, "https://") {
		return openPackageURL(path)
	}

	return openPackageFile(path)
}

// openPackageFile reads package info from the file system
func openPackageFile(path string) (*PackageFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	p, err := ReadPackageFile(f)
	if err != nil {
		return nil, err
	}
	p.path = path
	p.fileSize = uint64(fi.Size())
	p.fileTime = fi.ModTime()
	return p, nil
}

// openPackageURL reads package info from a HTTP URL
func openPackageURL(path string) (*PackageFile, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	p, err := ReadPackageFile(resp.Body)
	if err != nil {
		return nil, err
	}
	p.path = path
	p.fileSize = uint64(resp.ContentLength)
	if lm := resp.Header.Get("Last-Modified"); len(lm) > 0 {
		t, _ := time.Parse(time.RFC1123, lm) // ignore malformed timestamps
		p.fileTime = t
	}
	return p, nil
}

// OpenPackageFiles reads all rpm packages with the .rpm suffix from the given
// directory on the file systems and returns a slice of pointers to the loaded
// packages.
func OpenPackageFiles(path string) ([]*PackageFile, error) {
	// read directory
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// list *.rpm files
	files := make([]string, 0)
	for _, f := range dir {
		if strings.HasSuffix(f.Name(), ".rpm") {
			files = append(files, filepath.Join(path, f.Name()))
		}
	}

	// read packages
	packages := make([]*PackageFile, len(files))
	for i, f := range files {
		p, err := OpenPackageFile(f)
		if err != nil {
			return nil, err
		}

		packages[i] = p
	}

	return packages, nil
}

// dependencies translates the given tag values into a slice of package
// relationships such as provides, conflicts, obsoletes and requires.
func (c *PackageFile) dependencies(nevrsTagID, flagsTagID, namesTagID, versionsTagID int) []Dependency {
	// TODO: Implement NEVRS tags
	// TODO: error handling

	flgs := c.GetInts(1, flagsTagID)
	names := c.GetStrings(1, namesTagID)
	vers := c.GetStrings(1, versionsTagID)
	deps := make([]Dependency, len(names))
	for i := 0; i < len(names); i++ {
		deps[i] = &dependency{
			flags:   int(flgs[i]),
			name:    names[i],
			version: vers[i],
		}
	}
	return deps
}

// String returns the package identifier in the form
// '[name]-[version]-[release].[architecture]'.
func (c *PackageFile) String() string {
	return fmt.Sprintf("%s-%s-%s.%s", c.Name(), c.Version(), c.Release(), c.Architecture())
}

// GetBytes returns the value of the given tag in the given header. Nil is
// returned if the header or tag do not exist, or it the tag exists but is a
// different type.
func (c *PackageFile) GetBytes(header, tag int) []byte {
	if c == nil || header >= len(c.Headers) {
		return nil
	}
	return c.Headers[header].Indexes.BytesByTag(tag)
}

// GetStrings returns the string values of the given tag in the given header.
// Nil is returned if the header or tag do not exist, or if the tag exists but
// is a different type.
func (c *PackageFile) GetStrings(header, tag int) []string {
	if c == nil || header >= len(c.Headers) {
		return nil
	}
	return c.Headers[header].Indexes.StringsByTag(tag)
}

// GetString returns the string value of the given tag in the given header. Nil
// is returned if the header or tag do not exist, or if the tag exists but is a
// different type.
func (c *PackageFile) GetString(header, tag int) string {
	if c == nil || header >= len(c.Headers) {
		return ""
	}
	return c.Headers[header].Indexes.StringByTag(tag)
}

// GetInts returns the int64 values of the given tag in the given header. Nil is
// returned if the header or tag do not exist, or if the tag exists but is a
// different type.
func (c *PackageFile) GetInts(header, tag int) []int64 {
	if c == nil || header >= len(c.Headers) {
		return nil
	}
	return c.Headers[header].Indexes.IntsByTag(tag)
}

// GetInt returns the int64 value of the given tag in the given header. Zero is
// returned if the header or tag do not exist, or if the tag exists but is a
// different type.
func (c *PackageFile) GetInt(header, tag int) int64 {
	if c == nil || header >= len(c.Headers) {
		return 0
	}
	return c.Headers[header].Indexes.IntByTag(tag)
}

// Path returns the path which was given to open a package file if it was opened
// with OpenPackageFile.
func (c *PackageFile) Path() string {
	return c.path
}

// FileTime returns the time at which the RPM package file was last modified if
// it was opened with OpenPackageFile.
func (c *PackageFile) FileTime() time.Time {
	return c.fileTime
}

// FileSize returns the size of the package file in bytes if it was opened with
// OpenPackageFile.
func (c *PackageFile) FileSize() uint64 {
	return c.fileSize
}

// Checksum computes and returns the SHA256 checksum (encoded in hexadecimal) of
// the package file.
//
// Checksum is a convenience function for tools that make use of package file
// SHA256 checksums. These might include many of the databases files created by
// the createrepo tool.
//
// Checksum reopens the package using the file path that was given via
// OpenPackageFile. If the package was opened with any other method, Checksum
// will return "File not found".
func (c *PackageFile) Checksum() (string, error) {
	if c.Path() == "" {
		return "", fmt.Errorf("File not found")
	}

	f, err := os.Open(c.Path())
	if err != nil {
		return "", err
	}
	defer f.Close()

	s := sha256.New()
	if _, err := io.Copy(s, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(s.Sum(nil)), nil
}

// ChecksumType returns "sha256"
func (c *PackageFile) ChecksumType() string {
	return "sha256"
}

func (c *PackageFile) GPGSignature() GPGSignature {
	return GPGSignature(c.GetBytes(0, 1002))
}

// For tag definitions, see:
// https://github.com/rpm-software-management/rpm/blob/master/lib/rpmtag.h#L61

func (c *PackageFile) Name() string {
	return c.GetString(1, 1000)
}

func (c *PackageFile) Version() string {
	return c.GetString(1, 1001)
}

func (c *PackageFile) Release() string {
	return c.GetString(1, 1002)
}

func (c *PackageFile) Epoch() int {
	return int(c.GetInt(1, 1003))
}

func (c *PackageFile) Requires() []Dependency {
	return c.dependencies(5041, 1048, 1049, 1050)
}

func (c *PackageFile) Provides() []Dependency {
	return c.dependencies(5042, 1112, 1047, 1113)
}

func (c *PackageFile) Conflicts() []Dependency {
	return c.dependencies(5044, 1053, 1054, 1055)
}

func (c *PackageFile) Obsoletes() []Dependency {
	return c.dependencies(5043, 1114, 1090, 1115)
}

// Files returns file information for each file that is installed by this RPM
// package.
func (c *PackageFile) Files() []FileInfo {
	if c.files != nil {
		return c.files
	}

	ixs := c.GetInts(1, 1116)
	names := c.GetStrings(1, 1117)
	dirs := c.GetStrings(1, 1118)
	modes := c.GetInts(1, 1030)
	sizes := c.GetInts(1, 1028)
	times := c.GetInts(1, 1034)
	owners := c.GetStrings(1, 1039)
	groups := c.GetStrings(1, 1040)
	digests := c.GetStrings(1, 1035)
	linknames := c.GetStrings(1, 1036)

	c.files = make([]FileInfo, len(names))
	for i := 0; i < len(names); i++ {
		c.files[i] = FileInfo{
			name:     dirs[ixs[i]] + names[i],
			mode:     fileModeFromInt64(modes[i]),
			size:     sizes[i],
			modTime:  time.Unix(times[i], 0),
			owner:    owners[i],
			group:    groups[i],
			digest:   digests[i],
			linkname: linknames[i],
		}
	}

	return c.files
}

// fileModeFromInt64 converts the 16 bit value returned from a typical
// unix/linux stat call to the bitmask that go uses to produce an os
// neutral representation.  It is incorrect to just cast the 16 bit
// value directly to a os.FileMode.  The result of stat is 4 bits to
// specify the type of the object, this is a value in the range 0 to
// 15, rather than a bitfield, 3 bits to note suid, sgid and sticky,
// and 3 sets of 3 bits for rwx permissions for user, group and other.
// An os.FileMode has the same 9 bits for permissions, but rather than
// using an enum for the type it has individual bits.  As a concrete
// example, a block device has the 1<<26 bit set (os.ModeDevice) in
// the os.FileMode, but has type 0x6000 (syscall.S_IFBLK). A regular
// file is represented in os.FileMode by not having any of the bits in
// os.ModeType set (i.e. is not a directory, is not a symlink, is not
// a named pipe...) whilst a regular file has value syscall.S_IFREG
// (0x8000) in the mode field from stat.
func fileModeFromInt64(mode int64) os.FileMode {
	fm := os.FileMode(mode & 0777)
	switch mode & syscall.S_IFMT {
	case syscall.S_IFBLK:
		fm |= os.ModeDevice
	case syscall.S_IFCHR:
		fm |= os.ModeDevice | os.ModeCharDevice
	case syscall.S_IFDIR:
		fm |= os.ModeDir
	case syscall.S_IFIFO:
		fm |= os.ModeNamedPipe
	case syscall.S_IFLNK:
		fm |= os.ModeSymlink
	case syscall.S_IFREG:
		// nothing to do
	case syscall.S_IFSOCK:
		fm |= os.ModeSocket
	}
	if mode&syscall.S_ISGID != 0 {
		fm |= os.ModeSetgid
	}
	if mode&syscall.S_ISUID != 0 {
		fm |= os.ModeSetuid
	}
	if mode&syscall.S_ISVTX != 0 {
		fm |= os.ModeSticky
	}
	return fm
}

func (c *PackageFile) Summary() string {
	return strings.Join(c.GetStrings(1, 1004), "\n")
}

func (c *PackageFile) Description() string {
	return strings.Join(c.GetStrings(1, 1005), "\n")
}

func (c *PackageFile) BuildTime() time.Time {
	return c.Headers[1].Indexes.TimeByTag(1006)
}

func (c *PackageFile) BuildHost() string {
	return c.GetString(1, 1007)
}

func (c *PackageFile) InstallTime() time.Time {
	return c.Headers[1].Indexes.TimeByTag(1008)
}

// Size specifies the disk space consumed by installation of the package.
func (c *PackageFile) Size() uint64 {
	return uint64(c.GetInt(1, 1009))
}

// ArchiveSize specifies the size of the archived payload of the package in
// bytes.
func (c *PackageFile) ArchiveSize() uint64 {
	if i := uint64(c.GetInt(0, 1007)); i > 0 {
		return i
	}

	return uint64(c.GetInt(1, 1046))
}

func (c *PackageFile) Distribution() string {
	return c.GetString(1, 1010)
}

func (c *PackageFile) Vendor() string {
	return c.GetString(1, 1011)
}

func (c *PackageFile) GIFImage() []byte {
	return c.GetBytes(1, 1012)
}

func (c *PackageFile) XPMImage() []byte {
	return c.GetBytes(1, 1013)
}

func (c *PackageFile) License() string {
	return c.GetString(1, 1014)
}

func (c *PackageFile) Packager() string {
	return c.GetString(1, 1015)
}

func (c *PackageFile) Groups() []string {
	return c.GetStrings(1, 1016)
}

func (c *PackageFile) ChangeLog() []string {
	return c.GetStrings(1, 1017)
}

func (c *PackageFile) Source() []string {
	return c.GetStrings(1, 1018)
}

func (c *PackageFile) Patch() []string {
	return c.GetStrings(1, 1019)
}

func (c *PackageFile) URL() string {
	return c.GetString(1, 1020)
}

func (c *PackageFile) OperatingSystem() string {
	return c.GetString(1, 1021)
}

func (c *PackageFile) Architecture() string {
	return c.GetString(1, 1022)
}

func (c *PackageFile) PreInstallScript() string {
	return c.GetString(1, 1023)
}

func (c *PackageFile) PostInstallScript() string {
	return c.GetString(1, 1024)
}

func (c *PackageFile) PreUninstallScript() string {
	return c.GetString(1, 1025)
}

func (c *PackageFile) PostUninstallScript() string {
	return c.GetString(1, 1026)
}

func (c *PackageFile) OldFilenames() []string {
	return c.GetStrings(1, 1027)
}

func (c *PackageFile) Icon() []byte {
	return c.GetBytes(1, 1043)
}

func (c *PackageFile) SourceRPM() string {
	return c.GetString(1, 1044)
}

func (c *PackageFile) RPMVersion() string {
	return c.GetString(1, 1064)
}

func (c *PackageFile) Platform() string {
	return c.GetString(1, 1132)
}

// PayloadFormat returns the name of the format used for the package payload.
// Typically cpio.
func (c *PackageFile) PayloadFormat() string {
	return c.GetString(1, 1124)
}

// PayloadCompression returns the name of the compression used for the package
// payload. Typically xz.
func (c *PackageFile) PayloadCompression() string {
	return c.GetString(1, 1125)
}
