// Command dag-essentials-mcp is an MCP server exposing the essentials of a
// directed acyclic graph as tools — the verbs of "The Phenomenology of the DAG"
// (madeofdoors.com). It is a thin, stateless shell over the tested `dag`
// package: each tool unmarshals a graph, calls one function, returns a typed
// result. Transport is stdio. See README.md.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cruzr/dag-essentials-mcp/dag"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// graphInput is the shared shape: a list of nodes and directed [from,to] edges.
// Endpoints named only in edges are added automatically.
type graphInput struct {
	Nodes []string   `json:"nodes,omitempty" jsonschema:"node names (optional if every node also appears in an edge)"`
	Edges [][]string `json:"edges,omitempty" jsonschema:"directed edges, each a two-element [from, to] pair"`
}

func build(nodes []string, edges [][]string) (*dag.Graph, error) {
	es := make([][2]string, 0, len(edges))
	for i, e := range edges {
		if len(e) != 2 {
			return nil, fmt.Errorf("edge %d must be a [from, to] pair, got %d element(s)", i, len(e))
		}
		es = append(es, [2]string{e[0], e[1]})
	}
	return dag.New(nodes, es), nil
}

func pairsToLists(ps [][2]string) [][]string {
	out := make([][]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, []string{p[0], p[1]})
	}
	return out
}

// ---- validate: encounter / the cut ----

type validateOut struct {
	IsDAG bool     `json:"is_dag"`
	Cycle []string `json:"cycle,omitempty" jsonschema:"a witness cycle v0 -> ... -> v0 when the graph is not acyclic"`
}

func validate(ctx context.Context, _ *mcp.CallToolRequest, in graphInput) (*mcp.CallToolResult, validateOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, validateOut{}, err
	}
	cyc := g.FindCycle()
	return nil, validateOut{IsDAG: cyc == nil, Cycle: cyc}, nil
}

// ---- toposort: follow ----

type toposortOut struct {
	Order   []string `json:"order"`
	Acyclic bool     `json:"acyclic"`
	Note    string   `json:"note" jsonschema:"context — e.g. that this is one of many legal orders, or why none exists"`
}

func toposort(ctx context.Context, _ *mcp.CallToolRequest, in graphInput) (*mcp.CallToolResult, toposortOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, toposortOut{}, err
	}
	order, terr := g.TopoSort()
	if terr != nil {
		return nil, toposortOut{Order: order, Acyclic: false, Note: terr.Error()}, nil
	}
	return nil, toposortOut{Order: order, Acyclic: true, Note: "one legal flattening of possibly many"}, nil
}

// ---- critical_path: touch ----

type criticalPathInput struct {
	Nodes   []string           `json:"nodes,omitempty"`
	Edges   [][]string         `json:"edges,omitempty" jsonschema:"directed edges, each a two-element [from, to] pair"`
	Weights map[string]float64 `json:"weights,omitempty" jsonschema:"optional per-node weights; each node defaults to 1"`
}

type criticalPathOut struct {
	Path   []string `json:"path"`
	Length float64  `json:"length" jsonschema:"total weight of the longest chain — the floor no parallelism shortens"`
}

func criticalPath(ctx context.Context, _ *mcp.CallToolRequest, in criticalPathInput) (*mcp.CallToolResult, criticalPathOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, criticalPathOut{}, err
	}
	path, length, perr := g.LongestPath(in.Weights)
	if perr != nil {
		return nil, criticalPathOut{}, perr
	}
	return nil, criticalPathOut{Path: path, Length: length}, nil
}

// ---- width: work with ----

type widthOut struct {
	Width  int        `json:"width" jsonschema:"size of the largest antichain (Dilworth's theorem)"`
	Chains [][]string `json:"chains" jsonschema:"a minimum chain cover; the number of chains equals the width"`
}

func width(ctx context.Context, _ *mcp.CallToolRequest, in graphInput) (*mcp.CallToolResult, widthOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, widthOut{}, err
	}
	w, chains, werr := g.Width()
	if werr != nil {
		return nil, widthOut{}, werr
	}
	return nil, widthOut{Width: w, Chains: chains}, nil
}

// ---- frontier: hold / orchestrate ----

type frontierInput struct {
	Nodes     []string   `json:"nodes,omitempty"`
	Edges     [][]string `json:"edges,omitempty" jsonschema:"directed edges, each a two-element [from, to] pair"`
	Completed []string   `json:"completed,omitempty" jsonschema:"nodes already finished (default: none)"`
}

type frontierOut struct {
	Ready []string `json:"ready" jsonschema:"the ready set: not-yet-done nodes whose predecessors are all done"`
}

