package luau_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
)

// BenchmarkRenderAST renders every random golden program (solveTempIds plus
// render). Run with: go test -bench RenderAST -benchmem ./luau
func BenchmarkRenderAST(b *testing.B) {
	var f goldenFile
	readGolden(b, "random.json", &f)
	var lists []*luau.List
	bytes := 0
	for i := range f.Cases {
		c := &f.Cases[i]
		if c.Mode != "renderAST" || c.Error != nil {
			continue
		}
		l := loadGraph(b, &c.Graph)
		lists = append(lists, l.root(c.Graph.Roots["ast"]).(*luau.List))
		bytes += len(*c.Expected)
	}
	b.SetBytes(int64(bytes))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, l := range lists {
			_ = luau.RenderAST(l)
		}
	}
}
