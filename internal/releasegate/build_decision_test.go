package releasegate

import "testing"

func TestDecideBuild(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		status     string
		experience string
		diagnostic string
	}{
		{name: "flag off keeps incumbent", enabled: false, status: "failed", experience: "incumbent"},
		{name: "flag on annotates failed build", enabled: true, status: "failed", experience: "infrai", diagnostic: "inspect failed build stages"},
		{name: "flag on acknowledges passing build", enabled: true, status: "passed", experience: "infrai", diagnostic: "build completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideBuild(BuildEvent{BuildID: "build-184", Status: tt.status}, tt.enabled)
			if got.Experience != tt.experience || got.Diagnostic != tt.diagnostic {
				t.Fatalf("DecideBuild() = %#v", got)
			}
		})
	}
}