func frontier(ctx context.Context, _ *mcp.CallToolRequest, in frontierInput) (*mcp.CallToolResult, frontierOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, frontierOut{}, err
	}
	return nil, frontierOut{Ready: g.Frontier(in.Completed)}, nil
}

// ---- cone: encircle ----

type coneInput struct {
	Nodes []string   `json:"nodes,omitempty"`
	Edges [][]string `json:"edges,omitempty" jsonschema:"directed edges, each a two-element [from, to] pair"`
	Node  string     `json:"node" jsonschema:"the node to encircle (required)"`
}

type coneOut struct {
	Ancestors   []string `json:"ancestors"`
	Descendants []string `json:"descendants"`
}

func cone(ctx context.Context, _ *mcp.CallToolRequest, in coneInput) (*mcp.CallToolResult, coneOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, coneOut{}, err
	}
	anc, desc := g.Cone(in.Node)
	return nil, coneOut{Ancestors: anc, Descendants: desc}, nil
}

// ---- reduce: layer (transitive reduction) ----

type reduceOut struct {
	Edges [][]string `json:"edges" jsonschema:"the unique minimal edge set with the same reachability (the Hasse diagram)"`
}

func reduce(ctx context.Context, _ *mcp.CallToolRequest, in graphInput) (*mcp.CallToolResult, reduceOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, reduceOut{}, err
	}
	edges, rerr := g.TransitiveReduction()
	if rerr != nil {
		return nil, reduceOut{}, rerr
	}
	return nil, reduceOut{Edges: pairsToLists(edges)}, nil
}

// ---- reach: traverse ----

type reachInput struct {
	Nodes []string   `json:"nodes,omitempty"`
	Edges [][]string `json:"edges,omitempty" jsonschema:"directed edges, each a two-element [from, to] pair"`
	From  string     `json:"from" jsonschema:"source node (required)"`
	To    string     `json:"to" jsonschema:"target node (required)"`
}

type reachOut struct {
	Reachable bool `json:"reachable"`
}

func reach(ctx context.Context, _ *mcp.CallToolRequest, in reachInput) (*mcp.CallToolResult, reachOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, reachOut{}, err
	}
	return nil, reachOut{Reachable: g.Reachable(in.From, in.To)}, nil
}

// ---- layers: layer / the look ----

type layersOut struct {
	Ranks map[string]int `json:"ranks" jsonschema:"each node's longest-path rank; sources are 0"`
}

func layers(ctx context.Context, _ *mcp.CallToolRequest, in graphInput) (*mcp.CallToolResult, layersOut, error) {
	g, err := build(in.Nodes, in.Edges)
	if err != nil {
		return nil, layersOut{}, err
	}
	ranks, lerr := g.Layers()
	if lerr != nil {
		return nil, layersOut{}, lerr
	}
	return nil, layersOut{Ranks: ranks}, nil
}

func main() {
	s := mcp.NewServer(&mcp.Implementation{Name: "dag-essentials", Version: "0.1.0"}, nil)

	// dag-verbs:begin — generated from dag-verbs.json by `go run ./tools/gen-verbs.go`; do not edit by hand
	mcp.AddTool(s, &mcp.Tool{Name: "validate", Description: "Is the graph acyclic? If not, return a witness cycle (v0 -> ... -> v0)."}, validate)
	mcp.AddTool(s, &mcp.Tool{Name: "toposort", Description: "One legal topological order (Kahn's algorithm). Reports whether the graph is acyclic."}, toposort)
	mcp.AddTool(s, &mcp.Tool{Name: "critical_path", Description: "The longest (optionally weighted) chain — the floor on completion time no parallelism shortens."}, criticalPath)
	mcp.AddTool(s, &mcp.Tool{Name: "width", Description: "The largest antichain (Dilworth's theorem) plus a minimum chain cover."}, width)
	mcp.AddTool(s, &mcp.Tool{Name: "frontier", Description: "Given completed nodes, the ready set (nodes whose predecessors are all done)."}, frontier)
	mcp.AddTool(s, &mcp.Tool{Name: "cone", Description: "A node's ancestors and descendants — the only honest circle a DAG allows."}, cone)
	mcp.AddTool(s, &mcp.Tool{Name: "reduce", Description: "The transitive reduction: the unique minimal edge set with the same reachability (the Hasse diagram)."}, reduce)
	mcp.AddTool(s, &mcp.Tool{Name: "reach", Description: "Can 'from' reach 'to' by following edges?"}, reach)
	mcp.AddTool(s, &mcp.Tool{Name: "layers", Description: "Sugiyama longest-path rank of every node (sources are 0)."}, layers)
	// dag-verbs:end

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
