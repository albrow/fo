package transform

import (
	"fmt"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/astclone"
	"github.com/albrow/fo/astutil"
	"github.com/albrow/fo/token"
	"github.com/albrow/stringset"
)

func File(fset *token.FileSet, f *ast.File) (*ast.File, error) {
	usage, err := findGenericTypeUsage(fset, f)
	if err != nil {
		return nil, err
	}
	result := astutil.Apply(f, nil, postTransform(usage))
	resultFile, ok := result.(*ast.File)
	if !ok {
		panic(fmt.Errorf("astutil.Apply returned a non-file type: %T", result))
	}

	return resultFile, nil
}

func findGenericTypeUsage(fset *token.FileSet, f *ast.File) (map[string][][]string, error) {
	usage := map[string][][]string{}
	alreadySeen := map[string]stringset.Set{}
	var err error
	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if ident.TypeParams != nil {
				params := parseConcreteTypeParams(ident.TypeParams)
				stringifiedParams := strings.Join(params, ",")
				if alreadySeen[ident.Name] == nil {
					alreadySeen[ident.Name] = stringset.New()
				}
				if !alreadySeen[ident.Name].Contains(stringifiedParams) {
					usage[ident.Name] = append(usage[ident.Name], params)
					alreadySeen[ident.Name].Add(stringifiedParams)
				}
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return usage, nil
}

func parseConcreteTypeParams(list *ast.ConcreteTypeParamList) []string {
	params := []string{}
	for _, expr := range list.List {
		switch x := expr.(type) {
		case *ast.Ident:
			params = append(params, x.Name)
		default:
			panic(fmt.Errorf("unexpected concrete type in type param list: %T", expr))
		}
	}
	return params
}

func postTransform(usage map[string][][]string) func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.GenDecl:
			if len(n.Specs) == 1 {
				if typeSpec, ok := n.Specs[0].(*ast.TypeSpec); ok {
					switch t := typeSpec.Type.(type) {
					case *ast.StructType:
						if t.TypeParams != nil {
							newNodes := createStructTypeNodes(n, usage[typeSpec.Name.Name])
							for _, node := range newNodes {
								c.InsertBefore(node)
							}
							c.Delete()
						}
					}
				}
			}
		case *ast.FuncDecl:
			if n.TypeParams != nil {
				newNodes := createFuncDeclNodes(n, usage[n.Name.Name])
				for _, node := range newNodes {
					c.InsertBefore(node)
				}
				c.Delete()
			}
		case *ast.Ident:
			if n.TypeParams != nil {
				params := parseConcreteTypeParams(n.TypeParams)
				newName := generateTypeName(n.Name, params)
				c.Replace(ast.NewIdent(newName))
			}
		}
		return true
	}
}

func createStructTypeNodes(genDecl *ast.GenDecl, thisUsage [][]string) []ast.Node {
	newNodes := []ast.Node{}
	typeSpec := genDecl.Specs[0].(*ast.TypeSpec)
	structType := typeSpec.Type.(*ast.StructType)
	for _, params := range thisUsage {
		mappings := createTypeMappings(structType.TypeParams, params)
		newDecl := replaceIdentsInScope(astclone.Clone(genDecl), mappings).(*ast.GenDecl)
		newTypeSpec := newDecl.Specs[0].(*ast.TypeSpec)
		newStructType := newTypeSpec.Type.(*ast.StructType)
		newStructType.TypeParams = nil
		newTypeSpec.Name = ast.NewIdent(generateTypeName(typeSpec.Name.Name, params))
		newNodes = append(newNodes, newDecl)
	}
	return newNodes
}

func createFuncDeclNodes(funcDecl *ast.FuncDecl, thisUsage [][]string) []ast.Node {
	newNodes := []ast.Node{}
	for _, params := range thisUsage {
		mappings := createTypeMappings(funcDecl.TypeParams, params)
		newDecl := replaceIdentsInScope(astclone.Clone(funcDecl), mappings).(*ast.FuncDecl)
		newDecl.Name = ast.NewIdent(generateTypeName(funcDecl.Name.Name, params))
		newNodes = append(newNodes, newDecl)
	}
	return newNodes
}

// TODO: handle nested scopes here.
func replaceIdentsInScope(n ast.Node, mappings map[string]string) ast.Node {
	return astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if ident, ok := c.Node().(*ast.Ident); ok {
			if newName, found := mappings[ident.Name]; found {
				c.Replace(ast.NewIdent(newName))
			}
		}
		return true
	})
}

func generateTypeName(typeName string, params []string) string {
	return fmt.Sprintf("%s__%s", typeName, strings.Join(params, "__"))
}

func getTypeParamIndex(typeParams *ast.TypeParamList, typeName string) int {
	for i, paramName := range typeParams.List {
		if paramName.Name == typeName {
			return i
		}
	}
	return -1
}

func createTypeMappings(typeParams *ast.TypeParamList, params []string) map[string]string {
	if len(params) != len(typeParams.List) {
		panic(
			fmt.Errorf(
				"%v: wrong number of type parameters (expected %d but got %d)",
				typeParams.Pos(),
				len(typeParams.List),
				len(params),
			),
		)
	}
	mappings := map[string]string{}
	for i, oldIdent := range typeParams.List {
		mappings[oldIdent.Name] = params[i]
	}
	return mappings
}
