package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// relations.go — the session FAMILY graph (parent / children).
//
// Relationships are LOGICAL, not spatial: a session's parent is the session that
// spawned or owns it, which cmux's pane/workspace layout doesn't encode. So edges
// are declared (`communikey link`) or recorded at spawn time, and persisted here.
// Nodes are keyed by stable agent session-id (see sessionKey) so a surface
// renumber doesn't orphan the tree.

type Edge struct {
	Parent string `json:"parent"`
	Child  string `json:"child"`
}

type Relations struct {
	Edges []Edge            `json:"edges"`
	Names map[string]string `json:"names,omitempty"` // id -> friendly label (for offline nodes)
}

func relationsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "communikey", "relations.json")
}

func loadRelations() *Relations {
	r := &Relations{Names: map[string]string{}}
	if data, err := os.ReadFile(relationsPath()); err == nil {
		_ = json.Unmarshal(data, r)
	}
	if r.Names == nil {
		r.Names = map[string]string{}
	}
	return r
}

func (r *Relations) save() error {
	p := relationsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(r, "", "  ")
	return os.WriteFile(p, data, 0o644)
}

// link records that child's parent is parent. A child has at most one parent, so
// any previous parent edge for this child is replaced. Self-links and direct
// cycles are rejected by the caller.
func (r *Relations) link(parent, child string) {
	r.unlinkChild(child)
	r.Edges = append(r.Edges, Edge{Parent: parent, Child: child})
}

func (r *Relations) unlinkChild(child string) {
	out := r.Edges[:0]
	for _, e := range r.Edges {
		if e.Child != child {
			out = append(out, e)
		}
	}
	r.Edges = out
}

func (r *Relations) childrenOf(id string) []string {
	var c []string
	for _, e := range r.Edges {
		if e.Parent == id {
			c = append(c, e.Child)
		}
	}
	return c
}

func (r *Relations) parentOf(id string) (string, bool) {
	for _, e := range r.Edges {
		if e.Child == id {
			return e.Parent, true
		}
	}
	return "", false
}

// wouldCycle reports whether making `parent` the parent of `child` would create
// a cycle (i.e. parent is already a descendant of child).
func (r *Relations) wouldCycle(parent, child string) bool {
	cur := parent
	for i := 0; i < 1000; i++ {
		if cur == child {
			return true
		}
		p, ok := r.parentOf(cur)
		if !ok {
			return false
		}
		cur = p
	}
	return true // pathological depth → treat as cycle
}
