package main

import "testing"

func TestRejectUnsupportedArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"drop"}, {"down", "all"}, {"up", "extra"}} {
		if err := run(args); err == nil {
			t.Fatalf("accepted arguments: %v", args)
		}
	}
}
