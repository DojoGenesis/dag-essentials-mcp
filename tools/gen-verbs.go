//go:build ignore

// gen-verbs regenerates the mcp.AddTool(...) registration block in main.go from
// dag-verbs.json — the single source of truth for the DAG essentials' machine face.
// The tool handler funcs (validate, toposort, …) stay hand-written; only the
// registration lines between the `dag-verbs:begin` / `dag-verbs:end` markers are
// generated. Run from the module root:
//
//	go run ./tools/gen-verbs.go
//
// It is idempotent: with an unchanged dag-verbs.json it reproduces the block
// byte-for-byte. The parity gate (check_verb_parity.py) enforces that main.go,
// the contract, and the site stay in sync.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type tool struct {
	ID          string `json:"id"`
	Handler     string `json:"handler"`
	Description string `json:"description"`
}

type contract struct {
	Tools []tool `json:"tools"`
}

const (
	contractFile = "dag-verbs.json"
	mainFile     = "main.go"
	beginMarker  = "\t// dag-verbs:begin — generated from dag-verbs.json by `go run ./tools/gen-verbs.go`; do not edit by hand"
	endMarker    = "\t// dag-verbs:end"
)

// block spans the begin marker line through the end marker line, inclusive.
var block = regexp.MustCompile(`(?s)\t// dag-verbs:begin.*?\t// dag-verbs:end`)

func main() {
	raw, err := os.ReadFile(contractFile)
	if err != nil {
		fatal(err)
	}
	var c contract
	if err := json.Unmarshal(raw, &c); err != nil {
		fatal(fmt.Errorf("parsing %s: %w", contractFile, err))
	}
	if len(c.Tools) == 0 {
		fatal(fmt.Errorf("%s has no tools", contractFile))
	}

	var b strings.Builder
	b.WriteString(beginMarker + "\n")
	for _, t := range c.Tools {
		if t.ID == "" || t.Handler == "" {
			fatal(fmt.Errorf("tool missing id or handler: %+v", t))
		}
		fmt.Fprintf(&b, "\tmcp.AddTool(s, &mcp.Tool{Name: %q, Description: %q}, %s)\n", t.ID, t.Description, t.Handler)
	}
	b.WriteString(endMarker)

	src, err := os.ReadFile(mainFile)
	if err != nil {
		fatal(err)
	}
	if !block.Match(src) {
		fatal(fmt.Errorf("%s: dag-verbs:begin/end markers not found", mainFile))
	}
	out := block.ReplaceAllLiteral(src, []byte(b.String()))
	if err := os.WriteFile(mainFile, out, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("gen-verbs: regenerated %d tool registrations in %s\n", len(c.Tools), mainFile)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen-verbs:", err)
	os.Exit(1)
}
