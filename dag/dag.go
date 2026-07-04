// Package dag implements the essential operations of a directed acyclic graph —
// the verbs the essay "The Phenomenology of the DAG" (madeofdoors.com) names:
// validate (and confess a cycle), follow (topological order), touch (the
// critical path), work with (width, by Dilworth), hold (the frontier),
// encircle (the cone), layer (Sugiyama ranks), and the transitive reduction
// (what may be left unsaid).
//
// The graph core is pure Go (standard library only); the MCP server in package
// main wraps these functions as tools.
package dag

import (
	"fmt"
	"sort"
)

// Graph is a directed graph over string-named nodes. It MAY contain cycles; the
// acyclic-only operations return an error (or a cycle witness) when it does.
type Graph struct {
	nodes []string            // insertion-ordered, de-duplicated
	set   map[string]bool     // membership
	out   map[string][]string // adjacency: node -> successors
	in    map[string][]string // reverse adjacency: node -> predecessors
}

// New builds a graph from nodes and directed edges [from, to]. Edges may
// reference endpoints not listed in nodes; those are added automatically.
func New(nodes []string, edges [][2]string) *Graph {
	g := &Graph{set: map[string]bool{}, out: map[string][]string{}, in: map[string][]string{}}
	add := func(n string) {
		if !g.set[n] {
			g.set[n] = true
			g.nodes = append(g.nodes, n)
		}
	}
	for _, n := range nodes {
		add(n)
	}
	for _, e := range edges {
		add(e[0])
		add(e[1])
		g.out[e[0]] = append(g.out[e[0]], e[1])
		g.in[e[1]] = append(g.in[e[1]], e[0])
	}
	return g
}

// Nodes returns the nodes in insertion order (a copy).
func (g *Graph) Nodes() []string { return append([]string(nil), g.nodes...) }

