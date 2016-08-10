package source

import (
    "os"
    "errors"
    "path/filepath"
    "compress/gzip"
    "github.com/elastic/beats/libbeat/logp"
)

type File struct {
    *os.File
}

func NewFile(osf *os.File) (FileSource, error) {
    fileExt := filepath.Ext(osf.Name())
    logp.Debug("harvester", "file extension is %s", fileExt)
    switch fileExt {
        case ".gz": {
            logp.Debug("harvester", "reading compressed gzip file %s", osf.Name())
            return  newGZipFile(osf)
        }
        default: return File{osf}, nil
    }
}

func (File) Continuable() bool { return true }

type GZipFile struct {
    *os.File
    gzipReader  *gzip.Reader
}

func newGZipFile(osf *os.File) (FileSource, error) {
    gzipReader, err := gzip.NewReader(osf)
    if err != nil {
        return nil, err
    } else {
        return GZipFile{osf, gzipReader}, nil
    }
}

func (GZipFile) Continuable() bool { return false }

func (gf GZipFile) Close() error {
    err1 := gf.gzipReader.Close()
    err2 := gf.File.Close()
    if err2 != nil {
        return err2
    } else { 
        return err1
    }
}

func (gf GZipFile) Read(p []byte) (n int, err error) {
    return gf.gzipReader.Read(p)
}

func (gf GZipFile) Seek(offset int64, whence int) (int64, error) {
    // TODO: gzip package doesn't inherently support seeking, we maybe able to fake seeking using read
    err := errors.New("Seeking is not supported on gzip files. Offset: ")
    switch whence {
    case os.SEEK_SET:
    case os.SEEK_CUR:
        if (offset > 0) {
            return 0, err
        } else {
            return offset, nil
        }
    }
    return 0, err
}

