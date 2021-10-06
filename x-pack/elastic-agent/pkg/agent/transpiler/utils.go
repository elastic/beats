// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import "fmt"

// RenderInputs renders dynamic inputs section
func RenderInputs(inputs Node, varsArray []*Vars) (Node, error) {
	l, ok := inputs.Value().(*List)
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}
	nodes := []*Dict{}
	nodesMap := map[string]*Dict{}
	for _, vars := range varsArray {
		for _, node := range l.Value().([]Node) {
			dict, ok := node.Clone().(*Dict)
			if !ok {
				continue
			}
			n, err := dict.Apply(vars)
			if err == ErrNoMatch {
				// has a variable that didn't exist, so we ignore it
				continue
			}
			if err != nil {
				// another error that needs to be reported
				return nil, err
			}
			if n == nil {
				// condition removed it
				continue
			}
			dict = n.(*Dict)
			hash := string(dict.Hash())
			_, exists := nodesMap[hash]
			if !exists {
				nodesMap[hash] = dict
				nodes = append(nodes, dict)
			}
		}
	}
	nInputs := []Node{}
	for _, node := range nodes {
		nInputs = append(nInputs, promoteProcessors(node))
	}
	return NewList(nInputs), nil
}

func promoteProcessors(dict *Dict) *Dict {
	p := dict.Processors()
	if p == nil {
		return dict
	}
	var currentList *List
	current, ok := dict.Find("processors")
	if ok {
		currentList, ok = current.Value().(*List)
		if !ok {
			return dict
		}
	}
	ast, _ := NewAST(map[string]interface{}{
		"processors": p,
	})
	procs, _ := Lookup(ast, "processors")
	nodes := nodesFromList(procs.Value().(*List))
	if ok && currentList != nil {
		nodes = append(nodes, nodesFromList(currentList)...)
	}
	dictNodes := dict.Value().([]Node)
	set := false
	for i, node := range dictNodes {
		switch n := node.(type) {
		case *Key:
			if n.Name() == "processors" {
				dictNodes[i] = NewKey("processors", NewList(nodes))
				set = true
			}
		}
		if set {
			break
		}
	}
	if !set {
		dictNodes = append(dictNodes, NewKey("processors", NewList(nodes)))
	}
	return NewDict(dictNodes)
}

func nodesFromList(list *List) []Node {
	return list.Value().([]Node)
}
