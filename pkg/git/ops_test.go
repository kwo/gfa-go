package git

import "testing"

func TestParseRemoteStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
		wantOK     bool
	}{
		{
			name:       "up to date",
			input:      "  other abc123 [origin/other] commit\n* main def456 [origin/main] commit",
			wantStatus: "up-to-date",
			wantOK:     true,
		},
		{
			name:       "behind",
			input:      "* main def456 [origin/main: behind 3] commit",
			wantStatus: "behind 3",
			wantOK:     true,
		},
		{
			name:       "ahead and behind",
			input:      "* main def456 [origin/main: ahead 1, behind 2] commit",
			wantStatus: "ahead 1, behind 2",
			wantOK:     true,
		},
		{
			name:   "no remote branch",
			input:  "* main def456 commit",
			wantOK: false,
		},
		{
			name:   "no current branch",
			input:  "  main def456 [origin/main] commit",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOK := parseRemoteStatus(tt.input)
			if gotStatus != tt.wantStatus || gotOK != tt.wantOK {
				t.Fatalf("parseRemoteStatus() = %q, %v; want %q, %v", gotStatus, gotOK, tt.wantStatus, tt.wantOK)
			}
		})
	}
}
