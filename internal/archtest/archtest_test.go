package archtest_test

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const module = "github.com/williamokano/go-ddd-by-example"

// Bounded contexts: top-level trees under internal/ that hold a model.
var contexts = []string{"venue", "show", "ticketing", "notifications"}

func load(t *testing.T) []*packages.Package {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:  "../..",
	}
	pkgs, err := packages.Load(cfg, "./internal/...", "./cmd/...")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("packages failed to load")
	}
	return pkgs
}

func isStdlib(path string) bool { return !strings.Contains(strings.Split(path, "/")[0], ".") }

// contextOf returns the bounded context a package belongs to, or "".
func contextOf(path string) string {
	rest, ok := strings.CutPrefix(path, module+"/internal/")
	if !ok {
		return ""
	}
	ctx := strings.Split(rest, "/")[0]
	if slices.Contains(contexts, ctx) {
		return ctx
	}
	return ""
}

func layerIs(path, layer string) bool { return strings.Contains(path+"/", "/"+layer+"/") }

func TestDependencyRules(t *testing.T) {
	rules := []struct {
		name      string
		applies   func(pkg string) bool
		forbidden func(pkg, imp string) bool
	}{
		{
			"domain imports only stdlib, sharedkernel and uuid",
			func(p string) bool { return contextOf(p) != "" && strings.HasSuffix(p, "/domain") },
			func(_, imp string) bool {
				return !isStdlib(imp) && imp != module+"/internal/sharedkernel" && imp != "github.com/google/uuid"
			},
		},
		{
			"domain does not import context",
			func(p string) bool { return contextOf(p) != "" && strings.HasSuffix(p, "/domain") },
			func(_, imp string) bool { return imp == "context" },
		},
		{
			"application imports no adapters, platform or drivers",
			func(p string) bool { return contextOf(p) != "" && layerIs(p, "application") },
			func(_, imp string) bool {
				return layerIs(imp, "adapters") || strings.HasPrefix(imp, module+"/internal/platform") ||
					strings.HasPrefix(imp, "github.com/jackc/pgx") || strings.HasPrefix(imp, "github.com/twmb/franz-go")
			},
		},
		{
			"a context imports another context only through its contracts",
			func(p string) bool { return contextOf(p) != "" },
			func(p, imp string) bool {
				other := contextOf(imp)
				return other != "" && other != contextOf(p) && !strings.HasSuffix(imp, "/contracts")
			},
		},
		{
			"contracts import only the standard library",
			func(p string) bool { return contextOf(p) != "" && strings.HasSuffix(p, "/contracts") },
			func(_, imp string) bool { return !isStdlib(imp) },
		},
		{
			"the shared kernel imports only the standard library",
			func(p string) bool { return strings.HasPrefix(p, module+"/internal/sharedkernel") },
			func(_, imp string) bool { return !isStdlib(imp) },
		},
		{
			"platform imports no context",
			func(p string) bool { return strings.HasPrefix(p, module+"/internal/platform") },
			func(_, imp string) bool { return contextOf(imp) != "" },
		},
	}

	pkgs := load(t)
	for _, rule := range rules {
		t.Run(rule.name, func(t *testing.T) {
			for _, pkg := range pkgs {
				if !rule.applies(pkg.PkgPath) {
					continue
				}
				for imp := range pkg.Imports {
					if rule.forbidden(pkg.PkgPath, imp) {
						t.Errorf("%s imports %s", pkg.PkgPath, imp)
					}
				}
			}
		})
	}
}

// Reconstitution trusts its input, so only repositories may use it (1.9).
func TestOnlyDrivenAdaptersRehydrate(t *testing.T) {
	for _, pkg := range load(t) {
		if contextOf(pkg.PkgPath) != "" && (layerIs(pkg.PkgPath, "driven") || strings.HasSuffix(pkg.PkgPath, "/domain")) {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || !strings.HasPrefix(id.Name, "Rehydrate") {
					return true
				}
				if fn, ok := pkg.TypesInfo.Uses[id].(*types.Func); ok && strings.HasSuffix(fn.Pkg().Path(), "/domain") {
					t.Errorf("%s calls %s.%s: only adapters/driven may rehydrate", pkg.Fset.Position(id.Pos()), fn.Pkg().Name(), id.Name)
				}
				return true
			})
		}
	}
}
