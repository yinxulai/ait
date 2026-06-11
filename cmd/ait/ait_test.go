package main

import "testing"

func TestFlagRouting(t *testing.T) {
	tests := []struct {
		name string
		mcp  bool
		web  bool
		want string
	}{
		{name: "default tui", mcp: false, web: false, want: "tui"},
		{name: "mcp enabled", mcp: true, web: false, want: "mcp"},
		{name: "web enabled", mcp: false, web: true, want: "web"},
		{name: "mcp wins", mcp: true, web: true, want: "mcp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routeByFlags(tt.mcp, tt.web)
			if got != tt.want {
				t.Fatalf("route = %q, want %q", got, tt.want)
			}
		})
	}
}
