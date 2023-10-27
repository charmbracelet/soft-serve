package access

import "testing"

func TestParseAccessLevel(t *testing.T) {
	cases := []struct {
		in  string
		out AccessLevel
	}{
		{"", -1},
		{"foo", -1},
		{AdminAccess.String(), AdminAccess},
		{ReadOnlyAccess.String(), ReadOnlyAccess},
		{ReadWriteAccess.String(), ReadWriteAccess},
		{NoAccess.String(), NoAccess},
	}

	for _, c := range cases {
		out := ParseAccessLevel(c.in)
		if out != c.out {
			t.Errorf("ParseAccessLevel(%q) => %d, want %d", c.in, out, c.out)
		}
	}
}
