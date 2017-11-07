package rpm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A PackageFile is an RPM package definition loaded directly from the pacakge
// file itself.
type PackageFile struct {
	Lead    Lead
	Headers Headers

	path     string
	fileSize uint64
	fileTime time.Time
}

// ReadPackageFile reads a rpm package file from a stream and returns a pointer
// to it.
func ReadPackageFile(r io.Reader) (*PackageFile, error) {
	// See: http://www.rpm.org/max-rpm/s1-rpm-file-format-rpm-file-format.html
	p := &PackageFile{}

	// read the deprecated "lead"
	lead, err := ReadPackageLead(r)
	if err != nil {
		return nil, err
	}

	p.Lead = *lead

	// read signature and header headers
	offset := 96
	p.Headers = make(Headers, 2)
	for i := 0; i < 2; i++ {
		// parse header
		h, err := ReadPackageHeader(r)
		if err != nil {
			return nil, fmt.Errorf("%v (v%d.%d)", err, lead.VersionMajor, lead.VersionMinor)
		}

		// set start and end offsets
		h.Start = offset
		h.End = h.Start + 16 + (16 * h.IndexCount) + h.Length
		offset = h.End

		// calculate location of the end of the header by padding to a multiple of 8
		pad := 8 - int(math.Mod(float64(h.Length), 8))
		if pad < 8 {
			offset += pad
		}

		// append
		p.Headers[i] = *h
	}

	return p, nil
}

