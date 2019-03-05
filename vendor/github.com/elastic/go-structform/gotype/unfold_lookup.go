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

package gotype

/*
func lookupReflUnfolder(t reflect.Type) (reflUnfolder, error) {
	// we always expect a pointer to a value
	bt := t.Elem()

	switch bt.Kind() {
	case reflect.Array:
		switch bt.Elem().Kind() {
		case reflect.Int:
			return liftGoUnfolder(newUnfolderArrInt()), nil
		}

	case reflect.Slice:

	case reflect.Map:
		if bt.Key().Kind() != reflect.String {
			return nil, errMapRequiresStringKey
		}

		switch bt.Elem().Kind() {
		case reflect.Interface:
			return liftGoUnfolder(newUnfolderMapIfc()), nil
		}

	case reflect.Struct:

	}

	return nil, errTODO()
}
*/
