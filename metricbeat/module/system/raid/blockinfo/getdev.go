// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package blockinfo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// ListAll lists all the multi-disk devices in a RAID array
func ListAll(path string) ([]MDDevice, error) {
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read directory")
	}
	var mds []MDDevice
	for _, item := range dir {
		testpath := filepath.Join(path, item.Name())
		if !isMD(testpath) {
			continue
		}
		dev, err := getMDDevice(testpath)
		if err != nil {
			return nil, errors.Wrap(err, "could not get device info")
		}
		mds = append(mds, dev)
	}

	if len(mds) == 0 {
		return nil, fmt.Errorf("no matches from path %s,", path)
	}

	return mds, nil
}

// getMDDevice returns a MDDevice object representing a multi-disk device, or error if it's not a "real" md device
func getMDDevice(path string) (MDDevice, error) {
	_, err := os.Stat(path)
	if err != nil {
		return MDDevice{}, errors.Wrap(err, "path does not exist")
	}

	//This is the best heuristic I've found so far for identifying an md device.
	if !isMD(path) {
		return MDDevice{}, err
	}
	return newMD(path)
}

//check if a block device directory looks like an MD device
//I'm not convinced that using /sys/block/md* is a reliable glob, as you should be able to make those whatever you want.
//Right now, we're doing this by looking for an `md` directory in the device dir.
func isMD(path string) bool {
	_, err := os.Stat(filepath.Join(path, "md"))
	if err != nil {
		return false
	}
	return true
}