// OpenPackageFile reads a rpm package from the file systems and returns a pointer
// to it.
func OpenPackageFile(path string) (*PackageFile, error) {
	// stat file
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read package content
	p, err := ReadPackageFile(f)
	if err != nil {
		return nil, err
	}

	// set file info
	p.path = path
	p.fileSize = uint64(fi.Size())
	p.fileTime = fi.ModTime()

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
func (c *PackageFile) dependencies(nevrsTagId, flagsTagId, namesTagId, versionsTagId int) Dependencies {
	// TODO: Implement NEVRS tags

	flgs := c.Headers[1].Indexes.IntsByTag(flagsTagId)
	names := c.Headers[1].Indexes.StringsByTag(namesTagId)
	vers := c.Headers[1].Indexes.StringsByTag(versionsTagId)

	deps := make(Dependencies, len(names))
	for i := 0; i < len(names); i++ {
		deps[i] = NewDependency(int(flgs[i]), names[i], 0, vers[i], "")
	}

	return deps
}

// String returns the package identifier in the form
// '[name]-[version]-[release].[architecture]'.
func (c *PackageFile) String() string {
	return fmt.Sprintf("%s-%s-%s.%s", c.Name(), c.Version(), c.Release(), c.Architecture())
}

// Path returns the path which was given to open a package file if it was opened
// with OpenPackageFile.
func (c *PackageFile) Path() string {
	return c.path
}

// FileTime returns the time at which the RPM was last modified if known.
func (c *PackageFile) FileTime() time.Time {
	return c.fileTime
}

// FileSize returns the size of the package file in bytes.
func (c *PackageFile) FileSize() uint64 {
	return c.fileSize
}

// Checksum computes and returns the SHA256 checksum (encoded in hexidecimal) of
// the package file.
func (c *PackageFile) Checksum() (string, error) {
	if c.Path() == "" {
		return "", fmt.Errorf("File not found")
	}

	if f, err := os.Open(c.Path()); err != nil {
		return "", err
	} else {
		defer f.Close()

		s := sha256.New()
		if _, err := io.Copy(s, f); err != nil {
			return "", err
		}

		return hex.EncodeToString(s.Sum(nil)), nil
	}
}

// ChecksumType returns "sha256"
func (c *PackageFile) ChecksumType() string {
	return "sha256"
}

func (c *PackageFile) HeaderStart() uint64 {
	return uint64(c.Headers[1].Start)
}

func (c *PackageFile) HeaderEnd() uint64 {
	return uint64(c.Headers[1].End)
}

// For tag definitions, see:
// https://github.com/rpm-software-management/rpm/blob/master/lib/rpmtag.h#L61

func (c *PackageFile) Name() string {
	return c.Headers[1].Indexes.StringByTag(1000)
}

func (c *PackageFile) Version() string {
	return c.Headers[1].Indexes.StringByTag(1001)
}

func (c *PackageFile) Release() string {
	return c.Headers[1].Indexes.StringByTag(1002)
}

func (c *PackageFile) Epoch() int {
	return int(c.Headers[1].Indexes.IntByTag(1003))
}

func (c *PackageFile) Requires() Dependencies {
	return c.dependencies(5041, 1048, 1049, 1050)
}

func (c *PackageFile) Provides() Dependencies {
	return c.dependencies(5042, 1112, 1047, 1113)
}

func (c *PackageFile) Conflicts() Dependencies {
	return c.dependencies(5044, 1053, 1054, 1055)
}

func (c *PackageFile) Obsoletes() Dependencies {
	return c.dependencies(5043, 1114, 1090, 1115)
}

// Files returns file information for each file that is installed by this RPM
// package.
func (c *PackageFile) Files() []FileInfo {
	ixs := c.Headers[1].Indexes.IntsByTag(1116)
	names := c.Headers[1].Indexes.StringsByTag(1117)
	dirs := c.Headers[1].Indexes.StringsByTag(1118)
	modes := c.Headers[1].Indexes.IntsByTag(1030)
	sizes := c.Headers[1].Indexes.IntsByTag(1028)
	times := c.Headers[1].Indexes.IntsByTag(1034)
	owners := c.Headers[1].Indexes.StringsByTag(1039)
	groups := c.Headers[1].Indexes.StringsByTag(1040)

	files := make([]FileInfo, len(names))
	for i := 0; i < len(names); i++ {
		files[i] = FileInfo{
			name:    dirs[ixs[i]] + names[i],
			mode:    os.FileMode(modes[i]),
			size:    sizes[i],
			modTime: time.Unix(times[i], 0),
			owner:   owners[i],
			group:   groups[i],
		}
	}

	return files
}

func (c *PackageFile) Summary() string {
	return strings.Join(c.Headers[1].Indexes.StringsByTag(1004), "\n")
}

func (c *PackageFile) Description() string {
	return strings.Join(c.Headers[1].Indexes.StringsByTag(1005), "\n")
}

func (c *PackageFile) BuildTime() time.Time {
	return c.Headers[1].Indexes.TimeByTag(1006)
}

func (c *PackageFile) BuildHost() string {
	return c.Headers[1].Indexes.StringByTag(1007)
}

func (c *PackageFile) InstallTime() time.Time {
	return c.Headers[1].Indexes.TimeByTag(1008)
}

// Size specifies the disk space consumed by installation of the package.
func (c *PackageFile) Size() uint64 {
	return uint64(c.Headers[1].Indexes.IntByTag(1009))
}

// ArchiveSize specifies the size of the archived payload of the package in
// bytes.
func (c *PackageFile) ArchiveSize() uint64 {
	if i := uint64(c.Headers[0].Indexes.IntByTag(1007)); i > 0 {
		return i
	}

	return uint64(c.Headers[1].Indexes.IntByTag(1046))
}

func (c *PackageFile) Distribution() string {
	return c.Headers[1].Indexes.StringByTag(1010)
}

func (c *PackageFile) Vendor() string {
	return c.Headers[1].Indexes.StringByTag(1011)
}

func (c *PackageFile) GIFImage() []byte {
	return c.Headers[1].Indexes.BytesByTag(1012)
}

func (c *PackageFile) XPMImage() []byte {
	return c.Headers[1].Indexes.BytesByTag(1013)
}

func (c *PackageFile) License() string {
	return c.Headers[1].Indexes.StringByTag(1014)
}

func (c *PackageFile) Packager() string {
	return c.Headers[1].Indexes.StringByTag(1015)
}

func (c *PackageFile) Groups() []string {
	return c.Headers[1].Indexes.StringsByTag(1016)
}

func (c *PackageFile) ChangeLog() []string {
	return c.Headers[1].Indexes.StringsByTag(1017)
}

func (c *PackageFile) Source() []string {
	return c.Headers[1].Indexes.StringsByTag(1018)
}

func (c *PackageFile) Patch() []string {
	return c.Headers[1].Indexes.StringsByTag(1019)
}

func (c *PackageFile) URL() string {
	return c.Headers[1].Indexes.StringByTag(1020)
}

func (c *PackageFile) OperatingSystem() string {
	return c.Headers[1].Indexes.StringByTag(1021)
}

func (c *PackageFile) Architecture() string {
	return c.Headers[1].Indexes.StringByTag(1022)
}

func (c *PackageFile) PreInstallScript() string {
	return c.Headers[1].Indexes.StringByTag(1023)
}

func (c *PackageFile) PostInstallScript() string {
	return c.Headers[1].Indexes.StringByTag(1024)
}

func (c *PackageFile) PreUninstallScript() string {
	return c.Headers[1].Indexes.StringByTag(1025)
}

func (c *PackageFile) PostUninstallScript() string {
	return c.Headers[1].Indexes.StringByTag(1026)
}

func (c *PackageFile) OldFilenames() []string {
	return c.Headers[1].Indexes.StringsByTag(1027)
}

func (c *PackageFile) Icon() []byte {
	return c.Headers[1].Indexes.BytesByTag(1043)
}

func (c *PackageFile) SourceRPM() string {
	return c.Headers[1].Indexes.StringByTag(1044)
}

func (c *PackageFile) RPMVersion() string {
	return c.Headers[1].Indexes.StringByTag(1064)
}

func (c *PackageFile) Platform() string {
	return c.Headers[1].Indexes.StringByTag(1132)
}
