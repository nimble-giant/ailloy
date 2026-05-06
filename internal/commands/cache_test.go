package commands

import "testing"

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range cases {
		got := humanizeBytes(tc.in)
		if got != tc.want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
