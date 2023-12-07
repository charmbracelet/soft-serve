package config

import "testing"

func TestBadSSHKeyPair(t *testing.T) {
	for _, cfg := range []*Config{
		nil,
		{},
	} {
		if _, err := KeyPair(cfg); err == nil {
			t.Errorf("cfg.SSH.KeyPair() => _, nil, want non-nil error")
		}
	}
}

func TestGoodSSHKeyPair(t *testing.T) {
	cfg := &Config{
		SSH: SSHConfig{
			KeyPath: "testdata/ssh_host_ed25519_key",
		},
	}

	if _, err := KeyPair(cfg); err != nil {
		t.Errorf("cfg.SSH.KeyPair() => _, %v, want nil error", err)
	}
}
