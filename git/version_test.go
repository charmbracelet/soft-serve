package git

import (
	"errors"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in      string
		want    Version
		wantErr bool
	}{
		{"git version 2.39.3 (Apple Git-146)", Version{2, 39, 3}, false},
		{"git version 2.30.2", Version{2, 30, 2}, false},
		{"git version 2.45.0.windows.1", Version{2, 45, 0}, false},
		{"git version 2.54.0\n", Version{2, 54, 0}, false},
		{"git version 2.33.0.rc0", Version{2, 33, 0}, false},
		{"git version 2.33.0-rc0", Version{2, 33, 0}, false},
		{"git version 2.43.0.vfs.0.0", Version{2, 43, 0}, false},
		{"2.41.1", Version{2, 41, 1}, false},
		{"git version 2.33", Version{2, 33, 0}, false},
		{"2.54.0\n", Version{2, 54, 0}, false},

		{"", Version{}, true},
		{"git version", Version{}, true},
		{"git version abc", Version{}, true},
		{"git version 2", Version{}, true},
		{"git version .33.0", Version{}, true},
		{"version 2.33.0", Version{}, true},
	}

	for _, tt := range tests {
		got, err := ParseVersion(tt.in)
		if tt.wantErr {
			if !errors.Is(err, ErrVersionUnparseable) {
				t.Errorf("ParseVersion(%q) error = %v, want ErrVersionUnparseable", tt.in, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVersion(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	min := Version{2, 33, 0}
	tests := []struct {
		v, o Version
		want bool
	}{
		{Version{2, 30, 2}, min, true},
		{Version{2, 32, 9}, min, true},
		{Version{2, 33, 0}, min, false},
		{Version{2, 33, 1}, min, false},
		{Version{2, 45, 0}, min, false},
		{Version{3, 0, 0}, min, false},
		{Version{1, 99, 99}, min, true},
	}
	for _, tt := range tests {
		if got := tt.v.Less(tt.o); got != tt.want {
			t.Errorf("%v.Less(%v) = %v, want %v", tt.v, tt.o, got, tt.want)
		}
	}
}

// TestBinVersion checks the installed git parses, which also covers whatever
// vendor suffix the CI platform adds.
func TestBinVersion(t *testing.T) {
	v, err := BinVersion()
	if err != nil {
		t.Fatalf("BinVersion(): %v", err)
	}
	if v.Major < 2 {
		t.Errorf("BinVersion() = %v, implausible", v)
	}
}
