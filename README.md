# dag-essentials-mcp

An MCP server that exposes the essentials of a directed acyclic graph as tools —
the operations the essay [*The Phenomenology of the DAG*](https://madeofdoors.com)
names as verbs. A small, pure-compute server: no network, no I/O, no state. You
hand it nodes and edges; it hands back the frontier, the critical path, the
cone, the cycle it refuses to keep.

Built in Go (stdlib graph core + the official
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk),
stdio transport) so it drops into an agent stack as a single binary.

## Tools

| Tool | Returns | Verb |
|---|---|---|
| `validate` | acyclic? if not, a **cycle witness** (`v0 → … → v0`) | encounter / the cut |
| `toposort` | one legal topological order (Kahn) | follow |
| `critical_path` | the longest (optionally weighted) chain — the floor | touch |
| `width` | the largest antichain (Dilworth) + a min chain cover | work with |
| `frontier` | given completed nodes, the ready set (in-degree 0) | hold / orchestrate |
| `cone` | a node's ancestors and descendants | encircle |
| `reduce` | the transitive reduction (the unique Hasse diagram) | layer |
| `reach` | can A reach B? | traverse |
| `layers` | Sugiyama longest-path rank of every node | layer / the look |

Every tool takes a graph as `nodes` (string list) and `edges` (list of
`[from, to]` pairs). Endpoints named only in `edges` are added automatically.

## Build & test

```sh
go test ./...          # the pure DAG core is fully tested, no SDK required
go build -o dag-essentials-mcp.exe ./...   # CGO_ENABLED=0 for a portable binary
```

## Register with Claude Code

```sh
claude mcp add dag-essentials --transport stdio -- "C:\Users\cruzr\dag-essentials-mcp\dag-essentials-mcp.exe"
```

## Layout

```
dag-essentials-mcp/
  dag/            pure graph algorithms (stdlib only) + tests
    dag.go
    dag_test.go
  main.go         MCP server: wraps dag.* as stdio tools   (wired against go-sdk v1.5.0)
  go.mod
```

The `dag` package is independent of the MCP layer — usable as a plain Go library,
and tested without the SDK. The server is a thin shell around it.

— a companion to madeofdoors.com · *a creature made of doors*
