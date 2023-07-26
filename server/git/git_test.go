package git

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPktline(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		err  error
		out  []byte
	}{
		{
			name: "empty",
			in:   []byte{},
			out:  []byte("0005\n0000"),
		},
		{
			name: "simple",
			in:   []byte("hello"),
			out:  []byte("000ahello\n0000"),
		},
		{
			name: "newline",
			in:   []byte("hello\n"),
			out:  []byte("000bhello\n\n0000"),
		},
		{
			name: "error",
			err:  fmt.Errorf("foobar"),
			out:  []byte("000fERR foobar\n0000"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			if c.err == nil {
				if err := WritePktline(&out, string(c.in)); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := WritePktlineErr(&out, c.err); err != nil {
					t.Fatal(err)
				}
			}

			if !bytes.Equal(out.Bytes(), c.out) {
				t.Errorf("expected %q, got %q", c.out, out.Bytes())
			}
		})
	}
}
