package webhook

import "testing"

func TestParseContentType(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want ContentType
		err  error
	}{
		{
			name: "JSON",
			s:    "application/json",
			want: ContentTypeJSON,
		},
		{
			name: "Form",
			s:    "application/x-www-form-urlencoded",
			want: ContentTypeForm,
		},
		{
			name: "Invalid",
			s:    "application/invalid",
			err:  ErrInvalidContentType,
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContentType(tt.s)
			if err != tt.err {
				t.Errorf("ParseContentType() error = %v, wantErr %v", err, tt.err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseContentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		text    []byte
		want    ContentType
		wantErr bool
	}{
		{
			name: "JSON",
			text: []byte("application/json"),
			want: ContentTypeJSON,
		},
		{
			name: "Form",
			text: []byte("application/x-www-form-urlencoded"),
			want: ContentTypeForm,
		},
		{
			name:    "Invalid",
			text:    []byte("application/invalid"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := new(ContentType)
			if err := c.UnmarshalText(tt.text); (err != nil) != tt.wantErr {
				t.Errorf("ContentType.UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
			if *c != tt.want {
				t.Errorf("ContentType.UnmarshalText() got = %v, want %v", *c, tt.want)
			}
		})
	}
}

func TestMarshalText(t *testing.T) {
	tests := []struct {
		name    string
		c       ContentType
		want    []byte
		wantErr bool
	}{
		{
			name: "JSON",
			c:    ContentTypeJSON,
			want: []byte("application/json"),
		},
		{
			name: "Form",
			c:    ContentTypeForm,
			want: []byte("application/x-www-form-urlencoded"),
		},
		{
			name:    "Invalid",
			c:       ContentType(-1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.c.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContentType.MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(b) != string(tt.want) {
				t.Errorf("ContentType.MarshalText() got = %v, want %v", string(b), string(tt.want))
			}
		})
	}
}
