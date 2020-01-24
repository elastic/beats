// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const selectorSep = "."

// Selector defines a path to access an element in the Tree, currently selectors only works when the
// target is a Dictionary, accessing list values are not currently supported by any methods using
// selectors.
type Selector = string

var (
	trueVal  = []byte{1}
	falseVal = []byte{0}
)

// Node represents a node in the configuration Tree a Node can point to one or multiples children
// nodes.
type Node interface {
	fmt.Stringer

	// Find search a string in the current node.
	Find(string) (Node, bool)

	// Value returns the value of the node.
	Value() interface{}

	//Close clones the current node.
	Clone() Node

	// Hash compute a sha256 hash of the current node and recursively call any children.
	Hash() []byte
}

// AST represents a raw configuration which is purely data, only primitives are currently supported,
// Int, float, string and bool. Complex are not taking into consideration. The Tree allow to define
// operation on the retrieves options in a more structured way. We are using this new structure to
// create filtering rules or manipulation rules to convert a configuration to another format.
type AST struct {
	root Node
}

func (a *AST) String() string {
	return "{AST:" + a.root.String() + "}"
}

// Dict represents a dictionary in the Tree, where each key is a entry into an array. The Dict will
// keep the ordering.
type Dict struct {
	value []Node
}

// Find takes a string which is a key and try to find the elements in the associated K/V.
func (d *Dict) Find(key string) (Node, bool) {
	for _, i := range d.value {
		if i.(*Key).name == key {
			return i, true
		}
	}
	return nil, false
}

func (d *Dict) String() string {
	var sb strings.Builder
	for i := 0; i < len(d.value); i++ {
		sb.WriteString("{")
		sb.WriteString(d.value[i].String())
		sb.WriteString("}")
		if i < len(d.value)-1 {
			sb.WriteString(",")
		}
	}
	return sb.String()
}

// Value returns the value of dict which is a slice of node.
func (d *Dict) Value() interface{} {
	return d.value
}

// Clone clones the values and return a new dictionary.
func (d *Dict) Clone() Node {
	nodes := make([]Node, 0, len(d.value))
	for _, i := range d.value {
		nodes = append(nodes, i.Clone())
	}
	return &Dict{value: nodes}
}

// Hash compute a sha256 hash of the current node and recursively call any children.
func (d *Dict) Hash() []byte {
	h := sha256.New()
	for _, v := range d.value {
		h.Write(v.Hash())
	}
	return h.Sum(nil)
}

// Key represents a Key / value pair in the dictionary.
type Key struct {
	name  string
	value Node
}

func (k *Key) String() string {
	var sb strings.Builder
	sb.WriteString(k.name)
	sb.WriteString(":")
	if k.value == nil {
		sb.WriteString("nil")
	} else {
		sb.WriteString(k.value.String())
	}
	return sb.String()
}

// Find finds a key in a Dictionary or a list.
func (k *Key) Find(key string) (Node, bool) {
	switch v := k.value.(type) {
	case *Dict:
		return v.Find(key)
	case *List:
		return v.Find(key)
	default:
		return nil, false
	}
}

// Value returns the raw value.
func (k *Key) Value() interface{} {
	return k.value
}

// Clone returns a clone of the current key and his embedded values.
func (k *Key) Clone() Node {
	if k.value != nil {
		return &Key{name: k.name, value: k.value.Clone()}
	}

	return &Key{name: k.name, value: nil}
}

// Hash compute a sha256 hash of the current node and recursively call any children.
func (k *Key) Hash() []byte {
	h := sha256.New()
	h.Write([]byte(k.name))
	if k.value != nil {
		h.Write(k.value.Hash())
	}
	return h.Sum(nil)
}

// List represents a slice in our Tree.
type List struct {
	value []Node
}

