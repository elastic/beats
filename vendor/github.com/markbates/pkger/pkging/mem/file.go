package mem

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
)

const timeFmt = time.RFC3339Nano

var _ pkging.File = &File{}

type File struct {
	Here   here.Info
	info   *pkging.FileInfo
	path   here.Path
	data   []byte
	parent here.Path
	writer *bytes.Buffer
	reader io.Reader
	pkging pkging.Pkger
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted according to whence: 0 means relative to the origin of the file, 1 means relative to the current offset, and 2 means relative to the end. It returns the new offset and an error, if any.
func (f *File) Seek(ofpkginget int64, whence int) (int64, error) {
	if len(f.data) > 0 && f.reader == nil {
		f.reader = bytes.NewReader(f.data)
	}

	if sk, ok := f.reader.(io.Seeker); ok {
		return sk.Seek(ofpkginget, whence)
	}
	return 0, nil
}

// Close closes the File, rendering it unusable for I/O.
func (f *File) Close() error {
	defer func() {
		f.reader = nil
		f.writer = nil
	}()
	if f.reader != nil {
		if c, ok := f.reader.(io.Closer); ok {
			if err := c.Close(); err != nil {
				return err
			}
		}
	}

	if f.writer == nil {
		return nil
	}

	f.data = f.writer.Bytes()

	fi := f.info
	fi.Details.Size = int64(len(f.data))
	fi.Details.ModTime = pkging.ModTime(time.Now())
	f.info = fi
	return nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *File) Read(p []byte) (int, error) {
	if len(f.data) > 0 && f.reader == nil {
		f.reader = bytes.NewReader(f.data)
	}

	if f.reader != nil {
		return f.reader.Read(p)
	}

	return 0, fmt.Errorf("unable to read %s", f.Name())
}

// Write writes len(b) bytes to the File. It returns the number of bytes written and an error, if any. Write returns a non-nil error when n != len(b).
func (f *File) Write(b []byte) (int, error) {
	if f.writer == nil {
		f.writer = &bytes.Buffer{}
	}
	i, err := f.writer.Write(b)
	return i, err
}

// Info returns the here.Info of the file
func (f File) Info() here.Info {
	return f.Here
}

// Stat returns the FileInfo structure describing file. If there is an error, it will be of type *PathError.
func (f File) Stat() (os.FileInfo, error) {
	if f.info == nil {
		return nil, os.ErrNotExist
	}
	return f.info, nil
}

// Name retuns the name of the file in pkger format
func (f File) Name() string {
	return f.path.String()
}

// Path returns the here.Path of the file
func (f File) Path() here.Path {
	return f.path
}

func (f File) String() string {
	return f.Path().String()
}

// Readdir reads the contents of the directory associated with file and returns a slice of up to n FileInfo values, as would be returned by Lstat, in directory order. Subsequent calls on the same file will yield further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if Readdir returns an empty slice, it will return a non-nil error explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice. In this case, if Readdir succeeds (reads all the way to the end of the directory), it returns the slice and a nil error. If it encounters an error before the end of the directory, Readdir returns the FileInfo read until that point and a non-nil error.
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	var infos []os.FileInfo
	root := f.Path().String()
	err := f.pkging.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if count > 0 && len(infos) == count {
			return io.EOF
		}

		if root == path {
			return nil
		}

		pt, err := f.pkging.Parse(path)
		if err != nil {
			return err
		}
		if pt.Name == f.parent.Name {
			return nil
		}

		infos = append(infos, info)
		if info.IsDir() && path != root {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return infos, nil
		}
		if err != io.EOF {
			return nil, err
		}
	}
	return infos, nil

}

// Open implements the http.FileSystem interface. A FileSystem implements access to a collection of named files. The elements in a file path are separated by slash ('/', U+002F) characters, regardless of host operating system convention.
func (f *File) Open(name string) (http.File, error) {
	pt, err := f.Here.Parse(name)
	if err != nil {
		return nil, err
	}

	if pt == f.path {
		return f, nil
	}

	pt.Name = path.Join(f.Path().Name, pt.Name)

	di, err := f.pkging.Open(pt.String())
	if err != nil {
		return nil, err
	}

	fi, err := di.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		d2 := &File{
			info:   pkging.NewFileInfo(fi),
			Here:   di.Info(),
			path:   pt,
			parent: f.path,
			pkging: f.pkging,
		}
		di = d2
	}
	return di, nil
}
