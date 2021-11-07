// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package loader abstracts the reading document at given url.
//
// It allows developers to register loaders for different uri
// schemes.
package loader

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Loader is the interface that wraps the basic Load method.
//
// Load loads the document at given url and returns []byte,
// if successful.
type Loader interface {
	Load(url string) (io.ReadCloser, error)
}

type filePathLoader struct{}

func (filePathLoader) Load(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

type fileURLLoader struct{}

func (fileURLLoader) Load(s string) (io.ReadCloser, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	f := u.Path
	if runtime.GOOS == "windows" {
		f = strings.TrimPrefix(f, "/")
		f = filepath.FromSlash(f)
	}
	return os.Open(f)
}

var registry = make(map[string]Loader)
var mutex = sync.RWMutex{}

// SchemeNotRegisteredError is the error type returned by Load function.
// It tells that no Loader is registered for that URL Scheme.
type SchemeNotRegisteredError string

func (s SchemeNotRegisteredError) Error() string {
	return fmt.Sprintf("no Loader registered for scheme %s", string(s))
}

// Register registers given Loader for given URI Scheme.
func Register(scheme string, loader Loader) {
	mutex.Lock()
	defer mutex.Unlock()
	registry[scheme] = loader
}

// UnRegister unregisters the registered loader(if any) for given URI Scheme.
func UnRegister(scheme string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(registry, scheme)
}

func get(s string) (Loader, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if loader, ok := registry[u.Scheme]; ok {
		return loader, nil
	}
	return nil, SchemeNotRegisteredError(u.Scheme)
}

// Load loads the document at given url and returns []byte,
// if successful.
//
// If no Loader is registered against the URI Scheme, then it
// returns *SchemeNotRegisteredError
var Load = func(url string) (io.ReadCloser, error) {
	loader, err := get(url)
	if err != nil {
		return nil, err
	}
	return loader.Load(url)
}

func init() {
	Register("", filePathLoader{})
	Register("file", fileURLLoader{})
}
