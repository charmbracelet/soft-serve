package git

import (
	"os"
	"path/filepath"

	gcfg "github.com/go-git/go-git/v5/plumbing/format/config"
)

// Config returns the repository Git configuration.
func (r *Repository) Config() (*gcfg.Config, error) {
	cp := filepath.Join(r.Path, "config")
	f, err := os.Open(cp)
	if err != nil {
		return nil, err
	}

	defer f.Close() // nolint: errcheck
	d := gcfg.NewDecoder(f)
	cfg := gcfg.New()
	if err := d.Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SetConfig sets the repository Git configuration.
func (r *Repository) SetConfig(cfg *gcfg.Config) error {
	cp := filepath.Join(r.Path, "config")
	f, err := os.Create(cp)
	if err != nil {
		return err
	}

	defer f.Close() // nolint: errcheck
	e := gcfg.NewEncoder(f)
	return e.Encode(cfg)
}