func (l *List) String() string {
	var sb strings.Builder
	for i := 0; i < len(l.value); i++ {
		sb.WriteString("[")
		sb.WriteString(l.value[i].String())
		sb.WriteString("]")
		if i < len(l.value)-1 {
			sb.WriteString(",")
		}
	}
	return sb.String()
}

// Hash compute a sha256 hash of the current node and recursively call any children.
func (l *List) Hash() []byte {
	h := sha256.New()
	for _, v := range l.value {
		h.Write(v.Hash())
	}

	return h.Sum(nil)
}

// Find takes an index and return the values at that index.
func (l *List) Find(idx string) (Node, bool) {
	i, err := strconv.Atoi(idx)
	if err != nil {
		return nil, false
	}
	if i > len(l.value) || i < len(l.value) {
		return nil, false
	}

	return l.value[i], true
}

// Value returns the raw value.
func (l *List) Value() interface{} {
	return l.value
}

// Clone clones a new list and the clone items.
func (l *List) Clone() Node {
	nodes := make([]Node, 0, len(l.value))
	for _, i := range l.value {
		nodes = append(nodes, i.Clone())
	}
	return &List{value: nodes}
}

// StrVal represents a string.
type StrVal struct {
	value string
}

// Find receive a key and return false since the node is not a List or Dict.
func (s *StrVal) Find(key string) (Node, bool) {
	return nil, false
}

func (s *StrVal) String() string {
	return s.value
}

// Value returns the value.
func (s *StrVal) Value() interface{} {
	return s.value
}

// Clone clone the value.
func (s *StrVal) Clone() Node {
	k := *s
	return &k
}

// Hash we return the byte slice of the string.
func (s *StrVal) Hash() []byte {
	return []byte(s.value)
}

// IntVal represents an int.
type IntVal struct {
	value int
}

// Find receive a key and return false since the node is not a List or Dict.
func (s *IntVal) Find(key string) (Node, bool) {
	return nil, false
}

func (s *IntVal) String() string {
	return strconv.Itoa(s.value)
}

// Value returns the value.
func (s *IntVal) Value() interface{} {
	return s.value
}

// Clone clone the value.
func (s *IntVal) Clone() Node {
	k := *s
	return &k
}

// Hash we convert the value into a string and return the byte slice.
func (s *IntVal) Hash() []byte {
	return []byte(s.String())
}

// UIntVal represents an int.
type UIntVal struct {
	value uint64
}

// Find receive a key and return false since the node is not a List or Dict.
func (s *UIntVal) Find(key string) (Node, bool) {
	return nil, false
}

func (s *UIntVal) String() string {
	return strconv.FormatUint(s.value, 10)
}

// Value returns the value.
func (s *UIntVal) Value() interface{} {
	return s.value
}

// Clone clone the value.
func (s *UIntVal) Clone() Node {
	k := *s
	return &k
}

// Hash we convert the value into a string and return the byte slice.
func (s *UIntVal) Hash() []byte {
	return []byte(s.String())
}

// FloatVal represents a float.
// NOTE: We will convert float32 to a float64.
type FloatVal struct {
	value float64
}

// Find receive a key and return false since the node is not a List or Dict.
func (s *FloatVal) Find(key string) (Node, bool) {
	return nil, false
}

func (s *FloatVal) String() string {
	return fmt.Sprintf("%f", s.value)
}

// Value return the raw value.
func (s *FloatVal) Value() interface{} {
	return s.value
}

// Clone clones the value.
func (s *FloatVal) Clone() Node {
	k := *s
	return &k
}

// Hash return a string representation of the value, we try to return the minimal precision we can.
func (s *FloatVal) Hash() []byte {
	return []byte(strconv.FormatFloat(s.value, 'f', -1, 64))
}

// BoolVal represents a boolean in our Tree.
type BoolVal struct {
	value bool
}

// Find receive a key and return false since the node is not a List or Dict.
func (s *BoolVal) Find(key string) (Node, bool) {
	return nil, false
}

