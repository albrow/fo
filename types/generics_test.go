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
	if len(aDecl.Type.TypeParams()) != 1 {
		t.Errorf("wrong number of type arguments for A (expected 1 but got %d)", len(aDecl.Type.TypeParams()))
	}
	if len(aDecl.Usages) != 2 {
		t.Fatalf("wrong number of usages for A (expected 2 but got %d)", len(aDecl.Usages))
	}
	expectedUsages := map[string]struct{}{
		"string": struct{}{},
		"bool":   struct{}{},
	}
	for _, usage := range aDecl.Usages {
		mappedType := usage.TypeMap()["T"].Underlying().String()
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
	if len(aPartDecl.Type.TypeParams()) != 1 {
		t.Errorf("wrong number of type arguments for APart (expected 1 but got %d)", len(aPartDecl.Type.TypeParams()))
	}
	if len(aPartDecl.Usages) != 2 {
		t.Fatalf("wrong number of usages for APart (expected 2 but got %d)", len(aPartDecl.Usages))
	}
	expectedAPartUsages := map[string]struct{}{
		"int":  struct{}{},
		"bool": struct{}{},
	}
	for _, usage := range aPartDecl.Usages {
		mappedType := usage.TypeMap()["T"].Underlying().String()
		if _, found := expectedAPartUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for APart usage: T -> %s", mappedType)
		}
	}

	aDecl, found := pkg.generics["A"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(aDecl.Type.TypeParams()) != 2 {
		t.Errorf("wrong number of type arguments for A (expected 2 but got %d)", len(aDecl.Type.TypeParams()))
	}
	if len(aDecl.Usages) != 2 {
		t.Fatalf("wrong number of usages for A (expected 2 but got %d)", len(aDecl.Usages))
	}
	expectedAUsages := map[string]struct{}{
		"string,int":  struct{}{},
		"string,bool": struct{}{},
	}
	for _, usage := range aDecl.Usages {
		mappedT := usage.TypeMap()["T"].Underlying().String()
		mappedU := usage.TypeMap()["U"].Underlying().String()
		mappedType := mappedT + "," + mappedU
		if _, found := expectedAUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for A usage: %s", mappedType)
		}
	}
}

// TODO(albrow): Make this test pass.
func TestGenericsUsageInheritedInBody(t *testing.T) {
	src := `package genericstest

type A[T] T

func NewA[T]() {
	var _ A[T]
	F[T]()
}

func F[T]() T {
	var x T
	return x
}

func main() {
	NewA[string]()
}
`

	pkg := parseTestSource(t, src)
	if pkg.generics == nil {
		t.Fatal("pkg.generics was nil")
	}
	if len(pkg.generics) != 3 {
		t.Fatalf("wrong number of generic declarations (expected 3 but got %d)", len(pkg.generics))
	}
	aDecl, found := pkg.generics["A"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(aDecl.Type.TypeParams()) != 1 {
		t.Errorf("wrong number of type arguments for A (expected 1 but got %d)", len(aDecl.Type.TypeParams()))
	}
	if len(aDecl.Usages) != 1 {
		t.Fatalf("wrong number of usages for A (expected 1 but got %d)", len(aDecl.Usages))
	}
	expectedAUsages := map[string]struct{}{
		"string": struct{}{},
	}
	for _, usage := range aDecl.Usages {
		mappedType := usage.TypeMap()["T"].Underlying().String()
		if _, found := expectedAUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for A usage: T -> %s", mappedType)
		}
	}

	newADecl, found := pkg.generics["NewA"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(newADecl.Type.TypeParams()) != 1 {
		t.Errorf("wrong number of type arguments for NewA (expected 1 but got %d)", len(newADecl.Type.TypeParams()))
	}
	if len(newADecl.Usages) != 1 {
		t.Fatalf("wrong number of usages for NewA (expected 1 but got %d)", len(newADecl.Usages))
	}
	expectedNewAUsages := map[string]struct{}{
		"string": struct{}{},
	}
	for _, usage := range newADecl.Usages {
		mappedType := usage.TypeMap()["T"].Underlying().String()
		if _, found := expectedNewAUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for A usage: T -> %s", mappedType)
		}
	}

	fDecl, found := pkg.generics["F"]
	if !found {
		t.Fatal("could not find generic declaration for A")
	}
	if len(fDecl.Type.TypeParams()) != 1 {
		t.Errorf("wrong number of type arguments for F (expected 1 but got %d)", len(fDecl.Type.TypeParams()))
	}
	if len(fDecl.Usages) != 1 {
		t.Fatalf("wrong number of usages for F (expected 1 but got %d)", len(fDecl.Usages))
	}
	expectedFUsages := map[string]struct{}{
		"string": struct{}{},
	}
	for _, usage := range fDecl.Usages {
		mappedType := usage.TypeMap()["T"].Underlying().String()
		if _, found := expectedFUsages[mappedType]; !found {
			t.Errorf("unexpected typeMap for A usage: T -> %s", mappedType)
		}
	}
}
