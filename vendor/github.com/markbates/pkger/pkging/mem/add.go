package mem

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
)

var _ pkging.Adder = &Pkger{}

// Add copies the pkging.File into the *Pkger
func (fx *Pkger) Add(files ...*os.File) error {
	for _, f := range files {
		info, err := f.Stat()
		if err != nil {
			return err
		}
		pt, err := fx.Parse(f.Name())
		if err != nil {
			return err
		}

		dir := f.Name()
		if !info.IsDir() {
			dir = filepath.Dir(dir)
		}

		her, err := here.Dir(dir)
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = filepath.Walk(f.Name(), func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()

				pt, err := fx.Parse(path)
				if err != nil {
					return err
				}

				her, err := here.Package(pt.Pkg)
				if err != nil {
					return err
				}

				mf := &File{
					Here:   her,
					info:   pkging.NewFileInfo(info),
					path:   pt,
					pkging: fx,
				}

				if !info.IsDir() {
					bb := &bytes.Buffer{}
					_, err = io.Copy(bb, f)
					if err != nil {
						return err
					}
					mf.data = bb.Bytes()
				}

				fx.files.Store(mf.Path(), mf)

				return nil
			})
			if err != nil {
				return err
			}
			continue
		}

		mf := &File{
			Here:   her,
			info:   pkging.NewFileInfo(info),
			path:   pt,
			pkging: fx,
		}

		if !info.IsDir() {
			bb := &bytes.Buffer{}
			_, err = io.Copy(bb, f)
			if err != nil {
				return err
			}
			mf.data = bb.Bytes()
		}

		fx.files.Store(mf.Path(), mf)
	}

	return nil
}
