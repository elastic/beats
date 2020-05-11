// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package diag

import (
	"sort"
	"strings"
)

// Context represents the diagnostic context tree.
// A context contains a number of user defined and standardized fields,
// plus a reference a 'before' context and an 'after' context.
// The current context and the 'after' context overwrite all fields
// written to the 'before' context.
// The 'after' context overwrites all fields added to the 'before' or the
// current context.
type Context struct {
	totUser       int
	totStd        int
	fields        []Field
	mode          fieldSel
	before, after *Context
}

// Visitor can be used to iterate all fields in a context.
// Shadowed fields will only be reported once.
// Use with (*Context).VisitKeyValues to collect a flattened list of key value pairs.
// Use VisitStructured to recursively iterate the context.
type Visitor interface {
	OnObjStart(key string) error
	OnObjEnd() error
	OnValue(key string, v Value) error
}

// order represents the global flattened order of all fields in a Context tree.
// The Len() reports the number of fields in a context.
//
// The ith field is accessed via `order.ctx[i].fields[ctx.idx[i]]`
type order struct {
	idx []int      // index of field in context in 'fields' of the i-th context
	ctx []*Context // pointer to context the i-th field can be found in
}

// view is a temporary snapshot of a context with an applied order.
// The view object can be used to iterate through all fields in a context.
type view struct {
	Ctx   *Context
	order order
}

// fieldSel configures a context its field 'filtering' in case a projection
// like (*Context).User or (*Context).Standardized has been applied to a context.
type fieldSel uint8

const (
	allFields          fieldSel = standardizedFields | userFields | fieldsClosure
	standardizedFields fieldSel = 1 << 0 // list standardized fields
	userFields         fieldSel = 1 << 1 // list used fields
	fieldsClosure      fieldSel = 1 << 2 // use fields in before/after contexts
)

// NewContext creates a new context adding a before and after context
// for shadoing fields. When creating a context a snapshot of the before and
// after contexts is taken, such that they can still be manipulated, without
// affecting the current context.
func NewContext(before, after *Context) *Context {
	totStd, totUser := 0, 0
	if before != nil {
		totStd += before.totStd
		totUser += before.totUser
	}
	if after != nil {
		totStd += after.totStd
		totUser += after.totUser
	}

	return &Context{
		totStd:  totStd,
		totUser: totUser,
		before:  makeSnapshot(before),
		after:   makeSnapshot(after),
		mode:    allFields,
	}
}

// Len reports the number of fields in the current context. If two fields have
// the same key, both will be reported.
func (c *Context) Len() int {
	if c == nil {
		return 0
	}
	return c.totUser + c.totStd
}

// Local projection, that returns a snapshot of the current context
// without before and after contexts.
func (c *Context) Local() *Context {
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

	return &Context{
		totUser: totUser,
		totStd:  totStd,
		fields:  c.fields,
		mode:    c.mode &^ fieldsClosure,
	}
}

// User projection, that will only contain user fields. All standardized fields
// (even in after/before context) will be ignored.
func (c *Context) User() *Context {
	return &Context{
		totUser: c.totUser,
		totStd:  0,
		fields:  c.fields,
		mode:    c.mode &^ standardizedFields,
		before:  c.before,
		after:   c.after,
	}
}

// Standardized projection, that will only contain standardized fields. All
// user fields (even in after/before context) will be ignored.
func (c *Context) Standardized() *Context {
	return &Context{
		totStd:  c.totStd,
		totUser: 0,
		fields:  c.fields,
		mode:    c.mode &^ userFields,
		before:  c.before,
		after:   c.after,
	}
}

func makeSnapshot(ctx *Context) *Context {
	if ctx.Len() == 0 {
		return nil
	}
	if len(ctx.fields) > 0 {
		return cloneContext(ctx)
	}

	if ctx.before.Len() == 0 {
		return ctx.after
	} else if ctx.after.Len() == 0 {
		return ctx.before
	} else {
		return cloneContext(ctx)
	}
}

func cloneContext(ctx *Context) *Context {
	snapshot := *ctx
	return &snapshot
}

// Add creates and adds a new user field to the current context.
func (c *Context) Add(key string, value Value) {
	c.AddField(Field{Key: key, Value: value})
}

// AddField adds a new field to the current context.
func (c *Context) AddField(f Field) {
	c.fields = append(c.fields, f)
	if f.Standardized {
		c.totStd++
	} else {
		c.totUser++
	}
}

// AddFields adds a list a variable number of fields to the current context.
func (c *Context) AddFields(fs ...Field) {
	c.fields = append(c.fields, fs...)
	for i := range fs {
		if fs[i].Standardized {
			c.totStd++
		} else {
			c.totUser++
		}
	}
}

// AddAll adds a list of fields or key value pairs to the current context.
// For example ctx.AddAll("a": 1, diag.String("b", "test")) will
// create a context with the two fields a=1 and b=test.
func (c *Context) AddAll(args ...interface{}) {
	for i := 0; i < len(args); {
		arg := args[i]
		switch v := arg.(type) {
		case string:
			switch val := args[i+1].(type) {
			case Value:
				c.Add(v, val)
			default:
				c.AddField(Any(v, args[i+1]))
			}

			i += 2
		case Field:
			c.AddField(v)
			i++
		}
	}
}

// VisitKeyValues reports unique fields to the given visitor. Keys will be
// flattened, only calling the OnValue callback on the given visitor.
func (c *Context) VisitKeyValues(v Visitor) error {
	view := newView(c, c.mode&fieldsClosure == 0)
	return view.VisitKeyValues(v)
}

// VisitStructured reports the context its structure to the visitor.
// Fields having the same prefix separated by dots will be combined into
// a common object.
func (c *Context) VisitStructured(v Visitor) error {
	view := newView(c, c.mode&fieldsClosure == 0)
	return view.VisitStructured(v)
}

func newView(ctx *Context, localOnly bool) *view {
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

	// TODO: this function keeps track of the number of ObjStart and ObjEnd events
	// by scanning for `.` in the fields names.
	// Instead of scanning let's check if we can store the string index in
	// a slice (use as a stack) or if we should try to use recursion in order
	// to track the 'index' stack on the go-routine stack itself.

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

func (o *order) init(ctx *Context, localOnly, user, std bool) {
	l := ctx.Len()
	if l == 0 {
		return
	}

	o.idx = make([]int, l)
	o.ctx = make([]*Context, l)

	n := index(o, ctx, localOnly, user, std)
	o.idx = o.idx[:n]
	o.ctx = o.ctx[:n]
	sort.Stable(o)
}

func index(o *order, ctx *Context, localOnly, user, std bool) int {
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
