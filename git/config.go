package git

import (
	"fmt"
	"os"
	"path/filepath"

	gcfg "github.com/go-git/go-git/v5/plumbing/format/config"
)

// Config returns the repository Git configuration.
func (r *Repository) Config() (*gcfg.Config, error) {
	cp := filepath.Join(r.Path, "config")
	f, err := os.Open(cp)
	if err != nil {
		return nil, fmt.Errorf("failed to open git config file: %w", err)
	}

	defer f.Close()
	d := gcfg.NewDecoder(f)
	cfg := gcfg.New()
	if err := d.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode git config: %w", err)
	}

	return cfg, nil
}

// SetConfig sets the repository Git configuration.
func (r *Repository) SetConfig(cfg *gcfg.Config) error {
	cp := filepath.Join(r.Path, "config")
	f, err := os.Create(cp)
	if err != nil {
		return fmt.Errorf("failed to create git config file: %w", err)
	}

	defer f.Close()
	e := gcfg.NewEncoder(f)
	if err := e.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode git config: %w", err)
	}
	return nil
}
