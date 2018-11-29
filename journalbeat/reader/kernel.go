// Copyright 2017 Marcus Heese
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reader

import (
	"fmt"
)

func (r *Reader) addKernel() error {
	if len(r.config.Units) > 0 && r.config.Kernel {
		err := r.addMatchesForKernel()
		if err != nil {
			return fmt.Errorf("adding filter for kernel failed: %+v", err)
		}
		r.logger.Debug("Added matcher expression to monitor kernel logs")
	}
	return nil
}

func (r *Reader) addMatchesForKernel() error {
	err := r.journal.AddMatch("_TRANSPORT=kernel")
	if err != nil {
		return err
	}
	return r.journal.AddDisjunction()
}
