package dag

import (
	"reflect"
	"testing"
)

// diamond: a -> b -> d, a -> c -> d. Width 2 (antichain {b,c}), depth 3.
func diamond() *Graph {
	return New(
		[]string{"a", "b", "c", "d"},
		[][2]string{{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"}},
	)
}

func TestFindCycle(t *testing.T) {
	if c := diamond().FindCycle(); c != nil {
		t.Fatalf("diamond should be acyclic, got cycle %v", c)
	}
	g := New(nil, [][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}})
	if c := g.FindCycle(); c == nil {
		t.Fatal("expected a cycle witness for a->b->c->a")
	}
}

func TestTopoSort(t *testing.T) {
	order, err := diamond().TopoSort()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 || order[0] != "a" || order[3] != "d" {
		t.Fatalf("bad topo order %v", order)
	}
	if _, err := New(nil, [][2]string{{"a", "b"}, {"b", "a"}}).TopoSort(); err == nil {
		t.Fatal("expected a circular-dependency error")
	}
}

func TestFrontier(t *testing.T) {
	g := diamond()
	if got := g.Frontier(nil); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("frontier({}) = %v, want [a]", got)
	}
	if got := g.Frontier([]string{"a"}); !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("frontier({a}) = %v, want [b c]", got)
	}
}

func TestLongestPath(t *testing.T) {
	path, n, err := diamond().LongestPath(nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("critical-path length = %v, want 3", n)
	}
	if path[0] != "a" || path[len(path)-1] != "d" {
		t.Fatalf("critical path = %v, want a..d", path)
	}
}

func TestWidth(t *testing.T) {
	w, chains, err := diamond().Width()
	if err != nil {
		t.Fatal(err)
	}
	if w != 2 {
		t.Fatalf("width = %d, want 2", w)
	}
	if len(chains) != 2 {
		t.Fatalf("min chain cover = %v (%d chains), want 2", chains, len(chains))
	}
}

func TestLayers(t *testing.T) {
	r, err := diamond().Layers()
	if err != nil {
		t.Fatal(err)
	}
	if r["a"] != 0 || r["b"] != 1 || r["c"] != 1 || r["d"] != 2 {
		t.Fatalf("ranks = %v, want a:0 b:1 c:1 d:2", r)
	}
}

func TestTransitiveReduction(t *testing.T) {
	// diamond plus the redundant long edge a->d.
	g := New(
		[]string{"a", "b", "c", "d"},
		[][2]string{{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"}, {"a", "d"}},
	)
	edges, err := g.TransitiveReduction()
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 4 {
		t.Fatalf("reduction = %v (%d edges), want 4", edges, len(edges))
	}
	for _, e := range edges {
		if e[0] == "a" && e[1] == "d" {
			t.Fatal("a->d is implied by transitivity and should be removed")
		}
	}
}

func TestConeAndReach(t *testing.T) {
	g := diamond()
	if !g.Reachable("a", "d") {
		t.Fatal("a should reach d")
	}
	if g.Reachable("b", "c") {
		t.Fatal("b and c are an antichain; b must not reach c")
	}
	anc, desc := g.Cone("d")
	if !reflect.DeepEqual(anc, []string{"a", "b", "c"}) {
		t.Fatalf("ancestors(d) = %v, want [a b c]", anc)
	}
	if len(desc) != 0 {
		t.Fatalf("descendants(d) = %v, want none (d is a sink)", desc)
	}
}
