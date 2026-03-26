package cycles

import (
	"sort"

	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/resolve"
)

// IncludeGraph represents a directed graph of include relationships.
type IncludeGraph struct {
	// edges maps source file (absolute path) to included file (absolute path).
	edges map[string][]string
}

// Build constructs an include graph from resolved include results.
// Only successfully resolved include references are added.
func Build(results []*resolve.Result) *IncludeGraph {
	g := &IncludeGraph{edges: make(map[string][]string)}
	for _, r := range results {
		if r.Ref.RefType != model.RefTypeInclude {
			continue
		}
		if !r.Found || r.Resource == nil {
			continue
		}
		src := r.Ref.SourceFile
		dst := r.Resource.AbsPath
		g.edges[src] = append(g.edges[src], dst)
	}
	// Sort edges for deterministic traversal
	for k := range g.edges {
		sort.Strings(g.edges[k])
	}
	return g
}

// DetectCycles returns all cycles found in the include graph.
// Each cycle is a slice of file paths forming the loop.
func (g *IncludeGraph) DetectCycles() [][]string {
	const (
		unvisited  = 0
		inProgress = 1
		done       = 2
	)

	state := make(map[string]int)
	parent := make(map[string]string)
	var cycles [][]string
	seen := make(map[string]bool) // deduplicate cycles by their start node

	var dfs func(node string)
	dfs = func(node string) {
		state[node] = inProgress
		for _, next := range g.edges[node] {
			switch state[next] {
			case unvisited:
				parent[next] = node
				dfs(next)
			case inProgress:
				// Found a cycle — trace back from node to next
				if !seen[next] {
					cycle := traceCycle(parent, node, next)
					cycles = append(cycles, cycle)
					seen[next] = true
				}
			}
		}
		state[node] = done
	}

	// Sort nodes for deterministic traversal order
	nodes := make([]string, 0, len(g.edges))
	for node := range g.edges {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	// Visit all nodes (some may not be reachable from others)
	for _, node := range nodes {
		if state[node] == unvisited {
			dfs(node)
		}
	}

	return cycles
}

// traceCycle traces the cycle path from the back-edge.
// current is the node whose neighbour (cycleStart) is already in progress.
// We trace from cycleStart → ... → current → cycleStart using the parent map.
func traceCycle(parent map[string]string, current, cycleStart string) []string {
	// Build path from cycleStart to current by tracing parent backwards
	var reversed []string
	reversed = append(reversed, current)
	node := current
	for node != cycleStart {
		p, ok := parent[node]
		if !ok {
			break
		}
		reversed = append(reversed, p)
		node = p
	}

	// Reverse to get forward order: cycleStart → ... → current
	chain := make([]string, len(reversed))
	for i, v := range reversed {
		chain[len(reversed)-1-i] = v
	}

	// Close the cycle
	chain = append(chain, cycleStart)
	return chain
}
