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

type Transformer struct {
	Fset *token.FileSet
	Pkg  *types.Package
	Info *types.Info
}

func (trans *Transformer) File(f *ast.File) (*ast.File, error) {
	withConcreteTypes := astutil.Apply(f, trans.generateConcreteTypes(), nil)
	result := astutil.Apply(withConcreteTypes, trans.replaceGenericIdents(), nil)
	resultFile, ok := result.(*ast.File)
	if !ok {
		panic(fmt.Errorf("astutil.Apply returned a non-file type: %T", result))
	}

	return resultFile, nil
}

var safeSymbolMap = map[string]string{
	".": "_",
	"[": "_",
	"]": "_",
	"*": "_",
}

func replaceUnsafeSymbols(s string) string {
	for unsafe, safe := range safeSymbolMap {
		s = strings.Replace(s, unsafe, safe, -1)
	}
	return s
}

func formatTypeArgs(args []ast.Expr) string {
	result := ""
	for i, arg := range args {
		buf := bytes.Buffer{}
		format.Node(&buf, token.NewFileSet(), arg)
		if i != 0 {
			result += "__"
		}
		result += replaceUnsafeSymbols(buf.String())
	}
	return result
}

// TODO: this could be optimized
func concreteTypeName(decl *types.GenericDecl, usg types.ConcreteType) string {
	stringParams := []string{}
	for _, param := range decl.Type.TypeParams() {
		typeString := usg.TypeMap()[param.String()].String()
		safeParam := replaceUnsafeSymbols(typeString)
		stringParams = append(stringParams, safeParam)
	}
	if len(stringParams) == 0 {
		return decl.Name
	}
	return decl.Name + "__" + strings.Join(stringParams, "__")
}

func concreteTypeExpr(e *ast.TypeArgExpr) ast.Node {
	switch x := e.X.(type) {
	case *ast.Ident:
		newIdent := astclone.Clone(x).(*ast.Ident)
		newIdent.Name = newIdent.Name + "__" + formatTypeArgs(e.Types)
		return newIdent
	case *ast.SelectorExpr:
		newSel := astclone.Clone(x).(*ast.SelectorExpr)
		newSel.Sel = ast.NewIdent(newSel.Sel.Name + "__" + formatTypeArgs(e.Types))
		return newSel
	default:
		panic(fmt.Errorf("type arguments for expr %v of type %T are not yet supported", e.X, e.X))
	}
}

func recvTypeParams(typeParams []*types.TypeParam, typeMap map[string]types.Type) []ast.Expr {
	types := []ast.Expr{}
	for _, param := range typeParams {
		typeString := typeMap[param.String()].String()
		types = append(types, ast.NewIdent(typeString))
	}
	if len(types) > 0 {
		return types
	} else {
		return nil
	}
}

// expandReceiverType adds the appropriate type parameters to a receiver type
// if they were not included in the original source code.
func expandReceiverType(funcDecl *ast.FuncDecl, genDecl *types.GenericDecl, usg types.ConcreteType) {
	astutil.Apply(funcDecl.Recv, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.TypeArgExpr:
			// Don't change anything about existing type arguments
			return false
		case *ast.Ident:
			if n.Name == genDecl.Name {
				c.Replace(&ast.TypeArgExpr{
					X:      ast.NewIdent(n.Name),
					Lbrack: token.NoPos,
					Types:  recvTypeParams(genDecl.Type.TypeParams(), usg.TypeMap()),
					Rbrack: token.NoPos,
				})
			}
		}
		return true
	}, nil)
}

