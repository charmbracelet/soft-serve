package git

import (
	"testing"

	"github.com/matryer/is"
)

func TestParseAttr(t *testing.T) {
	cases := []struct {
		in   string
		file string
		want []Attribute
	}{
		{
			in:   "org/example/MyClass.java: diff: java\n",
			file: "org/example/MyClass.java",
			want: []Attribute{
				{
					Name:  "diff",
					Value: "java",
				},
			},
		},
		{
			in: `org/example/MyClass.java: crlf: unset
org/example/MyClass.java: diff: java
org/example/MyClass.java: myAttr: set`,
			file: "org/example/MyClass.java",
			want: []Attribute{
				{
					Name:  "crlf",
					Value: "unset",
				},
				{
					Name:  "diff",
					Value: "java",
				},
				{
					Name:  "myAttr",
					Value: "set",
				},
			},
		},
		{
			in: `org/example/MyClass.java: diff: java
org/example/MyClass.java: myAttr: set`,
			file: "org/example/MyClass.java",
			want: []Attribute{
				{
					Name:  "diff",
					Value: "java",
				},
				{
					Name:  "myAttr",
					Value: "set",
				},
			},
		},
		{
			in:   `README: caveat: unspecified`,
			file: "README",
			want: []Attribute{
				{
					Name:  "caveat",
					Value: "unspecified",
				},
			},
		},
		{
			in:   "",
			file: "foo",
			want: []Attribute{},
		},
		{
			in:   "\n",
			file: "foo",
			want: []Attribute{},
		},
	}

	is := is.New(t)
	for _, c := range cases {
		attrs := parseAttributes(c.file, []byte(c.in))
		if len(attrs) != len(c.want) {
			t.Fatalf("parseAttributes(%q, %q) = %v, want %v", c.file, c.in, attrs, c.want)
		}

		is.Equal(attrs, c.want)
	}
}
