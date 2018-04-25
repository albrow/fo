package types

import (
	"testing"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
)

// TODO(albrow): test multiple packages here where one imports and uses generics
// from the other.

func parseTestSource(t *testing.T, src string) *Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "genericstest.go", src, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	var conf Config
	pkg, err := conf.Check("genericstest", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestGenericsUsageSingleCase(t *testing.T) {
	var src = `package genericstest

type A[T] T

func main() {
	var _ = A[string]("")
	var _ A[bool] = true
	var _ = A[bool](true)
}
`

	pkg := parseTestSource(t, src)
	if pkg.generics == nil {
		t.Fatal("pkg.generics was nil")
	}
	if len(pkg.generics) != 1 {
		t.Fatalf("wrong number of generic declarations (expected 1 but got %d)", len(pkg.generics))
	}
	aDecl, found := pkg.generics["A"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(aDecl.typeParams) != 1 {
		t.Errorf("wrong number of type parameters for A (expected 1 but got %d)", len(aDecl.typeParams))
	}
	if len(aDecl.usages) != 2 {
		t.Fatalf("wrong number of usages for A (expected 2 but got %d)", len(aDecl.usages))
	}
	expectedUsages := map[string]struct{}{
		"string": struct{}{},
		"bool":   struct{}{},
	}
	for _, usage := range aDecl.usages {
		mappedType := usage.typeMap["T"].Underlying().String()
		if _, found := expectedUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for usage: T -> %s", mappedType)
		}
	}
}

func TestGenericsUsageInherited(t *testing.T) {
	src := `package genericstest

type A[T, U] map[T]U

type APart[T] struct {
  val A[string, T]
} 

func main() {
	var _ = APart[int]{}
	var _ = APart[bool]{
		val: A[string, bool]{
			"": true,
		},
	}
}
`

	pkg := parseTestSource(t, src)
	if pkg.generics == nil {
		t.Fatal("pkg.generics was nil")
	}
	if len(pkg.generics) != 2 {
		t.Fatalf("wrong number of generic declarations (expected 2 but got %d)", len(pkg.generics))
	}

	aPartDecl, found := pkg.generics["APart"]
	if !found {
		t.Fatal("could not find generic declaration for APart")
	}
	if len(aPartDecl.typeParams) != 1 {
		t.Errorf("wrong number of type parameters for APart (expected 1 but got %d)", len(aPartDecl.typeParams))
	}
	if len(aPartDecl.usages) != 2 {
		t.Fatalf("wrong number of usages for APart (expected 2 but got %d)", len(aPartDecl.usages))
	}
	expectedAPartUsages := map[string]struct{}{
		"int":  struct{}{},
		"bool": struct{}{},
	}
	for _, usage := range aPartDecl.usages {
		mappedType := usage.typeMap["T"].Underlying().String()
		if _, found := expectedAPartUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for APart usage: T -> %s", mappedType)
		}
	}

	aDecl, found := pkg.generics["A"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(aDecl.typeParams) != 2 {
		t.Errorf("wrong number of type parameters for A (expected 2 but got %d)", len(aDecl.typeParams))
	}
	if len(aDecl.usages) != 2 {
		t.Fatalf("wrong number of usages for A (expected 2 but got %d)", len(aDecl.usages))
	}
	expectedAUsages := map[string]struct{}{
		"string,int":  struct{}{},
		"string,bool": struct{}{},
	}
	for _, usage := range aDecl.usages {
		mappedT := usage.typeMap["T"].Underlying().String()
		mappedU := usage.typeMap["U"].Underlying().String()
		mappedType := mappedT + "," + mappedU
		if _, found := expectedAUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for A usage: %s", mappedType)
		}
	}
}