func (s *BoolVal) String() string {
	if s.value == true {
		return "true"
	}
	return "false"
}

// Value returns the value.
func (s *BoolVal) Value() interface{} {
	return s.value
}

// Clone clones the value.
func (s *BoolVal) Clone() Node {
	k := *s
	return &k
}

// Hash returns a single byte to represent the boolean value.
func (s *BoolVal) Hash() []byte {
	if s.value {
		return trueVal
	}
	return falseVal
}

// NewAST takes a map and convert it to an internal Tree, allowing us to executes rules on the
// data to shape it in a different way or to filter some of the information.
func NewAST(m map[string]interface{}) (*AST, error) {
	val := reflect.ValueOf(m)
	root, err := load(val)
	if err != nil {
		return nil, fmt.Errorf("could not parse configuration into a tree, error: %+v", err)
	}
	return &AST{root: root}, nil
}

// MustNewAST create a new AST based on a map[string]iface and panic on any errors.
func MustNewAST(m map[string]interface{}) *AST {
	v, err := NewAST(m)
	if err != nil {
		panic(err)
	}
	return v
}

func load(val reflect.Value) (Node, error) {
	val = lookupVal(val)

	switch val.Kind() {
	case reflect.Map:
		return loadMap(val)
	case reflect.Slice, reflect.Array:
		return loadSliceOrArray(val)
	case reflect.String:
		return &StrVal{value: val.Interface().(string)}, nil
	case reflect.Int, reflect.Int64:
		return &IntVal{value: val.Interface().(int)}, nil
	case reflect.Uint:
		return &UIntVal{value: uint64(val.Interface().(uint))}, nil
	case reflect.Uint64:
		return &UIntVal{value: val.Interface().(uint64)}, nil
	case reflect.Float64:
		return &FloatVal{value: val.Interface().(float64)}, nil
	case reflect.Float32:
		return &FloatVal{value: float64(val.Interface().(float32))}, nil
	case reflect.Bool:
		return &BoolVal{value: val.Interface().(bool)}, nil
	default:
		if val.IsNil() {
			return nil, nil
		}
		return nil, fmt.Errorf("unknown type %T for %+v", val.Interface(), val)
	}
}

// Accept takes a visitor and will visit each node of the Tree while calling the right methods on
// the visitor.
// NOTE(ph): Some operation could be refactored to use a visitor, I plan to add a checksum visitor.
func (a *AST) Accept(visitor Visitor) {
	a.dispatch(a.root, visitor)
}

func (a *AST) dispatch(n Node, visitor Visitor) {
	switch t := n.(type) {
	case *Dict:
		visitorDict := visitor.OnDict()
		for _, child := range t.value {
			key := child.(*Key)
			visitorDict.OnKey(key.name)
			subvisitor := visitorDict.Visitor()
			a.dispatch(key.value, subvisitor)
			visitorDict.OnValue(subvisitor)
		}
		visitorDict.OnComplete()
	case *List:
		visitorList := visitor.OnList()
		for _, child := range t.value {
			subvisitor := visitorList.Visitor()
			a.dispatch(child, subvisitor)
			visitorList.OnValue(subvisitor)
		}
		visitorList.OnComplete()
	case *StrVal:
		visitor.OnStr(t.value)
	case *IntVal:
		visitor.OnInt(t.value)
	case *UIntVal:
		visitor.OnUInt(t.value)
	case *BoolVal:
		visitor.OnBool(t.value)
	case *FloatVal:
		visitor.OnFloat(t.value)
	}
}

// Clone clones the object.
func (a *AST) Clone() *AST {
	return &AST{root: a.root.Clone()}
}

// Hash calculates a hash from all the included nodes in the tree.
func (a *AST) Hash() []byte {
	return a.root.Hash()
}

// HashStr return the calculated hash as a base64 url encoded string.
func (a *AST) HashStr() string {
	return base64.URLEncoding.EncodeToString(a.root.Hash())
}

