package server

import (
	"strings"
)

type tree struct {
	root *node // Root node
}

// node is a single element within the tree
type node struct {
	parent   *node
	entry    *entry           // entry
	children map[string]*node // Children nodes
}

// entry represents the key-value pair contained within nodes
type entry struct {
	key   string
	value *template
}

func (n *node) FindChild(key string) *node {
	child, ok := n.children[key]
	if ok {
		return child
	}
	return nil
}

func (n *node) AddChild(key string) *node {
	temp := &node{
		parent:   n,
		children: make(map[string]*node),
	}

	n.children[key] = temp
	return temp
}

func (n *node) GetTemplate() *template {
	if n.entry != nil {
		return n.entry.value
	}

	return nil
}

func (n *node) Search(parts []string) *template {
	if len(parts) == 0 || len(n.children) == 0 {
		return n.GetTemplate()
	}
	child := n.FindChild(parts[0])
	if child == nil {
		child = n.FindChild("*")
	}

	if child != nil {
		return child.Search(parts[1:])
	}

	return n.GetTemplate()
}

func (t *tree) Insert(filter string, template template) {
	cur := t.root
	parts := strings.Split(filter, ".")
	for _, part := range parts {
		child := cur.FindChild(part)
		if child == nil {
			child = cur.AddChild(part)
			if child != nil && part == "*" {
				child.entry = cur.entry
			}
		}
		cur = child
	}

	if cur != nil {
		cur.entry = &entry{
			key:   parts[len(parts)-1],
			value: &template,
		}
	}
}

func (t *tree) Search(parts []string) *template {
	return t.root.Search(parts)
}

func (t *tree) Delete(filter string) {
	parts := strings.Split(filter, ".")
	cur := t.root
	for _, part := range parts {
		child := cur.FindChild(part)
		if child == nil {
			// entry does not exist
			return
		}
		cur = child
	}

	// we are in the last element at this point
	if cur != nil {
		// There are more entries, so just make the template nil and make all subsequent '*' templates nil
		if len(cur.children) != 0 {
			cur.entry = nil
			doBreak := false
			temp := cur
			for doBreak == false {
				child := temp.FindChild("*")
				if child != nil {
					child.entry = nil
					temp = child
				} else {
					doBreak = true
				}
			}
		} else {
			// Keep removing parts till there is no more childless entry
			temp := cur
			length := len(parts)
			for temp != t.root {
				parent := temp.parent
				// Remove only if there is only one child for the parent
				if len(parent.children) == 1 {
					delete(parent.children, parts[length-1])
					temp = parent
					length = length - 1
				} else {
					break
				}

			}

		}
	}
}

func NewTree(defaultTemplate template) *tree {
	root := &node{
		entry: &entry{
			key:   "*",
			value: &defaultTemplate,
		},
		children: make(map[string]*node),
		parent:   nil,
	}

	return &tree{
		root: root,
	}
}
