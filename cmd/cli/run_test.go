package main

import "testing"

func TestParseSkillInvocation(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs string
	}{
		{
			input:    "/commit",
			wantName: "commit",
			wantArgs: "",
		},
		{
			input:    "/commit -m fix bug",
			wantName: "commit",
			wantArgs: "-m fix bug",
		},
		{
			input:    "/review-pr 123",
			wantName: "review-pr",
			wantArgs: "123",
		},
		{
			input:    "/skill-name arg1 arg2 arg3",
			wantName: "skill-name",
			wantArgs: "arg1 arg2 arg3",
		},
		{
			input:    "commit",
			wantName: "commit",
			wantArgs: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, args := parseSkillInvocation(tt.input)
			if name != tt.wantName {
				t.Errorf("parseSkillInvocation(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if args != tt.wantArgs {
				t.Errorf("parseSkillInvocation(%q) args = %q, want %q", tt.input, args, tt.wantArgs)
			}
		})
	}
}