// Equal check if two AST are equals by using the computed hash.
func (a *AST) Equal(other *AST) bool {
	return bytes.Equal(a.Hash(), other.Hash())
}

// MarshalYAML defines how to marshal the Tree, it will convert the tree to a
// map[string]interface{}.
func (a *AST) MarshalYAML() (interface{}, error) {
	m := &MapVisitor{}
	a.Accept(m)
	return m.Content, nil
}

// MarshalJSON concerts an AST to a valid JSON.
func (a *AST) MarshalJSON() ([]byte, error) {
	m := &MapVisitor{}
	a.Accept(m)

	b, err := json.Marshal(m.Content)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func splitPath(s Selector) []string {
	return strings.Split(s, selectorSep)
}

func loadMap(val reflect.Value) (Node, error) {
	node := &Dict{}

	mapKeys := val.MapKeys()
	names := make([]string, 0, len(mapKeys))
	for _, aKey := range mapKeys {
		names = append(names, aKey.Interface().(string))
	}
	sort.Strings(names)

	for _, name := range names {
		aValue, err := load(val.MapIndex(reflect.ValueOf(name)))
		if err != nil {
			return nil, err
		}

		keys := strings.Split(name, selectorSep)
		if !isDictOrKey(aValue) {
			node.value = append(node.value, &Key{name: name, value: aValue})
			continue
		}

		// get last known existing node
		var lastKnownKeyIdx int
		var knownNode Node = node
		for i, k := range keys {
			n, isDict := knownNode.Find(k)
			if !isDict {
				break
			}

			lastKnownKeyIdx = i
			knownNode = n
		}

		// Produce remainder
		restKeys := keys[lastKnownKeyIdx+1:]
		restDict := &Dict{}
		if len(restKeys) == 0 {
			if avd, ok := aValue.(*Dict); ok {
				restDict.value = avd.value
			} else if avd, ok := aValue.(*Key); ok {
				restDict.value = []Node{avd.value}
			} else {
				restDict.value = append(restDict.value, aValue)
			}
		} else {
			for i := len(restKeys) - 1; i >= 0; i-- {
				if len(restDict.value) == 0 {
					// this is the first one
					restDict.value = []Node{&Key{name: restKeys[i], value: aValue}}
					continue
				}

				restDict.value = []Node{&Key{name: restKeys[i], value: restDict.Clone()}}
			}
		}

		// Attach remainder to last known node
		restKey := &Key{name: keys[lastKnownKeyIdx], value: restDict}
		if knownNodeDict, ok := knownNode.(*Dict); ok {
			knownNodeDict.value = append(knownNodeDict.value, restKey)
		} else if knownNodeKey, ok := knownNode.(*Key); ok {
			dict, ok := knownNodeKey.value.(*Dict)
			if ok {
				dict.value = append(dict.value, restDict.value...)
			}
		}
	}

	return node, nil
}

func isDictOrKey(val Node) bool {
	if _, ok := val.(*Key); ok {
		return true
	}
	if _, ok := val.(*Dict); ok {
		return true
	}
	return false
}

func loadSliceOrArray(val reflect.Value) (Node, error) {
	node := &List{}
	for i := 0; i < val.Len(); i++ {
		aValue, err := load(val.Index(i))
		if err != nil {
			return nil, err
		}
		node.value = append(node.value, aValue)
	}
	return node, nil
}

func lookupVal(val reflect.Value) reflect.Value {
	for (val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && !val.IsNil() {
		val = val.Elem()
	}
	return val
}

// Select takes an AST and a selector and will return a sub AST based on the selector path, will
// return false if the path could not be found.
func Select(a *AST, selector Selector) (*AST, bool) {
	var appendTo []Node

	// Run through the graph and find matching nodes.
	current := a.root
	for _, part := range splitPath(selector) {
		n, ok := current.Find(part)
		if !ok {
			return nil, false
		}

		current = n
		appendTo = append(appendTo, current)
	}

	newAST := &Dict{}
	d := newAST
	for idx, n := range appendTo {
		d.value = append(d.value, n)
		// Prepare to add the next level.
		if idx < len(appendTo)-1 {
			node := n.(*Key)
			subdict := &Dict{}
			node.value = subdict
			d = subdict
		}
	}
	return &AST{root: newAST}, true
}

// Lookup accept an AST and a selector and return the matching Node at that position.
func Lookup(a *AST, selector Selector) (Node, bool) {
	// Run through the graph and find matching nodes.
	current := a.root
	for _, part := range splitPath(selector) {
		n, ok := current.Find(part)
		if !ok {
			return nil, false
		}

		current = n
	}

	return current, true
}

// Insert inserts a node into an existing AST, will return and error if the target position cannot
// accept a new node.
func Insert(a *AST, node Node, to Selector) error {
	current := a.root
	for _, part := range splitPath(to) {
		n, ok := current.Find(part)
		if !ok {
			switch t := current.(type) {
			case *Dict:
				newNode := &Key{name: part, value: &Dict{}}
				t.value = append(t.value, newNode)

				sort.Slice(t.value, func(i, j int) bool {
					return t.value[i].(*Key).name < t.value[j].(*Key).name
				})

				current = newNode
				continue
			default:
				return fmt.Errorf("expecting Dict and received %T", t)
			}
		}

		current = n
	}

	// Apply the current node and replace any existing elements,
	// that could exist after the selector.
	d, ok := current.(*Key)
	if !ok {
		return fmt.Errorf("expecting Key and received %T", current)
	}

	switch node.(type) {
	case *List:
		d.value = node
	case *Dict:
		d.value = &Dict{[]Node{node}}
	default:
		d.value = &Dict{[]Node{node}}
	}
	return nil
}

// Combine takes two AST and try to combine both of them into a single AST, notes that this operation
// is not a merges and will return an error if position to merge are not compatible type or
// if the key is already present in the target AST. This method useful if you use the Select methods
// to create 2 different sub AST and want to merge them together again.
func Combine(a, b *AST) (*AST, error) {
	newAST := &AST{}
	if reflect.TypeOf(b.root) != reflect.TypeOf(b.root) {
		return nil, fmt.Errorf("incompatible node type to combine, received %T and %T", a, b)
	}

	switch t := a.root.(type) {
	case *Dict:
		newAST.root = t
		for _, element := range b.root.Value().([]Node) {
			key := element.(*Key)
			_, ok := t.Find(key.name)
			if ok {
				return nil, fmt.Errorf("could not combine tree, key %s present in both trees", key.name)
			}
			t.value = append(t.value, key)
		}
	case *List:
		newAST.root = t
		t.value = append(t.value, b.root.(*List).value...)
	}

	return newAST, nil
}

// CompOp is operation used for comparing counts in CountComp
type CompOp func(actual int) bool

// CountComp is a comparison operation which returns true if compareOp evaluates true.
// provided to compareOp is the actual count of elements within a specified paths.
func CountComp(ast *AST, selector Selector, compareOp CompOp) bool {
	var actualCount int
	node, ok := Lookup(ast, selector)
	if ok {
		switch t := node.Value().(type) {
		case *Key:
			actualCount = 1
		case *Dict:
			actualCount = len(t.value)
		case *List:
			actualCount = len(t.value)
		default:
			actualCount = 1
		}
	}

	return compareOp(actualCount)
}

// Map transforms the AST into a map[string]interface{} and will abort and return any errors related
// to type conversion.
func (a *AST) Map() (map[string]interface{}, error) {
	m := &MapVisitor{}
	a.Accept(m)
	mapped, ok := m.Content.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not convert to map[string]iface, type is %T", m.Content)
	}
	return mapped, nil
}
