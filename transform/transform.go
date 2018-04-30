package transform

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/astclone"
	"github.com/albrow/fo/astutil"
	"github.com/albrow/fo/format"
	"github.com/albrow/fo/printer"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/types"
)

// TODO(albrow): Implement transform.Package for operating on all files in a
// given package at once.

type transformer struct {
	fset    *token.FileSet
	pkg     *types.Package
	methods map[string]*ast.FuncDecl
}

func File(fset *token.FileSet, f *ast.File, pkg *types.Package) (*ast.File, error) {
	trans := &transformer{
		fset: fset,
		pkg:  pkg,
	}
	withConcreteTypes := astutil.Apply(f, trans.generateConcreteTypes(), nil)
	result := astutil.Apply(withConcreteTypes, trans.replaceGenericIdents(), nil)
	resultFile, ok := result.(*ast.File)
	if !ok {
		panic(fmt.Errorf("astutil.Apply returned a non-file type: %T", result))
	}

	return resultFile, nil
}

// TODO: this could be optimized
func concreteTypeName(decl *types.GenericDecl, usg *types.GenericUsage) string {
	stringParams := []string{}
	for _, param := range decl.TypeParams() {
		typeString := usg.TypeMap()[param.String()].String()
		safeParam := strings.Replace(typeString, ".", "_", -1)
		stringParams = append(stringParams, safeParam)
	}
	return decl.Name() + "__" + strings.Join(stringParams, "__")
}

func concreteTypeExpr(e *ast.TypeParamExpr) ast.Node {
	switch x := e.X.(type) {
	case *ast.Ident:
		newIdent := astclone.Clone(x).(*ast.Ident)
		stringParams := []string{}
		for _, param := range e.Params {
			buf := bytes.Buffer{}
			format.Node(&buf, token.NewFileSet(), param)
			typeString := buf.String()
			safeParam := strings.Replace(typeString, ".", "_", -1)
			stringParams = append(stringParams, safeParam)
		}
		newIdent.Name = newIdent.Name + "__" + strings.Join(stringParams, "__")
		return newIdent
	case *ast.SelectorExpr:
		newSel := astclone.Clone(x).(*ast.SelectorExpr)
		stringParams := []string{}
		for _, param := range e.Params {
			buf := bytes.Buffer{}
			format.Node(&buf, token.NewFileSet(), param)
			typeString := buf.String()
			safeParam := strings.Replace(typeString, ".", "_", -1)
			stringParams = append(stringParams, safeParam)
		}
		newSel.Sel = ast.NewIdent(newSel.Sel.Name + "__" + strings.Join(stringParams, "__"))
		return newSel
	default:
		panic(fmt.Errorf("type parameters for expr %s of type %T are not yet supported", e.X, e.X))
	}
}

func (trans *transformer) generateMethods(n *ast.FuncDecl) []*ast.FuncDecl {
	var results []*ast.FuncDecl
	if n.Recv == nil {
		return nil
	}
	if n.TypeParams != nil {
		// funcs with type parameters will be handled later on.
		return nil
	}
	if len(n.Recv.List) != 1 {
		return nil
	}
	recv := n.Recv.List[0].Type
	hasTypeParams := false
	if selectorExpr, ok := recv.(*ast.StarExpr); ok {
		recv = selectorExpr.X
	}
	if typeParamExpr, ok := recv.(*ast.TypeParamExpr); ok {
		hasTypeParams = true
		recv = typeParamExpr.X
	}
	genTypeName, ok := recv.(*ast.Ident)
	if !ok {
		// TODO(albrow): handle *ast.SelectorExpr here so we can support generic
		// types from other packages.
		return nil
	}
	genDecl, found := trans.pkg.Generics()[genTypeName.Name]
	if !found {
		if hasTypeParams {
			panic(fmt.Errorf("could not find generic type declaration for %s", genTypeName.Name))
		}
		return nil
	}
	for _, usg := range genDecl.Usages() {
		newFunc := astclone.Clone(n).(*ast.FuncDecl)
		expandReceiverType(newFunc, genDecl, usg)
		replaceIdentsInScope(newFunc, usg.TypeMap())
		results = append(results, newFunc)
	}
	return results
}

// expandReceiverType adds the appropriate type parameters to a receiver type
// if they were not included in the original source code.
func expandReceiverType(funcDecl *ast.FuncDecl, genDecl *types.GenericDecl, usg *types.GenericUsage) {
	astutil.Apply(funcDecl.Recv, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.TypeParamExpr:
			// Don't convert an existing TypeParamExpr
			return false
		case *ast.Ident:
			if n.Name == genDecl.Name() {
				c.Replace(&ast.TypeParamExpr{
					X:      ast.NewIdent(n.Name),
					Lbrack: token.NoPos,
					Params: usg.TypeParams(),
					Rbrack: token.NoPos,
				})
			}
		}
		return true
	}, nil)
}

