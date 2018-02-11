package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/astutil"
	"github.com/albrow/fo/token"
)

func File(fset *token.FileSet, f *ast.File) (*ast.File, error) {
	decls := findGenericTypeDecls(f)
	usage, err := findGenericTypeUsage(fset, f, decls)
	if err != nil {
		return nil, err
	}

	result := astutil.Apply(f, func(*astutil.Cursor) bool { return true }, postTransform(decls, usage))
	resultFile, ok := result.(*ast.File)
	if !ok {
		panic(fmt.Errorf("astutil.Apply returned a non-file type: %T", result))
	}

	return resultFile, nil
}

func findGenericTypeDecls(f *ast.File) map[string]*ast.StructType {
	decls := map[string]*ast.StructType{}
	ast.Inspect(f, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				if structType.GenParams != nil {
					decls[typeSpec.Name.String()] = structType
				}
			}
		}
		return true
	})
	return decls
}

func findGenericTypeUsage(fset *token.FileSet, f *ast.File, decls map[string]*ast.StructType) (map[string][][]string, error) {
	usage := map[string][][]string{}
	var err error
	ast.Inspect(f, func(n ast.Node) bool {
		if literal, ok := n.(*ast.CompositeLit); ok {
			if typeIdent, ok := literal.Type.(*ast.Ident); ok {
				if structType, found := decls[typeIdent.String()]; found {
					params := parseGenParams(literal)
					if len(params) != len(structType.GenParams.List) {
						err = fmt.Errorf(
							"%s: wrong number of type parameters: expected %d but got %d",
							fset.Position(literal.GenParams.Pos()),
							len(structType.GenParams.List),
							len(params),
						)
						return false
					}
					usage[typeIdent.Name] = append(usage[typeIdent.Name], params)
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

func parseGenParams(literal *ast.CompositeLit) []string {
	params := []string{}
	for _, ident := range literal.GenParams.List {
		params = append(params, ident.Name)
	}
	return params
}

func postTransform(decls map[string]*ast.StructType, usage map[string][][]string) func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.GenDecl:
			if len(n.Specs) == 1 {
				if typeSpec, ok := n.Specs[0].(*ast.TypeSpec); ok {
					if _, found := decls[typeSpec.Name.Name]; found {
						newNodes := generateDeclNodes(n, typeSpec, usage[typeSpec.Name.Name])
						for _, node := range newNodes {
							c.InsertAfter(node)
						}
						c.Delete()
					}
				}
			}
		case *ast.CompositeLit:
			if n.GenParams != nil {
				params := parseGenParams(n)
				if typeIdent, ok := n.Type.(*ast.Ident); ok {
					newName := generateTypeName(typeIdent.Name, params)
					newLit := *n
					newLit.Type = ast.NewIdent(newName)
					newLit.GenParams = nil
					c.Replace(&newLit)
				}
			}
		}
		return true
	}
}

// TODO: make this more readable by renaming all the copies to newX
func generateDeclNodes(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, thisUsage [][]string) []ast.Node {
	newNodes := []ast.Node{}
	for _, params := range thisUsage {
		ts := *typeSpec
		ts.Name = ast.NewIdent(generateTypeName(typeSpec.Name.Name, params))
		structTypeRef, ok := ts.Type.(*ast.StructType)
		if !ok {
			return nil
		}
		structType := *structTypeRef
		list := make([]*ast.Field, len(structType.Fields.List))
		copy(list, structType.Fields.List)
		for i, field := range list {
			if fieldTypeIdent, ok := field.Type.(*ast.Ident); ok {
				paramIndex := getTypeParamIndex(structType.GenParams, fieldTypeIdent.Name)
				newField := *field
				newField.Type = ast.NewIdent(params[paramIndex])
				list[i] = &newField
			}
		}
		fields := *structType.Fields
		fields.List = list
		structType.Fields = &fields
		structType.GenParams = nil
		ts.Type = &structType
		newDecl := *genDecl
		newDecl.Specs = []ast.Spec{&ts}
		newNodes = append(newNodes, &newDecl)
	}
	return newNodes
}

func generateTypeName(typeName string, params []string) string {
	return fmt.Sprintf("%s__%s", typeName, strings.Join(params, "__"))
}

func getTypeParamIndex(genParams *ast.GenParamList, typeName string) int {
	for i, paramName := range genParams.List {
		if paramName.Name == typeName {
			return i
		}
	}
	panic(fmt.Errorf("could not find type parameter %s in %s", typeName, genParams.List))
}

func deepCopy(dst interface{}, src interface{}) error {
	srcBytes := bytes.NewBuffer(nil)
	if err := json.NewEncoder(srcBytes).Encode(src); err != nil {
		return err
	}
	return json.NewDecoder(srcBytes).Decode(dst)
}
