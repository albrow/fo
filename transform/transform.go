package transform

import (
	"fmt"
	"strings"

	"github.com/albrow/fo/ast"
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
	alreadySeen := stringset.New()
	var err error
	ast.Inspect(f, func(n ast.Node) bool {
		if genIdent, ok := n.(*ast.GenIdent); ok {
			if genIdent.GenParams != nil {
				params := parseGenParams(genIdent.GenParams)
				stringifiedParams := strings.Join(params, ",")
				if !alreadySeen.Contains(stringifiedParams) {
					usage[genIdent.Name] = append(usage[genIdent.Name], params)
					alreadySeen.Add(stringifiedParams)
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

func parseGenParams(genParams *ast.GenParamList) []string {
	params := []string{}
	for _, ident := range genParams.List {
		params = append(params, ident.Name)
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
						if t.GenParams != nil {
							newNodes := generateDeclNodes(n, typeSpec, usage[typeSpec.Name.Name])
							for _, node := range newNodes {
								c.InsertAfter(node)
							}
							c.Delete()
						}
					}
				}
			}
		case *ast.GenIdent:
			if n.GenParams != nil {
				params := parseGenParams(n.GenParams)
				newName := generateTypeName(n.Name, params)
				c.Replace(ast.NewIdent(newName))
			}
		}
		return true
	}
}

func generateDeclNodes(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, thisUsage [][]string) []ast.Node {
	newNodes := []ast.Node{}
	structTypeRef, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	for _, params := range thisUsage {
		newType := *typeSpec
		newType.Name = ast.NewIdent(generateTypeName(typeSpec.Name.Name, params))
		newStructType := *structTypeRef
		newFieldList := make([]*ast.Field, len(newStructType.Fields.List))
		copy(newFieldList, newStructType.Fields.List)
		for i, field := range newFieldList {
			if fieldTypeIdent, ok := field.Type.(*ast.Ident); ok {
				paramIndex := getTypeParamIndex(newStructType.GenParams, fieldTypeIdent.Name)
				if paramIndex == -1 {
					// This is a type not in the list of generic type parameters.
					continue
				}
				newField := *field
				newField.Type = ast.NewIdent(params[paramIndex])
				newFieldList[i] = &newField
			}
		}
		newFields := *newStructType.Fields
		newFields.List = newFieldList
		newStructType.Fields = &newFields
		newStructType.GenParams = nil
		newType.Type = &newStructType
		newDecl := *genDecl
		newDecl.Specs = []ast.Spec{&newType}
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
	return -1
}
