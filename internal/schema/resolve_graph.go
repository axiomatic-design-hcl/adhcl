package schema

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/axiomatic-design-hcl/adhcl/internal/model"
)

// digraph is a directed graph over declared identities. Each edge keeps the range
// of the authored reference that created it, so that a finding can point at it.
type digraph struct {
	nodes []model.ID
	out   map[model.ID][]edge
}

type edge struct {
	to  model.ID
	rng hcl.Range
}

func newDigraph(nodes []model.ID) *digraph {
	return &digraph{nodes: nodes, out: make(map[model.ID][]edge)}
}

func (g *digraph) addEdge(from, to model.ID, rng hcl.Range) {
	g.out[from] = append(g.out[from], edge{to: to, rng: rng})
}

// backEdgeRanges returns the range of each edge that closes a cycle in depth-first
// order over the nodes. A self-loop closes a cycle.
func (g *digraph) backEdgeRanges() []hcl.Range {
	const (
		unseen = iota
		onPath
		done
	)
	state := make(map[model.ID]int)
	var found []hcl.Range
	var visit func(model.ID)
	visit = func(n model.ID) {
		state[n] = onPath
		for _, e := range g.out[n] {
			switch state[e.to] {
			case onPath:
				found = append(found, e.rng)
			case unseen:
				visit(e.to)
			}
		}
		state[n] = done
	}
	for _, n := range g.nodes {
		if state[n] == unseen {
			visit(n)
		}
	}
	return found
}

// cyclic returns the nodes that lie on a cycle, which are the nodes that reach themselves.
func (g *digraph) cyclic() map[model.ID]bool {
	out := make(map[model.ID]bool)
	for _, n := range g.nodes {
		var next []model.ID
		for _, e := range g.out[n] {
			next = append(next, e.to)
		}
		if g.reach(next...)[n] {
			out[n] = true
		}
	}
	return out
}

// reach returns the starts and every node reachable from them.
func (g *digraph) reach(starts ...model.ID) map[model.ID]bool {
	seen := make(map[model.ID]bool)
	stack := append([]model.ID(nil), starts...)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[n] {
			continue
		}
		seen[n] = true
		for _, e := range g.out[n] {
			stack = append(stack, e.to)
		}
	}
	return seen
}