func (trans *transformer) generateConcreteTypes() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.GenDecl:
			var newTypeSpecs []ast.Spec
			used := false
			for _, spec := range n.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					newTypeSpecs = append(newTypeSpecs, spec)
					used = true
					continue
				}
				if _, found := trans.pkg.Generics()[typeSpec.Name.Name]; !found {
					newTypeSpecs = append(newTypeSpecs, typeSpec)
					used = true
					continue
				}
				newTypeSpecs = append(newTypeSpecs, trans.generateTypeSpecs(typeSpec)...)
			}
			if len(newTypeSpecs) > 0 {
				sort.Slice(newTypeSpecs, func(i int, j int) bool {
					iSpec, ok := newTypeSpecs[i].(*ast.TypeSpec)
					if !ok {
						return true
					}
					jSpec, ok := newTypeSpecs[j].(*ast.TypeSpec)
					if !ok {
						return true
					}
					return iSpec.Name.Name < jSpec.Name.Name
				})
				newDecl := astclone.Clone(n).(*ast.GenDecl)
				newDecl.Specs = newTypeSpecs
				c.Replace(newDecl)
			} else if !used {
				c.Delete()
			}
		case *ast.FuncDecl:
			newFuncs := trans.generateMethods(n)
			if n.TypeParams != nil {
				newFuncs = append(newFuncs, trans.generateFuncDecls(n)...)
			}
			if len(newFuncs) == 0 {
				if n.TypeParams != nil {
					c.Delete()
				}
				return true
			}
			sortFuncs(newFuncs)
			for _, newFunc := range newFuncs {
				c.InsertBefore(newFunc)
			}
			c.Delete()
		}
		return true
	}
}

func sortFuncs(funcs []*ast.FuncDecl) {
	sort.Slice(funcs, func(i int, j int) bool {
		if funcs[i].Name.Name == funcs[j].Name.Name {
			// If the two function names are the same, they must have different
			// receivers. There's lots of edge cases to consider, so as a shortcut
			// we use printer.Fprint to convert each FuncDecl to a string.
			// TODO(albrow): optimize this
			iBuff, jBuff := &bytes.Buffer{}, &bytes.Buffer{}
			_ = printer.Fprint(iBuff, token.NewFileSet(), funcs[i])
			_ = printer.Fprint(jBuff, token.NewFileSet(), funcs[j])
			return iBuff.String() < jBuff.String()
		}
		return funcs[i].Name.Name < funcs[j].Name.Name
	})
}

func (trans *transformer) replaceGenericIdents() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.TypeParamExpr:
			c.Replace(concreteTypeExpr(n))
		case *ast.IndexExpr:
			// Check if we are dealing with an ambiguous IndexExpr from the parser. In
			// some cases we need to disambiguate this by upgrading to a
			// TypeParamExpr.
			switch x := n.X.(type) {
			case *ast.Ident:
				if _, found := trans.pkg.Generics()[x.Name]; found {
					typeParamExpr := &ast.TypeParamExpr{
						X:      n.X,
						Lbrack: n.Lbrack,
						Params: []ast.Expr{n.Index},
						Rbrack: n.Rbrack,
					}
					c.Replace(concreteTypeExpr(typeParamExpr))
				}
			}
		}
		return true
	}
}

func (trans *transformer) generateTypeSpecs(typeSpec *ast.TypeSpec) []ast.Spec {
	name := typeSpec.Name.Name
	genericDecl, found := trans.pkg.Generics()[name]
	if !found {
		panic(fmt.Errorf("could not find generic type declaration for %s", name))
	}
	var results []ast.Spec
	// Check if we are dealing with an ambiguous ArrayType from the parser. In
	// some cases we need to disambiguate this by adding type parameters and
	// changing the type.
	if typeSpec.TypeParams == nil {
		if arrayType, ok := typeSpec.Type.(*ast.ArrayType); ok {
			if length, ok := arrayType.Len.(*ast.Ident); ok {
				typeSpec = astclone.Clone(typeSpec).(*ast.TypeSpec)
				typeSpec.TypeParams = &ast.TypeParamDecl{
					Lbrack: arrayType.Lbrack,
					Names:  []*ast.Ident{},
					Rbrack: arrayType.Lbrack + token.Pos(len(length.Name)),
				}
				typeSpec.Type = arrayType.Elt
			}
		}
	}
	for _, usg := range genericDecl.Usages() {
		newTypeSpec := astclone.Clone(typeSpec).(*ast.TypeSpec)
		newTypeSpec.Name = ast.NewIdent(concreteTypeName(genericDecl, usg))
		newTypeSpec.TypeParams = nil
		replaceIdentsInScope(newTypeSpec, usg.TypeMap())
		results = append(results, newTypeSpec)
	}
	return results
}

func (trans *transformer) generateFuncDecls(funcDecl *ast.FuncDecl) []*ast.FuncDecl {
	name := funcDecl.Name.Name
	genericDecl, found := trans.pkg.Generics()[name]
	if !found {
		panic(fmt.Errorf("could not find generic type declaration for %s", name))
	}
	var results []*ast.FuncDecl
	for _, usg := range genericDecl.Usages() {
		newFuncDecl := astclone.Clone(funcDecl).(*ast.FuncDecl)
		newFuncDecl.Name = ast.NewIdent(concreteTypeName(genericDecl, usg))
		newFuncDecl.TypeParams = nil
		replaceIdentsInScope(newFuncDecl, usg.TypeMap())
		results = append(results, newFuncDecl)
	}
	return results
}

func replaceIdentsInScope(n ast.Node, typeMap map[string]types.Type) ast.Node {
	return astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if ident, ok := c.Node().(*ast.Ident); ok {
			if typ, found := typeMap[ident.Name]; found {
				c.Replace(ast.NewIdent(typ.String()))
			}
		}
		return true
	})
}
