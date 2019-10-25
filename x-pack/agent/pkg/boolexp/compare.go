// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package boolexp

import "fmt"

type operand interface{}

type compare func(left, right operand) (bool, error)

func compareEQ(left, right operand) (bool, error) {
	switch v := left.(type) {
	case bool:
		rV, ok := right.(bool)
		if !ok {
			return false, nil
		}
		if rV == v {
			return true, nil
		}
		return false, nil
	case int:
		switch rv := right.(type) {
		case int:
			return v == rv, nil
		case float64:
			// TODO: check overflow, weird things will happen with precision here.
			// use modf
			return float64(v) == rv, nil
		default:
			return false, fmt.Errorf(
				"compare: ==, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v == float64(rv), nil
		case float64:
			return v == rv, nil
		default:
			return false, fmt.Errorf(
				"compare: ==, incompatible type to compare both operand must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case string:
		rV, ok := right.(string)
		if !ok {
			return false, nil
		}
		if rV == v {
			return true, nil
		}
		return false, nil
	default:
		return false, fmt.Errorf(
			"compare: ==, incompatible type to compare, left=%T, right=%T",
			left,
			right,
		)
	}
}

func compareNEQ(left, right operand) (bool, error) {
	switch v := left.(type) {
	case bool:
		rV, ok := right.(bool)
		if !ok {
			return false, nil
		}
		if rV == v {
			return false, nil
		}
		return true, nil
	case int:
		switch rv := right.(type) {
		case int:
			return v != rv, nil
		case float64:
			// TODO: check overflow, weird things will happen with precision here.
			// use modf
			return float64(v) != rv, nil
		default:
			return false, fmt.Errorf(
				"compare: ==, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v != float64(rv), nil
		case float64:
			return v != rv, nil
		default:
			return false, fmt.Errorf(
				"compare: ==, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case string:
		rV, ok := right.(string)
		if !ok {
			return false, nil
		}
		if rV == v {
			return false, nil
		}
		return true, nil
	default:
		return false, fmt.Errorf(
			"compare: !=, incompatible type to compare, left=%T, right=%T",
			left,
			right,
		)
	}
}

func compareLT(left, right operand) (bool, error) {
	switch v := left.(type) {
	case int:
		switch rv := right.(type) {
		case int:
			return v < rv, nil
		case float64:
			return float64(v) < rv, nil
		default:
			return false, fmt.Errorf(
				"compare: <, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v < float64(rv), nil
		case float64:
			return v < rv, nil
		default:
			return false, fmt.Errorf(
				"compare: <, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	default:
		return false, fmt.Errorf(
			"compare: <, incompatible type to compare both operands must be numbers, left=%T, right=%T",
			left,
			right,
		)
	}
}

func compareLTE(left, right operand) (bool, error) {
	switch v := left.(type) {
	case int:
		switch rv := right.(type) {
		case int:
			return v <= rv, nil
		case float64:
			return float64(v) <= rv, nil
		default:
			return false, fmt.Errorf(
				"compare: <=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v <= float64(rv), nil
		case float64:
			return v <= rv, nil
		default:
			return false, fmt.Errorf(
				"compare: <=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	default:
		return false, fmt.Errorf(
			"compare: <=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
			left,
			right,
		)
	}
}

func compareGT(left, right operand) (bool, error) {
	switch v := left.(type) {
	case int:
		switch rv := right.(type) {
		case int:
			return v > rv, nil
		case float64:
			return float64(v) > rv, nil
		default:
			return false, fmt.Errorf(
				"compare: >, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v > float64(rv), nil
		case float64:
			return v > rv, nil
		default:
			return false, fmt.Errorf(
				"compare: >, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	default:
		return false, fmt.Errorf(
			"compare: >, incompatible type to compare both operands must be numbers, left=%T, right=%T",
			left,
			right,
		)
	}
}

func compareGTE(left, right operand) (bool, error) {
	switch v := left.(type) {
	case int:
		switch rv := right.(type) {
		case int:
			return v >= rv, nil
		case float64:
			return float64(v) >= rv, nil
		default:
			return false, fmt.Errorf(
				"compare: >=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	case float64:
		switch rv := right.(type) {
		case int:
			return v >= float64(rv), nil
		case float64:
			return v >= rv, nil
		default:
			return false, fmt.Errorf(
				"compare: >=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
				left,
				right,
			)
		}
	default:
		return false, fmt.Errorf(
			"compare: >=, incompatible type to compare both operands must be numbers, left=%T, right=%T",
			left,
			right,
		)
	}
}

type logical func(left, right operand) (bool, error)

func logicalAND(left, right operand) (bool, error) {
	return left.(bool) && right.(bool), nil
}

func logicalOR(left, right operand) (bool, error) {
	return left.(bool) == true || right.(bool), nil
}
