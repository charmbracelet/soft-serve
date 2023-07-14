package sshutils

import (
	"testing"

	"github.com/charmbracelet/keygen"
	"golang.org/x/crypto/ssh"
)

func generateKeys(tb testing.TB) (*keygen.SSHKeyPair, *keygen.SSHKeyPair) {
	goodKey1, err := keygen.New("", keygen.WithKeyType(keygen.Ed25519))
	if err != nil {
		tb.Fatal(err)
	}
	goodKey2, err := keygen.New("", keygen.WithKeyType(keygen.RSA))
	if err != nil {
		tb.Fatal(err)
	}

	return goodKey1, goodKey2
}

func TestParseAuthorizedKey(t *testing.T) {
	goodKey1, goodKey2 := generateKeys(t)
	cases := []struct {
		in   string
		good bool
	}{
		{
			goodKey1.AuthorizedKey(),
			true,
		},
		{
			goodKey2.AuthorizedKey(),
			true,
		},
		{
			goodKey1.AuthorizedKey() + "test",
			false,
		},
		{
			goodKey2.AuthorizedKey() + "bad",
			false,
		},
	}
	for _, c := range cases {
		_, _, err := ParseAuthorizedKey(c.in)
		if c.good && err != nil {
			t.Errorf("ParseAuthorizedKey(%q) returned error: %v", c.in, err)
		}
		if !c.good && err == nil {
			t.Errorf("ParseAuthorizedKey(%q) did not return error", c.in)
		}
	}
}

func TestMarshalAuthorizedKey(t *testing.T) {
	goodKey1, goodKey2 := generateKeys(t)
	cases := []struct {
		in       ssh.PublicKey
		expected string
	}{
		{
			goodKey1.PublicKey(),
			goodKey1.AuthorizedKey(),
		},
		{
			goodKey2.PublicKey(),
			goodKey2.AuthorizedKey(),
		},
		{
			nil,
			"",
		},
	}
	for _, c := range cases {
		out := MarshalAuthorizedKey(c.in)
		if out != c.expected {
			t.Errorf("MarshalAuthorizedKey(%v) returned %q, expected %q", c.in, out, c.expected)
		}
	}
}

func TestKeysEqual(t *testing.T) {
	goodKey1, goodKey2 := generateKeys(t)
	cases := []struct {
		in1      ssh.PublicKey
		in2      ssh.PublicKey
		expected bool
	}{
		{
			goodKey1.PublicKey(),
			goodKey1.PublicKey(),
			true,
		},
		{
			goodKey2.PublicKey(),
			goodKey2.PublicKey(),
			true,
		},
		{
			goodKey1.PublicKey(),
			goodKey2.PublicKey(),
			false,
		},
		{
			nil,
			nil,
			false,
		},
		{
			nil,
			goodKey1.PublicKey(),
			false,
		},
	}

	for _, c := range cases {
		out := KeysEqual(c.in1, c.in2)
		if out != c.expected {
			t.Errorf("KeysEqual(%v, %v) returned %v, expected %v", c.in1, c.in2, out, c.expected)
		}
	}
}
