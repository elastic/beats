// +build !go1.9

package parse

import (
	"go/build"
	"go/token"
	"go/types"

	"github.com/magefile/mage/parse/srcimporter"
)

func getImporter(fset *token.FileSet) types.Importer {
	return srcimporter.New(&build.Default, fset, make(map[string]*types.Package))
}