func (trans *Transformer) generateConcreteTypes() func(c *astutil.Cursor) bool {
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
				if _, found := trans.Pkg.Generics()[typeSpec.Name.Name]; !found {
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
			newFuncs, recvIsGeneric := trans.generateFuncDecls(n)
			if len(newFuncs) == 0 {
				if recvIsGeneric || n.TypeParams != nil {
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

func (trans *Transformer) replaceGenericIdents() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.TypeArgExpr:
			c.Replace(concreteTypeExpr(n))
		case *ast.IndexExpr:
			// Check if we are dealing with an ambiguous IndexExpr from the parser. In
			// some cases we need to disambiguate this by upgrading to a
			// TypeArgExpr.
			switch x := n.X.(type) {
			case *ast.Ident:
				if _, found := trans.Pkg.Generics()[x.Name]; found {
					typeArgExpr := &ast.TypeArgExpr{
						X:      n.X,
						Lbrack: n.Lbrack,
						Types:  []ast.Expr{n.Index},
						Rbrack: n.Rbrack,
					}
					c.Replace(concreteTypeExpr(typeArgExpr))
				}
			case *ast.SelectorExpr:
				selection, found := trans.Info.Selections[x]
				if !found {
					return true
				}
				var key string
				switch selection.Kind() {
				case types.FieldVal:
					key = selection.Obj().Name()
				case types.MethodVal:
					if named, ok := selection.Recv().(*types.ConcreteNamed); ok {
						key = named.Obj().Name() + "." + selection.Obj().Name()
					}
				}
				if key != "" {
					if _, found := trans.Pkg.Generics()[key]; found {
						typeArgExpr := &ast.TypeArgExpr{
							X:      n.X,
							Lbrack: n.Lbrack,
							Types:  []ast.Expr{n.Index},
							Rbrack: n.Rbrack,
						}
						c.Replace(concreteTypeExpr(typeArgExpr))
						return false
					}
				}
			}
		}
		return true
	}
}

func (trans *Transformer) generateTypeSpecs(typeSpec *ast.TypeSpec) []ast.Spec {
	key := typeSpec.Name.Name
	genericDecl, found := trans.Pkg.Generics()[key]
	if !found {
		panic(fmt.Errorf("could not find generic type declaration for %s", key))
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
	for _, usg := range genericDecl.Usages {
		newTypeSpec := astclone.Clone(typeSpec).(*ast.TypeSpec)
		newTypeSpec.Name = ast.NewIdent(concreteTypeName(genericDecl, usg))
		newTypeSpec.TypeParams = nil
		replaceIdentsInScope(newTypeSpec, usg.TypeMap())
		results = append(results, newTypeSpec)
	}
	return results
}

func (trans *Transformer) generateFuncDecls(funcDecl *ast.FuncDecl) (newFuncs []*ast.FuncDecl, recvIsGeneric bool) {
	var recv ast.Expr
	recvHasTypeArgs := false
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) == 1 {
		recv = funcDecl.Recv.List[0].Type
	}
	if selectorExpr, ok := recv.(*ast.StarExpr); ok {
		recv = selectorExpr.X
	}
	if typeArgExpr, ok := recv.(*ast.TypeArgExpr); ok {
		recvHasTypeArgs = true
		recv = typeArgExpr.X
	}
	var genRecvDecl *types.GenericDecl
	var recvTypeName *ast.Ident
	if recv != nil {
		var ok bool
		recvTypeName, ok = recv.(*ast.Ident)
		if !ok {
			panic(fmt.Errorf("invalid receiver type expression: %T %s", recv, recv))
		}
		var found bool
		genRecvDecl, found = trans.Pkg.Generics()[recvTypeName.Name]
		if !found && recvHasTypeArgs {
			panic(fmt.Errorf("could not find generic type declaration for %s", recvTypeName.Name))
		} else {
			recvIsGeneric = true
		}
	}
	fkey := funcDecl.Name.Name
	if recvTypeName != nil {
		fkey = recvTypeName.Name + "." + fkey
	}
	genFuncDecl, found := trans.Pkg.Generics()[fkey]
	if !found && funcDecl.TypeParams != nil {
		panic(fmt.Errorf("could not find generic type declaration for %s", fkey))
	}
	if genFuncDecl != nil {
		for _, usg := range genFuncDecl.Usages {
			newFunc := astclone.Clone(funcDecl).(*ast.FuncDecl)
			expandReceiverType(newFunc, genRecvDecl, usg)
			newFunc.Name = ast.NewIdent(concreteTypeName(genFuncDecl, usg))
			newFunc.TypeParams = nil
			replaceIdentsInScope(newFunc, usg.TypeMap())
			newFuncs = append(newFuncs, newFunc)
		}
	} else if genRecvDecl != nil {
		for _, usg := range genRecvDecl.Usages {
			newFunc := astclone.Clone(funcDecl).(*ast.FuncDecl)
			expandReceiverType(newFunc, genRecvDecl, usg)
			replaceIdentsInScope(newFunc, usg.TypeMap())
			newFuncs = append(newFuncs, newFunc)
		}
	}
	return newFuncs, recvIsGeneric
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
