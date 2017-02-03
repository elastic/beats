package chrootarchive

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
)

var chrootArchiver = &archive.Archiver{Untar: Untar}

// Untar reads a stream of bytes from `archive`, parses it as a tar archive,
// and unpacks it into the directory at `dest`.
// The archive may be compressed with one of the following algorithms:
//  identity (uncompressed), gzip, bzip2, xz.
func Untar(tarArchive io.Reader, dest string, options *archive.TarOptions) error {
	return untarHandler(tarArchive, dest, options, true)
}

// UntarUncompressed reads a stream of bytes from `archive`, parses it as a tar archive,
// and unpacks it into the directory at `dest`.
// The archive must be an uncompressed stream.
func UntarUncompressed(tarArchive io.Reader, dest string, options *archive.TarOptions) error {
	return untarHandler(tarArchive, dest, options, false)
}

// Handler for teasing out the automatic decompression
func untarHandler(tarArchive io.Reader, dest string, options *archive.TarOptions, decompress bool) error {

	if tarArchive == nil {
		return fmt.Errorf("Empty archive")
	}
	if options == nil {
		options = &archive.TarOptions{}
	}
	if options.ExcludePatterns == nil {
		options.ExcludePatterns = []string{}
	}

	rootUID, rootGID, err := idtools.GetRootUIDGID(options.UIDMaps, options.GIDMaps)
	if err != nil {
		return err
	}

	dest = filepath.Clean(dest)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if err := idtools.MkdirAllNewAs(dest, 0755, rootUID, rootGID); err != nil {
			return err
		}
	}

	r := ioutil.NopCloser(tarArchive)
	if decompress {
		decompressedArchive, err := archive.DecompressStream(tarArchive)
		if err != nil {
			return err
		}
		defer decompressedArchive.Close()
		r = decompressedArchive
	}

	return invokeUnpack(r, dest, options)
}

// TarUntar is a convenience function which calls Tar and Untar, with the output of one piped into the other.
// If either Tar or Untar fails, TarUntar aborts and returns the error.
func TarUntar(src, dst string) error {
	return chrootArchiver.TarUntar(src, dst)
}

// CopyWithTar creates a tar archive of filesystem path `src`, and
// unpacks it at filesystem path `dst`.
// The archive is streamed directly with fixed buffering and no
// intermediary disk IO.
func CopyWithTar(src, dst string) error {
	return chrootArchiver.CopyWithTar(src, dst)
}

// CopyFileWithTar emulates the behavior of the 'cp' command-line
// for a single file. It copies a regular file from path `src` to
// path `dst`, and preserves all its metadata.
//
// If `dst` ends with a trailing slash '/' ('\' on Windows), the final
// destination path will be `dst/base(src)` or `dst\base(src)`
func CopyFileWithTar(src, dst string) (err error) {
	return chrootArchiver.CopyFileWithTar(src, dst)
}

// UntarPath is a convenience function which looks for an archive
// at filesystem path `src`, and unpacks it at `dst`.
func UntarPath(src, dst string) error {
	return chrootArchiver.UntarPath(src, dst)
}
