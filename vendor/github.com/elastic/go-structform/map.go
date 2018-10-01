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

package structform

type extObjVisitor struct {
	Visitor
}

func (e extObjVisitor) OnStringObject(m map[string]string) error {
	if err := e.OnObjectStart(len(m), StringType); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnString(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnBoolObject(m map[string]bool) error {
	if err := e.OnObjectStart(len(m), BoolType); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnBool(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnInt8Object(m map[string]int8) error {
	if err := e.OnObjectStart(len(m), Int8Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnInt8(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnInt16Object(m map[string]int16) error {
	if err := e.OnObjectStart(len(m), Int16Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnInt16(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnInt32Object(m map[string]int32) error {
	if err := e.OnObjectStart(len(m), Int32Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnInt32(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnInt64Object(m map[string]int64) error {
	if err := e.OnObjectStart(len(m), Int64Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnInt64(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnIntObject(m map[string]int) error {
	if err := e.OnObjectStart(len(m), IntType); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnInt(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnUint8Object(m map[string]uint8) error {
	if err := e.OnObjectStart(len(m), Uint8Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnUint8(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnUint16Object(m map[string]uint16) error {
	if err := e.OnObjectStart(len(m), Uint16Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnUint16(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnUint32Object(m map[string]uint32) error {
	if err := e.OnObjectStart(len(m), Uint32Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnUint32(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnUint64Object(m map[string]uint64) error {
	if err := e.OnObjectStart(len(m), Uint64Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnUint64(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnUintObject(m map[string]uint) error {
	if err := e.OnObjectStart(len(m), UintType); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnUint(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnFloat32Object(m map[string]float32) error {
	if err := e.OnObjectStart(len(m), Float32Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnFloat32(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}

func (e extObjVisitor) OnFloat64Object(m map[string]float64) error {
	if err := e.OnObjectStart(len(m), Float64Type); err != nil {
		return err
	}

	for k, v := range m {
		if err := e.OnKey(k); err != nil {
			return err
		}
		if err := e.OnFloat64(v); err != nil {
			return err
		}
	}

	return e.OnObjectFinished()
}
