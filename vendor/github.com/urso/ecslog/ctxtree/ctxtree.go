package ctxtree

import (
	"sort"
	"strings"

	"github.com/urso/ecslog/fld"
)

type Ctx struct {
	totUser       int
	totStd        int
	fields        []fld.Field
	mode          fieldSel
	before, after *Ctx
}

type Visitor interface {
	OnObjStart(key string) error
	OnObjEnd() error
	OnValue(key string, v fld.Value) error
}

type order struct {
	idx []int
	ctx []*Ctx
}

type view struct {
	Ctx   *Ctx
	order order
}

type fieldSel uint8

const (
	allFields          fieldSel = standardizedFields | userFields | fieldsClosure
	standardizedFields fieldSel = 1
	userFields         fieldSel = 2
	fieldsClosure      fieldSel = 4
)

func Make(before, after *Ctx) Ctx {
	totStd, totUser := 0, 0
	if before != nil {
		totStd += before.totStd
		totUser += before.totUser
	}
	if after != nil {
		totStd += after.totStd
		totUser += after.totUser
	}

	return Ctx{
		totStd:  totStd,
		totUser: totUser,
		before:  makeSnapshot(before),
		after:   makeSnapshot(after),
		mode:    allFields,
	}
}

func New(before, after *Ctx) *Ctx {
	tmp := Make(before, after)
	return &tmp
}

func makeSnapshot(ctx *Ctx) *Ctx {
	if ctx.Len() == 0 {
		return nil
	}
	snapshot := *ctx
	return &snapshot
}

func (c *Ctx) AddAll(args ...interface{}) {
	for i := 0; i < len(args); {
		arg := args[i]
		switch v := arg.(type) {
		case string:
			switch val := args[i+1].(type) {
			case fld.Value:
				c.Add(v, val)
			default:
				c.AddField(fld.Any(v, args[i+1]))
			}

			i += 2
		case fld.Field:
			c.AddField(v)
			i++
		}
	}
}

func (c *Ctx) Add(key string, value fld.Value) {
	c.AddField(fld.Field{Key: key, Value: value})
}

func (c *Ctx) AddField(f fld.Field) {
	c.fields = append(c.fields, f)
	if f.Standardized {
		c.totStd++
	} else {
		c.totUser++
	}
}

func (c *Ctx) AddFields(fs ...fld.Field) {
	c.fields = append(c.fields, fs...)
	for i := range fs {
		if fs[i].Standardized {
			c.totStd++
		} else {
			c.totUser++
		}
	}
}

func (c *Ctx) Local() Ctx {
	totUser, totStd := 0, 0
	for i := range c.fields {
		if c.fields[i].Standardized {
			totStd++
		} else {
			totUser++
		}
	}

	if (c.mode & userFields) == 0 {
		totUser = 0
	}
	if (c.mode & standardizedFields) == 0 {
		totStd = 0
	}

	return Ctx{
		totUser: totUser,
		totStd:  totStd,
		fields:  c.fields,
		mode:    c.mode &^ fieldsClosure,
	}
}

func (c *Ctx) User() Ctx {
	return Ctx{
		totUser: c.totUser,
		totStd:  0,
		fields:  c.fields,
		mode:    c.mode &^ standardizedFields,
		before:  c.before,
		after:   c.after,
	}
}

func (c *Ctx) Standardized() Ctx {
	return Ctx{
		totStd:  c.totStd,
		totUser: 0,
		fields:  c.fields,
		mode:    c.mode &^ userFields,
		before:  c.before,
		after:   c.after,
	}
}

func (c *Ctx) Len() int {
	if c == nil {
		return 0
	}
	return c.totUser + c.totStd
}

func (c *Ctx) VisitKeyValues(v Visitor) error {
	view := newView(c, c.mode&fieldsClosure == 0)
	return view.VisitKeyValues(v)
}

func (c *Ctx) VisitStructured(v Visitor) error {
	view := newView(c, c.mode&fieldsClosure == 0)
	return view.VisitStructured(v)
}

func newView(ctx *Ctx, localOnly bool) *view {
	v := &view{Ctx: ctx}
	v.order.init(ctx, localOnly, true, true)
	return v
}

