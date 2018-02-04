package lint

import (
	"go/token"
	"bytes"
	"strings"
	"sort"
	"go/parser"
	"go/format"
	"go/ast"
)

// Inspired by:
// https://github.com/golang/lint/blob/master/lint.go
// https://github.com/golang/example/tree/master/gotypes (CheckNilFuncComparison)
// https://github.com/golang/tools/commit/6d70fb2e85323e81c89374331d3d2b93304faa36 (tests)

type ErrorCheckLinter struct {
	positions []token.Pos
	fset *token.FileSet
}

func (l *ErrorCheckLinter) Lint(content string) ([]*Problem, error) {
	l.positions = []token.Pos{}

	l.fset = token.NewFileSet()

	file, err := parser.ParseFile(l.fset, "", content, parser.ParseComments)
	if err != nil {
		return []*Problem{}, err
	}

	var buf bytes.Buffer
	format.Node(&buf, l.fset, file)

	ast.Inspect(file, l.check)

	lines := strings.Split(content, "\n")

	var checkLines []int
	checkLinesMap := make(map[int]token.Position, 0)

	for _, tokenPos := range l.positions {
		position := l.fset.Position(tokenPos)
		checkLines = append(checkLines, position.Line)
		checkLinesMap[position.Line] = position
	}

	sort.Sort(sort.Reverse(sort.IntSlice(checkLines)))

	var problems []*Problem

	for _, line := range checkLines {
		if line < 2 {
			continue
		}

		if lines[line-2] == "" {
			problems = append(problems, &Problem{Position: checkLinesMap[line], Text: "Must be no newline before check error", LineText: lines[line-1]})
		}
	}

	return problems, nil
}

func (l *ErrorCheckLinter) check(n ast.Node) bool {
	e, ok := n.(*ast.BinaryExpr)
	if !ok {
		return true // not a binary operation
	}
	if e.Op != token.EQL && e.Op != token.NEQ {
		return true // not a comparison
	}

	var buf bytes.Buffer
	format.Node(&buf, l.fset, e.X)

	var bufY bytes.Buffer
	format.Node(&bufY, l.fset, e.Y)

	if (buf.String() == "err" || bufY.String() == "err") {
		l.positions = append(l.positions, e.Pos())
	}

	return true
}
