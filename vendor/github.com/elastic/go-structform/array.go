package structform

type extArrVisitor struct {
	Visitor
}

func (ev extArrVisitor) OnStringArray(a []string) error {
	if err := ev.OnArrayStart(len(a), StringType); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnString(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnBoolArray(a []bool) error {
	if err := ev.OnArrayStart(len(a), BoolType); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnBool(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnInt8Array(a []int8) error {
	if err := ev.OnArrayStart(len(a), Int8Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnInt8(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnInt16Array(a []int16) error {
	if err := ev.OnArrayStart(len(a), Int16Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnInt16(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnInt32Array(a []int32) error {
	if err := ev.OnArrayStart(len(a), Int32Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnInt32(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnInt64Array(a []int64) error {
	if err := ev.OnArrayStart(len(a), Int64Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnInt64(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnIntArray(a []int) error {
	if err := ev.OnArrayStart(len(a), IntType); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnInt(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnBytes(b []byte) error {
	if err := ev.OnArrayStart(len(b), ByteType); err != nil {
		return err
	}
	for _, v := range b {
		if err := ev.OnByte(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnUint8Array(a []uint8) error {
	if err := ev.OnArrayStart(len(a), Uint8Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnUint8(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnUint16Array(a []uint16) error {
	if err := ev.OnArrayStart(len(a), Uint16Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnUint16(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnUint32Array(a []uint32) error {
	if err := ev.OnArrayStart(len(a), Uint32Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnUint32(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnUint64Array(a []uint64) error {
	if err := ev.OnArrayStart(len(a), Uint64Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnUint64(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnUintArray(a []uint) error {
	if err := ev.OnArrayStart(len(a), UintType); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnUint(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnFloat32Array(a []float32) error {
	if err := ev.OnArrayStart(len(a), Float32Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnFloat32(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}

func (ev extArrVisitor) OnFloat64Array(a []float64) error {
	if err := ev.OnArrayStart(len(a), Float64Type); err != nil {
		return err
	}
	for _, v := range a {
		if err := ev.OnFloat64(v); err != nil {
			return err
		}
	}
	return ev.OnArrayFinished()
}
