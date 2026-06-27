package main

import (
	"os"
	"testing"
)

// TestCallerRefPrefersTreeOverEnv pins the Phase-0 safety fix: cmux exposes the
// caller's surface in the SAME "surface:N" space as every target ref, while
// CMUX_SURFACE_ID can be a UUID. Self-identity MUST come from the tree so that
// ref-equality self-checks actually fire (the bug: UUID never equals "surface:N",
// defeating the never-inject-into-self guard).
func TestCallerRefPrefersTreeOverEnv(t *testing.T) {
	os.Setenv("CMUX_SURFACE_ID", "6FA8FF39-C10C-4E13-A125-E2C31464DBD2") // a UUID
	defer os.Unsetenv("CMUX_SURFACE_ID")

	tree := &treeJSON{}
	tree.Caller.SurfaceRef = "surface:45"

	if got := callerRef(tree); got != "surface:45" {
		t.Fatalf("callerRef = %q, want surface:45 (tree must win over UUID env)", got)
	}
}

// TestCallerRefFallsBackToEnv: with no tree caller ref, fall back to the env var.
func TestCallerRefFallsBackToEnv(t *testing.T) {
	os.Setenv("CMUX_SURFACE_ID", "surface:7")
	defer os.Unsetenv("CMUX_SURFACE_ID")
	if got := callerRef(&treeJSON{}); got != "surface:7" {
		t.Fatalf("callerRef = %q, want surface:7 (env fallback)", got)
	}
}

// TestIsSelf: a surface is "self" if cmux marks it Here, OR its ref equals the
// authoritative self ref. Both signals are needed: Here is what cmux computes,
// ref-equality is the cross-check.
func TestIsSelf(t *testing.T) {
	cases := []struct {
		name string
		s    Surface
		self string
		want bool
	}{
		{"here-flag", Surface{Ref: "surface:45", Here: true}, "surface:99", true},
		{"ref-match", Surface{Ref: "surface:45"}, "surface:45", true},
		{"other", Surface{Ref: "surface:42"}, "surface:45", false},
		{"empty-self-not-self", Surface{Ref: "surface:42"}, "", false},
		{"uuid-self-no-false-match", Surface{Ref: "surface:42"}, "6FA8FF39-C10C", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isSelf(c.s, c.self); got != c.want {
				t.Fatalf("isSelf(%+v, %q) = %v, want %v", c.s, c.self, got, c.want)
			}
		})
	}
}