// FindCycle returns a directed cycle as a node path (v0 -> ... -> v0) if one
// exists, else nil. This is the confession: the witness that the graph is not,
// in fact, acyclic.
func (g *Graph) FindCycle() []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var dfs func(n string) []string
	dfs = func(n string) []string {
		color[n] = gray
		stack = append(stack, n)
		for _, m := range g.out[n] {
			switch color[m] {
			case gray:
				for i, s := range stack {
					if s == m {
						cyc := append([]string(nil), stack[i:]...)
						return append(cyc, m)
					}
				}
			case white:
				if c := dfs(m); c != nil {
					return c
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		return nil
	}
	for _, n := range g.nodes {
		if color[n] == white {
			if c := dfs(n); c != nil {
				return c
			}
		}
	}
	return nil
}

// IsDAG reports whether the graph is acyclic.
func (g *Graph) IsDAG() bool { return g.FindCycle() == nil }

// TopoSort returns one legal topological order (Kahn's algorithm), ties broken
// by name for determinism — one of many legal flattenings. If the graph is
// cyclic it returns the partial order it managed plus an error.
func (g *Graph) TopoSort() ([]string, error) {
	indeg := map[string]int{}
	for _, n := range g.nodes {
		indeg[n] = len(g.in[n])
	}
	var ready []string
	for _, n := range g.nodes {
		if indeg[n] == 0 {
			ready = append(ready, n)
		}
	}
	sort.Strings(ready)
	var order []string
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		order = append(order, n)
		for _, m := range g.out[n] {
			indeg[m]--
			if indeg[m] == 0 {
				ready = append(ready, m)
			}
		}
		sort.Strings(ready)
	}
	if len(order) < len(g.nodes) {
		return order, fmt.Errorf("circular dependency: the cycle was always there")
	}
	return order, nil
}

// Frontier returns the ready set given the completed nodes: every not-yet-done
// node all of whose predecessors are done. The moving waterline — what you can
// do now. Result is sorted.
func (g *Graph) Frontier(completed []string) []string {
	done := map[string]bool{}
	for _, c := range completed {
		done[c] = true
	}
	var ready []string
	for _, n := range g.nodes {
		if done[n] {
			continue
		}
		ok := true
		for _, p := range g.in[n] {
			if !done[p] {
				ok = false
				break
			}
		}
		if ok {
			ready = append(ready, n)
		}
	}
	sort.Strings(ready)
	return ready
}

// Descendants returns every node reachable from n (strict: excludes n), sorted.
func (g *Graph) Descendants(n string) []string {
	seen := map[string]bool{}
	var dfs func(x string)
	dfs = func(x string) {
		for _, m := range g.out[x] {
			if !seen[m] {
				seen[m] = true
				dfs(m)
			}
		}
	}
	dfs(n)
	return sortedKeys(seen)
}

// Ancestors returns every node that can reach n (strict: excludes n), sorted.
func (g *Graph) Ancestors(n string) []string {
	seen := map[string]bool{}
	var dfs func(x string)
	dfs = func(x string) {
		for _, p := range g.in[x] {
			if !seen[p] {
				seen[p] = true
				dfs(p)
			}
		}
	}
	dfs(n)
	return sortedKeys(seen)
}

// Cone returns n's ancestors and descendants — the only honest circle a DAG
// allows (relatedness is directional; siblings on a rank are not included).
func (g *Graph) Cone(n string) (ancestors, descendants []string) {
	return g.Ancestors(n), g.Descendants(n)
}

// Reachable reports whether a can reach b by following edges (strict; a != b).
func (g *Graph) Reachable(a, b string) bool {
	if a == b {
		return false
	}
	for _, d := range g.Descendants(a) {
		if d == b {
			return true
		}
	}
	return false
}

// LongestPath returns the critical path (longest chain) and its length. With
// nil weights each node counts 1, so length is the number of nodes on the path.
// The floor on wall-clock time no parallelism can shorten. Requires acyclic.
func (g *Graph) LongestPath(weights map[string]float64) ([]string, float64, error) {
	order, err := g.TopoSort()
	if err != nil {
		return nil, 0, err
	}
	w := func(n string) float64 {
		if weights != nil {
			if x, ok := weights[n]; ok {
				return x
			}
		}
		return 1
	}
	dist := map[string]float64{}
	next := map[string]string{}
	best := -1.0
	var bestNode string
	for i := len(order) - 1; i >= 0; i-- {
		n := order[i]
		dist[n] = w(n)
		for _, m := range g.out[n] {
			if w(n)+dist[m] > dist[n] {
				dist[n] = w(n) + dist[m]
				next[n] = m
			}
		}
		if dist[n] > best {
			best = dist[n]
			bestNode = n
		}
	}
	var path []string
	for cur := bestNode; cur != ""; {
		path = append(path, cur)
		nx, ok := next[cur]
		if !ok {
			break
		}
		cur = nx
	}
	return path, best, nil
}

// Layers assigns each node a rank equal to the longest chain of predecessors
// (sources are rank 0). The Sugiyama move that makes acyclicity visible.
// Requires acyclic.
func (g *Graph) Layers() (map[string]int, error) {
	order, err := g.TopoSort()
	if err != nil {
		return nil, err
	}
	rank := map[string]int{}
	for _, n := range order {
		r := 0
		for _, p := range g.in[n] {
			if rank[p]+1 > r {
				r = rank[p] + 1
			}
		}
		rank[n] = r
	}
	return rank, nil
}

// TransitiveReduction returns the unique minimal edge set with the same
// reachability — the Hasse diagram. The lines you may leave silent because
// transitivity already implies them. Requires acyclic. Result is sorted.
func (g *Graph) TransitiveReduction() ([][2]string, error) {
	if c := g.FindCycle(); c != nil {
		return nil, fmt.Errorf("not a DAG: cycle %v", c)
	}
	reach := map[string]map[string]bool{}
	for _, s := range g.nodes {
		m := map[string]bool{}
		for _, d := range g.Descendants(s) {
			m[d] = true
		}
		reach[s] = m
	}
	var edges [][2]string
	for _, u := range g.nodes {
		for _, v := range g.out[u] {
			redundant := false
			for _, w := range g.out[u] {
				if w != v && reach[w][v] {
					redundant = true
					break
				}
			}
			if !redundant {
				edges = append(edges, [2]string{u, v})
			}
		}
	}
	return dedupeEdges(edges), nil
}

// Width returns the size of the largest antichain — by Dilworth's theorem, the
// minimum number of chains that cover the order — together with one such
// minimum chain cover. Width is breath: how many independent threads the graph
// permits at once. Requires acyclic.
func (g *Graph) Width() (int, [][]string, error) {
	if c := g.FindCycle(); c != nil {
		return 0, nil, fmt.Errorf("not a DAG: cycle %v", c)
	}
	reach := map[string][]string{}
	for _, s := range g.nodes {
		reach[s] = g.Descendants(s) // strict, sorted — chains may skip
	}
	matchL := map[string]string{} // u -> its successor in a chain
	matchR := map[string]string{} // v -> its predecessor in a chain
	var aug func(u string, seen map[string]bool) bool
	aug = func(u string, seen map[string]bool) bool {
		for _, v := range reach[u] {
			if seen[v] {
				continue
			}
			seen[v] = true
			if r, ok := matchR[v]; !ok || aug(r, seen) {
				matchL[u] = v
				matchR[v] = u
				return true
			}
		}
		return false
	}
	matching := 0
	for _, u := range g.nodes {
		if aug(u, map[string]bool{}) {
			matching++
		}
	}
	width := len(g.nodes) - matching
	var chains [][]string
	for _, n := range g.nodes {
		if _, isSucc := matchR[n]; isSucc {
			continue // not a chain head
		}
		var chain []string
		for cur := n; ; {
			chain = append(chain, cur)
			nx, ok := matchL[cur]
			if !ok {
				break
			}
			cur = nx
		}
		chains = append(chains, chain)
	}
	return width, chains, nil
}

func sortedKeys(m map[string]bool) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

func dedupeEdges(es [][2]string) [][2]string {
	seen := map[[2]string]bool{}
	var out [][2]string
	for _, e := range es {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}
