package main

import "testing"

func TestMCPFlagRouting(t *testing.T) {
	tests := []struct {
		name string
		mcp  bool
		want string
	}{
		{name: "default tui", mcp: false, want: "tui"},
		{name: "mcp enabled", mcp: true, want: "mcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routeByMCPFlag(tt.mcp)
			if got != tt.want {
				t.Fatalf("route = %q, want %q", got, tt.want)
			}
		})
	}
}
