// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import "fmt"

type injector interface {
	Inject(target []Node, source interface{}) []Node
	InjectItem(target []Node, source Node) []Node
	InjectCollection(target []Node, source []Node) []Node
}

func mergeStrategy(strategy string) injector {

	switch strategy {
	case "insert_before":
		return injectBeforeInjector{}
	case "insert_after":
		return injectAfterInjector{}
	case "replace":
		return replaceInjector{}
	case "noop":
		return noopInjector{}
	}

	return injectAfterInjector{}
}

type noopInjector struct{}

func (i noopInjector) Inject(target []Node, source interface{}) []Node {
	return inject(i, target, source)
}

func (noopInjector) InjectItem(target []Node, source Node) []Node { return target }

func (noopInjector) InjectCollection(target []Node, source []Node) []Node { return target }

type injectAfterInjector struct{}

func (i injectAfterInjector) Inject(target []Node, source interface{}) []Node {
	return inject(i, target, source)
}

func (injectAfterInjector) InjectItem(target []Node, source Node) []Node {
	return append(target, source)
}

func (injectAfterInjector) InjectCollection(target []Node, source []Node) []Node {
	return append(target, source...)
}

type injectBeforeInjector struct{}

func (i injectBeforeInjector) Inject(target []Node, source interface{}) []Node {
	return inject(i, target, source)
}

func (injectBeforeInjector) InjectItem(target []Node, source Node) []Node {
	return append([]Node{source}, target...)
}

func (injectBeforeInjector) InjectCollection(target []Node, source []Node) []Node {
	return append(source, target...)
}

type replaceInjector struct{}

func (i replaceInjector) Inject(target []Node, source interface{}) []Node {
	return inject(i, target, source)
}

func (replaceInjector) InjectItem(target []Node, source Node) []Node {
	return []Node{source}
}

func (replaceInjector) InjectCollection(target []Node, source []Node) []Node {
	return source
}

func inject(i injector, target []Node, source interface{}) []Node {
	if sourceCollection, ok := source.([]Node); ok {
		fmt.Printf(">>[%T] list of nodes %T %d\n", i, source, len(sourceCollection))
		return i.InjectCollection(target, sourceCollection)
	}

	if node, ok := source.(Node); ok {
		fmt.Printf(">> one of nodes %T\n", source)
		return i.InjectItem(target, node)
	}

	return target
}
