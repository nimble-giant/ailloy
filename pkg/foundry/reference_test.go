package foundry

import (
	"testing"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Reference
		wantErr bool
	}{
		{
			name: "simple",
			raw:  "github.com/nimble-giant/nimble-mold",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Type: Latest,
			},
		},
		{
			name: "exact version",
			raw:  "github.com/nimble-giant/nimble-mold@v1.2.3",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "v1.2.3", Type: Exact,
			},
		},
		{
			name: "exact without v prefix",
			raw:  "github.com/nimble-giant/nimble-mold@1.2.3",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "1.2.3", Type: Exact,
			},
		},
		{
			name: "caret constraint",
			raw:  "github.com/nimble-giant/nimble-mold@^1.0.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "^1.0.0", Type: Constraint,
			},
		},
		{
			name: "tilde constraint",
			raw:  "github.com/nimble-giant/nimble-mold@~1.2.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "~1.2.0", Type: Constraint,
			},
		},
		{
			name: "range constraint",
			raw:  "github.com/nimble-giant/nimble-mold@>=1.0.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: ">=1.0.0", Type: Constraint,
			},
		},
		{
			name: "sha",
			raw:  "github.com/nimble-giant/nimble-mold@abc1234",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "abc1234", Type: SHA,
			},
		},
		{
			name: "full sha",
			raw:  "github.com/nimble-giant/nimble-mold@abc1234567890abc1234567890abc1234567890a",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "abc1234567890abc1234567890abc1234567890a", Type: SHA,
			},
		},
		{
			name: "explicit latest",
			raw:  "github.com/nimble-giant/nimble-mold@latest",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "latest", Type: Latest,
			},
		},
		{
			name: "branch",
			raw:  "github.com/nimble-giant/nimble-mold@main",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "main", Type: Branch,
			},
		},
		{
			name: "subpath",
			raw:  "github.com/nimble-giant/nimble-mold@v1.0.0//molds/claude",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "v1.0.0", Subpath: "molds/claude", Type: Exact,
			},
		},
		{
			name: "subpath without version",
			raw:  "github.com/nimble-giant/nimble-mold//molds/claude",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Subpath: "molds/claude", Type: Latest,
			},
		},
		{
			name: "https url",
			raw:  "https://github.com/nimble-giant/nimble-mold@v1.0.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "v1.0.0", Type: Exact,
			},
		},
		{
			name: "ssh url",
			raw:  "git@github.com:nimble-giant/nimble-mold@v1.0.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "v1.0.0", Type: Exact,
			},
		},
		{
			name: "trailing .git",
			raw:  "github.com/nimble-giant/nimble-mold.git@v1.0.0",
			want: Reference{
				Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold",
				Version: "v1.0.0", Type: Exact,
			},
		},
		{
			name: "gitlab host",
			raw:  "gitlab.com/team/project@v2.0.0",
			want: Reference{
				Host: "gitlab.com", Owner: "team", Repo: "project",
				Version: "v2.0.0", Type: Exact,
			},
		},
		{
			name:    "empty",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "missing repo",
			raw:     "github.com/owner",
			wantErr: true,
		},
		{
			name:    "single segment",
			raw:     "something",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("Owner = %q, want %q", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("Repo = %q, want %q", got.Repo, tt.want.Repo)
			}
			if got.Version != tt.want.Version {
				t.Errorf("Version = %q, want %q", got.Version, tt.want.Version)
			}
			if got.Subpath != tt.want.Subpath {
				t.Errorf("Subpath = %q, want %q", got.Subpath, tt.want.Subpath)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
		})
	}
}

func TestIsRemoteReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"github.com/nimble-giant/nimble-mold", true},
		{"github.com/nimble-giant/nimble-mold@v1.0.0", true},
		{"gitlab.com/team/project", true},
		{"https://github.com/nimble-giant/nimble-mold", true},
		{"http://github.com/nimble-giant/nimble-mold", true},
		{"git@github.com:nimble-giant/nimble-mold", true},

		{"./local-mold", false},
		{"../parent-mold", false},
		{"/absolute/path/to/mold", false},
		{"~/molds/my-mold", false},
		{"local-dir", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsRemoteReference(tt.input)
			if got != tt.want {
				t.Errorf("IsRemoteReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReference_CloneURL(t *testing.T) {
	ref := &Reference{Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold"}
	want := "https://github.com/nimble-giant/nimble-mold.git"
	if got := ref.CloneURL(); got != want {
		t.Errorf("CloneURL() = %q, want %q", got, want)
	}
}

func TestReference_CacheKey(t *testing.T) {
	ref := &Reference{Host: "github.com", Owner: "nimble-giant", Repo: "nimble-mold"}
	want := "github.com/nimble-giant/nimble-mold"
	if got := ref.CacheKey(); got != want {
		t.Errorf("CacheKey() = %q, want %q", got, want)
	}
}

func TestReference_String(t *testing.T) {
	tests := []struct {
		name string
		ref  Reference
		want string
	}{
		{
			name: "no version no subpath",
			ref:  Reference{Host: "github.com", Owner: "owner", Repo: "repo"},
			want: "github.com/owner/repo",
		},
		{
			name: "with version",
			ref:  Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "v1.0.0"},
			want: "github.com/owner/repo@v1.0.0",
		},
		{
			name: "with subpath",
			ref:  Reference{Host: "github.com", Owner: "owner", Repo: "repo", Version: "v1.0.0", Subpath: "sub/path"},
			want: "github.com/owner/repo@v1.0.0//sub/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
