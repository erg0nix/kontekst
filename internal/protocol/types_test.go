package protocol

import "testing"

func TestToolKindFromName(t *testing.T) {
	tests := []struct {
		name string
		want ToolKind
	}{
		{"read_file", ToolKindRead},
		{"write_file", ToolKindEdit},
		{"edit_file", ToolKindEdit},
		{"list_files", ToolKindSearch},
		{"run_command", ToolKindExecute},
		{"web_fetch", ToolKindFetch},
		{"skill", ToolKindOther},
		{"unknown_tool", ToolKindOther},
	}

	for _, tt := range tests {
		got := ToolKindFromName(tt.name)
		if got != tt.want {
			t.Errorf("ToolKindFromName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPermissionOptionKindIsAllow(t *testing.T) {
	tests := []struct {
		kind PermissionOptionKind
		want bool
	}{
		{PermissionOptionKindAllowOnce, true},
		{PermissionOptionKindAllowAlways, true},
		{PermissionOptionKindRejectOnce, false},
		{PermissionOptionKindRejectAlways, false},
	}

	for _, tt := range tests {
		if got := tt.kind.IsAllow(); got != tt.want {
			t.Errorf("%q.IsAllow() = %v, want %v", tt.kind, got, tt.want)
		}
	}
}
