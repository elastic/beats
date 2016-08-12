package ucfg

import (
	"fmt"
	"strconv"
	"strings"
)

type cfgPath struct {
	fields []field
	sep    string
}

type field interface {
	String() string
	SetValue(opt *options, elem value, v value) Error
	GetValue(opt *options, elem value) (value, Error)
}

type namedField struct {
	name string
}

type idxField struct {
	i int
}

func parsePathIdx(in, sep string, idx int) cfgPath {
	if in == "" {
		return cfgPath{
			sep:    sep,
			fields: []field{idxField{idx}},
		}
	}

	p := parsePath(in, sep)
	if idx >= 0 {
		p.fields = append(p.fields, idxField{idx})
	}

	return p
}

func parsePath(in, sep string) cfgPath {
	if sep == "" {
		return cfgPath{
			sep:    sep,
			fields: []field{namedField{in}},
		}
	}

	elems := strings.Split(in, sep)
	fields := make([]field, 0, len(elems))
	for _, elem := range elems {
		if idx, err := strconv.ParseInt(elem, 0, 64); err == nil {
			fields = append(fields, idxField{int(idx)})
		} else {
			fields = append(fields, namedField{elem})
		}
	}

	return cfgPath{fields: fields, sep: sep}
}

func (p cfgPath) String() string {
	if len(p.fields) == 0 {
		return ""
	}

	if len(p.fields) == 1 {
		return p.fields[0].String()
	}

	s := make([]string, 0, len(p.fields))
	for _, f := range p.fields {
		s = append(s, f.String())
	}

	sep := p.sep
	if sep == "" {
		sep = "."
	}
	return strings.Join(s, sep)
}

func (n namedField) String() string {
	return n.name
}

func (i idxField) String() string {
	return fmt.Sprintf("%d", i.i)
}

func (p cfgPath) GetValue(cfg *Config, opt *options) (value, Error) {
	fields := p.fields

	cur := value(cfgSub{cfg})
	for ; len(fields) > 1; fields = fields[1:] {
		field := fields[0]
		next, err := field.GetValue(opt, cur)
		if err != nil {
			return nil, err
		}

		if next == nil {
			return nil, raiseMissing(cfg, field.String())
		}

		cur = next
	}

	field := fields[0]
	v, err := field.GetValue(opt, cur)
	if err != nil {
		return nil, raiseMissing(cfg, field.String())
	}
	return v, nil
}

func (n namedField) GetValue(opts *options, elem value) (value, Error) {
	cfg, err := elem.toConfig(opts)
	if err != nil {
		return nil, raiseExpectedObject(opts, elem)
	}

	v, _ := cfg.fields.get(n.name)
	return v, nil
}

func (i idxField) GetValue(opts *options, elem value) (value, Error) {
	cfg, err := elem.toConfig(opts)
	if err != nil {
		if i.i == 0 {
			return elem, nil
		}

		return nil, raiseExpectedObject(opts, elem)
	}

	arr := cfg.fields.array()
	if i.i >= len(arr) {
		return nil, raiseMissing(cfg, i.String())
	}
	return arr[i.i], nil
}

func (p cfgPath) SetValue(cfg *Config, opt *options, val value) Error {
	fields := p.fields
	node := value(cfgSub{cfg})

	// 1. iterate until intermediate node not having some required child node
	for ; len(fields) > 1; fields = fields[1:] {
		field := fields[0]
		v, err := field.GetValue(opt, node)
		if err != nil {
			if err.Reason() == ErrMissing {
				break
			}
			return err
		}

		if _, isNil := v.(*cfgNil); v == nil || isNil {
			break
		}
		node = v
	}

	// 2. build intermediate nodes from bottom up

	for ; len(fields) > 1; fields = fields[:len(fields)-1] {
		field := fields[len(fields)-1]

		next := New()
		next.metadata = val.meta()
		v := cfgSub{next}
		if err := field.SetValue(opt, v, val); err != nil {
			return err
		}
		val = v
	}

	// 3. insert new sub-tree into config
	return fields[0].SetValue(opt, node, val)
}

func (n namedField) SetValue(opts *options, elem value, v value) Error {
	sub, ok := elem.(cfgSub)
	if !ok {
		return raiseExpectedObject(opts, elem)
	}

	sub.c.fields.set(n.name, v)
	v.SetContext(context{parent: elem, field: n.name})
	return nil
}

func (i idxField) SetValue(opts *options, elem value, v value) Error {
	sub, ok := elem.(cfgSub)
	if !ok {
		return raiseExpectedObject(opts, elem)
	}

	sub.c.fields.setAt(i.i, v)
	v.SetContext(context{parent: elem, field: i.String()})
	return nil
}
