package structform

type extStrVisitor struct {
	v Visitor
}

func MakeStringRefVisitor(v Visitor) StringRefVisitor {
	if sv, ok := v.(StringRefVisitor); ok {
		return sv
	}
	return extStrVisitor{v}
}

func (ev extStrVisitor) OnStringRef(s []byte) error {
	return ev.v.OnString(string(s))
}

func (ev extStrVisitor) OnKeyRef(s []byte) error {
	return ev.v.OnKey(string(s))
}
