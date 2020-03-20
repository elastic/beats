package target

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// expand takes a collection of sources as strings, and for each one, it expands
// environment variables in the form $FOO or ${FOO} using os.ExpandEnv
func expand(sources []string) []string {
	for i, s := range sources {
		// first expand the environment
		sources[i] = os.ExpandEnv(s)
	}
	return sources
}

// Path first expands environment variables like $FOO or ${FOO}, and then
// reports if any of the sources have been modified more recently than the
// destination. Path does not descend into directories, it literally just checks
// the modtime of each thing you pass to it. If the destination file doesn't
// exist, it always returns true and nil. It's an error if any of the sources
// don't exist.
func Path(dst string, sources ...string) (bool, error) {
	stat, err := os.Stat(os.ExpandEnv(dst))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	srcTime := stat.ModTime()
	dt, err := loadTargets(expand(sources))
	if err != nil {
		return false, err
	}
	t := dt.modTime()
	if t.After(srcTime) {
		return true, nil
	}
	return false, nil
}

// Glob expands each of the globs (file patterns) into individual sources and
// then calls Path on the result, reporting if any of the resulting sources have
// been modified more recently than the destination. Syntax for Glob patterns is
// the same as stdlib's filepath.Glob. Note that Glob does not expand
// environment variables before globbing -- env var expansion happens during
// the call to Path. It is an error for any glob to return an empty result.
func Glob(dst string, globs ...string) (bool, error) {
	for _, g := range globs {
		files, err := filepath.Glob(g)
		if err != nil {
			return false, err
		}
		if len(files) == 0 {
			return false, errors.New("glob didn't match any files: " + g)
		}
		// it's best to evaluate each glob as we do it
		// because we may be able to early-exit
		shouldDo, err := Path(dst, files...)
		if err != nil {
			return false, err
		}
		if shouldDo {
			return true, nil
		}
	}
	return false, nil
}

// Dir reports whether any of the sources have been modified more recently than
// the destination.  If a source or destination is a directory, modtimes of
// files under those directories are compared instead.  If the destination file
// doesn't exist, it always returns true and nil.  It's an error if any of the
// sources don't exist.
func Dir(dst string, sources ...string) (bool, error) {
	stat, err := os.Stat(os.ExpandEnv(dst))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	srcTime := stat.ModTime()
	if stat.IsDir() {
		srcTime, err = calDirModTimeRecursive(dst, stat)
		if err != nil {
			return false, err
		}
	}
	dt, err := loadTargets(expand(sources))
	if err != nil {
		return false, err
	}
	t, err := dt.modTimeDir()
	if err != nil {
		return false, err
	}
	if t.After(srcTime) {
		return true, nil
	}
	return false, nil
}

func calDirModTimeRecursive(name string, dir os.FileInfo) (time.Time, error) {
	t := dir.ModTime()
	ferr := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.ModTime().After(t) {
			t = info.ModTime()
		}
		return nil
	})
	if ferr != nil {
		return time.Time{}, ferr
	}
	return t, nil
}

type source struct {
	path string
	info os.FileInfo
}

type depTargets struct {
	src    []source
	hasdir bool
	latest time.Time
}

func loadTargets(targets []string) (*depTargets, error) {
	d := &depTargets{}
	for _, v := range targets {
		stat, err := os.Stat(v)
		if err != nil {
			return nil, err
		}
		if stat.IsDir() {
			d.hasdir = true
		}
		d.src = append(d.src, source{path: v, info: stat})
		if stat.ModTime().After(d.latest) {
			d.latest = stat.ModTime()
		}
	}
	return d, nil
}

func (d *depTargets) modTime() time.Time {
	return d.latest
}

func (d *depTargets) modTimeDir() (time.Time, error) {
	if !d.hasdir {
		return d.latest, nil
	}
	var err error
	for _, src := range d.src {
		t := src.info.ModTime()
		if src.info.IsDir() {
			t, err = calDirModTimeRecursive(src.path, src.info)
			if err != nil {
				return time.Time{}, err
			}
		}
		if t.After(d.latest) {
			d.latest = t
		}
	}
	return d.latest, nil
}