func (view *view) VisitKeyValues(v Visitor) error {
	o := &view.order
	L := o.Len()

	for i := 0; i < L; i++ {
		ctx, idx := o.ctx[i], o.idx[i]
		fld := &ctx.fields[idx]
		key := fld.Key

		if j := i + 1; j < L {
			other := o.key(j)
			if key == other {
				continue // ignore older duplicates
			}

			if strings.HasPrefix(other, key) && other[len(key)] == '.' {
				continue // ignore value if it's overwritten by an object
			}
		}

		if err := v.OnValue(key, fld.Value); err != nil {
			return err
		}
	}

	return nil
}

func (view *view) VisitStructured(v Visitor) error {
	o := &view.order
	L := o.Len()

	objPrefix := ""
	level := 0

	for i := 0; i < L; i++ {
		ctx, idx := o.ctx[i], o.idx[i]
		fld := &ctx.fields[idx]
		key := fld.Key

		if j := i + 1; j < L {
			other := o.key(j)
			if key == other {
				continue // ignore older duplicates
			}

			if strings.HasPrefix(other, key) && other[len(key)] == '.' {
				continue // ignore value if it's overwritten by an object
			}
		}

		// decrease object level until last and current key have same path prefix
		if L := commonPrefix(key, objPrefix); L < len(objPrefix) {
			for L > 0 && key[L-1] != '.' {
				L--
			}

			// remove levels
			if L > 0 {
				for delta := objPrefix[L:]; len(delta) > 0; {
					idx := strings.IndexRune(delta, '.')
					if idx < 0 {
						break
					}

					delta = delta[idx+1:]
					level--
					if err := v.OnObjEnd(); err != nil {
						return err
					}
				}

				objPrefix = key[:L]
			} else {
				for ; level > 0; level-- {
					if err := v.OnObjEnd(); err != nil {
						return err
					}
				}
				objPrefix = ""
			}
		}

		// increase object level
		for {
			start := len(objPrefix)
			idx := strings.IndexRune(key[start:], '.')
			if idx < 0 {
				break
			}

			level++
			objPrefix = key[:len(objPrefix)+idx+1]
			if err := v.OnObjStart(key[start : start+idx]); err != nil {
				return err
			}
		}

		k := key[len(objPrefix):]
		if err := v.OnValue(k, fld.Value); err != nil {
			return err
		}
	}

	for ; level > 0; level-- {
		if err := v.OnObjEnd(); err != nil {
			return err
		}
	}

	return nil
}

func (o *order) init(ctx *Ctx, localOnly, user, std bool) {
	l := ctx.Len()
	if l == 0 {
		return
	}

	o.idx = make([]int, l)
	o.ctx = make([]*Ctx, l)

	n := index(o, ctx, localOnly, user, std)
	o.idx = o.idx[:n]
	o.ctx = o.ctx[:n]
	sort.Stable(o)
}

func index(o *order, ctx *Ctx, localOnly, user, std bool) int {
	pos := 0
	user = user && (ctx.mode&userFields) == userFields
	std = std && (ctx.mode&standardizedFields) == standardizedFields

	if !localOnly {
		if L := ctx.before.Len(); L > 0 {
			pos += index(o, ctx.before, false, user, std)
		}
	}

	for i := range ctx.fields {
		idxField := (user && std) ||
			(user && !ctx.fields[i].Standardized) ||
			(std && ctx.fields[i].Standardized)
		if !idxField {
			continue
		}

		o.idx[pos] = i
		o.ctx[pos] = ctx
		pos++
	}

	if !localOnly {
		if L := ctx.after.Len(); L > 0 {
			tmp := &order{idx: o.idx[pos:], ctx: o.ctx[pos:]}
			pos += index(tmp, ctx.after, false, user, std)
		}
	}

	return pos
}

func (o *order) key(i int) string   { return o.ctx[i].fields[o.idx[i]].Key }
func (o *order) Len() int           { return len(o.idx) }
func (o *order) Less(i, j int) bool { return o.key(i) < o.key(j) }
func (o *order) Swap(i, j int) {
	o.idx[i], o.idx[j] = o.idx[j], o.idx[i]
	o.ctx[i], o.ctx[j] = o.ctx[j], o.ctx[i]
}

func commonPrefix(a, b string) int {
	end := len(a)
	if alt := len(b); alt < end {
		end = alt
	}

	for i := 0; i < end; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return end
}
